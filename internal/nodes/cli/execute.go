// Package cli provides nodes for executing CLI commands in sandboxed environments.
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
	"github.com/neul-labs/m9m/internal/sandbox"

	// Import bwrap to register it with the sandbox factory
	_ "github.com/neul-labs/m9m/internal/sandbox/bwrap"
)

// ExecuteNode executes CLI commands with optional sandboxing
type ExecuteNode struct {
	*base.BaseNode
	factory sandbox.SandboxFactory
}

// NewExecuteNode creates a new CLI execute node
func NewExecuteNode() *ExecuteNode {
	return &ExecuteNode{
		BaseNode: base.NewBaseNode(base.NodeDescription{
			Name:        "CLI Execute",
			Description: "Execute CLI commands with optional sandboxing",
			Category:    "cli",
		}),
		factory: sandbox.NewFactory(),
	}
}

// ValidateParameters validates the node parameters
func (n *ExecuteNode) ValidateParameters(params map[string]interface{}) error {
	if params == nil {
		return fmt.Errorf("parameters required")
	}

	command, ok := params["command"].(string)
	if !ok || command == "" {
		return fmt.Errorf("command parameter is required")
	}

	// Validate isolation level if provided
	if level, ok := params["isolationLevel"].(string); ok {
		validLevels := map[string]bool{
			"none": true, "minimal": true, "standard": true, "strict": true, "paranoid": true,
		}
		if !validLevels[level] {
			return fmt.Errorf("invalid isolation level: %s", level)
		}
	}

	// Validate network access if provided
	if network, ok := params["networkAccess"].(string); ok {
		validNetworks := map[string]bool{
			"host": true, "isolated": true, "loopback": true,
		}
		if !validNetworks[network] {
			return fmt.Errorf("invalid network access: %s", network)
		}
	}

	// Validate output format if provided
	if format, ok := params["outputFormat"].(string); ok {
		validFormats := map[string]bool{
			"text": true, "json": true, "lines": true,
		}
		if !validFormats[format] {
			return fmt.Errorf("invalid output format: %s", format)
		}
	}

	return nil
}

// Execute runs the CLI command
func (n *ExecuteNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	if err := n.ValidateParameters(nodeParams); err != nil {
		return nil, err
	}

	// Extract parameters
	command := n.GetStringParameter(nodeParams, "command", "")
	args := n.getStringArrayParameter(nodeParams, "args")
	envMap := n.getMapParameter(nodeParams, "env")
	workDir := n.GetStringParameter(nodeParams, "workDir", "")
	useShell := n.GetBoolParameter(nodeParams, "shell", false)
	sandboxEnabled := n.GetBoolParameter(nodeParams, "sandboxEnabled", true)
	isolationLevel := n.GetStringParameter(nodeParams, "isolationLevel", "standard")
	networkAccess := n.GetStringParameter(nodeParams, "networkAccess", "host")
	timeoutSecs := n.GetIntParameter(nodeParams, "timeout", 60)
	maxMemoryMB := n.GetIntParameter(nodeParams, "maxMemoryMB", 512)
	inputFromPrevious := n.GetBoolParameter(nodeParams, "inputFromPrevious", false)
	outputFormat := n.GetStringParameter(nodeParams, "outputFormat", "text")
	additionalMounts := n.getMountsParameter(nodeParams, "additionalMounts")

	// If shell mode, wrap command
	if useShell {
		args = []string{"-c", command + " " + strings.Join(args, " ")}
		command = "/bin/sh"
	}

	// Create sandbox
	var sb sandbox.Sandbox
	var err error

	if sandboxEnabled {
		sb, err = n.factory.DetectBest()
		if err != nil {
			return nil, fmt.Errorf("failed to create sandbox: %w", err)
		}
	} else {
		sb = sandbox.NewNoSandbox()
	}
	defer sb.Cleanup()

	// Build sandbox config
	config := sandbox.NewSandboxConfig(sandbox.IsolationLevelFromString(isolationLevel))
	config.NetworkMode = sandbox.NetworkModeFromString(networkAccess)
	config.WorkDir = workDir
	config.Limits.ExecutionTimeout = time.Duration(timeoutSecs) * time.Second
	config.Limits.MaxMemoryBytes = int64(maxMemoryMB) * 1024 * 1024

	// Add environment variables
	for k, v := range envMap {
		if str, ok := v.(string); ok {
			config.EnvVars = append(config.EnvVars, fmt.Sprintf("%s=%s", k, str))
		}
	}

	// Add additional mounts
	for _, m := range additionalMounts {
		config.Mounts = append(config.Mounts, m)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.Limits.ExecutionTimeout)
	defer cancel()

	// Handle streaming vs one-shot execution
	if inputFromPrevious && len(inputData) > 0 {
		// Use streaming execution to send input
		return n.executeStreaming(ctx, sb, config, command, args, inputData, outputFormat)
	}

	// One-shot execution
	return n.executeOneShot(ctx, sb, config, command, args, outputFormat)
}

func (n *ExecuteNode) executeOneShot(
	ctx context.Context,
	sb sandbox.Sandbox,
	config *sandbox.SandboxConfig,
	command string,
	args []string,
	outputFormat string,
) ([]model.DataItem, error) {
	result, err := sb.Execute(ctx, config, command, args...)
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	return n.buildOutput(result, outputFormat)
}

func (n *ExecuteNode) executeStreaming(
	ctx context.Context,
	sb sandbox.Sandbox,
	config *sandbox.SandboxConfig,
	command string,
	args []string,
	inputData []model.DataItem,
	outputFormat string,
) ([]model.DataItem, error) {
	execution, err := sb.ExecuteStreaming(ctx, config, command, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to start streaming execution: %w", err)
	}

	// Send input data as JSON to stdin
	inputJSON, err := json.Marshal(inputData)
	if err != nil {
		execution.Kill()
		return nil, fmt.Errorf("failed to marshal input data: %w", err)
	}

	go func() {
		defer execution.Stdin().Close()
		execution.Stdin().Write(inputJSON)
	}()

	// Read stdout
	var stdout bytes.Buffer
	go func() {
		io.Copy(&stdout, execution.Stdout())
	}()

	// Read stderr
	var stderr bytes.Buffer
	if stderrReader := execution.Stderr(); stderrReader != nil {
		go func() {
			io.Copy(&stderr, stderrReader)
		}()
	}

	// Wait for completion
	result, err := execution.Wait()
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	result.Stdout = stdout.Bytes()
	result.Stderr = stderr.Bytes()

	return n.buildOutput(result, outputFormat)
}

func (n *ExecuteNode) buildOutput(result *sandbox.ExecutionResult, outputFormat string) ([]model.DataItem, error) {
	output := make(map[string]interface{})

	// Always include exit code and metadata
	output["exitCode"] = result.ExitCode
	output["killed"] = result.Killed
	output["killReason"] = result.KillReason
	output["duration"] = result.Duration.Milliseconds()
	output["stderr"] = string(result.Stderr)

	// Resource usage
	if result.CPUTime > 0 {
		output["cpuTime"] = result.CPUTime.Milliseconds()
	}
	if result.MaxMemory > 0 {
		output["maxMemory"] = result.MaxMemory
	}

	// Format stdout based on output format
	stdoutStr := string(result.Stdout)

	switch outputFormat {
	case "json":
		var jsonOutput interface{}
		if err := json.Unmarshal(result.Stdout, &jsonOutput); err != nil {
			// If JSON parsing fails, return as string
			output["stdout"] = stdoutStr
			output["parseError"] = err.Error()
		} else {
			output["stdout"] = jsonOutput
		}

	case "lines":
		lines := strings.Split(strings.TrimSuffix(stdoutStr, "\n"), "\n")
		output["stdout"] = lines

	default: // "text"
		output["stdout"] = stdoutStr
	}

	return []model.DataItem{
		{JSON: output},
	}, nil
}

// Helper methods

func (n *ExecuteNode) getStringArrayParameter(params map[string]interface{}, name string) []string {
	if params == nil {
		return nil
	}

	value, ok := params[name]
	if !ok {
		return nil
	}

	switch v := value.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	default:
		return nil
	}
}

func (n *ExecuteNode) getMapParameter(params map[string]interface{}, name string) map[string]interface{} {
	if params == nil {
		return nil
	}

	value, ok := params[name]
	if !ok {
		return nil
	}

	if m, ok := value.(map[string]interface{}); ok {
		return m
	}

	return nil
}

func (n *ExecuteNode) getMountsParameter(params map[string]interface{}, name string) []sandbox.Mount {
	if params == nil {
		return nil
	}

	value, ok := params[name]
	if !ok {
		return nil
	}

	mounts, ok := value.([]interface{})
	if !ok {
		return nil
	}

	result := make([]sandbox.Mount, 0, len(mounts))
	for _, item := range mounts {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		source, _ := m["source"].(string)
		dest, _ := m["destination"].(string)
		readWrite, _ := m["readWrite"].(bool)

		if source == "" || dest == "" {
			continue
		}

		mountType := sandbox.MountReadOnly
		if readWrite {
			mountType = sandbox.MountReadWrite
		}

		// Validate source exists
		if _, err := os.Stat(source); err != nil {
			continue
		}

		result = append(result, sandbox.Mount{
			Type:        mountType,
			Source:      source,
			Destination: dest,
		})
	}

	return result
}
