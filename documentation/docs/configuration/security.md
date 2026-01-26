# Security Configuration

Configure authentication, authorization, and security settings.

## JWT Authentication

```yaml
security:
  jwt:
    enabled: true
    secret: "your-secret-key-minimum-32-characters"
    expiration: "24h"
```

| Setting | Default | Description |
|---------|---------|-------------|
| `enabled` | `true` | Enable JWT auth |
| `secret` | - | Signing secret (min 32 chars) |
| `expiration` | `24h` | Token lifetime |

!!! warning "Secret Key"
    Use a strong, random secret. Generate one with:
    ```bash
    openssl rand -base64 32
    ```

## API Keys

```yaml
security:
  api_keys:
    enabled: true
```

API keys provide programmatic access without JWT tokens.

## Basic Authentication

```yaml
security:
  basic_auth:
    enabled: false
    username: "admin"
    password: "changeme"
```

!!! note "Use Sparingly"
    Basic auth is best for development or when behind a secure reverse proxy.

## CORS Configuration

```yaml
security:
  cors:
    enabled: true
    allowed_origins:
      - "https://app.example.com"
      - "http://localhost:3000"
    allowed_methods:
      - "GET"
      - "POST"
      - "PUT"
      - "DELETE"
      - "PATCH"
    allowed_headers:
      - "Content-Type"
      - "Authorization"
      - "X-API-Key"
    allow_credentials: true
    max_age: 86400
```

| Setting | Description |
|---------|-------------|
| `allowed_origins` | Origins allowed to access API |
| `allowed_methods` | HTTP methods allowed |
| `allowed_headers` | Headers allowed in requests |
| `allow_credentials` | Allow cookies/auth headers |
| `max_age` | Preflight cache duration (seconds) |

### Development Mode

In dev mode, CORS is permissive:

```bash
m9m serve --dev
```

Equivalent to:

```yaml
security:
  cors:
    allowed_origins: ["*"]
```

## Credential Encryption

```yaml
credentials:
  encryption_key: "your-encryption-key-minimum-32-characters"
  storage: "database"
```

| Setting | Description |
|---------|-------------|
| `encryption_key` | AES-256 encryption key |
| `storage` | Where to store credentials |

!!! warning "Encryption Key"
    - Keep this key secret and backed up
    - Changing the key makes existing credentials unreadable
    - Generate with: `openssl rand -base64 32`

## Rate Limiting

```yaml
performance:
  rate_limit:
    enabled: true
    requests_per_minute: 60
    burst: 10
```

| Setting | Default | Description |
|---------|---------|-------------|
| `enabled` | `false` | Enable rate limiting |
| `requests_per_minute` | `60` | Requests per minute per IP |
| `burst` | `10` | Burst allowance |

## TLS/HTTPS

```yaml
server:
  tls:
    enabled: true
    cert_file: "/path/to/cert.pem"
    key_file: "/path/to/key.pem"
```

See [Server Configuration](server.md) for details.

## Environment Variables

```bash
# JWT
export M9M_JWT_SECRET=your-secret-key
export M9M_JWT_EXPIRATION=24h

# Credentials
export M9M_ENCRYPTION_KEY=your-encryption-key

# Also supports n8n-style for compatibility
export N8N_ENCRYPTION_KEY=your-encryption-key
```

## Security Headers

m9m sets security headers automatically:

| Header | Value |
|--------|-------|
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `X-XSS-Protection` | `1; mode=block` |

## Best Practices

### Production Configuration

```yaml
security:
  jwt:
    enabled: true
    secret: "${JWT_SECRET}"  # From environment
    expiration: "8h"

  api_keys:
    enabled: true

  cors:
    enabled: true
    allowed_origins:
      - "https://your-app.com"
    allow_credentials: true

credentials:
  encryption_key: "${ENCRYPTION_KEY}"  # From environment

server:
  tls:
    enabled: true
    cert_file: "/etc/ssl/certs/m9m.pem"
    key_file: "/etc/ssl/private/m9m.key"

performance:
  rate_limit:
    enabled: true
    requests_per_minute: 100
```

### Secret Management

Store secrets in environment variables:

```bash
export JWT_SECRET=$(cat /run/secrets/jwt_secret)
export ENCRYPTION_KEY=$(cat /run/secrets/encryption_key)
```

Or use a secret manager (Vault, AWS Secrets Manager, etc.).

### Network Security

- Run behind a reverse proxy (nginx, traefik)
- Use TLS for all connections
- Restrict network access to trusted IPs
- Use firewall rules

### Regular Audits

- Rotate JWT secrets periodically
- Revoke unused API keys
- Review access logs
- Update to latest versions

## Troubleshooting

### Invalid Token

```
Error: invalid or expired token
```

- Check JWT secret matches
- Check token hasn't expired
- Regenerate token

### CORS Errors

```
Access-Control-Allow-Origin missing
```

- Add origin to `allowed_origins`
- Check for trailing slashes in URLs
- Verify `allow_credentials` if using auth

### Encryption Errors

```
Error: cipher: message authentication failed
```

- Encryption key doesn't match
- Credential data corrupted
- Key must be exactly 32 characters
