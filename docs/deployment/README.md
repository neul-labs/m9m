# Deployment Guide

This guide covers production deployment strategies for m9m across different environments and platforms.

Note: official support is currently focused on the single binary/package-manager path. Kubernetes sections are experimental reference manifests.

## Overview

m9m is designed for cloud-native deployments with multiple deployment options:
- Single binary execution
- Docker containers
- Kubernetes clusters (experimental)
- Serverless environments (planned)

## Prerequisites

### System Requirements
- **CPU**: 2+ cores recommended for production
- **Memory**: 512MB minimum, 2GB recommended
- **Storage**: 1GB for application, additional for workflows/logs
- **Network**: HTTP/HTTPS access for node integrations

### Dependencies
- **Optional**: Redis (for queue scaling)
- **Optional**: RabbitMQ (alternative queue backend)
- **Optional**: PostgreSQL (for persistent storage)
- **Optional**: Prometheus (for metrics)
- **Optional**: Jaeger (for tracing)

## Docker Deployment

### Quick Start
```bash
# Pull latest image
docker pull neul-labs/m9m:latest

# Run with default settings
docker run -p 8080:8080 neul-labs/m9m:latest

# Run with custom configuration
docker run -p 8080:8080 \
  -e M9M_QUEUE_TYPE=redis \
  -e M9M_QUEUE_URL=redis://redis:6379 \
  -v /host/workflows:/app/workflows \
  neul-labs/m9m:latest
```

### Production Docker Setup
```dockerfile
# Dockerfile for custom build
FROM neul-labs/m9m:latest

# Copy custom configuration
COPY config.yaml /app/config.yaml

# Copy custom workflows
COPY workflows/ /app/workflows/

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

CMD ["m9m", "serve", "--config", "/app/config.yaml"]
```

## Docker Compose Deployment

### Complete Stack
```yaml
version: '3.8'

services:
  m9m:
    image: neul-labs/m9m:latest
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      # Server Configuration
      - M9M_PORT=8080
      - M9M_HOST=0.0.0.0

      # Queue Configuration
      - M9M_QUEUE_TYPE=redis
      - M9M_QUEUE_URL=redis://redis:6379
      - M9M_MAX_WORKERS=10

      # Database Configuration
      - M9M_DB_TYPE=postgresql
      - M9M_DB_URL=postgres://n8n_go:password@postgres:5432/n8n_go?sslmode=disable

      # Monitoring
      - M9M_METRICS_PORT=9090
      - M9M_TRACING_ENDPOINT=http://jaeger:14268/api/traces
    volumes:
      - ./workflows:/app/workflows
      - ./config.yaml:/app/config.yaml
    depends_on:
      - redis
      - postgres
      - jaeger
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 30s
      timeout: 10s
      retries: 3

  postgres:
    image: postgres:15
    environment:
      - POSTGRES_DB=n8n_go
      - POSTGRES_USER=n8n_go
      - POSTGRES_PASSWORD=password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U n8n_go"]
      interval: 30s
      timeout: 10s
      retries: 3

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9091:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
      - '--storage.tsdb.retention.time=200h'
      - '--web.enable-lifecycle'
    restart: unless-stopped

  jaeger:
    image: jaegertracing/all-in-one:1.45
    ports:
      - "16686:16686"
      - "14268:14268"
    environment:
      - COLLECTOR_OTLP_ENABLED=true
    restart: unless-stopped

volumes:
  redis_data:
  postgres_data:
  prometheus_data:
```

### Prometheus Configuration
```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'm9m'
    static_configs:
      - targets: ['m9m:9090']
    metrics_path: '/metrics'
    scrape_interval: 10s
```

## Kubernetes Deployment

### Namespace and ConfigMap
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: m9m
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: m9m-config
  namespace: m9m
data:
  config.yaml: |
    server:
      port: 8080
      host: "0.0.0.0"
    queue:
      type: "redis"
      url: "redis://redis-service:6379"
      max_workers: 20
    monitoring:
      metrics_port: 9090
      tracing:
        endpoint: "http://jaeger-service:14268/api/traces"
        service_name: "m9m"
```

### Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: m9m
  namespace: m9m
  labels:
    app: m9m
spec:
  replicas: 3
  selector:
    matchLabels:
      app: m9m
  template:
    metadata:
      labels:
        app: m9m
    spec:
      containers:
      - name: m9m
        image: neul-labs/m9m:latest
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: metrics
        env:
        - name: M9M_QUEUE_TYPE
          value: "redis"
        - name: M9M_QUEUE_URL
          value: "redis://redis-service:6379"
        - name: M9M_MAX_WORKERS
          value: "20"
        volumeMounts:
        - name: config
          mountPath: /app/config.yaml
          subPath: config.yaml
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "1Gi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: m9m-config
```

### Services
```yaml
apiVersion: v1
kind: Service
metadata:
  name: m9m-service
  namespace: m9m
spec:
  selector:
    app: m9m
  ports:
  - name: http
    port: 8080
    targetPort: 8080
  - name: metrics
    port: 9090
    targetPort: 9090
  type: LoadBalancer
---
apiVersion: v1
kind: Service
metadata:
  name: redis-service
  namespace: m9m
spec:
  selector:
    app: redis
  ports:
  - port: 6379
    targetPort: 6379
```

### Redis Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
  namespace: m9m
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: redis:7-alpine
        ports:
        - containerPort: 6379
        resources:
          requests:
            memory: "128Mi"
            cpu: "50m"
          limits:
            memory: "256Mi"
            cpu: "100m"
```

### Ingress (with TLS)
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: m9m-ingress
  namespace: m9m
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  tls:
  - hosts:
    - workflows.yourdomain.com
    secretName: m9m-tls
  rules:
  - host: workflows.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: m9m-service
            port:
              number: 8080
```

## High Availability Setup

### Multi-Region Deployment
```yaml
# Primary region deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: m9m-primary
spec:
  replicas: 5
  template:
    spec:
      containers:
      - name: m9m
        env:
        - name: M9M_REGION
          value: "us-east-1"
        - name: M9M_QUEUE_URL
          value: "redis://redis-primary:6379"
---
# Secondary region deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: m9m-secondary
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: m9m
        env:
        - name: M9M_REGION
          value: "us-west-2"
        - name: M9M_QUEUE_URL
          value: "redis://redis-secondary:6379"
```

## Security Considerations

### Network Security
```yaml
# Network Policy
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: m9m-network-policy
spec:
  podSelector:
    matchLabels:
      app: m9m
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: m9m
    ports:
    - protocol: TCP
      port: 6379  # Redis
    - protocol: TCP
      port: 5432  # PostgreSQL
```

### Secrets Management
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: m9m-secrets
  namespace: m9m
type: Opaque
stringData:
  db-password: "your-secure-password"
  redis-password: "your-redis-password"
  api-key: "your-api-key"
---
# Use in deployment
spec:
  template:
    spec:
      containers:
      - name: m9m
        env:
        - name: M9M_DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: m9m-secrets
              key: db-password
```

## Monitoring and Logging

### Monitoring Stack
```yaml
# ServiceMonitor for Prometheus Operator
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: m9m-metrics
  namespace: m9m
spec:
  selector:
    matchLabels:
      app: m9m
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
```

### Logging Configuration
```yaml
# Fluentd DaemonSet for log collection
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: fluentd-m9m
spec:
  template:
    spec:
      containers:
      - name: fluentd
        image: fluent/fluentd-kubernetes-daemonset:v1-debian-elasticsearch
        env:
        - name: FLUENT_ELASTICSEARCH_HOST
          value: "elasticsearch.logging.svc.cluster.local"
        - name: FLUENT_ELASTICSEARCH_PORT
          value: "9200"
```

## Backup and Recovery

### Automated Backups
```bash
#!/bin/bash
# backup-script.sh

# Backup workflows
kubectl exec -n m9m deployment/m9m -- \
  tar -czf - /app/workflows | \
  gzip > workflows-backup-$(date +%Y%m%d).tar.gz

# Backup database
kubectl exec -n m9m deployment/postgres -- \
  pg_dump -U n8n_go n8n_go | \
  gzip > db-backup-$(date +%Y%m%d).sql.gz

# Upload to cloud storage
aws s3 cp workflows-backup-$(date +%Y%m%d).tar.gz s3://your-backup-bucket/
aws s3 cp db-backup-$(date +%Y%m%d).sql.gz s3://your-backup-bucket/
```

## Performance Tuning

### Resource Optimization
```yaml
# Horizontal Pod Autoscaler
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: m9m-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: m9m
  minReplicas: 3
  maxReplicas: 50
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### Queue Optimization
```yaml
env:
- name: M9M_MAX_WORKERS
  value: "50"
- name: M9M_QUEUE_BATCH_SIZE
  value: "10"
- name: M9M_WORKER_TIMEOUT
  value: "300s"
```

## Troubleshooting

### Common Issues
1. **Out of Memory**: Increase memory limits or worker count
2. **Queue Backlog**: Scale workers or optimize node performance
3. **Connection Timeouts**: Check network policies and service endpoints
4. **Health Check Failures**: Verify dependencies are healthy

### Debug Commands
```bash
# Check pod status
kubectl get pods -n m9m

# Check logs
kubectl logs -f deployment/m9m -n m9m

# Check metrics
curl http://m9m-service:9090/metrics

# Check health
curl http://m9m-service:8080/health
```
