package storage

import (
	"time"

	"github.com/dipankar/m9m/internal/model"
)

// WorkflowStorage defines the interface for workflow persistence
type WorkflowStorage interface {
	// Workflow operations
	SaveWorkflow(workflow *model.Workflow) error
	GetWorkflow(id string) (*model.Workflow, error)
	ListWorkflows(filters WorkflowFilters) ([]*model.Workflow, int, error)
	UpdateWorkflow(id string, workflow *model.Workflow) error
	DeleteWorkflow(id string) error
	ActivateWorkflow(id string) error
	DeactivateWorkflow(id string) error

	// Execution operations
	SaveExecution(execution *model.WorkflowExecution) error
	GetExecution(id string) (*model.WorkflowExecution, error)
	ListExecutions(filters ExecutionFilters) ([]*model.WorkflowExecution, int, error)
	DeleteExecution(id string) error

	// Credential operations
	SaveCredential(credential *Credential) error
	GetCredential(id string) (*Credential, error)
	ListCredentials() ([]*Credential, error)
	UpdateCredential(id string, credential *Credential) error
	DeleteCredential(id string) error

	// Tag operations
	SaveTag(tag *Tag) error
	GetTag(id string) (*Tag, error)
	ListTags() ([]*Tag, error)
	UpdateTag(id string, tag *Tag) error
	DeleteTag(id string) error

	// Raw key-value operations (for webhooks and extensibility)
	SaveRaw(key string, value []byte) error
	GetRaw(key string) ([]byte, error)
	ListKeys(prefix string) ([]string, error)
	DeleteRaw(key string) error

	// Close the storage connection
	Close() error
}

// WorkflowFilters defines filters for listing workflows
type WorkflowFilters struct {
	Active *bool
	Search string
	Tags   []string
	Offset int
	Limit  int
}

// ExecutionFilters defines filters for listing executions
type ExecutionFilters struct {
	WorkflowID string
	Status     string
	Offset     int
	Limit      int
}

// Credential represents a stored credential
type Credential struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
}

// Tag represents a workflow tag
type Tag struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
