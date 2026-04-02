package transform

import (
	"testing"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoopNode_Each(t *testing.T) {
	node := NewLoopNode()

	items := []model.DataItem{
		{JSON: map[string]interface{}{"name": "Alice"}},
		{JSON: map[string]interface{}{"name": "Bob"}},
	}

	params := map[string]interface{}{"mode": "each"}

	result, err := node.Execute(items, params)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 0, result[0].JSON["$index"])
	assert.Equal(t, 1, result[1].JSON["$index"])
	assert.Equal(t, "Alice", result[0].JSON["name"])

	// $item should be the original item JSON
	itemData, ok := result[0].JSON["$item"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Alice", itemData["name"])
}

func TestLoopNode_EachEmpty(t *testing.T) {
	node := NewLoopNode()
	result, err := node.Execute(nil, map[string]interface{}{"mode": "each"})
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestLoopNode_Times(t *testing.T) {
	node := NewLoopNode()

	params := map[string]interface{}{"mode": "times", "times": 5}
	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Len(t, result, 5)
	for i, item := range result {
		assert.Equal(t, i, item.JSON["$index"])
	}
}

func TestLoopNode_TimesNegative(t *testing.T) {
	node := NewLoopNode()
	_, err := node.Execute(nil, map[string]interface{}{"mode": "times", "times": -1})
	assert.Error(t, err)
}

func TestLoopNode_TimesExceedsMax(t *testing.T) {
	node := NewLoopNode()
	result, err := node.Execute(nil, map[string]interface{}{"mode": "times", "times": 5000})
	require.NoError(t, err)
	assert.Len(t, result, 1000) // capped at defaultMaxIterations
}

func TestLoopNode_While(t *testing.T) {
	node := NewLoopNode()

	t.Run("basic while loop", func(t *testing.T) {
		params := map[string]interface{}{
			"mode":          "while",
			"condition":     "$index < 3",
			"maxIterations": 10,
		}
		result, err := node.Execute(nil, params)
		require.NoError(t, err)
		assert.Len(t, result, 3)
	})

	t.Run("false from start", func(t *testing.T) {
		params := map[string]interface{}{
			"mode":      "while",
			"condition": "false",
		}
		result, err := node.Execute(nil, params)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("missing condition", func(t *testing.T) {
		params := map[string]interface{}{"mode": "while"}
		_, err := node.Execute(nil, params)
		assert.Error(t, err)
	})

	t.Run("invalid condition expression", func(t *testing.T) {
		params := map[string]interface{}{
			"mode":      "while",
			"condition": "not valid {{}}",
		}
		_, err := node.Execute(nil, params)
		assert.Error(t, err)
	})
}

func TestLoopNode_DefaultMode(t *testing.T) {
	node := NewLoopNode()
	items := []model.DataItem{{JSON: map[string]interface{}{"x": 1}}}
	result, err := node.Execute(items, map[string]interface{}{})
	require.NoError(t, err)
	assert.Len(t, result, 1) // default mode is "each"
}

func TestLoopNode_InvalidMode(t *testing.T) {
	node := NewLoopNode()
	_, err := node.Execute(nil, map[string]interface{}{"mode": "invalid"})
	assert.Error(t, err)
}

func TestLoopNode_ValidateParameters(t *testing.T) {
	node := NewLoopNode()

	assert.NoError(t, node.ValidateParameters(nil))
	assert.NoError(t, node.ValidateParameters(map[string]interface{}{"mode": "each"}))
	assert.NoError(t, node.ValidateParameters(map[string]interface{}{"mode": "times", "times": 5}))
	assert.NoError(t, node.ValidateParameters(map[string]interface{}{"mode": "while", "condition": "$index < 10"}))
	assert.Error(t, node.ValidateParameters(map[string]interface{}{"mode": "while"}))
	assert.Error(t, node.ValidateParameters(map[string]interface{}{"mode": "invalid"}))
	assert.Error(t, node.ValidateParameters(map[string]interface{}{"mode": "times", "times": -1}))
}

func TestLoopNode_Description(t *testing.T) {
	node := NewLoopNode()
	desc := node.Description()
	assert.Equal(t, "Loop", desc.Name)
}
