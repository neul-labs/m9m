/*
Package base provides the base interfaces and types for node implementations.
*/
package base

import (
	"context"
	"fmt"
	"github.com/neul-labs/m9m/internal/expressions"
	"github.com/neul-labs/m9m/internal/model"
)

// NodeExecutor is the interface that all node types must implement
type NodeExecutor interface {
	// Execute processes the node with given input data and node parameters
	Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error)

	// Description returns metadata about the node
	Description() NodeDescription

	// ValidateParameters validates node parameters
	ValidateParameters(params map[string]interface{}) error
}

// ContextAwareNodeExecutor optionally supports context-aware node execution.
// Engines should prefer this interface when available to support cancellation.
type ContextAwareNodeExecutor interface {
	NodeExecutor
	ExecuteWithContext(ctx context.Context, inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error)
}

// NodeDescription provides metadata about a node type
type NodeDescription struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// NodeFactory is a function that creates a new node executor
type NodeFactory func() NodeExecutor

// BaseNode provides common functionality for node implementations
type BaseNode struct {
	description NodeDescription
}

// NewBaseNode creates a new base node
func NewBaseNode(description NodeDescription) *BaseNode {
	return &BaseNode{
		description: description,
	}
}

// Description returns the node description
func (b *BaseNode) Description() NodeDescription {
	return b.description
}

// ValidateParameters validates that required parameters are present
func (b *BaseNode) ValidateParameters(params map[string]interface{}) error {
	// Base implementation does minimal validation
	// Specific nodes should override this with their own validation logic
	if params == nil {
		return nil
	}

	// Just ensure params is a valid map
	// In a real implementation, we would validate against a schema
	return nil
}

// Execute is a default implementation that returns input data unchanged
// Specific nodes should override this with their own execution logic
func (b *BaseNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	// Base implementation just returns the input data
	// This should be overridden by specific node implementations
	return inputData, nil
}

// GetParameter retrieves a parameter value with a default fallback
func (b *BaseNode) GetParameter(params map[string]interface{}, name string, defaultValue interface{}) interface{} {
	if params == nil {
		return defaultValue
	}

	if value, exists := params[name]; exists {
		return value
	}

	return defaultValue
}

// GetStringParameter retrieves a string parameter with a default fallback
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

// GetIntParameter retrieves an integer parameter with a default fallback
func (b *BaseNode) GetIntParameter(params map[string]interface{}, name string, defaultValue int) int {
	if params == nil {
		return defaultValue
	}

	if value, exists := params[name]; exists {
		if num, ok := value.(int); ok {
			return num
		}
		if floatVal, ok := value.(float64); ok {
			return int(floatVal)
		}
	}

	return defaultValue
}

// GetBoolParameter retrieves a boolean parameter with a default fallback
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

// CreateError creates a standardized error for node execution
func (b *BaseNode) CreateError(message string, data map[string]interface{}) error {
	return fmt.Errorf("node %s error: %s", b.description.Name, message)
}

// EvaluateExpressions evaluates expressions in parameters using the given context
func (b *BaseNode) EvaluateExpressions(params map[string]interface{}, context *expressions.ExecutionContext) (map[string]interface{}, error) {
	evaluator := expressions.NewExpressionEvaluator()

	evaluatedParams := make(map[string]interface{})

	for key, value := range params {
		switch v := value.(type) {
		case string:
			// Check if this is an expression
			if expressions.IsExpression(v) {
				result, err := evaluator.Evaluate(v, context)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate expression for parameter %s: %v", key, err)
				}
				evaluatedParams[key] = result
			} else {
				evaluatedParams[key] = v
			}
		case map[string]interface{}:
			// Recursively evaluate nested maps
			nested, err := b.EvaluateExpressions(v, context)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate nested parameters for %s: %v", key, err)
			}
			evaluatedParams[key] = nested
		default:
			evaluatedParams[key] = v
		}
	}

	return evaluatedParams, nil
}

// Additional type definitions for compatibility with some node implementations

// Node is a compatibility type for older node implementations
type Node interface {
	Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error)
}

// NodeMetadata provides extended metadata about a node
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

// NodeProperty describes a node parameter/property
type NodeProperty struct {
	DisplayName string      `json:"displayName"`
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Default     interface{} `json:"default"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Options     []Option    `json:"options,omitempty"`
}

// Option represents a selection option for a property
type Option struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ExecutionParams holds parameters for node execution
type ExecutionParams struct {
	CurrentNodeName string
	InputData       []model.DataItem
	RunIndex        int
	ItemIndex       int
	NodeParams      map[string]interface{}
}

// NodeOutput represents the output from a node
type NodeOutput struct {
	Data  []model.DataItem
	Error error
}
