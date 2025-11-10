package expressions

import (
	"fmt"
	"testing"
	
	"github.com/dipankar/n8n-go/internal/model"
)

func TestExpressionEvaluatorCreation(t *testing.T) {
	evaluator := NewExpressionEvaluator()
	if evaluator == nil {
		t.Fatal("Expected evaluator to be created, got nil")
	}
	
	if len(evaluator.functions) == 0 {
		t.Error("Expected built-in functions to be registered")
	}
}

func TestExpressionEvaluatorValidate(t *testing.T) {
	evaluator := NewExpressionEvaluator()
	
	// Test valid expression
	err := evaluator.Validate("Hello {{ $json.name }}")
	if err != nil {
		t.Errorf("Expected valid expression, got error: %v", err)
	}
	
	// Test invalid expression with unbalanced braces
	err = evaluator.Validate("Hello {{ $json.name ")
	if err == nil {
		t.Error("Expected error for unbalanced braces, got nil")
	}
	
	// Test empty expression
	err = evaluator.Validate("")
	if err != nil {
		t.Errorf("Expected empty expression to be valid, got error: %v", err)
	}
}

func TestExpressionEvaluatorEvaluateLiterals(t *testing.T) {
	evaluator := NewExpressionEvaluator()
	context := &ExecutionContext{}
	
	// Test literal string
	result, err := evaluator.Evaluate("Hello World", context)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if result != "Hello World" {
		t.Errorf("Expected 'Hello World', got '%v'", result)
	}
	
	// Test empty string
	result, err = evaluator.Evaluate("", context)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if result != "" {
		t.Errorf("Expected empty string, got '%v'", result)
	}
}

func TestExpressionEvaluatorResolveVariable(t *testing.T) {
	evaluator := NewExpressionEvaluator()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"name": "John",
				"age":  30,
				"address": map[string]interface{}{
					"city": "New York",
					"zip":  "10001",
				},
			},
		},
	}
	
	context := &ExecutionContext{
		InputData: inputData,
		ItemIndex: 0,
		Variables: map[string]interface{}{
			"url": "https://api.example.com",
		},
	}
	
	// Test $json variable
	result, err := evaluator.Evaluate("{{ $json.name }}", context)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if result != "John" {
		t.Errorf("Expected 'John', got '%v'", result)
	}
	
	// Test $json nested variable
	result, err = evaluator.Evaluate("{{ $json.address.city }}", context)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if result != "New York" {
		t.Errorf("Expected 'New York', got '%v'", result)
	}
	
	// Test $parameter variable
	result, err = evaluator.Evaluate("{{ $parameter.url }}", context)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if result != "https://api.example.com" {
		t.Errorf("Expected 'https://api.example.com', got '%v'", result)
	}
	
	// Test $workflow variable
	result, err = evaluator.Evaluate("{{ $workflow.name }}", context)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if result != "Test Workflow" {
		t.Errorf("Expected 'Test Workflow', got '%v'", result)
	}
}

func TestExpressionEvaluatorCallFunction(t *testing.T) {
	evaluator := NewExpressionEvaluator()
	context := &ExecutionContext{}
	
	// Test uppercase function
	result, err := evaluator.Evaluate("{{ uppercase('hello') }}", context)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if result != "HELLO" {
		t.Errorf("Expected 'HELLO', got '%v'", result)
	}
	
	// Test lowercase function
	result, err = evaluator.Evaluate("{{ lowercase('WORLD') }}", context)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if result != "world" {
		t.Errorf("Expected 'world', got '%v'", result)
	}
	
	// Test add function
	result, err = evaluator.Evaluate("{{ add(1, 2, 3) }}", context)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	// Handle both int and float results
	resultStr := fmt.Sprintf("%v", result)
	if resultStr != "6" && resultStr != "6.0" {
		t.Errorf("Expected 6 or 6.0, got '%v'", result)
	}
	
	// Test length function with string
	result, err = evaluator.Evaluate("{{ length('hello') }}", context)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	resultStr = fmt.Sprintf("%v", result)
	if resultStr != "5" {
		t.Errorf("Expected 5, got '%v'", result)
	}
}

func TestExpressionEvaluatorComplexExpression(t *testing.T) {
	evaluator := NewExpressionEvaluator()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"firstName": "John",
				"lastName":  "Doe",
			},
		},
	}
	
	context := &ExecutionContext{
		InputData: inputData,
		ItemIndex: 0,
	}
	
	// Test complex expression with variable substitution and function
	result, err := evaluator.Evaluate("Hello {{ $json.firstName }} {{ uppercase($json.lastName) }}!", context)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	// The expression evaluator should replace variables first, then functions
	expected := "Hello John DOE!"
	if result != expected {
		t.Errorf("Expected '%s', got '%v'", expected, result)
	}
}

func TestExpressionEvaluatorWithEqualsPrefix(t *testing.T) {
	evaluator := NewExpressionEvaluator()
	context := &ExecutionContext{}
	
	// Test expression with = prefix
	result, err := evaluator.Evaluate("={{ uppercase('hello') }}", context)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if result != "HELLO" {
		t.Errorf("Expected 'HELLO', got '%v'", result)
	}
	
	// Test empty expression with = prefix
	result, err = evaluator.Evaluate("=", context)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if result != "" {
		t.Errorf("Expected empty string, got '%v'", result)
	}
	
	// Test literal with = prefix
	result, err = evaluator.Evaluate("=Hello World", context)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if result != "Hello World" {
		t.Errorf("Expected 'Hello World', got '%v'", result)
	}
}