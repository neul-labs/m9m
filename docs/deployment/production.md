# Production Deployment Guide

This comprehensive guide covers deploying m9m in production environments, including infrastructure setup, security configuration, monitoring, and operational best practices.

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Infrastructure Requirements](#infrastructure-requirements)
3. [Installation Methods](#installation-methods)
4. [Configuration](#configuration)
5. [Security Setup](#security-setup)
6. [Load Balancing](#load-balancing)
7. [Database Configuration](#database-configuration)
8. [Monitoring and Logging](#monitoring-and-logging)
9. [Backup and Recovery](#backup-and-recovery)
10. [Scaling and Performance](#scaling-and-performance)
11. [Troubleshooting](#troubleshooting)

## Architecture Overview

### Recommended Production Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Load Balancer │    │     Firewall    │    │   Monitoring    │
│    (nginx)      │────│   (iptables)    │────│  (Prometheus)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
          │                       │                       │
          ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   m9m-1      │    │   m9m-2      │    │   Log Aggregator│
│   (Primary)     │────│   (Secondary)   │────│    (Fluentd)    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
          │                       │                       │
          ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Database      │    │   Redis Cache   │    │   File Storage  │
│  (PostgreSQL)   │    │   (Optional)    │    │     (NFS)       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Key Components

- **Load Balancer**: Distributes traffic and provides SSL termination
- **m9m Instances**: Multiple instances for high availability
- **Database**: PostgreSQL for persistent storage
- **Monitoring**: Prometheus + Grafana for observability
- **Logging**: Centralized logging with Fluentd/ELK stack

## Infrastructure Requirements

### Minimum System Requirements

| Component | Minimum | Recommended | High Load |
|-----------|---------|-------------|-----------|
| **CPU** | 2 cores | 4 cores | 8+ cores |
| **Memory** | 2 GB | 8 GB | 16+ GB |
| **Storage** | 20 GB | 100 GB | 500+ GB |
| **Network** | 100 Mbps | 1 Gbps | 10+ Gbps |

### Operating System Support

- **Linux**: Ubuntu 20.04+, CentOS 8+, RHEL 8+, Debian 11+
- **Container**: Docker 20.10+, Kubernetes 1.20+
- **Cloud**: AWS, GCP, Azure, DigitalOcean

### Network Requirements

```bash
# Inbound ports
443/tcp   # HTTPS (external)
80/tcp    # HTTP (redirect to HTTPS)
3000/tcp  # m9m API (internal)
9090/tcp  # Metrics (internal)
8080/tcp  # Health checks (internal)

# Outbound ports
443/tcp   # HTTPS (external APIs)
80/tcp    # HTTP (external APIs)
5432/tcp  # PostgreSQL (internal)
6379/tcp  # Redis (internal)
```

## Installation Methods

### Method 1: Binary Installation

#### Download and Install

```bash
# Create installation directory
sudo mkdir -p /opt/m9m
cd /opt/m9m

# Download latest release
LATEST_VERSION=$(curl -s https://api.github.com/repos/m9m/m9m/releases/latest | grep tag_name | cut -d '"' -f 4)
curl -L "https://github.com/m9m/m9m/releases/download/${LATEST_VERSION}/m9m-linux-amd64" -o m9m

# Make executable
chmod +x m9m

# Create directories
sudo mkdir -p /opt/m9m/{config,data,logs,workflows,credentials}

# Set ownership
sudo useradd -r -s /bin/false -d /opt/m9m m9m
sudo chown -R m9m:m9m /opt/m9m
```

#### Create Systemd Service

```bash
sudo tee /etc/systemd/system/m9m.service << 'EOF'
[Unit]
Description=m9m Workflow Automation Engine
Documentation=https://docs.m9m.com
After=network.target postgresql.service
Wants=postgresql.service

[Service]
Type=simple
User=m9m
Group=m9m
WorkingDirectory=/opt/m9m
ExecStart=/opt/m9m/m9m server --config /opt/m9m/config/production.yaml
ExecReload=/bin/kill -HUP $MAINPID
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=m9m

# Security settings
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/m9m/data /opt/m9m/logs
PrivateTmp=true
ProtectKernelTunables=true
ProtectControlGroups=true
RestrictRealtime=true
MemoryDenyWriteExecute=true
LimitNOFILE=65536

# Environment
Environment=NODE_ENV=production
EnvironmentFile=-/opt/m9m/config/.env

[Install]
WantedBy=multi-user.target
EOF

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable m9m
```

### Method 2: Docker Deployment

#### Docker Compose Setup

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  m9m:
    image: m9m:latest
    container_name: m9m
    restart: unless-stopped
    ports:
      - "3000:3000"
      - "9090:9090"  # Metrics
    volumes:
      - ./config:/config:ro
      - ./data:/data:rw
      - ./workflows:/workflows:ro
      - ./logs:/logs:rw
    environment:
      - NODE_ENV=production
      - DB_TYPE=postgres
      - DB_HOST=postgres
      - DB_DATABASE=n8n
      - DB_USERNAME=n8n
      - DB_PASSWORD_FILE=/run/secrets/db_password
    secrets:
      - db_password
      - encryption_key
    depends_on:
      postgres:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    cap_add:
      - NET_BIND_SERVICE
    read_only: true
    tmpfs:
      - /tmp:noexec,nosuid,size=100m
    networks:
      - n8n-network

  postgres:
    image: postgres:15-alpine
    container_name: n8n-postgres
    restart: unless-stopped
    environment:
      - POSTGRES_DB=n8n
      - POSTGRES_USER=n8n
      - POSTGRES_PASSWORD_FILE=/run/secrets/db_password
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./postgres/init:/docker-entrypoint-initdb.d:ro
    secrets:
      - db_password
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U n8n -d n8n"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - n8n-network

  nginx:
    image: nginx:alpine
    container_name: n8n-nginx
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/ssl:/etc/nginx/ssl:ro
      - ./nginx/logs:/var/log/nginx:rw
    depends_on:
      - m9m
    networks:
      - n8n-network

  prometheus:
    image: prom/prometheus:latest
    container_name: n8n-prometheus
    restart: unless-stopped
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
      - '--storage.tsdb.retention.time=200h'
      - '--web.enable-lifecycle'
    networks:
      - n8n-network

  grafana:
    image: grafana/grafana:latest
    container_name: n8n-grafana
    restart: unless-stopped
    ports:
      - "3001:3000"
    volumes:
      - grafana_data:/var/lib/grafana
      - ./grafana/dashboards:/etc/grafana/provisioning/dashboards:ro
      - ./grafana/datasources:/etc/grafana/provisioning/datasources:ro
    environment:
      - GF_SECURITY_ADMIN_PASSWORD_FILE=/run/secrets/grafana_password
    secrets:
      - grafana_password
    networks:
      - n8n-network

secrets:
  db_password:
    external: true
  encryption_key:
    external: true
  grafana_password:
    external: true

volumes:
  postgres_data:
  prometheus_data:
  grafana_data:

networks:
  n8n-network:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
```

#### Create Docker Secrets

```bash
# Create secrets
echo "your-secure-db-password" | docker secret create db_password -
echo "$(openssl rand -hex 32)" | docker secret create encryption_key -
echo "your-grafana-admin-password" | docker secret create grafana_password -

# Deploy stack
docker-compose -f docker-compose.prod.yml up -d
```

### Method 3: Kubernetes Deployment

#### Namespace and ConfigMap

```yaml
# k8s/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: m9m
  labels:
    name: m9m
---
# k8s/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: m9m-config
  namespace: m9m
data:
  production.yaml: |
    server:
      port: 3000
      host: "0.0.0.0"

    database:
      type: postgres
      host: postgres-service
      port: 5432
      database: n8n
      username: n8n
      # Password from secret

    security:
      tls:
        enabled: false  # Terminated at ingress
      rateLimiting:
        enabled: true
        requests: 100
        window: "1m"

    monitoring:
      metrics:
        enabled: true
        port: 9090
      health:
        enabled: true
        port: 8080

    logging:
      level: info
      format: json
```

#### Deployment

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: m9m
  namespace: m9m
  labels:
    app: m9m
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
  selector:
    matchLabels:
      app: m9m
  template:
    metadata:
      labels:
        app: m9m
    spec:
      serviceAccountName: m9m
      securityContext:
        runAsNonRoot: true
        runAsUser: 65534
        fsGroup: 65534
      containers:
      - name: m9m
        image: m9m:latest
        imagePullPolicy: Always
        ports:
        - containerPort: 3000
          name: http
        - containerPort: 9090
          name: metrics
        - containerPort: 8080
          name: health
        env:
        - name: NODE_ENV
          value: "production"
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: m9m-secrets
              key: db-password
        - name: ENCRYPTION_KEY
          valueFrom:
            secretKeyRef:
              name: m9m-secrets
              key: encryption-key
        volumeMounts:
        - name: config
          mountPath: /config
          readOnly: true
        - name: data
          mountPath: /data
        resources:
          requests:
            memory: "512Mi"
            cpu: "250m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop:
            - ALL
      volumes:
      - name: config
        configMap:
          name: m9m-config
      - name: data
        persistentVolumeClaim:
          claimName: m9m-data
```

## Configuration

### Production Configuration File

```yaml
# config/production.yaml
# m9m Production Configuration

# Server settings
server:
  port: 3000
  host: "0.0.0.0"
  readTimeout: "30s"
  writeTimeout: "30s"
  idleTimeout: "120s"
  maxHeaderBytes: 1048576  # 1MB

# Database configuration
database:
  type: postgres
  host: "${DB_HOST:localhost}"
  port: "${DB_PORT:5432}"
  database: "${DB_DATABASE:n8n}"
  username: "${DB_USERNAME:n8n}"
  password: "${DB_PASSWORD}"
  maxOpenConns: 25
  maxIdleConns: 10
  connMaxLifetime: "1h"
  sslMode: "require"

# Security settings
security:
  # Encryption
  encryptionKey: "${ENCRYPTION_KEY}"

  # TLS configuration
  tls:
    enabled: true
    certFile: "${TLS_CERT_FILE:/etc/ssl/certs/m9m.crt}"
    keyFile: "${TLS_KEY_FILE:/etc/ssl/private/m9m.key}"
    minVersion: "1.3"
    cipherSuites:
      - "TLS_AES_256_GCM_SHA384"
      - "TLS_AES_128_GCM_SHA256"
      - "TLS_CHACHA20_POLY1305_SHA256"

  # HTTP security headers
  headers:
    strictTransportSecurity: "max-age=31536000; includeSubDomains; preload"
    contentSecurityPolicy: "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'"
    xFrameOptions: "DENY"
    xContentTypeOptions: "nosniff"
    referrerPolicy: "strict-origin-when-cross-origin"
    permissionsPolicy: "geolocation=(), microphone=(), camera=()"

  # Rate limiting
  rateLimiting:
    enabled: true
    requests: 100
    window: "1m"
    burst: 10
    skipIPRanges:
      - "10.0.0.0/8"
      - "172.16.0.0/12"
      - "192.168.0.0/16"

  # Resource limits
  limits:
    maxRequestSize: "10MB"
    maxExecutionTime: "300s"
    maxMemoryUsage: "512MB"
    maxConcurrentExecutions: 50

# Webhook configuration
webhooks:
  enabled: true
  path: "/webhook"
  timeout: "30s"
  maxPayloadSize: "10MB"
  authentication:
    default: "header"
    requireAuth: true

# Monitoring and observability
monitoring:
  # Metrics
  metrics:
    enabled: true
    port: 9090
    path: "/metrics"
    interval: "15s"

  # Health checks
  health:
    enabled: true
    port: 8080
    path: "/health"
    checks:
      - "database"
      - "memory"
      - "disk"

  # Tracing
  tracing:
    enabled: true
    jaegerEndpoint: "${JAEGER_ENDPOINT}"
    samplingRate: 0.1

# Logging configuration
logging:
  level: "info"
  format: "json"
  output: "/opt/m9m/logs/m9m.log"
  maxSize: 100  # MB
  maxBackups: 10
  maxAge: 30    # days
  compress: true

  # Audit logging
  audit:
    enabled: true
    logFile: "/opt/m9m/logs/audit.log"
    events:
      - "authentication"
      - "authorization"
      - "workflow_execution"
      - "configuration_change"

# Workflow execution
execution:
  # Concurrency
  maxConcurrent: 50
  queueMode: "memory"  # or "redis"

  # Timeouts
  defaultTimeout: "300s"
  maxTimeout: "3600s"

  # Retry settings
  defaultRetries: 3
  retryDelay: "1s"
  maxRetryDelay: "30s"

  # Data retention
  retentionDays: 30
  cleanupInterval: "24h"

# Credentials
credentials:
  encryptionKey: "${ENCRYPTION_KEY}"
  storageType: "database"  # or "file"

# Performance tuning
performance:
  # Expression evaluation
  expressionCacheSize: 1000
  expressionCacheTTL: "1h"

  # Connection pooling
  httpClientTimeout: "30s"
  httpClientMaxIdleConns: 100
  httpClientMaxConnsPerHost: 10

  # Memory management
  gcPercentage: 100
  maxHeapSize: "1GB"

# Feature flags
features:
  enableMetrics: true
  enableTracing: true
  enableAuditLog: true
  enableRateLimiting: true
  enableTLS: true
```

### Environment Variables

```bash
# .env
NODE_ENV=production

# Database
DB_TYPE=postgres
DB_HOST=db.example.com
DB_PORT=5432
DB_DATABASE=n8n
DB_USERNAME=n8n
DB_PASSWORD=secure-password

# Security
ENCRYPTION_KEY=your-32-byte-encryption-key-here
TLS_CERT_FILE=/etc/ssl/certs/m9m.crt
TLS_KEY_FILE=/etc/ssl/private/m9m.key

# External services
JAEGER_ENDPOINT=http://jaeger:14268/api/traces
PROMETHEUS_ENDPOINT=http://prometheus:9090

# Application
LOG_LEVEL=info
MAX_CONCURRENT_EXECUTIONS=50
WEBHOOK_TIMEOUT=30s
```

## Security Setup

### SSL/TLS Configuration

#### Generate SSL Certificate

```bash
# Self-signed certificate (development)
openssl req -x509 -newkey rsa:4096 -keyout m9m.key -out m9m.crt -days 365 -nodes \
  -subj "/C=US/ST=State/L=City/O=Organization/CN=m9m.example.com"

# Let's Encrypt certificate (production)
certbot certonly --standalone -d m9m.example.com
```

#### Nginx SSL Configuration

```nginx
# /etc/nginx/sites-available/m9m
server {
    listen 80;
    server_name m9m.example.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name m9m.example.com;

    # SSL configuration
    ssl_certificate /etc/letsencrypt/live/m9m.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/m9m.example.com/privkey.pem;
    ssl_protocols TLSv1.3 TLSv1.2;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains; preload" always;
    add_header X-Frame-Options DENY always;
    add_header X-Content-Type-Options nosniff always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
    limit_req zone=api burst=20 nodelay;

    # Proxy settings
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_connect_timeout 30s;
    proxy_send_timeout 30s;
    proxy_read_timeout 30s;

    # Main application
    location / {
        proxy_pass http://127.0.0.1:3000;
    }

    # Webhook endpoints
    location /webhook {
        proxy_pass http://127.0.0.1:3000;
        proxy_buffering off;
        proxy_request_buffering off;
    }

    # Health check
    location /health {
        proxy_pass http://127.0.0.1:8080;
        access_log off;
    }

    # Metrics (internal only)
    location /metrics {
        proxy_pass http://127.0.0.1:9090;
        allow 10.0.0.0/8;
        allow 172.16.0.0/12;
        allow 192.168.0.0/16;
        deny all;
    }
}
```

### Firewall Configuration

```bash
# iptables rules
#!/bin/bash

# Clear existing rules
iptables -F
iptables -X
iptables -t nat -F
iptables -t nat -X

# Default policies
iptables -P INPUT DROP
iptables -P FORWARD DROP
iptables -P OUTPUT ACCEPT

# Allow loopback
iptables -A INPUT -i lo -j ACCEPT

# Allow established connections
iptables -A INPUT -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT

# Allow SSH (restrict to management IPs)
iptables -A INPUT -p tcp --dport 22 -s 10.0.0.0/8 -j ACCEPT
iptables -A INPUT -p tcp --dport 22 -s 172.16.0.0/12 -j ACCEPT

# Allow HTTP/HTTPS
iptables -A INPUT -p tcp --dport 80 -j ACCEPT
iptables -A INPUT -p tcp --dport 443 -j ACCEPT

# Allow health checks from load balancer
iptables -A INPUT -p tcp --dport 8080 -s 10.0.0.0/8 -j ACCEPT

# Allow metrics from monitoring
iptables -A INPUT -p tcp --dport 9090 -s 10.0.0.0/8 -j ACCEPT

# Drop everything else
iptables -A INPUT -j DROP

# Save rules
iptables-save > /etc/iptables/rules.v4
```

## Load Balancing

### HAProxy Configuration

```
# /etc/haproxy/haproxy.cfg
global
    daemon
    log 127.0.0.1:514 local0
    chroot /var/lib/haproxy
    stats socket /run/haproxy/admin.sock mode 660 level admin
    stats timeout 30s
    user haproxy
    group haproxy

    # SSL
    ssl-default-bind-ciphers ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384
    ssl-default-bind-options ssl-min-ver TLSv1.2 no-tls-tickets

defaults
    mode http
    log global
    option httplog
    option dontlognull
    option log-health-checks
    option forwardfor
    option http-server-close
    timeout connect 5000
    timeout client 50000
    timeout server 50000
    errorfile 400 /etc/haproxy/errors/400.http
    errorfile 403 /etc/haproxy/errors/403.http
    errorfile 408 /etc/haproxy/errors/408.http
    errorfile 500 /etc/haproxy/errors/500.http
    errorfile 502 /etc/haproxy/errors/502.http
    errorfile 503 /etc/haproxy/errors/503.http
    errorfile 504 /etc/haproxy/errors/504.http

frontend n8n_frontend
    bind *:80
    bind *:443 ssl crt /etc/ssl/certs/m9m.pem

    # Redirect HTTP to HTTPS
    redirect scheme https code 301 if !{ ssl_fc }

    # Security headers
    http-response set-header Strict-Transport-Security "max-age=31536000; includeSubDomains; preload"
    http-response set-header X-Frame-Options DENY
    http-response set-header X-Content-Type-Options nosniff

    # Rate limiting
    stick-table type ip size 100k expire 30s store http_req_rate(10s)
    http-request track-sc0 src
    http-request reject if { sc_http_req_rate(0) gt 20 }

    # Backend selection
    use_backend n8n_backend

backend n8n_backend
    balance roundrobin
    option httpchk GET /health HTTP/1.1\r\nHost:\ localhost

    # Health check configuration
    default-server check inter 5s rise 2 fall 3

    # Backend servers
    server m9m-1 10.0.1.10:3000 check
    server m9m-2 10.0.1.11:3000 check
    server m9m-3 10.0.1.12:3000 check backup

# Statistics
listen stats
    bind *:8404
    stats enable
    stats uri /stats
    stats refresh 30s
    stats admin if TRUE
```

## Database Configuration

### PostgreSQL Setup

#### Installation and Configuration

```bash
# Install PostgreSQL
sudo apt-get update
sudo apt-get install -y postgresql-15 postgresql-contrib-15

# Configure PostgreSQL
sudo -u postgres createuser --interactive n8n
sudo -u postgres createdb n8n -O n8n

# Set password
sudo -u postgres psql -c "ALTER USER n8n PASSWORD 'secure-password';"
```

#### PostgreSQL Configuration

```sql
-- /etc/postgresql/15/main/postgresql.conf

# Connection settings
listen_addresses = '*'
port = 5432
max_connections = 100
shared_buffers = 256MB
effective_cache_size = 1GB
work_mem = 4MB
maintenance_work_mem = 64MB

# WAL settings
wal_level = replica
max_wal_size = 1GB
min_wal_size = 80MB
checkpoint_completion_target = 0.9

# Performance settings
random_page_cost = 1.1
effective_io_concurrency = 200

# Logging
log_destination = 'stderr'
log_statement = 'mod'
log_min_duration_statement = 1000
log_checkpoints = on
log_connections = on
log_disconnections = on
log_lock_waits = on

# Autovacuum
autovacuum = on
log_autovacuum_min_duration = 0
```

#### Database Schema

```sql
-- m9m database schema
CREATE TABLE IF NOT EXISTS workflows (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT false,
    workflow_data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS executions (
    id SERIAL PRIMARY KEY,
    workflow_id INTEGER REFERENCES workflows(id),
    status VARCHAR(50) NOT NULL,
    execution_data JSONB,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    finished_at TIMESTAMP,
    error_message TEXT
);

CREATE TABLE IF NOT EXISTS credentials (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL,
    encrypted_data TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_workflows_active ON workflows(active);
CREATE INDEX idx_executions_workflow_id ON executions(workflow_id);
CREATE INDEX idx_executions_status ON executions(status);
CREATE INDEX idx_executions_started_at ON executions(started_at);
CREATE INDEX idx_credentials_type ON credentials(type);
```

### Database Backup

```bash
#!/bin/bash
# backup-database.sh

DB_NAME="n8n"
DB_USER="n8n"
BACKUP_DIR="/opt/backups/database"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Create backup directory
mkdir -p $BACKUP_DIR

# Create backup
pg_dump -h localhost -U $DB_USER -d $DB_NAME -F c -b -v -f "$BACKUP_DIR/n8n_backup_$TIMESTAMP.dump"

# Compress backup
gzip "$BACKUP_DIR/n8n_backup_$TIMESTAMP.dump"

# Clean old backups (keep 7 days)
find $BACKUP_DIR -name "*.gz" -mtime +7 -delete

echo "Backup completed: n8n_backup_$TIMESTAMP.dump.gz"
```

## Monitoring and Logging

### Prometheus Configuration

```yaml
# prometheus/prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "n8n_rules.yml"

scrape_configs:
  - job_name: 'm9m'
    static_configs:
      - targets: ['localhost:9090']
    scrape_interval: 15s
    metrics_path: /metrics

  - job_name: 'postgres'
    static_configs:
      - targets: ['localhost:9187']

  - job_name: 'node-exporter'
    static_configs:
      - targets: ['localhost:9100']

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - localhost:9093
```

### Grafana Dashboard

```json
{
  "dashboard": {
    "id": null,
    "title": "m9m Production Dashboard",
    "tags": ["m9m", "production"],
    "timezone": "browser",
    "panels": [
      {
        "title": "Workflow Executions per Second",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(n8n_workflow_executions_total[5m])",
            "legendFormat": "Executions/sec"
          }
        ]
      },
      {
        "title": "Active Workflows",
        "type": "singlestat",
        "targets": [
          {
            "expr": "n8n_workflows_active_total",
            "legendFormat": "Active Workflows"
          }
        ]
      },
      {
        "title": "Error Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(n8n_workflow_executions_failed_total[5m])",
            "legendFormat": "Errors/sec"
          }
        ]
      },
      {
        "title": "Response Time",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(n8n_http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "95th percentile"
          }
        ]
      }
    ]
  }
}
```

### Log Configuration

```yaml
# fluentd/fluent.conf
<source>
  @type tail
  path /opt/m9m/logs/m9m.log
  pos_file /var/log/fluentd/m9m.log.pos
  tag n8n.application
  format json
  time_key timestamp
  time_format %Y-%m-%dT%H:%M:%S.%LZ
</source>

<source>
  @type tail
  path /opt/m9m/logs/audit.log
  pos_file /var/log/fluentd/audit.log.pos
  tag n8n.audit
  format json
  time_key timestamp
  time_format %Y-%m-%dT%H:%M:%S.%LZ
</source>

<filter n8n.**>
  @type record_transformer
  <record>
    hostname ${hostname}
    service m9m
  </record>
</filter>

<match n8n.**>
  @type elasticsearch
  host elasticsearch.example.com
  port 9200
  index_name m9m
  type_name _doc
  include_tag_key true
  tag_key @log_name
  flush_interval 10s
</match>
```

## Backup and Recovery

### Comprehensive Backup Strategy

```bash
#!/bin/bash
# comprehensive-backup.sh

BACKUP_ROOT="/opt/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=30

# Create backup directories
mkdir -p "$BACKUP_ROOT"/{database,workflows,credentials,config,logs}

echo "Starting comprehensive backup at $(date)"

# Database backup
echo "Backing up database..."
pg_dump -h localhost -U n8n -d n8n -F c -b -v -f "$BACKUP_ROOT/database/n8n_db_$TIMESTAMP.dump"
gzip "$BACKUP_ROOT/database/n8n_db_$TIMESTAMP.dump"

# Workflows backup
echo "Backing up workflows..."
tar -czf "$BACKUP_ROOT/workflows/workflows_$TIMESTAMP.tar.gz" -C /opt/m9m workflows/

# Credentials backup
echo "Backing up credentials..."
tar -czf "$BACKUP_ROOT/credentials/credentials_$TIMESTAMP.tar.gz" -C /opt/m9m credentials/

# Configuration backup
echo "Backing up configuration..."
tar -czf "$BACKUP_ROOT/config/config_$TIMESTAMP.tar.gz" -C /opt/m9m config/

# Log backup
echo "Backing up logs..."
tar -czf "$BACKUP_ROOT/logs/logs_$TIMESTAMP.tar.gz" -C /opt/m9m logs/

# Upload to cloud storage (optional)
if command -v aws &> /dev/null; then
    echo "Uploading to S3..."
    aws s3 sync "$BACKUP_ROOT" s3://your-backup-bucket/m9m/
fi

# Cleanup old backups
echo "Cleaning up old backups..."
find "$BACKUP_ROOT" -name "*.gz" -mtime +$RETENTION_DAYS -delete
find "$BACKUP_ROOT" -name "*.dump.gz" -mtime +$RETENTION_DAYS -delete

echo "Backup completed at $(date)"
```

### Recovery Procedures

```bash
#!/bin/bash
# recovery.sh

BACKUP_FILE="$1"
RECOVERY_TYPE="$2"

if [ -z "$BACKUP_FILE" ] || [ -z "$RECOVERY_TYPE" ]; then
    echo "Usage: $0 <backup_file> <recovery_type>"
    echo "Recovery types: database, workflows, credentials, config, full"
    exit 1
fi

case $RECOVERY_TYPE in
    "database")
        echo "Recovering database from $BACKUP_FILE..."
        systemctl stop m9m
        sudo -u postgres dropdb n8n
        sudo -u postgres createdb n8n -O n8n
        gunzip -c "$BACKUP_FILE" | pg_restore -h localhost -U n8n -d n8n
        systemctl start m9m
        ;;

    "workflows")
        echo "Recovering workflows from $BACKUP_FILE..."
        systemctl stop m9m
        rm -rf /opt/m9m/workflows/*
        tar -xzf "$BACKUP_FILE" -C /opt/m9m/
        chown -R m9m:m9m /opt/m9m/workflows/
        systemctl start m9m
        ;;

    "full")
        echo "Performing full recovery..."
        # Implement full system recovery
        ;;
esac
```

## Scaling and Performance

### Horizontal Scaling

#### Multi-Instance Setup

```yaml
# docker-compose.scale.yml
version: '3.8'

services:
  m9m-1:
    image: m9m:latest
    environment:
      - INSTANCE_ID=m9m-1
      - CLUSTER_MODE=true
    depends_on:
      - postgres
      - redis

  m9m-2:
    image: m9m:latest
    environment:
      - INSTANCE_ID=m9m-2
      - CLUSTER_MODE=true
    depends_on:
      - postgres
      - redis

  m9m-3:
    image: m9m:latest
    environment:
      - INSTANCE_ID=m9m-3
      - CLUSTER_MODE=true
    depends_on:
      - postgres
      - redis

  redis:
    image: redis:alpine
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
```

### Performance Tuning

#### System-Level Optimizations

```bash
# /etc/sysctl.conf
# Network optimizations
net.core.somaxconn = 32768
net.core.netdev_max_backlog = 5000
net.ipv4.tcp_max_syn_backlog = 8192
net.ipv4.tcp_slow_start_after_idle = 0
net.ipv4.tcp_tw_reuse = 1

# Memory optimizations
vm.swappiness = 10
vm.dirty_ratio = 15
vm.dirty_background_ratio = 5

# File descriptor limits
fs.file-max = 1000000
```

#### Application-Level Tuning

```yaml
# Performance configuration
performance:
  # Goroutine pool
  maxGoroutines: 1000
  goroutinePoolSize: 100

  # Memory management
  gcTargetPercentage: 100
  memoryLimit: "2GB"

  # Connection pooling
  httpClient:
    maxIdleConns: 100
    maxConnsPerHost: 10
    idleConnTimeout: "90s"

  # Caching
  cache:
    expressionCacheSize: 10000
    expressionCacheTTL: "1h"
    workflowCacheSize: 1000
    workflowCacheTTL: "10m"
```

## Troubleshooting

### Common Issues and Solutions

#### High Memory Usage

```bash
# Check memory usage
ps aux | grep m9m
top -p $(pgrep m9m)

# Enable memory profiling
./m9m server --profile-memory --config /opt/m9m/config/production.yaml

# Analyze memory profile
go tool pprof http://localhost:6060/debug/pprof/heap
```

#### Database Connection Issues

```bash
# Check database connectivity
pg_isready -h localhost -p 5432 -U n8n

# Monitor database connections
sudo -u postgres psql -c "SELECT count(*) FROM pg_stat_activity WHERE state = 'active';"

# Check connection pool status
curl http://localhost:9090/metrics | grep n8n_db_connections
```

#### Performance Bottlenecks

```bash
# Enable performance profiling
./m9m server --profile-cpu --config /opt/m9m/config/production.yaml

# Analyze CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile

# Check workflow execution metrics
curl http://localhost:9090/metrics | grep n8n_workflow_execution
```

### Diagnostic Scripts

```bash
#!/bin/bash
# diagnostic.sh

echo "=== m9m System Diagnostics ==="
echo "Date: $(date)"
echo "Host: $(hostname)"
echo

echo "=== Service Status ==="
systemctl status m9m
echo

echo "=== Resource Usage ==="
echo "CPU Usage:"
top -bn1 | grep "Cpu(s)" | awk '{print $2 + $4}' | sed 's/%us,/% CPU usage/'

echo "Memory Usage:"
free -h

echo "Disk Usage:"
df -h /opt/m9m
echo

echo "=== Network Connections ==="
netstat -tlnp | grep m9m
echo

echo "=== Recent Logs ==="
journalctl -u m9m --no-pager -n 20
echo

echo "=== Database Status ==="
sudo -u postgres psql -c "SELECT version();"
sudo -u postgres psql -c "SELECT count(*) as active_connections FROM pg_stat_activity;"
echo

echo "=== Health Check ==="
curl -s http://localhost:8080/health | jq .
echo

echo "=== Metrics Sample ==="
curl -s http://localhost:9090/metrics | grep -E "n8n_workflow|n8n_execution" | head -10
```

### Performance Monitoring

```bash
#!/bin/bash
# performance-monitor.sh

LOGFILE="/var/log/m9m-performance.log"

while true; do
    TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')

    # CPU usage
    CPU=$(top -bn1 | grep "Cpu(s)" | awk '{print $2 + $4}' | sed 's/%us,//')

    # Memory usage
    MEMORY=$(free | grep Mem | awk '{printf("%.1f", $3/$2 * 100.0)}')

    # Active connections
    CONNECTIONS=$(netstat -an | grep :3000 | grep ESTABLISHED | wc -l)

    # Workflow executions (from metrics)
    EXECUTIONS=$(curl -s http://localhost:9090/metrics | grep 'n8n_workflow_executions_total' | awk '{print $2}')

    echo "$TIMESTAMP CPU:${CPU}% MEM:${MEMORY}% CONN:$CONNECTIONS EXEC:$EXECUTIONS" >> $LOGFILE

    sleep 60
done
```

## Conclusion

This production deployment guide provides comprehensive instructions for deploying m9m in enterprise environments. Key considerations for successful deployment include:

1. **Security First**: Implement all security measures before going live
2. **Monitoring**: Set up comprehensive monitoring and alerting
3. **Backup Strategy**: Ensure reliable backup and recovery procedures
4. **Performance Tuning**: Optimize for your specific workload
5. **High Availability**: Plan for redundancy and failover
6. **Documentation**: Maintain operational runbooks and procedures

For additional support or specific deployment scenarios, consult the documentation or reach out to the m9m community.