// Package sandbox provides a platform-abstracted interface for sandboxed CLI execution.
package sandbox

import (
	"context"
	"io"
	"os"
	"time"
)

// SandboxType identifies the sandboxing backend
type SandboxType string

const (
	SandboxTypeBubblewrap SandboxType = "bwrap"
	SandboxTypeNone       SandboxType = "none"
)

// IsolationLevel defines the degree of isolation
type IsolationLevel int

const (
	IsolationNone     IsolationLevel = iota // No isolation (development only)
	IsolationMinimal                        // Basic filesystem isolation
	IsolationStandard                       // + PID namespace, resource limits
	IsolationStrict                         // + Network isolation, seccomp
	IsolationParanoid                       // + No access to host filesystem
)

// NetworkMode defines network isolation behavior
type NetworkMode int

const (
	NetworkHost     NetworkMode = iota // Full host network access
	NetworkIsolated                    // No network access
	NetworkLoopback                    // Loopback only
)

// MountType defines filesystem mount types
type MountType int

const (
	MountReadOnly  MountType = iota // Read-only bind mount
	MountReadWrite                  // Read-write bind mount
	MountTmpfs                      // Temporary in-memory filesystem
	MountDevNull                    // Mount /dev/null
	MountProc                       // Mount /proc
	MountDev                        // Mount /dev
)

// Mount represents a filesystem mount configuration
type Mount struct {
	Type        MountType
	Source      string // Host path (empty for tmpfs)
	Destination string // Path inside sandbox
	Options     []string
}

// ResourceLimits defines resource constraints for sandboxed execution
type ResourceLimits struct {
	MaxCPUTime       time.Duration // CPU time limit
	MaxMemoryBytes   int64         // Memory limit in bytes
	MaxFileSize      int64         // Max file size that can be created
	MaxProcesses     int           // Max number of processes
	MaxOpenFiles     int           // Max open file descriptors
	MaxOutputBytes   int64         // Max stdout/stderr combined
	ExecutionTimeout time.Duration // Wall-clock timeout
}

// SeccompProfile defines syscall filtering rules
type SeccompProfile struct {
	Name          string        // Profile name
	DefaultAction string        // "allow", "deny", "log"
	Syscalls      []SyscallRule // Specific syscall rules
}

// SyscallRule defines a rule for a specific syscall
type SyscallRule struct {
	Names  []string // Syscall names
	Action string   // "allow", "deny", "errno", "trace"
	Errno  int      // Error number for "errno" action
}

// SandboxConfig holds the complete sandbox configuration
type SandboxConfig struct {
	// Isolation settings
	IsolationLevel IsolationLevel
	NetworkMode    NetworkMode

	// Filesystem configuration
	Mounts     []Mount  // Bind mounts and special mounts
	EnvVars    []string // Environment variables (KEY=VALUE format)
	EnvInherit []string // Environment variable names to inherit from host
	WorkDir    string   // Working directory inside sandbox

	// User/group configuration
	UID       *int // User ID inside sandbox (nil = same as host)
	GID       *int // Group ID inside sandbox
	NewUserNS bool // Create new user namespace

	// Resource limits
	Limits ResourceLimits

	// Security configuration
	SeccompProfile   *SeccompProfile // Syscall filtering
	NoNewPrivileges  bool            // Prevent privilege escalation
	DropCapabilities []string        // Linux capabilities to drop

	// Execution configuration
	TTY             bool // Allocate pseudo-TTY
	PreserveSignals bool // Forward signals to sandboxed process
}

// ExecutionResult contains the result of a sandboxed execution
type ExecutionResult struct {
	ExitCode   int           // Process exit code
	Stdout     []byte        // Captured stdout (for one-shot mode)
	Stderr     []byte        // Captured stderr
	StartTime  time.Time     // When execution started
	EndTime    time.Time     // When execution completed
	Duration   time.Duration // Total execution time
	Killed     bool          // Whether process was killed (timeout, OOM, etc.)
	KillReason string        // Reason for kill if applicable

	// Resource usage statistics
	CPUTime   time.Duration
	MaxMemory int64
	IORead    int64
	IOWrite   int64
}

// StreamingExecution represents an ongoing interactive execution
type StreamingExecution interface {
	// Stdin returns a writer for sending input to the process
	Stdin() io.WriteCloser

	// Stdout returns a reader for receiving stdout from the process
	Stdout() io.Reader

	// Stderr returns a reader for receiving stderr from the process
	Stderr() io.Reader

	// Wait waits for the execution to complete and returns the result
	Wait() (*ExecutionResult, error)

	// Signal sends a signal to the sandboxed process
	Signal(sig os.Signal) error

	// Kill forcefully terminates the execution
	Kill() error

	// Resize resizes the pseudo-TTY (if TTY mode is enabled)
	Resize(cols, rows int) error

	// PID returns the process ID of the sandboxed process
	PID() int
}

// Sandbox is the main interface for sandbox implementations
type Sandbox interface {
	// Type returns the sandbox backend type
	Type() SandboxType

	// Available checks if this sandbox backend is available on the current system
	Available() error

	// Validate validates the configuration for this sandbox
	Validate(config *SandboxConfig) error

	// Execute runs a command and waits for completion (one-shot mode)
	Execute(ctx context.Context, config *SandboxConfig, command string, args ...string) (*ExecutionResult, error)

	// ExecuteStreaming starts a command with streaming I/O (interactive mode)
	ExecuteStreaming(ctx context.Context, config *SandboxConfig, command string, args ...string) (StreamingExecution, error)

	// Cleanup performs any necessary cleanup
	Cleanup() error
}

// SandboxFactory creates sandbox instances
type SandboxFactory interface {
	// Create creates a new sandbox with the given type
	Create(sandboxType SandboxType) (Sandbox, error)

	// DetectBest detects and returns the best available sandbox for the current platform
	DetectBest() (Sandbox, error)

	// ListAvailable returns all available sandbox backends
	ListAvailable() []SandboxType
}

// DefaultResourceLimits returns sensible default resource limits
func DefaultResourceLimits() ResourceLimits {
	return ResourceLimits{
		MaxCPUTime:       5 * time.Minute,
		MaxMemoryBytes:   512 * 1024 * 1024, // 512MB
		MaxFileSize:      100 * 1024 * 1024, // 100MB
		MaxProcesses:     50,
		MaxOpenFiles:     1024,
		MaxOutputBytes:   10 * 1024 * 1024, // 10MB
		ExecutionTimeout: 10 * time.Minute,
	}
}

// DefaultSeccompProfile returns a default seccomp profile for CLI tools
func DefaultSeccompProfile() *SeccompProfile {
	return &SeccompProfile{
		Name:          "default-cli",
		DefaultAction: "allow",
		Syscalls: []SyscallRule{
			// Block dangerous syscalls
			{Names: []string{"reboot", "kexec_load", "kexec_file_load"}, Action: "deny"},
			{Names: []string{"mount", "umount", "umount2"}, Action: "deny"},
			{Names: []string{"pivot_root", "chroot"}, Action: "deny"},
			{Names: []string{"init_module", "finit_module", "delete_module"}, Action: "deny"},
			{Names: []string{"acct", "settimeofday", "stime", "clock_settime"}, Action: "deny"},
			{Names: []string{"ptrace"}, Action: "deny"},
			{Names: []string{"setns"}, Action: "deny"},
		},
	}
}

// NewSandboxConfig creates a new sandbox config with defaults for the given isolation level
func NewSandboxConfig(level IsolationLevel) *SandboxConfig {
	config := &SandboxConfig{
		IsolationLevel:  level,
		NetworkMode:     NetworkHost,
		Limits:          DefaultResourceLimits(),
		NoNewPrivileges: true,
		EnvInherit:      []string{"PATH", "HOME", "LANG", "LC_ALL", "TERM"},
		Mounts:          DefaultMounts(level),
	}

	// Adjust network mode based on isolation level
	if level >= IsolationStrict {
		config.NetworkMode = NetworkIsolated
		config.SeccompProfile = DefaultSeccompProfile()
	}

	return config
}

// DefaultMounts returns default mounts for a given isolation level
func DefaultMounts(level IsolationLevel) []Mount {
	mounts := []Mount{
		{Type: MountProc, Destination: "/proc"},
		{Type: MountDev, Destination: "/dev"},
		{Type: MountTmpfs, Destination: "/tmp"},
		{Type: MountReadOnly, Source: "/usr", Destination: "/usr"},
		{Type: MountReadOnly, Source: "/bin", Destination: "/bin"},
		{Type: MountReadOnly, Source: "/sbin", Destination: "/sbin"},
	}

	// Add lib directories if they exist
	libDirs := []string{"/lib", "/lib64", "/lib32"}
	for _, dir := range libDirs {
		if _, err := os.Stat(dir); err == nil {
			mounts = append(mounts, Mount{Type: MountReadOnly, Source: dir, Destination: dir})
		}
	}

	// Add /etc for most isolation levels
	if level < IsolationParanoid {
		mounts = append(mounts, Mount{Type: MountReadOnly, Source: "/etc", Destination: "/etc"})
	}

	return mounts
}

// IsolationLevelFromString converts a string to IsolationLevel
func IsolationLevelFromString(s string) IsolationLevel {
	switch s {
	case "none":
		return IsolationNone
	case "minimal":
		return IsolationMinimal
	case "standard":
		return IsolationStandard
	case "strict":
		return IsolationStrict
	case "paranoid":
		return IsolationParanoid
	default:
		return IsolationStandard
	}
}

// NetworkModeFromString converts a string to NetworkMode
func NetworkModeFromString(s string) NetworkMode {
	switch s {
	case "host":
		return NetworkHost
	case "isolated":
		return NetworkIsolated
	case "loopback":
		return NetworkLoopback
	default:
		return NetworkHost
	}
}
