package webhooks

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/neul-labs/m9m/internal/storage"
)

// WebhookStorage defines the interface for webhook persistence
type WebhookStorage interface {
	// Webhook CRUD
	SaveWebhook(webhook *Webhook) error
	GetWebhook(id string) (*Webhook, error)
	GetWebhookByPath(path string, method string, isTest bool) (*Webhook, error)
	ListWebhooks(workflowID string) ([]*Webhook, error)
	DeleteWebhook(id string) error

	// Webhook executions
	SaveWebhookExecution(execution *WebhookExecution) error
	GetWebhookExecution(id string) (*WebhookExecution, error)
	ListWebhookExecutions(webhookID string, limit int) ([]*WebhookExecution, error)
}

// MemoryWebhookStorage implements WebhookStorage using in-memory storage
type MemoryWebhookStorage struct {
	storage           storage.WorkflowStorage
	webhooks          map[string]*Webhook
	webhookExecutions map[string]*WebhookExecution
	pathIndex         map[string]*Webhook // path:method:test -> webhook
	mu                sync.RWMutex
}

// NewMemoryWebhookStorage creates a new in-memory webhook storage
func NewMemoryWebhookStorage(workflowStorage storage.WorkflowStorage) *MemoryWebhookStorage {
	return &MemoryWebhookStorage{
		storage:           workflowStorage,
		webhooks:          make(map[string]*Webhook),
		webhookExecutions: make(map[string]*WebhookExecution),
		pathIndex:         make(map[string]*Webhook),
	}
}

// SaveWebhook saves a webhook
func (s *MemoryWebhookStorage) SaveWebhook(webhook *Webhook) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if webhook.ID == "" {
		return errors.New("webhook ID cannot be empty")
	}

	now := time.Now()
	if webhook.CreatedAt.IsZero() {
		webhook.CreatedAt = now
	}
	webhook.UpdatedAt = now

	s.webhooks[webhook.ID] = webhook

	// Update path index
	key := s.makePathKey(webhook.Path, webhook.Method, webhook.IsTest)
	s.pathIndex[key] = webhook

	return nil
}

// GetWebhook retrieves a webhook by ID
func (s *MemoryWebhookStorage) GetWebhook(id string) (*Webhook, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	webhook, exists := s.webhooks[id]
	if !exists {
		return nil, fmt.Errorf("webhook not found: %s", id)
	}

	return webhook, nil
}

// GetWebhookByPath retrieves a webhook by path, method, and test flag
func (s *MemoryWebhookStorage) GetWebhookByPath(path string, method string, isTest bool) (*Webhook, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := s.makePathKey(path, method, isTest)
	webhook, exists := s.pathIndex[key]
	if !exists {
		return nil, fmt.Errorf("webhook not found for path: %s %s (test=%v)", method, path, isTest)
	}

	return webhook, nil
}

// ListWebhooks lists all webhooks for a workflow
func (s *MemoryWebhookStorage) ListWebhooks(workflowID string) ([]*Webhook, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var webhooks []*Webhook
	for _, webhook := range s.webhooks {
		if workflowID == "" || webhook.WorkflowID == workflowID {
			webhooks = append(webhooks, webhook)
		}
	}

	return webhooks, nil
}

// DeleteWebhook deletes a webhook
func (s *MemoryWebhookStorage) DeleteWebhook(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	webhook, exists := s.webhooks[id]
	if !exists {
		return fmt.Errorf("webhook not found: %s", id)
	}

	// Remove from path index
	key := s.makePathKey(webhook.Path, webhook.Method, webhook.IsTest)
	delete(s.pathIndex, key)

	// Remove webhook
	delete(s.webhooks, id)

	return nil
}

// SaveWebhookExecution saves a webhook execution record
func (s *MemoryWebhookStorage) SaveWebhookExecution(execution *WebhookExecution) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if execution.ID == "" {
		return errors.New("webhook execution ID cannot be empty")
	}

	if execution.CreatedAt.IsZero() {
		execution.CreatedAt = time.Now()
	}

	s.webhookExecutions[execution.ID] = execution

	return nil
}

// GetWebhookExecution retrieves a webhook execution
func (s *MemoryWebhookStorage) GetWebhookExecution(id string) (*WebhookExecution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	execution, exists := s.webhookExecutions[id]
	if !exists {
		return nil, fmt.Errorf("webhook execution not found: %s", id)
	}

	return execution, nil
}

// ListWebhookExecutions lists webhook executions
func (s *MemoryWebhookStorage) ListWebhookExecutions(webhookID string, limit int) ([]*WebhookExecution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var executions []*WebhookExecution
	for _, execution := range s.webhookExecutions {
		if webhookID == "" || execution.WebhookID == webhookID {
			executions = append(executions, execution)
			if limit > 0 && len(executions) >= limit {
				break
			}
		}
	}

	return executions, nil
}

// makePathKey creates a unique key for path indexing
func (s *MemoryWebhookStorage) makePathKey(path, method string, isTest bool) string {
	testSuffix := ""
	if isTest {
		testSuffix = ":test"
	}
	return fmt.Sprintf("%s:%s%s", method, path, testSuffix)
}

// PersistentWebhookStorage implements WebhookStorage using WorkflowStorage backend
type PersistentWebhookStorage struct {
	storage storage.WorkflowStorage
	mu      sync.RWMutex
}

// NewPersistentWebhookStorage creates a webhook storage backed by WorkflowStorage
func NewPersistentWebhookStorage(workflowStorage storage.WorkflowStorage) *PersistentWebhookStorage {
	return &PersistentWebhookStorage{
		storage: workflowStorage,
	}
}

// SaveWebhook saves a webhook to persistent storage
func (s *PersistentWebhookStorage) SaveWebhook(webhook *Webhook) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if webhook.ID == "" {
		return errors.New("webhook ID cannot be empty")
	}

	now := time.Now()
	if webhook.CreatedAt.IsZero() {
		webhook.CreatedAt = now
	}
	webhook.UpdatedAt = now

	// Serialize webhook
	data, err := json.Marshal(webhook)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook: %w", err)
	}

	// Store in workflow storage with webhook: prefix
	key := fmt.Sprintf("webhook:%s", webhook.ID)
	return s.storage.SaveRaw(key, data)
}

// GetWebhook retrieves a webhook from persistent storage
func (s *PersistentWebhookStorage) GetWebhook(id string) (*Webhook, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("webhook:%s", id)
	data, err := s.storage.GetRaw(key)
	if err != nil {
		return nil, fmt.Errorf("webhook not found: %s", id)
	}

	var webhook Webhook
	if err := json.Unmarshal(data, &webhook); err != nil {
		return nil, fmt.Errorf("failed to unmarshal webhook: %w", err)
	}

	return &webhook, nil
}

// GetWebhookByPath retrieves a webhook by path (requires scanning)
func (s *PersistentWebhookStorage) GetWebhookByPath(path string, method string, isTest bool) (*Webhook, error) {
	// Note: This is inefficient for large datasets - consider adding an index
	webhooks, err := s.ListWebhooks("")
	if err != nil {
		return nil, err
	}

	for _, webhook := range webhooks {
		if webhook.Path == path && webhook.Method == method && webhook.IsTest == isTest && webhook.Active {
			return webhook, nil
		}
	}

	return nil, fmt.Errorf("webhook not found for path: %s %s (test=%v)", method, path, isTest)
}

// ListWebhooks lists all webhooks
func (s *PersistentWebhookStorage) ListWebhooks(workflowID string) ([]*Webhook, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// List all keys with webhook: prefix
	keys, err := s.storage.ListKeys("webhook:")
	if err != nil {
		return nil, err
	}

	var webhooks []*Webhook
	for _, key := range keys {
		data, err := s.storage.GetRaw(key)
		if err != nil {
			continue
		}

		var webhook Webhook
		if err := json.Unmarshal(data, &webhook); err != nil {
			continue
		}

		if workflowID == "" || webhook.WorkflowID == workflowID {
			webhooks = append(webhooks, &webhook)
		}
	}

	return webhooks, nil
}

// DeleteWebhook deletes a webhook
func (s *PersistentWebhookStorage) DeleteWebhook(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("webhook:%s", id)
	return s.storage.DeleteRaw(key)
}

// SaveWebhookExecution saves a webhook execution
func (s *PersistentWebhookStorage) SaveWebhookExecution(execution *WebhookExecution) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if execution.ID == "" {
		return errors.New("webhook execution ID cannot be empty")
	}

	if execution.CreatedAt.IsZero() {
		execution.CreatedAt = time.Now()
	}

	data, err := json.Marshal(execution)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook execution: %w", err)
	}

	key := fmt.Sprintf("webhook_execution:%s", execution.ID)
	return s.storage.SaveRaw(key, data)
}

// GetWebhookExecution retrieves a webhook execution
func (s *PersistentWebhookStorage) GetWebhookExecution(id string) (*WebhookExecution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("webhook_execution:%s", id)
	data, err := s.storage.GetRaw(key)
	if err != nil {
		return nil, fmt.Errorf("webhook execution not found: %s", id)
	}

	var execution WebhookExecution
	if err := json.Unmarshal(data, &execution); err != nil {
		return nil, fmt.Errorf("failed to unmarshal webhook execution: %w", err)
	}

	return &execution, nil
}

// ListWebhookExecutions lists webhook executions
func (s *PersistentWebhookStorage) ListWebhookExecutions(webhookID string, limit int) ([]*WebhookExecution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys, err := s.storage.ListKeys("webhook_execution:")
	if err != nil {
		return nil, err
	}

	var executions []*WebhookExecution
	for _, key := range keys {
		data, err := s.storage.GetRaw(key)
		if err != nil {
			continue
		}

		var execution WebhookExecution
		if err := json.Unmarshal(data, &execution); err != nil {
			continue
		}

		if webhookID == "" || execution.WebhookID == webhookID {
			executions = append(executions, &execution)
			if limit > 0 && len(executions) >= limit {
				break
			}
		}
	}

	return executions, nil
}
