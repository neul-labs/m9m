package transform

import (
	"testing"
	
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

func TestFunctionNodeCreation(t *testing.T) {
	node := NewFunctionNode()
	if node == nil {
		t.Fatal("Expected node to be created, got nil")
	}
	
	desc := node.Description()
	if desc.Name != "Function" {
		t.Errorf("Expected name 'Function', got '%s'", desc.Name)
	}
}

func TestFunctionNodeValidateParameters(t *testing.T) {
	node := NewFunctionNode()
	
	// Test with nil params
	err := node.ValidateParameters(nil)
	if err == nil {
		t.Error("Expected error with nil params, got nil")
	}
	
	// Test with missing jsCode
	params := map[string]interface{}{}
	err = node.ValidateParameters(params)
	if err == nil {
		t.Error("Expected error with missing jsCode, got nil")
	}
	
	// Test with invalid jsCode type
	invalidTypeParams := map[string]interface{}{
		"jsCode": 123, // Not a string
	}
	err = node.ValidateParameters(invalidTypeParams)
	if err == nil {
		t.Error("Expected error with invalid jsCode type, got nil")
	}
	
	// Test with empty jsCode
	emptyJsCodeParams := map[string]interface{}{
		"jsCode": "", // Empty string
	}
	err = node.ValidateParameters(emptyJsCodeParams)
	if err == nil {
		t.Error("Expected error with empty jsCode, got nil")
	}
	
	// Test with valid jsCode
	validJsCodeParams := map[string]interface{}{
		"jsCode": "var result = $json; result;", // Valid JavaScript code
	}
	err = node.ValidateParameters(validJsCodeParams)
	if err != nil {
		t.Errorf("Expected no error with valid jsCode, got %v", err)
	}
	
	// Test with complex jsCode
	complexJsCodeParams := map[string]interface{}{
		"jsCode": `
			var result = {};
			result.name = $json.name ? $json.name.toUpperCase() : 'UNKNOWN';
			result.processed = true;
			result;
		`,
	}
	err = node.ValidateParameters(complexJsCodeParams)
	if err != nil {
		t.Errorf("Expected no error with complex jsCode, got %v", err)
	}
}

func TestFunctionNodeExecuteWithEmptyInput(t *testing.T) {
	node := NewFunctionNode()
	
	inputData := []model.DataItem{}
	nodeParams := map[string]interface{}{
		"jsCode": "var result = $json; result;",
	}
	
	result, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d items", len(result))
	}
}

func TestFunctionNodeExecuteSimpleReturn(t *testing.T) {
	node := NewFunctionNode()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"name": "John",
				"age":  float64(30), // JSON numbers are float64
			},
		},
	}
	
	nodeParams := map[string]interface{}{
		"jsCode": "var result = $json; result;", // Valid JavaScript expression
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

func TestFunctionNodeExecuteWithStringManipulation(t *testing.T) {
	node := NewFunctionNode()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"name": "john",
			},
		},
	}
	
	nodeParams := map[string]interface{}{
		"jsCode": "var result = { uppercaseName: $json.name.toUpperCase() }; result;", // Valid JavaScript expression
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
	if uppercaseName, ok := item.JSON["uppercaseName"].(string); !ok || uppercaseName != "JOHN" {
		t.Errorf("Expected uppercaseName 'JOHN', got %v", item.JSON["uppercaseName"])
	}
}

func TestFunctionNodeExecuteWithMathOperations(t *testing.T) {
	node := NewFunctionNode()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"a": float64(10),
				"b": float64(5),
			},
		},
	}
	
	nodeParams := map[string]interface{}{
		"jsCode": "var result = { sum: $json.a + $json.b, product: $json.a * $json.b }; result;", // Valid JavaScript expression
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
	// Note: goja returns int64 for integer results, so we handle both int64 and float64
	sum := getNumberValue(item.JSON["sum"])
	if sum != 15 {
		t.Errorf("Expected sum 15, got %v (type %T)", item.JSON["sum"], item.JSON["sum"])
	}

	product := getNumberValue(item.JSON["product"])
	if product != 50 {
		t.Errorf("Expected product 50, got %v (type %T)", item.JSON["product"], item.JSON["product"])
	}
}

// getNumberValue extracts a number from an interface{} regardless of its underlying type
func getNumberValue(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	default:
		return 0
	}
}

func TestFunctionNodeExecuteWithConditionalLogic(t *testing.T) {
	node := NewFunctionNode()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"score": float64(85),
			},
		},
	}
	
	nodeParams := map[string]interface{}{
		"jsCode": `
			var result = {};
			result.grade = $json.score >= 90 ? 'A' : 
			              $json.score >= 80 ? 'B' : 
			              $json.score >= 70 ? 'C' : 'F';
			result.passed = $json.score >= 70;
			result;
		`, // Valid JavaScript expression
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
	if grade, ok := item.JSON["grade"].(string); !ok || grade != "B" {
		t.Errorf("Expected grade 'B', got %v", item.JSON["grade"])
	}
	
	if passed, ok := item.JSON["passed"].(bool); !ok || !passed {
		t.Errorf("Expected passed true, got %v", item.JSON["passed"])
	}
}

func TestFunctionNodeExecuteWithError(t *testing.T) {
	node := NewFunctionNode()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"name": "John",
			},
		},
	}
	
	nodeParams := map[string]interface{}{
		"jsCode": "throw new Error('Test error');", // Valid JavaScript that throws an error
	}
	
	_, err := node.Execute(inputData, nodeParams)
	if err == nil {
		t.Error("Expected error with invalid JavaScript, got nil")
	}
	
	// Check that the error contains the expected message
	if !contains(err.Error(), "Function error") {
		t.Errorf("Expected error to contain 'Function error', got %v", err)
	}
}

func TestFunctionNodeExecuteWithSyntaxError(t *testing.T) {
	node := NewFunctionNode()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"name": "John",
			},
		},
	}
	
	nodeParams := map[string]interface{}{
		"jsCode": "var result = $json.name.toUpperCase(; result;", // Missing closing parenthesis
	}
	
	_, err := node.Execute(inputData, nodeParams)
	if err == nil {
		t.Error("Expected error with invalid JavaScript syntax, got nil")
	}
	
	// Check that the error contains the expected message
	if !contains(err.Error(), "Function error") {
		t.Errorf("Expected error to contain 'Function error', got %v", err)
	}
}

func TestFunctionNodeImplementsNodeExecutor(t *testing.T) {
	var _ base.NodeExecutor = (*FunctionNode)(nil)
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}