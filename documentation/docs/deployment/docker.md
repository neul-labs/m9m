# Docker Deployment

Deploy m9m using Docker.

## Quick Start

```bash
docker run -d \
  --name m9m \
  -p 8080:8080 \
  neul-labs/m9m:latest
```

## Docker Images

| Image | Description |
|-------|-------------|
| `neul-labs/m9m:latest` | Latest stable release |
| `neul-labs/m9m:1.0.0` | Specific version |
| `neul-labs/m9m:alpine` | Minimal Alpine-based |

## Basic Configuration

### With Environment Variables

```bash
docker run -d \
  --name m9m \
  -p 8080:8080 \
  -e M9M_LOG_LEVEL=info \
  -e M9M_JWT_SECRET=your-secret \
  neul-labs/m9m:latest
```

### With Persistent Storage

```bash
docker run -d \
  --name m9m \
  -p 8080:8080 \
  -v m9m-data:/data \
  neul-labs/m9m:latest
```

### With Config File

```bash
docker run -d \
  --name m9m \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/etc/m9m/config.yaml \
  neul-labs/m9m:latest
```

## Docker Compose

### Basic Setup

```yaml
# docker-compose.yml
version: '3.8'

services:
  m9m:
    image: neul-labs/m9m:latest
    ports:
      - "8080:8080"
    volumes:
      - m9m-data:/data
    environment:
      - M9M_LOG_LEVEL=info

volumes:
  m9m-data:
```

Start:

```bash
docker compose up -d
```

### With PostgreSQL

```yaml
version: '3.8'

services:
  m9m:
    image: neul-labs/m9m:latest
    ports:
      - "8080:8080"
    environment:
      - M9M_DATABASE_TYPE=postgres
      - M9M_DATABASE_URL=postgres://m9m:password@postgres:5432/m9m
      - M9M_JWT_SECRET=${JWT_SECRET}
    depends_on:
      postgres:
        condition: service_healthy

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_USER=m9m
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=m9m
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U m9m"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  postgres-data:
```

### With Redis Queue

```yaml
version: '3.8'

services:
  m9m:
    image: neul-labs/m9m:latest
    ports:
      - "8080:8080"
    environment:
      - M9M_DATABASE_TYPE=postgres
      - M9M_DATABASE_URL=postgres://m9m:password@postgres:5432/m9m
      - M9M_QUEUE_TYPE=redis
      - M9M_QUEUE_URL=redis://redis:6379
      - M9M_WORKERS=5
    depends_on:
      - postgres
      - redis

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_USER=m9m
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=m9m
    volumes:
      - postgres-data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    volumes:
      - redis-data:/data

volumes:
  postgres-data:
  redis-data:
```

### Full Production Stack

```yaml
version: '3.8'

services:
  m9m:
    image: neul-labs/m9m:latest
    deploy:
      replicas: 2
      resources:
        limits:
          memory: 512M
        reservations:
          memory: 256M
    ports:
      - "8080:8080"
    environment:
      - M9M_DATABASE_TYPE=postgres
      - M9M_DATABASE_URL=postgres://m9m:${DB_PASSWORD}@postgres:5432/m9m
      - M9M_QUEUE_TYPE=redis
      - M9M_QUEUE_URL=redis://redis:6379
      - M9M_JWT_SECRET=${JWT_SECRET}
      - M9M_LOG_FORMAT=json
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    depends_on:
      - postgres
      - redis

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_USER=m9m
      - POSTGRES_PASSWORD=${DB_PASSWORD}
      - POSTGRES_DB=m9m
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U m9m"]
      interval: 10s

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes
    volumes:
      - redis-data:/data

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./certs:/etc/nginx/certs:ro
    depends_on:
      - m9m

volumes:
  postgres-data:
  redis-data:
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `M9M_PORT` | 8080 | Server port |
| `M9M_HOST` | 0.0.0.0 | Listen address |
| `M9M_DATABASE_TYPE` | sqlite | Database type |
| `M9M_DATABASE_URL` | /data/m9m.db | Connection string |
| `M9M_QUEUE_TYPE` | memory | Queue backend |
| `M9M_QUEUE_URL` | - | Queue connection |
| `M9M_WORKERS` | 3 | Worker count |
| `M9M_JWT_SECRET` | - | JWT signing key |
| `M9M_LOG_LEVEL` | info | Log verbosity |
| `M9M_LOG_FORMAT` | text | Log format |

## Volume Mounts

| Path | Purpose |
|------|---------|
| `/data` | SQLite database, file storage |
| `/etc/m9m/config.yaml` | Configuration file |
| `/var/log/m9m` | Log files |

## Networking

### Internal Network

```yaml
services:
  m9m:
    networks:
      - internal
      - frontend

networks:
  internal:
    internal: true
  frontend:
```

### Expose Specific Ports

```yaml
services:
  m9m:
    ports:
      - "127.0.0.1:8080:8080"  # Local only
```

## Resource Limits

```yaml
services:
  m9m:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
```

## Health Checks

```yaml
services:
  m9m:
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
```

## Logging

### JSON Logging

```yaml
services:
  m9m:
    environment:
      - M9M_LOG_FORMAT=json
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
```

### External Logging

```yaml
services:
  m9m:
    logging:
      driver: syslog
      options:
        syslog-address: "tcp://logserver:514"
```

## Building Custom Image

### Dockerfile

```dockerfile
FROM neul-labs/m9m:latest

# Add custom config
COPY config.yaml /etc/m9m/config.yaml

# Add custom scripts
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
```

Build:

```bash
docker build -t my-m9m:latest .
```

## Troubleshooting

### View Logs

```bash
docker logs m9m
docker logs -f m9m  # Follow
```

### Shell Access

```bash
docker exec -it m9m /bin/sh
```

### Check Health

```bash
docker inspect --format='{{.State.Health.Status}}' m9m
```

### Restart Container

```bash
docker restart m9m
```

## Upgrading

### Pull New Image

```bash
docker pull neul-labs/m9m:latest
```

### Upgrade with Docker Compose

```bash
docker compose pull
docker compose up -d
```

### Backup Before Upgrade

```bash
# Backup volume
docker run --rm -v m9m-data:/data -v $(pwd):/backup alpine tar czf /backup/m9m-backup.tar.gz /data

# Backup database
docker exec m9m m9m backup /data/backup.sql
```
