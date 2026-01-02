# Scaling m9m

Scale m9m to handle increased workflow volume and concurrency.

## Scaling Strategies

### Vertical Scaling

Increase resources on a single instance:

| Load | CPU | Memory | Workers |
|------|-----|--------|---------|
| Light | 1 core | 512MB | 5 |
| Medium | 2 cores | 1GB | 10 |
| Heavy | 4 cores | 2GB | 20 |
| Extreme | 8+ cores | 4GB+ | 50+ |

```yaml
resources:
  limits:
    cpu: "4"
    memory: 4Gi
  requests:
    cpu: "2"
    memory: 2Gi
```

### Horizontal Scaling

Add more m9m instances:

```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  replicas: 5  # Scale out
```

Requirements for horizontal scaling:
- External database (PostgreSQL)
- External queue (Redis/RabbitMQ)
- Shared storage if needed

## Queue-Based Scaling

### Architecture

```
┌─────────────┐
│ API Servers │ ──→ Queue ──→ Workers
└─────────────┘        ↑
                       │
              ┌────────┴────────┐
              │  Worker Pool    │
              │  ┌───┐ ┌───┐    │
              │  │ W │ │ W │... │
              │  └───┘ └───┘    │
              └─────────────────┘
```

### Configuration

```yaml
queue:
  type: redis
  url: "redis://redis:6379"
  maxWorkers: 20
  batchSize: 10

  # Separate API and worker roles
  role: worker  # or "api" or "both"
```

### Dedicated Workers

Deploy separate worker pods:

```yaml
# m9m-workers.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: m9m-workers
spec:
  replicas: 10
  template:
    spec:
      containers:
      - name: m9m
        image: m9m/m9m:latest
        args: ["worker"]
        env:
        - name: M9M_ROLE
          value: "worker"
        - name: M9M_MAX_WORKERS
          value: "10"
```

## Auto-Scaling

### Kubernetes HPA

Scale based on CPU/memory:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: m9m-workers
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: m9m-workers
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

### Custom Metrics Scaling

Scale based on queue depth:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: m9m-workers
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: m9m-workers
  minReplicas: 3
  maxReplicas: 50
  metrics:
  - type: External
    external:
      metric:
        name: m9m_queue_size
      target:
        type: AverageValue
        averageValue: "100"
```

### KEDA (Kubernetes Event-Driven Autoscaling)

```yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: m9m-workers
spec:
  scaleTargetRef:
    name: m9m-workers
  minReplicaCount: 1
  maxReplicaCount: 50
  triggers:
  - type: redis
    metadata:
      address: redis:6379
      listName: m9m:queue
      listLength: "100"
```

## Database Scaling

### Connection Pooling

```yaml
# PgBouncer configuration
[pgbouncer]
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 50
```

### Read Replicas

Route read queries to replicas:

```yaml
database:
  primary:
    url: "postgres://primary:5432/m9m"
  replicas:
    - url: "postgres://replica1:5432/m9m"
    - url: "postgres://replica2:5432/m9m"
  readFromReplica: true
```

### Database Sharding

For very large deployments:

```
Workflow ID → Hash → Shard
     ↓
┌─────────┐  ┌─────────┐  ┌─────────┐
│ Shard 1 │  │ Shard 2 │  │ Shard 3 │
└─────────┘  └─────────┘  └─────────┘
```

## Queue Scaling

### Redis Cluster

```yaml
redis:
  type: cluster
  nodes:
    - redis-1:6379
    - redis-2:6379
    - redis-3:6379
```

### RabbitMQ for High Throughput

```yaml
queue:
  type: rabbitmq
  url: "amqp://user:pass@rabbitmq:5672"
  prefetchCount: 10
  exchanges:
    workflows:
      type: direct
      durable: true
```

## Performance Optimization

### Batch Processing

Process items in batches:

```yaml
execution:
  batchSize: 100
  batchTimeout: 5s
```

### Caching

Enable response caching:

```yaml
cache:
  enabled: true
  type: redis
  url: "redis://redis:6379"
  ttl: 5m
  maxSize: 1000
```

### Connection Reuse

```yaml
http:
  maxIdleConns: 100
  maxIdleConnsPerHost: 10
  idleConnTimeout: 90s
  keepAlive: true
```

## Load Balancing

### Nginx

```nginx
upstream m9m {
    least_conn;
    server m9m-1:8080 weight=3;
    server m9m-2:8080 weight=3;
    server m9m-3:8080 weight=3;
    keepalive 32;
}

server {
    location / {
        proxy_pass http://m9m;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
    }
}
```

### Kubernetes Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: m9m
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: nlb
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 8080
  selector:
    app: m9m
```

## Monitoring Scale

### Key Metrics

```promql
# Throughput
rate(m9m_workflow_executions_total[5m])

# Latency percentiles
histogram_quantile(0.99, rate(m9m_execution_duration_seconds_bucket[5m]))

# Queue depth
m9m_queue_size

# Worker utilization
m9m_active_workers / m9m_max_workers

# Error rate
rate(m9m_workflow_errors_total[5m]) / rate(m9m_workflow_executions_total[5m])
```

### Capacity Planning

Calculate capacity:

```
Capacity = Workers × (1 / Avg Execution Time)

Example:
20 workers × (1 / 2s) = 10 workflows/second = 36,000/hour
```

### Scaling Alerts

```yaml
- alert: HighQueueDepth
  expr: m9m_queue_size > 500
  for: 5m
  annotations:
    summary: "Queue depth high, consider scaling workers"

- alert: WorkerSaturation
  expr: m9m_active_workers / m9m_max_workers > 0.9
  for: 10m
  annotations:
    summary: "Workers near capacity, scale out"
```

## Cost Optimization

### Right-Sizing

Monitor actual usage:

```bash
# Get resource recommendations
kubectl top pods -n m9m
```

### Spot/Preemptible Instances

Use for workers:

```yaml
# AWS Spot
nodeSelector:
  node.kubernetes.io/lifecycle: spot

# GCP Preemptible
nodeSelector:
  cloud.google.com/gke-preemptible: "true"
```

### Scale to Zero

For development environments:

```yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: m9m-dev
spec:
  minReplicaCount: 0  # Scale to zero
  cooldownPeriod: 300
```

## Benchmarking

### Load Testing

```bash
# Using hey
hey -n 10000 -c 100 http://m9m:8080/api/v1/workflows/test/execute

# Using k6
k6 run --vus 100 --duration 5m load-test.js
```

### Baseline Metrics

| Metric | Target |
|--------|--------|
| Throughput | 1000+ req/s |
| Latency (p99) | < 100ms |
| Error rate | < 0.1% |
| Queue depth | < 100 |

## Next Steps

- [Production](production.md) - Production best practices
- [Configuration](../reference/configuration.md) - Tuning options
- [Troubleshooting](../reference/troubleshooting.md) - Performance issues
