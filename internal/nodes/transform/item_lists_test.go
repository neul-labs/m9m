package transform

import (
	"testing"
	
	"github.com/dipankar/n8n-go/internal/model"
	"github.com/dipankar/n8n-go/internal/nodes/base"
)

func TestItemListsNodeCreation(t *testing.T) {
	node := NewItemListsNode()
	if node == nil {
		t.Fatal("Expected node to be created, got nil")
	}
	
	desc := node.Description()
	if desc.Name != "Item Lists" {
		t.Errorf("Expected name 'Item Lists', got '%s'", desc.Name)
	}
}

func TestItemListsNodeValidateParameters(t *testing.T) {
	node := NewItemListsNode()
	
	// Test with nil params
	err := node.ValidateParameters(nil)
	if err != nil {
		t.Errorf("Expected no error with nil params, got %v", err)
	}
	
	// Test with empty params
	params := map[string]interface{}{}
	err = node.ValidateParameters(params)
	if err != nil {
		t.Errorf("Expected no error with empty params, got %v", err)
	}
	
	// Test with invalid mode type
	invalidModeTypeParams := map[string]interface{}{
		"mode": 123, // Not a string
	}
	err = node.ValidateParameters(invalidModeTypeParams)
	if err == nil {
		t.Error("Expected error with invalid mode type, got nil")
	}
	
	// Test with invalid mode value
	invalidModeValueParams := map[string]interface{}{
		"mode": "invalid",
	}
	err = node.ValidateParameters(invalidModeValueParams)
	if err == nil {
		t.Error("Expected error with invalid mode value, got nil")
	}
	
	// Test with valid mode values
	validCombineModeParams := map[string]interface{}{
		"mode": "combine",
	}
	err = node.ValidateParameters(validCombineModeParams)
	if err != nil {
		t.Errorf("Expected no error with valid combine mode, got %v", err)
	}
	
	validSplitModeParams := map[string]interface{}{
		"mode": "split",
	}
	err = node.ValidateParameters(validSplitModeParams)
	if err != nil {
		t.Errorf("Expected no error with valid split mode, got %v", err)
	}
}

func TestItemListsNodeExecuteWithEmptyInput(t *testing.T) {
	node := NewItemListsNode()
	
	inputData := []model.DataItem{}
	nodeParams := map[string]interface{}{
		"mode": "combine",
	}
	
	result, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d items", len(result))
	}
}

func TestItemListsNodeExecuteCombineMode(t *testing.T) {
	node := NewItemListsNode()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"id":   float64(1),
				"name": "Item 1",
			},
		},
		{
			JSON: map[string]interface{}{
				"id":   float64(2),
				"name": "Item 2",
			},
		},
	}
	
	nodeParams := map[string]interface{}{
		"mode": "combine",
	}
	
	result, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result) != 1 {
		t.Fatalf("Expected 1 result item, got %d", len(result))
	}
	
	item := result[0]
	
	// Check that items array is created
	items, ok := item.JSON["items"].([]interface{})
	if !ok {
		t.Fatal("Expected items array in result")
	}
	
	if len(items) != 2 {
		t.Errorf("Expected 2 items in array, got %d", len(items))
	}
	
	// Check first item
	firstItem, ok := items[0].(map[string]interface{})
	if !ok {
		t.Fatal("Expected first item to be a map")
	}
	
	if id, ok := firstItem["id"].(float64); !ok || int(id) != 1 {
		t.Errorf("Expected first item id to be 1, got %v", firstItem["id"])
	}
	
	if name, ok := firstItem["name"].(string); !ok || name != "Item 1" {
		t.Errorf("Expected first item name to be 'Item 1', got %v", firstItem["name"])
	}
	
	// Check second item
	secondItem, ok := items[1].(map[string]interface{})
	if !ok {
		t.Fatal("Expected second item to be a map")
	}
	
	if id, ok := secondItem["id"].(float64); !ok || int(id) != 2 {
		t.Errorf("Expected second item id to be 2, got %v", secondItem["id"])
	}
	
	if name, ok := secondItem["name"].(string); !ok || name != "Item 2" {
		t.Errorf("Expected second item name to be 'Item 2', got %v", secondItem["name"])
	}
}

func TestItemListsNodeExecuteSplitMode(t *testing.T) {
	node := NewItemListsNode()
	
	// Create input data with items array
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{
						"id":   float64(1),
						"name": "Item 1",
					},
					map[string]interface{}{
						"id":   float64(2),
						"name": "Item 2",
					},
				},
			},
		},
	}
	
	nodeParams := map[string]interface{}{
		"mode": "split",
	}
	
	result, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result) != 2 {
		t.Fatalf("Expected 2 result items, got %d", len(result))
	}
	
	// Check first item
	if id, ok := result[0].JSON["id"].(float64); !ok || int(id) != 1 {
		t.Errorf("Expected first item id to be 1, got %v", result[0].JSON["id"])
	}
	
	if name, ok := result[0].JSON["name"].(string); !ok || name != "Item 1" {
		t.Errorf("Expected first item name to be 'Item 1', got %v", result[0].JSON["name"])
	}
	
	// Check second item
	if id, ok := result[1].JSON["id"].(float64); !ok || int(id) != 2 {
		t.Errorf("Expected second item id to be 2, got %v", result[1].JSON["id"])
	}
	
	if name, ok := result[1].JSON["name"].(string); !ok || name != "Item 2" {
		t.Errorf("Expected second item name to be 'Item 2', got %v", result[1].JSON["name"])
	}
}

func TestItemListsNodeExecuteSplitModeWithoutItemsField(t *testing.T) {
	node := NewItemListsNode()
	
	// Create input data without items array (should pass through)
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"id":   float64(1),
				"name": "Item 1",
			},
		},
	}
	
	nodeParams := map[string]interface{}{
		"mode": "split",
	}
	
	result, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result) != 1 {
		t.Fatalf("Expected 1 result item, got %d", len(result))
	}
	
	// Item should be passed through unchanged
	if id, ok := result[0].JSON["id"].(float64); !ok || int(id) != 1 {
		t.Errorf("Expected item id to be 1, got %v", result[0].JSON["id"])
	}
	
	if name, ok := result[0].JSON["name"].(string); !ok || name != "Item 1" {
		t.Errorf("Expected item name to be 'Item 1', got %v", result[0].JSON["name"])
	}
}

func TestItemListsNodeExecuteDefaultMode(t *testing.T) {
	node := NewItemListsNode()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"id":   float64(1),
				"name": "Item 1",
			},
		},
		{
			JSON: map[string]interface{}{
				"id":   float64(2),
				"name": "Item 2",
			},
		},
	}
	
	// No mode specified - should default to combine
	nodeParams := map[string]interface{}{}
	
	result, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result) != 1 {
		t.Fatalf("Expected 1 result item, got %d", len(result))
	}
	
	item := result[0]
	
	// Should behave like combine mode by default
	items, ok := item.JSON["items"].([]interface{})
	if !ok {
		t.Fatal("Expected items array in result")
	}
	
	if len(items) != 2 {
		t.Errorf("Expected 2 items in array, got %d", len(items))
	}
}

func TestItemListsNodeImplementsNodeExecutor(t *testing.T) {
	var _ base.NodeExecutor = (*ItemListsNode)(nil)
}