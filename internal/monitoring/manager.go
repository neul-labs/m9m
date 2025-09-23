package monitoring

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"
)

// MonitoringManager coordinates all monitoring components
type MonitoringManager struct {
	mu               sync.RWMutex
	metrics          *MetricsCollector
	alertManager     *AlertManager
	optimizer        *PerformanceOptimizer
	systemInfo       SystemInfo
	running          bool
	ctx              context.Context
	cancel           context.CancelFunc
	collectInterval  time.Duration
	alertInterval    time.Duration
	optimizeInterval time.Duration
}

// MonitoringConfig contains configuration for monitoring
type MonitoringConfig struct {
	CollectInterval  time.Duration `json:"collectInterval"`
	AlertInterval    time.Duration `json:"alertInterval"`
	OptimizeInterval time.Duration `json:"optimizeInterval"`
	EnableAlerts     bool          `json:"enableAlerts"`
	EnableOptimizer  bool          `json:"enableOptimizer"`
	MemoryThreshold  int64         `json:"memoryThreshold"`
	AlertRules       []*AlertRule  `json:"alertRules"`
}

// HealthStatus represents the overall system health
type HealthStatus struct {
	Status       string                 `json:"status"` // healthy, warning, critical
	Score        int                    `json:"score"`  // 0-100
	Issues       []HealthIssue          `json:"issues"`
	Metrics      MetricsSummary         `json:"metrics"`
	LastChecked  time.Time              `json:"lastChecked"`
	Uptime       time.Duration          `json:"uptime"`
	SystemInfo   SystemInfo             `json:"systemInfo"`
	ActiveAlerts []Alert                `json:"activeAlerts"`
	Details      map[string]interface{} `json:"details"`
}

// HealthIssue represents a health issue
type HealthIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Metric      string `json:"metric,omitempty"`
	Value       string `json:"value,omitempty"`
	Threshold   string `json:"threshold,omitempty"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// MonitoringDashboard provides dashboard data
type MonitoringDashboard struct {
	Health           HealthStatus                 `json:"health"`
	Metrics          MetricsSummary               `json:"metrics"`
	RecentExecutions []ExecutionMetric            `json:"recentExecutions"`
	RecentErrors     []ErrorMetric                `json:"recentErrors"`
	PerformanceData  []PerformanceMetric          `json:"performanceData"`
	ActiveAlerts     []Alert                      `json:"activeAlerts"`
	Recommendations  []OptimizationRecommendation `json:"recommendations"`
	SystemInfo       SystemInfo                   `json:"systemInfo"`
	Timestamp        time.Time                    `json:"timestamp"`
}

// NewMonitoringManager creates a new monitoring manager
func NewMonitoringManager(config MonitoringConfig) *MonitoringManager {
	metrics := NewMetricsCollector()
	alertManager := NewAlertManager(metrics)
	optimizer := NewPerformanceOptimizer(metrics, alertManager)

	// Configure alert manager
	if config.EnableAlerts {
		alertManager.Enable()
		// Add default log handler
		alertManager.AddHandler(NewLogAlertHandler())

		// Add default rules if none provided
		if len(config.AlertRules) == 0 {
			config.AlertRules = CreateDefaultAlertRules()
		}

		for _, rule := range config.AlertRules {
			alertManager.AddRule(rule)
		}
	} else {
		alertManager.Disable()
	}

	// Configure optimizer
	if config.EnableOptimizer {
		optimizer.Enable()
		if config.MemoryThreshold > 0 {
			optimizer.SetMemoryThreshold(config.MemoryThreshold)
		}
	} else {
		optimizer.Disable()
	}

	// Set default intervals if not provided
	if config.CollectInterval == 0 {
		config.CollectInterval = 30 * time.Second
	}
	if config.AlertInterval == 0 {
		config.AlertInterval = 1 * time.Minute
	}
	if config.OptimizeInterval == 0 {
		config.OptimizeInterval = 5 * time.Minute
	}

	systemInfo := collectSystemInfo()

	return &MonitoringManager{
		metrics:          metrics,
		alertManager:     alertManager,
		optimizer:        optimizer,
		systemInfo:       systemInfo,
		collectInterval:  config.CollectInterval,
		alertInterval:    config.AlertInterval,
		optimizeInterval: config.OptimizeInterval,
	}
}

// Start starts the monitoring manager
func (mm *MonitoringManager) Start() error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if mm.running {
		return fmt.Errorf("monitoring manager is already running")
	}

	mm.ctx, mm.cancel = context.WithCancel(context.Background())
	mm.running = true

	// Start metrics collection
	go mm.metricsCollectionLoop()

	// Start alert evaluation
	go mm.alertEvaluationLoop()

	// Start performance optimization
	go mm.optimizationLoop()

	fmt.Println("Monitoring manager started successfully")
	return nil
}

// Stop stops the monitoring manager
func (mm *MonitoringManager) Stop() error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if !mm.running {
		return fmt.Errorf("monitoring manager is not running")
	}

	mm.cancel()
	mm.running = false

	fmt.Println("Monitoring manager stopped")
	return nil
}

// IsRunning returns whether the monitoring manager is running
func (mm *MonitoringManager) IsRunning() bool {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.running
}

// metricsCollectionLoop runs the metrics collection loop
func (mm *MonitoringManager) metricsCollectionLoop() {
	ticker := time.NewTicker(mm.collectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-mm.ctx.Done():
			return
		case <-ticker.C:
			mm.metrics.CollectSystemMetrics()
		}
	}
}

// alertEvaluationLoop runs the alert evaluation loop
func (mm *MonitoringManager) alertEvaluationLoop() {
	ticker := time.NewTicker(mm.alertInterval)
	defer ticker.Stop()

	for {
		select {
		case <-mm.ctx.Done():
			return
		case <-ticker.C:
			if err := mm.alertManager.EvaluateRules(); err != nil {
				fmt.Printf("Error evaluating alert rules: %v\n", err)
			}
		}
	}
}

// optimizationLoop runs the optimization loop
func (mm *MonitoringManager) optimizationLoop() {
	ticker := time.NewTicker(mm.optimizeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-mm.ctx.Done():
			return
		case <-ticker.C:
			if err := mm.optimizer.RunOptimization(); err != nil {
				fmt.Printf("Error running optimization: %v\n", err)
			}
		}
	}
}

// GetMetrics returns the metrics collector
func (mm *MonitoringManager) GetMetrics() *MetricsCollector {
	return mm.metrics
}

// GetAlertManager returns the alert manager
func (mm *MonitoringManager) GetAlertManager() *AlertManager {
	return mm.alertManager
}

// GetOptimizer returns the performance optimizer
func (mm *MonitoringManager) GetOptimizer() *PerformanceOptimizer {
	return mm.optimizer
}

// GetHealthStatus returns the current system health status
func (mm *MonitoringManager) GetHealthStatus() HealthStatus {
	metrics := mm.metrics.GetMetricsSummary()
	activeAlerts := mm.alertManager.GetActiveAlerts()

	// Calculate health score and status
	score := 100
	var issues []HealthIssue
	status := "healthy"

	// Check memory usage
	memoryUsageMB := float64(metrics.CurrentMemoryUsage) / (1024 * 1024)
	if memoryUsageMB > 500 {
		score -= 15
		issues = append(issues, HealthIssue{
			Type:       "memory",
			Severity:   "warning",
			Message:    "High memory usage",
			Metric:     "memory_usage",
			Value:      fmt.Sprintf("%.2f MB", memoryUsageMB),
			Threshold:  "500 MB",
			Suggestion: "Consider optimizing memory usage or running garbage collection",
		})
		if status == "healthy" {
			status = "warning"
		}
	}

	// Check success rate
	if metrics.SuccessRate < 95 {
		score -= 20
		issues = append(issues, HealthIssue{
			Type:       "reliability",
			Severity:   "warning",
			Message:    "Low success rate",
			Metric:     "success_rate",
			Value:      fmt.Sprintf("%.2f%%", metrics.SuccessRate),
			Threshold:  "95%",
			Suggestion: "Review error logs and improve error handling",
		})
		if status == "healthy" {
			status = "warning"
		}
	}

	// Check goroutine count
	if metrics.GoroutineCount > 1000 {
		score -= 10
		issues = append(issues, HealthIssue{
			Type:       "concurrency",
			Severity:   "warning",
			Message:    "High goroutine count",
			Metric:     "goroutine_count",
			Value:      fmt.Sprintf("%d", metrics.GoroutineCount),
			Threshold:  "1000",
			Suggestion: "Check for goroutine leaks",
		})
		if status == "healthy" {
			status = "warning"
		}
	}

	// Check response time
	if metrics.AverageResponseTime > 2*time.Second {
		score -= 10
		issues = append(issues, HealthIssue{
			Type:       "performance",
			Severity:   "warning",
			Message:    "High response time",
			Metric:     "response_time",
			Value:      metrics.AverageResponseTime.String(),
			Threshold:  "2s",
			Suggestion: "Optimize API endpoints and database queries",
		})
		if status == "healthy" {
			status = "warning"
		}
	}

	// Check critical alerts
	criticalAlerts := 0
	for _, alert := range activeAlerts {
		if alert.Severity == "critical" {
			criticalAlerts++
		}
	}

	if criticalAlerts > 0 {
		score -= 30
		status = "critical"
		issues = append(issues, HealthIssue{
			Type:       "alerts",
			Severity:   "critical",
			Message:    fmt.Sprintf("%d critical alerts active", criticalAlerts),
			Suggestion: "Address critical alerts immediately",
		})
	} else if len(activeAlerts) > 0 {
		score -= 5
		if status == "healthy" {
			status = "warning"
		}
	}

	if score < 0 {
		score = 0
	}

	details := map[string]interface{}{
		"activeAlertsCount":    len(activeAlerts),
		"criticalAlertsCount":  criticalAlerts,
		"memoryUsageMB":        memoryUsageMB,
		"cpuCount":             runtime.NumCPU(),
		"goVersion":            runtime.Version(),
	}

	return HealthStatus{
		Status:       status,
		Score:        score,
		Issues:       issues,
		Metrics:      metrics,
		LastChecked:  time.Now(),
		Uptime:       metrics.Uptime,
		SystemInfo:   mm.systemInfo,
		ActiveAlerts: activeAlerts,
		Details:      details,
	}
}

// GetDashboard returns comprehensive dashboard data
func (mm *MonitoringManager) GetDashboard() MonitoringDashboard {
	return MonitoringDashboard{
		Health:           mm.GetHealthStatus(),
		Metrics:          mm.metrics.GetMetricsSummary(),
		RecentExecutions: mm.metrics.GetExecutionHistory(10),
		RecentErrors:     mm.metrics.GetErrorHistory(10),
		PerformanceData:  mm.metrics.GetPerformanceHistory(20),
		ActiveAlerts:     mm.alertManager.GetActiveAlerts(),
		Recommendations:  mm.optimizer.GetRecommendations(),
		SystemInfo:       mm.systemInfo,
		Timestamp:        time.Now(),
	}
}

// RecordWorkflowExecution records a workflow execution metric
func (mm *MonitoringManager) RecordWorkflowExecution(metric ExecutionMetric) {
	mm.metrics.RecordWorkflowExecution(metric)
}

// RecordError records an error metric
func (mm *MonitoringManager) RecordError(errorMetric ErrorMetric) {
	mm.metrics.RecordError(errorMetric)
}

// RecordHTTPRequest records an HTTP request metric
func (mm *MonitoringManager) RecordHTTPRequest(responseTime time.Duration, success bool) {
	mm.metrics.RecordHTTPRequest(responseTime, success)
}

// AddAlertHandler adds an alert handler
func (mm *MonitoringManager) AddAlertHandler(handler AlertHandler) {
	mm.alertManager.AddHandler(handler)
}

// AddAlertRule adds an alert rule
func (mm *MonitoringManager) AddAlertRule(rule *AlertRule) error {
	return mm.alertManager.AddRule(rule)
}

// collectSystemInfo collects system information
func collectSystemInfo() SystemInfo {
	hostname, _ := os.Hostname()
	wd, _ := os.Getwd()

	return SystemInfo{
		Version:          "0.2.0",
		BuildDate:        "2024-01-22",
		GoVersion:        runtime.Version(),
		Platform:         fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		Uptime:           0, // Will be calculated from start time
		StartTime:        time.Now(),
		ProcessID:        os.Getpid(),
		WorkingDirectory: wd,
		HostName:         hostname,
		MemoryLimit:      0, // Will be set if available
		CPUCount:         runtime.NumCPU(),
	}
}

// GetSystemStats returns detailed system statistics
func (mm *MonitoringManager) GetSystemStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"memory": map[string]interface{}{
			"alloc":        m.Alloc,
			"totalAlloc":   m.TotalAlloc,
			"sys":          m.Sys,
			"heapAlloc":    m.HeapAlloc,
			"heapSys":      m.HeapSys,
			"heapInuse":    m.HeapInuse,
			"heapObjects":  m.HeapObjects,
			"stackInuse":   m.StackInuse,
			"stackSys":     m.StackSys,
			"nextGC":       m.NextGC,
			"lastGC":       m.LastGC,
			"numGC":        m.NumGC,
			"gcCPUFraction": m.GCCPUFraction,
		},
		"runtime": map[string]interface{}{
			"version":      runtime.Version(),
			"numGoroutine": runtime.NumGoroutine(),
			"numCPU":       runtime.NumCPU(),
			"maxProcs":     runtime.GOMAXPROCS(0),
			"compiler":     runtime.Compiler,
		},
		"system": mm.systemInfo,
	}
}

// TriggerOptimization manually triggers optimization
func (mm *MonitoringManager) TriggerOptimization() error {
	return mm.optimizer.RunOptimization()
}

// TriggerGC manually triggers garbage collection
func (mm *MonitoringManager) TriggerGC() {
	runtime.GC()
}

// Reset resets all monitoring data (useful for testing)
func (mm *MonitoringManager) Reset() {
	mm.metrics.Reset()
}

// SetActiveWorkflows sets the number of active workflows
func (mm *MonitoringManager) SetActiveWorkflows(count int64) {
	mm.metrics.SetActiveWorkflows(count)
}

// IncrementActiveExecutions increments active executions
func (mm *MonitoringManager) IncrementActiveExecutions() {
	mm.metrics.IncrementActiveExecutions()
}

// DecrementActiveExecutions decrements active executions
func (mm *MonitoringManager) DecrementActiveExecutions() {
	mm.metrics.DecrementActiveExecutions()
}