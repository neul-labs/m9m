# API Authentication

m9m supports multiple authentication methods for API access.

## Authentication Methods

| Method | Use Case |
|--------|----------|
| JWT Token | User sessions, web UI |
| API Key | Programmatic access, integrations |
| Basic Auth | Simple authentication (if enabled) |

## JWT Authentication

JWT (JSON Web Token) authentication is the primary method for user sessions.

### Obtaining a Token

```bash
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "your-password"
}
```

Response:

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expiresAt": "2024-01-27T10:00:00Z",
  "user": {
    "id": "user-123",
    "username": "admin"
  }
}
```

### Using the Token

Include the token in the `Authorization` header:

```bash
curl http://localhost:8080/api/v1/workflows \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### Token Refresh

```bash
POST /api/v1/auth/refresh
Authorization: Bearer <current-token>
```

Response:

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expiresAt": "2024-01-28T10:00:00Z"
}
```

### Configuration

Configure JWT in `config.yaml`:

```yaml
security:
  jwt:
    enabled: true
    secret: "your-secret-key-min-32-characters"
    expiration: "24h"
```

## API Key Authentication

API keys are ideal for programmatic access and integrations.

### Creating an API Key

```bash
POST /api/v1/api-keys
Authorization: Bearer <jwt-token>
Content-Type: application/json

{
  "name": "CI/CD Integration",
  "expiresAt": "2025-01-01T00:00:00Z"
}
```

Response:

```json
{
  "id": "key-123",
  "name": "CI/CD Integration",
  "key": "m9m_sk_abc123def456...",
  "createdAt": "2024-01-26T10:00:00Z",
  "expiresAt": "2025-01-01T00:00:00Z"
}
```

!!! warning "Save the Key"
    The API key is only shown once. Store it securely.

### Using API Keys

Include the key in the `X-API-Key` header:

```bash
curl http://localhost:8080/api/v1/workflows \
  -H "X-API-Key: m9m_sk_abc123def456..."
```

### Managing API Keys

**List keys:**

```bash
GET /api/v1/api-keys
```

**Revoke a key:**

```bash
DELETE /api/v1/api-keys/key-123
```

### Configuration

Enable API keys in `config.yaml`:

```yaml
security:
  api_keys:
    enabled: true
```

## Basic Authentication

Basic auth provides simple username/password authentication.

### Usage

```bash
curl http://localhost:8080/api/v1/workflows \
  -u "admin:password"

# Or with explicit header
curl http://localhost:8080/api/v1/workflows \
  -H "Authorization: Basic YWRtaW46cGFzc3dvcmQ="
```

### Configuration

Enable basic auth in `config.yaml`:

```yaml
security:
  basic_auth:
    enabled: true
    username: "admin"
    password: "your-secure-password"
```

!!! note "Production Usage"
    Basic auth is recommended only for development or when behind a reverse proxy with TLS.

## Authentication Errors

### 401 Unauthorized

```json
{
  "error": "Unauthorized",
  "code": "AUTH_REQUIRED",
  "message": "Authentication required"
}
```

**Causes:**

- Missing authentication header
- Invalid or expired token
- Invalid API key

### 403 Forbidden

```json
{
  "error": "Forbidden",
  "code": "ACCESS_DENIED",
  "message": "Insufficient permissions"
}
```

**Causes:**

- Valid auth but insufficient permissions
- Resource access not allowed

## Public Endpoints

Some endpoints don't require authentication:

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Health check |
| `GET /ready` | Readiness check |
| `POST /api/v1/auth/login` | Login |
| `POST /webhook/*` | Webhook endpoints (use webhook auth) |

## Security Best Practices

### Token Storage

- **Browser**: Store in httpOnly cookies or secure storage
- **Server**: Store in environment variables or secret managers
- **Never**: Store in code, logs, or version control

### Token Rotation

- Use short-lived JWT tokens (hours, not days)
- Implement token refresh for long sessions
- Rotate API keys periodically

### HTTPS

Always use HTTPS in production:

```yaml
server:
  tls:
    enabled: true
    cert_file: "/path/to/cert.pem"
    key_file: "/path/to/key.pem"
```

### Rate Limiting

Configure rate limiting:

```yaml
performance:
  rate_limit:
    enabled: true
    requests_per_minute: 60
    burst: 10
```

## CORS Configuration

Configure CORS for web applications:

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
    allowed_headers:
      - "Content-Type"
      - "Authorization"
    allow_credentials: true
```

## Example: Full Authentication Flow

```bash
# 1. Login to get JWT token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "password"}' \
  | jq -r '.token')

# 2. Create an API key for automation
API_KEY=$(curl -s -X POST http://localhost:8080/api/v1/api-keys \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "automation"}' \
  | jq -r '.key')

# 3. Use API key for subsequent requests
curl http://localhost:8080/api/v1/workflows \
  -H "X-API-Key: $API_KEY"
```

## See Also

- [Security Configuration](../configuration/security.md) - Full security settings
- [Deployment](../deployment/production.md) - Production security
