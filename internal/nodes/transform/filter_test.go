package transform

import (
	"testing"
	
	"github.com/yourusername/n8n-go/internal/model"
	"github.com/yourusername/n8n-go/internal/nodes/base"
)

func TestFilterNodeCreation(t *testing.T) {
	node := NewFilterNode()
	if node == nil {
		t.Fatal("Expected node to be created, got nil")
	}
	
	desc := node.Description()
	if desc.Name != "Filter" {
		t.Errorf("Expected name 'Filter', got '%s'", desc.Name)
	}
}

func TestFilterNodeValidateParameters(t *testing.T) {
	node := NewFilterNode()
	
	// Test with nil params
	err := node.ValidateParameters(nil)
	if err == nil {
		t.Error("Expected error with nil params, got nil")
	}
	
	// Test with missing conditions
	params := map[string]interface{}{}
	err = node.ValidateParameters(params)
	if err == nil {
		t.Error("Expected error with missing conditions, got nil")
	}
	
	// Test with invalid conditions type
	invalidTypeParams := map[string]interface{}{
		"conditions": "not an array",
	}
	err = node.ValidateParameters(invalidTypeParams)
	if err == nil {
		t.Error("Expected error with invalid conditions type, got nil")
	}
	
	// Test with empty conditions array
	emptyConditionsParams := map[string]interface{}{
		"conditions": []interface{}{},
	}
	err = node.ValidateParameters(emptyConditionsParams)
	if err != nil {
		t.Errorf("Expected no error with empty conditions, got %v", err)
	}
	
	// Test with invalid condition object
	invalidConditionParams := map[string]interface{}{
		"conditions": []interface{}{
			"not an object",
		},
	}
	err = node.ValidateParameters(invalidConditionParams)
	if err == nil {
		t.Error("Expected error with invalid condition object, got nil")
	}
	
	// Test with condition missing leftValue
	missingLeftValueParams := map[string]interface{}{
		"conditions": []interface{}{
			map[string]interface{}{
				"rightValue": "test",
				"operator":   "equals",
			},
		},
	}
	err = node.ValidateParameters(missingLeftValueParams)
	if err == nil {
		t.Error("Expected error with missing leftValue, got nil")
	}
	
	// Test with condition missing rightValue
	missingRightValueParams := map[string]interface{}{
		"conditions": []interface{}{
			map[string]interface{}{
				"leftValue": "$json.name",
				"operator":  "equals",
			},
		},
	}
	err = node.ValidateParameters(missingRightValueParams)
	if err == nil {
		t.Error("Expected error with missing rightValue, got nil")
	}
	
	// Test with condition missing operator
	missingOperatorParams := map[string]interface{}{
		"conditions": []interface{}{
			map[string]interface{}{
				"leftValue":  "$json.name",
				"rightValue": "test",
			},
		},
	}
	err = node.ValidateParameters(missingOperatorParams)
	if err == nil {
		t.Error("Expected error with missing operator, got nil")
	}
	
	// Test with invalid operator type
	invalidOperatorTypeParams := map[string]interface{}{
		"conditions": []interface{}{
			map[string]interface{}{
				"leftValue":  "$json.name",
				"rightValue": "test",
				"operator":   123, // Not a string
			},
		},
	}
	err = node.ValidateParameters(invalidOperatorTypeParams)
	if err == nil {
		t.Error("Expected error with invalid operator type, got nil")
	}
	
	// Test with invalid operator value
	invalidOperatorValueParams := map[string]interface{}{
		"conditions": []interface{}{
			map[string]interface{}{
				"leftValue":  "$json.name",
				"rightValue": "test",
				"operator":   "invalid",
			},
		},
	}
	err = node.ValidateParameters(invalidOperatorValueParams)
	if err == nil {
		t.Error("Expected error with invalid operator value, got nil")
	}
	
	// Test with valid conditions
	validConditionsParams := map[string]interface{}{
		"conditions": []interface{}{
			map[string]interface{}{
				"leftValue":  "$json.name",
				"rightValue": "John",
				"operator":   "equals",
			},
		},
	}
	err = node.ValidateParameters(validConditionsParams)
	if err != nil {
		t.Errorf("Expected no error with valid conditions, got %v", err)
	}
	
	// Test with invalid combiner
	invalidCombinerParams := map[string]interface{}{
		"conditions": []interface{}{
			map[string]interface{}{
				"leftValue":  "$json.name",
				"rightValue": "John",
				"operator":   "equals",
			},
		},
		"combiner": "invalid",
	}
	err = node.ValidateParameters(invalidCombinerParams)
	if err == nil {
		t.Error("Expected error with invalid combiner, got nil")
	}
	
	// Test with valid combiner
	validCombinerParams := map[string]interface{}{
		"conditions": []interface{}{
			map[string]interface{}{
				"leftValue":  "$json.name",
				"rightValue": "John",
				"operator":   "equals",
			},
		},
		"combiner": "and",
	}
	err = node.ValidateParameters(validCombinerParams)
	if err != nil {
		t.Errorf("Expected no error with valid combiner, got %v", err)
	}
	
	// Test with another valid combiner
	anotherValidCombinerParams := map[string]interface{}{
		"conditions": []interface{}{
			map[string]interface{}{
				"leftValue":  "$json.name",
				"rightValue": "John",
				"operator":   "equals",
			},
		},
		"combiner": "or",
	}
	err = node.ValidateParameters(anotherValidCombinerParams)
	if err != nil {
		t.Errorf("Expected no error with another valid combiner, got %v", err)
	}
}

func TestFilterNodeExecuteWithEmptyInput(t *testing.T) {
	node := NewFilterNode()
	
	inputData := []model.DataItem{}
	nodeParams := map[string]interface{}{
		"conditions": []interface{}{
			map[string]interface{}{
				"leftValue":  "$json.name",
				"rightValue": "John",
				"operator":   "equals",
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

func TestFilterNodeExecuteEqualsCondition(t *testing.T) {
	node := NewFilterNode()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"name": "John",
				"age":  float64(30),
			},
		},
		{
			JSON: map[string]interface{}{
				"name": "Jane",
				"age":  float64(25),
			},
		},
	}
	
	nodeParams := map[string]interface{}{
		"conditions": []interface{}{
			map[string]interface{}{
				"leftValue":  "$json.name",
				"rightValue": "John",
				"operator":   "equals",
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
	
	// Check that the item has the expected JSON data
	if name, ok := item.JSON["name"].(string); !ok || name != "John" {
		t.Errorf("Expected name 'John', got %v", item.JSON["name"])
	}
	
	if age, ok := item.JSON["age"].(float64); !ok || int(age) != 30 {
		t.Errorf("Expected age 30, got %v", item.JSON["age"])
	}
}

func TestFilterNodeExecuteMultipleConditionsAND(t *testing.T) {
	node := NewFilterNode()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"name": "John",
				"age":  float64(30),
			},
		},
		{
			JSON: map[string]interface{}{
				"name": "Jane",
				"age":  float64(25),
			},
		},
		{
			JSON: map[string]interface{}{
				"name": "John",
				"age":  float64(25),
			},
		},
	}
	
	nodeParams := map[string]interface{}{
		"conditions": []interface{}{
			map[string]interface{}{
				"leftValue":  "$json.name",
				"rightValue": "John",
				"operator":   "equals",
			},
			map[string]interface{}{
				"leftValue":  "$json.age",
				"rightValue": float64(30),
				"operator":   "equals",
			},
		},
		"combiner": "and",
	}
	
	result, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result) != 1 {
		t.Fatalf("Expected 1 result item, got %d", len(result))
	}
	
	item := result[0]
	
	// Check that the item has the expected JSON data
	if name, ok := item.JSON["name"].(string); !ok || name != "John" {
		t.Errorf("Expected name 'John', got %v", item.JSON["name"])
	}
	
	if age, ok := item.JSON["age"].(float64); !ok || int(age) != 30 {
		t.Errorf("Expected age 30, got %v", item.JSON["age"])
	}
}

func TestFilterNodeExecuteMultipleConditionsOR(t *testing.T) {
	node := NewFilterNode()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"name": "John",
				"age":  float64(30),
			},
		},
		{
			JSON: map[string]interface{}{
				"name": "Jane",
				"age":  float64(25),
			},
		},
		{
			JSON: map[string]interface{}{
				"name": "Bob",
				"age":  float64(35),
			},
		},
	}
	
	nodeParams := map[string]interface{}{
		"conditions": []interface{}{
			map[string]interface{}{
				"leftValue":  "$json.name",
				"rightValue": "John",
				"operator":   "equals",
			},
			map[string]interface{}{
				"leftValue":  "$json.name",
				"rightValue": "Jane",
				"operator":   "equals",
			},
		},
		"combiner": "or",
	}
	
	result, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result) != 2 {
		t.Fatalf("Expected 2 result items, got %d", len(result))
	}
	
	// Check first item
	if name, ok := result[0].JSON["name"].(string); !ok || (name != "John" && name != "Jane") {
		t.Errorf("Expected name 'John' or 'Jane', got %v", result[0].JSON["name"])
	}
	
	// Check second item
	if name, ok := result[1].JSON["name"].(string); !ok || (name != "John" && name != "Jane") {
		t.Errorf("Expected name 'John' or 'Jane', got %v", result[1].JSON["name"])
	}
	
	// Check that both items are different
	if result[0].JSON["name"] == result[1].JSON["name"] {
		t.Error("Expected different items, got same name")
	}
}

func TestFilterNodeExecuteContainsCondition(t *testing.T) {
	node := NewFilterNode()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"email": "john.doe@example.com",
			},
		},
		{
			JSON: map[string]interface{}{
				"email": "jane.smith@test.com",
			},
		},
	}
	
	nodeParams := map[string]interface{}{
		"conditions": []interface{}{
			map[string]interface{}{
				"leftValue":  "$json.email",
				"rightValue": "@example.com",
				"operator":   "contains",
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
	
	// Check that the item has the expected JSON data
	if email, ok := item.JSON["email"].(string); !ok || email != "john.doe@example.com" {
		t.Errorf("Expected email 'john.doe@example.com', got %v", item.JSON["email"])
	}
}

func TestFilterNodeExecuteGreaterThanCondition(t *testing.T) {
	node := NewFilterNode()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"score": float64(85),
			},
		},
		{
			JSON: map[string]interface{}{
				"score": float64(75),
			},
		},
		{
			JSON: map[string]interface{}{
				"score": float64(95),
			},
		},
	}
	
	nodeParams := map[string]interface{}{
		"conditions": []interface{}{
			map[string]interface{}{
				"leftValue":  "$json.score",
				"rightValue": float64(80),
				"operator":   "greaterThan",
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
	
	// Check that both items have scores > 80
	for i, item := range result {
		if score, ok := item.JSON["score"].(float64); !ok || int(score) <= 80 {
			t.Errorf("Expected score > 80 for item %d, got %v", i, item.JSON["score"])
		}
	}
}

func TestFilterNodeImplementsNodeExecutor(t *testing.T) {
	var _ base.NodeExecutor = (*FilterNode)(nil)
}