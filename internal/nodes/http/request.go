/*
Package http provides HTTP-related node implementations for n8n-go.
*/
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	
	"github.com/dipankar/m9m/internal/model"
	"github.com/dipankar/m9m/internal/nodes/base"
	"github.com/dipankar/m9m/internal/expressions"
)

// HTTPRequestNode implements the HTTP Request node functionality
type HTTPRequestNode struct {
	*base.BaseNode
	client *http.Client
}

// NewHTTPRequestNode creates a new HTTP Request node
func NewHTTPRequestNode() *HTTPRequestNode {
	description := base.NodeDescription{
		Name:        "HTTP Request",
		Description: "Makes HTTP requests to REST APIs and other web services",
		Category:    "HTTP",
	}
	
	return &HTTPRequestNode{
		BaseNode: base.NewBaseNode(description),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Description returns the node description
func (h *HTTPRequestNode) Description() base.NodeDescription {
	return h.BaseNode.Description()
}

// ValidateParameters validates HTTP Request node parameters
func (h *HTTPRequestNode) ValidateParameters(params map[string]interface{}) error {
	if params == nil {
		return h.CreateError("parameters cannot be nil", nil)
	}
	
	// Check if url exists
	url := h.GetStringParameter(params, "url", "")
	if url == "" {
		return h.CreateError("url parameter is required", nil)
	}
	
	// Check if method exists
	method := h.GetStringParameter(params, "method", "GET")
	
	// Allow expressions in method parameter
	if expressions.IsExpression(method) {
		// Method is an expression, so we can't validate it at this stage
		// It will be validated during execution after expression evaluation
		return nil
	}
	
	// Validate method if it's not an expression
	validMethods := map[string]bool{
		"GET":     true,
		"POST":    true,
		"PUT":     true,
		"PATCH":   true,
		"DELETE":  true,
		"HEAD":    true,
		"OPTIONS": true,
	}
	
	if !validMethods[strings.ToUpper(method)] {
		return h.CreateError(fmt.Sprintf("invalid HTTP method: %s", method), nil)
	}
	
	return nil
}

// Execute processes the HTTP Request node operation
func (h *HTTPRequestNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	if len(inputData) == 0 {
		return []model.DataItem{}, nil
	}
	
	// Create execution context for expression evaluation
	execContext := &expressions.ExecutionContext{
		InputData: inputData,
		ItemIndex: 0,
		Variables: make(map[string]interface{}),
	}
	
	// Process each input data item
	result := make([]model.DataItem, len(inputData))
	
	for i, item := range inputData {
		// Update execution context for current item
		execContext.ItemIndex = i
		execContext.Variables["json"] = item.JSON
		
		// Evaluate node parameters with expression context
		evaluatedParams, err := h.evaluateParameters(nodeParams, execContext)
		if err != nil {
			return nil, h.CreateError(fmt.Sprintf("failed to evaluate parameters: %v", err), nil)
		}
		
		// Get required parameters from evaluated parameters
		url := h.GetStringParameter(evaluatedParams, "url", "")
		if url == "" {
			return nil, h.CreateError("url parameter cannot be empty", nil)
		}
		
		method := h.GetStringParameter(evaluatedParams, "method", "GET")
		
		// Validate method after evaluation
		validMethods := map[string]bool{
			"GET":     true,
			"POST":    true,
			"PUT":     true,
			"PATCH":   true,
			"DELETE":  true,
			"HEAD":    true,
			"OPTIONS": true,
		}
		
		if !validMethods[strings.ToUpper(method)] {
			return nil, h.CreateError(fmt.Sprintf("invalid HTTP method: %s", method), nil)
		}
		
		// Create HTTP request
		req, err := http.NewRequest(strings.ToUpper(method), url, nil)
		if err != nil {
			return nil, h.CreateError(fmt.Sprintf("failed to create request: %v", err), nil)
		}
		
		// Add headers if provided
		if headers, ok := evaluatedParams["headers"].(map[string]interface{}); ok {
			for key, value := range headers {
				if strValue, ok := value.(string); ok {
					req.Header.Set(key, strValue)
				}
			}
		}
		
		// Add body for POST, PUT, PATCH requests
		if method == "POST" || method == "PUT" || method == "PATCH" {
			if body, ok := evaluatedParams["body"].(string); ok && body != "" {
				req.Body = io.NopCloser(strings.NewReader(body))
				if req.Header.Get("Content-Type") == "" {
					req.Header.Set("Content-Type", "application/json")
				}
			} else if body, ok := evaluatedParams["body"].(map[string]interface{}); ok {
				// Convert map to JSON
				jsonBody, err := json.Marshal(body)
				if err != nil {
					return nil, h.CreateError(fmt.Sprintf("failed to marshal body: %v", err), nil)
				}
				req.Body = io.NopCloser(bytes.NewReader(jsonBody))
				if req.Header.Get("Content-Type") == "" {
					req.Header.Set("Content-Type", "application/json")
				}
			}
		}
		
		// Set default user agent if not provided
		if req.Header.Get("User-Agent") == "" {
			req.Header.Set("User-Agent", "n8n-go/1.0")
		}
		
		// Execute request
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		req = req.WithContext(ctx)
		
		resp, err := h.client.Do(req)
		if err != nil {
			return nil, h.CreateError(fmt.Sprintf("request failed: %v", err), nil)
		}
		defer resp.Body.Close()
		
		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, h.CreateError(fmt.Sprintf("failed to read response body: %v", err), nil)
		}
		
		// Parse response headers
		headers := make(map[string]interface{})
		for key, values := range resp.Header {
			if len(values) > 0 {
				headers[key] = values[0] // Take first value
			}
		}
		
		// Create response data item
		responseData := model.DataItem{
			JSON: map[string]interface{}{
				"statusCode": resp.StatusCode,
				"headers":    headers,
				"body":       string(body),
			},
		}
		
		// Try to parse JSON response
		if strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
			var jsonResponse interface{}
			if err := json.Unmarshal(body, &jsonResponse); err == nil {
				responseData.JSON["json"] = jsonResponse
			}
		}
		
		// Copy binary data if present
		if item.Binary != nil {
			responseData.Binary = make(map[string]model.BinaryData)
			for k, v := range item.Binary {
				responseData.Binary[k] = v
			}
		}
		
		// Copy paired item data if present
		if item.PairedItem != nil {
			responseData.PairedItem = item.PairedItem
		}
		
		result[i] = responseData
	}
	
	return result, nil
}

// evaluateParameters evaluates node parameters with expression context
func (h *HTTPRequestNode) evaluateParameters(params map[string]interface{}, context *expressions.ExecutionContext) (map[string]interface{}, error) {
	evaluator := expressions.NewExpressionEvaluator()
	
	evaluatedParams := make(map[string]interface{})
	
	for key, value := range params {
		switch v := value.(type) {
		case string:
			// Check if this is an expression
			if expressions.IsExpression(v) {
				// Evaluate the expression
				evaluatedValue, err := evaluator.Evaluate(v, context)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate expression for parameter %s: %v", key, err)
				}
				evaluatedParams[key] = evaluatedValue
			} else {
				// Use the literal value
				evaluatedParams[key] = v
			}
		case map[string]interface{}:
			// Recursively evaluate nested maps
			nested, err := h.evaluateParameters(v, context)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate nested parameters for %s: %v", key, err)
			}
			evaluatedParams[key] = nested
		default:
			// Use the literal value
			evaluatedParams[key] = v
		}
	}
	
	return evaluatedParams, nil
}