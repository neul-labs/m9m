package engine

import (
	"github.com/dipankar/m9m/internal/credentials"
	"github.com/dipankar/m9m/internal/model"
	"github.com/dipankar/m9m/internal/nodes/base"
	"testing"
)

// mockNodeExecutor is a mock implementation of NodeExecutor for testing
type mockNodeExecutor struct {
	name        string
	description base.NodeDescription
}

func (m *mockNodeExecutor) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	// Return the input data as output for testing
	return inputData, nil
}

func (m *mockNodeExecutor) Description() base.NodeDescription {
	return m.description
}

func (m *mockNodeExecutor) ValidateParameters(params map[string]interface{}) error {
	// Always valid for testing
	return nil
}

func TestWorkflowEngineCreation(t *testing.T) {
	engine := NewWorkflowEngine()
	if engine == nil {
		t.Fatal("Expected engine to be created, got nil")
	}
}

func TestExecuteWorkflowWithNilWorkflow(t *testing.T) {
	engine := NewWorkflowEngine()

	_, err := engine.ExecuteWorkflow(nil, nil)
	if err == nil {
		t.Error("Expected error when executing nil workflow, got nil")
	}
}

func TestNodeRegistrationAndRetrieval(t *testing.T) {
	engine := NewWorkflowEngine().(*workflowEngineImpl)

	// Create a mock executor
	executor := &mockNodeExecutor{
		name: "test-executor",
		description: base.NodeDescription{
			Name:        "Test Executor",
			Description: "A test node executor",
			Category:    "Test",
		},
	}

	// Register the executor
	engine.RegisterNodeExecutor("test-node-type", executor)

	// Retrieve the executor
	retrieved, err := engine.GetNodeExecutor("test-node-type")
	if err != nil {
		t.Fatalf("Failed to retrieve executor: %v", err)
	}

	// We can't directly compare interfaces, so we'll just check that it's not nil
	if retrieved == nil {
		t.Error("Retrieved executor is nil")
	}

	// Try to retrieve a non-existent executor
	_, err = engine.GetNodeExecutor("non-existent-type")
	if err == nil {
		t.Error("Expected error when retrieving non-existent executor, got nil")
	}
}

func TestExecuteWorkflowWithRegisteredNode(t *testing.T) {
	engine := NewWorkflowEngine().(*workflowEngineImpl)

	// Create and register a mock executor
	executor := &mockNodeExecutor{
		name: "http-request-executor",
		description: base.NodeDescription{
			Name:        "HTTP Request",
			Description: "Makes HTTP requests",
			Category:    "HTTP",
		},
	}

	engine.RegisterNodeExecutor("n8n-nodes-base.httpRequest", executor)

	// Create a workflow with an HTTP Request node
	workflow := &model.Workflow{
		ID:     "test-1",
		Name:   "Test Workflow",
		Active: true,
		Nodes: []model.Node{
			{
				ID:          "node-1",
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
		Connections: make(map[string]model.Connections),
	}

	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"test": "data",
			},
		},
	}

	result, err := engine.ExecuteWorkflow(workflow, inputData)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if len(result.Data) != len(inputData) {
		t.Errorf("Expected %d data items, got %d", len(inputData), len(result.Data))
	}
}

func TestExecuteWorkflowWithoutNodes(t *testing.T) {
	engine := NewWorkflowEngine()

	workflow := &model.Workflow{
		ID:          "test-1",
		Name:        "Test Workflow",
		Active:      true,
		Nodes:       []model.Node{}, // No nodes
		Connections: make(map[string]model.Connections),
	}

	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"test": "data",
			},
		},
	}

	result, err := engine.ExecuteWorkflow(workflow, inputData)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if len(result.Data) != len(inputData) {
		t.Errorf("Expected %d data items, got %d", len(inputData), len(result.Data))
	}
}

func TestWorkflowEngineWithCredentialManager(t *testing.T) {
	engine := NewWorkflowEngine().(*workflowEngineImpl)

	// Test setting credential manager
	credManager, err := credentials.NewCredentialManager()
	if err != nil {
		t.Fatalf("Failed to create credential manager: %v", err)
	}

	engine.SetCredentialManager(credManager)

	// Verify credential manager is set
	if engine.credentialManager == nil {
		t.Error("Expected credential manager to be set")
	}
}
