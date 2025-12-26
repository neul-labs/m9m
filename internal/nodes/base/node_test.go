package base

import (
	"testing"
	"github.com/dipankar/m9m/internal/model"
)

func TestBaseNodeCreation(t *testing.T) {
	description := NodeDescription{
		Name:        "Test Node",
		Description: "A test node",
		Category:    "Test",
	}
	
	node := NewBaseNode(description)
	if node == nil {
		t.Fatal("Expected node to be created, got nil")
	}
	
	desc := node.Description()
	if desc.Name != description.Name {
		t.Errorf("Expected name %s, got %s", description.Name, desc.Name)
	}
}

func TestBaseNodeParameterHelpers(t *testing.T) {
	description := NodeDescription{
		Name:        "Test Node",
		Description: "A test node",
		Category:    "Test",
	}
	
	node := NewBaseNode(description)
	
	params := map[string]interface{}{
		"stringParam": "testValue",
		"intParam":    42,
		"boolParam":   true,
		"floatParam":  3.14,
	}
	
	// Test GetStringParameter
	strValue := node.GetStringParameter(params, "stringParam", "default")
	if strValue != "testValue" {
		t.Errorf("Expected 'testValue', got '%s'", strValue)
	}
	
	strDefault := node.GetStringParameter(params, "missingParam", "default")
	if strDefault != "default" {
		t.Errorf("Expected 'default', got '%s'", strDefault)
	}
	
	// Test GetIntParameter
	intValue := node.GetIntParameter(params, "intParam", 0)
	if intValue != 42 {
		t.Errorf("Expected 42, got %d", intValue)
	}
	
	intDefault := node.GetIntParameter(params, "missingParam", 100)
	if intDefault != 100 {
		t.Errorf("Expected 100, got %d", intDefault)
	}
	
	// Test GetIntParameter with float64 (common in JSON)
	intFromFloat := node.GetIntParameter(params, "floatParam", 0)
	if intFromFloat != 3 {
		t.Errorf("Expected 3, got %d", intFromFloat)
	}
	
	// Test GetBoolParameter
	boolValue := node.GetBoolParameter(params, "boolParam", false)
	if !boolValue {
		t.Errorf("Expected true, got %v", boolValue)
	}
	
	boolDefault := node.GetBoolParameter(params, "missingParam", true)
	if !boolDefault {
		t.Errorf("Expected true, got %v", boolDefault)
	}
	
	// Test with nil params
	nilStr := node.GetStringParameter(nil, "anyParam", "nilDefault")
	if nilStr != "nilDefault" {
		t.Errorf("Expected 'nilDefault', got '%s'", nilStr)
	}
}

func TestBaseNodeValidateParameters(t *testing.T) {
	description := NodeDescription{
		Name:        "Test Node",
		Description: "A test node",
		Category:    "Test",
	}
	
	node := NewBaseNode(description)
	
	// Test with nil params
	err := node.ValidateParameters(nil)
	if err != nil {
		t.Errorf("Expected no error with nil params, got %v", err)
	}
	
	// Test with empty params
	err = node.ValidateParameters(make(map[string]interface{}))
	if err != nil {
		t.Errorf("Expected no error with empty params, got %v", err)
	}
	
	// Test with valid params
	params := map[string]interface{}{
		"param1": "value1",
		"param2": 42,
	}
	
	err = node.ValidateParameters(params)
	if err != nil {
		t.Errorf("Expected no error with valid params, got %v", err)
	}
}

func TestBaseNodeExecute(t *testing.T) {
	description := NodeDescription{
		Name:        "Test Node",
		Description: "A test node",
		Category:    "Test",
	}
	
	node := NewBaseNode(description)
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"test": "data",
			},
		},
	}
	
	nodeParams := map[string]interface{}{
		"nodeParam": "value",
	}
	
	outputData, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(outputData) != len(inputData) {
		t.Errorf("Expected %d data items, got %d", len(inputData), len(outputData))
	}
}

func TestBaseNodeCreateError(t *testing.T) {
	description := NodeDescription{
		Name:        "Test Node",
		Description: "A test node",
		Category:    "Test",
	}
	
	node := NewBaseNode(description)
	
	err := node.CreateError("test error", nil)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	
	if err.Error() != "node Test Node error: test error" {
		t.Errorf("Expected 'node Test Node error: test error', got '%s'", err.Error())
	}
}