# Server Configuration

Configure the m9m HTTP server.

## Basic Settings

```yaml
server:
  host: "0.0.0.0"
  port: 8080
```

| Setting | Default | Description |
|---------|---------|-------------|
| `host` | `0.0.0.0` | Bind address |
| `port` | `8080` | HTTP port |

## TLS/HTTPS

Enable HTTPS with TLS certificates:

```yaml
server:
  host: "0.0.0.0"
  port: 443
  tls:
    enabled: true
    cert_file: "/path/to/cert.pem"
    key_file: "/path/to/key.pem"
```

| Setting | Description |
|---------|-------------|
| `tls.enabled` | Enable HTTPS |
| `tls.cert_file` | Path to certificate file |
| `tls.key_file` | Path to private key file |

### Generate Self-Signed Certificate

For development:

```bash
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout key.pem -out cert.pem \
  -subj "/CN=localhost"
```

## Timeouts

```yaml
server:
  read_timeout: "30s"
  write_timeout: "60s"
  idle_timeout: "120s"
```

| Setting | Default | Description |
|---------|---------|-------------|
| `read_timeout` | `30s` | Max time to read request |
| `write_timeout` | `60s` | Max time to write response |
| `idle_timeout` | `120s` | Keep-alive timeout |

## Command-Line Flags

Override via CLI:

```bash
m9m serve --host 127.0.0.1 --port 3000
```

| Flag | Config Equivalent |
|------|-------------------|
| `--host` | `server.host` |
| `--port` | `server.port` |

## Environment Variables

```bash
export M9M_HOST=0.0.0.0
export M9M_PORT=8080
```

## Development Mode

Enable development mode for permissive CORS:

```bash
m9m serve --dev
```

Or in configuration:

```yaml
development:
  debug: true
  hot_reload: true
```

## Bind Address Examples

| Address | Description |
|---------|-------------|
| `0.0.0.0` | All interfaces (default) |
| `127.0.0.1` | Localhost only |
| `192.168.1.100` | Specific interface |
| `::` | All IPv6 interfaces |

## Behind a Reverse Proxy

When running behind nginx/traefik:

```yaml
server:
  host: "127.0.0.1"
  port: 8080
  # Trust X-Forwarded-* headers
  trusted_proxies:
    - "127.0.0.1"
    - "10.0.0.0/8"
```

### Nginx Configuration

```nginx
server {
    listen 443 ssl;
    server_name m9m.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket support
    location /push {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

## Health Endpoints

Health check endpoints are always available:

| Endpoint | Description |
|----------|-------------|
| `/health` | Basic health check |
| `/ready` | Readiness check |
| `/healthz` | Kubernetes-style |

## Metrics

Enable Prometheus metrics:

```yaml
monitoring:
  metrics:
    enabled: true
    port: 9090
    path: "/metrics"
```

Or via CLI:

```bash
m9m serve --metrics-port 9090
```

Access at: `http://localhost:9090/metrics`
