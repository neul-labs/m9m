package monitoring

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

// PerformanceOptimizer provides automated performance optimizations
type PerformanceOptimizer struct {
	mu                sync.RWMutex
	metrics           *MetricsCollector
	alertManager      *AlertManager
	enabled           bool
	gcTuningEnabled   bool
	memoryThreshold   int64
	optimizationLog   []OptimizationAction
	maxLogSize        int
	lastOptimization  time.Time
	optimizationCooldown time.Duration
}

// OptimizationAction represents a performance optimization action
type OptimizationAction struct {
	Timestamp   time.Time `json:"timestamp"`
	Action      string    `json:"action"`
	Reason      string    `json:"reason"`
	MetricsBefore map[string]interface{} `json:"metricsBefore"`
	MetricsAfter  map[string]interface{} `json:"metricsAfter"`
	Success     bool      `json:"success"`
	Error       string    `json:"error,omitempty"`
	Impact      string    `json:"impact"` // low, medium, high
}

// OptimizationRecommendation represents a suggested optimization
type OptimizationRecommendation struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Priority    string    `json:"priority"` // low, medium, high, critical
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Action      string    `json:"action"`
	EstimatedImpact string `json:"estimatedImpact"`
	CreatedAt   time.Time `json:"createdAt"`
	Applied     bool      `json:"applied"`
	AppliedAt   *time.Time `json:"appliedAt,omitempty"`
}

// PerformanceTuning contains various tuning parameters
type PerformanceTuning struct {
	GCTargetPercent     int           `json:"gcTargetPercent"`
	MaxProcs           int           `json:"maxProcs"`
	MemoryLimit        int64         `json:"memoryLimit"`
	GoroutineLimit     int           `json:"goroutineLimit"`
	ConnectionPoolSize int           `json:"connectionPoolSize"`
	HTTPTimeout        time.Duration `json:"httpTimeout"`
	WorkerPoolSize     int           `json:"workerPoolSize"`
	CacheSize          int           `json:"cacheSize"`
}

// NewPerformanceOptimizer creates a new performance optimizer
func NewPerformanceOptimizer(metrics *MetricsCollector, alertManager *AlertManager) *PerformanceOptimizer {
	return &PerformanceOptimizer{
		metrics:           metrics,
		alertManager:      alertManager,
		enabled:           true,
		gcTuningEnabled:   true,
		memoryThreshold:   500 * 1024 * 1024, // 500MB
		maxLogSize:        100,
		optimizationCooldown: 5 * time.Minute, // Wait 5 minutes between optimizations
	}
}

// Enable enables the performance optimizer
func (po *PerformanceOptimizer) Enable() {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.enabled = true
}

// Disable disables the performance optimizer
func (po *PerformanceOptimizer) Disable() {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.enabled = false
}

// IsEnabled returns whether the optimizer is enabled
func (po *PerformanceOptimizer) IsEnabled() bool {
	po.mu.RLock()
	defer po.mu.RUnlock()
	return po.enabled
}

// RunOptimization performs automatic performance optimizations
func (po *PerformanceOptimizer) RunOptimization() error {
	if !po.enabled {
		return nil
	}

	// Check cooldown period
	po.mu.RLock()
	if time.Since(po.lastOptimization) < po.optimizationCooldown {
		po.mu.RUnlock()
		return nil
	}
	po.mu.RUnlock()

	// Get current metrics
	currentMetrics := po.metrics.GetMetricsSummary()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metricsBefore := map[string]interface{}{
		"memoryUsage":   m.Alloc,
		"gcPauses":      m.PauseNs,
		"numGoroutines": runtime.NumGoroutine(),
		"numGC":         m.NumGC,
	}

	optimizationsApplied := false

	// Memory optimization
	if m.Alloc > uint64(po.memoryThreshold) {
		if err := po.optimizeMemory(); err != nil {
			po.logOptimization("memory_gc", "High memory usage detected", metricsBefore, nil, false, err.Error(), "high")
		} else {
			optimizationsApplied = true
			po.logOptimization("memory_gc", "High memory usage detected", metricsBefore, nil, true, "", "high")
		}
	}

	// GC tuning
	if po.gcTuningEnabled && po.shouldTuneGC(currentMetrics) {
		if err := po.tuneGarbageCollector(); err != nil {
			po.logOptimization("gc_tuning", "GC performance optimization", metricsBefore, nil, false, err.Error(), "medium")
		} else {
			optimizationsApplied = true
			po.logOptimization("gc_tuning", "GC performance optimization", metricsBefore, nil, true, "", "medium")
		}
	}

	// Goroutine leak detection
	if runtime.NumGoroutine() > 1000 {
		po.detectGoroutineLeaks()
		po.logOptimization("goroutine_analysis", "High goroutine count detected", metricsBefore, nil, true, "", "medium")
	}

	if optimizationsApplied {
		po.mu.Lock()
		po.lastOptimization = time.Now()
		po.mu.Unlock()

		// Give some time for optimizations to take effect
		time.Sleep(1 * time.Second)

		// Capture metrics after optimization
		runtime.ReadMemStats(&m)
		metricsAfter := map[string]interface{}{
			"memoryUsage":   m.Alloc,
			"gcPauses":      m.PauseNs,
			"numGoroutines": runtime.NumGoroutine(),
			"numGC":         m.NumGC,
		}

		// Update the last optimization log entry with after metrics
		po.updateLastOptimizationMetrics(metricsAfter)
	}

	return nil
}

// optimizeMemory performs memory optimization
func (po *PerformanceOptimizer) optimizeMemory() error {
	// Force garbage collection
	runtime.GC()

	// Free OS memory
	debug.FreeOSMemory()

	return nil
}

// tuneGarbageCollector adjusts GC settings for better performance
func (po *PerformanceOptimizer) tuneGarbageCollector() error {
	// Get current GC stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Adjust GOGC based on current memory usage and allocation rate
	currentGOGC := debug.SetGCPercent(-1) // Get current value
	debug.SetGCPercent(currentGOGC)       // Restore it

	var newGOGC int
	if m.Alloc > 100*1024*1024 { // > 100MB
		// More aggressive GC for high memory usage
		newGOGC = 50
	} else if m.Alloc < 10*1024*1024 { // < 10MB
		// Less aggressive GC for low memory usage
		newGOGC = 200
	} else {
		// Default tuning
		newGOGC = 100
	}

	if newGOGC != currentGOGC {
		debug.SetGCPercent(newGOGC)
	}

	return nil
}

// shouldTuneGC determines if GC tuning is needed
func (po *PerformanceOptimizer) shouldTuneGC(metrics MetricsSummary) bool {
	// Tune GC if:
	// 1. Memory usage is high
	// 2. Average execution time is high (might be due to GC pauses)
	// 3. HTTP response time is high

	return metrics.CurrentMemoryUsage > po.memoryThreshold ||
		metrics.AverageExecutionTime > 5*time.Second ||
		metrics.AverageResponseTime > 1*time.Second
}

// detectGoroutineLeaks detects potential goroutine leaks
func (po *PerformanceOptimizer) detectGoroutineLeaks() {
	// Create a goroutine dump for analysis
	buf := make([]byte, 1<<20) // 1MB buffer
	n := runtime.Stack(buf, true)
	stackTrace := string(buf[:n])

	// Simple leak detection: look for common patterns
	leakPatterns := []string{
		"net/http.(*persistConn).writeLoop",
		"net/http.(*persistConn).readLoop",
		"database/sql.(*DB).connectionOpener",
		"sync.(*WaitGroup).Wait",
	}

	detectedLeaks := make(map[string]int)
	for _, pattern := range leakPatterns {
		count := 0
		for i := 0; i < len(stackTrace); {
			if idx := findString(stackTrace[i:], pattern); idx != -1 {
				count++
				i = i + idx + len(pattern)
			} else {
				break
			}
		}
		if count > 10 { // Threshold for considering it a leak
			detectedLeaks[pattern] = count
		}
	}

	if len(detectedLeaks) > 0 {
		fmt.Printf("Potential goroutine leaks detected: %v\n", detectedLeaks)
	}
}

// findString is a simple string search function
func findString(haystack, needle string) int {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

// GetRecommendations returns performance optimization recommendations
func (po *PerformanceOptimizer) GetRecommendations() []OptimizationRecommendation {
	recommendations := make([]OptimizationRecommendation, 0)
	currentMetrics := po.metrics.GetMetricsSummary()

	// Memory recommendations
	if currentMetrics.CurrentMemoryUsage > po.memoryThreshold {
		recommendations = append(recommendations, OptimizationRecommendation{
			ID:          "mem-high",
			Type:        "memory",
			Priority:    "high",
			Title:       "High Memory Usage",
			Description: fmt.Sprintf("Current memory usage (%.2f MB) exceeds threshold (%.2f MB)",
				float64(currentMetrics.CurrentMemoryUsage)/(1024*1024),
				float64(po.memoryThreshold)/(1024*1024)),
			Action:      "Consider running garbage collection or optimizing memory usage in workflows",
			EstimatedImpact: "High - Will reduce memory pressure and improve performance",
			CreatedAt:   time.Now(),
		})
	}

	// Performance recommendations
	if currentMetrics.AverageExecutionTime > 10*time.Second {
		recommendations = append(recommendations, OptimizationRecommendation{
			ID:          "exec-slow",
			Type:        "performance",
			Priority:    "medium",
			Title:       "Slow Workflow Execution",
			Description: fmt.Sprintf("Average execution time (%.2f seconds) is high",
				currentMetrics.AverageExecutionTime.Seconds()),
			Action:      "Review workflow complexity, optimize node configurations, or consider parallel execution",
			EstimatedImpact: "Medium - Will improve workflow response times",
			CreatedAt:   time.Now(),
		})
	}

	// Goroutine recommendations
	if currentMetrics.GoroutineCount > 1000 {
		recommendations = append(recommendations, OptimizationRecommendation{
			ID:          "goroutine-high",
			Type:        "concurrency",
			Priority:    "medium",
			Title:       "High Goroutine Count",
			Description: fmt.Sprintf("Current goroutine count (%d) is high", currentMetrics.GoroutineCount),
			Action:      "Check for goroutine leaks, review concurrent operations, consider connection pooling",
			EstimatedImpact: "Medium - Will reduce resource usage and improve stability",
			CreatedAt:   time.Now(),
		})
	}

	// Success rate recommendations
	if currentMetrics.SuccessRate < 95 {
		recommendations = append(recommendations, OptimizationRecommendation{
			ID:          "success-low",
			Type:        "reliability",
			Priority:    "high",
			Title:       "Low Success Rate",
			Description: fmt.Sprintf("Workflow success rate (%.2f%%) is below optimal", currentMetrics.SuccessRate),
			Action:      "Review error logs, improve error handling, add retry mechanisms",
			EstimatedImpact: "High - Will improve system reliability",
			CreatedAt:   time.Now(),
		})
	}

	// HTTP performance recommendations
	if currentMetrics.AverageResponseTime > 2*time.Second {
		recommendations = append(recommendations, OptimizationRecommendation{
			ID:          "http-slow",
			Type:        "api",
			Priority:    "medium",
			Title:       "Slow API Response Time",
			Description: fmt.Sprintf("Average API response time (%.2f seconds) is high",
				currentMetrics.AverageResponseTime.Seconds()),
			Action:      "Optimize API endpoints, add caching, review database queries",
			EstimatedImpact: "Medium - Will improve API performance",
			CreatedAt:   time.Now(),
		})
	}

	return recommendations
}

// ApplyRecommendation applies a specific optimization recommendation
func (po *PerformanceOptimizer) ApplyRecommendation(recommendationID string) error {
	switch recommendationID {
	case "mem-high":
		return po.optimizeMemory()
	case "goroutine-high":
		po.detectGoroutineLeaks()
		return nil
	default:
		return fmt.Errorf("unknown recommendation ID: %s", recommendationID)
	}
}

// GetCurrentTuning returns current performance tuning parameters
func (po *PerformanceOptimizer) GetCurrentTuning() PerformanceTuning {
	currentGOGC := debug.SetGCPercent(-1)
	debug.SetGCPercent(currentGOGC)

	return PerformanceTuning{
		GCTargetPercent:     currentGOGC,
		MaxProcs:           runtime.GOMAXPROCS(0),
		MemoryLimit:        po.memoryThreshold,
		GoroutineLimit:     1000,
		ConnectionPoolSize: 25, // Default from database packages
		HTTPTimeout:        30 * time.Second,
		WorkerPoolSize:     runtime.NumCPU(),
		CacheSize:          100,
	}
}

// UpdateTuning updates performance tuning parameters
func (po *PerformanceOptimizer) UpdateTuning(tuning PerformanceTuning) error {
	po.mu.Lock()
	defer po.mu.Unlock()

	// Apply GC tuning
	if tuning.GCTargetPercent > 0 && tuning.GCTargetPercent <= 1000 {
		debug.SetGCPercent(tuning.GCTargetPercent)
	}

	// Apply GOMAXPROCS tuning
	if tuning.MaxProcs > 0 && tuning.MaxProcs <= runtime.NumCPU()*4 {
		runtime.GOMAXPROCS(tuning.MaxProcs)
	}

	// Update memory threshold
	if tuning.MemoryLimit > 0 {
		po.memoryThreshold = tuning.MemoryLimit
	}

	po.logOptimizationLocked("tuning_update", "Performance tuning parameters updated",
		map[string]interface{}{"tuning": tuning}, nil, true, "", "medium")

	return nil
}

// GetOptimizationHistory returns the optimization action history
func (po *PerformanceOptimizer) GetOptimizationHistory(limit int) []OptimizationAction {
	po.mu.RLock()
	defer po.mu.RUnlock()

	if limit <= 0 || limit > len(po.optimizationLog) {
		limit = len(po.optimizationLog)
	}

	start := len(po.optimizationLog) - limit
	if start < 0 {
		start = 0
	}

	result := make([]OptimizationAction, limit)
	copy(result, po.optimizationLog[start:])
	return result
}

// logOptimization logs an optimization action (acquires lock internally)
func (po *PerformanceOptimizer) logOptimization(action, reason string, metricsBefore, metricsAfter map[string]interface{}, success bool, errorMsg, impact string) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.logOptimizationLocked(action, reason, metricsBefore, metricsAfter, success, errorMsg, impact)
}

// logOptimizationLocked is the same as logOptimization but assumes the caller already holds po.mu.
func (po *PerformanceOptimizer) logOptimizationLocked(action, reason string, metricsBefore, metricsAfter map[string]interface{}, success bool, errorMsg, impact string) {
	optimization := OptimizationAction{
		Timestamp:     time.Now(),
		Action:        action,
		Reason:        reason,
		MetricsBefore: metricsBefore,
		MetricsAfter:  metricsAfter,
		Success:       success,
		Error:         errorMsg,
		Impact:        impact,
	}

	po.optimizationLog = append(po.optimizationLog, optimization)
	if len(po.optimizationLog) > po.maxLogSize {
		po.optimizationLog = po.optimizationLog[1:]
	}
}

// updateLastOptimizationMetrics updates the metrics after for the last optimization
func (po *PerformanceOptimizer) updateLastOptimizationMetrics(metricsAfter map[string]interface{}) {
	po.mu.Lock()
	defer po.mu.Unlock()

	if len(po.optimizationLog) > 0 {
		po.optimizationLog[len(po.optimizationLog)-1].MetricsAfter = metricsAfter
	}
}

// StartAutoOptimization starts automatic optimization in a background goroutine
func (po *PerformanceOptimizer) StartAutoOptimization(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := po.RunOptimization(); err != nil {
					fmt.Printf("Auto-optimization error: %v\n", err)
				}
			}
		}
	}()
}

// GetMemoryProfile returns current memory profile information
func (po *PerformanceOptimizer) GetMemoryProfile() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"alloc":         m.Alloc,
		"totalAlloc":    m.TotalAlloc,
		"sys":           m.Sys,
		"lookups":       m.Lookups,
		"mallocs":       m.Mallocs,
		"frees":         m.Frees,
		"heapAlloc":     m.HeapAlloc,
		"heapSys":       m.HeapSys,
		"heapIdle":      m.HeapIdle,
		"heapInuse":     m.HeapInuse,
		"heapReleased":  m.HeapReleased,
		"heapObjects":   m.HeapObjects,
		"stackInuse":    m.StackInuse,
		"stackSys":      m.StackSys,
		"mSpanInuse":    m.MSpanInuse,
		"mSpanSys":      m.MSpanSys,
		"mCacheInuse":   m.MCacheInuse,
		"mCacheSys":     m.MCacheSys,
		"buckHashSys":   m.BuckHashSys,
		"gcSys":         m.GCSys,
		"otherSys":      m.OtherSys,
		"nextGC":        m.NextGC,
		"lastGC":        m.LastGC,
		"pauseTotalNs":  m.PauseTotalNs,
		"pauseNs":       m.PauseNs,
		"numGC":         m.NumGC,
		"numForcedGC":   m.NumForcedGC,
		"gcCPUFraction": m.GCCPUFraction,
	}
}

// SetMemoryThreshold sets the memory threshold for optimizations
func (po *PerformanceOptimizer) SetMemoryThreshold(threshold int64) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.memoryThreshold = threshold
}

// SetGCTuningEnabled enables or disables automatic GC tuning
func (po *PerformanceOptimizer) SetGCTuningEnabled(enabled bool) {
	po.mu.Lock()
	defer po.mu.Unlock()
	po.gcTuningEnabled = enabled
}