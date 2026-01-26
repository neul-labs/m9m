package bwrap

import (
	"os"

	"github.com/neul-labs/m9m/internal/sandbox"
)

// CommandBuilder builds bwrap command arguments
type CommandBuilder struct {
	bwrapPath string
	tmpFiles  []string // Files to clean up
}

// NewCommandBuilder creates a new command builder
func NewCommandBuilder(bwrapPath string) *CommandBuilder {
	return &CommandBuilder{
		bwrapPath: bwrapPath,
		tmpFiles:  make([]string, 0),
	}
}

// Build constructs the bwrap arguments from config
func (cb *CommandBuilder) Build(config *sandbox.SandboxConfig) ([]string, func(), error) {
	var args []string

	// Unshare namespaces based on isolation level
	args = append(args, cb.buildNamespaceArgs(config)...)

	// Add mount arguments
	args = append(args, cb.buildMountArgs(config)...)

	// Add user/group configuration
	args = append(args, cb.buildUserArgs(config)...)

	// Add network configuration
	args = append(args, cb.buildNetworkArgs(config)...)

	// Add security arguments
	args = append(args, cb.buildSecurityArgs(config)...)

	// Add working directory
	if config.WorkDir != "" {
		args = append(args, "--chdir", config.WorkDir)
	}

	// Die with parent - ensure child dies if bwrap dies
	args = append(args, "--die-with-parent")

	// Create new session for better signal handling
	args = append(args, "--new-session")

	cleanup := func() {
		for _, f := range cb.tmpFiles {
			os.Remove(f)
		}
	}

	return args, cleanup, nil
}

func (cb *CommandBuilder) buildNamespaceArgs(config *sandbox.SandboxConfig) []string {
	var args []string

	switch config.IsolationLevel {
	case sandbox.IsolationParanoid, sandbox.IsolationStrict:
		// Unshare all namespaces
		args = append(args, "--unshare-all")
		// Re-share network if not isolated
		if config.NetworkMode == sandbox.NetworkHost {
			args = append(args, "--share-net")
		}
	case sandbox.IsolationStandard:
		// Unshare PID, IPC, and UTS namespaces
		args = append(args, "--unshare-pid")
		args = append(args, "--unshare-ipc")
		args = append(args, "--unshare-uts")
	case sandbox.IsolationMinimal:
		// Only basic isolation, no namespace changes
	case sandbox.IsolationNone:
		// No isolation
	}

	// User namespace if requested
	if config.NewUserNS {
		args = append(args, "--unshare-user")
	}

	return args
}

func (cb *CommandBuilder) buildMountArgs(config *sandbox.SandboxConfig) []string {
	var args []string

	for _, mount := range config.Mounts {
		switch mount.Type {
		case sandbox.MountReadOnly:
			args = append(args, "--ro-bind", mount.Source, mount.Destination)
		case sandbox.MountReadWrite:
			args = append(args, "--bind", mount.Source, mount.Destination)
		case sandbox.MountTmpfs:
			args = append(args, "--tmpfs", mount.Destination)
		case sandbox.MountDevNull:
			args = append(args, "--ro-bind", "/dev/null", mount.Destination)
		case sandbox.MountProc:
			args = append(args, "--proc", mount.Destination)
		case sandbox.MountDev:
			args = append(args, "--dev", mount.Destination)
		}
	}

	return args
}

func (cb *CommandBuilder) buildUserArgs(config *sandbox.SandboxConfig) []string {
	var args []string

	if config.UID != nil {
		args = append(args, "--uid", itoa(*config.UID))
	}
	if config.GID != nil {
		args = append(args, "--gid", itoa(*config.GID))
	}

	return args
}

func (cb *CommandBuilder) buildNetworkArgs(config *sandbox.SandboxConfig) []string {
	var args []string

	switch config.NetworkMode {
	case sandbox.NetworkIsolated:
		// Already handled by --unshare-all or add explicitly
		if config.IsolationLevel < sandbox.IsolationStrict {
			args = append(args, "--unshare-net")
		}
	case sandbox.NetworkLoopback:
		// Unshare network but loopback is automatically available
		if config.IsolationLevel < sandbox.IsolationStrict {
			args = append(args, "--unshare-net")
		}
	case sandbox.NetworkHost:
		// Default - no special args needed unless we unshared all
	}

	return args
}

func (cb *CommandBuilder) buildSecurityArgs(config *sandbox.SandboxConfig) []string {
	var args []string

	// Drop capabilities
	for _, cap := range config.DropCapabilities {
		args = append(args, "--cap-drop", cap)
	}

	// Note: seccomp support requires passing a BPF program via --seccomp
	// This is more complex and would require generating the BPF bytecode
	// For now, we rely on other isolation mechanisms

	return args
}

// itoa converts int to string without importing strconv
func itoa(i int) string {
	if i == 0 {
		return "0"
	}

	negative := false
	if i < 0 {
		negative = true
		i = -i
	}

	var b [20]byte
	pos := len(b)

	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}

	if negative {
		pos--
		b[pos] = '-'
	}

	return string(b[pos:])
}
