# Credentials Overview

Securely store and manage authentication credentials for nodes.

## What are Credentials?

Credentials store sensitive authentication data:

- API keys
- OAuth tokens
- Database passwords
- Service account keys

## Security Features

| Feature | Description |
|---------|-------------|
| AES-256 encryption | Credentials encrypted at rest |
| Access control | Per-workflow credential access |
| Audit logging | Track credential usage |
| Secure storage | Never exposed in logs or API |

## Credential Structure

```json
{
  "id": "cred-123",
  "name": "My API Key",
  "type": "apiKey",
  "data": {
    "apiKey": "encrypted..."
  },
  "createdAt": "2024-01-15T10:00:00Z",
  "updatedAt": "2024-01-15T10:00:00Z"
}
```

## Creating Credentials

### Via CLI

```bash
m9m credential create --name "GitHub Token" --type oauth2 --data @creds.json
```

### Via API

```bash
curl -X POST http://localhost:8080/api/v1/credentials \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My API Key",
    "type": "apiKey",
    "data": {
      "apiKey": "sk-xxx..."
    }
  }'
```

## Using Credentials

### In Workflow JSON

Reference credentials by ID or name:

```json
{
  "nodes": [
    {
      "type": "n8n-nodes-base.httpRequest",
      "parameters": {
        "authentication": "genericCredentialType",
        "genericAuthType": "httpHeaderAuth"
      },
      "credentials": {
        "httpHeaderAuth": {
          "id": "cred-123",
          "name": "My API Key"
        }
      }
    }
  ]
}
```

### In Expressions

Access credential data:

```javascript
{{ $credentials.apiKey }}
```

## Managing Credentials

### List Credentials

```bash
m9m credential list
```

### Get Credential

```bash
m9m credential get cred-123
```

### Update Credential

```bash
m9m credential update cred-123 --data @new-creds.json
```

### Delete Credential

```bash
m9m credential delete cred-123
```

## Best Practices

### 1. Use Descriptive Names

```
Good: "Production GitHub - CI/CD"
Bad:  "github1"
```

### 2. Rotate Regularly

- Set up credential rotation schedules
- Update workflows after rotation
- Monitor for failures after rotation

### 3. Limit Scope

- Grant minimal permissions
- Use separate credentials per environment
- Avoid shared credentials across workflows

### 4. Audit Access

```bash
# View credential usage
m9m credential audit cred-123
```

## Environment Variables

For local development, use environment variables:

```bash
export API_KEY="your-key"
```

Access in expressions:

```javascript
{{ $env.API_KEY }}
```

## Encryption

### Configuration

Set encryption key:

```yaml
credentials:
  encryptionKey: "your-32-byte-encryption-key-here"
```

Or via environment:

```bash
export M9M_CREDENTIALS_ENCRYPTION_KEY="your-key"
```

### Key Management

- Use strong, random 32-byte keys
- Store keys securely (vault, KMS)
- Rotate keys periodically

## Credential Types

See [Credential Types](types.md) for all supported types:

- API Key
- OAuth2
- Basic Auth
- HTTP Header
- Database connections
- Cloud provider credentials

## Next Steps

- [Credential Types](types.md) - All credential types
- [API Reference](../api/credentials.md) - Credential API endpoints
