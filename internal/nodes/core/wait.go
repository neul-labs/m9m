package core

import (
	"context"
	"fmt"
	"time"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// WaitNode pauses workflow execution for a configurable duration.
type WaitNode struct {
	*base.BaseNode
}

// NewWaitNode creates a new Wait node.
func NewWaitNode() *WaitNode {
	return &WaitNode{
		BaseNode: base.NewBaseNode(base.NodeDescription{
			Name:        "Wait",
			Description: "Pauses execution for a specified duration",
			Category:    "Core",
		}),
	}
}

// Execute pauses and then passes through input data.
func (w *WaitNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	return w.ExecuteWithContext(context.Background(), inputData, nodeParams)
}

// ExecuteWithContext pauses execution honoring context cancellation.
func (w *WaitNode) ExecuteWithContext(ctx context.Context, inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	duration, err := w.parseDuration(nodeParams)
	if err != nil {
		return nil, err
	}

	select {
	case <-time.After(duration):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	if len(inputData) == 0 {
		return []model.DataItem{{JSON: map[string]interface{}{}}}, nil
	}
	return inputData, nil
}

func (w *WaitNode) parseDuration(params map[string]interface{}) (time.Duration, error) {
	amount := w.GetIntParameter(params, "amount", 1)
	unit := w.GetStringParameter(params, "unit", "seconds")

	switch unit {
	case "milliseconds":
		return time.Duration(amount) * time.Millisecond, nil
	case "seconds":
		return time.Duration(amount) * time.Second, nil
	case "minutes":
		return time.Duration(amount) * time.Minute, nil
	default:
		return 0, fmt.Errorf("node Wait error: invalid unit: %s (expected milliseconds, seconds, or minutes)", unit)
	}
}

// ValidateParameters validates Wait node parameters.
func (w *WaitNode) ValidateParameters(params map[string]interface{}) error {
	if params == nil {
		return nil
	}

	unit := w.GetStringParameter(params, "unit", "seconds")
	switch unit {
	case "milliseconds", "seconds", "minutes":
	default:
		return fmt.Errorf("node Wait error: invalid unit: %s", unit)
	}

	amount := w.GetIntParameter(params, "amount", 1)
	if amount < 0 {
		return fmt.Errorf("node Wait error: amount must be non-negative")
	}

	return nil
}
