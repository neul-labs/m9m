# Monitoring and Observability

n8n-go provides comprehensive monitoring and observability features built for production environments. This includes Prometheus metrics, OpenTelemetry tracing, structured logging, and health monitoring.

## Overview

### Built-in Monitoring Features
- **Prometheus Metrics**: Performance and business metrics
- **OpenTelemetry Tracing**: Distributed tracing across workflow executions
- **Health Checks**: Readiness and liveness endpoints
- **Structured Logging**: JSON-formatted logs with correlation IDs
- **Performance Profiling**: Go runtime metrics

### Monitoring Architecture
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Application   │───▶│   Prometheus    │───▶│    Grafana      │
│     Metrics     │    │    Server       │    │   Dashboard     │
└─────────────────┘    └─────────────────┘    └─────────────────┘

┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Distributed    │───▶│     Jaeger      │───▶│   Trace UI      │
│    Traces       │    │   Collector     │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘

┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Application    │───▶│   Log Shipper   │───▶│  Log Analytics  │
│     Logs        │    │  (Fluentd/etc)  │    │   (ELK Stack)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Prometheus Metrics

### Available Metrics

#### Workflow Metrics
```
# Workflow execution count by status
n8n_go_workflow_executions_total{workflow_id="", workflow_name="", status="", mode=""} counter

# Workflow execution duration
n8n_go_workflow_duration_seconds{workflow_id="", workflow_name=""} histogram

# Workflow errors by type
n8n_go_workflow_errors_total{workflow_id="", workflow_name="", error_type=""} counter

# Concurrent workflow executions
n8n_go_workflow_concurrent_executions gauge
```

#### Node Metrics
```
# Node execution count
n8n_go_node_executions_total{node_type="", node_name="", status=""} counter

# Node execution duration
n8n_go_node_duration_seconds{node_type="", node_name=""} histogram

# Node errors
n8n_go_node_errors_total{node_type="", node_name="", error_type=""} counter
```

#### System Metrics
```
# Queue size
n8n_go_system_queue_size gauge

# Database connections
n8n_go_system_database_connections gauge

# Memory usage
n8n_go_system_memory_bytes gauge

# CPU usage percentage
n8n_go_system_cpu_percent gauge

# Goroutine count
n8n_go_system_goroutines gauge
```

#### HTTP Metrics
```
# HTTP request count
n8n_go_http_requests_total{method="", path="", status=""} counter

# HTTP request duration
n8n_go_http_duration_seconds{method="", path=""} histogram
```

### Configuration

#### Enable Metrics
```yaml
monitoring:
  metrics_port: 9090
  metrics_path: "/metrics"
  enable_metrics: true
```

#### Environment Variables
```bash
N8N_GO_METRICS_PORT=9090
N8N_GO_ENABLE_METRICS=true
```

### Prometheus Configuration
```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'n8n-go'
    static_configs:
      - targets: ['n8n-go:9090']
    metrics_path: '/metrics'
    scrape_interval: 10s
    scrape_timeout: 10s
    metrics_path: /metrics
```

## OpenTelemetry Tracing

### Trace Configuration
```yaml
monitoring:
  tracing:
    endpoint: "http://jaeger:14268/api/traces"
    service_name: "n8n-go"
    sampling_rate: 1.0
    exporter_type: "jaeger"  # or "otlp"
```

### Environment Variables
```bash
N8N_GO_TRACING_ENDPOINT=http://jaeger:14268/api/traces
N8N_GO_TRACING_SERVICE_NAME=n8n-go
N8N_GO_TRACING_SAMPLING_RATE=1.0
```

### Trace Structure
```
Workflow Execution Span
├── Node: HTTP Request Span
│   ├── HTTP Client Request Span
│   └── Credential Resolution Span
├── Node: Database Query Span
│   ├── Database Connection Span
│   └── Query Execution Span
└── Node: Data Transformation Span
    └── Expression Evaluation Span
```

### Custom Instrumentation
```go
// Start a custom span
ctx, span := tracingManager.StartSpan(ctx, "custom-operation")
defer span.End()

// Add attributes
span.SetAttributes(
    attribute.String("operation.type", "data-processing"),
    attribute.Int("record.count", recordCount),
)

// Record error
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
}
```

## Health Monitoring

### Health Endpoints

#### Health Check
```bash
curl http://localhost:8080/health
```
Response:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "checks": {
    "database": "healthy",
    "queue": "healthy",
    "redis": "healthy"
  }
}
```

#### Readiness Check
```bash
curl http://localhost:8080/ready
```
Response:
```json
{
  "status": "ready",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Kubernetes Health Probes
```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 30
  timeoutSeconds: 10
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 5
  failureThreshold: 3
```

## Grafana Dashboards

### Workflow Performance Dashboard
```json
{
  "dashboard": {
    "title": "n8n-go Workflow Performance",
    "panels": [
      {
        "title": "Workflow Execution Rate",
        "type": "stat",
        "targets": [
          {
            "expr": "rate(n8n_go_workflow_executions_total[5m])"
          }
        ]
      },
      {
        "title": "Workflow Success Rate",
        "type": "stat",
        "targets": [
          {
            "expr": "rate(n8n_go_workflow_executions_total{status=\"success\"}[5m]) / rate(n8n_go_workflow_executions_total[5m]) * 100"
          }
        ]
      },
      {
        "title": "Average Execution Time",
        "type": "stat",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(n8n_go_workflow_duration_seconds_bucket[5m]))"
          }
        ]
      },
      {
        "title": "Queue Size",
        "type": "graph",
        "targets": [
          {
            "expr": "n8n_go_system_queue_size"
          }
        ]
      }
    ]
  }
}
```

### System Resource Dashboard
```json
{
  "dashboard": {
    "title": "n8n-go System Resources",
    "panels": [
      {
        "title": "Memory Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "n8n_go_system_memory_bytes / 1024 / 1024"
          }
        ]
      },
      {
        "title": "CPU Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "n8n_go_system_cpu_percent"
          }
        ]
      },
      {
        "title": "Goroutines",
        "type": "graph",
        "targets": [
          {
            "expr": "n8n_go_system_goroutines"
          }
        ]
      }
    ]
  }
}
```

## Alerting

### Prometheus Alert Rules
```yaml
# alerts.yml
groups:
- name: n8n-go.rules
  rules:
  # High error rate
  - alert: HighWorkflowErrorRate
    expr: rate(n8n_go_workflow_errors_total[5m]) > 0.1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High workflow error rate detected"
      description: "Workflow error rate is {{ $value }} errors/second"

  # High queue size
  - alert: HighQueueSize
    expr: n8n_go_system_queue_size > 1000
    for: 10m
    labels:
      severity: warning
    annotations:
      summary: "Queue size is high"
      description: "Queue size is {{ $value }} jobs"

  # Memory usage alert
  - alert: HighMemoryUsage
    expr: n8n_go_system_memory_bytes > 1073741824  # 1GB
    for: 15m
    labels:
      severity: warning
    annotations:
      summary: "High memory usage"
      description: "Memory usage is {{ $value | humanize1024 }}B"

  # Service down
  - alert: ServiceDown
    expr: up{job="n8n-go"} == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "n8n-go service is down"
      description: "n8n-go instance {{ $labels.instance }} is down"
```

### Alertmanager Configuration
```yaml
# alertmanager.yml
global:
  smtp_smarthost: 'localhost:587'

route:
  group_by: ['alertname']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'web.hook'

receivers:
- name: 'web.hook'
  email_configs:
  - to: 'admin@company.com'
    from: 'alerts@company.com'
    subject: 'n8n-go Alert: {{ .GroupLabels.alertname }}'
    body: |
      {{ range .Alerts }}
      Alert: {{ .Annotations.summary }}
      Description: {{ .Annotations.description }}
      {{ end }}

  slack_configs:
  - api_url: 'YOUR_SLACK_WEBHOOK_URL'
    channel: '#alerts'
    title: 'n8n-go Alert'
    text: '{{ range .Alerts }}{{ .Annotations.summary }}{{ end }}'
```

## Logging

### Log Configuration
```yaml
logging:
  level: "info"  # debug, info, warn, error
  format: "json"  # json, text
  output: "stdout"  # stdout, file
  file_path: "/var/log/n8n-go.log"
  max_size: 100  # MB
  max_backups: 5
  max_age: 30  # days
```

### Structured Logging Format
```json
{
  "timestamp": "2024-01-15T10:30:00.123Z",
  "level": "info",
  "message": "Workflow execution completed",
  "workflow_id": "wf_123",
  "execution_id": "exec_456",
  "duration_ms": 1250,
  "status": "success",
  "node_count": 5,
  "trace_id": "abc123def456",
  "span_id": "789ghi012jkl"
}
```

### Log Aggregation with ELK Stack
```yaml
# filebeat.yml
filebeat.inputs:
- type: container
  paths:
    - '/var/lib/docker/containers/*/*.log'
  processors:
  - add_kubernetes_metadata:
      host: ${NODE_NAME}
      matchers:
      - logs_path:
          logs_path: "/var/lib/docker/containers/"

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
  index: "n8n-go-logs-%{+yyyy.MM.dd}"
```

## Performance Profiling

### Enable Profiling
```bash
# Start with profiling enabled
N8N_GO_ENABLE_PPROF=true n8n-go serve
```

### Profiling Endpoints
```bash
# CPU profile
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof

# Memory profile
curl http://localhost:6060/debug/pprof/heap > heap.prof

# Goroutine profile
curl http://localhost:6060/debug/pprof/goroutine > goroutine.prof

# Analyze profiles
go tool pprof cpu.prof
go tool pprof heap.prof
```

## Monitoring Best Practices

### 1. Metric Selection
- Focus on key business metrics (workflow success rate, execution time)
- Monitor resource utilization (CPU, memory, queue size)
- Track error rates and types
- Monitor external service dependencies

### 2. Alert Strategy
- Use multi-level alerting (warning, critical)
- Set appropriate thresholds based on SLA requirements
- Implement alert fatigue prevention
- Use runbooks for common scenarios

### 3. Dashboard Design
- Create role-specific dashboards (operations, development, business)
- Use appropriate time ranges and aggregations
- Implement drill-down capabilities
- Include context and documentation

### 4. Trace Analysis
- Sample traces appropriately (1.0 for development, 0.1 for production)
- Correlate traces with metrics and logs
- Use trace data for performance optimization
- Implement distributed context propagation

### 5. Log Management
- Use structured logging consistently
- Include correlation IDs for request tracing
- Implement log level controls
- Set up log rotation and retention policies