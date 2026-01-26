package webhooks

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/neul-labs/m9m/internal/engine"
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/storage"
)

// WebhookManager manages webhook registration and execution
type WebhookManager struct {
	storage         WebhookStorage
	workflowStorage storage.WorkflowStorage
	engine          engine.WorkflowEngine
	activeHooks     map[string]*Webhook // path:method:test -> webhook
	mu              sync.RWMutex
}

// NewWebhookManager creates a new webhook manager
func NewWebhookManager(webhookStorage WebhookStorage, workflowStorage storage.WorkflowStorage, engine engine.WorkflowEngine) *WebhookManager {
	return &WebhookManager{
		storage:         webhookStorage,
		workflowStorage: workflowStorage,
		engine:          engine,
		activeHooks:     make(map[string]*Webhook),
	}
}

// RegisterWebhook registers a webhook for a workflow node
func (m *WebhookManager) RegisterWebhook(webhook *Webhook) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate webhook
	if err := m.validateWebhook(webhook); err != nil {
		return fmt.Errorf("invalid webhook: %w", err)
	}

	// Generate ID if not provided
	if webhook.ID == "" {
		webhook.ID = generateWebhookID()
	}

	// Normalize path (ensure leading slash)
	webhook.Path = normalizePath(webhook.Path)

	// Set defaults
	if webhook.Method == "" {
		webhook.Method = "POST"
	}
	if webhook.ResponseMode == "" {
		webhook.ResponseMode = "onReceived"
	}
	if webhook.ResponseData == "" {
		webhook.ResponseData = "firstEntryJson"
	}

	// Save to storage
	if err := m.storage.SaveWebhook(webhook); err != nil {
		return fmt.Errorf("failed to save webhook: %w", err)
	}

	// Add to active hooks if active
	if webhook.Active {
		key := makeWebhookKey(webhook.Path, webhook.Method, webhook.IsTest)
		m.activeHooks[key] = webhook
		log.Printf("✅ Webhook registered: %s %s (workflow=%s, test=%v)",
			webhook.Method, webhook.Path, webhook.WorkflowID, webhook.IsTest)
	}

	return nil
}

// UnregisterWebhook removes a webhook
func (m *WebhookManager) UnregisterWebhook(webhookID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	webhook, err := m.storage.GetWebhook(webhookID)
	if err != nil {
		return err
	}

	// Remove from active hooks
	key := makeWebhookKey(webhook.Path, webhook.Method, webhook.IsTest)
	delete(m.activeHooks, key)

	// Delete from storage
	if err := m.storage.DeleteWebhook(webhookID); err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	log.Printf("❌ Webhook unregistered: %s %s", webhook.Method, webhook.Path)
	return nil
}

// GetWebhook retrieves a webhook by ID
func (m *WebhookManager) GetWebhook(webhookID string) (*Webhook, error) {
	return m.storage.GetWebhook(webhookID)
}

// GetWebhookByPath retrieves a webhook by path and method
func (m *WebhookManager) GetWebhookByPath(path string, method string, isTest bool) (*Webhook, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	path = normalizePath(path)
	key := makeWebhookKey(path, method, isTest)

	webhook, exists := m.activeHooks[key]
	if !exists {
		return nil, fmt.Errorf("webhook not found: %s %s (test=%v)", method, path, isTest)
	}

	return webhook, nil
}

// ListWebhooks lists all webhooks for a workflow
func (m *WebhookManager) ListWebhooks(workflowID string) ([]*Webhook, error) {
	return m.storage.ListWebhooks(workflowID)
}

// RegisterWorkflowWebhooks registers all webhooks for a workflow
func (m *WebhookManager) RegisterWorkflowWebhooks(workflow *model.Workflow, isTest bool) error {
	// Find all webhook nodes in the workflow
	for _, node := range workflow.Nodes {
		if isWebhookNode(node.Type) {
			webhook := m.createWebhookFromNode(workflow, &node, isTest)
			if err := m.RegisterWebhook(webhook); err != nil {
				log.Printf("⚠️  Failed to register webhook for node %s: %v", node.Name, err)
			}
		}
	}

	return nil
}

// UnregisterWorkflowWebhooks removes all webhooks for a workflow
func (m *WebhookManager) UnregisterWorkflowWebhooks(workflowID string) error {
	webhooks, err := m.storage.ListWebhooks(workflowID)
	if err != nil {
		return err
	}

	for _, webhook := range webhooks {
		if err := m.UnregisterWebhook(webhook.ID); err != nil {
			log.Printf("⚠️  Failed to unregister webhook %s: %v", webhook.ID, err)
		}
	}

	return nil
}

// ExecuteWebhook executes a webhook and returns the response
func (m *WebhookManager) ExecuteWebhook(webhook *Webhook, request *WebhookRequest) (*WebhookResponse, error) {
	startTime := time.Now()

	// Get workflow
	workflow, err := m.workflowStorage.GetWorkflow(webhook.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %s", webhook.WorkflowID)
	}

	// Prepare execution input from webhook request
	inputData := m.prepareInputData(request)

	// Execute workflow
	executionID := generateExecutionID()
	result, err := m.engine.ExecuteWorkflow(workflow, inputData)

	// Create execution record
	execution := &WebhookExecution{
		ID:          generateWebhookExecutionID(),
		WebhookID:   webhook.ID,
		ExecutionID: executionID,
		Request:     request,
		CreatedAt:   time.Now(),
		Duration:    time.Since(startTime).Milliseconds(),
	}

	if err != nil {
		execution.Status = "failed"
		execution.Error = err.Error()
		m.storage.SaveWebhookExecution(execution)
		return nil, fmt.Errorf("workflow execution failed: %w", err)
	}

	execution.Status = "success"

	// Prepare response based on webhook configuration
	response := m.prepareResponse(webhook, result)
	execution.Response = response

	// Save execution record
	if err := m.storage.SaveWebhookExecution(execution); err != nil {
		log.Printf("⚠️  Failed to save webhook execution: %v", err)
	}

	return response, nil
}

// LoadActiveWebhooks loads all active webhooks into memory
func (m *WebhookManager) LoadActiveWebhooks() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	webhooks, err := m.storage.ListWebhooks("")
	if err != nil {
		return fmt.Errorf("failed to load webhooks: %w", err)
	}

	count := 0
	for _, webhook := range webhooks {
		if webhook.Active {
			key := makeWebhookKey(webhook.Path, webhook.Method, webhook.IsTest)
			m.activeHooks[key] = webhook
			count++
		}
	}

	log.Printf("📡 Loaded %d active webhooks", count)
	return nil
}

// Helper methods

func (m *WebhookManager) validateWebhook(webhook *Webhook) error {
	if webhook.WorkflowID == "" {
		return fmt.Errorf("workflow ID is required")
	}
	if webhook.Path == "" {
		return fmt.Errorf("path is required")
	}
	return nil
}

func (m *WebhookManager) createWebhookFromNode(workflow *model.Workflow, node *model.Node, isTest bool) *Webhook {
	// Extract webhook configuration from node parameters
	path := getStringParam(node.Parameters, "path", "")
	method := strings.ToUpper(getStringParam(node.Parameters, "httpMethod", "POST"))
	authType := getStringParam(node.Parameters, "authentication", "none")
	responseMode := getStringParam(node.Parameters, "responseMode", "onReceived")

	return &Webhook{
		WorkflowID:   workflow.ID,
		NodeID:       node.Name,
		Path:         normalizePath(path),
		Method:       method,
		IsTest:       isTest,
		Active:       workflow.Active && !isTest,
		AuthType:     authType,
		ResponseMode: responseMode,
		ResponseData: "firstEntryJson",
	}
}

func (m *WebhookManager) prepareInputData(request *WebhookRequest) []model.DataItem {
	data := map[string]interface{}{
		"headers": request.Headers,
		"params":  request.Query,
		"query":   request.Query,
		"body":    request.Body,
		"method":  request.Method,
		"path":    request.Path,
	}

	return []model.DataItem{
		{
			JSON: data,
		},
	}
}

func (m *WebhookManager) prepareResponse(webhook *Webhook, result *engine.ExecutionResult) *WebhookResponse {
	response := &WebhookResponse{
		StatusCode: 200,
		Headers:    webhook.ResponseHeaders,
	}

	if response.Headers == nil {
		response.Headers = make(map[string]string)
	}
	response.Headers["Content-Type"] = "application/json"

	// Based on response mode
	switch webhook.ResponseData {
	case "firstEntryJson":
		if len(result.Data) > 0 {
			response.Body = result.Data[0].JSON
		} else {
			response.Body = map[string]interface{}{"message": "success"}
		}
	case "allEntries":
		entries := make([]map[string]interface{}, len(result.Data))
		for i, item := range result.Data {
			entries[i] = item.JSON
		}
		response.Body = entries
	case "noData":
		response.Body = map[string]interface{}{"message": "success"}
	default:
		if len(result.Data) > 0 {
			response.Body = result.Data[0].JSON
		}
	}

	return response
}

// Helper functions

func isWebhookNode(nodeType string) bool {
	return nodeType == "n8n-nodes-base.webhook" ||
		strings.Contains(strings.ToLower(nodeType), "webhook")
}

func normalizePath(path string) string {
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func makeWebhookKey(path, method string, isTest bool) string {
	testSuffix := ""
	if isTest {
		testSuffix = ":test"
	}
	return fmt.Sprintf("%s:%s%s", strings.ToUpper(method), path, testSuffix)
}

func getStringParam(params map[string]interface{}, key, defaultValue string) string {
	if val, ok := params[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

func generateWebhookID() string {
	return fmt.Sprintf("webhook_%d", time.Now().UnixNano())
}

func generateExecutionID() string {
	return fmt.Sprintf("exec_%d", time.Now().UnixNano())
}

func generateWebhookExecutionID() string {
	return fmt.Sprintf("wh_exec_%d", time.Now().UnixNano())
}
