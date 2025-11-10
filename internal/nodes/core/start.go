package core

import (
	"github.com/dipankar/n8n-go/internal/model"
	"github.com/dipankar/n8n-go/internal/nodes/base"
)

// StartNode implements the n8n start trigger node
type StartNode struct {
	*base.BaseNode
}

// NewStartNode creates a new Start node
func NewStartNode() *StartNode {
	description := base.NodeDescription{
		Name:        "Start",
		Description: "Triggers the workflow execution",
		Category:    "Core",
	}

	return &StartNode{
		BaseNode: base.NewBaseNode(description),
	}
}

// Execute executes the Start node
func (s *StartNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	// Start node just passes through the input data
	// If no input data is provided, create a single empty item
	if len(inputData) == 0 {
		return []model.DataItem{
			{
				JSON: map[string]interface{}{},
			},
		}, nil
	}

	return inputData, nil
}

// ValidateParameters validates node parameters (start node has no parameters)
func (s *StartNode) ValidateParameters(params map[string]interface{}) error {
	// Start node has no parameters to validate
	return nil
}