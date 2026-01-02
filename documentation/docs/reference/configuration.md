# Configuration Reference

Complete reference for all m9m configuration options.

## Configuration Methods

Configuration can be set via:

1. **Configuration file** (`config.yaml`)
2. **Environment variables** (prefix: `M9M_`)
3. **Command-line flags**

Priority: CLI flags > Environment variables > Config file > Defaults

## Server Configuration

### Core Settings

| Option | Env Var | Default | Description |
|--------|---------|---------|-------------|
| `server.port` | `M9M_PORT` | `8080` | HTTP server port |
| `server.host` | `M9M_HOST` | `0.0.0.0` | Server bind address |
| `server.baseUrl` | `M9M_BASE_URL` | `http://localhost:8080` | Public URL |

```yaml
server:
  port: 8080
  host: "0.0.0.0"
  baseUrl: "https://m9m.example.com"
```

### HTTP Settings

| Option | Env Var | Default | Description |
|--------|---------|---------|-------------|
| `server.readTimeout` | `M9M_READ_TIMEOUT` | `30s` | Request read timeout |
| `server.writeTimeout` | `M9M_WRITE_TIMEOUT` | `30s` | Response write timeout |
| `server.idleTimeout` | `M9M_IDLE_TIMEOUT` | `120s` | Keep-alive timeout |
| `server.maxHeaderBytes` | `M9M_MAX_HEADER_BYTES` | `1MB` | Max header size |

```yaml
server:
  readTimeout: 30s
  writeTimeout: 30s
  idleTimeout: 120s
  maxHeaderBytes: 1048576
```

## Database Configuration

### Connection Settings

| Option | Env Var | Default | Description |
|--------|---------|---------|-------------|
| `database.type` | `M9M_DB_TYPE` | `sqlite` | Database type |
| `database.url` | `M9M_DB_URL` | `file:m9m.db` | Connection URL |

Supported types: `sqlite`, `postgres`, `mysql`

```yaml
database:
  type: postgres
  url: "postgres://user:pass@localhost:5432/m9m?sslmode=require"
```

### Pool Settings

| Option | Env Var | Default | Description |
|--------|---------|---------|-------------|
| `database.maxConnections` | `M9M_DB_MAX_CONNS` | `25` | Max open connections |
| `database.maxIdleConnections` | `M9M_DB_MAX_IDLE` | `5` | Max idle connections |
| `database.connectionTimeout` | `M9M_DB_CONN_TIMEOUT` | `5s` | Connection timeout |
| `database.connMaxLifetime` | `M9M_DB_CONN_LIFETIME` | `1h` | Max connection age |

```yaml
database:
  maxConnections: 50
  maxIdleConnections: 10
  connectionTimeout: 5s
  connMaxLifetime: 1h
```

## Queue Configuration

### Queue Settings

| Option | Env Var | Default | Description |
|--------|---------|---------|-------------|
| `queue.type` | `M9M_QUEUE_TYPE` | `memory` | Queue backend |
| `queue.url` | `M9M_QUEUE_URL` | - | Queue connection URL |
| `queue.maxWorkers` | `M9M_MAX_WORKERS` | `10` | Concurrent workers |

Supported types: `memory`, `redis`, `rabbitmq`

```yaml
queue:
  type: redis
  url: "redis://:password@localhost:6379/0"
  maxWorkers: 20
```

### Worker Settings

| Option | Env Var | Default | Description |
|--------|---------|---------|-------------|
| `queue.batchSize` | `M9M_BATCH_SIZE` | `10` | Items per batch |
| `queue.pollInterval` | `M9M_POLL_INTERVAL` | `100ms` | Queue poll frequency |
| `queue.visibilityTimeout` | `M9M_VISIBILITY_TIMEOUT` | `30s` | Task lock timeout |
| `queue.retryDelay` | `M9M_RETRY_DELAY` | `1s` | Retry backoff |
| `queue.maxRetries` | `M9M_MAX_RETRIES` | `3` | Max retry attempts |

```yaml
queue:
  batchSize: 10
  pollInterval: 100ms
  visibilityTimeout: 30s
  retryDelay: 1s
  maxRetries: 3
```

## Security Configuration

### Encryption

| Option | Env Var | Default | Description |
|--------|---------|---------|-------------|
| `security.encryptionKey` | `M9M_ENCRYPTION_KEY` | - | AES encryption key (32 chars) |

```yaml
security:
  encryptionKey: "your-32-character-encryption-key"
```

### Authentication

| Option | Env Var | Default | Description |
|--------|---------|---------|-------------|
| `auth.enabled` | `M9M_AUTH_ENABLED` | `false` | Enable authentication |
| `auth.apiKeys.enabled` | `M9M_API_KEYS_ENABLED` | `true` | Enable API key auth |

```yaml
auth:
  enabled: true
  apiKeys:
    enabled: true
  jwt:
    enabled: true
    secret: "your-jwt-secret"
    accessTokenExpiry: 1h
    refreshTokenExpiry: 7d
```

### CORS

| Option | Env Var | Default | Description |
|--------|---------|---------|-------------|
| `security.cors.enabled` | `M9M_CORS_ENABLED` | `false` | Enable CORS |
| `security.cors.origins` | `M9M_CORS_ORIGINS` | `*` | Allowed origins |

```yaml
security:
  cors:
    enabled: true
    origins:
      - "https://app.example.com"
    methods: ["GET", "POST", "PUT", "DELETE"]
    headers: ["Authorization", "Content-Type"]
    credentials: true
```

### Rate Limiting

| Option | Env Var | Default | Description |
|--------|---------|---------|-------------|
| `security.rateLimit.enabled` | `M9M_RATE_LIMIT_ENABLED` | `false` | Enable rate limiting |
| `security.rateLimit.requests` | `M9M_RATE_LIMIT_REQUESTS` | `100` | Requests per period |
| `security.rateLimit.period` | `M9M_RATE_LIMIT_PERIOD` | `1m` | Rate limit period |

```yaml
security:
  rateLimit:
    enabled: true
    requests: 100
    period: 1m
    byApiKey: true
```

## Monitoring Configuration

### Metrics

| Option | Env Var | Default | Description |
|--------|---------|---------|-------------|
| `monitoring.metricsEnabled` | `M9M_METRICS_ENABLED` | `true` | Enable Prometheus metrics |
| `monitoring.metricsPort` | `M9M_METRICS_PORT` | `9090` | Metrics server port |
| `monitoring.metricsPath` | `M9M_METRICS_PATH` | `/metrics` | Metrics endpoint |

```yaml
monitoring:
  metricsEnabled: true
  metricsPort: 9090
  metricsPath: /metrics
```

### Tracing

| Option | Env Var | Default | Description |
|--------|---------|---------|-------------|
| `monitoring.tracingEnabled` | `M9M_TRACING_ENABLED` | `false` | Enable tracing |
| `monitoring.tracingEndpoint` | `M9M_TRACING_ENDPOINT` | - | Jaeger/OTLP endpoint |
| `monitoring.serviceName` | `M9M_SERVICE_NAME` | `m9m` | Service name for tracing |

```yaml
monitoring:
  tracingEnabled: true
  tracingEndpoint: "http://jaeger:14268/api/traces"
  serviceName: "m9m"
  sampleRate: 0.1
```

## Logging Configuration

| Option | Env Var | Default | Description |
|--------|---------|---------|-------------|
| `logging.level` | `M9M_LOG_LEVEL` | `info` | Log level |
| `logging.format` | `M9M_LOG_FORMAT` | `text` | Log format |
| `logging.output` | `M9M_LOG_OUTPUT` | `stdout` | Log output |

Log levels: `debug`, `info`, `warn`, `error`
Formats: `text`, `json`

```yaml
logging:
  level: info
  format: json
  output: stdout
  fields:
    environment: production
    version: "1.0.0"
```

## Execution Configuration

| Option | Env Var | Default | Description |
|--------|---------|---------|-------------|
| `execution.timeout` | `M9M_EXECUTION_TIMEOUT` | `300s` | Default workflow timeout |
| `execution.saveSuccessData` | `M9M_SAVE_SUCCESS_DATA` | `all` | Save successful executions |
| `execution.saveErrorData` | `M9M_SAVE_ERROR_DATA` | `all` | Save failed executions |
| `execution.pruneAfter` | `M9M_PRUNE_AFTER` | `30d` | Execution retention |

```yaml
execution:
  timeout: 300s
  saveSuccessData: all  # all, none, summary
  saveErrorData: all
  pruneAfter: 30d
```

## Webhooks Configuration

| Option | Env Var | Default | Description |
|--------|---------|---------|-------------|
| `webhooks.timeout` | `M9M_WEBHOOK_TIMEOUT` | `30s` | Webhook timeout |
| `webhooks.maxBodySize` | `M9M_WEBHOOK_MAX_BODY` | `10MB` | Max request body |
| `webhooks.path` | `M9M_WEBHOOK_PATH` | `/webhook` | Webhook base path |

```yaml
webhooks:
  timeout: 30s
  maxBodySize: 10485760  # 10MB
  path: /webhook
  allowedIps:
    - "0.0.0.0/0"
```

## External Integrations

### SMTP

```yaml
smtp:
  host: "smtp.example.com"
  port: 587
  user: "notifications@example.com"
  password: "${SMTP_PASSWORD}"
  from: "m9m <notifications@example.com>"
  tls: true
```

### S3 Storage

```yaml
storage:
  type: s3
  bucket: "m9m-storage"
  region: "us-east-1"
  accessKeyId: "${AWS_ACCESS_KEY_ID}"
  secretAccessKey: "${AWS_SECRET_ACCESS_KEY}"
```

## Complete Example

```yaml
# config.yaml
server:
  port: 8080
  host: "0.0.0.0"
  baseUrl: "https://m9m.example.com"
  readTimeout: 30s
  writeTimeout: 30s

database:
  type: postgres
  url: "${DATABASE_URL}"
  maxConnections: 50
  maxIdleConnections: 10

queue:
  type: redis
  url: "${REDIS_URL}"
  maxWorkers: 20
  batchSize: 10

security:
  encryptionKey: "${ENCRYPTION_KEY}"
  cors:
    enabled: true
    origins:
      - "https://app.example.com"
  rateLimit:
    enabled: true
    requests: 1000
    period: 1m

auth:
  enabled: true
  jwt:
    secret: "${JWT_SECRET}"
    accessTokenExpiry: 1h

monitoring:
  metricsEnabled: true
  metricsPort: 9090
  tracingEnabled: true
  tracingEndpoint: "http://jaeger:14268/api/traces"

logging:
  level: info
  format: json

execution:
  timeout: 300s
  saveSuccessData: summary
  pruneAfter: 30d

webhooks:
  timeout: 30s
  maxBodySize: 10485760
```

## Environment Variable Reference

All config options can be set via environment variables using the `M9M_` prefix and snake_case:

```bash
# Server
export M9M_PORT=8080
export M9M_HOST=0.0.0.0

# Database
export M9M_DB_TYPE=postgres
export M9M_DB_URL="postgres://..."

# Queue
export M9M_QUEUE_TYPE=redis
export M9M_QUEUE_URL="redis://..."
export M9M_MAX_WORKERS=20

# Security
export M9M_ENCRYPTION_KEY="..."
export M9M_AUTH_ENABLED=true

# Monitoring
export M9M_METRICS_ENABLED=true
export M9M_LOG_LEVEL=info
```

## Next Steps

- [CLI Reference](cli.md) - Command-line options
- [Troubleshooting](troubleshooting.md) - Common issues
- [Deployment](../deployment/overview.md) - Deployment guides
