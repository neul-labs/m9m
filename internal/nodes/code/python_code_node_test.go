package code

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/n8n-go/internal/core/interfaces"
)

// MockExecutionParams implements interfaces.ExecutionParams for testing
type MockExecutionParams struct {
	nodeParams map[string]interface{}
	inputData  []interfaces.ItemData
	workflow   interface{}
}

func (m *MockExecutionParams) GetNodeParameter(name string, defaultValue interface{}) interface{} {
	if val, ok := m.nodeParams[name]; ok {
		return val
	}
	return defaultValue
}

func (m *MockExecutionParams) GetInputData() []interfaces.ItemData {
	return m.inputData
}

func (m *MockExecutionParams) GetWorkflow() interface{} {
	return m.workflow
}

func (m *MockExecutionParams) GetCredentials(credType string) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockExecutionParams) GetExecutionID() string {
	return "test-execution-id"
}

func (m *MockExecutionParams) GetNodeID() string {
	return "test-node-id"
}

func TestPythonCodeNode_Execute(t *testing.T) {
	tests := []struct {
		name       string
		pythonCode string
		mode       string
		inputData  []interfaces.ItemData
		expected   []interface{}
		wantError  bool
	}{
		{
			name: "simple calculation",
			pythonCode: `
result = 2 + 2
return result
`,
			mode: "runOnce",
			inputData: []interfaces.ItemData{
				{JSON: map[string]interface{}{"test": "data"}},
			},
			expected:  []interface{}{4},
			wantError: false,
		},
		{
			name: "process array",
			pythonCode: `
result = []
for item in $input:
    item['processed'] = True
    result.append(item)
return result
`,
			mode: "runOnce",
			inputData: []interfaces.ItemData{
				{JSON: map[string]interface{}{"id": 1}},
				{JSON: map[string]interface{}{"id": 2}},
			},
			expected: []interface{}{
				map[string]interface{}{"id": 1, "processed": true},
				map[string]interface{}{"id": 2, "processed": true},
			},
			wantError: false,
		},
		{
			name: "use Python built-ins",
			pythonCode: `
numbers = [1, 2, 3, 4, 5]
result = {
    'sum': sum(numbers),
    'max': max(numbers),
    'min': min(numbers),
    'len': len(numbers),
    'range': list(range(5))
}
return result
`,
			mode:      "runOnce",
			inputData: []interfaces.ItemData{},
			expected: []interface{}{
				map[string]interface{}{
					"sum":   15,
					"max":   5,
					"min":   1,
					"len":   5,
					"range": []interface{}{0, 1, 2, 3, 4},
				},
			},
			wantError: false,
		},
		{
			name: "use Python modules",
			pythonCode: `
json_module = import('json')
math_module = import('math')

result = {
    'json_test': json_module.dumps({'test': 'data'}),
    'pi': math_module.pi,
    'sqrt_16': math_module.sqrt(16)
}
return result
`,
			mode:      "runOnce",
			inputData: []interfaces.ItemData{},
			expected: []interface{}{
				map[string]interface{}{
					"json_test": `{"test":"data"}`,
					"pi":        3.141592653589793,
					"sqrt_16":   4,
				},
			},
			wantError: false,
		},
		{
			name: "run for each item",
			pythonCode: `
result = $input
result['doubled'] = result.get('value', 0) * 2
return result
`,
			mode: "runForEach",
			inputData: []interfaces.ItemData{
				{JSON: map[string]interface{}{"value": 5}, Index: 0},
				{JSON: map[string]interface{}{"value": 10}, Index: 1},
			},
			expected: []interface{}{
				map[string]interface{}{"value": 5, "doubled": 10},
				map[string]interface{}{"value": 10, "doubled": 20},
			},
			wantError: false,
		},
		{
			name: "error handling",
			pythonCode: `
raise Exception("Test error")
`,
			mode:      "runOnce",
			inputData: []interfaces.ItemData{},
			expected:  nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create node
			node := NewPythonCodeNode()
			require.NotNil(t, node)

			// Prepare execution params
			params := &MockExecutionParams{
				nodeParams: map[string]interface{}{
					"pythonCode":     tt.pythonCode,
					"mode":          tt.mode,
					"continueOnFail": false,
				},
				inputData: tt.inputData,
				workflow: map[string]interface{}{
					"name":   "TestWorkflow",
					"active": true,
				},
			}

			// Execute node
			result, err := node.Execute(context.Background(), params)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result.Items)

				// Check results
				if tt.expected != nil {
					assert.Equal(t, len(tt.expected), len(result.Items))
					for i, expected := range tt.expected {
						if i < len(result.Items) {
							actual := result.Items[i].JSON
							if expectedMap, ok := expected.(map[string]interface{}); ok {
								assert.Equal(t, expectedMap, actual)
							} else {
								// Handle primitive values
								assert.Equal(t, expected, actual["value"])
							}
						}
					}
				}
			}
		})
	}
}

func TestPythonCodeNode_Metadata(t *testing.T) {
	node := NewPythonCodeNode()
	metadata := node.GetMetadata()

	assert.Equal(t, "PythonCode", metadata.Name)
	assert.Equal(t, "Python Code", metadata.DisplayName)
	assert.Contains(t, metadata.Description, "Python")
	assert.Contains(t, metadata.Group, "Code")
	assert.Equal(t, 1, metadata.Version)

	// Check properties
	assert.Len(t, metadata.Properties, 3)

	// Check pythonCode property
	codeProperty := metadata.Properties[0]
	assert.Equal(t, "pythonCode", codeProperty.Name)
	assert.Equal(t, "string", codeProperty.Type)
	assert.True(t, codeProperty.Required)

	// Check mode property
	modeProperty := metadata.Properties[1]
	assert.Equal(t, "mode", modeProperty.Name)
	assert.Equal(t, "options", modeProperty.Type)
	assert.Len(t, modeProperty.Options, 2)
}

func TestPythonCodeNode_ValidateParameters(t *testing.T) {
	node := NewPythonCodeNode()

	tests := []struct {
		name      string
		params    map[string]interface{}
		wantError bool
	}{
		{
			name: "valid parameters",
			params: map[string]interface{}{
				"pythonCode": "return 42",
				"mode":       "runOnce",
			},
			wantError: false,
		},
		{
			name: "missing Python code",
			params: map[string]interface{}{
				"mode": "runOnce",
			},
			wantError: true,
		},
		{
			name: "empty Python code",
			params: map[string]interface{}{
				"pythonCode": "   ",
				"mode":       "runOnce",
			},
			wantError: true,
		},
		{
			name: "invalid mode",
			params: map[string]interface{}{
				"pythonCode": "return 42",
				"mode":       "invalidMode",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := node.ValidateParameters(tt.params)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}