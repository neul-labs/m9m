package variables

import (
	"time"
)

// Variable represents a configuration variable
type Variable struct {
	ID          string                 `json:"id"`
	Key         string                 `json:"key"`
	Value       string                 `json:"value"`
	Type        VariableType           `json:"type"`
	Description string                 `json:"description,omitempty"`
	Encrypted   bool                   `json:"encrypted"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// VariableType represents the scope/type of a variable
type VariableType string

const (
	// GlobalVariable is accessible across all workflows and environments
	GlobalVariable VariableType = "global"

	// WorkflowVariable is scoped to a specific workflow
	WorkflowVariable VariableType = "workflow"

	// EnvironmentVariable is scoped to a specific environment
	EnvironmentVariable VariableType = "environment"
)

// Environment represents an execution environment (dev, staging, prod, etc.)
type Environment struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Key         string                 `json:"key"` // e.g., "dev", "staging", "prod"
	Description string                 `json:"description,omitempty"`
	Variables   map[string]string      `json:"variables"` // Key-value pairs
	Active      bool                   `json:"active"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// WorkflowVariables represents variables specific to a workflow
type WorkflowVariables struct {
	WorkflowID string            `json:"workflowId"`
	Variables  map[string]string `json:"variables"`
	UpdatedAt  time.Time         `json:"updatedAt"`
}

// VariableCreateRequest represents a request to create a variable
type VariableCreateRequest struct {
	Key         string                 `json:"key"`
	Value       string                 `json:"value"`
	Type        VariableType           `json:"type"`
	Description string                 `json:"description,omitempty"`
	Encrypted   bool                   `json:"encrypted"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// VariableUpdateRequest represents a request to update a variable
type VariableUpdateRequest struct {
	Value       *string                `json:"value,omitempty"`
	Description *string                `json:"description,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// EnvironmentCreateRequest represents a request to create an environment
type EnvironmentCreateRequest struct {
	Name        string                 `json:"name"`
	Key         string                 `json:"key"`
	Description string                 `json:"description,omitempty"`
	Variables   map[string]string      `json:"variables,omitempty"`
	Active      bool                   `json:"active"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// EnvironmentUpdateRequest represents a request to update an environment
type EnvironmentUpdateRequest struct {
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	Variables   map[string]string      `json:"variables,omitempty"`
	Active      *bool                  `json:"active,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// VariableListFilters defines filters for listing variables
type VariableListFilters struct {
	Type   VariableType
	Tags   []string
	Search string
	Limit  int
	Offset int
}

// VariableContext holds all variables available during workflow execution
type VariableContext struct {
	Global      map[string]string `json:"global"`
	Environment map[string]string `json:"environment"`
	Workflow    map[string]string `json:"workflow"`
}

// GetValue retrieves a variable value with priority: workflow > environment > global
func (vc *VariableContext) GetValue(key string) (string, bool) {
	// Check workflow variables first (highest priority)
	if val, exists := vc.Workflow[key]; exists {
		return val, true
	}

	// Check environment variables
	if val, exists := vc.Environment[key]; exists {
		return val, true
	}

	// Check global variables (lowest priority)
	if val, exists := vc.Global[key]; exists {
		return val, true
	}

	return "", false
}

// MergeAll returns all variables merged with proper priority
func (vc *VariableContext) MergeAll() map[string]string {
	merged := make(map[string]string)

	// Start with global (lowest priority)
	for k, v := range vc.Global {
		merged[k] = v
	}

	// Override with environment
	for k, v := range vc.Environment {
		merged[k] = v
	}

	// Override with workflow (highest priority)
	for k, v := range vc.Workflow {
		merged[k] = v
	}

	return merged
}
