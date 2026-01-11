package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/dipankar/m9m/internal/mcp"
	"github.com/dipankar/m9m/internal/model"
	"github.com/dipankar/m9m/internal/storage"
)

// DebugExecutionLogsTool gets detailed execution logs
type DebugExecutionLogsTool struct {
	*BaseTool
	storage storage.WorkflowStorage
}

// NewDebugExecutionLogsTool creates a new debug execution logs tool
func NewDebugExecutionLogsTool(store storage.WorkflowStorage) *DebugExecutionLogsTool {
	return &DebugExecutionLogsTool{
		BaseTool: NewBaseTool(
			"debug_execution_logs",
			"Get detailed logs for a workflow execution including node-by-node outputs, timing, and any errors.",
			ObjectSchema(map[string]interface{}{
				"executionId": StringProp("Execution ID to get logs for"),
				"level": StringEnumProp("Detail level for logs", []string{
					"summary",  // High-level execution flow
					"detailed", // Node inputs/outputs
					"verbose",  // Full data + expression evaluations
				}),
				"nodeFilter": StringProp("Filter logs for a specific node name"),
			}, []string{"executionId"}),
		),
		storage: store,
	}
}

// Execute gets execution logs
func (t *DebugExecutionLogsTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if t.storage == nil {
		return mcp.ErrorContent("Storage not initialized"), nil
	}

	executionId := GetString(args, "executionId")
	level := GetStringOr(args, "level", "detailed")
	nodeFilter := GetString(args, "nodeFilter")

	execution, err := t.storage.GetExecution(executionId)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to get execution: %v", err)), nil
	}
	if execution == nil {
		return mcp.ErrorContent(fmt.Sprintf("Execution not found: %s", executionId)), nil
	}

	// Build log entries
	logs := make([]map[string]interface{}, 0)

	// Execution start
	logs = append(logs, map[string]interface{}{
		"timestamp": execution.StartedAt,
		"type":      "execution_start",
		"message":   fmt.Sprintf("Execution started (mode: %s)", execution.Mode),
		"data": map[string]interface{}{
			"executionId": execution.ID,
			"workflowId":  execution.WorkflowID,
			"mode":        execution.Mode,
		},
	})

	// Node data logs (if available and level permits)
	if execution.NodeData != nil && (level == "detailed" || level == "verbose") {
		for nodeName, data := range execution.NodeData {
			if nodeFilter != "" && nodeName != nodeFilter {
				continue
			}

			nodeLog := map[string]interface{}{
				"timestamp": execution.StartedAt, // Would be actual node start time if tracked
				"type":      "node_execution",
				"node":      nodeName,
				"message":   fmt.Sprintf("Node '%s' executed", nodeName),
			}

			if level == "verbose" {
				nodeLog["outputData"] = data
				nodeLog["itemCount"] = len(data)
			} else {
				nodeLog["itemCount"] = len(data)
			}

			logs = append(logs, nodeLog)
		}
	}

	// Execution end
	var endTime time.Time
	if execution.FinishedAt != nil {
		endTime = *execution.FinishedAt
	} else {
		endTime = time.Now()
	}

	duration := endTime.Sub(execution.StartedAt)

	endLog := map[string]interface{}{
		"timestamp": endTime,
		"type":      "execution_end",
		"status":    execution.Status,
		"message":   fmt.Sprintf("Execution %s after %v", execution.Status, duration),
		"data": map[string]interface{}{
			"duration":   duration.String(),
			"durationMs": duration.Milliseconds(),
		},
	}

	if execution.Error != nil {
		endLog["error"] = execution.Error.Error()
	}

	if level == "verbose" && execution.Data != nil {
		endLog["outputData"] = execution.Data
	}

	logs = append(logs, endLog)

	return mcp.SuccessJSON(map[string]interface{}{
		"executionId": executionId,
		"workflowId":  execution.WorkflowID,
		"status":      execution.Status,
		"level":       level,
		"logs":        logs,
		"summary": map[string]interface{}{
			"startedAt":  execution.StartedAt,
			"finishedAt": execution.FinishedAt,
			"duration":   duration.String(),
			"nodeCount":  len(execution.NodeData),
		},
	}), nil
}

// DebugNodeOutputTool gets output from a specific node
type DebugNodeOutputTool struct {
	*BaseTool
	storage storage.WorkflowStorage
}

// NewDebugNodeOutputTool creates a new debug node output tool
func NewDebugNodeOutputTool(store storage.WorkflowStorage) *DebugNodeOutputTool {
	return &DebugNodeOutputTool{
		BaseTool: NewBaseTool(
			"debug_node_output",
			"Get the output data from a specific node in a workflow execution.",
			ObjectSchema(map[string]interface{}{
				"executionId": StringProp("Execution ID"),
				"nodeName":    StringProp("Name of the node to get output from"),
			}, []string{"executionId", "nodeName"}),
		),
		storage: store,
	}
}

// Execute gets node output
func (t *DebugNodeOutputTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if t.storage == nil {
		return mcp.ErrorContent("Storage not initialized"), nil
	}

	executionId := GetString(args, "executionId")
	nodeName := GetString(args, "nodeName")

	execution, err := t.storage.GetExecution(executionId)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to get execution: %v", err)), nil
	}
	if execution == nil {
		return mcp.ErrorContent(fmt.Sprintf("Execution not found: %s", executionId)), nil
	}

	if execution.NodeData == nil {
		return mcp.ErrorContent("No node data available for this execution"), nil
	}

	nodeData, ok := execution.NodeData[nodeName]
	if !ok {
		// List available nodes
		availableNodes := make([]string, 0, len(execution.NodeData))
		for name := range execution.NodeData {
			availableNodes = append(availableNodes, name)
		}
		return mcp.ErrorContent(fmt.Sprintf("Node '%s' not found. Available nodes: %v", nodeName, availableNodes)), nil
	}

	return mcp.SuccessJSON(map[string]interface{}{
		"executionId": executionId,
		"nodeName":    nodeName,
		"itemCount":   len(nodeData),
		"data":        nodeData,
	}), nil
}

// DebugListEventsTool lists audit events
type DebugListEventsTool struct {
	*BaseTool
	storage storage.WorkflowStorage
}

// NewDebugListEventsTool creates a new debug list events tool
func NewDebugListEventsTool(store storage.WorkflowStorage) *DebugListEventsTool {
	return &DebugListEventsTool{
		BaseTool: NewBaseTool(
			"debug_list_events",
			"List recent audit events (execution started, completed, failed, etc.).",
			ObjectSchema(map[string]interface{}{
				"eventType":  StringEnumProp("Filter by event type", []string{"execution.started", "execution.completed", "execution.failed", "workflow.created", "workflow.updated", "workflow.deleted"}),
				"workflowId": StringProp("Filter by workflow ID"),
				"limit":      IntPropWithDefault("Maximum number of events", 50),
				"since":      StringProp("Only events after this timestamp (RFC3339)"),
			}, nil),
		),
		storage: store,
	}
}

// Execute lists events
func (t *DebugListEventsTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if t.storage == nil {
		return mcp.ErrorContent("Storage not initialized"), nil
	}

	limit := GetIntOr(args, "limit", 50)
	workflowId := GetString(args, "workflowId")
	eventType := GetString(args, "eventType")

	// Get recent executions as a proxy for events
	filters := storage.ExecutionFilters{
		WorkflowID: workflowId,
		Limit:      limit,
	}

	executions, _, err := t.storage.ListExecutions(filters)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to list executions: %v", err)), nil
	}

	// Convert executions to events
	events := make([]map[string]interface{}, 0)
	for _, exec := range executions {
		// Filter by event type if specified
		if eventType != "" {
			if eventType == "execution.started" && exec.Status == "running" {
				// Include
			} else if eventType == "execution.completed" && exec.Status == "completed" {
				// Include
			} else if eventType == "execution.failed" && exec.Status == "failed" {
				// Include
			} else {
				continue
			}
		}

		event := map[string]interface{}{
			"timestamp":   exec.StartedAt,
			"eventType":   "execution." + exec.Status,
			"executionId": exec.ID,
			"workflowId":  exec.WorkflowID,
			"mode":        exec.Mode,
		}

		if exec.FinishedAt != nil {
			event["finishedAt"] = exec.FinishedAt
			event["duration"] = exec.FinishedAt.Sub(exec.StartedAt).String()
		}

		if exec.Error != nil {
			event["error"] = exec.Error.Error()
		}

		events = append(events, event)
	}

	return mcp.SuccessJSON(map[string]interface{}{
		"events": events,
		"count":  len(events),
	}), nil
}

// DebugPerformanceTool gets performance metrics
type DebugPerformanceTool struct {
	*BaseTool
	storage storage.WorkflowStorage
}

// NewDebugPerformanceTool creates a new debug performance tool
func NewDebugPerformanceTool(store storage.WorkflowStorage) *DebugPerformanceTool {
	return &DebugPerformanceTool{
		BaseTool: NewBaseTool(
			"debug_performance",
			"Get performance metrics for workflow or node executions.",
			ObjectSchema(map[string]interface{}{
				"workflowId": StringProp("Workflow ID to analyze"),
				"limit":      IntPropWithDefault("Number of recent executions to analyze", 100),
			}, nil),
		),
		storage: store,
	}
}

// Execute gets performance metrics
func (t *DebugPerformanceTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if t.storage == nil {
		return mcp.ErrorContent("Storage not initialized"), nil
	}

	workflowId := GetString(args, "workflowId")
	limit := GetIntOr(args, "limit", 100)

	filters := storage.ExecutionFilters{
		WorkflowID: workflowId,
		Limit:      limit,
	}

	executions, total, err := t.storage.ListExecutions(filters)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to list executions: %v", err)), nil
	}

	// Calculate metrics
	var totalDuration time.Duration
	var completedCount, failedCount int
	var minDuration, maxDuration time.Duration
	durations := make([]time.Duration, 0)

	for _, exec := range executions {
		if exec.FinishedAt == nil {
			continue
		}

		duration := exec.FinishedAt.Sub(exec.StartedAt)
		durations = append(durations, duration)
		totalDuration += duration

		if minDuration == 0 || duration < minDuration {
			minDuration = duration
		}
		if duration > maxDuration {
			maxDuration = duration
		}

		if exec.Status == "completed" {
			completedCount++
		} else if exec.Status == "failed" {
			failedCount++
		}
	}

	var avgDuration time.Duration
	if len(durations) > 0 {
		avgDuration = totalDuration / time.Duration(len(durations))
	}

	successRate := float64(0)
	if completedCount+failedCount > 0 {
		successRate = float64(completedCount) / float64(completedCount+failedCount) * 100
	}

	return mcp.SuccessJSON(map[string]interface{}{
		"workflowId":     workflowId,
		"totalExecutions": total,
		"analyzedCount":  len(executions),
		"metrics": map[string]interface{}{
			"successRate":     fmt.Sprintf("%.1f%%", successRate),
			"completedCount":  completedCount,
			"failedCount":     failedCount,
			"avgDuration":     avgDuration.String(),
			"avgDurationMs":   avgDuration.Milliseconds(),
			"minDuration":     minDuration.String(),
			"maxDuration":     maxDuration.String(),
			"totalDuration":   totalDuration.String(),
		},
	}), nil
}

// DebugLiveStatusTool gets real-time status
type DebugLiveStatusTool struct {
	*BaseTool
	storage storage.WorkflowStorage
	manager *ExecutionManager
}

// NewDebugLiveStatusTool creates a new debug live status tool
func NewDebugLiveStatusTool(store storage.WorkflowStorage, manager *ExecutionManager) *DebugLiveStatusTool {
	return &DebugLiveStatusTool{
		BaseTool: NewBaseTool(
			"debug_live_status",
			"Get real-time status of a running execution, including progress through nodes.",
			ObjectSchema(map[string]interface{}{
				"executionId": StringProp("Execution ID to get live status for"),
			}, []string{"executionId"}),
		),
		storage: store,
		manager: manager,
	}
}

// Execute gets live status
func (t *DebugLiveStatusTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	executionId := GetString(args, "executionId")

	var execution *model.WorkflowExecution

	// Check in-memory first
	if t.manager != nil {
		t.manager.mu.RLock()
		if exec, ok := t.manager.executions[executionId]; ok {
			execution = exec
		}
		t.manager.mu.RUnlock()
	}

	// Fall back to storage
	if execution == nil && t.storage != nil {
		var err error
		execution, err = t.storage.GetExecution(executionId)
		if err != nil {
			return mcp.ErrorContent(fmt.Sprintf("Failed to get execution: %v", err)), nil
		}
	}

	if execution == nil {
		return mcp.ErrorContent(fmt.Sprintf("Execution not found: %s", executionId)), nil
	}

	elapsed := time.Since(execution.StartedAt)

	result := map[string]interface{}{
		"executionId":  executionId,
		"workflowId":   execution.WorkflowID,
		"status":       execution.Status,
		"mode":         execution.Mode,
		"startedAt":    execution.StartedAt,
		"elapsed":      elapsed.String(),
		"elapsedMs":    elapsed.Milliseconds(),
	}

	if execution.FinishedAt != nil {
		result["finishedAt"] = execution.FinishedAt
		result["duration"] = execution.FinishedAt.Sub(execution.StartedAt).String()
	}

	if execution.Status == "running" {
		result["message"] = "Execution in progress..."
	} else if execution.Status == "completed" {
		result["message"] = "Execution completed successfully"
		if execution.Data != nil {
			result["outputItemCount"] = len(execution.Data)
		}
	} else if execution.Status == "failed" {
		result["message"] = "Execution failed"
		if execution.Error != nil {
			result["error"] = execution.Error.Error()
		}
	}

	return mcp.SuccessJSON(result), nil
}

// RegisterDebugTools registers all debugging tools with a registry
func RegisterDebugTools(registry *Registry, store storage.WorkflowStorage, manager *ExecutionManager) {
	registry.Register(NewDebugExecutionLogsTool(store))
	registry.Register(NewDebugNodeOutputTool(store))
	registry.Register(NewDebugListEventsTool(store))
	registry.Register(NewDebugPerformanceTool(store))
	registry.Register(NewDebugLiveStatusTool(store, manager))
}
