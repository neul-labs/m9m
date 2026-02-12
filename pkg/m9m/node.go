package m9m

import (
	"fmt"
)

// NodeExecutor is the interface that all custom node types must implement.
type NodeExecutor interface {
	// Execute processes the node with given input data and parameters.
	// It returns the output data items or an error if execution fails.
	Execute(inputData []DataItem, nodeParams map[string]interface{}) ([]DataItem, error)

	// Description returns metadata about the node.
	Description() NodeDescription

	// ValidateParameters validates node parameters before execution.
	// Return nil if parameters are valid, or an error describing the issue.
	ValidateParameters(params map[string]interface{}) error
}

// NodeDescription provides metadata about a node type.
type NodeDescription struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// BaseNode provides common functionality for node implementations.
// Embed this in your custom node structs to get default implementations.
type BaseNode struct {
	description NodeDescription
}

// NewBaseNode creates a new base node with the given description.
func NewBaseNode(description NodeDescription) *BaseNode {
	return &BaseNode{
		description: description,
	}
}

// Description returns the node description.
func (b *BaseNode) Description() NodeDescription {
	return b.description
}

// ValidateParameters provides a default implementation that accepts all parameters.
// Override this in your node implementation for custom validation.
func (b *BaseNode) ValidateParameters(params map[string]interface{}) error {
	return nil
}

// Execute is a default implementation that returns input data unchanged.
// Override this in your node implementation with custom execution logic.
func (b *BaseNode) Execute(inputData []DataItem, nodeParams map[string]interface{}) ([]DataItem, error) {
	return inputData, nil
}

// GetParameter retrieves a parameter value with a default fallback.
func (b *BaseNode) GetParameter(params map[string]interface{}, name string, defaultValue interface{}) interface{} {
	if params == nil {
		return defaultValue
	}
	if value, exists := params[name]; exists {
		return value
	}
	return defaultValue
}

// GetStringParameter retrieves a string parameter with a default fallback.
func (b *BaseNode) GetStringParameter(params map[string]interface{}, name string, defaultValue string) string {
	if params == nil {
		return defaultValue
	}
	if value, exists := params[name]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

// GetIntParameter retrieves an integer parameter with a default fallback.
func (b *BaseNode) GetIntParameter(params map[string]interface{}, name string, defaultValue int) int {
	if params == nil {
		return defaultValue
	}
	if value, exists := params[name]; exists {
		switch v := value.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return defaultValue
}

// GetFloat64Parameter retrieves a float64 parameter with a default fallback.
func (b *BaseNode) GetFloat64Parameter(params map[string]interface{}, name string, defaultValue float64) float64 {
	if params == nil {
		return defaultValue
	}
	if value, exists := params[name]; exists {
		switch v := value.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case int64:
			return float64(v)
		}
	}
	return defaultValue
}

// GetBoolParameter retrieves a boolean parameter with a default fallback.
func (b *BaseNode) GetBoolParameter(params map[string]interface{}, name string, defaultValue bool) bool {
	if params == nil {
		return defaultValue
	}
	if value, exists := params[name]; exists {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// GetSliceParameter retrieves a slice parameter with a default fallback.
func (b *BaseNode) GetSliceParameter(params map[string]interface{}, name string) []interface{} {
	if params == nil {
		return nil
	}
	if value, exists := params[name]; exists {
		if slice, ok := value.([]interface{}); ok {
			return slice
		}
	}
	return nil
}

// GetMapParameter retrieves a map parameter with a default fallback.
func (b *BaseNode) GetMapParameter(params map[string]interface{}, name string) map[string]interface{} {
	if params == nil {
		return nil
	}
	if value, exists := params[name]; exists {
		if m, ok := value.(map[string]interface{}); ok {
			return m
		}
	}
	return nil
}

// CreateError creates a standardized error for node execution.
func (b *BaseNode) CreateError(message string) error {
	return fmt.Errorf("node %s error: %s", b.description.Name, message)
}

// CreateErrorf creates a standardized formatted error for node execution.
func (b *BaseNode) CreateErrorf(format string, args ...interface{}) error {
	return fmt.Errorf("node %s error: %s", b.description.Name, fmt.Sprintf(format, args...))
}

// NodeProperty describes a node parameter/property for documentation purposes.
type NodeProperty struct {
	DisplayName string      `json:"displayName"`
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Default     interface{} `json:"default"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Options     []SelectOption `json:"options,omitempty"`
}

// SelectOption represents a selection option for a property.
type SelectOption struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// NodeMetadata provides extended metadata about a node for registration.
type NodeMetadata struct {
	Name        string                 `json:"name"`
	DisplayName string                 `json:"displayName"`
	Description string                 `json:"description"`
	Version     int                    `json:"version"`
	Defaults    map[string]interface{} `json:"defaults"`
	Inputs      []string               `json:"inputs"`
	Outputs     []string               `json:"outputs"`
	Properties  []NodeProperty         `json:"properties"`
}

// FunctionNode is a helper for creating simple nodes from functions.
type FunctionNode struct {
	*BaseNode
	executeFunc func(inputData []DataItem, params map[string]interface{}) ([]DataItem, error)
}

// NewFunctionNode creates a node from a function.
func NewFunctionNode(desc NodeDescription, fn func([]DataItem, map[string]interface{}) ([]DataItem, error)) *FunctionNode {
	return &FunctionNode{
		BaseNode:    NewBaseNode(desc),
		executeFunc: fn,
	}
}

// Execute calls the wrapped function.
func (n *FunctionNode) Execute(inputData []DataItem, nodeParams map[string]interface{}) ([]DataItem, error) {
	if n.executeFunc == nil {
		return inputData, nil
	}
	return n.executeFunc(inputData, nodeParams)
}
