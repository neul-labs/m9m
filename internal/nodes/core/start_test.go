package core

import (
	"testing"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStartNode_Execute_PassThrough verifies that the Start node passes input
// data through unchanged when input is provided.
func TestStartNode_Execute_PassThrough(t *testing.T) {
	node := NewStartNode()

	tests := []struct {
		name  string
		input []model.DataItem
	}{
		{
			name: "single item passes through",
			input: []model.DataItem{
				{
					JSON: map[string]interface{}{
						"key": "value",
					},
				},
			},
		},
		{
			name: "multiple items pass through",
			input: []model.DataItem{
				{
					JSON: map[string]interface{}{
						"name": "Alice",
						"age":  float64(30),
					},
				},
				{
					JSON: map[string]interface{}{
						"name": "Bob",
						"age":  float64(25),
					},
				},
			},
		},
		{
			name: "item with nested data passes through",
			input: []model.DataItem{
				{
					JSON: map[string]interface{}{
						"user": map[string]interface{}{
							"name":  "Charlie",
							"email": "charlie@example.com",
						},
						"active": true,
					},
				},
			},
		},
		{
			name: "item with empty JSON passes through",
			input: []model.DataItem{
				{
					JSON: map[string]interface{}{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := node.Execute(tt.input, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.input, result, "output should be identical to input")
			assert.Len(t, result, len(tt.input), "output length should match input length")
		})
	}
}

// TestStartNode_Execute_EmptyInput verifies that the Start node returns a
// single empty DataItem when no input data is provided.
func TestStartNode_Execute_EmptyInput(t *testing.T) {
	node := NewStartNode()

	tests := []struct {
		name  string
		input []model.DataItem
	}{
		{
			name:  "nil input returns single empty item",
			input: nil,
		},
		{
			name:  "empty slice returns single empty item",
			input: []model.DataItem{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := node.Execute(tt.input, nil)
			require.NoError(t, err)
			require.Len(t, result, 1, "should return exactly one item")
			assert.NotNil(t, result[0].JSON, "JSON field should not be nil")
			assert.Empty(t, result[0].JSON, "JSON map should be empty")
		})
	}
}

// TestStartNode_Description verifies that the Start node description is
// correctly populated with the expected name, description, and category.
func TestStartNode_Description(t *testing.T) {
	node := NewStartNode()
	desc := node.Description()

	assert.Equal(t, "Start", desc.Name, "node name should be 'Start'")
	assert.Equal(t, "Triggers the workflow execution", desc.Description, "description should match")
	assert.Equal(t, "Core", desc.Category, "category should be 'Core'")
}

// TestStartNode_ValidateParameters verifies that the Start node accepts any
// parameters since it has no required parameters.
func TestStartNode_ValidateParameters(t *testing.T) {
	node := NewStartNode()

	tests := []struct {
		name   string
		params map[string]interface{}
	}{
		{
			name:   "nil params are valid",
			params: nil,
		},
		{
			name:   "empty params are valid",
			params: map[string]interface{}{},
		},
		{
			name: "arbitrary params are valid",
			params: map[string]interface{}{
				"someKey":    "someValue",
				"anotherKey": 42,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := node.ValidateParameters(tt.params)
			assert.NoError(t, err, "start node should accept any parameters")
		})
	}
}

// TestStartNode_ImplementsNodeExecutor verifies at compile time that StartNode
// implements the base.NodeExecutor interface.
func TestStartNode_ImplementsNodeExecutor(t *testing.T) {
	var _ base.NodeExecutor = (*StartNode)(nil)
}

// TestStartNode_Execute_ParamsIgnored verifies that node parameters do not
// affect the output of the Start node.
func TestStartNode_Execute_ParamsIgnored(t *testing.T) {
	node := NewStartNode()

	input := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"data": "test",
			},
		},
	}

	params := map[string]interface{}{
		"irrelevant": "param",
		"count":      99,
	}

	result, err := node.Execute(input, params)
	require.NoError(t, err)
	assert.Equal(t, input, result, "params should not alter the output")
}
