package transform

import (
	"testing"
	
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

func TestCodeNodeCreation(t *testing.T) {
	node := NewCodeNode()
	if node == nil {
		t.Fatal("Expected node to be created, got nil")
	}
	
	desc := node.Description()
	if desc.Name != "Code" {
		t.Errorf("Expected name 'Code', got '%s'", desc.Name)
	}
}

func TestCodeNodeValidateParameters(t *testing.T) {
	node := NewCodeNode()
	
	// Test with nil params
	err := node.ValidateParameters(nil)
	if err == nil {
		t.Error("Expected error with nil params, got nil")
	}
	
	// Test with missing mode
	params := map[string]interface{}{}
	err = node.ValidateParameters(params)
	if err == nil {
		t.Error("Expected error with missing mode, got nil")
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
	
	// Test with valid mode
	validModeParams := map[string]interface{}{
		"mode": "runOnceForAllItems",
	}
	err = node.ValidateParameters(validModeParams)
	if err == nil {
		t.Error("Expected error with missing language, got nil")
	}
	
	// Test with missing language
	missingLanguageParams := map[string]interface{}{
		"mode": "runOnceForAllItems",
	}
	err = node.ValidateParameters(missingLanguageParams)
	if err == nil {
		t.Error("Expected error with missing language, got nil")
	}
	
	// Test with invalid language type
	invalidLanguageTypeParams := map[string]interface{}{
		"mode":     "runOnceForAllItems",
		"language": 123, // Not a string
	}
	err = node.ValidateParameters(invalidLanguageTypeParams)
	if err == nil {
		t.Error("Expected error with invalid language type, got nil")
	}
	
	// Test with invalid language value
	invalidLanguageValueParams := map[string]interface{}{
		"mode":     "runOnceForAllItems",
		"language": "invalid",
	}
	err = node.ValidateParameters(invalidLanguageValueParams)
	if err == nil {
		t.Error("Expected error with invalid language value, got nil")
	}
	
	// Test with valid language but missing code
	validLanguageParams := map[string]interface{}{
		"mode":     "runOnceForAllItems",
		"language": "javascript",
	}
	err = node.ValidateParameters(validLanguageParams)
	if err == nil {
		t.Error("Expected error with missing code, got nil")
	}
	
	// Test with invalid code type
	invalidCodeTypeParams := map[string]interface{}{
		"mode":     "runOnceForAllItems",
		"language": "javascript",
		"code":     123, // Not a string
	}
	err = node.ValidateParameters(invalidCodeTypeParams)
	if err == nil {
		t.Error("Expected error with invalid code type, got nil")
	}
	
	// Test with valid parameters
	validParams := map[string]interface{}{
		"mode":     "runOnceForAllItems",
		"language": "javascript",
		"code":     "var result = $json; result;", // Valid JavaScript expression
	}
	err = node.ValidateParameters(validParams)
	if err != nil {
		t.Errorf("Expected no error with valid parameters, got %v", err)
	}
	
	// Test with another valid mode
	anotherValidModeParams := map[string]interface{}{
		"mode":     "runOnceForEachItem",
		"language": "python",
		"code":     "print('Hello, World!')",
	}
	err = node.ValidateParameters(anotherValidModeParams)
	if err != nil {
		t.Errorf("Expected no error with another valid mode, got %v", err)
	}
	
	// Test with another valid language
	anotherValidLanguageParams := map[string]interface{}{
		"mode":     "runOnceForAllItems",
		"language": "go",
		"code":     "fmt.Println(\"Hello, World!\")",
	}
	err = node.ValidateParameters(anotherValidLanguageParams)
	if err != nil {
		t.Errorf("Expected no error with another valid language, got %v", err)
	}
}

func TestCodeNodeExecuteWithEmptyInput(t *testing.T) {
	node := NewCodeNode()
	
	inputData := []model.DataItem{}
	nodeParams := map[string]interface{}{
		"mode":     "runOnceForAllItems",
		"language": "javascript",
		"code":     "var result = $json; result;", // Valid JavaScript expression
	}
	
	result, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d items", len(result))
	}
}

func TestCodeNodeExecuteJavaScript(t *testing.T) {
	node := NewCodeNode()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"name": "John",
				"age":  float64(30),
			},
		},
	}
	
	nodeParams := map[string]interface{}{
		"mode":     "runOnceForAllItems",
		"language": "javascript",
		"code":     "var result = $json; result;", // Valid JavaScript expression
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

func TestCodeNodeExecutePython(t *testing.T) {
	node := NewCodeNode()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"name": "John",
				"age":  float64(30),
			},
		},
	}
	
	nodeParams := map[string]interface{}{
		"mode":     "runOnceForAllItems",
		"language": "python",
		"code":     "print('Hello, World!')",
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
	
	// Check that Python execution result is present
	if pythonResult, ok := item.JSON["pythonResult"].(string); !ok || pythonResult == "" {
		t.Errorf("Expected Python execution result, got %v", item.JSON["pythonResult"])
	}
}

func TestCodeNodeExecuteGo(t *testing.T) {
	node := NewCodeNode()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"name": "John",
				"age":  float64(30),
			},
		},
	}
	
	nodeParams := map[string]interface{}{
		"mode":     "runOnceForAllItems",
		"language": "go",
		"code":     "fmt.Println(\"Hello, World!\")",
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
	
	// Check that Go execution result is present
	if goResult, ok := item.JSON["goResult"].(string); !ok || goResult == "" {
		t.Errorf("Expected Go execution result, got %v", item.JSON["goResult"])
	}
}

func TestCodeNodeExecuteWithInvalidLanguage(t *testing.T) {
	node := NewCodeNode()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"name": "John",
				"age":  float64(30),
			},
		},
	}
	
	nodeParams := map[string]interface{}{
		"mode":     "runOnceForAllItems",
		"language": "invalid",
		"code":     "var result = $json; result;", // Valid JavaScript expression
	}
	
	_, err := node.Execute(inputData, nodeParams)
	if err == nil {
		t.Error("Expected error with invalid language, got nil")
	}
}

func TestCodeNodeExecuteWithMissingCode(t *testing.T) {
	node := NewCodeNode()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"name": "John",
				"age":  float64(30),
			},
		},
	}
	
	nodeParams := map[string]interface{}{
		"mode":     "runOnceForAllItems",
		"language": "javascript",
		"code":     "", // Empty code
	}
	
	_, err := node.Execute(inputData, nodeParams)
	if err == nil {
		t.Error("Expected error with missing code, got nil")
	}
}

func TestCodeNodeImplementsNodeExecutor(t *testing.T) {
	var _ base.NodeExecutor = (*CodeNode)(nil)
}