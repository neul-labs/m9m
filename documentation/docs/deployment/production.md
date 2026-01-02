# Production Deployment

Best practices for running m9m in production environments.

## Checklist

- [ ] Use PostgreSQL for database
- [ ] Use Redis or RabbitMQ for queue
- [ ] Enable HTTPS
- [ ] Set strong encryption key
- [ ] Configure authentication
- [ ] Set up monitoring
- [ ] Configure logging
- [ ] Set up backups
- [ ] Test disaster recovery
- [ ] Load test before launch

## Security

### Encryption Key

Generate a strong encryption key:

```bash
openssl rand -hex 16
```

Set as environment variable:
```bash
export M9M_ENCRYPTION_KEY="your-32-character-key"
```

### HTTPS

Always use HTTPS in production:

```yaml
# nginx.conf
server {
    listen 443 ssl http2;
    server_name m9m.example.com;

    ssl_certificate /etc/ssl/certs/cert.pem;
    ssl_certificate_key /etc/ssl/private/key.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
    ssl_prefer_server_ciphers off;

    location / {
        proxy_pass http://m9m:8080;
        proxy_set_header X-Forwarded-Proto https;
    }
}
```

### Authentication

Enable API authentication:

```yaml
auth:
  enabled: true
  apiKeys:
    enabled: true
  jwt:
    enabled: true
    secret: "your-jwt-secret"
    expiry: 1h
```

### Secrets Management

Use external secret managers:

```yaml
# Kubernetes external secrets
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: m9m-secrets
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault
    kind: ClusterSecretStore
  target:
    name: m9m-secrets
  data:
  - secretKey: database-url
    remoteRef:
      key: m9m/database
      property: url
```

## Database

### PostgreSQL Configuration

```yaml
# postgresql.conf recommendations
max_connections = 200
shared_buffers = 256MB
effective_cache_size = 768MB
maintenance_work_mem = 64MB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
effective_io_concurrency = 200
min_wal_size = 1GB
max_wal_size = 4GB
max_worker_processes = 4
max_parallel_workers_per_gather = 2
max_parallel_workers = 4
```

### Connection Pooling

Use PgBouncer for connection pooling:

```ini
# pgbouncer.ini
[databases]
m9m = host=postgres port=5432 dbname=m9m

[pgbouncer]
listen_addr = 0.0.0.0
listen_port = 6432
auth_type = md5
auth_file = /etc/pgbouncer/userlist.txt
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 20
```

### Backups

Automated daily backups:

```bash
#!/bin/bash
# backup.sh
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups"

# Database backup
pg_dump -h postgres -U m9m m9m | gzip > "$BACKUP_DIR/db_$DATE.sql.gz"

# Upload to S3
aws s3 cp "$BACKUP_DIR/db_$DATE.sql.gz" s3://m9m-backups/db/

# Cleanup old backups (keep 30 days)
find $BACKUP_DIR -type f -mtime +30 -delete
```

## Queue

### Redis Configuration

```conf
# redis.conf
maxmemory 1gb
maxmemory-policy allkeys-lru
appendonly yes
appendfsync everysec
```

### High Availability

Redis Sentinel for HA:

```yaml
# docker-compose.yml
services:
  redis-master:
    image: redis:7-alpine
    command: redis-server --appendonly yes

  redis-sentinel:
    image: redis:7-alpine
    command: redis-sentinel /etc/redis/sentinel.conf
    volumes:
      - ./sentinel.conf:/etc/redis/sentinel.conf
```

## Monitoring

### Prometheus Metrics

Key metrics to monitor:

| Metric | Alert Threshold |
|--------|-----------------|
| `m9m_workflow_executions_total` | Track trends |
| `m9m_workflow_errors_total` | > 1% error rate |
| `m9m_execution_duration_seconds` | p99 > 30s |
| `m9m_queue_size` | > 1000 pending |
| `m9m_active_workflows` | Capacity planning |

### Alerting Rules

```yaml
# alerts.yaml
groups:
- name: m9m
  rules:
  - alert: HighErrorRate
    expr: rate(m9m_workflow_errors_total[5m]) / rate(m9m_workflow_executions_total[5m]) > 0.01
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: High workflow error rate

  - alert: QueueBacklog
    expr: m9m_queue_size > 1000
    for: 10m
    labels:
      severity: warning
    annotations:
      summary: Queue backlog growing

  - alert: HighLatency
    expr: histogram_quantile(0.99, m9m_execution_duration_seconds_bucket) > 30
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: High workflow execution latency
```

### Grafana Dashboard

Import the m9m dashboard:

```json
{
  "dashboard": {
    "title": "m9m Overview",
    "panels": [
      {
        "title": "Executions/minute",
        "type": "graph",
        "targets": [
          {"expr": "rate(m9m_workflow_executions_total[1m])"}
        ]
      },
      {
        "title": "Error Rate",
        "type": "gauge",
        "targets": [
          {"expr": "rate(m9m_workflow_errors_total[5m]) / rate(m9m_workflow_executions_total[5m])"}
        ]
      }
    ]
  }
}
```

## Logging

### Structured Logging

```yaml
logging:
  level: info
  format: json
  output: stdout
```

### Log Aggregation

Forward logs to centralized logging:

```yaml
# Fluentd config
<source>
  @type tail
  path /var/log/m9m/*.log
  tag m9m
  <parse>
    @type json
  </parse>
</source>

<match m9m.**>
  @type elasticsearch
  host elasticsearch
  port 9200
  index_name m9m-logs
</match>
```

## Performance Tuning

### Worker Configuration

```yaml
queue:
  maxWorkers: 20        # Based on CPU cores
  batchSize: 100        # Items per batch
  pollInterval: 100ms   # Queue poll frequency
```

### Connection Limits

```yaml
database:
  maxConnections: 50
  maxIdleConnections: 10
  connectionTimeout: 5s

http:
  maxConnections: 100
  timeout: 30s
  keepAlive: true
```

### Memory Optimization

```yaml
# Kubernetes resources
resources:
  limits:
    memory: 2Gi
  requests:
    memory: 512Mi

# Go runtime
env:
  - name: GOGC
    value: "100"
  - name: GOMEMLIMIT
    value: "1800MiB"
```

## High Availability

### Multi-Region

```
                    ┌─────────────────┐
                    │   Global LB     │
                    └────────┬────────┘
           ┌─────────────────┼─────────────────┐
           │                 │                 │
    ┌──────┴──────┐   ┌──────┴──────┐   ┌──────┴──────┐
    │  Region A   │   │  Region B   │   │  Region C   │
    │  (Primary)  │   │  (Standby)  │   │  (Standby)  │
    └─────────────┘   └─────────────┘   └─────────────┘
```

### Database Replication

```yaml
# PostgreSQL streaming replication
primary:
  postgresql.conf:
    wal_level: replica
    max_wal_senders: 3

replica:
  recovery.conf:
    standby_mode: on
    primary_conninfo: 'host=primary user=replicator'
```

## Disaster Recovery

### RTO/RPO Targets

| Scenario | RTO | RPO |
|----------|-----|-----|
| Database failure | 5 min | 1 min |
| Region failure | 15 min | 5 min |
| Full outage | 1 hour | 15 min |

### Recovery Procedures

1. **Database Recovery**
   ```bash
   # Restore from backup
   gunzip -c backup.sql.gz | psql -h postgres -U m9m m9m
   ```

2. **Failover to Standby**
   ```bash
   # Promote standby
   pg_ctl promote -D /var/lib/postgresql/data
   ```

3. **Queue Recovery**
   ```bash
   # Redis failover
   redis-cli SENTINEL failover m9m-master
   ```

## Maintenance

### Rolling Updates

```bash
# Kubernetes
kubectl set image deployment/m9m m9m=m9m/m9m:v1.2.0 --record

# Docker Compose
docker compose pull
docker compose up -d --no-deps m9m
```

### Health Checks

```bash
# Automated health check script
#!/bin/bash
HEALTH=$(curl -s http://localhost:8080/health)
if [ "$HEALTH" != '{"status":"healthy"}' ]; then
  echo "Health check failed: $HEALTH"
  exit 1
fi
```

## Next Steps

- [Scaling](scaling.md) - Scale your deployment
- [Configuration](../reference/configuration.md) - All configuration options
- [Troubleshooting](../reference/troubleshooting.md) - Common issues
