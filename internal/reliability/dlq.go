// Package reliability provides reliability features for workflow execution
package reliability

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// DeadLetterItem represents an item in the dead letter queue
type DeadLetterItem struct {
	ID            string                 `json:"id"`
	WorkflowID    string                 `json:"workflowId"`
	ExecutionID   string                 `json:"executionId"`
	NodeName      string                 `json:"nodeName,omitempty"`
	Error         string                 `json:"error"`
	ErrorType     string                 `json:"errorType"`
	InputData     interface{}            `json:"inputData,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	RetryCount    int                    `json:"retryCount"`
	MaxRetries    int                    `json:"maxRetries"`
	FirstFailedAt time.Time              `json:"firstFailedAt"`
	LastFailedAt  time.Time              `json:"lastFailedAt"`
	Status        DLQStatus              `json:"status"`
}

// DLQStatus represents the status of a DLQ item
type DLQStatus string

const (
	DLQStatusPending    DLQStatus = "pending"
	DLQStatusRetrying   DLQStatus = "retrying"
	DLQStatusResolved   DLQStatus = "resolved"
	DLQStatusDiscarded  DLQStatus = "discarded"
	DLQStatusManualHold DLQStatus = "manual_hold"
)

// DLQConfig configures the dead letter queue
type DLQConfig struct {
	MaxSize         int
	RetentionPeriod time.Duration
	AutoRetry       bool
	RetryInterval   time.Duration
	MaxAutoRetries  int
	OnItemAdded     func(item *DeadLetterItem)
	OnItemResolved  func(item *DeadLetterItem)
}

// DefaultDLQConfig returns default configuration
func DefaultDLQConfig() *DLQConfig {
	return &DLQConfig{
		MaxSize:         10000,
		RetentionPeriod: 7 * 24 * time.Hour, // 7 days
		AutoRetry:       false,
		RetryInterval:   5 * time.Minute,
		MaxAutoRetries:  3,
	}
}

// DeadLetterQueue manages failed executions
type DeadLetterQueue struct {
	config   *DLQConfig
	items    map[string]*DeadLetterItem
	order    []string // Maintains insertion order
	mu       sync.RWMutex
	stopCh   chan struct{}
}

// NewDeadLetterQueue creates a new dead letter queue
func NewDeadLetterQueue(config *DLQConfig) *DeadLetterQueue {
	if config == nil {
		config = DefaultDLQConfig()
	}

	dlq := &DeadLetterQueue{
		config: config,
		items:  make(map[string]*DeadLetterItem),
		order:  make([]string, 0),
		stopCh: make(chan struct{}),
	}

	// Start cleanup goroutine
	go dlq.cleanupLoop()

	return dlq
}

// Add adds an item to the dead letter queue
func (dlq *DeadLetterQueue) Add(item *DeadLetterItem) error {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	// Check size limit
	if len(dlq.items) >= dlq.config.MaxSize {
		// Remove oldest item
		if len(dlq.order) > 0 {
			oldestID := dlq.order[0]
			delete(dlq.items, oldestID)
			dlq.order = dlq.order[1:]
		}
	}

	// Set defaults
	if item.ID == "" {
		item.ID = fmt.Sprintf("dlq-%d", time.Now().UnixNano())
	}
	if item.FirstFailedAt.IsZero() {
		item.FirstFailedAt = time.Now()
	}
	item.LastFailedAt = time.Now()
	if item.Status == "" {
		item.Status = DLQStatusPending
	}

	dlq.items[item.ID] = item
	dlq.order = append(dlq.order, item.ID)

	// Notify callback
	if dlq.config.OnItemAdded != nil {
		go dlq.config.OnItemAdded(item)
	}

	return nil
}

// Get retrieves an item by ID
func (dlq *DeadLetterQueue) Get(id string) (*DeadLetterItem, bool) {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	item, ok := dlq.items[id]
	return item, ok
}

// List returns all items with optional filtering
func (dlq *DeadLetterQueue) List(filter *DLQFilter) []*DeadLetterItem {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	result := make([]*DeadLetterItem, 0)

	for _, id := range dlq.order {
		item := dlq.items[id]
		if filter == nil || filter.Matches(item) {
			result = append(result, item)
		}
	}

	return result
}

// DLQFilter defines filtering criteria
type DLQFilter struct {
	WorkflowID string
	Status     DLQStatus
	ErrorType  string
	Since      time.Time
	Limit      int
}

// Matches checks if an item matches the filter
func (f *DLQFilter) Matches(item *DeadLetterItem) bool {
	if f.WorkflowID != "" && item.WorkflowID != f.WorkflowID {
		return false
	}
	if f.Status != "" && item.Status != f.Status {
		return false
	}
	if f.ErrorType != "" && item.ErrorType != f.ErrorType {
		return false
	}
	if !f.Since.IsZero() && item.LastFailedAt.Before(f.Since) {
		return false
	}
	return true
}

// Retry marks an item for retry
func (dlq *DeadLetterQueue) Retry(id string) error {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	item, ok := dlq.items[id]
	if !ok {
		return fmt.Errorf("item not found: %s", id)
	}

	item.Status = DLQStatusRetrying
	item.RetryCount++

	return nil
}

// Resolve marks an item as resolved
func (dlq *DeadLetterQueue) Resolve(id string) error {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	item, ok := dlq.items[id]
	if !ok {
		return fmt.Errorf("item not found: %s", id)
	}

	item.Status = DLQStatusResolved

	// Notify callback
	if dlq.config.OnItemResolved != nil {
		go dlq.config.OnItemResolved(item)
	}

	return nil
}

// Discard marks an item as discarded
func (dlq *DeadLetterQueue) Discard(id string) error {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	item, ok := dlq.items[id]
	if !ok {
		return fmt.Errorf("item not found: %s", id)
	}

	item.Status = DLQStatusDiscarded
	return nil
}

// Hold puts an item on manual hold
func (dlq *DeadLetterQueue) Hold(id string) error {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	item, ok := dlq.items[id]
	if !ok {
		return fmt.Errorf("item not found: %s", id)
	}

	item.Status = DLQStatusManualHold
	return nil
}

// Delete removes an item from the queue
func (dlq *DeadLetterQueue) Delete(id string) error {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	if _, ok := dlq.items[id]; !ok {
		return fmt.Errorf("item not found: %s", id)
	}

	delete(dlq.items, id)

	// Remove from order slice
	for i, itemID := range dlq.order {
		if itemID == id {
			dlq.order = append(dlq.order[:i], dlq.order[i+1:]...)
			break
		}
	}

	return nil
}

// Stats returns DLQ statistics
func (dlq *DeadLetterQueue) Stats() map[string]interface{} {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	statusCounts := make(map[string]int)
	for _, item := range dlq.items {
		statusCounts[string(item.Status)]++
	}

	return map[string]interface{}{
		"totalItems":   len(dlq.items),
		"maxSize":      dlq.config.MaxSize,
		"statusCounts": statusCounts,
	}
}

// GetPendingForRetry returns items ready for retry
func (dlq *DeadLetterQueue) GetPendingForRetry() []*DeadLetterItem {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	result := make([]*DeadLetterItem, 0)

	for _, item := range dlq.items {
		if item.Status == DLQStatusPending && item.RetryCount < item.MaxRetries {
			result = append(result, item)
		}
	}

	return result
}

// cleanupLoop periodically cleans up old items
func (dlq *DeadLetterQueue) cleanupLoop() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			dlq.cleanup()
		case <-dlq.stopCh:
			return
		}
	}
}

func (dlq *DeadLetterQueue) cleanup() {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	cutoff := time.Now().Add(-dlq.config.RetentionPeriod)
	toDelete := make([]string, 0)

	for id, item := range dlq.items {
		// Remove resolved/discarded items older than retention period
		if (item.Status == DLQStatusResolved || item.Status == DLQStatusDiscarded) &&
			item.LastFailedAt.Before(cutoff) {
			toDelete = append(toDelete, id)
		}
	}

	for _, id := range toDelete {
		delete(dlq.items, id)
	}

	// Rebuild order slice
	newOrder := make([]string, 0, len(dlq.items))
	for _, id := range dlq.order {
		if _, ok := dlq.items[id]; ok {
			newOrder = append(newOrder, id)
		}
	}
	dlq.order = newOrder
}

// Stop stops the DLQ background processes
func (dlq *DeadLetterQueue) Stop() {
	close(dlq.stopCh)
}

// Export exports DLQ items to JSON
func (dlq *DeadLetterQueue) Export() ([]byte, error) {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	items := make([]*DeadLetterItem, 0, len(dlq.items))
	for _, id := range dlq.order {
		items = append(items, dlq.items[id])
	}

	return json.Marshal(items)
}

// Import imports DLQ items from JSON
func (dlq *DeadLetterQueue) Import(data []byte) error {
	var items []*DeadLetterItem
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}

	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	for _, item := range items {
		dlq.items[item.ID] = item
		dlq.order = append(dlq.order, item.ID)
	}

	return nil
}

// HealthChecker provides health checking functionality
type HealthChecker struct {
	checks   map[string]HealthCheck
	mu       sync.RWMutex
	interval time.Duration
	stopCh   chan struct{}
	status   map[string]*HealthStatus
}

// HealthCheck is a function that performs a health check
type HealthCheck func(ctx context.Context) error

// HealthStatus represents the health status of a component
type HealthStatus struct {
	Name        string    `json:"name"`
	Healthy     bool      `json:"healthy"`
	LastCheck   time.Time `json:"lastCheck"`
	LastError   string    `json:"lastError,omitempty"`
	Consecutive int       `json:"consecutiveFailures"`
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(interval time.Duration) *HealthChecker {
	hc := &HealthChecker{
		checks:   make(map[string]HealthCheck),
		interval: interval,
		stopCh:   make(chan struct{}),
		status:   make(map[string]*HealthStatus),
	}

	go hc.runChecks()

	return hc
}

// Register registers a health check
func (hc *HealthChecker) Register(name string, check HealthCheck) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.checks[name] = check
	hc.status[name] = &HealthStatus{Name: name, Healthy: true}
}

// Unregister removes a health check
func (hc *HealthChecker) Unregister(name string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	delete(hc.checks, name)
	delete(hc.status, name)
}

// Status returns the health status of all checks
func (hc *HealthChecker) Status() map[string]*HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	result := make(map[string]*HealthStatus)
	for k, v := range hc.status {
		result[k] = v
	}
	return result
}

// IsHealthy returns true if all checks are healthy
func (hc *HealthChecker) IsHealthy() bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	for _, status := range hc.status {
		if !status.Healthy {
			return false
		}
	}
	return true
}

func (hc *HealthChecker) runChecks() {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hc.checkAll()
		case <-hc.stopCh:
			return
		}
	}
}

func (hc *HealthChecker) checkAll() {
	hc.mu.RLock()
	checks := make(map[string]HealthCheck)
	for k, v := range hc.checks {
		checks[k] = v
	}
	hc.mu.RUnlock()

	for name, check := range checks {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := check(ctx)
		cancel()

		hc.mu.Lock()
		status := hc.status[name]
		if status == nil {
			status = &HealthStatus{Name: name}
			hc.status[name] = status
		}

		status.LastCheck = time.Now()
		if err != nil {
			status.Healthy = false
			status.LastError = err.Error()
			status.Consecutive++
		} else {
			status.Healthy = true
			status.LastError = ""
			status.Consecutive = 0
		}
		hc.mu.Unlock()
	}
}

// Stop stops the health checker
func (hc *HealthChecker) Stop() {
	close(hc.stopCh)
}

// CheckNow runs all health checks immediately
func (hc *HealthChecker) CheckNow() {
	hc.checkAll()
}
