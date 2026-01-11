package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/dipankar/m9m/internal/mcp"
	"github.com/dipankar/m9m/internal/model"
	"github.com/dipankar/m9m/internal/storage"
	"github.com/google/uuid"
)

// WorkflowTools provides workflow management tools
type WorkflowTools struct {
	storage storage.WorkflowStorage
}

// NewWorkflowTools creates a new set of workflow tools
func NewWorkflowTools(store storage.WorkflowStorage) *WorkflowTools {
	return &WorkflowTools{storage: store}
}

// WorkflowListTool lists workflows
type WorkflowListTool struct {
	*BaseTool
	storage storage.WorkflowStorage
}

// NewWorkflowListTool creates a new workflow list tool
func NewWorkflowListTool(store storage.WorkflowStorage) *WorkflowListTool {
	return &WorkflowListTool{
		BaseTool: NewBaseTool(
			"workflow_list",
			"List all workflows with optional filtering by active status, search term, or tags.",
			ObjectSchema(map[string]interface{}{
				"active": BoolProp("Filter by active status"),
				"search": StringProp("Search term to filter by name or description"),
				"tags":   ArrayProp("Filter by tags", map[string]interface{}{"type": "string"}),
				"limit":  IntPropWithDefault("Maximum number of results", 50),
				"offset": IntPropWithDefault("Offset for pagination", 0),
			}, nil),
		),
		storage: store,
	}
}

// Execute lists workflows
func (t *WorkflowListTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if t.storage == nil {
		return mcp.ErrorContent("Storage not initialized"), nil
	}

	filters := storage.WorkflowFilters{
		Limit:  GetIntOr(args, "limit", 50),
		Offset: GetIntOr(args, "offset", 0),
		Search: GetString(args, "search"),
	}

	if active, ok := args["active"].(bool); ok {
		filters.Active = &active
	}

	if tagsArr := GetArray(args, "tags"); tagsArr != nil {
		tags := make([]string, 0, len(tagsArr))
		for _, t := range tagsArr {
			if s, ok := t.(string); ok {
				tags = append(tags, s)
			}
		}
		filters.Tags = tags
	}

	workflows, total, err := t.storage.ListWorkflows(filters)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to list workflows: %v", err)), nil
	}

	// Create summaries without full node details
	summaries := make([]map[string]interface{}, 0, len(workflows))
	for _, wf := range workflows {
		summaries = append(summaries, map[string]interface{}{
			"id":          wf.ID,
			"name":        wf.Name,
			"description": wf.Description,
			"active":      wf.Active,
			"nodeCount":   len(wf.Nodes),
			"tags":        wf.Tags,
			"createdAt":   wf.CreatedAt,
			"updatedAt":   wf.UpdatedAt,
		})
	}

	return mcp.SuccessJSON(map[string]interface{}{
		"workflows": summaries,
		"total":     total,
		"limit":     filters.Limit,
		"offset":    filters.Offset,
	}), nil
}

// WorkflowGetTool gets a specific workflow
type WorkflowGetTool struct {
	*BaseTool
	storage storage.WorkflowStorage
}

// NewWorkflowGetTool creates a new workflow get tool
func NewWorkflowGetTool(store storage.WorkflowStorage) *WorkflowGetTool {
	return &WorkflowGetTool{
		BaseTool: NewBaseTool(
			"workflow_get",
			"Get a specific workflow by ID with full details including nodes and connections.",
			ObjectSchema(map[string]interface{}{
				"id": StringProp("Workflow ID"),
			}, []string{"id"}),
		),
		storage: store,
	}
}

// Execute gets a workflow
func (t *WorkflowGetTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if t.storage == nil {
		return mcp.ErrorContent("Storage not initialized"), nil
	}

	id := GetString(args, "id")

	workflow, err := t.storage.GetWorkflow(id)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to get workflow: %v", err)), nil
	}

	if workflow == nil {
		return mcp.ErrorContent(fmt.Sprintf("Workflow not found: %s", id)), nil
	}

	return mcp.SuccessJSON(workflow), nil
}

// WorkflowCreateTool creates a new workflow
type WorkflowCreateTool struct {
	*BaseTool
	storage storage.WorkflowStorage
}

// NewWorkflowCreateTool creates a new workflow create tool
func NewWorkflowCreateTool(store storage.WorkflowStorage) *WorkflowCreateTool {
	return &WorkflowCreateTool{
		BaseTool: NewBaseTool(
			"workflow_create",
			"Create a new workflow with nodes and connections. Returns the created workflow with its ID.",
			ObjectSchema(map[string]interface{}{
				"name":        StringProp("Workflow name"),
				"description": StringProp("Workflow description"),
				"nodes": ArrayProp("Array of node definitions", map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name":       map[string]interface{}{"type": "string", "description": "Node name (unique within workflow)"},
						"type":       map[string]interface{}{"type": "string", "description": "Node type (e.g., n8n-nodes-base.httpRequest)"},
						"parameters": map[string]interface{}{"type": "object", "description": "Node parameters"},
						"position":   map[string]interface{}{"type": "array", "description": "Position [x, y]"},
					},
				}),
				"connections": ObjectProp("Node connections mapping source node to targets"),
				"active":      BoolPropWithDefault("Whether the workflow is active", false),
				"tags":        ArrayProp("Workflow tags", map[string]interface{}{"type": "string"}),
			}, []string{"name", "nodes"}),
		),
		storage: store,
	}
}

// Execute creates a workflow
func (t *WorkflowCreateTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if t.storage == nil {
		return mcp.ErrorContent("Storage not initialized"), nil
	}

	name := GetString(args, "name")
	description := GetString(args, "description")
	active := GetBool(args, "active")
	nodesArg := GetArray(args, "nodes")
	connectionsArg := GetMap(args, "connections")

	// Parse nodes
	nodes := make([]model.Node, 0)
	if nodesArg != nil {
		for i, n := range nodesArg {
			nodeMap, ok := n.(map[string]interface{})
			if !ok {
				continue
			}

			node := model.Node{
				ID:         fmt.Sprintf("node_%d", i),
				Name:       GetString(nodeMap, "name"),
				Type:       GetString(nodeMap, "type"),
				Parameters: GetMap(nodeMap, "parameters"),
			}

			if pos := GetArray(nodeMap, "position"); pos != nil && len(pos) >= 2 {
				x, _ := pos[0].(float64)
				y, _ := pos[1].(float64)
				node.Position = []int{int(x), int(y)}
			} else {
				// Default position
				node.Position = []int{250 + i*200, 300}
			}

			nodes = append(nodes, node)
		}
	}

	// Parse connections
	connections := make(map[string]model.Connections)
	if connectionsArg != nil {
		for sourceName, conns := range connectionsArg {
			if connMap, ok := conns.(map[string]interface{}); ok {
				nodeConns := model.Connections{}
				if mainConns, ok := connMap["main"].([]interface{}); ok {
					for _, mc := range mainConns {
						if connArr, ok := mc.([]interface{}); ok {
							var connList []model.Connection
							for _, c := range connArr {
								if cm, ok := c.(map[string]interface{}); ok {
									connList = append(connList, model.Connection{
										Node:  GetString(cm, "node"),
										Type:  GetStringOr(cm, "type", "main"),
										Index: GetInt(cm, "index"),
									})
								}
							}
							nodeConns.Main = append(nodeConns.Main, connList)
						}
					}
				}
				connections[sourceName] = nodeConns
			}
		}
	}

	// Parse tags
	var tags []string
	if tagsArg := GetArray(args, "tags"); tagsArg != nil {
		for _, t := range tagsArg {
			if s, ok := t.(string); ok {
				tags = append(tags, s)
			}
		}
	}

	workflow := &model.Workflow{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Active:      active,
		Nodes:       nodes,
		Connections: connections,
		Tags:        tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := t.storage.SaveWorkflow(workflow); err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to save workflow: %v", err)), nil
	}

	return mcp.SuccessJSON(map[string]interface{}{
		"success":  true,
		"workflow": workflow,
		"message":  fmt.Sprintf("Workflow '%s' created with ID: %s", name, workflow.ID),
	}), nil
}

// WorkflowUpdateTool updates an existing workflow
type WorkflowUpdateTool struct {
	*BaseTool
	storage storage.WorkflowStorage
}

// NewWorkflowUpdateTool creates a new workflow update tool
func NewWorkflowUpdateTool(store storage.WorkflowStorage) *WorkflowUpdateTool {
	return &WorkflowUpdateTool{
		BaseTool: NewBaseTool(
			"workflow_update",
			"Update an existing workflow. Only provided fields will be updated.",
			ObjectSchema(map[string]interface{}{
				"id":          StringProp("Workflow ID to update"),
				"name":        StringProp("New workflow name"),
				"description": StringProp("New description"),
				"nodes":       ArrayProp("Updated nodes", map[string]interface{}{"type": "object"}),
				"connections": ObjectProp("Updated connections"),
				"tags":        ArrayProp("Updated tags", map[string]interface{}{"type": "string"}),
			}, []string{"id"}),
		),
		storage: store,
	}
}

// Execute updates a workflow
func (t *WorkflowUpdateTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if t.storage == nil {
		return mcp.ErrorContent("Storage not initialized"), nil
	}

	id := GetString(args, "id")

	workflow, err := t.storage.GetWorkflow(id)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to get workflow: %v", err)), nil
	}
	if workflow == nil {
		return mcp.ErrorContent(fmt.Sprintf("Workflow not found: %s", id)), nil
	}

	// Update fields if provided
	if name := GetString(args, "name"); name != "" {
		workflow.Name = name
	}
	if desc := GetString(args, "description"); desc != "" {
		workflow.Description = desc
	}

	workflow.UpdatedAt = time.Now()

	if err := t.storage.UpdateWorkflow(id, workflow); err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to update workflow: %v", err)), nil
	}

	return mcp.SuccessJSON(map[string]interface{}{
		"success":  true,
		"workflow": workflow,
		"message":  fmt.Sprintf("Workflow '%s' updated", workflow.Name),
	}), nil
}

// WorkflowDeleteTool deletes a workflow
type WorkflowDeleteTool struct {
	*BaseTool
	storage storage.WorkflowStorage
}

// NewWorkflowDeleteTool creates a new workflow delete tool
func NewWorkflowDeleteTool(store storage.WorkflowStorage) *WorkflowDeleteTool {
	return &WorkflowDeleteTool{
		BaseTool: NewBaseTool(
			"workflow_delete",
			"Delete a workflow by ID. This action cannot be undone.",
			ObjectSchema(map[string]interface{}{
				"id": StringProp("Workflow ID to delete"),
			}, []string{"id"}),
		),
		storage: store,
	}
}

// Execute deletes a workflow
func (t *WorkflowDeleteTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if t.storage == nil {
		return mcp.ErrorContent("Storage not initialized"), nil
	}

	id := GetString(args, "id")

	// Verify workflow exists
	workflow, err := t.storage.GetWorkflow(id)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to get workflow: %v", err)), nil
	}
	if workflow == nil {
		return mcp.ErrorContent(fmt.Sprintf("Workflow not found: %s", id)), nil
	}

	if err := t.storage.DeleteWorkflow(id); err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to delete workflow: %v", err)), nil
	}

	return mcp.SuccessJSON(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Workflow '%s' deleted", workflow.Name),
	}), nil
}

// WorkflowActivateTool activates a workflow
type WorkflowActivateTool struct {
	*BaseTool
	storage storage.WorkflowStorage
}

// NewWorkflowActivateTool creates a new workflow activate tool
func NewWorkflowActivateTool(store storage.WorkflowStorage) *WorkflowActivateTool {
	return &WorkflowActivateTool{
		BaseTool: NewBaseTool(
			"workflow_activate",
			"Activate a workflow so it can respond to triggers.",
			ObjectSchema(map[string]interface{}{
				"id": StringProp("Workflow ID to activate"),
			}, []string{"id"}),
		),
		storage: store,
	}
}

// Execute activates a workflow
func (t *WorkflowActivateTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if t.storage == nil {
		return mcp.ErrorContent("Storage not initialized"), nil
	}

	id := GetString(args, "id")

	if err := t.storage.ActivateWorkflow(id); err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to activate workflow: %v", err)), nil
	}

	return mcp.SuccessJSON(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Workflow %s activated", id),
	}), nil
}

// WorkflowDeactivateTool deactivates a workflow
type WorkflowDeactivateTool struct {
	*BaseTool
	storage storage.WorkflowStorage
}

// NewWorkflowDeactivateTool creates a new workflow deactivate tool
func NewWorkflowDeactivateTool(store storage.WorkflowStorage) *WorkflowDeactivateTool {
	return &WorkflowDeactivateTool{
		BaseTool: NewBaseTool(
			"workflow_deactivate",
			"Deactivate a workflow so it stops responding to triggers.",
			ObjectSchema(map[string]interface{}{
				"id": StringProp("Workflow ID to deactivate"),
			}, []string{"id"}),
		),
		storage: store,
	}
}

// Execute deactivates a workflow
func (t *WorkflowDeactivateTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if t.storage == nil {
		return mcp.ErrorContent("Storage not initialized"), nil
	}

	id := GetString(args, "id")

	if err := t.storage.DeactivateWorkflow(id); err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to deactivate workflow: %v", err)), nil
	}

	return mcp.SuccessJSON(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Workflow %s deactivated", id),
	}), nil
}

// WorkflowDuplicateTool duplicates a workflow
type WorkflowDuplicateTool struct {
	*BaseTool
	storage storage.WorkflowStorage
}

// NewWorkflowDuplicateTool creates a new workflow duplicate tool
func NewWorkflowDuplicateTool(store storage.WorkflowStorage) *WorkflowDuplicateTool {
	return &WorkflowDuplicateTool{
		BaseTool: NewBaseTool(
			"workflow_duplicate",
			"Create a copy of an existing workflow with a new name.",
			ObjectSchema(map[string]interface{}{
				"id":      StringProp("Workflow ID to duplicate"),
				"newName": StringProp("Name for the new workflow (optional, defaults to 'Copy of <original>')"),
			}, []string{"id"}),
		),
		storage: store,
	}
}

// Execute duplicates a workflow
func (t *WorkflowDuplicateTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if t.storage == nil {
		return mcp.ErrorContent("Storage not initialized"), nil
	}

	id := GetString(args, "id")
	newName := GetString(args, "newName")

	// Get original workflow
	original, err := t.storage.GetWorkflow(id)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to get workflow: %v", err)), nil
	}
	if original == nil {
		return mcp.ErrorContent(fmt.Sprintf("Workflow not found: %s", id)), nil
	}

	// Create copy
	if newName == "" {
		newName = fmt.Sprintf("Copy of %s", original.Name)
	}

	copy := &model.Workflow{
		ID:          uuid.New().String(),
		Name:        newName,
		Description: original.Description,
		Active:      false, // New workflows start inactive
		Nodes:       original.Nodes,
		Connections: original.Connections,
		Settings:    original.Settings,
		Tags:        original.Tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := t.storage.SaveWorkflow(copy); err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to save workflow copy: %v", err)), nil
	}

	return mcp.SuccessJSON(map[string]interface{}{
		"success":      true,
		"workflow":     copy,
		"message":      fmt.Sprintf("Workflow duplicated as '%s' with ID: %s", newName, copy.ID),
		"originalId":   id,
		"originalName": original.Name,
	}), nil
}

// WorkflowValidateTool validates a workflow structure
type WorkflowValidateTool struct {
	*BaseTool
}

// NewWorkflowValidateTool creates a new workflow validate tool
func NewWorkflowValidateTool() *WorkflowValidateTool {
	return &WorkflowValidateTool{
		BaseTool: NewBaseTool(
			"workflow_validate",
			"Validate a workflow structure before saving. Checks for node types, connections, and potential issues.",
			ObjectSchema(map[string]interface{}{
				"name":        StringProp("Workflow name"),
				"nodes":       ArrayProp("Array of node definitions", map[string]interface{}{"type": "object"}),
				"connections": ObjectProp("Node connections"),
			}, []string{"nodes"}),
		),
	}
}

// Execute validates a workflow
func (t *WorkflowValidateTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	nodesArg := GetArray(args, "nodes")
	connectionsArg := GetMap(args, "connections")

	errors := make([]string, 0)
	warnings := make([]string, 0)

	// Validate nodes
	nodeNames := make(map[string]bool)
	if nodesArg == nil || len(nodesArg) == 0 {
		errors = append(errors, "Workflow must have at least one node")
	} else {
		for i, n := range nodesArg {
			nodeMap, ok := n.(map[string]interface{})
			if !ok {
				errors = append(errors, fmt.Sprintf("Node %d is not a valid object", i))
				continue
			}

			name := GetString(nodeMap, "name")
			nodeType := GetString(nodeMap, "type")

			if name == "" {
				errors = append(errors, fmt.Sprintf("Node %d is missing 'name'", i))
			} else if nodeNames[name] {
				errors = append(errors, fmt.Sprintf("Duplicate node name: %s", name))
			} else {
				nodeNames[name] = true
			}

			if nodeType == "" {
				errors = append(errors, fmt.Sprintf("Node '%s' is missing 'type'", name))
			}
		}
	}

	// Validate connections
	if connectionsArg != nil {
		for sourceName := range connectionsArg {
			if !nodeNames[sourceName] {
				errors = append(errors, fmt.Sprintf("Connection from unknown node: %s", sourceName))
			}
		}
	}

	// Check for disconnected nodes
	connectedNodes := make(map[string]bool)
	if connectionsArg != nil {
		for sourceName, conns := range connectionsArg {
			connectedNodes[sourceName] = true
			if connMap, ok := conns.(map[string]interface{}); ok {
				if mainConns, ok := connMap["main"].([]interface{}); ok {
					for _, mc := range mainConns {
						if connArr, ok := mc.([]interface{}); ok {
							for _, c := range connArr {
								if cm, ok := c.(map[string]interface{}); ok {
									targetNode := GetString(cm, "node")
									connectedNodes[targetNode] = true
								}
							}
						}
					}
				}
			}
		}
	}

	for name := range nodeNames {
		if !connectedNodes[name] && len(nodeNames) > 1 {
			warnings = append(warnings, fmt.Sprintf("Node '%s' is not connected to any other node", name))
		}
	}

	isValid := len(errors) == 0

	return mcp.SuccessJSON(map[string]interface{}{
		"valid":    isValid,
		"errors":   errors,
		"warnings": warnings,
	}), nil
}

// RegisterWorkflowTools registers all workflow management tools with a registry
func RegisterWorkflowTools(registry *Registry, store storage.WorkflowStorage) {
	registry.Register(NewWorkflowListTool(store))
	registry.Register(NewWorkflowGetTool(store))
	registry.Register(NewWorkflowCreateTool(store))
	registry.Register(NewWorkflowUpdateTool(store))
	registry.Register(NewWorkflowDeleteTool(store))
	registry.Register(NewWorkflowActivateTool(store))
	registry.Register(NewWorkflowDeactivateTool(store))
	registry.Register(NewWorkflowDuplicateTool(store))
	registry.Register(NewWorkflowValidateTool())
}
