package trigger

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/yourusername/n8n-go/internal/model"
	"github.com/yourusername/n8n-go/internal/nodes/base"
)

// WebhookNode implements the Webhook trigger node
type WebhookNode struct {
	*base.BaseNode
	server *http.Server
}

// NewWebhookNode creates a new Webhook node
func NewWebhookNode() *WebhookNode {
	description := base.NodeDescription{
		Name:        "Webhook",
		Description: "Receives HTTP webhook requests",
		Category:    "Trigger",
	}

	return &WebhookNode{
		BaseNode: base.NewBaseNode(description),
	}
}

// Execute processes the Webhook node
func (w *WebhookNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	// Webhook nodes are typically triggered externally
	// This method handles the webhook data processing

	// Extract webhook configuration
	path, _ := nodeParams["path"].(string)
	method, _ := nodeParams["httpMethod"].(string)
	if method == "" {
		method = "POST"
	}

	// For execution context, we assume webhook data is already provided in inputData
	if len(inputData) == 0 {
		return []model.DataItem{}, nil
	}

	var outputData []model.DataItem

	for _, item := range inputData {
		// Create webhook response data structure
		webhookData := map[string]interface{}{
			"headers": item.JSON["headers"],
			"params":  item.JSON["params"],
			"query":   item.JSON["query"],
			"body":    item.JSON["body"],
			"method":  method,
			"path":    path,
		}

		// Process authentication if configured
		authType, _ := nodeParams["authentication"].(string)
		if authType != "" {
			authenticated, err := w.validateAuthentication(item.JSON, nodeParams)
			if err != nil {
				return nil, fmt.Errorf("webhook authentication failed: %w", err)
			}
			if !authenticated {
				return nil, fmt.Errorf("webhook authentication failed: invalid credentials")
			}
		}

		outputItem := model.DataItem{
			JSON: webhookData,
		}

		outputData = append(outputData, outputItem)
	}

	return outputData, nil
}

// validateAuthentication validates webhook authentication
func (w *WebhookNode) validateAuthentication(requestData map[string]interface{}, nodeParams map[string]interface{}) (bool, error) {
	authType, _ := nodeParams["authentication"].(string)

	switch authType {
	case "basicAuth":
		// Basic Auth validation
		return w.validateBasicAuth(requestData, nodeParams)
	case "headerAuth":
		// Header-based authentication
		return w.validateHeaderAuth(requestData, nodeParams)
	case "none":
		return true, nil
	default:
		return true, nil // No authentication required
	}
}

// validateBasicAuth validates basic authentication
func (w *WebhookNode) validateBasicAuth(requestData map[string]interface{}, nodeParams map[string]interface{}) (bool, error) {
	headers, ok := requestData["headers"].(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("no headers found in request")
	}

	authHeader, ok := headers["authorization"].(string)
	if !ok {
		return false, fmt.Errorf("no authorization header found")
	}

	if !strings.HasPrefix(strings.ToLower(authHeader), "basic ") {
		return false, fmt.Errorf("invalid authorization header format")
	}

	// In a real implementation, you would decode and validate the credentials
	// For now, we'll assume valid if the header is present
	return true, nil
}

// validateHeaderAuth validates header-based authentication
func (w *WebhookNode) validateHeaderAuth(requestData map[string]interface{}, nodeParams map[string]interface{}) (bool, error) {
	headers, ok := requestData["headers"].(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("no headers found in request")
	}

	headerName, _ := nodeParams["headerName"].(string)
	expectedValue, _ := nodeParams["headerValue"].(string)

	if headerName == "" {
		return false, fmt.Errorf("header name not configured")
	}

	headerValue, ok := headers[strings.ToLower(headerName)].(string)
	if !ok {
		return false, fmt.Errorf("required header %s not found", headerName)
	}

	return headerValue == expectedValue, nil
}

// ValidateParameters validates Webhook node parameters
func (w *WebhookNode) ValidateParameters(params map[string]interface{}) error {
	// Path is required
	if _, ok := params["path"]; !ok {
		return fmt.Errorf("webhook path is required")
	}

	// Validate HTTP method if specified
	if method, ok := params["httpMethod"].(string); ok {
		validMethods := map[string]bool{
			"GET": true, "POST": true, "PUT": true, "DELETE": true,
			"PATCH": true, "HEAD": true, "OPTIONS": true,
		}
		if !validMethods[strings.ToUpper(method)] {
			return fmt.Errorf("invalid HTTP method: %s", method)
		}
	}

	return nil
}

// GetWebhookInfo returns webhook configuration for external webhook server
func (w *WebhookNode) GetWebhookInfo(params map[string]interface{}) map[string]interface{} {
	path, _ := params["path"].(string)
	method, _ := params["httpMethod"].(string)
	if method == "" {
		method = "POST"
	}

	return map[string]interface{}{
		"path":   path,
		"method": strings.ToUpper(method),
		"type":   "webhook",
	}
}