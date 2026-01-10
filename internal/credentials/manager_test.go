package credentials

import (
	"testing"

	"github.com/dipankar/m9m/internal/model"
)

func TestCredentialManagerCreation(t *testing.T) {
	manager, err := NewCredentialManager()
	if err != nil {
		t.Fatalf("Failed to create credential manager: %v", err)
	}

	if manager == nil {
		t.Fatal("Expected credential manager to be created, got nil")
	}
}

func TestNodeCredentialRegistration(t *testing.T) {
	manager, err := NewCredentialManager()
	if err != nil {
		t.Fatalf("Failed to create credential manager: %v", err)
	}

	// First store a credential
	credData := map[string]interface{}{
		"id":   "cred-123",
		"name": "Test Credential",
		"type": "httpBasicAuth",
		"data": map[string]interface{}{
			"username": "testuser",
			"password": "testpass",
		},
	}

	err = manager.StoreCredentialFromWorkflow(credData)
	if err != nil {
		t.Fatalf("Failed to store credential: %v", err)
	}

	// Now register credentials for a node
	manager.RegisterNodeCredentials("node-1", "httpBasicAuth", "cred-123")

	// Try to get credentials for the node
	creds, err := manager.GetNodeCredentials("node-1")
	if err != nil {
		t.Fatalf("Failed to get node credentials: %v", err)
	}

	// Check that we got a credentials map
	if creds == nil {
		t.Error("Expected credentials map, got nil")
	}
}

func TestCredentialStorageAndRetrievalManager(t *testing.T) {
	manager, err := NewCredentialManager()
	if err != nil {
		t.Fatalf("Failed to create credential manager: %v", err)
	}

	// Create test credential data
	credData := map[string]interface{}{
		"id":   "test-cred-1",
		"name": "Test API Key",
		"type": "apiKey",
		"data": map[string]interface{}{
			"apiKey": "test-api-key-123",
			"url":    "https://api.example.com",
		},
	}

	// Store the credential
	err = manager.StoreCredentialFromWorkflow(credData)
	if err != nil {
		t.Fatalf("Failed to store credential: %v", err)
	}

	// Try to register it for a node
	manager.RegisterNodeCredentials("test-node-1", "apiAuth", "test-cred-1")

	// Try to get credentials for the node
	creds, err := manager.GetNodeCredentials("test-node-1")
	if err != nil {
		t.Fatalf("Failed to get node credentials: %v", err)
	}

	// Check that we got a credentials map
	if creds == nil {
		t.Error("Expected credentials map, got nil")
	}
}

func TestWorkflowCredentialResolution(t *testing.T) {
	manager, err := NewCredentialManager()
	if err != nil {
		t.Fatalf("Failed to create credential manager: %v", err)
	}

	// First store the credential that will be referenced
	credData := map[string]interface{}{
		"id":   "cred-123",
		"name": "HTTP Basic Auth",
		"type": "httpBasicAuth",
		"data": map[string]interface{}{
			"username": "testuser",
			"password": "testpass",
		},
	}

	err = manager.StoreCredentialFromWorkflow(credData)
	if err != nil {
		t.Fatalf("Failed to store credential: %v", err)
	}

	// Create a test workflow with credentials
	workflow := &model.Workflow{
		ID:   "test-workflow-1",
		Name: "Test Workflow",
		Nodes: []model.Node{
			{
				ID:   "node-1",
				Name: "HTTP Request Node",
				Type: "n8n-nodes-base.httpRequest",
				Credentials: map[string]model.Credential{
					"httpBasicAuth": {
						ID:   "cred-123",
						Name: "HTTP Basic Auth",
						Type: "httpBasicAuth",
					},
				},
			},
		},
		Connections: make(map[string]model.Connections),
	}

	// Resolve workflow credentials
	err = manager.ResolveWorkflowCredentials(workflow)
	if err != nil {
		t.Fatalf("Failed to resolve workflow credentials: %v", err)
	}

	// Try to get credentials for the node
	creds, err := manager.GetNodeCredentials("node-1")
	if err != nil {
		t.Fatalf("Failed to get node credentials: %v", err)
	}

	// Check that we got a credentials map
	if creds == nil {
		t.Error("Expected credentials map, got nil")
	}
}

func TestParameterInjection(t *testing.T) {
	manager, err := NewCredentialManager()
	if err != nil {
		t.Fatalf("Failed to create credential manager: %v", err)
	}

	// Store a credential first
	credData := map[string]interface{}{
		"id":   "inject-test-cred",
		"name": "Inject Test Credential",
		"type": "apiKey",
		"data": map[string]interface{}{
			"apiKey": "inject-api-key",
		},
	}

	err = manager.StoreCredentialFromWorkflow(credData)
	if err != nil {
		t.Fatalf("Failed to store credential: %v", err)
	}

	// Register the credential for a node
	manager.RegisterNodeCredentials("inject-test-node", "auth", "inject-test-cred")

	// Create test node parameters
	nodeParams := map[string]interface{}{
		"url":    "https://api.example.com",
		"method": "GET",
	}

	// Inject credentials into parameters
	injectedParams, err := manager.InjectCredentialsIntoNodeParameters("inject-test-node", nodeParams)
	if err != nil {
		t.Fatalf("Failed to inject credentials: %v", err)
	}

	// Check that original parameters are preserved
	if injectedParams["url"] != "https://api.example.com" {
		t.Errorf("Expected url 'https://api.example.com', got '%v'", injectedParams["url"])
	}

	if injectedParams["method"] != "GET" {
		t.Errorf("Expected method 'GET', got '%v'", injectedParams["method"])
	}
}

func TestCredentialManagerIntegration(t *testing.T) {
	manager, err := NewCredentialManager()
	if err != nil {
		t.Fatalf("Failed to create credential manager: %v", err)
	}

	// Create and store a credential
	credData := map[string]interface{}{
		"id":   "integration-test-cred",
		"name": "Integration Test Credential",
		"type": "apiKey",
		"data": map[string]interface{}{
			"apiKey": "integration-api-key-456",
			"secret": "integration-secret-789",
		},
	}

	err = manager.StoreCredentialFromWorkflow(credData)
	if err != nil {
		t.Fatalf("Failed to store credential: %v", err)
	}

	// Register the credential for a node
	manager.RegisterNodeCredentials("integration-test-node", "apiAuth", "integration-test-cred")

	// Get credentials for the node
	creds, err := manager.GetNodeCredentials("integration-test-node")
	if err != nil {
		t.Fatalf("Failed to get node credentials: %v", err)
	}

	// Check that we got a credentials map
	if creds == nil {
		t.Error("Expected credentials map, got nil")
	}

	// Test parameter injection
	nodeParams := map[string]interface{}{
		"endpoint": "/test",
		"timeout":  30,
	}

	injectedParams, err := manager.InjectCredentialsIntoNodeParameters("integration-test-node", nodeParams)
	if err != nil {
		t.Fatalf("Failed to inject credentials: %v", err)
	}

	// Check that original parameters are preserved
	if injectedParams["endpoint"] != "/test" {
		t.Errorf("Expected endpoint '/test', got '%v'", injectedParams["endpoint"])
	}

	if injectedParams["timeout"] != 30 {
		t.Errorf("Expected timeout 30, got '%v'", injectedParams["timeout"])
	}
}
