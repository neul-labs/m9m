# serve Command

Start the m9m server with REST API and optional web UI.

## Synopsis

```bash
m9m serve [flags]
```

## Description

The `serve` command starts m9m as a full server, providing:

- REST API for workflow management
- Web UI for visual workflow editing
- Webhook endpoints for triggers
- Scheduled workflow execution
- Job queue for async processing

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `8080` | Server port |
| `--host` | `0.0.0.0` | Server bind address |
| `--db` | `~/.m9m/data/m9m.db` | SQLite database path |
| `--postgres` | - | PostgreSQL connection URL |
| `--queue` | `sqlite` | Queue type (memory, sqlite) |
| `--queue-db` | `~/.m9m/data/queue.db` | Queue SQLite database path |
| `--workers` | `4` | Number of worker threads |
| `--metrics-port` | `0` | Metrics port (0 = disabled) |
| `--dev` | `false` | Enable development mode |

## Examples

### Basic Usage

```bash
# Start with defaults
m9m serve

# Custom port
m9m serve --port 3000

# Bind to localhost only
m9m serve --host 127.0.0.1 --port 8080
```

### Database Configuration

```bash
# Custom SQLite path
m9m serve --db /data/m9m.db

# Use PostgreSQL
m9m serve --postgres "postgres://user:pass@localhost:5432/m9m"
```

### Queue Configuration

```bash
# SQLite queue (persistent, default)
m9m serve --queue sqlite --queue-db /data/queue.db

# Memory queue (faster, no persistence)
m9m serve --queue memory

# More workers for high throughput
m9m serve --workers 8
```

### Development Mode

```bash
# Enable dev mode (permissive CORS, hot reload)
m9m serve --dev
```

### Production Setup

```bash
# Full production configuration
m9m serve \
  --host 0.0.0.0 \
  --port 8080 \
  --postgres "postgres://m9m:secret@db:5432/m9m?sslmode=require" \
  --queue sqlite \
  --queue-db /data/queue.db \
  --workers 8 \
  --metrics-port 9090
```

## Endpoints

Once running, the server provides:

| Endpoint | Description |
|----------|-------------|
| `http://host:port/` | Web UI |
| `http://host:port/api/v1` | REST API |
| `http://host:port/health` | Health check |
| `http://host:port/webhook/*` | Webhook triggers |
| `http://host:port/metrics` | Prometheus metrics |

## Output

```
[m9m] Starting m9m server v1.0.0
[m9m] Using SQLite storage: /home/user/.m9m/data/m9m.db
[m9m] Using SQLite job queue: /home/user/.m9m/data/queue.db
[m9m] Started 4 workers for job processing
[m9m] Development mode enabled (permissive CORS)
[m9m] Web UI enabled (embedded: true)
[m9m] Server listening on http://0.0.0.0:8080
[m9m] Web UI: http://0.0.0.0:8080
[m9m] API: http://0.0.0.0:8080/api/v1
[m9m] Health: http://0.0.0.0:8080/health
```

## Graceful Shutdown

The server handles SIGINT and SIGTERM for graceful shutdown:

```bash
# Stop with Ctrl+C
^C
[m9m] Shutting down server...
[m9m] Worker pool stopped
[m9m] Server stopped
```

## Health Checks

```bash
# Basic health check
curl http://localhost:8080/health
# {"status":"ok"}

# Readiness check
curl http://localhost:8080/ready
# {"status":"ready"}
```

## Metrics

When `--metrics-port` is set:

```bash
m9m serve --metrics-port 9090

# Access Prometheus metrics
curl http://localhost:9090/metrics
```

## Docker Usage

```bash
docker run -p 8080:8080 \
  -v m9m-data:/app/data \
  ghcr.io/neul-labs/m9m:latest \
  serve --port 8080
```

## See Also

- [Configuration](../configuration/index.md) - Full configuration options
- [API Reference](../api/index.md) - REST API documentation
- [Deployment](../deployment/index.md) - Production deployment guide
