package database

import (
	"testing"
	
	"github.com/dipankar/n8n-go/internal/model"
	"github.com/dipankar/n8n-go/internal/nodes/base"
)

func TestMySQLNodeCreation(t *testing.T) {
	node := NewMySQLNode()
	if node == nil {
		t.Fatal("Expected node to be created, got nil")
	}
	
	desc := node.Description()
	if desc.Name != "MySQL" {
		t.Errorf("Expected name 'MySQL', got '%s'", desc.Name)
	}
}

func TestMySQLNodeValidateParameters(t *testing.T) {
	node := NewMySQLNode()
	
	// Test with nil params
	err := node.ValidateParameters(nil)
	if err == nil {
		t.Error("Expected error with nil params, got nil")
	}
	
	// Test with missing connection parameters
	params := map[string]interface{}{}
	err = node.ValidateParameters(params)
	if err == nil {
		t.Error("Expected error with missing connection parameters, got nil")
	}
	
	// Test with connection URL
	connectionParams := map[string]interface{}{
		"connectionUrl": "user:pass@tcp(localhost:3306)/db",
		"operation":     "executeQuery",
		"query":         "SELECT * FROM users",
	}
	err = node.ValidateParameters(connectionParams)
	if err != nil {
		t.Errorf("Expected no error with connection URL, got %v", err)
	}
	
	// Test with individual connection parameters
	individualParams := map[string]interface{}{
		"host":      "localhost",
		"database":  "testdb",
		"user":      "testuser",
		"operation": "executeQuery",
		"query":     "SELECT * FROM users",
	}
	err = node.ValidateParameters(individualParams)
	if err != nil {
		t.Errorf("Expected no error with individual parameters, got %v", err)
	}
	
	// Test with missing operation
	missingOperationParams := map[string]interface{}{
		"connectionUrl": "user:pass@tcp(localhost:3306)/db",
	}
	err = node.ValidateParameters(missingOperationParams)
	if err == nil {
		t.Error("Expected error with missing operation, got nil")
	}
	
	// Test with invalid operation
	invalidOperationParams := map[string]interface{}{
		"connectionUrl": "user:pass@tcp(localhost:3306)/db",
		"operation":     "invalidOperation",
	}
	err = node.ValidateParameters(invalidOperationParams)
	if err == nil {
		t.Error("Expected error with invalid operation, got nil")
	}
	
	// Test with executeQuery operation but missing query
	missingQueryParams := map[string]interface{}{
		"connectionUrl": "user:pass@tcp(localhost:3306)/db",
		"operation":     "executeQuery",
	}
	err = node.ValidateParameters(missingQueryParams)
	if err == nil {
		t.Error("Expected error with missing query for executeQuery operation, got nil")
	}
	
	// Test with valid executeQuery parameters
	validParams := map[string]interface{}{
		"connectionUrl": "user:pass@tcp(localhost:3306)/db",
		"operation":     "executeQuery",
		"query":         "SELECT * FROM users",
	}
	err = node.ValidateParameters(validParams)
	if err != nil {
		t.Errorf("Expected no error with valid parameters, got %v", err)
	}
}

func TestMySQLNodeExecuteWithEmptyInput(t *testing.T) {
	node := NewMySQLNode()
	
	inputData := []model.DataItem{}
	nodeParams := map[string]interface{}{
		"connectionUrl": "user:pass@tcp(localhost:3306)/db",
		"operation":     "executeQuery",
		"query":         "SELECT * FROM users",
	}
	
	result, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d items", len(result))
	}
}

func TestMySQLNodeImplementsNodeExecutor(t *testing.T) {
	var _ base.NodeExecutor = (*MySQLNode)(nil)
}