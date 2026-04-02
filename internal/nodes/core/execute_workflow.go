package core

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// WorkflowExecutor is the subset of engine.WorkflowEngine needed by this node.
type WorkflowExecutor interface {
	ExecuteWorkflow(workflow *model.Workflow, inputData []model.DataItem) (*WorkflowResult, error)
}

// WorkflowResult mirrors engine.ExecutionResult so we don't import the engine package.
type WorkflowResult struct {
	Data  []model.DataItem
	Error error
}

// EngineAdapter adapts any engine that satisfies the full interface.
type EngineAdapter struct {
	ExecuteFn func(workflow *model.Workflow, inputData []model.DataItem) ([]model.DataItem, error)
}

// ExecuteWorkflow satisfies WorkflowExecutor.
func (a *EngineAdapter) ExecuteWorkflow(workflow *model.Workflow, inputData []model.DataItem) (*WorkflowResult, error) {
	data, err := a.ExecuteFn(workflow, inputData)
	if err != nil {
		return nil, err
	}
	return &WorkflowResult{Data: data}, nil
}

// ExecuteWorkflowNode loads and executes a sub-workflow.
type ExecuteWorkflowNode struct {
	*base.BaseNode
	executor WorkflowExecutor
	depth    int // current recursion depth, set by parent
}

// NewExecuteWorkflowNode creates a new Execute Workflow node.
// The engine parameter is used to execute the sub-workflow. Pass nil for a
// placeholder that will fail at execution time (useful for catalog listing).
func NewExecuteWorkflowNode(executor WorkflowExecutor) *ExecuteWorkflowNode {
	return &ExecuteWorkflowNode{
		BaseNode: base.NewBaseNode(base.NodeDescription{
			Name:        "Execute Workflow",
			Description: "Executes another workflow as a sub-workflow",
			Category:    "Core",
		}),
		executor: executor,
	}
}

// Execute loads and runs the sub-workflow.
func (n *ExecuteWorkflowNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	if n.executor == nil {
		return nil, n.CreateError("no workflow engine configured", nil)
	}

	maxDepth := n.GetIntParameter(nodeParams, "maxDepth", 10)
	if n.depth >= maxDepth {
		return nil, n.CreateError(fmt.Sprintf("maximum recursion depth (%d) reached", maxDepth), nil)
	}

	source := n.GetStringParameter(nodeParams, "source", "localFile")
	if source != "localFile" {
		return nil, n.CreateError(fmt.Sprintf("unsupported source: %s (only localFile is supported)", source), nil)
	}

	workflowPath := n.GetStringParameter(nodeParams, "workflowPath", "")
	if workflowPath == "" {
		return nil, n.CreateError("workflowPath is required", nil)
	}

	// Load workflow file
	data, err := os.ReadFile(workflowPath)
	if err != nil {
		return nil, n.CreateError(fmt.Sprintf("cannot read workflow file: %v", err), nil)
	}

	var workflow model.Workflow
	if err := json.Unmarshal(data, &workflow); err != nil {
		return nil, n.CreateError(fmt.Sprintf("invalid workflow JSON: %v", err), nil)
	}

	// Provide input data
	if len(inputData) == 0 {
		inputData = []model.DataItem{{JSON: map[string]interface{}{}}}
	}

	result, err := n.executor.ExecuteWorkflow(&workflow, inputData)
	if err != nil {
		return nil, n.CreateError(fmt.Sprintf("sub-workflow execution failed: %v", err), nil)
	}

	if result.Error != nil {
		return nil, n.CreateError(fmt.Sprintf("sub-workflow error: %v", result.Error), nil)
	}

	return result.Data, nil
}

// ValidateParameters validates Execute Workflow parameters.
func (n *ExecuteWorkflowNode) ValidateParameters(params map[string]interface{}) error {
	if params == nil {
		return nil
	}

	source := n.GetStringParameter(params, "source", "localFile")
	if source != "localFile" {
		return n.CreateError(fmt.Sprintf("unsupported source: %s", source), nil)
	}

	workflowPath := n.GetStringParameter(params, "workflowPath", "")
	if workflowPath == "" {
		return n.CreateError("workflowPath is required", nil)
	}

	return nil
}
