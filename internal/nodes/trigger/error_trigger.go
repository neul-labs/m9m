package trigger

import (
	"time"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// ErrorTriggerNode is a workflow entry point that receives and formats error data.
type ErrorTriggerNode struct {
	*base.BaseNode
}

// NewErrorTriggerNode creates a new Error Trigger node.
func NewErrorTriggerNode() *ErrorTriggerNode {
	return &ErrorTriggerNode{
		BaseNode: base.NewBaseNode(base.NodeDescription{
			Name:        "Error Trigger",
			Description: "Triggers when a workflow error occurs",
			Category:    "Trigger",
		}),
	}
}

// Execute formats error data from input or generates a default error structure.
func (n *ErrorTriggerNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	if len(inputData) > 0 {
		// Input already contains error data — enrich and pass through
		var result []model.DataItem
		for _, item := range inputData {
			enriched := make(map[string]interface{}, len(item.JSON)+2)
			for k, v := range item.JSON {
				enriched[k] = v
			}
			if _, ok := enriched["timestamp"]; !ok {
				enriched["timestamp"] = time.Now().UTC().Format(time.RFC3339)
			}
			if _, ok := enriched["trigger"]; !ok {
				enriched["trigger"] = "error"
			}
			result = append(result, model.DataItem{JSON: enriched})
		}
		return result, nil
	}

	// No input data — generate a default error trigger item
	return []model.DataItem{
		{JSON: map[string]interface{}{
			"error":    "Unknown error",
			"message":  "Error trigger activated without error context",
			"trigger":  "error",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}},
	}, nil
}

// ValidateParameters validates Error Trigger parameters.
func (n *ErrorTriggerNode) ValidateParameters(params map[string]interface{}) error {
	return nil
}
