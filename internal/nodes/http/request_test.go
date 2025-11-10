package http

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	
	"github.com/dipankar/n8n-go/internal/model"
	"github.com/dipankar/n8n-go/internal/nodes/base"
)

func TestHTTPRequestNodeCreation(t *testing.T) {
	node := NewHTTPRequestNode()
	if node == nil {
		t.Fatal("Expected node to be created, got nil")
	}
	
	desc := node.Description()
	if desc.Name != "HTTP Request" {
		t.Errorf("Expected name 'HTTP Request', got '%s'", desc.Name)
	}
}

func TestHTTPRequestNodeValidateParameters(t *testing.T) {
	node := NewHTTPRequestNode()
	
	// Test with nil params
	err := node.ValidateParameters(nil)
	if err == nil {
		t.Error("Expected error with nil params, got nil")
	}
	
	// Test with empty params
	err = node.ValidateParameters(make(map[string]interface{}))
	if err == nil {
		t.Error("Expected error with empty params, got nil")
	}
	
	// Test with valid params
	params := map[string]interface{}{
		"url":    "https://api.example.com",
		"method": "GET",
	}
	
	err = node.ValidateParameters(params)
	if err != nil {
		t.Errorf("Expected no error with valid params, got %v", err)
	}
	
	// Test with invalid method
	invalidParams := map[string]interface{}{
		"url":    "https://api.example.com",
		"method": "INVALID",
	}
	
	err = node.ValidateParameters(invalidParams)
	if err == nil {
		t.Error("Expected error with invalid method, got nil")
	}
}

func TestHTTPRequestNodeExecuteWithMissingURL(t *testing.T) {
	node := NewHTTPRequestNode()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"method": "GET",
			},
		},
	}
	
	nodeParams := map[string]interface{}{
		"method": "GET",
	}
	
	_, err := node.Execute(inputData, nodeParams)
	if err == nil {
		t.Error("Expected error with missing URL, got nil")
	}
}

func TestHTTPRequestNodeExecuteWithValidRequest(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "success",
			"data":    "test",
		})
	}))
	defer server.Close()
	
	node := NewHTTPRequestNode()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{},
		},
	}
	
	nodeParams := map[string]interface{}{
		"url":    server.URL,
		"method": "GET",
	}
	
	result, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result) != 1 {
		t.Fatalf("Expected 1 result item, got %d", len(result))
	}
	
	responseData := result[0]
	
	// Check status code
	statusCode, ok := responseData.JSON["statusCode"]
	if !ok {
		t.Fatal("Expected statusCode to be present")
	}
	
	// Convert to int for comparison
	var statusCodeInt int
	switch v := statusCode.(type) {
	case float64:
		statusCodeInt = int(v)
	case int:
		statusCodeInt = v
	default:
		t.Fatalf("Unexpected status code type: %T", statusCode)
	}
	
	if statusCodeInt != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, statusCodeInt)
	}
	
	// Check that JSON response is parsed
	jsonData, ok := responseData.JSON["json"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected JSON response to be parsed")
	}
	
	message, ok := jsonData["message"].(string)
	if !ok || message != "success" {
		t.Errorf("Expected message 'success', got '%v'", message)
	}
}

func TestHTTPRequestNodeExecutePostRequest(t *testing.T) {
	// Create a test server
	var receivedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}
		receivedBody = string(body)
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"received": string(body),
		})
	}))
	defer server.Close()
	
	node := NewHTTPRequestNode()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{},
		},
	}
	
	nodeParams := map[string]interface{}{
		"url":    server.URL,
		"method": "POST",
		"body": map[string]interface{}{
			"name": "test",
			"age":  30,
		},
	}
	
	result, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result) != 1 {
		t.Fatalf("Expected 1 result item, got %d", len(result))
	}
	
	// Check that we received the body
	if receivedBody == "" {
		t.Error("Expected to receive body data")
	}
	
	responseData := result[0]
	
	// Check status code
	statusCode, ok := responseData.JSON["statusCode"]
	if !ok {
		t.Fatal("Expected statusCode to be present")
	}
	
	// Convert to int for comparison
	var statusCodeInt int
	switch v := statusCode.(type) {
	case float64:
		statusCodeInt = int(v)
	case int:
		statusCodeInt = v
	default:
		t.Fatalf("Unexpected status code type: %T", statusCode)
	}
	
	if statusCodeInt != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, statusCodeInt)
	}
}

func TestHTTPRequestNodeImplementsNodeExecutor(t *testing.T) {
	var _ base.NodeExecutor = (*HTTPRequestNode)(nil)
}