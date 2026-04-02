package core

import (
	"testing"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoOpNode_Execute(t *testing.T) {
	node := NewNoOpNode()

	t.Run("passes through input data", func(t *testing.T) {
		input := []model.DataItem{
			{JSON: map[string]interface{}{"name": "test", "value": 42}},
			{JSON: map[string]interface{}{"name": "test2", "value": 99}},
		}
		result, err := node.Execute(input, nil)
		require.NoError(t, err)
		assert.Equal(t, input, result)
	})

	t.Run("returns empty item when no input", func(t *testing.T) {
		result, err := node.Execute(nil, nil)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, map[string]interface{}{}, result[0].JSON)
	})

	t.Run("description", func(t *testing.T) {
		desc := node.Description()
		assert.Equal(t, "NoOp", desc.Name)
		assert.Equal(t, "Core", desc.Category)
	})

	t.Run("validate parameters", func(t *testing.T) {
		assert.NoError(t, node.ValidateParameters(nil))
		assert.NoError(t, node.ValidateParameters(map[string]interface{}{"anything": true}))
	})
}
