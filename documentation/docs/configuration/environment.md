# Environment Variables

All configuration options can be set via environment variables.

## Naming Convention

Environment variables use the `M9M_` prefix with underscores:

```
config.yaml path → Environment variable
server.port     → M9M_SERVER_PORT
database.type   → M9M_DATABASE_TYPE
```

## Core Variables

### Server

| Variable | Default | Description |
|----------|---------|-------------|
| `M9M_HOST` | `0.0.0.0` | Server bind address |
| `M9M_PORT` | `8080` | Server port |
| `M9M_DEV_MODE` | `false` | Enable development mode |

### Database

| Variable | Default | Description |
|----------|---------|-------------|
| `M9M_DB_TYPE` | `sqlite` | Database type |
| `M9M_DB_PATH` | `~/.m9m/data/m9m.db` | SQLite path |
| `M9M_POSTGRES_URL` | - | PostgreSQL connection URL |
| `M9M_POSTGRES_HOST` | `localhost` | PostgreSQL host |
| `M9M_POSTGRES_PORT` | `5432` | PostgreSQL port |
| `M9M_POSTGRES_DATABASE` | `m9m` | PostgreSQL database |
| `M9M_POSTGRES_USER` | `m9m` | PostgreSQL user |
| `M9M_POSTGRES_PASSWORD` | - | PostgreSQL password |

### Queue

| Variable | Default | Description |
|----------|---------|-------------|
| `M9M_QUEUE_TYPE` | `sqlite` | Queue type |
| `M9M_QUEUE_PATH` | `~/.m9m/data/queue.db` | Queue SQLite path |
| `M9M_MAX_WORKERS` | `4` | Worker threads |

### Security

| Variable | Default | Description |
|----------|---------|-------------|
| `M9M_JWT_SECRET` | - | JWT signing secret |
| `M9M_JWT_EXPIRATION` | `24h` | JWT expiration |
| `M9M_ENCRYPTION_KEY` | - | Credential encryption key |

### Logging

| Variable | Default | Description |
|----------|---------|-------------|
| `M9M_LOG_LEVEL` | `info` | Log level |
| `M9M_LOG_FORMAT` | `json` | Log format (json, text) |

### Monitoring

| Variable | Default | Description |
|----------|---------|-------------|
| `M9M_METRICS_PORT` | `0` | Metrics port (0=disabled) |

## n8n Compatibility Variables

For compatibility with existing n8n configurations:

| Variable | Maps To |
|----------|---------|
| `N8N_ENCRYPTION_KEY` | `M9M_ENCRYPTION_KEY` |
| `N8N_PORT` | `M9M_PORT` |
| `N8N_HOST` | `M9M_HOST` |

## Usage Examples

### Basic Setup

```bash
export M9M_PORT=3000
export M9M_LOG_LEVEL=debug

m9m serve
```

### Production Setup

```bash
export M9M_PORT=8080
export M9M_DB_TYPE=postgres
export M9M_POSTGRES_URL="postgres://user:pass@db:5432/m9m?sslmode=require"
export M9M_QUEUE_TYPE=sqlite
export M9M_MAX_WORKERS=8
export M9M_JWT_SECRET="$(openssl rand -base64 32)"
export M9M_ENCRYPTION_KEY="$(openssl rand -base64 32)"
export M9M_LOG_LEVEL=info

m9m serve
```

### Docker

```dockerfile
# Dockerfile
ENV M9M_PORT=8080
ENV M9M_DB_TYPE=sqlite
ENV M9M_DB_PATH=/data/m9m.db
```

```yaml
# docker-compose.yml
services:
  m9m:
    image: ghcr.io/neul-labs/m9m:latest
    environment:
      M9M_PORT: 8080
      M9M_DB_TYPE: postgres
      M9M_POSTGRES_URL: postgres://user:pass@db:5432/m9m
      M9M_JWT_SECRET: ${JWT_SECRET}
      M9M_ENCRYPTION_KEY: ${ENCRYPTION_KEY}
```

### Kubernetes

```yaml
# ConfigMap for non-sensitive values
apiVersion: v1
kind: ConfigMap
metadata:
  name: m9m-config
data:
  M9M_PORT: "8080"
  M9M_DB_TYPE: "postgres"
  M9M_LOG_LEVEL: "info"
  M9M_MAX_WORKERS: "8"
```

```yaml
# Secret for sensitive values
apiVersion: v1
kind: Secret
metadata:
  name: m9m-secrets
type: Opaque
stringData:
  M9M_JWT_SECRET: "your-jwt-secret"
  M9M_ENCRYPTION_KEY: "your-encryption-key"
  M9M_POSTGRES_URL: "postgres://..."
```

```yaml
# Deployment
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
        - name: m9m
          envFrom:
            - configMapRef:
                name: m9m-config
            - secretRef:
                name: m9m-secrets
```

## .env File

Create a `.env` file in your project:

```bash
# .env
M9M_PORT=8080
M9M_DB_TYPE=sqlite
M9M_LOG_LEVEL=debug
M9M_JWT_SECRET=development-secret-key-32-chars-min
M9M_ENCRYPTION_KEY=development-encryption-key-32chars
```

Load with:

```bash
source .env
m9m serve

# Or with direnv
# .envrc
dotenv
```

## Priority Order

Configuration is applied in this order (later overrides earlier):

1. Default values
2. Configuration file (`config.yaml`)
3. Environment variables
4. Command-line flags

Example:

```bash
# config.yaml has port: 8080
# Environment has M9M_PORT=3000
# CLI has --port 4000

# Result: port 4000 is used
```

## Viewing Configuration

See effective configuration:

```bash
m9m config show
```

Check a specific variable:

```bash
m9m config get database.type
```

## Troubleshooting

### Variable Not Applied

- Check spelling and prefix (`M9M_`)
- Verify export: `echo $M9M_PORT`
- Check override order (CLI > env > file)

### Secret in Logs

- Use `M9M_LOG_LEVEL=info` (not debug)
- Sensitive values are redacted in logs
