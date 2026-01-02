# Docker Deployment

Deploy m9m using Docker and Docker Compose.

## Quick Start

### Single Container

```bash
docker run -d \
  --name m9m \
  -p 8080:8080 \
  -p 9090:9090 \
  -v m9m_data:/data \
  m9m/m9m:latest
```

Access at http://localhost:8080

### With PostgreSQL

```bash
# Create network
docker network create m9m-net

# Start PostgreSQL
docker run -d \
  --name postgres \
  --network m9m-net \
  -e POSTGRES_USER=m9m \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=m9m \
  -v postgres_data:/var/lib/postgresql/data \
  postgres:15

# Start m9m
docker run -d \
  --name m9m \
  --network m9m-net \
  -p 8080:8080 \
  -e M9M_DB_TYPE=postgres \
  -e M9M_DB_URL=postgres://m9m:password@postgres/m9m \
  m9m/m9m:latest
```

## Docker Compose

### Basic Setup

```yaml
# docker-compose.yml
version: '3.8'

services:
  m9m:
    image: m9m/m9m:latest
    container_name: m9m
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      - M9M_DB_TYPE=postgres
      - M9M_DB_URL=postgres://m9m:password@postgres/m9m
      - M9M_ENCRYPTION_KEY=${ENCRYPTION_KEY}
    volumes:
      - ./data:/data
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped

  postgres:
    image: postgres:15-alpine
    container_name: m9m-postgres
    environment:
      - POSTGRES_USER=m9m
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=m9m
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U m9m"]
      interval: 5s
      timeout: 5s
      retries: 5
    restart: unless-stopped

volumes:
  postgres_data:
```

Start:
```bash
docker compose up -d
```

### With Redis Queue

```yaml
version: '3.8'

services:
  m9m:
    image: m9m/m9m:latest
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      - M9M_DB_TYPE=postgres
      - M9M_DB_URL=postgres://m9m:password@postgres/m9m
      - M9M_QUEUE_TYPE=redis
      - M9M_QUEUE_URL=redis://redis:6379
      - M9M_MAX_WORKERS=10
    depends_on:
      - postgres
      - redis
    restart: unless-stopped

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_USER=m9m
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=m9m
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
```

### Production Setup

```yaml
version: '3.8'

services:
  m9m:
    image: m9m/m9m:latest
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      - M9M_DB_TYPE=postgres
      - M9M_DB_URL=postgres://m9m:${DB_PASSWORD}@postgres/m9m?sslmode=require
      - M9M_QUEUE_TYPE=redis
      - M9M_QUEUE_URL=redis://:${REDIS_PASSWORD}@redis:6379
      - M9M_ENCRYPTION_KEY=${ENCRYPTION_KEY}
      - M9M_MAX_WORKERS=20
      - M9M_LOG_LEVEL=info
    volumes:
      - ./config:/app/config:ro
      - ./data:/data
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 512M
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    restart: unless-stopped
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_USER=m9m
      - POSTGRES_PASSWORD=${DB_PASSWORD}
      - POSTGRES_DB=m9m
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./init.sql:/docker-entrypoint-initdb.d/init.sql:ro
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U m9m"]
      interval: 5s
      timeout: 5s
      retries: 5
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 1G
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD} --appendonly yes
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "-a", "${REDIS_PASSWORD}", "ping"]
      interval: 5s
      timeout: 5s
      retries: 5
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 512M
    restart: unless-stopped

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
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
```

### Environment File

```bash
# .env
DB_PASSWORD=your-secure-db-password
REDIS_PASSWORD=your-secure-redis-password
ENCRYPTION_KEY=your-32-character-encryption-key
```

## Nginx Configuration

```nginx
# nginx.conf
events {
    worker_connections 1024;
}

http {
    upstream m9m {
        server m9m:8080;
    }

    server {
        listen 80;
        server_name your-domain.com;
        return 301 https://$server_name$request_uri;
    }

    server {
        listen 443 ssl http2;
        server_name your-domain.com;

        ssl_certificate /etc/nginx/certs/cert.pem;
        ssl_certificate_key /etc/nginx/certs/key.pem;

        location / {
            proxy_pass http://m9m;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection 'upgrade';
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            proxy_cache_bypass $http_upgrade;
        }

        location /webhook {
            proxy_pass http://m9m;
            proxy_http_version 1.1;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            proxy_read_timeout 120s;
        }
    }
}
```

## Monitoring Stack

```yaml
# docker-compose.monitoring.yml
version: '3.8'

services:
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
    ports:
      - "9091:9090"
    restart: unless-stopped

  grafana:
    image: grafana/grafana:latest
    volumes:
      - grafana_data:/var/lib/grafana
      - ./grafana/dashboards:/etc/grafana/provisioning/dashboards:ro
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    ports:
      - "3000:3000"
    depends_on:
      - prometheus
    restart: unless-stopped

volumes:
  prometheus_data:
  grafana_data:
```

Prometheus config:
```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'm9m'
    static_configs:
      - targets: ['m9m:9090']
```

## Backup & Restore

### Backup

```bash
# Backup database
docker exec m9m-postgres pg_dump -U m9m m9m > backup.sql

# Backup volumes
docker run --rm \
  -v m9m_postgres_data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/postgres-backup.tar.gz /data
```

### Restore

```bash
# Restore database
docker exec -i m9m-postgres psql -U m9m m9m < backup.sql

# Restore volumes
docker run --rm \
  -v m9m_postgres_data:/data \
  -v $(pwd):/backup \
  alpine tar xzf /backup/postgres-backup.tar.gz -C /
```

## Updating

```bash
# Pull latest image
docker compose pull

# Restart with new image
docker compose up -d

# Check logs
docker compose logs -f m9m
```

## Troubleshooting

### View Logs

```bash
docker compose logs -f m9m
docker compose logs -f postgres
```

### Shell Access

```bash
docker exec -it m9m /bin/sh
```

### Check Health

```bash
docker exec m9m curl -s http://localhost:8080/health
```

### Reset Database

```bash
docker compose down -v
docker compose up -d
```

## Next Steps

- [Kubernetes](kubernetes.md) - Orchestrated deployment
- [Production](production.md) - Production best practices
- [Scaling](scaling.md) - Scale your deployment
