package database

import (
	"testing"
	
	"github.com/dipankar/m9m/internal/model"
	"github.com/dipankar/m9m/internal/nodes/base"
)

func TestSQLiteNodeCreation(t *testing.T) {
	node := NewSQLiteNode()
	if node == nil {
		t.Fatal("Expected node to be created, got nil")
	}
	
	desc := node.Description()
	if desc.Name != "SQLite" {
		t.Errorf("Expected name 'SQLite', got '%s'", desc.Name)
	}
}

func TestSQLiteNodeValidateParameters(t *testing.T) {
	node := NewSQLiteNode()
	
	// Test with nil params
	err := node.ValidateParameters(nil)
	if err == nil {
		t.Error("Expected error with nil params, got nil")
	}
	
	// Test with missing filename
	params := map[string]interface{}{}
	err = node.ValidateParameters(params)
	if err == nil {
		t.Error("Expected error with missing filename, got nil")
	}
	
	// Test with filename but missing operation
	filenameParams := map[string]interface{}{
		"filename": "test.db",
	}
	err = node.ValidateParameters(filenameParams)
	if err == nil {
		t.Error("Expected error with missing operation, got nil")
	}
	
	// Test with invalid operation
	invalidOperationParams := map[string]interface{}{
		"filename":  "test.db",
		"operation": "invalidOperation",
	}
	err = node.ValidateParameters(invalidOperationParams)
	if err == nil {
		t.Error("Expected error with invalid operation, got nil")
	}
	
	// Test with executeQuery operation but missing query
	missingQueryParams := map[string]interface{}{
		"filename":  "test.db",
		"operation": "executeQuery",
	}
	err = node.ValidateParameters(missingQueryParams)
	if err == nil {
		t.Error("Expected error with missing query for executeQuery operation, got nil")
	}
	
	// Test with valid executeQuery parameters
	validParams := map[string]interface{}{
		"filename":  "test.db",
		"operation": "executeQuery",
		"query":     "SELECT * FROM users",
	}
	err = node.ValidateParameters(validParams)
	if err != nil {
		t.Errorf("Expected no error with valid parameters, got %v", err)
	}
}

func TestSQLiteNodeExecuteWithEmptyInput(t *testing.T) {
	node := NewSQLiteNode()
	
	inputData := []model.DataItem{}
	nodeParams := map[string]interface{}{
		"filename":  ":memory:",
		"operation": "executeQuery",
		"query":     "SELECT * FROM users",
	}
	
	result, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d items", len(result))
	}
}

func TestSQLiteNodeImplementsNodeExecutor(t *testing.T) {
	var _ base.NodeExecutor = (*SQLiteNode)(nil)
}