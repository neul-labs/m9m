package trigger

import (
	"testing"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorTriggerNode_Execute(t *testing.T) {
	node := NewErrorTriggerNode()

	t.Run("enriches error input data", func(t *testing.T) {
		input := []model.DataItem{
			{JSON: map[string]interface{}{
				"error":   "node execution failed",
				"message": "connection timeout",
				"node":    "HTTP Request",
			}},
		}

		result, err := node.Execute(input, nil)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "node execution failed", result[0].JSON["error"])
		assert.Equal(t, "connection timeout", result[0].JSON["message"])
		assert.Equal(t, "HTTP Request", result[0].JSON["node"])
		assert.Equal(t, "error", result[0].JSON["trigger"])
		assert.NotEmpty(t, result[0].JSON["timestamp"])
	})

	t.Run("preserves existing timestamp and trigger", func(t *testing.T) {
		input := []model.DataItem{
			{JSON: map[string]interface{}{
				"error":     "test",
				"timestamp": "2024-01-01T00:00:00Z",
				"trigger":   "custom",
			}},
		}

		result, err := node.Execute(input, nil)
		require.NoError(t, err)
		assert.Equal(t, "2024-01-01T00:00:00Z", result[0].JSON["timestamp"])
		assert.Equal(t, "custom", result[0].JSON["trigger"])
	})

	t.Run("generates default when no input", func(t *testing.T) {
		result, err := node.Execute(nil, nil)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "Unknown error", result[0].JSON["error"])
		assert.Equal(t, "error", result[0].JSON["trigger"])
		assert.NotEmpty(t, result[0].JSON["timestamp"])
	})

	t.Run("description", func(t *testing.T) {
		desc := node.Description()
		assert.Equal(t, "Error Trigger", desc.Name)
		assert.Equal(t, "Trigger", desc.Category)
	})

	t.Run("validate parameters", func(t *testing.T) {
		assert.NoError(t, node.ValidateParameters(nil))
	})
}
