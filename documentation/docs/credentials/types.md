# Credential Types

All supported credential types and their configuration.

## API Key

Simple API key authentication.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `apiKey` | string | Yes | The API key |

### Example

```json
{
  "type": "apiKey",
  "data": {
    "apiKey": "sk-xxxxxxxxxxxxxxxx"
  }
}
```

### Usage

```json
{
  "credentials": {
    "apiKey": {
      "id": "cred-123"
    }
  }
}
```

## HTTP Header Auth

Custom header-based authentication.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Header name |
| `value` | string | Yes | Header value |

### Example

```json
{
  "type": "httpHeaderAuth",
  "data": {
    "name": "X-API-Key",
    "value": "your-api-key"
  }
}
```

## Basic Auth

HTTP Basic authentication.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `username` | string | Yes | Username |
| `password` | string | Yes | Password |

### Example

```json
{
  "type": "httpBasicAuth",
  "data": {
    "username": "user",
    "password": "pass123"
  }
}
```

## OAuth2

OAuth 2.0 authentication.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `clientId` | string | Yes | Client ID |
| `clientSecret` | string | Yes | Client secret |
| `accessToken` | string | No | Current access token |
| `refreshToken` | string | No | Refresh token |
| `tokenUrl` | string | Yes | Token endpoint |
| `authUrl` | string | No | Authorization URL |
| `scope` | string | No | Requested scopes |

### Example

```json
{
  "type": "oauth2",
  "data": {
    "clientId": "your-client-id",
    "clientSecret": "your-client-secret",
    "accessToken": "ya29.xxx...",
    "refreshToken": "1//xxx...",
    "tokenUrl": "https://oauth2.googleapis.com/token"
  }
}
```

## Bearer Token

Bearer token authentication.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `token` | string | Yes | Bearer token |

### Example

```json
{
  "type": "bearerToken",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs..."
  }
}
```

## PostgreSQL

PostgreSQL database credentials.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `host` | string | Yes | Database host |
| `port` | integer | No | Port (default: 5432) |
| `database` | string | Yes | Database name |
| `user` | string | Yes | Username |
| `password` | string | Yes | Password |
| `ssl` | boolean | No | Use SSL (default: false) |

### Example

```json
{
  "type": "postgres",
  "data": {
    "host": "localhost",
    "port": 5432,
    "database": "myapp",
    "user": "postgres",
    "password": "secret",
    "ssl": true
  }
}
```

## MySQL

MySQL database credentials.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `host` | string | Yes | Database host |
| `port` | integer | No | Port (default: 3306) |
| `database` | string | Yes | Database name |
| `user` | string | Yes | Username |
| `password` | string | Yes | Password |

### Example

```json
{
  "type": "mysql",
  "data": {
    "host": "localhost",
    "port": 3306,
    "database": "myapp",
    "user": "root",
    "password": "secret"
  }
}
```

## AWS

AWS credentials for S3, Lambda, etc.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `accessKeyId` | string | Yes | AWS access key ID |
| `secretAccessKey` | string | Yes | AWS secret access key |
| `region` | string | Yes | AWS region |

### Example

```json
{
  "type": "aws",
  "data": {
    "accessKeyId": "AKIAXXXXXXXXXXXXXXXX",
    "secretAccessKey": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "region": "us-east-1"
  }
}
```

## Slack

Slack API credentials.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `accessToken` | string | Yes | Bot or user token |

### Example

```json
{
  "type": "slack",
  "data": {
    "accessToken": "xoxb-xxxxxxxxxxxx-xxxxxxxxxxxx-xxxxxxxxxxxxxxxxxxxxxxxx"
  }
}
```

## GitHub

GitHub API credentials.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `accessToken` | string | Yes | Personal access token |

### Example

```json
{
  "type": "github",
  "data": {
    "accessToken": "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
  }
}
```

## OpenAI

OpenAI API credentials.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `apiKey` | string | Yes | OpenAI API key |
| `organization` | string | No | Organization ID |

### Example

```json
{
  "type": "openai",
  "data": {
    "apiKey": "sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "organization": "org-xxxxxxxxxxxxxxxxxxxxxxxx"
  }
}
```

## SMTP

Email SMTP credentials.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `host` | string | Yes | SMTP server |
| `port` | integer | Yes | SMTP port |
| `user` | string | Yes | Username |
| `password` | string | Yes | Password |
| `secure` | boolean | No | Use TLS |

### Example

```json
{
  "type": "smtp",
  "data": {
    "host": "smtp.gmail.com",
    "port": 587,
    "user": "you@gmail.com",
    "password": "app-password",
    "secure": true
  }
}
```

## Webhook

Webhook authentication.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `headerName` | string | No | Auth header name |
| `headerValue` | string | No | Auth header value |

### Example

```json
{
  "type": "webhook",
  "data": {
    "headerName": "X-Webhook-Secret",
    "headerValue": "your-secret-token"
  }
}
```

## Custom

For services not listed above.

### Fields

Any custom key-value pairs.

### Example

```json
{
  "type": "custom",
  "data": {
    "apiKey": "xxx",
    "accountId": "123",
    "customField": "value"
  }
}
```

## Credential Validation

Each credential type is validated on creation:

```bash
# Test credential validity
m9m credential test cred-123
```

Returns:

```json
{
  "valid": true,
  "message": "Successfully authenticated"
}
```

## Service-Specific Notes

### OAuth2 Token Refresh

m9m automatically refreshes expired OAuth2 tokens:

1. Detects expired access token
2. Uses refresh token to get new access token
3. Updates stored credential
4. Retries the request

### AWS IAM Roles

For EKS deployments, use IAM roles instead of credentials:

```yaml
# No credentials needed - uses pod IAM role
serviceAccountName: m9m-worker
```

### Vault Integration

For production, integrate with HashiCorp Vault:

```yaml
credentials:
  provider: vault
  vault:
    address: https://vault.example.com
    path: secret/m9m
```
