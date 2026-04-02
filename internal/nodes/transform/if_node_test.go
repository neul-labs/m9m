package transform

import (
	"testing"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIfNode_Execute(t *testing.T) {
	node := NewIfNode()

	items := []model.DataItem{
		{JSON: map[string]interface{}{"name": "Alice", "age": float64(30)}},
		{JSON: map[string]interface{}{"name": "Bob", "age": float64(17)}},
		{JSON: map[string]interface{}{"name": "Charlie", "age": float64(25)}},
	}

	t.Run("filters items matching condition", func(t *testing.T) {
		params := map[string]interface{}{
			"conditions": []interface{}{
				map[string]interface{}{
					"leftValue":  "$json.age",
					"operator":   "greaterThanOrEqual",
					"rightValue": float64(18),
				},
			},
			"combiner": "and",
		}

		result, err := node.Execute(items, params)
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "Alice", result[0].JSON["name"])
		assert.Equal(t, "Charlie", result[1].JSON["name"])
	})

	t.Run("returns both branches when requested", func(t *testing.T) {
		params := map[string]interface{}{
			"conditions": []interface{}{
				map[string]interface{}{
					"leftValue":  "$json.age",
					"operator":   "greaterThanOrEqual",
					"rightValue": float64(18),
				},
			},
			"combiner":           "and",
			"returnBothBranches": true,
		}

		result, err := node.Execute(items, params)
		require.NoError(t, err)
		assert.Len(t, result, 3)

		trueCount := 0
		falseCount := 0
		for _, item := range result {
			if item.JSON["_ifResult"] == true {
				trueCount++
			} else {
				falseCount++
			}
		}
		assert.Equal(t, 2, trueCount)
		assert.Equal(t, 1, falseCount)
	})

	t.Run("empty input", func(t *testing.T) {
		params := map[string]interface{}{
			"conditions": []interface{}{},
			"combiner":   "and",
		}
		result, err := node.Execute(nil, params)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("or combiner", func(t *testing.T) {
		params := map[string]interface{}{
			"conditions": []interface{}{
				map[string]interface{}{
					"leftValue":  "$json.name",
					"operator":   "equals",
					"rightValue": "Alice",
				},
				map[string]interface{}{
					"leftValue":  "$json.name",
					"operator":   "equals",
					"rightValue": "Bob",
				},
			},
			"combiner": "or",
		}

		result, err := node.Execute(items, params)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("missing conditions", func(t *testing.T) {
		_, err := node.Execute(items, map[string]interface{}{})
		assert.Error(t, err)
	})

	t.Run("string equals condition", func(t *testing.T) {
		params := map[string]interface{}{
			"conditions": []interface{}{
				map[string]interface{}{
					"leftValue":  "$json.name",
					"operator":   "equals",
					"rightValue": "Bob",
				},
			},
			"combiner": "and",
		}

		result, err := node.Execute(items, params)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "Bob", result[0].JSON["name"])
	})
}

func TestIfNode_ValidateParameters(t *testing.T) {
	node := NewIfNode()

	assert.Error(t, node.ValidateParameters(nil))
	assert.Error(t, node.ValidateParameters(map[string]interface{}{}))
	assert.NoError(t, node.ValidateParameters(map[string]interface{}{
		"conditions": []interface{}{
			map[string]interface{}{
				"leftValue":  "$json.name",
				"operator":   "equals",
				"rightValue": "test",
			},
		},
		"combiner": "and",
	}))
}

func TestIfNode_Description(t *testing.T) {
	node := NewIfNode()
	desc := node.Description()
	assert.Equal(t, "IF", desc.Name)
}
