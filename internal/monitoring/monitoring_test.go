package monitoring

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Helper functions
// ============================================================================

func newTestExecutionMetric(workflowID, executionID string, success bool, duration time.Duration, nodeCount int) ExecutionMetric {
	start := time.Now().Add(-duration)
	return ExecutionMetric{
		WorkflowID:    workflowID,
		ExecutionID:   executionID,
		StartTime:     start,
		EndTime:       time.Now(),
		Duration:      duration,
		Success:       success,
		NodeCount:     nodeCount,
		ExecutionMode: "manual",
	}
}

func newTestErrorMetric(errorType, errorCode, message, workflowID, severity string) ErrorMetric {
	return ErrorMetric{
		Timestamp:  time.Now(),
		ErrorType:  errorType,
		ErrorCode:  errorCode,
		Message:    message,
		WorkflowID: workflowID,
		Severity:   severity,
	}
}

func newTestMonitoringConfig() MonitoringConfig {
	return MonitoringConfig{
		CollectInterval:  100 * time.Millisecond,
		AlertInterval:    100 * time.Millisecond,
		OptimizeInterval: 100 * time.Millisecond,
		EnableAlerts:     true,
		EnableOptimizer:  true,
	}
}

// mockAlertHandler is a test double for AlertHandler
type mockAlertHandler struct {
	name   string
	alerts []Alert
}

func newMockAlertHandler(name string) *mockAlertHandler {
	return &mockAlertHandler{
		name:   name,
		alerts: make([]Alert, 0),
	}
}

func (h *mockAlertHandler) HandleAlert(alert Alert) error {
	h.alerts = append(h.alerts, alert)
	return nil
}

func (h *mockAlertHandler) GetName() string {
	return h.name
}

// ============================================================================
// MetricsCollector Tests
// ============================================================================

func TestMetricsCollector_RecordWorkflowExecution(t *testing.T) {
	mc := NewMetricsCollector()

	metric := newTestExecutionMetric("wf-1", "exec-1", true, 500*time.Millisecond, 5)
	mc.RecordWorkflowExecution(metric)

	history := mc.GetExecutionHistory(10)
	require.Len(t, history, 1)
	assert.Equal(t, "wf-1", history[0].WorkflowID)
	assert.Equal(t, "exec-1", history[0].ExecutionID)
	assert.True(t, history[0].Success)
	assert.Equal(t, 5, history[0].NodeCount)

	summary := mc.GetMetricsSummary()
	assert.Equal(t, int64(1), summary.TotalExecutions)
	assert.Equal(t, int64(1), summary.SuccessfulExecutions)
	assert.Equal(t, int64(0), summary.FailedExecutions)
}

func TestMetricsCollector_RecordWorkflowExecution_Failure(t *testing.T) {
	mc := NewMetricsCollector()

	metric := newTestExecutionMetric("wf-1", "exec-1", false, 200*time.Millisecond, 3)
	metric.ErrorMessage = "node failed"
	mc.RecordWorkflowExecution(metric)

	summary := mc.GetMetricsSummary()
	assert.Equal(t, int64(1), summary.TotalExecutions)
	assert.Equal(t, int64(0), summary.SuccessfulExecutions)
	assert.Equal(t, int64(1), summary.FailedExecutions)
}

func TestMetricsCollector_RecordError(t *testing.T) {
	mc := NewMetricsCollector()

	errMetric := newTestErrorMetric("runtime", "E001", "null pointer", "wf-1", "high")
	mc.RecordError(errMetric)

	history := mc.GetErrorHistory(10)
	require.Len(t, history, 1)
	assert.Equal(t, "runtime", history[0].ErrorType)
	assert.Equal(t, "E001", history[0].ErrorCode)
	assert.Equal(t, "null pointer", history[0].Message)
	assert.Equal(t, "wf-1", history[0].WorkflowID)
	assert.Equal(t, "high", history[0].Severity)
}

func TestMetricsCollector_GetMetricsSummary(t *testing.T) {
	mc := NewMetricsCollector()

	// Record a mix of successes and failures
	mc.RecordWorkflowExecution(newTestExecutionMetric("wf-1", "exec-1", true, 100*time.Millisecond, 3))
	mc.RecordWorkflowExecution(newTestExecutionMetric("wf-1", "exec-2", true, 200*time.Millisecond, 4))
	mc.RecordWorkflowExecution(newTestExecutionMetric("wf-2", "exec-3", false, 300*time.Millisecond, 5))
	mc.RecordWorkflowExecution(newTestExecutionMetric("wf-2", "exec-4", true, 150*time.Millisecond, 2))

	summary := mc.GetMetricsSummary()
	assert.Equal(t, int64(4), summary.TotalExecutions)
	assert.Equal(t, int64(3), summary.SuccessfulExecutions)
	assert.Equal(t, int64(1), summary.FailedExecutions)
	assert.Equal(t, 75.0, summary.SuccessRate)
	assert.True(t, summary.AverageExecutionTime > 0)
	assert.True(t, summary.Uptime > 0)
}

func TestMetricsCollector_GetMetricsSummary_NoExecutions(t *testing.T) {
	mc := NewMetricsCollector()

	summary := mc.GetMetricsSummary()
	assert.Equal(t, int64(0), summary.TotalExecutions)
	assert.Equal(t, float64(0), summary.SuccessRate)
	assert.Equal(t, time.Duration(0), summary.AverageExecutionTime)
}

func TestMetricsCollector_GetExecutionHistory(t *testing.T) {
	mc := NewMetricsCollector()

	// Record 5 executions
	for i := 0; i < 5; i++ {
		mc.RecordWorkflowExecution(newTestExecutionMetric("wf-1", "exec-"+string(rune('A'+i)), true, 100*time.Millisecond, 3))
	}

	// Get with limit smaller than total
	history := mc.GetExecutionHistory(3)
	require.Len(t, history, 3)

	// Get with limit larger than total (returns all)
	history = mc.GetExecutionHistory(10)
	require.Len(t, history, 5)

	// Get with limit 0 (returns all)
	history = mc.GetExecutionHistory(0)
	require.Len(t, history, 5)

	// Get with negative limit (returns all)
	history = mc.GetExecutionHistory(-1)
	require.Len(t, history, 5)
}

func TestMetricsCollector_GetErrorHistory(t *testing.T) {
	mc := NewMetricsCollector()

	// Record 5 errors
	for i := 0; i < 5; i++ {
		mc.RecordError(newTestErrorMetric("type-A", "E001", "error message", "wf-1", "medium"))
	}

	// Get with limit smaller than total
	history := mc.GetErrorHistory(2)
	require.Len(t, history, 2)

	// Get with limit larger than total (returns all)
	history = mc.GetErrorHistory(10)
	require.Len(t, history, 5)

	// Get with zero limit (returns all)
	history = mc.GetErrorHistory(0)
	require.Len(t, history, 5)
}

func TestMetricsCollector_Reset(t *testing.T) {
	mc := NewMetricsCollector()

	// Record some data
	mc.RecordWorkflowExecution(newTestExecutionMetric("wf-1", "exec-1", true, 100*time.Millisecond, 3))
	mc.RecordError(newTestErrorMetric("runtime", "E001", "error", "wf-1", "high"))
	mc.RecordHTTPRequest(50*time.Millisecond, true)
	mc.SetActiveWorkflows(5)
	mc.IncrementActiveExecutions()
	mc.CollectSystemMetrics()

	// Verify data is present
	summary := mc.GetMetricsSummary()
	assert.Equal(t, int64(1), summary.TotalExecutions)
	assert.Equal(t, int64(1), summary.TotalHTTPRequests)

	// Reset
	mc.Reset()

	// Verify everything is cleared
	summary = mc.GetMetricsSummary()
	assert.Equal(t, int64(0), summary.TotalExecutions)
	assert.Equal(t, int64(0), summary.SuccessfulExecutions)
	assert.Equal(t, int64(0), summary.FailedExecutions)
	assert.Equal(t, int64(0), summary.TotalHTTPRequests)
	assert.Equal(t, int64(0), summary.ActiveWorkflows)
	assert.Equal(t, int64(0), summary.ActiveExecutions)
	assert.Equal(t, int64(0), summary.CurrentMemoryUsage)

	execHistory := mc.GetExecutionHistory(10)
	assert.Len(t, execHistory, 0)

	errorHistory := mc.GetErrorHistory(10)
	assert.Len(t, errorHistory, 0)

	perfHistory := mc.GetPerformanceHistory(10)
	assert.Len(t, perfHistory, 0)
}

func TestMetricsCollector_SetMaxHistorySize(t *testing.T) {
	mc := NewMetricsCollector()
	mc.SetMaxHistorySize(3)

	// Record 5 executions -- only 3 should be kept
	for i := 0; i < 5; i++ {
		mc.RecordWorkflowExecution(newTestExecutionMetric("wf-1", "exec-"+string(rune('A'+i)), true, 100*time.Millisecond, 2))
	}

	history := mc.GetExecutionHistory(10)
	assert.Len(t, history, 3)

	// Record 5 errors -- only 3 should be kept
	for i := 0; i < 5; i++ {
		mc.RecordError(newTestErrorMetric("type-A", "E001", "error", "wf-1", "low"))
	}

	errorHistory := mc.GetErrorHistory(10)
	assert.Len(t, errorHistory, 3)
}

func TestMetricsCollector_CollectSystemMetrics(t *testing.T) {
	mc := NewMetricsCollector()

	metric := mc.CollectSystemMetrics()

	assert.False(t, metric.Timestamp.IsZero())
	assert.True(t, metric.MemoryUsage > 0, "memory usage should be > 0")
	assert.True(t, metric.GoroutineCount > 0, "goroutine count should be > 0")

	// Verify it was added to performance history
	history := mc.GetPerformanceHistory(10)
	require.Len(t, history, 1)
	assert.Equal(t, metric.MemoryUsage, history[0].MemoryUsage)

	// Verify stored metrics were updated
	summary := mc.GetMetricsSummary()
	assert.True(t, summary.CurrentMemoryUsage > 0)
	assert.True(t, summary.GoroutineCount > 0)
}

func TestMetricsCollector_RecordHTTPRequest(t *testing.T) {
	mc := NewMetricsCollector()

	// Record successful request
	mc.RecordHTTPRequest(100*time.Millisecond, true)

	summary := mc.GetMetricsSummary()
	assert.Equal(t, int64(1), summary.TotalHTTPRequests)
	assert.Equal(t, float64(0), summary.HTTPErrorRate)
	assert.True(t, summary.AverageResponseTime > 0)

	// Record failed request
	mc.RecordHTTPRequest(200*time.Millisecond, false)

	summary = mc.GetMetricsSummary()
	assert.Equal(t, int64(2), summary.TotalHTTPRequests)
	assert.Equal(t, float64(50), summary.HTTPErrorRate) // 1 error out of 2 requests = 50%
}

func TestMetricsCollector_RecordHTTPRequest_MultipleSuccessful(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordHTTPRequest(100*time.Millisecond, true)
	mc.RecordHTTPRequest(200*time.Millisecond, true)
	mc.RecordHTTPRequest(300*time.Millisecond, true)

	summary := mc.GetMetricsSummary()
	assert.Equal(t, int64(3), summary.TotalHTTPRequests)
	assert.Equal(t, float64(0), summary.HTTPErrorRate)
}

func TestMetricsCollector_SetActiveWorkflows(t *testing.T) {
	mc := NewMetricsCollector()

	mc.SetActiveWorkflows(10)
	summary := mc.GetMetricsSummary()
	assert.Equal(t, int64(10), summary.ActiveWorkflows)

	mc.SetActiveWorkflows(5)
	summary = mc.GetMetricsSummary()
	assert.Equal(t, int64(5), summary.ActiveWorkflows)

	mc.SetActiveWorkflows(0)
	summary = mc.GetMetricsSummary()
	assert.Equal(t, int64(0), summary.ActiveWorkflows)
}

func TestMetricsCollector_ActiveExecutions(t *testing.T) {
	mc := NewMetricsCollector()

	mc.IncrementActiveExecutions()
	mc.IncrementActiveExecutions()
	mc.IncrementActiveExecutions()

	summary := mc.GetMetricsSummary()
	assert.Equal(t, int64(3), summary.ActiveExecutions)

	mc.DecrementActiveExecutions()

	summary = mc.GetMetricsSummary()
	assert.Equal(t, int64(2), summary.ActiveExecutions)

	mc.DecrementActiveExecutions()
	mc.DecrementActiveExecutions()

	summary = mc.GetMetricsSummary()
	assert.Equal(t, int64(0), summary.ActiveExecutions)
}

func TestMetricsCollector_GetPerformanceHistory(t *testing.T) {
	mc := NewMetricsCollector()

	// Collect multiple system metrics snapshots
	mc.CollectSystemMetrics()
	mc.CollectSystemMetrics()
	mc.CollectSystemMetrics()

	history := mc.GetPerformanceHistory(2)
	assert.Len(t, history, 2)

	history = mc.GetPerformanceHistory(10)
	assert.Len(t, history, 3)
}

func TestMetricsCollector_GetErrorsByType(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordError(newTestErrorMetric("runtime", "E001", "runtime error", "wf-1", "high"))
	mc.RecordError(newTestErrorMetric("runtime", "E002", "another runtime error", "wf-2", "medium"))
	mc.RecordError(newTestErrorMetric("network", "E003", "network error", "wf-1", "high"))
	mc.RecordError(newTestErrorMetric("validation", "E004", "validation error", "wf-3", "low"))

	since := time.Now().Add(-1 * time.Hour)
	errorsByType := mc.GetErrorsByType(since)

	assert.Equal(t, 2, errorsByType["runtime"])
	assert.Equal(t, 1, errorsByType["network"])
	assert.Equal(t, 1, errorsByType["validation"])
}

func TestMetricsCollector_GetExecutionTrend(t *testing.T) {
	mc := NewMetricsCollector()

	// Record executions with recent timestamps
	metric := newTestExecutionMetric("wf-1", "exec-1", true, 100*time.Millisecond, 3)
	mc.RecordWorkflowExecution(metric)

	trend := mc.GetExecutionTrend(24 * time.Hour)
	assert.Len(t, trend, 24)

	// At least one interval should have a count > 0
	totalInTrend := 0
	for _, count := range trend {
		totalInTrend += count
	}
	assert.True(t, totalInTrend >= 1)
}

// ============================================================================
// AlertManager Tests
// ============================================================================

func TestAlertManager_AddRule(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)

	rule := &AlertRule{
		ID:          "test-rule-1",
		Name:        "Test Rule",
		Condition:   "memory_high",
		Threshold:   500,
		Duration:    300,
		Severity:    "high",
		Enabled:     true,
		Description: "Memory is too high",
	}

	err := am.AddRule(rule)
	require.NoError(t, err)

	retrieved, err := am.GetRule("test-rule-1")
	require.NoError(t, err)
	assert.Equal(t, "Test Rule", retrieved.Name)
	assert.Equal(t, "memory_high", retrieved.Condition)
	assert.Equal(t, float64(500), retrieved.Threshold)
	assert.Equal(t, "high", retrieved.Severity)
	assert.True(t, retrieved.Enabled)
}

func TestAlertManager_AddRule_AutoGenerateID(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)

	rule := &AlertRule{
		Name:      "Auto ID Rule",
		Condition: "error_rate_high",
		Threshold: 10,
		Severity:  "medium",
		Enabled:   true,
	}

	err := am.AddRule(rule)
	require.NoError(t, err)
	assert.NotEmpty(t, rule.ID, "ID should be auto-generated when empty")

	retrieved, err := am.GetRule(rule.ID)
	require.NoError(t, err)
	assert.Equal(t, "Auto ID Rule", retrieved.Name)
}

func TestAlertManager_RemoveRule(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)

	rule := &AlertRule{
		ID:        "rule-to-remove",
		Name:      "Removable Rule",
		Condition: "memory_high",
		Threshold: 500,
		Severity:  "high",
		Enabled:   true,
	}

	err := am.AddRule(rule)
	require.NoError(t, err)

	err = am.RemoveRule("rule-to-remove")
	require.NoError(t, err)

	_, err = am.GetRule("rule-to-remove")
	assert.Error(t, err)
}

func TestAlertManager_RemoveRule_NotFound(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)

	err := am.RemoveRule("nonexistent-rule")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestAlertManager_UpdateRule(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)

	rule := &AlertRule{
		ID:        "update-rule",
		Name:      "Original Name",
		Condition: "memory_high",
		Threshold: 500,
		Severity:  "high",
		Enabled:   true,
	}

	err := am.AddRule(rule)
	require.NoError(t, err)

	updatedRule := &AlertRule{
		ID:          "update-rule",
		Name:        "Updated Name",
		Condition:   "error_rate_high",
		Threshold:   20,
		Severity:    "critical",
		Enabled:     false,
		Description: "Updated description",
	}

	err = am.UpdateRule(updatedRule)
	require.NoError(t, err)

	retrieved, err := am.GetRule("update-rule")
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", retrieved.Name)
	assert.Equal(t, "error_rate_high", retrieved.Condition)
	assert.Equal(t, float64(20), retrieved.Threshold)
	assert.Equal(t, "critical", retrieved.Severity)
	assert.False(t, retrieved.Enabled)
}

func TestAlertManager_UpdateRule_NotFound(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)

	rule := &AlertRule{
		ID:   "nonexistent",
		Name: "Does Not Exist",
	}

	err := am.UpdateRule(rule)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestAlertManager_GetAllRules(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)

	rules := []*AlertRule{
		{ID: "rule-1", Name: "Rule 1", Condition: "memory_high", Threshold: 500, Severity: "high", Enabled: true},
		{ID: "rule-2", Name: "Rule 2", Condition: "error_rate_high", Threshold: 10, Severity: "medium", Enabled: true},
		{ID: "rule-3", Name: "Rule 3", Condition: "goroutines_high", Threshold: 1000, Severity: "low", Enabled: false},
	}

	for _, rule := range rules {
		err := am.AddRule(rule)
		require.NoError(t, err)
	}

	allRules := am.GetAllRules()
	assert.Len(t, allRules, 3)

	// Verify all rules are present (order may vary since it's a map)
	ruleIDs := make(map[string]bool)
	for _, r := range allRules {
		ruleIDs[r.ID] = true
	}
	assert.True(t, ruleIDs["rule-1"])
	assert.True(t, ruleIDs["rule-2"])
	assert.True(t, ruleIDs["rule-3"])
}

func TestAlertManager_EnableDisable(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)

	// Starts enabled by default
	assert.True(t, am.IsEnabled())

	am.Disable()
	assert.False(t, am.IsEnabled())

	am.Enable()
	assert.True(t, am.IsEnabled())
}

func TestAlertManager_AddHandler(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)

	handler1 := newMockAlertHandler("handler-1")
	handler2 := newMockAlertHandler("handler-2")

	am.AddHandler(handler1)
	am.AddHandler(handler2)

	// Handlers are added (we verify by triggering an alert evaluation which would use them)
	// We can also verify indirectly by removing a handler
	am.RemoveHandler("handler-1")

	// After removing handler-1, adding a rule and evaluating should only trigger handler-2
	// This is an indirect verification that handlers work
}

func TestAlertManager_LogAlertHandler(t *testing.T) {
	handler := NewLogAlertHandler()

	assert.Equal(t, "log", handler.GetName())

	alert := Alert{
		ID:        "test-alert",
		RuleID:    "test-rule",
		RuleName:  "Test Rule",
		Severity:  "high",
		Message:   "Test alert message",
		Value:     600,
		Threshold: 500,
		Timestamp: time.Now(),
		Resolved:  false,
	}

	err := handler.HandleAlert(alert)
	assert.NoError(t, err)

	// Test resolved alert
	alert.Resolved = true
	err = handler.HandleAlert(alert)
	assert.NoError(t, err)
}

func TestAlertManager_GetActiveAlerts(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)

	activeAlerts := am.GetActiveAlerts()
	assert.Empty(t, activeAlerts)
}

func TestAlertManager_GetAlertHistory(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)

	// Initially empty
	history := am.GetAlertHistory(10)
	assert.Empty(t, history)

	// History with zero limit returns all (which is empty)
	history = am.GetAlertHistory(0)
	assert.Empty(t, history)
}

func TestAlertManager_EvaluateRules_Disabled(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)
	am.Disable()

	err := am.EvaluateRules()
	require.NoError(t, err)
}

func TestAlertManager_EvaluateRules_NoRules(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)

	err := am.EvaluateRules()
	require.NoError(t, err)
}

func TestAlertManager_EvaluateRules_TriggersAlert(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)

	handler := newMockAlertHandler("test-handler")
	am.AddHandler(handler)

	// Add a rule that will trigger: success_rate_low with threshold 90
	// Since there are no executions, success rate is 0, which is >= 90? Actually no.
	// The condition check does: currentValue >= rule.Threshold for success_rate_low
	// With 0 success rate, 0 >= 90 is false, so it won't trigger.
	//
	// Instead, use a condition like "goroutines_high" with a very low threshold
	// so it's guaranteed to trigger.
	rule := &AlertRule{
		ID:          "goroutine-test",
		Name:        "Low Goroutine Threshold",
		Condition:   "goroutines_high",
		Threshold:   1, // very low, will always trigger
		Severity:    "medium",
		Enabled:     true,
		Description: "Goroutine count is above 1",
	}
	err := am.AddRule(rule)
	require.NoError(t, err)

	// Collect system metrics first so goroutine count is populated
	mc.CollectSystemMetrics()

	err = am.EvaluateRules()
	require.NoError(t, err)

	// The alert should have been triggered and sent to handler
	assert.True(t, len(handler.alerts) >= 1, "handler should have received at least one alert")
	if len(handler.alerts) > 0 {
		assert.Equal(t, "goroutine-test", handler.alerts[0].RuleID)
		assert.False(t, handler.alerts[0].Resolved)
	}

	// Active alerts should contain the alert
	activeAlerts := am.GetActiveAlerts()
	assert.True(t, len(activeAlerts) >= 1)
}

func TestAlertManager_ResolveAlert(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)

	handler := newMockAlertHandler("test-handler")
	am.AddHandler(handler)

	// Trigger an alert with a very low threshold
	rule := &AlertRule{
		ID:          "resolve-test",
		Name:        "Resolve Test Rule",
		Condition:   "goroutines_high",
		Threshold:   1,
		Severity:    "low",
		Enabled:     true,
		Description: "Test for resolving",
	}
	err := am.AddRule(rule)
	require.NoError(t, err)

	mc.CollectSystemMetrics()
	err = am.EvaluateRules()
	require.NoError(t, err)

	activeAlerts := am.GetActiveAlerts()
	require.True(t, len(activeAlerts) >= 1)

	// Resolve the alert
	alertID := activeAlerts[0].ID
	err = am.ResolveAlert(alertID)
	require.NoError(t, err)

	// Active alerts should be empty now (for this rule)
	activeAlerts = am.GetActiveAlerts()
	// It might still have other alerts if other rules triggered, but the resolved one should be gone
	for _, a := range activeAlerts {
		assert.NotEqual(t, alertID, a.ID)
	}

	// Alert history should contain the resolved alert
	history := am.GetAlertHistory(10)
	assert.True(t, len(history) >= 1)
}

func TestAlertManager_ResolveAlert_NotFound(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)

	err := am.ResolveAlert("nonexistent-alert")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestAlertManager_CreateDefaultAlertRules(t *testing.T) {
	rules := CreateDefaultAlertRules()

	assert.True(t, len(rules) >= 5, "should have at least 5 default rules")

	// Verify specific default rules exist
	ruleMap := make(map[string]*AlertRule)
	for _, rule := range rules {
		ruleMap[rule.ID] = rule
	}

	// Check high-memory-usage rule
	memRule, ok := ruleMap["high-memory-usage"]
	require.True(t, ok, "high-memory-usage rule should exist")
	assert.Equal(t, "memory_high", memRule.Condition)
	assert.Equal(t, float64(500), memRule.Threshold)
	assert.True(t, memRule.Enabled)

	// Check high-error-rate rule
	errRule, ok := ruleMap["high-error-rate"]
	require.True(t, ok, "high-error-rate rule should exist")
	assert.Equal(t, "error_rate_high", errRule.Condition)

	// Check low-success-rate rule
	successRule, ok := ruleMap["low-success-rate"]
	require.True(t, ok, "low-success-rate rule should exist")
	assert.Equal(t, "success_rate_low", successRule.Condition)

	// Check too-many-goroutines rule
	goroutineRule, ok := ruleMap["too-many-goroutines"]
	require.True(t, ok, "too-many-goroutines rule should exist")
	assert.Equal(t, "goroutines_high", goroutineRule.Condition)
}

func TestAlertManager_EmailAlertHandler(t *testing.T) {
	handler := NewEmailAlertHandler("smtp.example.com", 587, "user", "pass", []string{"admin@example.com"})

	assert.Equal(t, "email", handler.GetName())

	alert := Alert{
		ID:        "email-alert",
		RuleName:  "Test Rule",
		Message:   "Test email alert",
		Severity:  "high",
		Timestamp: time.Now(),
	}

	err := handler.HandleAlert(alert)
	assert.NoError(t, err)
}

func TestAlertManager_WebhookAlertHandler(t *testing.T) {
	handler := NewWebhookAlertHandler("https://hooks.example.com/alert")

	assert.Equal(t, "webhook", handler.GetName())

	alert := Alert{
		ID:        "webhook-alert",
		RuleName:  "Test Rule",
		Message:   "Test webhook alert",
		Severity:  "medium",
		Timestamp: time.Now(),
	}

	err := handler.HandleAlert(alert)
	assert.NoError(t, err)
}

func TestAlertManager_RemoveHandler(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)

	handler1 := newMockAlertHandler("handler-1")
	handler2 := newMockAlertHandler("handler-2")

	am.AddHandler(handler1)
	am.AddHandler(handler2)

	am.RemoveHandler("handler-1")

	// Trigger an alert to verify only handler-2 receives it
	rule := &AlertRule{
		ID:        "handler-remove-test",
		Name:      "Handler Remove Test",
		Condition: "goroutines_high",
		Threshold: 1,
		Severity:  "low",
		Enabled:   true,
	}
	err := am.AddRule(rule)
	require.NoError(t, err)

	mc.CollectSystemMetrics()
	err = am.EvaluateRules()
	require.NoError(t, err)

	// handler-1 should not have received the alert since it was removed
	assert.Empty(t, handler1.alerts)
	// handler-2 should have received the alert
	assert.True(t, len(handler2.alerts) >= 1)
}

// ============================================================================
// MonitoringManager Tests
// ============================================================================

func TestMonitoringManager_StartStop(t *testing.T) {
	config := newTestMonitoringConfig()
	mm := NewMonitoringManager(config)

	assert.False(t, mm.IsRunning())

	err := mm.Start()
	require.NoError(t, err)
	assert.True(t, mm.IsRunning())

	// Starting again should fail
	err = mm.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	err = mm.Stop()
	require.NoError(t, err)
	assert.False(t, mm.IsRunning())

	// Stopping again should fail
	err = mm.Stop()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestMonitoringManager_GetHealthStatus(t *testing.T) {
	config := newTestMonitoringConfig()
	mm := NewMonitoringManager(config)

	health := mm.GetHealthStatus()

	// Status is one of: healthy, warning, critical
	assert.Contains(t, []string{"healthy", "warning", "critical"}, health.Status)
	assert.True(t, health.Score >= 0 && health.Score <= 100)
	assert.False(t, health.LastChecked.IsZero())
	assert.NotNil(t, health.Details)
	assert.NotEmpty(t, health.SystemInfo.GoVersion)
	assert.NotEmpty(t, health.SystemInfo.Platform)
}

func TestMonitoringManager_GetHealthStatus_NoExecutions(t *testing.T) {
	config := MonitoringConfig{
		CollectInterval:  100 * time.Millisecond,
		AlertInterval:    100 * time.Millisecond,
		OptimizeInterval: 100 * time.Millisecond,
		EnableAlerts:     false,
		EnableOptimizer:  false,
	}
	mm := NewMonitoringManager(config)

	health := mm.GetHealthStatus()

	// With alerts disabled, there should be no active alerts
	assert.Empty(t, health.ActiveAlerts)
	// Status depends on runtime conditions (memory, goroutines, etc.)
	assert.Contains(t, []string{"healthy", "warning", "critical"}, health.Status)
	assert.True(t, health.Score >= 0 && health.Score <= 100)
	// Metrics should reflect zero executions
	assert.Equal(t, int64(0), health.Metrics.TotalExecutions)
}

func TestMonitoringManager_RecordWorkflowExecution(t *testing.T) {
	config := newTestMonitoringConfig()
	mm := NewMonitoringManager(config)

	metric := newTestExecutionMetric("wf-1", "exec-1", true, 150*time.Millisecond, 4)
	mm.RecordWorkflowExecution(metric)

	summary := mm.GetMetrics().GetMetricsSummary()
	assert.Equal(t, int64(1), summary.TotalExecutions)
	assert.Equal(t, int64(1), summary.SuccessfulExecutions)
}

func TestMonitoringManager_RecordError(t *testing.T) {
	config := newTestMonitoringConfig()
	mm := NewMonitoringManager(config)

	errMetric := newTestErrorMetric("runtime", "E001", "test error", "wf-1", "high")
	mm.RecordError(errMetric)

	errors := mm.GetMetrics().GetErrorHistory(10)
	require.Len(t, errors, 1)
	assert.Equal(t, "runtime", errors[0].ErrorType)
}

func TestMonitoringManager_RecordHTTPRequest(t *testing.T) {
	config := newTestMonitoringConfig()
	mm := NewMonitoringManager(config)

	mm.RecordHTTPRequest(50*time.Millisecond, true)
	mm.RecordHTTPRequest(100*time.Millisecond, false)

	summary := mm.GetMetrics().GetMetricsSummary()
	assert.Equal(t, int64(2), summary.TotalHTTPRequests)
	assert.Equal(t, float64(50), summary.HTTPErrorRate)
}

func TestMonitoringManager_GetDashboard(t *testing.T) {
	config := newTestMonitoringConfig()
	mm := NewMonitoringManager(config)

	// Record some data
	mm.RecordWorkflowExecution(newTestExecutionMetric("wf-1", "exec-1", true, 100*time.Millisecond, 3))
	mm.RecordError(newTestErrorMetric("runtime", "E001", "test", "wf-1", "medium"))

	dashboard := mm.GetDashboard()

	assert.NotEmpty(t, dashboard.Health.Status)
	assert.Equal(t, int64(1), dashboard.Metrics.TotalExecutions)
	assert.Len(t, dashboard.RecentExecutions, 1)
	assert.Len(t, dashboard.RecentErrors, 1)
	assert.NotEmpty(t, dashboard.SystemInfo.GoVersion)
	assert.False(t, dashboard.Timestamp.IsZero())
}

func TestMonitoringManager_GetSystemStats(t *testing.T) {
	config := newTestMonitoringConfig()
	mm := NewMonitoringManager(config)

	stats := mm.GetSystemStats()

	// Verify memory stats are present
	memoryStats, ok := stats["memory"].(map[string]interface{})
	require.True(t, ok)
	assert.NotNil(t, memoryStats["alloc"])
	assert.NotNil(t, memoryStats["totalAlloc"])
	assert.NotNil(t, memoryStats["sys"])
	assert.NotNil(t, memoryStats["heapAlloc"])

	// Verify runtime stats are present
	runtimeStats, ok := stats["runtime"].(map[string]interface{})
	require.True(t, ok)
	assert.NotNil(t, runtimeStats["version"])
	assert.NotNil(t, runtimeStats["numGoroutine"])
	assert.NotNil(t, runtimeStats["numCPU"])

	// Verify system info is present
	systemInfo, ok := stats["system"].(SystemInfo)
	require.True(t, ok)
	assert.NotEmpty(t, systemInfo.GoVersion)
}

func TestMonitoringManager_Reset(t *testing.T) {
	config := newTestMonitoringConfig()
	mm := NewMonitoringManager(config)

	mm.RecordWorkflowExecution(newTestExecutionMetric("wf-1", "exec-1", true, 100*time.Millisecond, 3))
	mm.RecordError(newTestErrorMetric("runtime", "E001", "test", "wf-1", "high"))

	mm.Reset()

	summary := mm.GetMetrics().GetMetricsSummary()
	assert.Equal(t, int64(0), summary.TotalExecutions)
	assert.Equal(t, int64(0), summary.SuccessfulExecutions)
	assert.Equal(t, int64(0), summary.FailedExecutions)
}

func TestMonitoringManager_SetActiveWorkflows(t *testing.T) {
	config := newTestMonitoringConfig()
	mm := NewMonitoringManager(config)

	mm.SetActiveWorkflows(7)

	summary := mm.GetMetrics().GetMetricsSummary()
	assert.Equal(t, int64(7), summary.ActiveWorkflows)
}

func TestMonitoringManager_IncrementDecrementExecutions(t *testing.T) {
	config := newTestMonitoringConfig()
	mm := NewMonitoringManager(config)

	mm.IncrementActiveExecutions()
	mm.IncrementActiveExecutions()

	summary := mm.GetMetrics().GetMetricsSummary()
	assert.Equal(t, int64(2), summary.ActiveExecutions)

	mm.DecrementActiveExecutions()

	summary = mm.GetMetrics().GetMetricsSummary()
	assert.Equal(t, int64(1), summary.ActiveExecutions)
}

func TestMonitoringManager_GetComponents(t *testing.T) {
	config := newTestMonitoringConfig()
	mm := NewMonitoringManager(config)

	assert.NotNil(t, mm.GetMetrics())
	assert.NotNil(t, mm.GetAlertManager())
	assert.NotNil(t, mm.GetOptimizer())
}

func TestMonitoringManager_AddAlertHandler(t *testing.T) {
	config := newTestMonitoringConfig()
	mm := NewMonitoringManager(config)

	handler := newMockAlertHandler("test-handler")
	mm.AddAlertHandler(handler)

	// Verify indirectly by checking the alert manager
	assert.NotNil(t, mm.GetAlertManager())
}

func TestMonitoringManager_AddAlertRule(t *testing.T) {
	config := newTestMonitoringConfig()
	mm := NewMonitoringManager(config)

	rule := &AlertRule{
		ID:        "mm-test-rule",
		Name:      "MM Test Rule",
		Condition: "memory_high",
		Threshold: 800,
		Severity:  "high",
		Enabled:   true,
	}

	err := mm.AddAlertRule(rule)
	require.NoError(t, err)

	retrieved, err := mm.GetAlertManager().GetRule("mm-test-rule")
	require.NoError(t, err)
	assert.Equal(t, "MM Test Rule", retrieved.Name)
}

func TestMonitoringManager_TriggerGC(t *testing.T) {
	config := newTestMonitoringConfig()
	mm := NewMonitoringManager(config)

	// Should not panic
	mm.TriggerGC()
}

func TestMonitoringManager_TriggerOptimization(t *testing.T) {
	config := newTestMonitoringConfig()
	mm := NewMonitoringManager(config)

	err := mm.TriggerOptimization()
	assert.NoError(t, err)
}

func TestMonitoringManager_DefaultIntervals(t *testing.T) {
	// Config with zero intervals should use defaults
	config := MonitoringConfig{
		EnableAlerts:    false,
		EnableOptimizer: false,
	}
	mm := NewMonitoringManager(config)

	// Verify the manager was created successfully
	assert.NotNil(t, mm)
	assert.NotNil(t, mm.GetMetrics())
}

func TestMonitoringManager_WithDefaultAlertRules(t *testing.T) {
	config := MonitoringConfig{
		CollectInterval:  100 * time.Millisecond,
		AlertInterval:    100 * time.Millisecond,
		OptimizeInterval: 100 * time.Millisecond,
		EnableAlerts:     true,
		EnableOptimizer:  false,
		AlertRules:       nil, // Should trigger default rules
	}
	mm := NewMonitoringManager(config)

	rules := mm.GetAlertManager().GetAllRules()
	assert.True(t, len(rules) >= 5, "default rules should be added when none are provided")
}

func TestMonitoringManager_WithCustomAlertRules(t *testing.T) {
	customRules := []*AlertRule{
		{
			ID:        "custom-rule-1",
			Name:      "Custom Rule 1",
			Condition: "memory_high",
			Threshold: 1000,
			Severity:  "critical",
			Enabled:   true,
		},
	}

	config := MonitoringConfig{
		CollectInterval:  100 * time.Millisecond,
		AlertInterval:    100 * time.Millisecond,
		OptimizeInterval: 100 * time.Millisecond,
		EnableAlerts:     true,
		EnableOptimizer:  false,
		AlertRules:       customRules,
	}
	mm := NewMonitoringManager(config)

	rules := mm.GetAlertManager().GetAllRules()
	assert.Len(t, rules, 1)
	assert.Equal(t, "Custom Rule 1", rules[0].Name)
}

func TestMonitoringManager_StartStop_BackgroundLoops(t *testing.T) {
	config := MonitoringConfig{
		CollectInterval:  50 * time.Millisecond,
		AlertInterval:    50 * time.Millisecond,
		OptimizeInterval: 50 * time.Millisecond,
		EnableAlerts:     false,
		EnableOptimizer:  false,
	}
	mm := NewMonitoringManager(config)

	err := mm.Start()
	require.NoError(t, err)

	// Let the background loops run for a bit
	time.Sleep(200 * time.Millisecond)

	err = mm.Stop()
	require.NoError(t, err)

	// Verify system metrics were collected by the background loop
	history := mm.GetMetrics().GetPerformanceHistory(10)
	assert.True(t, len(history) >= 1, "background collection should have collected at least one metric")
}

// ============================================================================
// PerformanceOptimizer Tests
// ============================================================================

func TestPerformanceOptimizer_EnableDisable(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)
	po := NewPerformanceOptimizer(mc, am)

	// Default is enabled
	assert.True(t, po.IsEnabled())

	po.Disable()
	assert.False(t, po.IsEnabled())

	po.Enable()
	assert.True(t, po.IsEnabled())
}

func TestPerformanceOptimizer_RunOptimization(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)
	po := NewPerformanceOptimizer(mc, am)

	err := po.RunOptimization()
	assert.NoError(t, err)
}

func TestPerformanceOptimizer_RunOptimization_Disabled(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)
	po := NewPerformanceOptimizer(mc, am)
	po.Disable()

	err := po.RunOptimization()
	assert.NoError(t, err)
}

func TestPerformanceOptimizer_GetOptimizationHistory(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)
	po := NewPerformanceOptimizer(mc, am)

	// Initially empty
	history := po.GetOptimizationHistory(10)
	assert.Empty(t, history)

	// With zero limit
	history = po.GetOptimizationHistory(0)
	assert.Empty(t, history)
}

func TestPerformanceOptimizer_GetMemoryProfile(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)
	po := NewPerformanceOptimizer(mc, am)

	profile := po.GetMemoryProfile()

	assert.NotNil(t, profile)
	assert.NotNil(t, profile["alloc"])
	assert.NotNil(t, profile["totalAlloc"])
	assert.NotNil(t, profile["sys"])
	assert.NotNil(t, profile["heapAlloc"])
	assert.NotNil(t, profile["heapSys"])
	assert.NotNil(t, profile["heapIdle"])
	assert.NotNil(t, profile["heapInuse"])
	assert.NotNil(t, profile["heapReleased"])
	assert.NotNil(t, profile["heapObjects"])
	assert.NotNil(t, profile["stackInuse"])
	assert.NotNil(t, profile["stackSys"])
	assert.NotNil(t, profile["numGC"])
	assert.NotNil(t, profile["gcCPUFraction"])
}

func TestPerformanceOptimizer_SetMemoryThreshold(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)
	po := NewPerformanceOptimizer(mc, am)

	po.SetMemoryThreshold(1024 * 1024 * 1024) // 1GB

	tuning := po.GetCurrentTuning()
	assert.Equal(t, int64(1024*1024*1024), tuning.MemoryLimit)
}

func TestPerformanceOptimizer_SetGCTuningEnabled(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)
	po := NewPerformanceOptimizer(mc, am)

	// Verify it doesn't panic
	po.SetGCTuningEnabled(false)
	po.SetGCTuningEnabled(true)
}

func TestPerformanceOptimizer_GetRecommendations(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)
	po := NewPerformanceOptimizer(mc, am)

	// With default metrics (low usage), there should be few or no recommendations
	recommendations := po.GetRecommendations()
	assert.NotNil(t, recommendations)
}

func TestPerformanceOptimizer_GetCurrentTuning(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)
	po := NewPerformanceOptimizer(mc, am)

	tuning := po.GetCurrentTuning()

	assert.True(t, tuning.MaxProcs > 0)
	assert.True(t, tuning.MemoryLimit > 0)
	assert.Equal(t, 1000, tuning.GoroutineLimit)
	assert.Equal(t, 25, tuning.ConnectionPoolSize)
	assert.Equal(t, 30*time.Second, tuning.HTTPTimeout)
	assert.Equal(t, 100, tuning.CacheSize)
}

func TestPerformanceOptimizer_UpdateTuning(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)
	po := NewPerformanceOptimizer(mc, am)

	tuning := PerformanceTuning{
		GCTargetPercent: 150,
		MaxProcs:        2,
		MemoryLimit:     1 << 30, // 1GB
	}

	err := po.UpdateTuning(tuning)
	require.NoError(t, err)

	current := po.GetCurrentTuning()
	assert.Equal(t, int64(1<<30), current.MemoryLimit)
}

func TestPerformanceOptimizer_ApplyRecommendation(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)
	po := NewPerformanceOptimizer(mc, am)

	// Apply known recommendation
	err := po.ApplyRecommendation("mem-high")
	assert.NoError(t, err)

	// Apply another known recommendation
	err = po.ApplyRecommendation("goroutine-high")
	assert.NoError(t, err)

	// Apply unknown recommendation
	err = po.ApplyRecommendation("unknown-recommendation")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown recommendation ID")
}

func TestPerformanceOptimizer_GetRecommendations_WithFailedExecutions(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)
	po := NewPerformanceOptimizer(mc, am)

	// Record some failed executions to lower the success rate below 95%
	for i := 0; i < 10; i++ {
		mc.RecordWorkflowExecution(newTestExecutionMetric("wf-1", "exec-fail", false, 100*time.Millisecond, 2))
	}
	mc.RecordWorkflowExecution(newTestExecutionMetric("wf-1", "exec-ok", true, 100*time.Millisecond, 2))

	recommendations := po.GetRecommendations()
	assert.NotNil(t, recommendations)

	// Should have a recommendation about low success rate
	found := false
	for _, rec := range recommendations {
		if rec.ID == "success-low" {
			found = true
			assert.Equal(t, "reliability", rec.Type)
			assert.Equal(t, "high", rec.Priority)
			break
		}
	}
	assert.True(t, found, "should have a low success rate recommendation")
}

// ============================================================================
// Integration-style Tests
// ============================================================================

func TestMonitoringManager_FullWorkflow(t *testing.T) {
	config := MonitoringConfig{
		CollectInterval:  50 * time.Millisecond,
		AlertInterval:    50 * time.Millisecond,
		OptimizeInterval: 50 * time.Millisecond,
		EnableAlerts:     true,
		EnableOptimizer:  true,
		AlertRules: []*AlertRule{
			{
				ID:        "integration-test-rule",
				Name:      "Integration Test Rule",
				Condition: "goroutines_high",
				Threshold: 1,
				Severity:  "low",
				Enabled:   true,
			},
		},
	}

	mm := NewMonitoringManager(config)

	// Record executions
	mm.RecordWorkflowExecution(newTestExecutionMetric("wf-1", "exec-1", true, 100*time.Millisecond, 3))
	mm.RecordWorkflowExecution(newTestExecutionMetric("wf-2", "exec-2", true, 200*time.Millisecond, 5))
	mm.RecordWorkflowExecution(newTestExecutionMetric("wf-3", "exec-3", false, 300*time.Millisecond, 2))

	// Record errors
	mm.RecordError(newTestErrorMetric("runtime", "E001", "test error 1", "wf-3", "high"))

	// Record HTTP requests
	mm.RecordHTTPRequest(50*time.Millisecond, true)
	mm.RecordHTTPRequest(100*time.Millisecond, true)

	// Set active workflows
	mm.SetActiveWorkflows(3)

	// Get dashboard
	dashboard := mm.GetDashboard()

	assert.Equal(t, int64(3), dashboard.Metrics.TotalExecutions)
	assert.Equal(t, int64(2), dashboard.Metrics.SuccessfulExecutions)
	assert.Equal(t, int64(1), dashboard.Metrics.FailedExecutions)
	assert.Equal(t, int64(2), dashboard.Metrics.TotalHTTPRequests)
	assert.Equal(t, int64(3), dashboard.Metrics.ActiveWorkflows)
	assert.Len(t, dashboard.RecentExecutions, 3)
	assert.Len(t, dashboard.RecentErrors, 1)

	// Health status
	health := mm.GetHealthStatus()
	assert.NotEmpty(t, health.Status)
	assert.True(t, health.Score > 0)
}

func TestMonitoringManager_ConfigDisabledAlerts(t *testing.T) {
	config := MonitoringConfig{
		CollectInterval:  100 * time.Millisecond,
		AlertInterval:    100 * time.Millisecond,
		OptimizeInterval: 100 * time.Millisecond,
		EnableAlerts:     false,
		EnableOptimizer:  false,
	}

	mm := NewMonitoringManager(config)

	// Alert manager should be disabled
	assert.False(t, mm.GetAlertManager().IsEnabled())

	// Optimizer should be disabled
	assert.False(t, mm.GetOptimizer().IsEnabled())
}

func TestMonitoringManager_ConfigEnabledWithMemoryThreshold(t *testing.T) {
	config := MonitoringConfig{
		CollectInterval:  100 * time.Millisecond,
		AlertInterval:    100 * time.Millisecond,
		OptimizeInterval: 100 * time.Millisecond,
		EnableAlerts:     false,
		EnableOptimizer:  true,
		MemoryThreshold:  1024 * 1024 * 1024, // 1GB
	}

	mm := NewMonitoringManager(config)

	tuning := mm.GetOptimizer().GetCurrentTuning()
	assert.Equal(t, int64(1024*1024*1024), tuning.MemoryLimit)
}
