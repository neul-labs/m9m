package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"time"
)

// SandboxCreator is a function type for creating sandbox instances
type SandboxCreator func() (Sandbox, error)

// sandboxRegistry holds registered sandbox creators
var sandboxRegistry = make(map[SandboxType]SandboxCreator)

// RegisterSandbox registers a sandbox creator for a given type
// This is called by sandbox implementations at init time
func RegisterSandbox(sandboxType SandboxType, creator SandboxCreator) {
	sandboxRegistry[sandboxType] = creator
}

// defaultFactory implements SandboxFactory
type defaultFactory struct{}

// NewFactory creates a new sandbox factory
func NewFactory() SandboxFactory {
	return &defaultFactory{}
}

func (f *defaultFactory) Create(sandboxType SandboxType) (Sandbox, error) {
	switch sandboxType {
	case SandboxTypeNone:
		return NewNoSandbox(), nil
	default:
		// Check registry for other sandbox types
		if creator, ok := sandboxRegistry[sandboxType]; ok {
			return creator()
		}
		return nil, fmt.Errorf("unsupported sandbox type: %s", sandboxType)
	}
}

func (f *defaultFactory) DetectBest() (Sandbox, error) {
	if runtime.GOOS == "linux" {
		// Check if bubblewrap is available and registered
		if _, err := exec.LookPath("bwrap"); err == nil {
			if creator, ok := sandboxRegistry[SandboxTypeBubblewrap]; ok {
				sb, err := creator()
				if err == nil {
					return sb, nil
				}
				// Fall through to NoSandbox on error
			}
		}
	}

	// Fall back to no sandbox
	return NewNoSandbox(), nil
}

func (f *defaultFactory) ListAvailable() []SandboxType {
	available := []SandboxType{SandboxTypeNone}

	if runtime.GOOS == "linux" {
		if _, err := exec.LookPath("bwrap"); err == nil {
			available = append(available, SandboxTypeBubblewrap)
		}
	}

	return available
}

// NoSandbox implements Sandbox without any isolation (for development/fallback)
type NoSandbox struct{}

// NewNoSandbox creates a new no-op sandbox
func NewNoSandbox() *NoSandbox {
	return &NoSandbox{}
}

func (ns *NoSandbox) Type() SandboxType {
	return SandboxTypeNone
}

func (ns *NoSandbox) Available() error {
	return nil // Always available
}

func (ns *NoSandbox) Validate(config *SandboxConfig) error {
	return nil // No validation needed
}

// Execute runs a command without sandboxing
func (ns *NoSandbox) Execute(ctx context.Context, config *SandboxConfig, command string, args ...string) (*ExecutionResult, error) {
	// Apply timeout if configured
	if config != nil && config.Limits.ExecutionTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, config.Limits.ExecutionTimeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, command, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set working directory
	if config != nil && config.WorkDir != "" {
		cmd.Dir = config.WorkDir
	}

	// Set environment
	if config != nil {
		env := os.Environ()
		// Add explicit env vars
		env = append(env, config.EnvVars...)
		cmd.Env = env
	}

	startTime := time.Now()
	err := cmd.Run()
	endTime := time.Now()

	result := &ExecutionResult{
		Stdout:    stdout.Bytes(),
		Stderr:    stderr.Bytes(),
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(startTime),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
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
			return nil, fmt.Errorf("command execution failed: %w", err)
		}
	}

	return result, nil
}

// ExecuteStreaming starts an interactive execution without sandboxing
func (ns *NoSandbox) ExecuteStreaming(ctx context.Context, config *SandboxConfig, command string, args ...string) (StreamingExecution, error) {
	return newNoSandboxStreaming(ctx, config, command, args)
}

func (ns *NoSandbox) Cleanup() error {
	return nil
}

// noSandboxStreaming implements StreamingExecution for NoSandbox
type noSandboxStreaming struct {
	cmd    *exec.Cmd
	stdin  *pipeWriter
	stdout *pipeReader
	stderr *pipeReader

	startTime time.Time
	done      chan struct{}
	result    *ExecutionResult
	resultErr error
}

// pipeWriter wraps an io.WriteCloser
type pipeWriter struct {
	pipe io.WriteCloser
}

func (w *pipeWriter) Write(p []byte) (n int, err error) {
	return w.pipe.Write(p)
}

func (w *pipeWriter) Close() error {
	return w.pipe.Close()
}

// pipeReader wraps an io.ReadCloser
type pipeReader struct {
	pipe io.ReadCloser
}

func (r *pipeReader) Read(p []byte) (n int, err error) {
	return r.pipe.Read(p)
}

func newNoSandboxStreaming(ctx context.Context, config *SandboxConfig, command string, args []string) (*noSandboxStreaming, error) {
	cmd := exec.CommandContext(ctx, command, args...)

	// Set working directory
	if config != nil && config.WorkDir != "" {
		cmd.Dir = config.WorkDir
	}

	// Set environment
	if config != nil {
		env := os.Environ()
		env = append(env, config.EnvVars...)
		cmd.Env = env
	}

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	se := &noSandboxStreaming{
		cmd:       cmd,
		stdin:     &pipeWriter{pipe: stdinPipe},
		stdout:    &pipeReader{pipe: stdoutPipe},
		stderr:    &pipeReader{pipe: stderrPipe},
		startTime: time.Now(),
		done:      make(chan struct{}),
	}

	// Start background waiter
	go se.waitLoop()

	return se, nil
}

func (se *noSandboxStreaming) Stdin() io.WriteCloser {
	return se.stdin
}

func (se *noSandboxStreaming) Stdout() io.Reader {
	return se.stdout
}

func (se *noSandboxStreaming) Stderr() io.Reader {
	return se.stderr
}

func (se *noSandboxStreaming) Wait() (*ExecutionResult, error) {
	<-se.done
	return se.result, se.resultErr
}

func (se *noSandboxStreaming) Signal(sig os.Signal) error {
	if se.cmd.Process == nil {
		return fmt.Errorf("process not started")
	}
	return se.cmd.Process.Signal(sig)
}

func (se *noSandboxStreaming) Kill() error {
	if se.cmd.Process == nil {
		return fmt.Errorf("process not started")
	}
	return se.cmd.Process.Kill()
}

func (se *noSandboxStreaming) Resize(cols, rows int) error {
	// No-op for non-TTY mode
	return nil
}

func (se *noSandboxStreaming) PID() int {
	if se.cmd.Process == nil {
		return 0
	}
	return se.cmd.Process.Pid
}

func (se *noSandboxStreaming) waitLoop() {
	defer close(se.done)

	err := se.cmd.Wait()
	endTime := time.Now()

	se.result = &ExecutionResult{
		StartTime: se.startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(se.startTime),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			se.result.ExitCode = exitErr.ExitCode()

			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				if status.Signaled() {
					se.result.Killed = true
					se.result.KillReason = fmt.Sprintf("signal %d", status.Signal())
				}
			}
		} else {
			se.resultErr = err
		}
	}
}

