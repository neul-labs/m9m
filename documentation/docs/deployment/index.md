# Deployment Overview

Deploy m9m to your infrastructure.

## Deployment Options

| Method | Best For |
|--------|----------|
| Docker | Quick setup, single server |
| Docker Compose | Multi-service local dev |
| Kubernetes | Production, scalability |
| Binary | Minimal overhead, edge |

## Quick Start

### Docker (Simplest)

```bash
docker run -d \
  --name m9m \
  -p 8080:8080 \
  -v m9m-data:/data \
  neullabs/m9m:latest
```

### Binary

```bash
# Download binary
curl -L https://github.com/neul-labs/m9m/releases/latest/download/m9m-linux-amd64 -o m9m
chmod +x m9m

# Run
./m9m serve --port 8080
```

## System Requirements

### Minimum

| Resource | Requirement |
|----------|-------------|
| CPU | 1 core |
| Memory | 256 MB |
| Disk | 1 GB |

### Recommended (Production)

| Resource | Requirement |
|----------|-------------|
| CPU | 2+ cores |
| Memory | 512 MB+ |
| Disk | 10 GB+ SSD |

## Resource Comparison

m9m vs n8n:

| Metric | m9m | n8n |
|--------|-----|-----|
| Memory | ~150 MB | ~512 MB |
| Container size | ~300 MB | ~1.2 GB |
| Startup time | ~500 ms | ~3 s |
| CPU (idle) | <1% | ~5% |

## Architecture Patterns

### Single Instance

Simple deployment for low traffic:

```
┌─────────────────┐
│     m9m         │
│  ┌───────────┐  │
│  │  SQLite   │  │
│  └───────────┘  │
└─────────────────┘
```

### High Availability

Scalable deployment:

```
         ┌─────────────┐
         │Load Balancer│
         └──────┬──────┘
    ┌───────────┼───────────┐
    ▼           ▼           ▼
┌───────┐  ┌───────┐  ┌───────┐
│ m9m 1 │  │ m9m 2 │  │ m9m 3 │
└───┬───┘  └───┬───┘  └───┬───┘
    └───────────┼───────────┘
                ▼
         ┌───────────┐
         │ PostgreSQL│
         └───────────┘
```

## Configuration

### Environment Variables

```bash
# Server
M9M_PORT=8080
M9M_HOST=0.0.0.0

# Database
M9M_DATABASE_TYPE=postgres
M9M_DATABASE_URL=postgres://user:pass@localhost/m9m

# Security
M9M_JWT_SECRET=your-secure-secret
M9M_ENCRYPTION_KEY=32-byte-key-here
```

### Config File

```yaml
server:
  port: 8080
  host: "0.0.0.0"

database:
  type: postgres
  url: "postgres://user:pass@localhost/m9m"

security:
  jwtSecret: "your-secure-secret"
```

## Health Checks

### HTTP Health

```bash
curl http://localhost:8080/health
```

Response:

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": 3600
}
```

### Readiness Check

```bash
curl http://localhost:8080/ready
```

## Monitoring

### Prometheus Metrics

```bash
curl http://localhost:8080/metrics
```

Key metrics:

| Metric | Description |
|--------|-------------|
| `m9m_executions_total` | Total workflow executions |
| `m9m_execution_duration` | Execution time histogram |
| `m9m_active_workflows` | Currently active workflows |
| `m9m_queue_size` | Jobs in queue |

### Logging

Configure log level:

```bash
M9M_LOG_LEVEL=info  # debug, info, warn, error
```

JSON logging for production:

```bash
M9M_LOG_FORMAT=json
```

## Security Checklist

- [ ] Set strong `JWT_SECRET`
- [ ] Enable HTTPS/TLS
- [ ] Configure CORS appropriately
- [ ] Set up authentication
- [ ] Enable audit logging
- [ ] Regular security updates

## Next Steps

- [Docker Deployment](docker.md) - Detailed Docker guide
- [Kubernetes Deployment](kubernetes.md) - K8s deployment
- [Production Checklist](production.md) - Production readiness
