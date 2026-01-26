package cli

import (
	"testing"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExecuteNode(t *testing.T) {
	node := NewExecuteNode()

	assert.NotNil(t, node)
	assert.Equal(t, "CLI Execute", node.Description().Name)
	assert.Equal(t, "cli", node.Description().Category)
}

func TestExecuteNode_ValidateParameters(t *testing.T) {
	node := NewExecuteNode()

	tests := []struct {
		name      string
		params    map[string]interface{}
		expectErr bool
	}{
		{
			name:      "nil params",
			params:    nil,
			expectErr: true,
		},
		{
			name:      "empty params",
			params:    map[string]interface{}{},
			expectErr: true,
		},
		{
			name: "missing command",
			params: map[string]interface{}{
				"args": []string{"foo"},
			},
			expectErr: true,
		},
		{
			name: "valid command",
			params: map[string]interface{}{
				"command": "echo",
			},
			expectErr: false,
		},
		{
			name: "invalid isolation level",
			params: map[string]interface{}{
				"command":        "echo",
				"isolationLevel": "invalid",
			},
			expectErr: true,
		},
		{
			name: "valid isolation level",
			params: map[string]interface{}{
				"command":        "echo",
				"isolationLevel": "standard",
			},
			expectErr: false,
		},
		{
			name: "invalid network access",
			params: map[string]interface{}{
				"command":       "echo",
				"networkAccess": "invalid",
			},
			expectErr: true,
		},
		{
			name: "valid network access",
			params: map[string]interface{}{
				"command":       "echo",
				"networkAccess": "host",
			},
			expectErr: false,
		},
		{
			name: "invalid output format",
			params: map[string]interface{}{
				"command":      "echo",
				"outputFormat": "invalid",
			},
			expectErr: true,
		},
		{
			name: "valid output format",
			params: map[string]interface{}{
				"command":      "echo",
				"outputFormat": "json",
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := node.ValidateParameters(tt.params)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExecuteNode_Execute_SimpleEcho(t *testing.T) {
	node := NewExecuteNode()

	params := map[string]interface{}{
		"command":        "echo",
		"args":           []interface{}{"hello", "world"},
		"sandboxEnabled": false, // Disable sandbox for simple test
		"outputFormat":   "text",
		"timeout":        30,
	}

	inputData := []model.DataItem{}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, result, 1)

	output := result[0].JSON
	assert.Equal(t, 0, output["exitCode"])
	assert.Equal(t, "hello world\n", output["stdout"])
	assert.Equal(t, false, output["killed"])
}

func TestExecuteNode_Execute_JSONOutput(t *testing.T) {
	node := NewExecuteNode()

	params := map[string]interface{}{
		"command":        "echo",
		"args":           []interface{}{`{"key": "value"}`},
		"sandboxEnabled": false,
		"outputFormat":   "json",
		"timeout":        30,
	}

	inputData := []model.DataItem{}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, result, 1)

	output := result[0].JSON
	assert.Equal(t, 0, output["exitCode"])

	// Check parsed JSON
	stdout := output["stdout"].(map[string]interface{})
	assert.Equal(t, "value", stdout["key"])
}

func TestExecuteNode_Execute_LinesOutput(t *testing.T) {
	node := NewExecuteNode()

	params := map[string]interface{}{
		"command":        "sh",
		"args":           []interface{}{"-c", "echo line1; echo line2; echo line3"},
		"sandboxEnabled": false,
		"outputFormat":   "lines",
		"timeout":        30,
	}

	inputData := []model.DataItem{}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, result, 1)

	output := result[0].JSON
	assert.Equal(t, 0, output["exitCode"])

	lines := output["stdout"].([]string)
	assert.Equal(t, []string{"line1", "line2", "line3"}, lines)
}

func TestExecuteNode_Execute_NonZeroExitCode(t *testing.T) {
	node := NewExecuteNode()

	params := map[string]interface{}{
		"command":        "sh",
		"args":           []interface{}{"-c", "exit 42"},
		"sandboxEnabled": false,
		"timeout":        30,
	}

	inputData := []model.DataItem{}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, result, 1)

	output := result[0].JSON
	assert.Equal(t, 42, output["exitCode"])
}

func TestExecuteNode_Execute_WithEnv(t *testing.T) {
	node := NewExecuteNode()

	params := map[string]interface{}{
		"command": "sh",
		"args":    []interface{}{"-c", "echo $TEST_VAR"},
		"env": map[string]interface{}{
			"TEST_VAR": "test_value",
		},
		"sandboxEnabled": false,
		"timeout":        30,
	}

	inputData := []model.DataItem{}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, result, 1)

	output := result[0].JSON
	assert.Equal(t, 0, output["exitCode"])
	assert.Equal(t, "test_value\n", output["stdout"])
}

func TestExecuteNode_Execute_ShellMode(t *testing.T) {
	node := NewExecuteNode()

	params := map[string]interface{}{
		"command":        "echo hello && echo world",
		"shell":          true,
		"sandboxEnabled": false,
		"outputFormat":   "lines",
		"timeout":        30,
	}

	inputData := []model.DataItem{}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, result, 1)

	output := result[0].JSON
	assert.Equal(t, 0, output["exitCode"])

	lines := output["stdout"].([]string)
	assert.Equal(t, []string{"hello", "world"}, lines)
}
