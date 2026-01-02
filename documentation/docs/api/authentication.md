# API Authentication

Configure authentication for the m9m REST API.

## Authentication Methods

m9m supports multiple authentication methods:

1. **API Key** - Simple key-based authentication
2. **Bearer Token** - JWT token authentication
3. **Basic Auth** - Username/password (development only)
4. **OAuth2** - Third-party OAuth providers

## API Key Authentication

### Generate API Key

```bash
m9m apikey create --name "My Application"
```

Output:
```
API Key created successfully
Name: My Application
Key: m9m_sk_abc123def456...
Created: 2024-01-15T10:00:00Z

Save this key securely - it cannot be retrieved again.
```

### Use API Key

Include in request header:

```bash
curl -H "X-API-Key: m9m_sk_abc123def456..." \
  http://localhost:8080/api/v1/workflows
```

### Manage API Keys

```bash
# List API keys
m9m apikey list

# Revoke API key
m9m apikey revoke m9m_sk_abc123def456

# Rotate API key
m9m apikey rotate m9m_sk_abc123def456
```

### API Key Scopes

Create keys with limited permissions:

```bash
m9m apikey create --name "Read Only" --scopes "read:workflows,read:executions"
```

Available scopes:
| Scope | Description |
|-------|-------------|
| `read:workflows` | Read workflow data |
| `write:workflows` | Create/update workflows |
| `execute:workflows` | Execute workflows |
| `read:executions` | Read execution data |
| `read:credentials` | Read credential metadata |
| `write:credentials` | Create/update credentials |
| `admin` | Full access |

## Bearer Token Authentication

### Generate Token

```http
POST /api/v1/auth/token
Content-Type: application/json

{
  "username": "admin",
  "password": "password"
}
```

Response:
```json
{
  "success": true,
  "data": {
    "accessToken": "eyJhbGciOiJIUzI1NiIs...",
    "refreshToken": "eyJhbGciOiJIUzI1NiIs...",
    "expiresIn": 3600
  }
}
```

### Use Bearer Token

```bash
curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  http://localhost:8080/api/v1/workflows
```

### Refresh Token

```http
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refreshToken": "eyJhbGciOiJIUzI1NiIs..."
}
```

### Token Configuration

```yaml
# config.yaml
auth:
  jwt:
    secret: "your-secret-key"
    accessTokenExpiry: 1h
    refreshTokenExpiry: 7d
    issuer: "m9m"
```

## Basic Authentication

!!! warning "Development Only"
    Basic authentication should only be used in development environments.

```bash
curl -u admin:password \
  http://localhost:8080/api/v1/workflows
```

Enable in configuration:

```yaml
auth:
  basicAuth:
    enabled: true
    users:
      - username: admin
        password: $2a$10$... # bcrypt hash
```

## OAuth2 Integration

### Configure OAuth Provider

```yaml
auth:
  oauth2:
    enabled: true
    providers:
      github:
        clientId: "your-client-id"
        clientSecret: "your-client-secret"
        scopes: ["user:email"]
      google:
        clientId: "your-client-id"
        clientSecret: "your-client-secret"
        scopes: ["email", "profile"]
```

### OAuth Flow

1. Redirect user to:
   ```
   GET /api/v1/auth/oauth2/github
   ```

2. User authenticates with provider

3. Callback returns tokens:
   ```
   GET /api/v1/auth/oauth2/callback?code=...
   ```

## Service Accounts

For machine-to-machine authentication:

### Create Service Account

```bash
m9m service-account create --name "CI/CD Pipeline"
```

Output:
```
Service Account: ci-cd-pipeline
Client ID: sa_abc123
Client Secret: sa_secret_xyz789
```

### Authenticate

```http
POST /api/v1/auth/token
Content-Type: application/json

{
  "grantType": "client_credentials",
  "clientId": "sa_abc123",
  "clientSecret": "sa_secret_xyz789"
}
```

## Security Configuration

### CORS

```yaml
security:
  cors:
    enabled: true
    origins:
      - "https://app.example.com"
    methods: ["GET", "POST", "PUT", "DELETE"]
    headers: ["Authorization", "Content-Type"]
```

### Rate Limiting

```yaml
security:
  rateLimit:
    enabled: true
    requests: 100
    period: 1m
    byApiKey: true
```

### IP Whitelist

```yaml
security:
  ipWhitelist:
    enabled: true
    allowed:
      - "10.0.0.0/8"
      - "192.168.1.0/24"
```

## Authentication Errors

| Code | Message | Solution |
|------|---------|----------|
| 401 | `Missing authentication` | Include auth header |
| 401 | `Invalid API key` | Check key is correct |
| 401 | `Token expired` | Refresh token |
| 403 | `Insufficient permissions` | Check scopes |

## SDK Authentication

### Go

```go
import "github.com/m9m/m9m-go-sdk"

// API Key
client := m9m.NewClient(
    m9m.WithBaseURL("http://localhost:8080"),
    m9m.WithAPIKey("m9m_sk_abc123"),
)

// Bearer Token
client := m9m.NewClient(
    m9m.WithBaseURL("http://localhost:8080"),
    m9m.WithBearerToken("eyJhbGciOiJIUzI1NiIs..."),
)
```

### Python

```python
import m9m

# API Key
client = m9m.Client(
    base_url="http://localhost:8080",
    api_key="m9m_sk_abc123"
)

# Bearer Token
client = m9m.Client(
    base_url="http://localhost:8080",
    token="eyJhbGciOiJIUzI1NiIs..."
)
```

### JavaScript

```javascript
const { M9MClient } = require('@m9m/sdk');

// API Key
const client = new M9MClient({
  baseUrl: 'http://localhost:8080',
  apiKey: 'm9m_sk_abc123'
});

// Bearer Token
const client = new M9MClient({
  baseUrl: 'http://localhost:8080',
  token: 'eyJhbGciOiJIUzI1NiIs...'
});
```

## Best Practices

1. **Use API keys** for server-to-server communication
2. **Use OAuth2** for user authentication
3. **Rotate keys regularly** (every 90 days)
4. **Use scoped permissions** - least privilege
5. **Store secrets securely** - never in code
6. **Enable HTTPS** in production
7. **Monitor authentication failures**

## Next Steps

- [REST API](rest-api.md) - API usage
- [Webhooks](webhooks.md) - Webhook security
- [Production](../deployment/production.md) - Production security
