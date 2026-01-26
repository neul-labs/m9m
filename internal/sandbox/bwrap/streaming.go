package bwrap

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/neul-labs/m9m/internal/sandbox"
)

// StreamingExecution implements sandbox.StreamingExecution for bubblewrap
type StreamingExecution struct {
	cmd         *exec.Cmd
	ptyFile     *os.File
	stdin       io.WriteCloser
	stdout      io.Reader
	stderr      io.Reader
	cgroupPath  string
	cgroupMgr   *CgroupManager
	cleanup     func()
	startTime   time.Time
	done        chan struct{}
	result      *sandbox.ExecutionResult
	resultErr   error
	mu          sync.Mutex
	usePTY      bool
}

// NewStreamingExecution creates a new streaming execution
func NewStreamingExecution(
	ctx context.Context,
	sandbox *BubblewrapSandbox,
	config *sandbox.SandboxConfig,
	command string,
	args []string,
) (*StreamingExecution, error) {
	// Build bwrap command
	builder := NewCommandBuilder(sandbox.bwrapPath)
	bwrapArgs, cleanup, err := builder.Build(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build bwrap command: %w", err)
	}

	// Add the actual command and args
	bwrapArgs = append(bwrapArgs, command)
	bwrapArgs = append(bwrapArgs, args...)

	// Create command
	cmd := exec.CommandContext(ctx, sandbox.bwrapPath, bwrapArgs...)

	// Set environment
	cmd.Env = sandbox.buildEnv(config)

	se := &StreamingExecution{
		cmd:       cmd,
		cleanup:   cleanup,
		cgroupMgr: sandbox.cgroupManager,
		done:      make(chan struct{}),
		usePTY:    config.TTY,
	}

	// Create cgroup for resource limits
	if config.Limits.MaxMemoryBytes > 0 || config.Limits.MaxProcesses > 0 {
		cgroupPath, err := sandbox.cgroupManager.CreateCgroup(config.Limits)
		if err != nil {
			// Log warning but continue without cgroup limits
			fmt.Fprintf(os.Stderr, "warning: failed to create cgroup: %v\n", err)
		} else {
			se.cgroupPath = cgroupPath
		}
	}

	// Set up I/O based on TTY mode
	if config.TTY {
		if err := se.setupPTY(); err != nil {
			se.cleanupResources()
			return nil, err
		}
	} else {
		if err := se.setupPipes(); err != nil {
			se.cleanupResources()
			return nil, err
		}
	}

	// Start the command
	if err := se.start(); err != nil {
		se.cleanupResources()
		return nil, err
	}

	return se, nil
}

func (se *StreamingExecution) setupPTY() error {
	// Start command with a PTY
	ptmx, err := pty.Start(se.cmd)
	if err != nil {
		return fmt.Errorf("failed to start with PTY: %w", err)
	}

	se.ptyFile = ptmx
	se.stdin = ptmx
	se.stdout = ptmx
	se.stderr = nil // PTY combines stdout/stderr

	return nil
}

func (se *StreamingExecution) setupPipes() error {
	stdinPipe, err := se.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	se.stdin = stdinPipe

	stdoutPipe, err := se.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	se.stdout = stdoutPipe

	stderrPipe, err := se.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	se.stderr = stderrPipe

	return nil
}

func (se *StreamingExecution) start() error {
	se.startTime = time.Now()

	// If using PTY, it's already started via pty.Start
	if !se.usePTY {
		if err := se.cmd.Start(); err != nil {
			return fmt.Errorf("failed to start sandboxed process: %w", err)
		}
	}

	// Move process to cgroup if applicable
	if se.cgroupPath != "" && se.cmd.Process != nil {
		if err := se.cgroupMgr.AddProcess(se.cgroupPath, se.cmd.Process.Pid); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to add process to cgroup: %v\n", err)
		}
	}

	// Start background waiter
	go se.waitLoop()

	return nil
}

func (se *StreamingExecution) waitLoop() {
	defer close(se.done)

	err := se.cmd.Wait()
	endTime := time.Now()

	se.mu.Lock()
	defer se.mu.Unlock()

	se.result = &sandbox.ExecutionResult{
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

	// Get resource usage from cgroup
	if se.cgroupPath != "" {
		stats, _ := se.cgroupMgr.GetStats(se.cgroupPath)
		if stats != nil {
			se.result.CPUTime = stats.CPUTime
			se.result.MaxMemory = stats.MaxMemory
		}
	}

	// Cleanup resources
	se.cleanupResources()
}

func (se *StreamingExecution) cleanupResources() {
	if se.ptyFile != nil {
		se.ptyFile.Close()
	}
	if se.cgroupPath != "" {
		se.cgroupMgr.RemoveCgroup(se.cgroupPath)
	}
	if se.cleanup != nil {
		se.cleanup()
	}
}

// Stdin returns a writer for sending input to the process
func (se *StreamingExecution) Stdin() io.WriteCloser {
	return se.stdin
}

// Stdout returns a reader for receiving stdout from the process
func (se *StreamingExecution) Stdout() io.Reader {
	return se.stdout
}

// Stderr returns a reader for receiving stderr from the process
// Returns nil in PTY mode as stderr is combined with stdout
func (se *StreamingExecution) Stderr() io.Reader {
	return se.stderr
}

// Wait waits for the execution to complete and returns the result
func (se *StreamingExecution) Wait() (*sandbox.ExecutionResult, error) {
	<-se.done
	se.mu.Lock()
	defer se.mu.Unlock()
	return se.result, se.resultErr
}

// Signal sends a signal to the sandboxed process
func (se *StreamingExecution) Signal(sig os.Signal) error {
	if se.cmd.Process == nil {
		return fmt.Errorf("process not started")
	}
	return se.cmd.Process.Signal(sig)
}

// Kill forcefully terminates the execution
func (se *StreamingExecution) Kill() error {
	if se.cmd.Process == nil {
		return fmt.Errorf("process not started")
	}
	return se.cmd.Process.Kill()
}

// Resize resizes the pseudo-TTY
func (se *StreamingExecution) Resize(cols, rows int) error {
	if se.ptyFile == nil {
		return nil // No-op for non-TTY mode
	}
	return pty.Setsize(se.ptyFile, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
}

// PID returns the process ID of the sandboxed process
func (se *StreamingExecution) PID() int {
	if se.cmd.Process == nil {
		return 0
	}
	return se.cmd.Process.Pid
}
