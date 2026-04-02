package core

import (
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// NoOpNode implements a pass-through node that returns input unchanged.
type NoOpNode struct {
	*base.BaseNode
}

// NewNoOpNode creates a new NoOp node.
func NewNoOpNode() *NoOpNode {
	return &NoOpNode{
		BaseNode: base.NewBaseNode(base.NodeDescription{
			Name:        "NoOp",
			Description: "Does nothing, passes data through unchanged",
			Category:    "Core",
		}),
	}
}

// Execute returns the input data unchanged.
func (n *NoOpNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	if len(inputData) == 0 {
		return []model.DataItem{{JSON: map[string]interface{}{}}}, nil
	}
	return inputData, nil
}

// ValidateParameters validates node parameters (NoOp has no parameters).
func (n *NoOpNode) ValidateParameters(params map[string]interface{}) error {
	return nil
}
