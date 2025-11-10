# n8n-go Deployment Guide

This guide provides comprehensive instructions for deploying n8n-go in various environments.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start with Docker](#quick-start-with-docker)
- [Building from Source](#building-from-source)
- [Docker Deployment](#docker-deployment)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Production Considerations](#production-considerations)
- [Monitoring & Observability](#monitoring--observability)
- [Troubleshooting](#troubleshooting)

## Prerequisites

- **Docker**: 20.10+ (for containerized deployment)
- **Docker Compose**: 2.0+ (for local development)
- **Go**: 1.24+ (for building from source)
- **PostgreSQL**: 16+ (recommended database)
- **Redis**: 7+ (for queue management)

## Quick Start with Docker

The fastest way to get started with n8n-go using the existing n8n frontend:

```bash
# Clone the repository
git clone https://github.com/dipankar/n8n-go.git
cd n8n-go

# Start all services (n8n-go backend + n8n frontend + dependencies)
docker-compose up -d

# View logs
docker-compose logs -f n8n-go

# Access the application
# n8n Frontend: http://localhost:5678
# n8n-go Backend API: http://localhost:5678
# Metrics: http://localhost:9090/metrics
# Grafana: http://localhost:3000 (admin/admin)
# RabbitMQ Management: http://localhost:15672 (guest/guest)
```

The setup includes:
- **n8n-go**: High-performance backend (Go)
- **n8n-frontend**: Original n8n UI
- **PostgreSQL**: Workflow storage
- **Redis**: Queue management
- **MongoDB**: Document storage
- **RabbitMQ**: Advanced message queuing
- **Prometheus**: Metrics collection
- **Grafana**: Visualization

## Building from Source

### 1. Install Dependencies

```bash
# Install Go 1.24+
# Visit https://golang.org/dl/

# Verify installation
go version
```

### 2. Clone and Build

```bash
# Clone repository
git clone https://github.com/dipankar/n8n-go.git
cd n8n-go

# Download dependencies
go mod download

# Build
make build

# Or build manually
CGO_ENABLED=1 go build -o n8n-go ./cmd/n8n-go

# Run
./n8n-go serve --config config/config.yaml
```

### 3. Build for Different Platforms

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 make build

# Linux ARM64
GOOS=linux GOARCH=arm64 make build

# macOS AMD64
GOOS=darwin GOARCH=amd64 make build

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 make build

# Windows AMD64
GOOS=windows GOARCH=amd64 make build
```

## Docker Deployment

### Single Container Deployment

```bash
# Build Docker image
docker build -t n8n-go:latest .

# Run with environment variables
docker run -d \
  --name n8n-go \
  -p 5678:8080 \
  -p 9090:9090 \
  -e N8N_GO_PORT=8080 \
  -e N8N_GO_DB_TYPE=postgres \
  -e N8N_GO_DB_POSTGRES_HOST=postgres \
  -e N8N_GO_QUEUE_TYPE=redis \
  -e N8N_GO_QUEUE_URL=redis://redis:6379 \
  -v n8n-go-data:/app/data \
  n8n-go:latest
```

### Multi-Container with Docker Compose

**Minimal Setup (n8n-go + PostgreSQL + Redis)**:

Create `docker-compose.minimal.yml`:

```yaml
version: '3.8'

services:
  n8n-go:
    image: dipankar/n8n-go:latest
    ports:
      - "5678:8080"
      - "9090:9090"
    environment:
      - N8N_GO_DB_TYPE=postgres
      - N8N_GO_DB_POSTGRES_HOST=postgres
      - N8N_GO_QUEUE_TYPE=redis
      - N8N_GO_QUEUE_URL=redis://redis:6379
    volumes:
      - n8n-go-data:/app/data
    depends_on:
      - postgres
      - redis

  postgres:
    image: postgres:16-alpine
    environment:
      - POSTGRES_DB=n8n
      - POSTGRES_USER=n8n
      - POSTGRES_PASSWORD=changeme
    volumes:
      - postgres-data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes
    volumes:
      - redis-data:/data

volumes:
  n8n-go-data:
  postgres-data:
  redis-data:
```

Run:

```bash
docker-compose -f docker-compose.minimal.yml up -d
```

**Full Setup (with n8n Frontend)**:

```bash
# Use the provided docker-compose.yml
docker-compose up -d

# Scale workers
docker-compose up -d --scale n8n-go=3

# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Stop and remove volumes
docker-compose down -v
```

## Kubernetes Deployment

### Prerequisites

- Kubernetes cluster (1.25+)
- kubectl configured
- Helm 3+ (optional but recommended)

### Using Helm (Recommended)

```bash
# Add n8n-go Helm repository
helm repo add n8n-go https://charts.n8n-go.io
helm repo update

# Install with default values
helm install n8n-go n8n-go/n8n-go

# Install with custom values
helm install n8n-go n8n-go/n8n-go \
  --set replicaCount=3 \
  --set postgresql.enabled=true \
  --set redis.enabled=true \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=n8n.example.com
```

### Using Kubectl (Manifests)

Create `k8s/deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: n8n-go
  labels:
    app: n8n-go
spec:
  replicas: 3
  selector:
    matchLabels:
      app: n8n-go
  template:
    metadata:
      labels:
        app: n8n-go
    spec:
      containers:
      - name: n8n-go
        image: dipankar/n8n-go:latest
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: metrics
        env:
        - name: N8N_GO_PORT
          value: "8080"
        - name: N8N_GO_DB_TYPE
          value: "postgres"
        - name: N8N_GO_DB_POSTGRES_HOST
          value: "postgres-service"
        - name: N8N_GO_QUEUE_TYPE
          value: "redis"
        - name: N8N_GO_QUEUE_URL
          value: "redis://redis-service:6379"
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: n8n-go-service
spec:
  selector:
    app: n8n-go
  ports:
  - name: http
    port: 80
    targetPort: 8080
  - name: metrics
    port: 9090
    targetPort: 9090
  type: LoadBalancer
```

Deploy:

```bash
kubectl apply -f k8s/
```

## Production Considerations

### Security

1. **Use HTTPS**:
   ```yaml
   server:
     tls:
       enabled: true
       cert_file: "/app/config/tls/cert.pem"
       key_file: "/app/config/tls/key.pem"
   ```

2. **Enable Authentication**:
   ```yaml
   security:
     jwt:
       enabled: true
       secret: "your-secure-random-secret"
     api_keys:
       enabled: true
   ```

3. **Secure Credentials**:
   ```yaml
   credentials:
     encryption_key: "minimum-32-character-secure-key"
     storage: "vault"  # Use HashiCorp Vault
   ```

### Performance Tuning

1. **Database Connection Pooling**:
   ```yaml
   database:
     postgres:
       max_connections: 50
       max_idle_connections: 10
       connection_lifetime: "30m"
   ```

2. **Queue Workers**:
   ```yaml
   queue:
     max_workers: 20  # Adjust based on CPU cores
   ```

3. **Caching**:
   ```yaml
   performance:
     cache:
       enabled: true
       backend: "redis"
       ttl: "10m"
   ```

### High Availability

1. **Multiple Replicas**:
   ```yaml
   # docker-compose
   services:
     n8n-go:
       deploy:
         replicas: 3
   ```

2. **Load Balancer**:
   - Use nginx, HAProxy, or cloud load balancer
   - Health check endpoint: `/health`

3. **Database Replication**:
   - PostgreSQL: Use streaming replication
   - Redis: Use Redis Sentinel or Cluster

### Backup & Recovery

1. **Database Backups**:
   ```bash
   # PostgreSQL backup
   pg_dump -h localhost -U n8n n8n > backup.sql

   # Restore
   psql -h localhost -U n8n n8n < backup.sql
   ```

2. **Workflow Backups**:
   ```bash
   # Export workflows
   ./n8n-go export-workflows --output /backups/workflows.json

   # Import workflows
   ./n8n-go import-workflows --input /backups/workflows.json
   ```

## Monitoring & Observability

### Prometheus Metrics

n8n-go exposes Prometheus metrics on port 9090:

```bash
curl http://localhost:9090/metrics
```

Key metrics:
- `n8n_go_workflow_executions_total`
- `n8n_go_workflow_execution_duration_seconds`
- `n8n_go_node_executions_total`
- `n8n_go_active_workflows`
- `n8n_go_queue_size`

### Grafana Dashboards

Access Grafana at `http://localhost:3000` (admin/admin)

Pre-built dashboards available for:
- Workflow execution performance
- Node execution statistics
- Queue health
- Database performance
- System resources

### Logging

Configure logging in `config/config.yaml`:

```yaml
logging:
  level: "info"
  format: "json"
  output: "/app/logs/n8n-go.log"
```

View logs:

```bash
# Docker
docker logs -f n8n-go

# Kubernetes
kubectl logs -f deployment/n8n-go

# File
tail -f /app/logs/n8n-go.log
```

### Tracing

Enable distributed tracing for debugging:

```yaml
monitoring:
  tracing:
    enabled: true
    provider: "jaeger"
    jaeger:
      endpoint: "http://jaeger:14268/api/traces"
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `N8N_GO_PORT` | `8080` | HTTP server port |
| `N8N_GO_HOST` | `0.0.0.0` | HTTP server host |
| `N8N_GO_LOG_LEVEL` | `info` | Log level (trace/debug/info/warn/error) |
| `N8N_GO_DB_TYPE` | `postgres` | Database type (postgres/mysql/sqlite/mongodb) |
| `N8N_GO_QUEUE_TYPE` | `redis` | Queue type (memory/redis/rabbitmq) |
| `N8N_GO_MAX_WORKERS` | `10` | Number of workflow execution workers |
| `N8N_GO_METRICS_PORT` | `9090` | Metrics server port |
| `N8N_GO_COMPATIBILITY_MODE` | `true` | Enable n8n API compatibility |

See `config/config.yaml` for complete configuration options.

## Troubleshooting

### Build Errors

**Issue**: `cannot find package`
```bash
# Solution: Download dependencies
go mod download
go mod tidy
```

**Issue**: CGO errors
```bash
# Solution: Ensure GCC is installed
# Ubuntu/Debian
sudo apt-get install build-essential

# macOS
xcode-select --install
```

### Runtime Errors

**Issue**: Database connection failed
```bash
# Check database is running
docker ps | grep postgres

# Check connection details
psql -h localhost -U n8n -d n8n

# Verify environment variables
docker exec n8n-go env | grep DB
```

**Issue**: Redis connection failed
```bash
# Check Redis is running
docker ps | grep redis

# Test connection
redis-cli -h localhost ping

# Check queue URL
docker exec n8n-go env | grep QUEUE
```

### Performance Issues

**Issue**: Slow workflow execution
```bash
# Check metrics
curl http://localhost:9090/metrics | grep duration

# Increase workers
docker-compose up -d --scale n8n-go=5

# Check database performance
# Add indexes, tune connection pool
```

**Issue**: High memory usage
```bash
# Monitor memory
docker stats n8n-go

# Reduce worker count
# Implement workflow execution limits
# Enable caching
```

### n8n Frontend Integration Issues

**Issue**: Frontend can't connect to backend
```bash
# Verify backend is running
curl http://localhost:5678/health

# Check backend URL in frontend config
docker exec n8n-frontend env | grep BACKEND

# Check network connectivity
docker network inspect n8n-network
```

## Support

- **Documentation**: https://docs.n8n-go.io
- **Issues**: https://github.com/dipankar/n8n-go/issues
- **Discord**: https://discord.gg/n8n-go
- **Email**: support@n8n-go.io

## License

n8n-go is licensed under the Apache License 2.0. See LICENSE for details.
