package sandbox

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSandboxConfig(t *testing.T) {
	tests := []struct {
		name          string
		level         IsolationLevel
		expectNetwork NetworkMode
	}{
		{
			name:          "none isolation",
			level:         IsolationNone,
			expectNetwork: NetworkHost,
		},
		{
			name:          "minimal isolation",
			level:         IsolationMinimal,
			expectNetwork: NetworkHost,
		},
		{
			name:          "standard isolation",
			level:         IsolationStandard,
			expectNetwork: NetworkHost,
		},
		{
			name:          "strict isolation",
			level:         IsolationStrict,
			expectNetwork: NetworkIsolated,
		},
		{
			name:          "paranoid isolation",
			level:         IsolationParanoid,
			expectNetwork: NetworkIsolated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewSandboxConfig(tt.level)

			assert.Equal(t, tt.level, config.IsolationLevel)
			assert.Equal(t, tt.expectNetwork, config.NetworkMode)
			assert.True(t, config.NoNewPrivileges)
			assert.NotEmpty(t, config.Mounts)
		})
	}
}

func TestDefaultResourceLimits(t *testing.T) {
	limits := DefaultResourceLimits()

	assert.Equal(t, 5*time.Minute, limits.MaxCPUTime)
	assert.Equal(t, int64(512*1024*1024), limits.MaxMemoryBytes)
	assert.Equal(t, int64(100*1024*1024), limits.MaxFileSize)
	assert.Equal(t, 50, limits.MaxProcesses)
	assert.Equal(t, 1024, limits.MaxOpenFiles)
	assert.Equal(t, int64(10*1024*1024), limits.MaxOutputBytes)
	assert.Equal(t, 10*time.Minute, limits.ExecutionTimeout)
}

func TestDefaultSeccompProfile(t *testing.T) {
	profile := DefaultSeccompProfile()

	assert.Equal(t, "default-cli", profile.Name)
	assert.Equal(t, "allow", profile.DefaultAction)
	assert.NotEmpty(t, profile.Syscalls)

	// Verify some dangerous syscalls are blocked
	blockedSyscalls := make(map[string]bool)
	for _, rule := range profile.Syscalls {
		if rule.Action == "deny" {
			for _, name := range rule.Names {
				blockedSyscalls[name] = true
			}
		}
	}

	assert.True(t, blockedSyscalls["reboot"])
	assert.True(t, blockedSyscalls["mount"])
	assert.True(t, blockedSyscalls["ptrace"])
}

func TestIsolationLevelFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected IsolationLevel
	}{
		{"none", IsolationNone},
		{"minimal", IsolationMinimal},
		{"standard", IsolationStandard},
		{"strict", IsolationStrict},
		{"paranoid", IsolationParanoid},
		{"invalid", IsolationStandard}, // Default
		{"", IsolationStandard},        // Default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := IsolationLevelFromString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNetworkModeFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected NetworkMode
	}{
		{"host", NetworkHost},
		{"isolated", NetworkIsolated},
		{"loopback", NetworkLoopback},
		{"invalid", NetworkHost}, // Default
		{"", NetworkHost},        // Default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NetworkModeFromString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNoSandbox_Execute(t *testing.T) {
	sb := NewNoSandbox()

	assert.Equal(t, SandboxTypeNone, sb.Type())
	assert.NoError(t, sb.Available())
	assert.NoError(t, sb.Validate(nil))
	assert.NoError(t, sb.Cleanup())

	// Test simple command execution
	ctx := context.Background()
	config := NewSandboxConfig(IsolationNone)

	result, err := sb.Execute(ctx, config, "echo", "hello", "world")
	require.NoError(t, err)

	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "hello world\n", string(result.Stdout))
	assert.False(t, result.Killed)
}

func TestNoSandbox_ExecuteWithTimeout(t *testing.T) {
	sb := NewNoSandbox()

	ctx := context.Background()
	config := NewSandboxConfig(IsolationNone)
	config.Limits.ExecutionTimeout = 100 * time.Millisecond

	result, err := sb.Execute(ctx, config, "sleep", "10")
	require.NoError(t, err)

	assert.True(t, result.Killed)
	// Kill reason can be "timeout" or "signal 9" depending on how the process was terminated
	assert.True(t,
		result.KillReason == "timeout" || result.KillReason == "signal 9",
		"Expected 'timeout' or 'signal 9', got '%s'", result.KillReason,
	)
}

func TestNoSandbox_ExecuteStreaming(t *testing.T) {
	sb := NewNoSandbox()

	ctx := context.Background()
	config := NewSandboxConfig(IsolationNone)

	execution, err := sb.ExecuteStreaming(ctx, config, "cat")
	require.NoError(t, err)

	// Write to stdin
	_, err = execution.Stdin().Write([]byte("hello\n"))
	require.NoError(t, err)
	execution.Stdin().Close()

	// Read from stdout
	buf := make([]byte, 100)
	n, err := execution.Stdout().Read(buf)
	require.NoError(t, err)
	assert.Equal(t, "hello\n", string(buf[:n]))

	// Wait for completion
	result, err := execution.Wait()
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
}

func TestFactory_Create(t *testing.T) {
	factory := NewFactory()

	// Create NoSandbox
	sb, err := factory.Create(SandboxTypeNone)
	require.NoError(t, err)
	assert.Equal(t, SandboxTypeNone, sb.Type())

	// Create unknown type
	_, err = factory.Create(SandboxType("unknown"))
	assert.Error(t, err)
}

func TestFactory_ListAvailable(t *testing.T) {
	factory := NewFactory()

	available := factory.ListAvailable()

	// NoSandbox should always be available
	assert.Contains(t, available, SandboxTypeNone)
}

func TestFactory_DetectBest(t *testing.T) {
	factory := NewFactory()

	sb, err := factory.DetectBest()
	require.NoError(t, err)
	require.NotNil(t, sb)

	// Should be either NoSandbox or Bubblewrap depending on platform
	assert.True(t,
		sb.Type() == SandboxTypeNone || sb.Type() == SandboxTypeBubblewrap,
		"Expected NoSandbox or Bubblewrap, got %s", sb.Type(),
	)
}

func TestDefaultMounts(t *testing.T) {
	tests := []struct {
		name     string
		level    IsolationLevel
		expectEtc bool
	}{
		{
			name:      "minimal includes etc",
			level:     IsolationMinimal,
			expectEtc: true,
		},
		{
			name:      "standard includes etc",
			level:     IsolationStandard,
			expectEtc: true,
		},
		{
			name:      "paranoid excludes etc",
			level:     IsolationParanoid,
			expectEtc: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mounts := DefaultMounts(tt.level)

			hasEtc := false
			for _, m := range mounts {
				if m.Destination == "/etc" {
					hasEtc = true
					break
				}
			}

			assert.Equal(t, tt.expectEtc, hasEtc)
		})
	}
}
