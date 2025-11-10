/*
Package model provides data structures that match n8n's exact JSON structure
for workflows, nodes, connections, and data items.
*/
package model

import (
	"encoding/json"
	"io"
	"os"
	"time"
)

// Workflow represents an n8n workflow structure
type Workflow struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Active      bool                   `json:"active"`
	Nodes       []Node                 `json:"nodes"`
	Connections map[string]Connections `json:"connections"`
	Settings    *WorkflowSettings      `json:"settings,omitempty"`
	StaticData  map[string]interface{} `json:"staticData,omitempty"`
	PinData     map[string][]DataItem  `json:"pinData,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	VersionID   string                 `json:"versionId,omitempty"`
	IsArchived  bool                   `json:"isArchived,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	CreatedBy   string                 `json:"createdBy,omitempty"`
}

// Node represents an individual node in a workflow
type Node struct {
	ID          string                 `json:"id,omitempty"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	TypeVersion int                    `json:"typeVersion"`
	Position    []int                  `json:"position"`
	Parameters  map[string]interface{} `json:"parameters"`
	Credentials map[string]Credential  `json:"credentials,omitempty"`
	WebhookID   string                 `json:"webhookId,omitempty"`
	Notes       string                 `json:"notes,omitempty"`
	Disabled    *bool                  `json:"disabled,omitempty"`
}

// Credential represents a credential used by a node
type Credential struct {
	ID   string                 `json:"id,omitempty"`
	Name string                 `json:"name"`
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data,omitempty"`
}

// WorkflowSettings represents workflow-level settings
type WorkflowSettings struct {
	ExecutionOrder       string      `json:"executionOrder,omitempty"`
	Timezone             string      `json:"timezone,omitempty"`
	SaveDataError        interface{} `json:"saveDataError,omitempty"`
	SaveDataSuccess      interface{} `json:"saveDataSuccess,omitempty"`
	SaveManualExecutions interface{} `json:"saveManualExecutions,omitempty"`
}

// Connections represents all connections from a node
type Connections struct {
	Main [][]Connection `json:"main,omitempty"`
}

// Connection represents a single connection between nodes
type Connection struct {
	Node  string `json:"node"`
	Type  string `json:"type"`
	Index int    `json:"index"`
}

// DataItem represents a data item flowing through the workflow
type DataItem struct {
	JSON       map[string]interface{} `json:"json"`
	Binary     map[string]BinaryData  `json:"binary,omitempty"`
	PairedItem interface{}            `json:"pairedItem,omitempty"`
	Error      interface{}            `json:"error,omitempty"`
}

// BinaryData represents binary data in a workflow
type BinaryData struct {
	Data         string `json:"data"`
	MimeType     string `json:"mimeType"`
	FileSize     string `json:"fileSize,omitempty"`
	FileName     string `json:"fileName,omitempty"`
	Directory    string `json:"directory,omitempty"`
	FileExtension string `json:"fileExtension,omitempty"`
}

// FromJSON parses a workflow from JSON data
func FromJSON(data []byte) (*Workflow, error) {
	var workflow Workflow
	err := json.Unmarshal(data, &workflow)
	if err != nil {
		return nil, err
	}
	return &workflow, nil
}

// FromReader parses a workflow from an io.Reader
func FromReader(reader io.Reader) (*Workflow, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return FromJSON(data)
}

// FromFile parses a workflow from a JSON file
func FromFile(filename string) (*Workflow, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return FromReader(file)
}

// ToJSON serializes a workflow to JSON data
func (w *Workflow) ToJSON() ([]byte, error) {
	return json.Marshal(w)
}

// ToFile writes a workflow to a JSON file
func (w *Workflow) ToFile(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	data, err := w.ToJSON()
	if err != nil {
		return err
	}
	
	_, err = file.Write(data)
	return err
}
// Additional type definitions for compatibility

// NodeExecutionInput represents input to a node execution
type NodeExecutionInput struct {
	Data []DataItem
	RunIndex int
	ItemIndex int
}

// NodeExecutionOutput represents output from a node execution
type NodeExecutionOutput struct {
	Data []DataItem
	Error error
}

// NodeDefinition describes a node type's definition
type NodeDefinition struct {
	Name        string
	DisplayName string
	Description string
	Version     int
	Inputs      []string
	Outputs     []string
	Properties  []interface{}
}

// WorkflowExecution represents the execution of a workflow
type WorkflowExecution struct {
	ID          string                 `json:"id"`
	WorkflowID  string                 `json:"workflowId"`
	Status      string                 `json:"status"` // running, completed, failed, cancelled
	Mode        string                 `json:"mode"`   // manual, trigger, test
	StartedAt   time.Time              `json:"startedAt"`
	FinishedAt  *time.Time             `json:"finishedAt,omitempty"`
	Data        []DataItem             `json:"data,omitempty"`
	Error       error                  `json:"error,omitempty"`
	NodeData    map[string][]DataItem  `json:"nodeData,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// NodeConnections represents connections configuration for a node
type NodeConnections struct {
	Main [][]Connection `json:"main,omitempty"`
}
