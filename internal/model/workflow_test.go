package model

import (
	"encoding/json"
	"os"
	"testing"
)

// TestWorkflowJSONSerialization tests that our workflow model can be serialized and deserialized
func TestWorkflowJSONSerialization(t *testing.T) {
	// Create a sample workflow that matches n8n's structure
	workflow := &Workflow{
		ID:     "1",
		Name:   "Test Workflow",
		Active: true,
		Nodes: []Node{
			{
				ID:          "2",
				Name:        "HTTP Request",
				Type:        "n8n-nodes-base.httpRequest",
				TypeVersion: 1,
				Position:    []int{250, 300},
				Parameters: map[string]interface{}{
					"url":    "https://api.example.com",
					"method": "GET",
				},
			},
		},
		Connections: map[string]Connections{
			"HTTP Request": {
				Main: [][]Connection{
					{
						{
							Node:  "Debug",
							Type:  "main",
							Index: 0,
						},
					},
				},
			},
		},
		Settings: &WorkflowSettings{
			ExecutionOrder: "v1",
			Timezone:       "America/New_York",
		},
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(workflow)
	if err != nil {
		t.Fatalf("Failed to marshal workflow: %v", err)
	}

	// Deserialize back from JSON
	var deserializedWorkflow Workflow
	err = json.Unmarshal(jsonData, &deserializedWorkflow)
	if err != nil {
		t.Fatalf("Failed to unmarshal workflow: %v", err)
	}

	// Verify the deserialized data matches the original
	if deserializedWorkflow.ID != workflow.ID {
		t.Errorf("Expected ID %s, got %s", workflow.ID, deserializedWorkflow.ID)
	}
	if deserializedWorkflow.Name != workflow.Name {
		t.Errorf("Expected Name %s, got %s", workflow.Name, deserializedWorkflow.Name)
	}
	if deserializedWorkflow.Active != workflow.Active {
		t.Errorf("Expected Active %v, got %v", workflow.Active, deserializedWorkflow.Active)
	}
	if len(deserializedWorkflow.Nodes) != len(workflow.Nodes) {
		t.Errorf("Expected %d nodes, got %d", len(workflow.Nodes), len(deserializedWorkflow.Nodes))
	}
}

// TestFromFileAndToFile tests file I/O functionality
func TestFromFileAndToFile(t *testing.T) {
	// Create a sample workflow
	workflow := &Workflow{
		ID:     "test-1",
		Name:   "Test File Workflow",
		Active: true,
		Nodes: []Node{
			{
				Name:        "Start",
				Type:        "n8n-nodes-base.manualTrigger",
				TypeVersion: 1,
				Position:    []int{250, 300},
			},
		},
		Connections: make(map[string]Connections),
	}

	// Write to file
	filename := "/tmp/test-workflow.json"
	err := workflow.ToFile(filename)
	if err != nil {
		t.Fatalf("Failed to write workflow to file: %v", err)
	}
	defer os.Remove(filename) // Clean up

	// Read from file
	readWorkflow, err := FromFile(filename)
	if err != nil {
		t.Fatalf("Failed to read workflow from file: %v", err)
	}

	// Verify data
	if readWorkflow.ID != workflow.ID {
		t.Errorf("Expected ID %s, got %s", workflow.ID, readWorkflow.ID)
	}
	if readWorkflow.Name != workflow.Name {
		t.Errorf("Expected Name %s, got %s", workflow.Name, readWorkflow.Name)
	}
}