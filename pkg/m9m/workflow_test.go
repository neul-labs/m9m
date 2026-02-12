package m9m

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseWorkflow(t *testing.T) {
	workflowJSON := `{
		"id": "test-123",
		"name": "Test Workflow",
		"active": true,
		"nodes": [
			{
				"name": "Start",
				"type": "n8n-nodes-base.start",
				"typeVersion": 1,
				"position": [0, 0],
				"parameters": {}
			}
		],
		"connections": {}
	}`

	workflow, err := ParseWorkflow([]byte(workflowJSON))
	if err != nil {
		t.Fatalf("ParseWorkflow failed: %v", err)
	}

	if workflow.ID != "test-123" {
		t.Errorf("Expected ID 'test-123', got '%s'", workflow.ID)
	}
	if workflow.Name != "Test Workflow" {
		t.Errorf("Expected name 'Test Workflow', got '%s'", workflow.Name)
	}
	if !workflow.Active {
		t.Error("Expected Active to be true")
	}
	if len(workflow.Nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(workflow.Nodes))
	}
}

func TestParseWorkflow_Invalid(t *testing.T) {
	_, err := ParseWorkflow([]byte("invalid json"))
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestWorkflow_ToJSON(t *testing.T) {
	workflow := &Workflow{
		ID:     "test-123",
		Name:   "Test",
		Active: true,
		Nodes:  []Node{},
	}

	data, err := workflow.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if parsed["id"] != "test-123" {
		t.Errorf("Expected id 'test-123', got '%v'", parsed["id"])
	}
}

func TestLoadWorkflow(t *testing.T) {
	// Create a temporary workflow file
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test-workflow.json")

	workflowJSON := `{
		"id": "file-test",
		"name": "File Test Workflow",
		"active": false,
		"nodes": [],
		"connections": {}
	}`

	if err := os.WriteFile(filePath, []byte(workflowJSON), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	workflow, err := LoadWorkflow(filePath)
	if err != nil {
		t.Fatalf("LoadWorkflow failed: %v", err)
	}

	if workflow.ID != "file-test" {
		t.Errorf("Expected ID 'file-test', got '%s'", workflow.ID)
	}
}

func TestLoadWorkflow_NotFound(t *testing.T) {
	_, err := LoadWorkflow("/nonexistent/path/workflow.json")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestWorkflow_ToFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "output-workflow.json")

	workflow := &Workflow{
		ID:     "output-test",
		Name:   "Output Test",
		Active: true,
		Nodes:  []Node{},
	}

	if err := workflow.ToFile(filePath); err != nil {
		t.Fatalf("ToFile failed: %v", err)
	}

	// Read back and verify
	loaded, err := LoadWorkflow(filePath)
	if err != nil {
		t.Fatalf("Failed to load written file: %v", err)
	}

	if loaded.ID != "output-test" {
		t.Errorf("Expected ID 'output-test', got '%s'", loaded.ID)
	}
}

func TestNewDataItem(t *testing.T) {
	item := NewDataItem(map[string]interface{}{"key": "value"})
	if item.JSON["key"] != "value" {
		t.Errorf("Expected key='value', got '%v'", item.JSON["key"])
	}
}

func TestNewDataItem_Nil(t *testing.T) {
	item := NewDataItem(nil)
	if item.JSON == nil {
		t.Error("Expected non-nil JSON map")
	}
}

func TestNewDataItems(t *testing.T) {
	data := []map[string]interface{}{
		{"id": 1},
		{"id": 2},
	}

	items := NewDataItems(data)
	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}
	if items[0].JSON["id"] != 1 {
		t.Errorf("Expected first item id=1, got %v", items[0].JSON["id"])
	}
}

func TestWorkflow_ToInternal(t *testing.T) {
	workflow := &Workflow{
		ID:     "test",
		Name:   "Test",
		Active: true,
		Nodes: []Node{
			{
				ID:          "node1",
				Name:        "Node 1",
				Type:        "test.node",
				TypeVersion: 1,
				Position:    []int{0, 0},
				Parameters:  map[string]interface{}{"param": "value"},
			},
		},
		Connections: map[string]Connections{
			"Node 1": {
				Main: [][]Connection{
					{
						{Node: "Node 2", Type: "main", Index: 0},
					},
				},
			},
		},
		Settings: &WorkflowSettings{
			Timezone: "UTC",
		},
	}

	internal := workflow.toInternal()
	if internal == nil {
		t.Fatal("toInternal returned nil")
	}
	if internal.ID != "test" {
		t.Errorf("Expected ID 'test', got '%s'", internal.ID)
	}
	if len(internal.Nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(internal.Nodes))
	}
	if internal.Settings == nil || internal.Settings.Timezone != "UTC" {
		t.Error("Settings not converted correctly")
	}
}

func TestDataItemConversion(t *testing.T) {
	original := []DataItem{
		{
			JSON: map[string]interface{}{"key": "value"},
			Binary: map[string]BinaryData{
				"file": {
					Data:     "base64data",
					MimeType: "text/plain",
					FileName: "test.txt",
				},
			},
		},
	}

	// Convert to internal and back
	internal := dataItemsToInternal(original)
	back := dataItemsFromInternal(internal)

	if len(back) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(back))
	}
	if back[0].JSON["key"] != "value" {
		t.Errorf("JSON data not preserved")
	}
	if back[0].Binary["file"].FileName != "test.txt" {
		t.Errorf("Binary data not preserved")
	}
}
