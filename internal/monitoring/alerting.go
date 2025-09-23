package monitoring

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AlertManager manages alerts and alert rules
type AlertManager struct {
	mu           sync.RWMutex
	rules        map[string]*AlertRule
	activeAlerts map[string]*Alert
	alertHistory []Alert
	metrics      *MetricsCollector
	handlers     []AlertHandler
	maxHistory   int
	enabled      bool
}

// AlertHandler defines the interface for handling alerts
type AlertHandler interface {
	HandleAlert(alert Alert) error
	GetName() string
}

// AlertCondition represents different types of alert conditions
type AlertCondition struct {
	Type      string  `json:"type"`      // memory_usage, error_rate, execution_time, etc.
	Operator  string  `json:"operator"`  // >, <, >=, <=, ==
	Value     float64 `json:"value"`     // threshold value
	Duration  int     `json:"duration"`  // evaluation duration in seconds
	Aggregate string  `json:"aggregate"` // avg, max, min, sum, count
}

// AlertEvaluationContext provides context for alert evaluation
type AlertEvaluationContext struct {
	CurrentMetrics MetricsSummary
	RecentErrors   []ErrorMetric
	RecentPerf     []PerformanceMetric
	Timestamp      time.Time
}

// LogAlertHandler logs alerts to the console
type LogAlertHandler struct {
	name string
}

// EmailAlertHandler sends alerts via email (placeholder)
type EmailAlertHandler struct {
	name      string
	smtpHost  string
	smtpPort  int
	username  string
	password  string
	recipients []string
}

// WebhookAlertHandler sends alerts to a webhook
type WebhookAlertHandler struct {
	name string
	url  string
}

// NewAlertManager creates a new alert manager
func NewAlertManager(metrics *MetricsCollector) *AlertManager {
	return &AlertManager{
		rules:        make(map[string]*AlertRule),
		activeAlerts: make(map[string]*Alert),
		metrics:      metrics,
		handlers:     make([]AlertHandler, 0),
		maxHistory:   1000,
		enabled:      true,
	}
}

// AddRule adds a new alert rule
func (am *AlertManager) AddRule(rule *AlertRule) error {
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	am.rules[rule.ID] = rule
	return nil
}

// RemoveRule removes an alert rule
func (am *AlertManager) RemoveRule(ruleID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, exists := am.rules[ruleID]; !exists {
		return fmt.Errorf("rule with ID %s not found", ruleID)
	}

	delete(am.rules, ruleID)

	// Resolve any active alerts for this rule
	for alertID, alert := range am.activeAlerts {
		if alert.RuleID == ruleID {
			alert.Resolved = true
			now := time.Now()
			alert.ResolvedAt = &now
			am.moveToHistory(alert)
			delete(am.activeAlerts, alertID)
		}
	}

	return nil
}

// UpdateRule updates an existing alert rule
func (am *AlertManager) UpdateRule(rule *AlertRule) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, exists := am.rules[rule.ID]; !exists {
		return fmt.Errorf("rule with ID %s not found", rule.ID)
	}

	am.rules[rule.ID] = rule
	return nil
}

// GetRule retrieves an alert rule by ID
func (am *AlertManager) GetRule(ruleID string) (*AlertRule, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	rule, exists := am.rules[ruleID]
	if !exists {
		return nil, fmt.Errorf("rule with ID %s not found", ruleID)
	}

	return rule, nil
}

// GetAllRules returns all alert rules
func (am *AlertManager) GetAllRules() []*AlertRule {
	am.mu.RLock()
	defer am.mu.RUnlock()

	rules := make([]*AlertRule, 0, len(am.rules))
	for _, rule := range am.rules {
		rules = append(rules, rule)
	}
	return rules
}

// AddHandler adds an alert handler
func (am *AlertManager) AddHandler(handler AlertHandler) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.handlers = append(am.handlers, handler)
}

// RemoveHandler removes an alert handler by name
func (am *AlertManager) RemoveHandler(name string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	for i, handler := range am.handlers {
		if handler.GetName() == name {
			am.handlers = append(am.handlers[:i], am.handlers[i+1:]...)
			break
		}
	}
}

// EvaluateRules evaluates all active rules and triggers alerts if necessary
func (am *AlertManager) EvaluateRules() error {
	if !am.enabled {
		return nil
	}

	// Collect current metrics
	context := AlertEvaluationContext{
		CurrentMetrics: am.metrics.GetMetricsSummary(),
		RecentErrors:   am.metrics.GetErrorHistory(100),
		RecentPerf:     am.metrics.GetPerformanceHistory(100),
		Timestamp:      time.Now(),
	}

	am.mu.RLock()
	rules := make([]*AlertRule, 0, len(am.rules))
	for _, rule := range am.rules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}
	am.mu.RUnlock()

	// Evaluate each rule
	for _, rule := range rules {
		if err := am.evaluateRule(rule, context); err != nil {
			// Log evaluation error but continue with other rules
			fmt.Printf("Error evaluating rule %s: %v\n", rule.Name, err)
		}
	}

	return nil
}

// evaluateRule evaluates a single rule
func (am *AlertManager) evaluateRule(rule *AlertRule, context AlertEvaluationContext) error {
	triggered, value, err := am.checkCondition(rule, context)
	if err != nil {
		return fmt.Errorf("failed to check condition for rule %s: %w", rule.Name, err)
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	existingAlert, hasActiveAlert := am.activeAlerts[rule.ID]

	if triggered && !hasActiveAlert {
		// Create new alert
		alert := &Alert{
			ID:        uuid.New().String(),
			RuleID:    rule.ID,
			RuleName:  rule.Name,
			Severity:  rule.Severity,
			Message:   fmt.Sprintf("Alert: %s - Current value: %.2f, Threshold: %.2f", rule.Description, value, rule.Threshold),
			Value:     value,
			Threshold: rule.Threshold,
			Timestamp: context.Timestamp,
			Resolved:  false,
		}

		am.activeAlerts[rule.ID] = alert
		return am.sendAlert(*alert)

	} else if !triggered && hasActiveAlert {
		// Resolve existing alert
		existingAlert.Resolved = true
		now := context.Timestamp
		existingAlert.ResolvedAt = &now

		resolvedAlert := *existingAlert
		am.moveToHistory(existingAlert)
		delete(am.activeAlerts, rule.ID)

		return am.sendAlert(resolvedAlert)
	}

	return nil
}

// checkCondition checks if an alert condition is met
func (am *AlertManager) checkCondition(rule *AlertRule, context AlertEvaluationContext) (bool, float64, error) {
	var currentValue float64

	switch rule.Condition {
	case "memory_high":
		currentValue = float64(context.CurrentMetrics.CurrentMemoryUsage) / (1024 * 1024) // Convert to MB

	case "error_rate_high":
		currentValue = context.CurrentMetrics.HTTPErrorRate

	case "execution_time_high":
		currentValue = float64(context.CurrentMetrics.AverageExecutionTime.Milliseconds())

	case "success_rate_low":
		currentValue = context.CurrentMetrics.SuccessRate

	case "goroutines_high":
		currentValue = float64(context.CurrentMetrics.GoroutineCount)

	case "active_executions_high":
		currentValue = float64(context.CurrentMetrics.ActiveExecutions)

	case "response_time_high":
		currentValue = float64(context.CurrentMetrics.AverageResponseTime.Milliseconds())

	case "execution_failures":
		// Count failures in the last N minutes (rule.Duration)
		since := context.Timestamp.Add(-time.Duration(rule.Duration) * time.Second)
		failures := 0
		for _, exec := range am.metrics.GetExecutionHistory(100) {
			if exec.StartTime.After(since) && !exec.Success {
				failures++
			}
		}
		currentValue = float64(failures)

	default:
		return false, 0, fmt.Errorf("unknown condition: %s", rule.Condition)
	}

	triggered := currentValue >= rule.Threshold
	return triggered, currentValue, nil
}

// sendAlert sends an alert to all registered handlers
func (am *AlertManager) sendAlert(alert Alert) error {
	for _, handler := range am.handlers {
		if err := handler.HandleAlert(alert); err != nil {
			fmt.Printf("Error sending alert via %s: %v\n", handler.GetName(), err)
		}
	}
	return nil
}

// moveToHistory moves an alert to the history
func (am *AlertManager) moveToHistory(alert *Alert) {
	am.alertHistory = append(am.alertHistory, *alert)
	if len(am.alertHistory) > am.maxHistory {
		am.alertHistory = am.alertHistory[1:]
	}
}

// GetActiveAlerts returns all active alerts
func (am *AlertManager) GetActiveAlerts() []Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	alerts := make([]Alert, 0, len(am.activeAlerts))
	for _, alert := range am.activeAlerts {
		alerts = append(alerts, *alert)
	}
	return alerts
}

// GetAlertHistory returns alert history
func (am *AlertManager) GetAlertHistory(limit int) []Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if limit <= 0 || limit > len(am.alertHistory) {
		limit = len(am.alertHistory)
	}

	start := len(am.alertHistory) - limit
	if start < 0 {
		start = 0
	}

	result := make([]Alert, limit)
	copy(result, am.alertHistory[start:])
	return result
}

// ResolveAlert manually resolves an active alert
func (am *AlertManager) ResolveAlert(alertID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	for ruleID, alert := range am.activeAlerts {
		if alert.ID == alertID {
			alert.Resolved = true
			now := time.Now()
			alert.ResolvedAt = &now

			resolvedAlert := *alert
			am.moveToHistory(alert)
			delete(am.activeAlerts, ruleID)

			// Send resolution notification
			return am.sendAlert(resolvedAlert)
		}
	}

	return fmt.Errorf("active alert with ID %s not found", alertID)
}

// Enable enables the alert manager
func (am *AlertManager) Enable() {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.enabled = true
}

// Disable disables the alert manager
func (am *AlertManager) Disable() {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.enabled = false
}

// IsEnabled returns whether the alert manager is enabled
func (am *AlertManager) IsEnabled() bool {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.enabled
}

// Alert Handler Implementations

// NewLogAlertHandler creates a new log alert handler
func NewLogAlertHandler() *LogAlertHandler {
	return &LogAlertHandler{
		name: "log",
	}
}

func (h *LogAlertHandler) HandleAlert(alert Alert) error {
	status := "TRIGGERED"
	if alert.Resolved {
		status = "RESOLVED"
	}

	fmt.Printf("[ALERT %s] [%s] %s: %s (Value: %.2f, Threshold: %.2f) at %s\n",
		status, alert.Severity, alert.RuleName, alert.Message,
		alert.Value, alert.Threshold, alert.Timestamp.Format(time.RFC3339))

	return nil
}

func (h *LogAlertHandler) GetName() string {
	return h.name
}

// NewEmailAlertHandler creates a new email alert handler
func NewEmailAlertHandler(smtpHost string, smtpPort int, username, password string, recipients []string) *EmailAlertHandler {
	return &EmailAlertHandler{
		name:       "email",
		smtpHost:   smtpHost,
		smtpPort:   smtpPort,
		username:   username,
		password:   password,
		recipients: recipients,
	}
}

func (h *EmailAlertHandler) HandleAlert(alert Alert) error {
	// Placeholder for email implementation
	// In a real implementation, this would send an email using SMTP
	fmt.Printf("EMAIL ALERT to %v: %s\n", h.recipients, alert.Message)
	return nil
}

func (h *EmailAlertHandler) GetName() string {
	return h.name
}

// NewWebhookAlertHandler creates a new webhook alert handler
func NewWebhookAlertHandler(url string) *WebhookAlertHandler {
	return &WebhookAlertHandler{
		name: "webhook",
		url:  url,
	}
}

func (h *WebhookAlertHandler) HandleAlert(alert Alert) error {
	// Placeholder for webhook implementation
	// In a real implementation, this would send an HTTP POST to the webhook URL
	fmt.Printf("WEBHOOK ALERT to %s: %s\n", h.url, alert.Message)
	return nil
}

func (h *WebhookAlertHandler) GetName() string {
	return h.name
}

// Predefined Alert Rules

// CreateDefaultAlertRules creates a set of default alert rules
func CreateDefaultAlertRules() []*AlertRule {
	return []*AlertRule{
		{
			ID:          "high-memory-usage",
			Name:        "High Memory Usage",
			Condition:   "memory_high",
			Threshold:   500, // 500 MB
			Duration:    300, // 5 minutes
			Severity:    "high",
			Enabled:     true,
			Description: "Memory usage is above 500MB",
		},
		{
			ID:          "high-error-rate",
			Name:        "High HTTP Error Rate",
			Condition:   "error_rate_high",
			Threshold:   10, // 10%
			Duration:    300,
			Severity:    "medium",
			Enabled:     true,
			Description: "HTTP error rate is above 10%",
		},
		{
			ID:          "low-success-rate",
			Name:        "Low Workflow Success Rate",
			Condition:   "success_rate_low",
			Threshold:   90, // Below 90%
			Duration:    600,
			Severity:    "high",
			Enabled:     true,
			Description: "Workflow success rate is below 90%",
		},
		{
			ID:          "high-execution-time",
			Name:        "High Average Execution Time",
			Condition:   "execution_time_high",
			Threshold:   10000, // 10 seconds
			Duration:    300,
			Severity:    "medium",
			Enabled:     true,
			Description: "Average execution time is above 10 seconds",
		},
		{
			ID:          "too-many-goroutines",
			Name:        "Too Many Goroutines",
			Condition:   "goroutines_high",
			Threshold:   1000,
			Duration:    300,
			Severity:    "medium",
			Enabled:     true,
			Description: "Number of goroutines is above 1000",
		},
		{
			ID:          "high-response-time",
			Name:        "High API Response Time",
			Condition:   "response_time_high",
			Threshold:   2000, // 2 seconds
			Duration:    300,
			Severity:    "medium",
			Enabled:     true,
			Description: "Average API response time is above 2 seconds",
		},
	}
}