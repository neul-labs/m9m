# Deployment Overview

m9m supports multiple deployment options from local development to enterprise-scale production.

## Deployment Options

| Option | Best For | Complexity |
|--------|----------|------------|
| Binary | Development, small teams | Low |
| Docker | Single server, development | Low |
| Docker Compose | Small to medium deployments | Medium |
| Kubernetes | Production, enterprise | High |

## Architecture

### Single Instance

```
┌─────────────────────────────────────┐
│              m9m                     │
│  ┌─────────┐  ┌─────────┐           │
│  │ Web UI  │  │ REST API │          │
│  └────┬────┘  └────┬────┘           │
│       └─────┬──────┘                │
│             │                        │
│  ┌──────────┴──────────┐            │
│  │   Workflow Engine    │           │
│  └──────────┬──────────┘            │
│             │                        │
│  ┌──────────┴──────────┐            │
│  │   Memory Queue       │           │
│  └─────────────────────┘            │
└─────────────────────────────────────┘
              │
    ┌─────────┴─────────┐
    │    PostgreSQL     │
    └───────────────────┘
```

### Distributed

```
                    ┌─────────────┐
                    │ Load Balancer│
                    └──────┬──────┘
           ┌───────────────┼───────────────┐
           │               │               │
    ┌──────┴──────┐ ┌──────┴──────┐ ┌──────┴──────┐
    │   m9m #1    │ │   m9m #2    │ │   m9m #3    │
    └──────┬──────┘ └──────┬──────┘ └──────┬──────┘
           │               │               │
           └───────────────┼───────────────┘
                           │
              ┌────────────┴────────────┐
              │                          │
       ┌──────┴──────┐          ┌────────┴────────┐
       │    Redis    │          │   PostgreSQL    │
       │   (Queue)   │          │   (Database)    │
       └─────────────┘          └─────────────────┘
```

## Quick Start

### Binary

```bash
# Download
wget https://github.com/m9m/m9m/releases/latest/download/m9m-linux-amd64
chmod +x m9m-linux-amd64

# Run
./m9m-linux-amd64 serve
```

### Docker

```bash
docker run -d \
  --name m9m \
  -p 8080:8080 \
  -v m9m_data:/data \
  m9m/m9m:latest
```

### Docker Compose

```yaml
version: '3.8'
services:
  m9m:
    image: m9m/m9m:latest
    ports:
      - "8080:8080"
    environment:
      - M9M_DB_TYPE=postgres
      - M9M_DB_URL=postgres://m9m:password@postgres/m9m
    depends_on:
      - postgres

  postgres:
    image: postgres:15
    environment:
      - POSTGRES_USER=m9m
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=m9m
    volumes:
      - postgres_data:/var/lib/postgresql/data

volumes:
  postgres_data:
```

## System Requirements

### Minimum

| Component | Requirement |
|-----------|-------------|
| CPU | 1 core |
| Memory | 256MB |
| Disk | 500MB |
| OS | Linux, macOS, Windows |

### Recommended (Production)

| Component | Requirement |
|-----------|-------------|
| CPU | 2+ cores |
| Memory | 1GB+ |
| Disk | 10GB+ SSD |
| OS | Linux (Ubuntu 22.04, RHEL 8+) |

### Scaling Guidelines

| Workflows/Hour | Workers | Memory | CPU |
|----------------|---------|--------|-----|
| < 100 | 1 | 512MB | 1 |
| 100-1000 | 2-4 | 1GB | 2 |
| 1000-10000 | 4-8 | 2GB | 4 |
| > 10000 | 8+ | 4GB+ | 8+ |

## Configuration

### Environment Variables

```bash
# Core
M9M_PORT=8080
M9M_HOST=0.0.0.0

# Database
M9M_DB_TYPE=postgres
M9M_DB_URL=postgres://user:pass@host/db

# Queue
M9M_QUEUE_TYPE=redis
M9M_QUEUE_URL=redis://localhost:6379

# Security
M9M_ENCRYPTION_KEY=your-32-char-key
```

### Configuration File

```yaml
# config.yaml
server:
  port: 8080
  host: "0.0.0.0"

database:
  type: postgres
  url: "postgres://user:pass@host/db"
  maxConnections: 20

queue:
  type: redis
  url: "redis://localhost:6379"
  maxWorkers: 10

security:
  encryptionKey: "your-32-character-encryption-key"
```

## Storage Options

### Database

| Database | Use Case |
|----------|----------|
| SQLite | Development, single instance |
| PostgreSQL | Production (recommended) |
| MySQL | Production alternative |

### Queue

| Queue | Use Case |
|-------|----------|
| Memory | Development, single instance |
| Redis | Production (recommended) |
| RabbitMQ | High-throughput, enterprise |

## Health Checks

```bash
# Health
curl http://localhost:8080/health

# Readiness
curl http://localhost:8080/ready

# Metrics
curl http://localhost:9090/metrics
```

## Deployment Guides

1. [Docker](docker.md) - Container deployment
2. [Kubernetes](kubernetes.md) - Orchestrated deployment
3. [Production](production.md) - Production best practices
4. [Scaling](scaling.md) - Horizontal scaling

## Next Steps

Choose your deployment method and follow the appropriate guide:

- **Just getting started?** → [Docker](docker.md)
- **Running in production?** → [Production](production.md)
- **Need to scale?** → [Kubernetes](kubernetes.md)
