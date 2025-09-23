package transform

import (
	"testing"
	"github.com/yourusername/n8n-go/internal/model"
	"github.com/yourusername/n8n-go/internal/nodes/base"
)

func TestSplitInBatchesNodeCreation(t *testing.T) {
	node := NewSplitInBatchesNode()
	if node == nil {
		t.Fatal("Expected node to be created, got nil")
	}
	
	desc := node.Description()
	if desc.Name != "Split In Batches" {
		t.Errorf("Expected name 'Split In Batches', got '%s'", desc.Name)
	}
}

func TestSplitInBatchesNodeValidateParameters(t *testing.T) {
	node := NewSplitInBatchesNode()
	
	// Test with nil params
	err := node.ValidateParameters(nil)
	if err == nil {
		t.Error("Expected error with nil params, got nil")
	}
	
	// Test with missing batchSize
	params := map[string]interface{}{}
	err = node.ValidateParameters(params)
	if err == nil {
		t.Error("Expected error with missing batchSize, got nil")
	}
	
	// Test with invalid batchSize type
	invalidTypeParams := map[string]interface{}{
		"batchSize": "not a number",
	}
	err = node.ValidateParameters(invalidTypeParams)
	if err == nil {
		t.Error("Expected error with invalid batchSize type, got nil")
	}
	
	// Test with zero batchSize (should be rejected during validation)
	zeroBatchSizeParams := map[string]interface{}{
		"batchSize": 0,
	}
	err = node.ValidateParameters(zeroBatchSizeParams)
	if err == nil {
		t.Error("Expected error with zero batchSize, got nil")
	}
	
	// Test with negative batchSize (should be rejected during validation)
	negativeBatchSizeParams := map[string]interface{}{
		"batchSize": -5,
	}
	err = node.ValidateParameters(negativeBatchSizeParams)
	if err == nil {
		t.Error("Expected error with negative batchSize, got nil")
	}
	
	// Test with valid batchSize
	validBatchSizeParams := map[string]interface{}{
		"batchSize": 10,
	}
	err = node.ValidateParameters(validBatchSizeParams)
	if err != nil {
		t.Errorf("Expected no error with valid batchSize, got %v", err)
	}
	
	// Test with valid options
	validOptionsParams := map[string]interface{}{
		"batchSize": 10,
		"options": map[string]interface{}{
			"reset": true,
		},
	}
	err = node.ValidateParameters(validOptionsParams)
	if err != nil {
		t.Errorf("Expected no error with valid options, got %v", err)
	}
	
	// Test with valid options and reset false
	validOptionsResetFalseParams := map[string]interface{}{
		"batchSize": 10,
		"options": map[string]interface{}{
			"reset": false,
		},
	}
	err = node.ValidateParameters(validOptionsResetFalseParams)
	if err != nil {
		t.Errorf("Expected no error with valid options and reset false, got %v", err)
	}
}

func TestSplitInBatchesNodeExecuteWithEmptyInput(t *testing.T) {
	node := NewSplitInBatchesNode()
	
	inputData := []model.DataItem{}
	nodeParams := map[string]interface{}{
		"batchSize": 10,
	}
	
	result, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d items", len(result))
	}
}

func TestSplitInBatchesNodeExecuteWithSmallBatchSize(t *testing.T) {
	node := NewSplitInBatchesNode()
	
	// Create 5 data items
	inputData := make([]model.DataItem, 5)
	for i := 0; i < 5; i++ {
		inputData[i] = model.DataItem{
			JSON: map[string]interface{}{
				"id":   float64(i + 1),
				"name": "Item 1",
			},
		}
	}
	
	nodeParams := map[string]interface{}{
		"batchSize": 2, // Batch size of 2
	}
	
	result, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	// Should return the first batch (2 items)
	if len(result) != 2 {
		t.Fatalf("Expected 2 items in first batch, got %d", len(result))
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
	
	if name, ok := result[1].JSON["name"].(string); !ok || name != "Item 1" {
		t.Errorf("Expected second item name to be 'Item 1', got %v", result[1].JSON["name"])
	}
}

func TestSplitInBatchesNodeExecuteWithLargeBatchSize(t *testing.T) {
	node := NewSplitInBatchesNode()
	
	// Create 3 data items
	inputData := make([]model.DataItem, 3)
	for i := 0; i < 3; i++ {
		inputData[i] = model.DataItem{
			JSON: map[string]interface{}{
				"id":   float64(i + 1),
				"name": "Item 1",
			},
		}
	}
	
	nodeParams := map[string]interface{}{
		"batchSize": 10, // Batch size larger than data
	}
	
	result, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result) != 3 {
		t.Fatalf("Expected 3 result items, got %d", len(result))
	}
	
	// Check that all items are present
	for i := 0; i < 3; i++ {
		if id, ok := result[i].JSON["id"].(float64); !ok || int(id) != i+1 {
			t.Errorf("Expected item %d id to be %d, got %v", i, i+1, result[i].JSON["id"])
		}
		
		if name, ok := result[i].JSON["name"].(string); !ok || name != "Item 1" {
			t.Errorf("Expected item %d name to be 'Item 1', got %v", i, result[i].JSON["name"])
		}
	}
}

func TestSplitInBatchesNodeExecuteWithDefaultBatchSize(t *testing.T) {
	node := NewSplitInBatchesNode()
	
	// Create 15 data items
	inputData := make([]model.DataItem, 15)
	for i := 0; i < 15; i++ {
		inputData[i] = model.DataItem{
			JSON: map[string]interface{}{
				"id":   float64(i + 1),
				"name": "Item 1",
			},
		}
	}
	
	nodeParams := map[string]interface{}{
		"batchSize": 10, // Default batch size
	}
	
	result, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	// Should return the first batch (10 items)
	if len(result) != 10 {
		t.Fatalf("Expected 10 items in first batch, got %d", len(result))
	}
	
	// Check that the first 10 items are present
	for i := 0; i < 10; i++ {
		if id, ok := result[i].JSON["id"].(float64); !ok || int(id) != i+1 {
			t.Errorf("Expected item %d id to be %d, got %v", i, i+1, result[i].JSON["id"])
		}
		
		if name, ok := result[i].JSON["name"].(string); !ok || name != "Item 1" {
			t.Errorf("Expected item %d name to be 'Item 1', got %v", i, result[i].JSON["name"])
		}
	}
}

func TestSplitInBatchesNodeExecuteWithOptions(t *testing.T) {
	node := NewSplitInBatchesNode()
	
	// Create 5 data items
	inputData := make([]model.DataItem, 5)
	for i := 0; i < 5; i++ {
		inputData[i] = model.DataItem{
			JSON: map[string]interface{}{
				"id":   float64(i + 1),
				"name": "Item 1",
			},
		}
	}
	
	nodeParams := map[string]interface{}{
		"batchSize": 2,
		"options": map[string]interface{}{
			"reset": true,
		},
	}
	
	result, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result) != 2 {
		t.Fatalf("Expected 2 items in first batch, got %d", len(result))
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
	
	if name, ok := result[1].JSON["name"].(string); !ok || name != "Item 1" {
		t.Errorf("Expected second item name to be 'Item 1', got %v", result[1].JSON["name"])
	}
}

func TestSplitInBatchesNodeImplementsNodeExecutor(t *testing.T) {
	var _ base.NodeExecutor = (*SplitInBatchesNode)(nil)
}