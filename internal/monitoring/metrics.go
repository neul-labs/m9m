package monitoring

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// MetricsCollector collects and manages system and application metrics
type MetricsCollector struct {
	mu                    sync.RWMutex
	startTime             time.Time
	workflowExecutions    int64
	nodeExecutions        int64
	successfulExecutions  int64
	failedExecutions      int64
	totalExecutionTime    int64 // in nanoseconds
	activeWorkflows       int64
	activeExecutions      int64
	httpRequests          int64
	httpErrors            int64
	responseTime          int64 // average response time in nanoseconds
	memoryUsage           int64
	goroutineCount        int64
	executionHistory      []ExecutionMetric
	errorHistory          []ErrorMetric
	performanceHistory    []PerformanceMetric
	maxHistorySize        int
}

// ExecutionMetric represents metrics for a single execution
type ExecutionMetric struct {
	WorkflowID    string        `json:"workflowId"`
	ExecutionID   string        `json:"executionId"`
	StartTime     time.Time     `json:"startTime"`
	EndTime       time.Time     `json:"endTime"`
	Duration      time.Duration `json:"duration"`
	Success       bool          `json:"success"`
	NodeCount     int           `json:"nodeCount"`
	ErrorMessage  string        `json:"errorMessage,omitempty"`
	ExecutionMode string        `json:"executionMode"` // manual, trigger, schedule
}

// ErrorMetric represents error metrics
type ErrorMetric struct {
	Timestamp   time.Time `json:"timestamp"`
	ErrorType   string    `json:"errorType"`
	ErrorCode   string    `json:"errorCode,omitempty"`
	Message     string    `json:"message"`
	WorkflowID  string    `json:"workflowId,omitempty"`
	NodeID      string    `json:"nodeId,omitempty"`
	StackTrace  string    `json:"stackTrace,omitempty"`
	Severity    string    `json:"severity"` // low, medium, high, critical
}

// PerformanceMetric represents system performance metrics
type PerformanceMetric struct {
	Timestamp         time.Time `json:"timestamp"`
	CPUUsage          float64   `json:"cpuUsage"`
	MemoryUsage       int64     `json:"memoryUsage"`
	MemoryPercent     float64   `json:"memoryPercent"`
	GoroutineCount    int       `json:"goroutineCount"`
	ActiveWorkflows   int64     `json:"activeWorkflows"`
	ActiveExecutions  int64     `json:"activeExecutions"`
	DatabaseConns     int       `json:"databaseConnections,omitempty"`
	RedisConns        int       `json:"redisConnections,omitempty"`
	HTTPConnections   int       `json:"httpConnections,omitempty"`
}

// SystemInfo represents overall system information
type SystemInfo struct {
	Version           string        `json:"version"`
	BuildDate         string        `json:"buildDate"`
	GoVersion         string        `json:"goVersion"`
	Platform          string        `json:"platform"`
	Uptime            time.Duration `json:"uptime"`
	StartTime         time.Time     `json:"startTime"`
	ProcessID         int           `json:"processId"`
	WorkingDirectory  string        `json:"workingDirectory"`
	HostName          string        `json:"hostName"`
	MemoryLimit       int64         `json:"memoryLimit"`
	CPUCount          int           `json:"cpuCount"`
}

// MetricsSummary provides aggregated metrics
type MetricsSummary struct {
	TotalExecutions      int64         `json:"totalExecutions"`
	SuccessfulExecutions int64         `json:"successfulExecutions"`
	FailedExecutions     int64         `json:"failedExecutions"`
	SuccessRate          float64       `json:"successRate"`
	AverageExecutionTime time.Duration `json:"averageExecutionTime"`
	ExecutionsPerMinute  float64       `json:"executionsPerMinute"`
	ActiveWorkflows      int64         `json:"activeWorkflows"`
	ActiveExecutions     int64         `json:"activeExecutions"`
	TotalHTTPRequests    int64         `json:"totalHttpRequests"`
	HTTPErrorRate        float64       `json:"httpErrorRate"`
	AverageResponseTime  time.Duration `json:"averageResponseTime"`
	CurrentMemoryUsage   int64         `json:"currentMemoryUsage"`
	PeakMemoryUsage      int64         `json:"peakMemoryUsage"`
	GoroutineCount       int64         `json:"goroutineCount"`
	Uptime               time.Duration `json:"uptime"`
}

// AlertRule defines conditions for triggering alerts
type AlertRule struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Condition   string  `json:"condition"`   // memory_high, error_rate_high, execution_time_high, etc.
	Threshold   float64 `json:"threshold"`   // threshold value
	Duration    int     `json:"duration"`    // duration in seconds to evaluate
	Severity    string  `json:"severity"`    // low, medium, high, critical
	Enabled     bool    `json:"enabled"`
	Description string  `json:"description"`
}

// Alert represents a triggered alert
type Alert struct {
	ID          string    `json:"id"`
	RuleID      string    `json:"ruleId"`
	RuleName    string    `json:"ruleName"`
	Severity    string    `json:"severity"`
	Message     string    `json:"message"`
	Value       float64   `json:"value"`
	Threshold   float64   `json:"threshold"`
	Timestamp   time.Time `json:"timestamp"`
	Resolved    bool      `json:"resolved"`
	ResolvedAt  *time.Time `json:"resolvedAt,omitempty"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		startTime:      time.Now(),
		maxHistorySize: 1000, // Keep last 1000 entries for each history
	}
}

// RecordWorkflowExecution records metrics for a workflow execution
func (mc *MetricsCollector) RecordWorkflowExecution(metric ExecutionMetric) {
	atomic.AddInt64(&mc.workflowExecutions, 1)
	atomic.AddInt64(&mc.nodeExecutions, int64(metric.NodeCount))
	atomic.AddInt64(&mc.totalExecutionTime, int64(metric.Duration))

	if metric.Success {
		atomic.AddInt64(&mc.successfulExecutions, 1)
	} else {
		atomic.AddInt64(&mc.failedExecutions, 1)
	}

	// Add to execution history
	mc.mu.Lock()
	mc.executionHistory = append(mc.executionHistory, metric)
	if len(mc.executionHistory) > mc.maxHistorySize {
		mc.executionHistory = mc.executionHistory[1:]
	}
	mc.mu.Unlock()
}

// RecordError records an error metric
func (mc *MetricsCollector) RecordError(errorMetric ErrorMetric) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.errorHistory = append(mc.errorHistory, errorMetric)
	if len(mc.errorHistory) > mc.maxHistorySize {
		mc.errorHistory = mc.errorHistory[1:]
	}
}

// RecordHTTPRequest records HTTP request metrics
func (mc *MetricsCollector) RecordHTTPRequest(responseTime time.Duration, success bool) {
	atomic.AddInt64(&mc.httpRequests, 1)

	if !success {
		atomic.AddInt64(&mc.httpErrors, 1)
	}

	// Update average response time using atomic operations
	currentAvg := atomic.LoadInt64(&mc.responseTime)
	requests := atomic.LoadInt64(&mc.httpRequests)

	// Calculate new average: (old_avg * (n-1) + new_value) / n
	newAvg := (currentAvg*int64(requests-1) + int64(responseTime)) / requests
	atomic.StoreInt64(&mc.responseTime, newAvg)
}

// SetActiveWorkflows sets the number of active workflows
func (mc *MetricsCollector) SetActiveWorkflows(count int64) {
	atomic.StoreInt64(&mc.activeWorkflows, count)
}

// IncrementActiveExecutions increments active executions counter
func (mc *MetricsCollector) IncrementActiveExecutions() {
	atomic.AddInt64(&mc.activeExecutions, 1)
}

// DecrementActiveExecutions decrements active executions counter
func (mc *MetricsCollector) DecrementActiveExecutions() {
	atomic.AddInt64(&mc.activeExecutions, -1)
}

// CollectSystemMetrics collects current system performance metrics
func (mc *MetricsCollector) CollectSystemMetrics() PerformanceMetric {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metric := PerformanceMetric{
		Timestamp:        time.Now(),
		MemoryUsage:      int64(m.Alloc),
		GoroutineCount:   runtime.NumGoroutine(),
		ActiveWorkflows:  atomic.LoadInt64(&mc.activeWorkflows),
		ActiveExecutions: atomic.LoadInt64(&mc.activeExecutions),
	}

	// Update stored memory usage
	atomic.StoreInt64(&mc.memoryUsage, int64(m.Alloc))
	atomic.StoreInt64(&mc.goroutineCount, int64(runtime.NumGoroutine()))

	// Add to performance history
	mc.mu.Lock()
	mc.performanceHistory = append(mc.performanceHistory, metric)
	if len(mc.performanceHistory) > mc.maxHistorySize {
		mc.performanceHistory = mc.performanceHistory[1:]
	}
	mc.mu.Unlock()

	return metric
}

// GetMetricsSummary returns a summary of all metrics
func (mc *MetricsCollector) GetMetricsSummary() MetricsSummary {
	totalExec := atomic.LoadInt64(&mc.workflowExecutions)
	successfulExec := atomic.LoadInt64(&mc.successfulExecutions)
	failedExec := atomic.LoadInt64(&mc.failedExecutions)
	totalExecTime := atomic.LoadInt64(&mc.totalExecutionTime)
	httpReqs := atomic.LoadInt64(&mc.httpRequests)
	httpErrs := atomic.LoadInt64(&mc.httpErrors)
	avgResponseTime := atomic.LoadInt64(&mc.responseTime)

	var successRate float64
	if totalExec > 0 {
		successRate = float64(successfulExec) / float64(totalExec) * 100
	}

	var avgExecTime time.Duration
	if totalExec > 0 {
		avgExecTime = time.Duration(totalExecTime / totalExec)
	}

	var httpErrorRate float64
	if httpReqs > 0 {
		httpErrorRate = float64(httpErrs) / float64(httpReqs) * 100
	}

	uptime := time.Since(mc.startTime)
	var execPerMinute float64
	if uptime.Minutes() > 0 {
		execPerMinute = float64(totalExec) / uptime.Minutes()
	}

	// Get peak memory usage from history
	var peakMemory int64
	mc.mu.RLock()
	for _, metric := range mc.performanceHistory {
		if metric.MemoryUsage > peakMemory {
			peakMemory = metric.MemoryUsage
		}
	}
	mc.mu.RUnlock()

	return MetricsSummary{
		TotalExecutions:      totalExec,
		SuccessfulExecutions: successfulExec,
		FailedExecutions:     failedExec,
		SuccessRate:          successRate,
		AverageExecutionTime: avgExecTime,
		ExecutionsPerMinute:  execPerMinute,
		ActiveWorkflows:      atomic.LoadInt64(&mc.activeWorkflows),
		ActiveExecutions:     atomic.LoadInt64(&mc.activeExecutions),
		TotalHTTPRequests:    httpReqs,
		HTTPErrorRate:        httpErrorRate,
		AverageResponseTime:  time.Duration(avgResponseTime),
		CurrentMemoryUsage:   atomic.LoadInt64(&mc.memoryUsage),
		PeakMemoryUsage:      peakMemory,
		GoroutineCount:       atomic.LoadInt64(&mc.goroutineCount),
		Uptime:               uptime,
	}
}

// GetExecutionHistory returns recent execution history
func (mc *MetricsCollector) GetExecutionHistory(limit int) []ExecutionMetric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if limit <= 0 || limit > len(mc.executionHistory) {
		limit = len(mc.executionHistory)
	}

	start := len(mc.executionHistory) - limit
	if start < 0 {
		start = 0
	}

	result := make([]ExecutionMetric, limit)
	copy(result, mc.executionHistory[start:])
	return result
}

// GetErrorHistory returns recent error history
func (mc *MetricsCollector) GetErrorHistory(limit int) []ErrorMetric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if limit <= 0 || limit > len(mc.errorHistory) {
		limit = len(mc.errorHistory)
	}

	start := len(mc.errorHistory) - limit
	if start < 0 {
		start = 0
	}

	result := make([]ErrorMetric, limit)
	copy(result, mc.errorHistory[start:])
	return result
}

// GetPerformanceHistory returns recent performance history
func (mc *MetricsCollector) GetPerformanceHistory(limit int) []PerformanceMetric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if limit <= 0 || limit > len(mc.performanceHistory) {
		limit = len(mc.performanceHistory)
	}

	start := len(mc.performanceHistory) - limit
	if start < 0 {
		start = 0
	}

	result := make([]PerformanceMetric, limit)
	copy(result, mc.performanceHistory[start:])
	return result
}

// GetErrorsByType returns errors grouped by type
func (mc *MetricsCollector) GetErrorsByType(since time.Time) map[string]int {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	errorsByType := make(map[string]int)
	for _, errorMetric := range mc.errorHistory {
		if errorMetric.Timestamp.After(since) {
			errorsByType[errorMetric.ErrorType]++
		}
	}

	return errorsByType
}

// GetExecutionTrend returns execution trend data
func (mc *MetricsCollector) GetExecutionTrend(duration time.Duration) []int {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	since := time.Now().Add(-duration)
	intervals := 24 // 24 intervals for the duration
	intervalDuration := duration / time.Duration(intervals)
	trend := make([]int, intervals)

	for _, exec := range mc.executionHistory {
		if exec.StartTime.After(since) {
			intervalIndex := int(exec.StartTime.Sub(since) / intervalDuration)
			if intervalIndex >= 0 && intervalIndex < intervals {
				trend[intervalIndex]++
			}
		}
	}

	return trend
}

// Reset resets all metrics (useful for testing)
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	atomic.StoreInt64(&mc.workflowExecutions, 0)
	atomic.StoreInt64(&mc.nodeExecutions, 0)
	atomic.StoreInt64(&mc.successfulExecutions, 0)
	atomic.StoreInt64(&mc.failedExecutions, 0)
	atomic.StoreInt64(&mc.totalExecutionTime, 0)
	atomic.StoreInt64(&mc.activeWorkflows, 0)
	atomic.StoreInt64(&mc.activeExecutions, 0)
	atomic.StoreInt64(&mc.httpRequests, 0)
	atomic.StoreInt64(&mc.httpErrors, 0)
	atomic.StoreInt64(&mc.responseTime, 0)
	atomic.StoreInt64(&mc.memoryUsage, 0)
	atomic.StoreInt64(&mc.goroutineCount, 0)

	mc.executionHistory = nil
	mc.errorHistory = nil
	mc.performanceHistory = nil
	mc.startTime = time.Now()
}

// SetMaxHistorySize sets the maximum size for history arrays
func (mc *MetricsCollector) SetMaxHistorySize(size int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.maxHistorySize = size
}