package transform

import (
	"testing"
	
	"github.com/yourusername/n8n-go/internal/model"
	"github.com/yourusername/n8n-go/internal/nodes/base"
)

func TestSetNodeCreation(t *testing.T) {
	node := NewSetNode()
	if node == nil {
		t.Fatal("Expected node to be created, got nil")
	}
	
	desc := node.Description()
	if desc.Name != "Set" {
		t.Errorf("Expected name 'Set', got '%s'", desc.Name)
	}
}

func TestSetNodeValidateParameters(t *testing.T) {
	node := NewSetNode()
	
	// Test with nil params
	err := node.ValidateParameters(nil)
	if err == nil {
		t.Error("Expected error with nil params, got nil")
	}
	
	// Test with missing assignments
	params := map[string]interface{}{}
	err = node.ValidateParameters(params)
	if err == nil {
		t.Error("Expected error with missing assignments, got nil")
	}
	
	// Test with invalid assignments type
	invalidParams := map[string]interface{}{
		"assignments": "not an array",
	}
	err = node.ValidateParameters(invalidParams)
	if err == nil {
		t.Error("Expected error with invalid assignments type, got nil")
	}
	
	// Test with empty assignments array
	emptyParams := map[string]interface{}{
		"assignments": []interface{}{},
	}
	err = node.ValidateParameters(emptyParams)
	if err != nil {
		t.Errorf("Expected no error with empty assignments, got %v", err)
	}
	
	// Test with invalid assignment object
	invalidAssignmentParams := map[string]interface{}{
		"assignments": []interface{}{
			"not an object",
		},
	}
	err = node.ValidateParameters(invalidAssignmentParams)
	if err == nil {
		t.Error("Expected error with invalid assignment object, got nil")
	}
	
	// Test with assignment missing name
	missingNameParams := map[string]interface{}{
		"assignments": []interface{}{
			map[string]interface{}{
				"value": "test",
			},
		},
	}
	err = node.ValidateParameters(missingNameParams)
	if err == nil {
		t.Error("Expected error with missing name, got nil")
	}
	
	// Test with assignment missing value
	missingValueParams := map[string]interface{}{
		"assignments": []interface{}{
			map[string]interface{}{
				"name": "field1",
			},
		},
	}
	err = node.ValidateParameters(missingValueParams)
	if err == nil {
		t.Error("Expected error with missing value, got nil")
	}
	
	// Test with valid assignments
	validParams := map[string]interface{}{
		"assignments": []interface{}{
			map[string]interface{}{
				"name":  "field1",
				"value": "value1",
			},
			map[string]interface{}{
				"name":  "field2",
				"value": 42,
			},
		},
	}
	err = node.ValidateParameters(validParams)
	if err != nil {
		t.Errorf("Expected no error with valid params, got %v", err)
	}
}

func TestSetNodeExecuteWithEmptyInput(t *testing.T) {
	node := NewSetNode()
	
	inputData := []model.DataItem{}
	nodeParams := map[string]interface{}{
		"assignments": []interface{}{
			map[string]interface{}{
				"name":  "field1",
				"value": "value1",
			},
		},
	}
	
	result, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d items", len(result))
	}
}

func TestSetNodeExecuteWithValidAssignments(t *testing.T) {
	node := NewSetNode()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"existing": "value",
			},
		},
	}
	
	nodeParams := map[string]interface{}{
		"assignments": []interface{}{
			map[string]interface{}{
				"name":  "newField",
				"value": "newValue",
			},
			map[string]interface{}{
				"name":  "numberField",
				"value": 123,
			},
		},
	}
	
	result, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result) != 1 {
		t.Fatalf("Expected 1 result item, got %d", len(result))
	}
	
	item := result[0]
	
	// Check that existing field is preserved
	if existing, ok := item.JSON["existing"].(string); !ok || existing != "value" {
		t.Errorf("Expected existing field to be preserved, got %v", item.JSON["existing"])
	}
	
	// Check that new fields are added
	if newValue, ok := item.JSON["newField"].(string); !ok || newValue != "newValue" {
		t.Errorf("Expected newField to be 'newValue', got %v", item.JSON["newField"])
	}
	
	if numberValue, ok := item.JSON["numberField"].(int); !ok || numberValue != 123 {
		t.Errorf("Expected numberField to be 123, got %v", item.JSON["numberField"])
	}
}

func TestSetNodeExecuteWithMultipleItems(t *testing.T) {
	node := NewSetNode()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"id": 1,
			},
		},
		{
			JSON: map[string]interface{}{
				"id": 2,
			},
		},
	}
	
	nodeParams := map[string]interface{}{
		"assignments": []interface{}{
			map[string]interface{}{
				"name":  "processed",
				"value": true,
			},
		},
	}
	
	result, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result) != 2 {
		t.Fatalf("Expected 2 result items, got %d", len(result))
	}
	
	// Check first item
	if id, ok := result[0].JSON["id"].(int); !ok || id != 1 {
		t.Errorf("Expected first item id to be 1, got %v", result[0].JSON["id"])
	}
	
	if processed, ok := result[0].JSON["processed"].(bool); !ok || !processed {
		t.Errorf("Expected first item processed to be true, got %v", result[0].JSON["processed"])
	}
	
	// Check second item
	if id, ok := result[1].JSON["id"].(int); !ok || id != 2 {
		t.Errorf("Expected second item id to be 2, got %v", result[1].JSON["id"])
	}
	
	if processed, ok := result[1].JSON["processed"].(bool); !ok || !processed {
		t.Errorf("Expected second item processed to be true, got %v", result[1].JSON["processed"])
	}
}

func TestSetNodeImplementsNodeExecutor(t *testing.T) {
	var _ base.NodeExecutor = (*SetNode)(nil)
}