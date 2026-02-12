package m9m

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"time"

	"github.com/neul-labs/m9m/internal/model"
)

// Common errors
var (
	ErrNilWorkflow = errors.New("workflow cannot be nil")
)

// Workflow represents a workflow automation definition.
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

// Node represents an individual node in a workflow.
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

// Credential represents a credential reference used by a node.
type Credential struct {
	ID   string                 `json:"id,omitempty"`
	Name string                 `json:"name"`
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data,omitempty"`
}

// WorkflowSettings represents workflow-level settings.
type WorkflowSettings struct {
	ExecutionOrder       string      `json:"executionOrder,omitempty"`
	Timezone             string      `json:"timezone,omitempty"`
	SaveDataError        interface{} `json:"saveDataError,omitempty"`
	SaveDataSuccess      interface{} `json:"saveDataSuccess,omitempty"`
	SaveManualExecutions interface{} `json:"saveManualExecutions,omitempty"`
}

// Connections represents all connections from a node.
type Connections struct {
	Main [][]Connection `json:"main,omitempty"`
}

// Connection represents a single connection between nodes.
type Connection struct {
	Node  string `json:"node"`
	Type  string `json:"type"`
	Index int    `json:"index"`
}

// DataItem represents a data item flowing through the workflow.
type DataItem struct {
	JSON       map[string]interface{} `json:"json"`
	Binary     map[string]BinaryData  `json:"binary,omitempty"`
	PairedItem interface{}            `json:"pairedItem,omitempty"`
	Error      interface{}            `json:"error,omitempty"`
}

// BinaryData represents binary data in a workflow.
type BinaryData struct {
	Data          string `json:"data"`
	MimeType      string `json:"mimeType"`
	FileSize      string `json:"fileSize,omitempty"`
	FileName      string `json:"fileName,omitempty"`
	Directory     string `json:"directory,omitempty"`
	FileExtension string `json:"fileExtension,omitempty"`
}

// ExecutionResult represents the result of a workflow execution.
type ExecutionResult struct {
	Data  []DataItem `json:"data"`
	Error error      `json:"error,omitempty"`
}

// LoadWorkflow loads a workflow from a JSON file.
func LoadWorkflow(path string) (*Workflow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return ParseWorkflowFromReader(file)
}

// ParseWorkflow parses a workflow from JSON bytes.
func ParseWorkflow(data []byte) (*Workflow, error) {
	var workflow Workflow
	if err := json.Unmarshal(data, &workflow); err != nil {
		return nil, err
	}
	return &workflow, nil
}

// ParseWorkflowFromReader parses a workflow from an io.Reader.
func ParseWorkflowFromReader(r io.Reader) (*Workflow, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return ParseWorkflow(data)
}

// ToJSON serializes the workflow to JSON bytes.
func (w *Workflow) ToJSON() ([]byte, error) {
	return json.Marshal(w)
}

// ToFile saves the workflow to a JSON file.
func (w *Workflow) ToFile(path string) error {
	data, err := w.ToJSON()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// NewDataItem creates a new DataItem with JSON data.
func NewDataItem(data map[string]interface{}) DataItem {
	if data == nil {
		data = make(map[string]interface{})
	}
	return DataItem{JSON: data}
}

// NewDataItems creates a slice of DataItems from a slice of maps.
func NewDataItems(data []map[string]interface{}) []DataItem {
	items := make([]DataItem, len(data))
	for i, d := range data {
		items[i] = NewDataItem(d)
	}
	return items
}

// toInternal converts a public Workflow to an internal model.Workflow.
func (w *Workflow) toInternal() *model.Workflow {
	if w == nil {
		return nil
	}

	internal := &model.Workflow{
		ID:          w.ID,
		Name:        w.Name,
		Description: w.Description,
		Active:      w.Active,
		StaticData:  w.StaticData,
		Tags:        w.Tags,
		VersionID:   w.VersionID,
		IsArchived:  w.IsArchived,
		CreatedAt:   w.CreatedAt,
		UpdatedAt:   w.UpdatedAt,
		CreatedBy:   w.CreatedBy,
	}

	// Convert nodes
	internal.Nodes = make([]model.Node, len(w.Nodes))
	for i, n := range w.Nodes {
		internal.Nodes[i] = nodeToInternal(n)
	}

	// Convert connections
	internal.Connections = make(map[string]model.Connections)
	for k, v := range w.Connections {
		internal.Connections[k] = connectionsToInternal(v)
	}

	// Convert settings
	if w.Settings != nil {
		internal.Settings = &model.WorkflowSettings{
			ExecutionOrder:       w.Settings.ExecutionOrder,
			Timezone:             w.Settings.Timezone,
			SaveDataError:        w.Settings.SaveDataError,
			SaveDataSuccess:      w.Settings.SaveDataSuccess,
			SaveManualExecutions: w.Settings.SaveManualExecutions,
		}
	}

	// Convert pin data
	if w.PinData != nil {
		internal.PinData = make(map[string][]model.DataItem)
		for k, v := range w.PinData {
			internal.PinData[k] = dataItemsToInternal(v)
		}
	}

	return internal
}

// workflowFromInternal converts an internal model.Workflow to a public Workflow.
func workflowFromInternal(w *model.Workflow) *Workflow {
	if w == nil {
		return nil
	}

	public := &Workflow{
		ID:          w.ID,
		Name:        w.Name,
		Description: w.Description,
		Active:      w.Active,
		StaticData:  w.StaticData,
		Tags:        w.Tags,
		VersionID:   w.VersionID,
		IsArchived:  w.IsArchived,
		CreatedAt:   w.CreatedAt,
		UpdatedAt:   w.UpdatedAt,
		CreatedBy:   w.CreatedBy,
	}

	// Convert nodes
	public.Nodes = make([]Node, len(w.Nodes))
	for i, n := range w.Nodes {
		public.Nodes[i] = nodeFromInternal(n)
	}

	// Convert connections
	public.Connections = make(map[string]Connections)
	for k, v := range w.Connections {
		public.Connections[k] = connectionsFromInternal(v)
	}

	// Convert settings
	if w.Settings != nil {
		public.Settings = &WorkflowSettings{
			ExecutionOrder:       w.Settings.ExecutionOrder,
			Timezone:             w.Settings.Timezone,
			SaveDataError:        w.Settings.SaveDataError,
			SaveDataSuccess:      w.Settings.SaveDataSuccess,
			SaveManualExecutions: w.Settings.SaveManualExecutions,
		}
	}

	// Convert pin data
	if w.PinData != nil {
		public.PinData = make(map[string][]DataItem)
		for k, v := range w.PinData {
			public.PinData[k] = dataItemsFromInternal(v)
		}
	}

	return public
}

func nodeToInternal(n Node) model.Node {
	internal := model.Node{
		ID:          n.ID,
		Name:        n.Name,
		Type:        n.Type,
		TypeVersion: n.TypeVersion,
		Position:    n.Position,
		Parameters:  n.Parameters,
		WebhookID:   n.WebhookID,
		Notes:       n.Notes,
		Disabled:    n.Disabled,
	}

	// Convert credentials
	if n.Credentials != nil {
		internal.Credentials = make(map[string]model.Credential)
		for k, v := range n.Credentials {
			internal.Credentials[k] = model.Credential{
				ID:   v.ID,
				Name: v.Name,
				Type: v.Type,
				Data: v.Data,
			}
		}
	}

	return internal
}

func nodeFromInternal(n model.Node) Node {
	public := Node{
		ID:          n.ID,
		Name:        n.Name,
		Type:        n.Type,
		TypeVersion: n.TypeVersion,
		Position:    n.Position,
		Parameters:  n.Parameters,
		WebhookID:   n.WebhookID,
		Notes:       n.Notes,
		Disabled:    n.Disabled,
	}

	// Convert credentials
	if n.Credentials != nil {
		public.Credentials = make(map[string]Credential)
		for k, v := range n.Credentials {
			public.Credentials[k] = Credential{
				ID:   v.ID,
				Name: v.Name,
				Type: v.Type,
				Data: v.Data,
			}
		}
	}

	return public
}

func connectionsToInternal(c Connections) model.Connections {
	internal := model.Connections{}
	if c.Main != nil {
		internal.Main = make([][]model.Connection, len(c.Main))
		for i, conns := range c.Main {
			internal.Main[i] = make([]model.Connection, len(conns))
			for j, conn := range conns {
				internal.Main[i][j] = model.Connection{
					Node:  conn.Node,
					Type:  conn.Type,
					Index: conn.Index,
				}
			}
		}
	}
	return internal
}

func connectionsFromInternal(c model.Connections) Connections {
	public := Connections{}
	if c.Main != nil {
		public.Main = make([][]Connection, len(c.Main))
		for i, conns := range c.Main {
			public.Main[i] = make([]Connection, len(conns))
			for j, conn := range conns {
				public.Main[i][j] = Connection{
					Node:  conn.Node,
					Type:  conn.Type,
					Index: conn.Index,
				}
			}
		}
	}
	return public
}

func dataItemsToInternal(items []DataItem) []model.DataItem {
	if items == nil {
		return nil
	}
	internal := make([]model.DataItem, len(items))
	for i, item := range items {
		internal[i] = dataItemToInternal(item)
	}
	return internal
}

func dataItemsFromInternal(items []model.DataItem) []DataItem {
	if items == nil {
		return nil
	}
	public := make([]DataItem, len(items))
	for i, item := range items {
		public[i] = dataItemFromInternal(item)
	}
	return public
}

func dataItemToInternal(item DataItem) model.DataItem {
	internal := model.DataItem{
		JSON:       item.JSON,
		PairedItem: item.PairedItem,
		Error:      item.Error,
	}

	if item.Binary != nil {
		internal.Binary = make(map[string]model.BinaryData)
		for k, v := range item.Binary {
			internal.Binary[k] = model.BinaryData{
				Data:          v.Data,
				MimeType:      v.MimeType,
				FileSize:      v.FileSize,
				FileName:      v.FileName,
				Directory:     v.Directory,
				FileExtension: v.FileExtension,
			}
		}
	}

	return internal
}

func dataItemFromInternal(item model.DataItem) DataItem {
	public := DataItem{
		JSON:       item.JSON,
		PairedItem: item.PairedItem,
		Error:      item.Error,
	}

	if item.Binary != nil {
		public.Binary = make(map[string]BinaryData)
		for k, v := range item.Binary {
			public.Binary[k] = BinaryData{
				Data:          v.Data,
				MimeType:      v.MimeType,
				FileSize:      v.FileSize,
				FileName:      v.FileName,
				Directory:     v.Directory,
				FileExtension: v.FileExtension,
			}
		}
	}

	return public
}
