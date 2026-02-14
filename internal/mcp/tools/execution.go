package tools

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/neul-labs/m9m/internal/engine"
	"github.com/neul-labs/m9m/internal/mcp"
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/storage"
)

// ExecutionManager manages workflow executions
type ExecutionManager struct {
	mu         sync.RWMutex
	engine     engine.WorkflowEngine
	storage    storage.WorkflowStorage
	executions map[string]*model.WorkflowExecution
	cancels    map[string]context.CancelFunc
}

// NewExecutionManager creates a new execution manager
func NewExecutionManager(eng engine.WorkflowEngine, store storage.WorkflowStorage) *ExecutionManager {
	return &ExecutionManager{
		engine:     eng,
		storage:    store,
		executions: make(map[string]*model.WorkflowExecution),
		cancels:    make(map[string]context.CancelFunc),
	}
}

// ExecutionRunTool executes a workflow synchronously
type ExecutionRunTool struct {
	*BaseTool
	engine  engine.WorkflowEngine
	storage storage.WorkflowStorage
}

// NewExecutionRunTool creates a new execution run tool
func NewExecutionRunTool(eng engine.WorkflowEngine, store storage.WorkflowStorage) *ExecutionRunTool {
	return &ExecutionRunTool{
		BaseTool: NewBaseTool(
			"execution_run",
			"Execute a workflow immediately and return the result. For long-running workflows, use execution_run_async.",
			ObjectSchema(map[string]interface{}{
				"workflowId": StringProp("ID of the workflow to execute"),
				"inputData":  ArrayProp("Input data to pass to the workflow", map[string]interface{}{"type": "object"}),
			}, []string{"workflowId"}),
		),
		engine:  eng,
		storage: store,
	}
}

// Execute runs a workflow
func (t *ExecutionRunTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if t.engine == nil || t.storage == nil {
		return mcp.ErrorContent("Engine or storage not initialized"), nil
	}

	workflowId := GetString(args, "workflowId")
	inputDataArg := GetArray(args, "inputData")

	// Get workflow
	workflow, err := t.storage.GetWorkflow(workflowId)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to get workflow: %v", err)), nil
	}
	if workflow == nil {
		return mcp.ErrorContent(fmt.Sprintf("Workflow not found: %s", workflowId)), nil
	}

	// Parse input data
	inputData := make([]model.DataItem, 0)
	if inputDataArg != nil {
		for _, item := range inputDataArg {
			if itemMap, ok := item.(map[string]interface{}); ok {
				dataItem := model.DataItem{}
				if json, ok := itemMap["json"].(map[string]interface{}); ok {
					dataItem.JSON = json
				} else {
					dataItem.JSON = itemMap
				}
				inputData = append(inputData, dataItem)
			}
		}
	}

	// If no input data, use empty item
	if len(inputData) == 0 {
		inputData = []model.DataItem{{JSON: map[string]interface{}{}}}
	}

	// Create execution record
	executionId := uuid.New().String()
	execution := &model.WorkflowExecution{
		ID:         executionId,
		WorkflowID: workflowId,
		Status:     "running",
		Mode:       "manual",
		StartedAt:  time.Now(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Save initial execution state
	if err := t.storage.SaveExecution(execution); err != nil {
		// Non-fatal, continue with execution
	}

	// Execute workflow
	result, err := engine.ExecuteWorkflowWithContext(ctx, t.engine, workflow, inputData)
	executionErr := engine.ResolveExecutionError(result, err)

	// Update execution status
	finishedAt := time.Now()
	execution.FinishedAt = &finishedAt
	execution.UpdatedAt = finishedAt

	if executionErr != nil {
		execution.Status = "failed"
		execution.Error = executionErr
		t.storage.SaveExecution(execution)
		return mcp.ErrorContent(fmt.Sprintf("Workflow execution failed: %v", executionErr)), nil
	}

	execution.Status = "completed"
	execution.Data = result.Data
	t.storage.SaveExecution(execution)

	return mcp.SuccessJSON(map[string]interface{}{
		"executionId": executionId,
		"status":      "completed",
		"workflowId":  workflowId,
		"data":        result.Data,
		"startedAt":   execution.StartedAt,
		"finishedAt":  execution.FinishedAt,
		"duration":    finishedAt.Sub(execution.StartedAt).String(),
	}), nil
}

// ExecutionRunAsyncTool executes a workflow asynchronously
type ExecutionRunAsyncTool struct {
	*BaseTool
	manager *ExecutionManager
}

// NewExecutionRunAsyncTool creates a new async execution tool
func NewExecutionRunAsyncTool(manager *ExecutionManager) *ExecutionRunAsyncTool {
	return &ExecutionRunAsyncTool{
		BaseTool: NewBaseTool(
			"execution_run_async",
			"Execute a workflow asynchronously. Returns immediately with an execution ID that can be polled for status.",
			ObjectSchema(map[string]interface{}{
				"workflowId": StringProp("ID of the workflow to execute"),
				"inputData":  ArrayProp("Input data to pass to the workflow", map[string]interface{}{"type": "object"}),
			}, []string{"workflowId"}),
		),
		manager: manager,
	}
}

// Execute starts an async workflow execution
func (t *ExecutionRunAsyncTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if t.manager == nil || t.manager.engine == nil || t.manager.storage == nil {
		return mcp.ErrorContent("Execution manager not initialized"), nil
	}

	workflowId := GetString(args, "workflowId")
	inputDataArg := GetArray(args, "inputData")

	// Get workflow
	workflow, err := t.manager.storage.GetWorkflow(workflowId)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to get workflow: %v", err)), nil
	}
	if workflow == nil {
		return mcp.ErrorContent(fmt.Sprintf("Workflow not found: %s", workflowId)), nil
	}

	// Parse input data
	inputData := make([]model.DataItem, 0)
	if inputDataArg != nil {
		for _, item := range inputDataArg {
			if itemMap, ok := item.(map[string]interface{}); ok {
				dataItem := model.DataItem{}
				if json, ok := itemMap["json"].(map[string]interface{}); ok {
					dataItem.JSON = json
				} else {
					dataItem.JSON = itemMap
				}
				inputData = append(inputData, dataItem)
			}
		}
	}

	if len(inputData) == 0 {
		inputData = []model.DataItem{{JSON: map[string]interface{}{}}}
	}

	// Create execution record
	executionId := uuid.New().String()
	execution := &model.WorkflowExecution{
		ID:         executionId,
		WorkflowID: workflowId,
		Status:     "running",
		Mode:       "manual",
		StartedAt:  time.Now(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Track execution
	t.manager.mu.Lock()
	t.manager.executions[executionId] = execution
	t.manager.mu.Unlock()

	// Save initial state
	t.manager.storage.SaveExecution(execution)

	// Run async
	execCtx, cancel := context.WithCancel(context.Background())
	t.manager.mu.Lock()
	t.manager.cancels[executionId] = cancel
	t.manager.mu.Unlock()

	go func() {
		result, err := engine.ExecuteWorkflowWithContext(execCtx, t.manager.engine, workflow, inputData)
		executionErr := engine.ResolveExecutionError(result, err)

		t.manager.mu.Lock()
		defer t.manager.mu.Unlock()
		delete(t.manager.cancels, executionId)

		finishedAt := time.Now()
		execution.FinishedAt = &finishedAt
		execution.UpdatedAt = finishedAt

		if executionErr != nil {
			if errors.Is(executionErr, context.Canceled) {
				execution.Status = "cancelled"
			} else {
				execution.Status = "failed"
			}
			execution.Error = executionErr
		} else {
			execution.Status = "completed"
			execution.Data = result.Data
		}

		t.manager.storage.SaveExecution(execution)
	}()

	return mcp.SuccessJSON(map[string]interface{}{
		"executionId": executionId,
		"status":      "running",
		"workflowId":  workflowId,
		"message":     "Workflow execution started. Use execution_get to check status.",
	}), nil
}

// ExecutionGetTool gets execution details
type ExecutionGetTool struct {
	*BaseTool
	storage storage.WorkflowStorage
	manager *ExecutionManager
}

// NewExecutionGetTool creates a new execution get tool
func NewExecutionGetTool(store storage.WorkflowStorage, manager *ExecutionManager) *ExecutionGetTool {
	return &ExecutionGetTool{
		BaseTool: NewBaseTool(
			"execution_get",
			"Get details of a workflow execution including status, data, and timing.",
			ObjectSchema(map[string]interface{}{
				"executionId": StringProp("Execution ID to get"),
			}, []string{"executionId"}),
		),
		storage: store,
		manager: manager,
	}
}

// Execute gets an execution
func (t *ExecutionGetTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	executionId := GetString(args, "executionId")

	// Check in-memory first for running executions
	if t.manager != nil {
		t.manager.mu.RLock()
		if exec, ok := t.manager.executions[executionId]; ok {
			t.manager.mu.RUnlock()
			return mcp.SuccessJSON(exec), nil
		}
		t.manager.mu.RUnlock()
	}

	// Check storage
	if t.storage == nil {
		return mcp.ErrorContent("Storage not initialized"), nil
	}

	execution, err := t.storage.GetExecution(executionId)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to get execution: %v", err)), nil
	}

	if execution == nil {
		return mcp.ErrorContent(fmt.Sprintf("Execution not found: %s", executionId)), nil
	}

	return mcp.SuccessJSON(execution), nil
}

// ExecutionListTool lists executions
type ExecutionListTool struct {
	*BaseTool
	storage storage.WorkflowStorage
}

// NewExecutionListTool creates a new execution list tool
func NewExecutionListTool(store storage.WorkflowStorage) *ExecutionListTool {
	return &ExecutionListTool{
		BaseTool: NewBaseTool(
			"execution_list",
			"List workflow executions with optional filtering.",
			ObjectSchema(map[string]interface{}{
				"workflowId": StringProp("Filter by workflow ID"),
				"status":     StringEnumProp("Filter by status", []string{"running", "completed", "failed", "cancelled"}),
				"limit":      IntPropWithDefault("Maximum number of results", 50),
				"offset":     IntPropWithDefault("Offset for pagination", 0),
			}, nil),
		),
		storage: store,
	}
}

// Execute lists executions
func (t *ExecutionListTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if t.storage == nil {
		return mcp.ErrorContent("Storage not initialized"), nil
	}

	filters := storage.ExecutionFilters{
		WorkflowID: GetString(args, "workflowId"),
		Status:     GetString(args, "status"),
		Limit:      GetIntOr(args, "limit", 50),
		Offset:     GetIntOr(args, "offset", 0),
	}

	executions, total, err := t.storage.ListExecutions(filters)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to list executions: %v", err)), nil
	}

	return mcp.SuccessJSON(map[string]interface{}{
		"executions": executions,
		"total":      total,
		"limit":      filters.Limit,
		"offset":     filters.Offset,
	}), nil
}

// ExecutionCancelTool cancels a running execution
type ExecutionCancelTool struct {
	*BaseTool
	manager *ExecutionManager
}

// NewExecutionCancelTool creates a new execution cancel tool
func NewExecutionCancelTool(manager *ExecutionManager) *ExecutionCancelTool {
	return &ExecutionCancelTool{
		BaseTool: NewBaseTool(
			"execution_cancel",
			"Cancel a running workflow execution.",
			ObjectSchema(map[string]interface{}{
				"executionId": StringProp("Execution ID to cancel"),
			}, []string{"executionId"}),
		),
		manager: manager,
	}
}

// Execute cancels an execution
func (t *ExecutionCancelTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	executionId := GetString(args, "executionId")

	if t.manager == nil {
		return mcp.ErrorContent("Execution manager not initialized"), nil
	}

	t.manager.mu.Lock()
	defer t.manager.mu.Unlock()

	exec, ok := t.manager.executions[executionId]
	if !ok {
		return mcp.ErrorContent(fmt.Sprintf("Execution not found or not running: %s", executionId)), nil
	}

	if exec.Status != "running" {
		return mcp.ErrorContent(fmt.Sprintf("Execution is not running (status: %s)", exec.Status)), nil
	}

	cancel, exists := t.manager.cancels[executionId]
	if !exists {
		return mcp.ErrorContent(fmt.Sprintf("Execution is running but cannot be cancelled: %s", executionId)), nil
	}
	cancel()

	return mcp.SuccessJSON(map[string]interface{}{
		"success":     true,
		"executionId": executionId,
		"status":      "cancel_requested",
		"message":     "Execution cancellation requested",
	}), nil
}

// ExecutionRetryTool retries a failed execution
type ExecutionRetryTool struct {
	*BaseTool
	engine  engine.WorkflowEngine
	storage storage.WorkflowStorage
}

// NewExecutionRetryTool creates a new execution retry tool
func NewExecutionRetryTool(eng engine.WorkflowEngine, store storage.WorkflowStorage) *ExecutionRetryTool {
	return &ExecutionRetryTool{
		BaseTool: NewBaseTool(
			"execution_retry",
			"Retry a failed workflow execution with the same input data.",
			ObjectSchema(map[string]interface{}{
				"executionId": StringProp("Execution ID to retry"),
			}, []string{"executionId"}),
		),
		engine:  eng,
		storage: store,
	}
}

// Execute retries an execution
func (t *ExecutionRetryTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if t.engine == nil || t.storage == nil {
		return mcp.ErrorContent("Engine or storage not initialized"), nil
	}

	executionId := GetString(args, "executionId")

	// Get original execution
	original, err := t.storage.GetExecution(executionId)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to get execution: %v", err)), nil
	}
	if original == nil {
		return mcp.ErrorContent(fmt.Sprintf("Execution not found: %s", executionId)), nil
	}

	if original.Status != "failed" {
		return mcp.ErrorContent(fmt.Sprintf("Can only retry failed executions (status: %s)", original.Status)), nil
	}

	// Get workflow
	workflow, err := t.storage.GetWorkflow(original.WorkflowID)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to get workflow: %v", err)), nil
	}

	// Create new execution
	newExecutionId := uuid.New().String()
	execution := &model.WorkflowExecution{
		ID:         newExecutionId,
		WorkflowID: original.WorkflowID,
		Status:     "running",
		Mode:       "retry",
		StartedAt:  time.Now(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Metadata: map[string]interface{}{
			"retryOf": executionId,
		},
	}

	t.storage.SaveExecution(execution)

	// Execute
	inputData := original.Data
	if len(inputData) == 0 {
		inputData = []model.DataItem{{JSON: map[string]interface{}{}}}
	}

	result, err := engine.ExecuteWorkflowWithContext(ctx, t.engine, workflow, inputData)
	executionErr := engine.ResolveExecutionError(result, err)

	finishedAt := time.Now()
	execution.FinishedAt = &finishedAt
	execution.UpdatedAt = finishedAt

	if executionErr != nil {
		execution.Status = "failed"
		execution.Error = executionErr
		t.storage.SaveExecution(execution)
		return mcp.ErrorContent(fmt.Sprintf("Retry failed: %v", executionErr)), nil
	}

	execution.Status = "completed"
	execution.Data = result.Data
	t.storage.SaveExecution(execution)

	return mcp.SuccessJSON(map[string]interface{}{
		"newExecutionId":      newExecutionId,
		"originalExecutionId": executionId,
		"status":              "completed",
		"data":                result.Data,
	}), nil
}

// ExecutionWaitTool waits for an execution to complete
type ExecutionWaitTool struct {
	*BaseTool
	storage storage.WorkflowStorage
	manager *ExecutionManager
}

// NewExecutionWaitTool creates a new execution wait tool
func NewExecutionWaitTool(store storage.WorkflowStorage, manager *ExecutionManager) *ExecutionWaitTool {
	return &ExecutionWaitTool{
		BaseTool: NewBaseTool(
			"execution_wait",
			"Wait for a workflow execution to complete. Polls until completion or timeout.",
			ObjectSchema(map[string]interface{}{
				"executionId": StringProp("Execution ID to wait for"),
				"timeoutMs":   IntPropWithDefault("Maximum time to wait in milliseconds", 60000),
				"pollMs":      IntPropWithDefault("Polling interval in milliseconds", 500),
			}, []string{"executionId"}),
		),
		storage: store,
		manager: manager,
	}
}

// Execute waits for an execution
func (t *ExecutionWaitTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	executionId := GetString(args, "executionId")
	timeoutMs := GetIntOr(args, "timeoutMs", 60000)
	pollMs := GetIntOr(args, "pollMs", 500)

	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	pollInterval := time.Duration(pollMs) * time.Millisecond

	for time.Now().Before(deadline) {
		// Check in-memory
		if t.manager != nil {
			t.manager.mu.RLock()
			if exec, ok := t.manager.executions[executionId]; ok {
				if exec.Status == "completed" || exec.Status == "failed" || exec.Status == "cancelled" {
					t.manager.mu.RUnlock()
					return mcp.SuccessJSON(exec), nil
				}
			}
			t.manager.mu.RUnlock()
		}

		// Check storage
		if t.storage != nil {
			if exec, err := t.storage.GetExecution(executionId); err == nil && exec != nil {
				if exec.Status == "completed" || exec.Status == "failed" || exec.Status == "cancelled" {
					return mcp.SuccessJSON(exec), nil
				}
			}
		}

		select {
		case <-ctx.Done():
			return mcp.ErrorContent("Context cancelled while waiting"), nil
		case <-time.After(pollInterval):
			continue
		}
	}

	return mcp.ErrorContent(fmt.Sprintf("Timeout waiting for execution %s after %dms", executionId, timeoutMs)), nil
}

// RegisterExecutionTools registers all execution tools with a registry
func RegisterExecutionTools(registry *Registry, eng engine.WorkflowEngine, store storage.WorkflowStorage) *ExecutionManager {
	manager := NewExecutionManager(eng, store)

	registry.Register(NewExecutionRunTool(eng, store))
	registry.Register(NewExecutionRunAsyncTool(manager))
	registry.Register(NewExecutionGetTool(store, manager))
	registry.Register(NewExecutionListTool(store))
	registry.Register(NewExecutionCancelTool(manager))
	registry.Register(NewExecutionRetryTool(eng, store))
	registry.Register(NewExecutionWaitTool(store, manager))

	return manager
}
