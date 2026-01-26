// Package bwrap provides a bubblewrap-based sandbox implementation for Linux.
package bwrap

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/neul-labs/m9m/internal/sandbox"
)

const (
	bwrapBinary = "bwrap"
)

// BubblewrapSandbox implements the sandbox.Sandbox interface using bubblewrap
type BubblewrapSandbox struct {
	bwrapPath     string
	cgroupManager *CgroupManager
	mu            sync.Mutex
}

// New creates a new bubblewrap sandbox
func New() (*BubblewrapSandbox, error) {
	bwrapPath, err := exec.LookPath(bwrapBinary)
	if err != nil {
		return nil, fmt.Errorf("bubblewrap not found: %w", err)
	}

	return &BubblewrapSandbox{
		bwrapPath:     bwrapPath,
		cgroupManager: NewCgroupManager(),
	}, nil
}

func (b *BubblewrapSandbox) Type() sandbox.SandboxType {
	return sandbox.SandboxTypeBubblewrap
}

func (b *BubblewrapSandbox) Available() error {
	if _, err := exec.LookPath(bwrapBinary); err != nil {
		return fmt.Errorf("bubblewrap not installed: %w", err)
	}
	return nil
}

func (b *BubblewrapSandbox) Validate(config *sandbox.SandboxConfig) error {
	if config == nil {
		return fmt.Errorf("sandbox config cannot be nil")
	}

	// Validate mounts
	for _, mount := range config.Mounts {
		if mount.Type == sandbox.MountReadOnly || mount.Type == sandbox.MountReadWrite {
			if mount.Source == "" {
				return fmt.Errorf("mount source required for bind mounts to %s", mount.Destination)
			}
			if _, err := os.Stat(mount.Source); err != nil {
				return fmt.Errorf("mount source does not exist: %s", mount.Source)
			}
		}
		if mount.Destination == "" {
			return fmt.Errorf("mount destination required")
		}
	}

	return nil
}

// Execute runs a command in the sandbox (one-shot mode)
func (b *BubblewrapSandbox) Execute(
	ctx context.Context,
	config *sandbox.SandboxConfig,
	command string,
	args ...string,
) (*sandbox.ExecutionResult, error) {
	if err := b.Validate(config); err != nil {
		return nil, err
	}

	// Build bwrap command
	builder := NewCommandBuilder(b.bwrapPath)
	bwrapArgs, cleanup, err := builder.Build(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build bwrap command: %w", err)
	}
	defer cleanup()

	// Add the actual command and args
	bwrapArgs = append(bwrapArgs, command)
	bwrapArgs = append(bwrapArgs, args...)

	// Create cgroup for resource limits if enabled
	var cgroupPath string
	if config.Limits.MaxMemoryBytes > 0 || config.Limits.MaxProcesses > 0 {
		var err error
		cgroupPath, err = b.cgroupManager.CreateCgroup(config.Limits)
		if err != nil {
			// Log warning but continue without cgroup limits
			fmt.Fprintf(os.Stderr, "warning: failed to create cgroup: %v\n", err)
		} else {
			defer b.cgroupManager.RemoveCgroup(cgroupPath)
		}
	}

	// Apply timeout context
	if config.Limits.ExecutionTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, config.Limits.ExecutionTimeout)
		defer cancel()
	}

	// Create command
	cmd := exec.CommandContext(ctx, b.bwrapPath, bwrapArgs...)

	// Set up I/O
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set environment
	cmd.Env = b.buildEnv(config)

	// Start execution
	startTime := time.Now()

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start sandboxed process: %w", err)
	}

	// Move process to cgroup if applicable
	if cgroupPath != "" {
		if err := b.cgroupManager.AddProcess(cgroupPath, cmd.Process.Pid); err != nil {
			// Log warning but continue
			fmt.Fprintf(os.Stderr, "warning: failed to add process to cgroup: %v\n", err)
		}
	}

	// Wait for completion
	waitErr := cmd.Wait()
	endTime := time.Now()

	result := &sandbox.ExecutionResult{
		Stdout:    stdout.Bytes(),
		Stderr:    stderr.Bytes(),
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(startTime),
	}

	// Handle exit status
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()

			// Check if killed by signal
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				if status.Signaled() {
					result.Killed = true
					result.KillReason = fmt.Sprintf("signal %d", status.Signal())
				}
			}
		} else if ctx.Err() == context.DeadlineExceeded {
			result.Killed = true
			result.KillReason = "timeout"
			result.ExitCode = -1
		} else {
			return nil, fmt.Errorf("command execution failed: %w", waitErr)
		}
	}

	// Get resource usage from cgroup
	if cgroupPath != "" {
		stats, _ := b.cgroupManager.GetStats(cgroupPath)
		if stats != nil {
			result.CPUTime = stats.CPUTime
			result.MaxMemory = stats.MaxMemory
		}
	}

	return result, nil
}

// ExecuteStreaming starts an interactive sandboxed execution
func (b *BubblewrapSandbox) ExecuteStreaming(
	ctx context.Context,
	config *sandbox.SandboxConfig,
	command string,
	args ...string,
) (sandbox.StreamingExecution, error) {
	if err := b.Validate(config); err != nil {
		return nil, err
	}

	return NewStreamingExecution(ctx, b, config, command, args)
}

func (b *BubblewrapSandbox) buildEnv(config *sandbox.SandboxConfig) []string {
	env := make([]string, 0, len(config.EnvVars)+len(config.EnvInherit))

	// Add explicit env vars
	env = append(env, config.EnvVars...)

	// Inherit specified vars from host
	for _, name := range config.EnvInherit {
		if value := os.Getenv(name); value != "" {
			env = append(env, fmt.Sprintf("%s=%s", name, value))
		}
	}

	return env
}

func (b *BubblewrapSandbox) Cleanup() error {
	return b.cgroupManager.Cleanup()
}
