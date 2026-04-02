package core

import (
	"context"
	"testing"
	"time"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaitNode_Execute(t *testing.T) {
	node := NewWaitNode()

	t.Run("waits and passes through data", func(t *testing.T) {
		input := []model.DataItem{{JSON: map[string]interface{}{"key": "value"}}}
		params := map[string]interface{}{"amount": 1, "unit": "milliseconds"}

		start := time.Now()
		result, err := node.Execute(input, params)
		elapsed := time.Since(start)

		require.NoError(t, err)
		assert.Equal(t, input, result)
		assert.GreaterOrEqual(t, elapsed, 1*time.Millisecond)
	})

	t.Run("returns empty item when no input", func(t *testing.T) {
		params := map[string]interface{}{"amount": 1, "unit": "milliseconds"}
		result, err := node.Execute(nil, params)
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately

		params := map[string]interface{}{"amount": 10, "unit": "seconds"}
		_, err := node.ExecuteWithContext(ctx, nil, params)
		assert.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("default params", func(t *testing.T) {
		params := map[string]interface{}{"amount": 1, "unit": "milliseconds"}
		result, err := node.Execute([]model.DataItem{{JSON: map[string]interface{}{}}}, params)
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("invalid unit", func(t *testing.T) {
		params := map[string]interface{}{"unit": "hours"}
		_, err := node.Execute(nil, params)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid unit")
	})
}

func TestWaitNode_ValidateParameters(t *testing.T) {
	node := NewWaitNode()

	assert.NoError(t, node.ValidateParameters(nil))
	assert.NoError(t, node.ValidateParameters(map[string]interface{}{"amount": 5, "unit": "seconds"}))
	assert.NoError(t, node.ValidateParameters(map[string]interface{}{"unit": "milliseconds"}))
	assert.NoError(t, node.ValidateParameters(map[string]interface{}{"unit": "minutes"}))
	assert.Error(t, node.ValidateParameters(map[string]interface{}{"unit": "hours"}))
	assert.Error(t, node.ValidateParameters(map[string]interface{}{"amount": -1}))
}

func TestWaitNode_Description(t *testing.T) {
	node := NewWaitNode()
	desc := node.Description()
	assert.Equal(t, "Wait", desc.Name)
	assert.Equal(t, "Core", desc.Category)
}
