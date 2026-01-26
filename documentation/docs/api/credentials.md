# Credentials API

API endpoints for managing credentials securely.

## Overview

Credentials store sensitive authentication data (API keys, passwords, tokens) for use in workflow nodes. Credentials are encrypted at rest.

!!! warning "Security Note"
    Credential values are never returned in API responses. Only metadata is returned.

---

## List Credentials

Retrieve all credentials (metadata only).

```http
GET /api/v1/credentials
```

### Query Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `type` | string | Filter by credential type |

### Example Request

```bash
curl http://localhost:8080/api/v1/credentials \
  -H "Authorization: Bearer <token>"
```

### Response

```json
{
  "data": [
    {
      "id": "cred-123",
      "name": "Production API Key",
      "type": "apiKey",
      "createdAt": "2024-01-01T00:00:00Z",
      "updatedAt": "2024-01-15T00:00:00Z"
    },
    {
      "id": "cred-456",
      "name": "Slack Webhook",
      "type": "slackWebhook",
      "createdAt": "2024-01-10T00:00:00Z",
      "updatedAt": "2024-01-10T00:00:00Z"
    }
  ],
  "total": 2
}
```

---

## Get Credential

Retrieve credential metadata (not the secret values).

```http
GET /api/v1/credentials/{id}
```

### Example Request

```bash
curl http://localhost:8080/api/v1/credentials/cred-123 \
  -H "Authorization: Bearer <token>"
```

### Response

```json
{
  "id": "cred-123",
  "name": "Production API Key",
  "type": "apiKey",
  "fields": ["apiKey"],
  "nodesUsing": [
    {
      "workflowId": "wf-123",
      "workflowName": "Data Sync",
      "nodeId": "http-1",
      "nodeName": "API Request"
    }
  ],
  "createdAt": "2024-01-01T00:00:00Z",
  "updatedAt": "2024-01-15T00:00:00Z"
}
```

---

## Create Credential

Create a new credential.

```http
POST /api/v1/credentials
```

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Display name |
| `type` | string | Yes | Credential type |
| `data` | object | Yes | Credential values |

### Credential Types

| Type | Fields |
|------|--------|
| `apiKey` | `apiKey` |
| `basicAuth` | `username`, `password` |
| `oauth2` | `accessToken`, `refreshToken`, `clientId`, `clientSecret` |
| `slackWebhook` | `webhookUrl` |
| `slackApi` | `token` |
| `awsCredentials` | `accessKeyId`, `secretAccessKey`, `region` |
| `postgresCredentials` | `host`, `port`, `database`, `user`, `password` |
| `smtpCredentials` | `host`, `port`, `user`, `password` |

### Example Request

```bash
curl -X POST http://localhost:8080/api/v1/credentials \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My API Key",
    "type": "apiKey",
    "data": {
      "apiKey": "sk-abc123def456"
    }
  }'
```

### Response

```json
{
  "id": "cred-789",
  "name": "My API Key",
  "type": "apiKey",
  "createdAt": "2024-01-26T16:00:00Z"
}
```

---

## Update Credential

Update credential values.

```http
PUT /api/v1/credentials/{id}
```

### Request Body

```json
{
  "name": "Updated Name",
  "data": {
    "apiKey": "new-api-key-value"
  }
}
```

### Example Request

```bash
curl -X PUT http://localhost:8080/api/v1/credentials/cred-123 \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated API Key",
    "data": {
      "apiKey": "sk-new-key-value"
    }
  }'
```

### Response

```json
{
  "id": "cred-123",
  "name": "Updated API Key",
  "type": "apiKey",
  "updatedAt": "2024-01-26T16:00:00Z"
}
```

---

## Delete Credential

Delete a credential.

```http
DELETE /api/v1/credentials/{id}
```

### Example Request

```bash
curl -X DELETE http://localhost:8080/api/v1/credentials/cred-123 \
  -H "Authorization: Bearer <token>"
```

### Response

```
204 No Content
```

### Error: Credential In Use

```json
{
  "error": "Credential is in use by workflows",
  "code": "CREDENTIAL_IN_USE",
  "details": {
    "workflows": ["Data Sync", "Daily Report"]
  }
}
```

Use `?force=true` to delete anyway:

```bash
curl -X DELETE "http://localhost:8080/api/v1/credentials/cred-123?force=true" \
  -H "Authorization: Bearer <token>"
```

---

## Test Credential

Test if credential values are valid.

```http
POST /api/v1/credentials/{id}/test
```

### Example Request

```bash
curl -X POST http://localhost:8080/api/v1/credentials/cred-123/test \
  -H "Authorization: Bearer <token>"
```

### Response (Success)

```json
{
  "valid": true,
  "message": "Credential is valid"
}
```

### Response (Failure)

```json
{
  "valid": false,
  "message": "Authentication failed",
  "details": {
    "error": "Invalid API key"
  }
}
```

---

## Using Credentials in Workflows

Reference credentials in node parameters:

```json
{
  "type": "n8n-nodes-base.httpRequest",
  "parameters": {
    "url": "https://api.example.com/data",
    "method": "GET",
    "headers": {
      "Authorization": "Bearer {{ $credentials.myApiKey.apiKey }}"
    }
  },
  "credentials": {
    "myApiKey": "cred-123"
  }
}
```

---

## Credential Type Examples

### API Key

```json
{
  "name": "OpenAI API Key",
  "type": "apiKey",
  "data": {
    "apiKey": "sk-..."
  }
}
```

### Basic Auth

```json
{
  "name": "Service Login",
  "type": "basicAuth",
  "data": {
    "username": "user@example.com",
    "password": "secret123"
  }
}
```

### AWS Credentials

```json
{
  "name": "AWS Production",
  "type": "awsCredentials",
  "data": {
    "accessKeyId": "AKIAIOSFODNN7EXAMPLE",
    "secretAccessKey": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
    "region": "us-east-1"
  }
}
```

### Database

```json
{
  "name": "Production Database",
  "type": "postgresCredentials",
  "data": {
    "host": "db.example.com",
    "port": 5432,
    "database": "myapp",
    "user": "appuser",
    "password": "secret123"
  }
}
```

---

## Error Responses

### 400 Bad Request

```json
{
  "error": "Invalid credential data",
  "code": "VALIDATION_ERROR",
  "details": {
    "apiKey": "required field missing"
  }
}
```

### 404 Not Found

```json
{
  "error": "Credential not found",
  "code": "NOT_FOUND"
}
```

---

## See Also

- [Credentials Guide](../credentials/index.md) - Credential management
- [Credential Types](../credentials/types.md) - All credential types
