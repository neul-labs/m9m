package webhooks

import (
	"time"
)

// Webhook represents a webhook configuration
type Webhook struct {
	ID              string                 `json:"id"`
	WorkflowID      string                 `json:"workflowId"`
	NodeID          string                 `json:"nodeId"`
	Path            string                 `json:"path"`            // URL path (e.g., "/my-webhook")
	Method          string                 `json:"method"`          // GET, POST, PUT, DELETE, etc.
	IsTest          bool                   `json:"isTest"`          // Test vs production webhook
	Active          bool                   `json:"active"`          // Whether webhook is active
	AuthType        string                 `json:"authType"`        // none, basic, apiKey, header
	AuthData        map[string]interface{} `json:"authData"`        // Authentication configuration
	ResponseMode    string                 `json:"responseMode"`    // onReceived, lastNode, responseNode
	ResponseData    string                 `json:"responseData"`    // firstEntryJson, allEntries, noData
	ResponseHeaders map[string]string      `json:"responseHeaders"` // Custom response headers
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
}

// WebhookRequest represents an incoming webhook request
type WebhookRequest struct {
	Method  string              `json:"method"`
	Path    string              `json:"path"`
	Headers map[string][]string `json:"headers"`
	Query   map[string][]string `json:"query"`
	Body    interface{}         `json:"body"`
}

// WebhookResponse represents a webhook execution response
type WebhookResponse struct {
	StatusCode int                    `json:"statusCode"`
	Headers    map[string]string      `json:"headers"`
	Body       interface{}            `json:"body"`
	Data       []map[string]interface{} `json:"data,omitempty"`
}

// WebhookExecution tracks webhook execution
type WebhookExecution struct {
	ID          string        `json:"id"`
	WebhookID   string        `json:"webhookId"`
	ExecutionID string        `json:"executionId"`
	Status      string        `json:"status"` // success, failed, timeout
	Request     *WebhookRequest  `json:"request"`
	Response    *WebhookResponse `json:"response,omitempty"`
	Error       string        `json:"error,omitempty"`
	Duration    int64         `json:"duration"` // milliseconds
	CreatedAt   time.Time     `json:"createdAt"`
}
