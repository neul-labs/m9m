package monitoring

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusCollector handles Prometheus metrics
type PrometheusCollector struct {
	// Workflow metrics
	workflowExecutions      *prometheus.CounterVec
	workflowDuration        *prometheus.HistogramVec
	workflowErrors          *prometheus.CounterVec
	concurrentWorkflows     prometheus.Gauge

	// Node metrics
	nodeExecutions          *prometheus.CounterVec
	nodeDuration            *prometheus.HistogramVec
	nodeErrors              *prometheus.CounterVec

	// System metrics
	queueSize               prometheus.Gauge
	databaseConnections     prometheus.Gauge
	httpRequests            *prometheus.CounterVec
	httpDuration            *prometheus.HistogramVec
	memoryUsage             prometheus.Gauge
	cpuUsage                prometheus.Gauge
	goroutines              prometheus.Gauge
}

// NewPrometheusCollector creates a new Prometheus metrics collector
func NewPrometheusCollector() *PrometheusCollector {
	pc := &PrometheusCollector{
		// Workflow metrics
		workflowExecutions: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "m9m",
				Subsystem: "workflow",
				Name:      "executions_total",
				Help:      "Total number of workflow executions",
			},
			[]string{"workflow_id", "workflow_name", "status", "mode"},
		),
		workflowDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "m9m",
				Subsystem: "workflow",
				Name:      "duration_seconds",
				Help:      "Duration of workflow executions",
				Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300},
			},
			[]string{"workflow_id", "workflow_name"},
		),
		workflowErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "m9m",
				Subsystem: "workflow",
				Name:      "errors_total",
				Help:      "Total number of workflow errors",
			},
			[]string{"workflow_id", "workflow_name", "error_type"},
		),
		concurrentWorkflows: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "m9m",
				Subsystem: "workflow",
				Name:      "concurrent_executions",
				Help:      "Number of workflows currently executing",
			},
		),

		// Node metrics
		nodeExecutions: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "m9m",
				Subsystem: "node",
				Name:      "executions_total",
				Help:      "Total number of node executions",
			},
			[]string{"node_type", "node_name", "status"},
		),
		nodeDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "m9m",
				Subsystem: "node",
				Name:      "duration_seconds",
				Help:      "Duration of node executions",
				Buckets:   []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10},
			},
			[]string{"node_type", "node_name"},
		),
		nodeErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "m9m",
				Subsystem: "node",
				Name:      "errors_total",
				Help:      "Total number of node errors",
			},
			[]string{"node_type", "node_name", "error_type"},
		),

		// System metrics
		queueSize: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "m9m",
				Subsystem: "system",
				Name:      "queue_size",
				Help:      "Number of workflows in the execution queue",
			},
		),
		databaseConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "m9m",
				Subsystem: "system",
				Name:      "database_connections",
				Help:      "Number of active database connections",
			},
		),
		httpRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "m9m",
				Subsystem: "http",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		httpDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "m9m",
				Subsystem: "http",
				Name:      "duration_seconds",
				Help:      "Duration of HTTP requests",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		memoryUsage: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "m9m",
				Subsystem: "system",
				Name:      "memory_bytes",
				Help:      "Current memory usage in bytes",
			},
		),
		cpuUsage: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "m9m",
				Subsystem: "system",
				Name:      "cpu_percent",
				Help:      "Current CPU usage percentage",
			},
		),
		goroutines: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "m9m",
				Subsystem: "system",
				Name:      "goroutines",
				Help:      "Number of goroutines",
			},
		),
	}

	// Register all metrics
	prometheus.MustRegister(
		pc.workflowExecutions,
		pc.workflowDuration,
		pc.workflowErrors,
		pc.concurrentWorkflows,
		pc.nodeExecutions,
		pc.nodeDuration,
		pc.nodeErrors,
		pc.queueSize,
		pc.databaseConnections,
		pc.httpRequests,
		pc.httpDuration,
		pc.memoryUsage,
		pc.cpuUsage,
		pc.goroutines,
	)

	return pc
}

// StartServer starts the Prometheus metrics HTTP server
func (pc *PrometheusCollector) StartServer(port int) error {
	addr := fmt.Sprintf(":%d", port)
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/ready", readyHandler)

	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			fmt.Printf("Failed to start Prometheus metrics server: %v\n", err)
		}
	}()

	fmt.Printf("Prometheus metrics server started on port %d\n", port)
	return nil
}

// Workflow metrics methods

func (pc *PrometheusCollector) RecordWorkflowExecution(workflowID, workflowName, status, mode string) {
	pc.workflowExecutions.WithLabelValues(workflowID, workflowName, status, mode).Inc()
}

func (pc *PrometheusCollector) RecordWorkflowDuration(workflowID, workflowName string, duration time.Duration) {
	pc.workflowDuration.WithLabelValues(workflowID, workflowName).Observe(duration.Seconds())
}

func (pc *PrometheusCollector) RecordWorkflowError(workflowID, workflowName, errorType string) {
	pc.workflowErrors.WithLabelValues(workflowID, workflowName, errorType).Inc()
}

func (pc *PrometheusCollector) SetConcurrentWorkflows(count int) {
	pc.concurrentWorkflows.Set(float64(count))
}

func (pc *PrometheusCollector) IncrementConcurrentWorkflows() {
	pc.concurrentWorkflows.Inc()
}

func (pc *PrometheusCollector) DecrementConcurrentWorkflows() {
	pc.concurrentWorkflows.Dec()
}

// Node metrics methods

func (pc *PrometheusCollector) RecordNodeExecution(nodeType, nodeName, status string) {
	pc.nodeExecutions.WithLabelValues(nodeType, nodeName, status).Inc()
}

func (pc *PrometheusCollector) RecordNodeDuration(nodeType, nodeName string, duration time.Duration) {
	pc.nodeDuration.WithLabelValues(nodeType, nodeName).Observe(duration.Seconds())
}

func (pc *PrometheusCollector) RecordNodeError(nodeType, nodeName, errorType string) {
	pc.nodeErrors.WithLabelValues(nodeType, nodeName, errorType).Inc()
}

// System metrics methods

func (pc *PrometheusCollector) SetQueueSize(size int) {
	pc.queueSize.Set(float64(size))
}

func (pc *PrometheusCollector) SetDatabaseConnections(count int) {
	pc.databaseConnections.Set(float64(count))
}

func (pc *PrometheusCollector) SetMemoryUsage(bytes uint64) {
	pc.memoryUsage.Set(float64(bytes))
}

func (pc *PrometheusCollector) SetCPUUsage(percent float64) {
	pc.cpuUsage.Set(percent)
}

func (pc *PrometheusCollector) SetGoroutines(count int) {
	pc.goroutines.Set(float64(count))
}

// HTTP metrics methods

func (pc *PrometheusCollector) RecordHTTPRequest(method, path, status string) {
	pc.httpRequests.WithLabelValues(method, path, status).Inc()
}

func (pc *PrometheusCollector) RecordHTTPDuration(method, path string, duration time.Duration) {
	pc.httpDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

// HTTP middleware for automatic metrics collection
func (pc *PrometheusCollector) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create response wrapper to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Serve request
		next.ServeHTTP(wrapped, r)

		// Record metrics
		duration := time.Since(start)
		status := fmt.Sprintf("%d", wrapped.statusCode)
		pc.RecordHTTPRequest(r.Method, r.URL.Path, status)
		pc.RecordHTTPDuration(r.Method, r.URL.Path, duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ready"}`))
}