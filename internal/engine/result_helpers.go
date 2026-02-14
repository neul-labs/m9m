package engine

import (
	"context"
	"fmt"

	"github.com/neul-labs/m9m/internal/model"
)

// ResolveExecutionError normalizes workflow execution errors from both return channels:
// function return error and ExecutionResult.Error.
func ResolveExecutionError(result *ExecutionResult, execErr error) error {
	if execErr != nil {
		return execErr
	}
	if result == nil {
		return fmt.Errorf("workflow execution returned nil result")
	}
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// ContextWorkflowEngine is an optional interface for engines that support
// context-aware workflow execution and cancellation.
type ContextWorkflowEngine interface {
	ExecuteWorkflowWithContext(ctx context.Context, workflow *model.Workflow, inputData []model.DataItem) (*ExecutionResult, error)
}

// ExecuteWorkflowWithContext executes a workflow honoring context cancellation.
// If the engine does not natively support context, this function still returns on
// context cancellation but cannot forcibly stop non-context-aware node execution.
func ExecuteWorkflowWithContext(
	ctx context.Context,
	eng WorkflowEngine,
	workflow *model.Workflow,
	inputData []model.DataItem,
) (*ExecutionResult, error) {
	if eng == nil {
		return nil, fmt.Errorf("workflow engine is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	if contextEngine, ok := eng.(ContextWorkflowEngine); ok {
		return contextEngine.ExecuteWorkflowWithContext(ctx, workflow, inputData)
	}

	type workflowResult struct {
		result *ExecutionResult
		err    error
	}
	resultChan := make(chan workflowResult, 1)
	go func() {
		result, err := eng.ExecuteWorkflow(workflow, inputData)
		resultChan <- workflowResult{result: result, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case executionResult := <-resultChan:
		return executionResult.result, executionResult.err
	}
}
