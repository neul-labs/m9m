// Package tools provides MCP tool implementations for m9m.
package tools

import (
	"context"
	"sync"

	"github.com/neul-labs/m9m/internal/mcp"
)

// Tool is the interface that all MCP tools must implement
type Tool interface {
	// Name returns the tool name
	Name() string

	// Description returns the tool description
	Description() string

	// InputSchema returns the JSON Schema for the tool's input
	InputSchema() map[string]interface{}

	// Execute runs the tool with the given arguments
	Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error)
}

// Registry manages tool registration and lookup
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry
func (r *Registry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
}

// Get returns a tool by name
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

// List returns all registered tools
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ListDefinitions returns MCP tool definitions for all registered tools
func (r *Registry) ListDefinitions() []mcp.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]mcp.Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, mcp.Tool{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: tool.InputSchema(),
		})
	}
	return tools
}

// RegisterAll registers all tools with an MCP server
func (r *Registry) RegisterAll(server *mcp.Server) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for name, tool := range r.tools {
		t := tool // capture for closure
		server.RegisterTool(name, func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
			return t.Execute(ctx, args)
		})
	}
}

// BaseTool provides common functionality for tools
type BaseTool struct {
	name        string
	description string
	schema      map[string]interface{}
}

// NewBaseTool creates a new base tool
func NewBaseTool(name, description string, schema map[string]interface{}) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		schema:      schema,
	}
}

// Name returns the tool name
func (t *BaseTool) Name() string {
	return t.name
}

// Description returns the tool description
func (t *BaseTool) Description() string {
	return t.description
}

// InputSchema returns the tool's input schema
func (t *BaseTool) InputSchema() map[string]interface{} {
	return t.schema
}

// Helper functions for building schemas

// ObjectSchema creates an object schema with properties
func ObjectSchema(properties map[string]interface{}, required []string) map[string]interface{} {
	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// StringProp creates a string property
func StringProp(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "string",
		"description": description,
	}
}

// StringPropWithDefault creates a string property with a default
func StringPropWithDefault(description, defaultValue string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "string",
		"description": description,
		"default":     defaultValue,
	}
}

// StringEnumProp creates a string property with enum values
func StringEnumProp(description string, values []string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "string",
		"description": description,
		"enum":        values,
	}
}

// IntProp creates an integer property
func IntProp(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "integer",
		"description": description,
	}
}

// IntPropWithDefault creates an integer property with a default
func IntPropWithDefault(description string, defaultValue int) map[string]interface{} {
	return map[string]interface{}{
		"type":        "integer",
		"description": description,
		"default":     defaultValue,
	}
}

// BoolProp creates a boolean property
func BoolProp(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "boolean",
		"description": description,
	}
}

// BoolPropWithDefault creates a boolean property with a default
func BoolPropWithDefault(description string, defaultValue bool) map[string]interface{} {
	return map[string]interface{}{
		"type":        "boolean",
		"description": description,
		"default":     defaultValue,
	}
}

// ArrayProp creates an array property
func ArrayProp(description string, items map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"type":        "array",
		"description": description,
		"items":       items,
	}
}

// ObjectProp creates a nested object property
func ObjectProp(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"description": description,
	}
}

// AnyProp creates an any-type property
func AnyProp(description string) map[string]interface{} {
	return map[string]interface{}{
		"description": description,
	}
}

// Helper functions for getting typed values from args

// GetString gets a string from args
func GetString(args map[string]interface{}, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetStringOr gets a string from args with a default
func GetStringOr(args map[string]interface{}, key, defaultValue string) string {
	if v := GetString(args, key); v != "" {
		return v
	}
	return defaultValue
}

// GetInt gets an int from args
func GetInt(args map[string]interface{}, key string) int {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case int64:
			return int(n)
		case float64:
			return int(n)
		}
	}
	return 0
}

// GetIntOr gets an int from args with a default
func GetIntOr(args map[string]interface{}, key string, defaultValue int) int {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case int64:
			return int(n)
		case float64:
			return int(n)
		}
	}
	return defaultValue
}

// GetBool gets a bool from args
func GetBool(args map[string]interface{}, key string) bool {
	if v, ok := args[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// GetBoolOr gets a bool from args with a default
func GetBoolOr(args map[string]interface{}, key string, defaultValue bool) bool {
	if v, ok := args[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// GetMap gets a map from args
func GetMap(args map[string]interface{}, key string) map[string]interface{} {
	if v, ok := args[key]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			return m
		}
	}
	return nil
}

// GetArray gets an array from args
func GetArray(args map[string]interface{}, key string) []interface{} {
	if v, ok := args[key]; ok {
		if a, ok := v.([]interface{}); ok {
			return a
		}
	}
	return nil
}
