# REST API

m9m provides a comprehensive REST API for managing workflows, executions, and credentials.

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

### API Key

Include API key in header:

```bash
curl -H "X-API-Key: your-api-key" \
  http://localhost:8080/api/v1/workflows
```

### Bearer Token

```bash
curl -H "Authorization: Bearer your-token" \
  http://localhost:8080/api/v1/workflows
```

## Response Format

All responses are JSON:

```json
{
  "success": true,
  "data": {},
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 100
  }
}
```

### Error Response

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid workflow definition",
    "details": {
      "field": "nodes",
      "reason": "At least one node is required"
    }
  }
}
```

## Workflows

### List Workflows

```http
GET /api/v1/workflows
```

Query parameters:
| Parameter | Type | Description |
|-----------|------|-------------|
| `page` | int | Page number (default: 1) |
| `limit` | int | Items per page (default: 20, max: 100) |
| `active` | bool | Filter by active status |
| `search` | string | Search in name/description |

Response:
```json
{
  "success": true,
  "data": [
    {
      "id": "wf-001",
      "name": "My Workflow",
      "active": true,
      "createdAt": "2024-01-15T10:00:00Z",
      "updatedAt": "2024-01-15T12:30:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 45
  }
}
```

### Get Workflow

```http
GET /api/v1/workflows/:id
```

Response:
```json
{
  "success": true,
  "data": {
    "id": "wf-001",
    "name": "My Workflow",
    "active": true,
    "nodes": [...],
    "connections": {...},
    "settings": {...},
    "createdAt": "2024-01-15T10:00:00Z",
    "updatedAt": "2024-01-15T12:30:00Z"
  }
}
```

### Create Workflow

```http
POST /api/v1/workflows
```

Request body:
```json
{
  "name": "New Workflow",
  "nodes": [
    {
      "id": "start",
      "type": "n8n-nodes-base.start",
      "position": [250, 300],
      "parameters": {}
    }
  ],
  "connections": {},
  "settings": {}
}
```

Response:
```json
{
  "success": true,
  "data": {
    "id": "wf-002",
    "name": "New Workflow",
    "active": false,
    "createdAt": "2024-01-15T14:00:00Z"
  }
}
```

### Update Workflow

```http
PUT /api/v1/workflows/:id
```

Request body:
```json
{
  "name": "Updated Workflow",
  "nodes": [...],
  "connections": {...}
}
```

### Delete Workflow

```http
DELETE /api/v1/workflows/:id
```

### Activate Workflow

```http
POST /api/v1/workflows/:id/activate
```

### Deactivate Workflow

```http
POST /api/v1/workflows/:id/deactivate
```

## Executions

### Execute Workflow

```http
POST /api/v1/workflows/:id/execute
```

Request body (optional):
```json
{
  "inputData": {
    "key": "value"
  },
  "mode": "manual"
}
```

Response:
```json
{
  "success": true,
  "data": {
    "executionId": "exec-001",
    "status": "running",
    "startedAt": "2024-01-15T14:30:00Z"
  }
}
```

### Execute Workflow (Inline)

Execute a workflow without saving it:

```http
POST /api/v1/workflows/execute
```

Request body:
```json
{
  "nodes": [...],
  "connections": {...},
  "inputData": {}
}
```

### Get Execution

```http
GET /api/v1/executions/:id
```

Response:
```json
{
  "success": true,
  "data": {
    "id": "exec-001",
    "workflowId": "wf-001",
    "status": "success",
    "startedAt": "2024-01-15T14:30:00Z",
    "finishedAt": "2024-01-15T14:30:02Z",
    "duration": 2000,
    "data": {
      "resultData": {...}
    }
  }
}
```

### List Executions

```http
GET /api/v1/executions
```

Query parameters:
| Parameter | Type | Description |
|-----------|------|-------------|
| `workflowId` | string | Filter by workflow |
| `status` | string | Filter by status (success, error, running) |
| `startedAfter` | datetime | Filter by start time |
| `startedBefore` | datetime | Filter by start time |
| `limit` | int | Items per page |

### Stop Execution

```http
POST /api/v1/executions/:id/stop
```

### Retry Execution

```http
POST /api/v1/executions/:id/retry
```

## Credentials

### List Credentials

```http
GET /api/v1/credentials
```

Response:
```json
{
  "success": true,
  "data": [
    {
      "id": "1",
      "name": "My API Key",
      "type": "apiKey",
      "createdAt": "2024-01-10T10:00:00Z"
    }
  ]
}
```

### Create Credential

```http
POST /api/v1/credentials
```

Request body:
```json
{
  "name": "New Credential",
  "type": "apiKey",
  "data": {
    "apiKey": "your-api-key"
  }
}
```

### Update Credential

```http
PUT /api/v1/credentials/:id
```

### Delete Credential

```http
DELETE /api/v1/credentials/:id
```

### Test Credential

```http
POST /api/v1/credentials/:id/test
```

Response:
```json
{
  "success": true,
  "data": {
    "valid": true,
    "message": "Connection successful"
  }
}
```

## Variables

### List Variables

```http
GET /api/v1/variables
```

### Get Variable

```http
GET /api/v1/variables/:key
```

### Set Variable

```http
PUT /api/v1/variables/:key
```

Request body:
```json
{
  "value": "variable-value",
  "secret": false
}
```

### Delete Variable

```http
DELETE /api/v1/variables/:key
```

## Nodes

### List Available Nodes

```http
GET /api/v1/nodes
```

Response:
```json
{
  "success": true,
  "data": [
    {
      "type": "n8n-nodes-base.httpRequest",
      "name": "HTTP Request",
      "category": "http",
      "description": "Make HTTP requests"
    }
  ]
}
```

### Get Node Info

```http
GET /api/v1/nodes/:type
```

## Health & Status

### Health Check

```http
GET /health
```

Response:
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": "5h30m",
  "checks": {
    "database": "ok",
    "queue": "ok"
  }
}
```

### Readiness Check

```http
GET /ready
```

### Metrics

```http
GET /metrics
```

Returns Prometheus-format metrics.

## Pagination

All list endpoints support pagination:

```http
GET /api/v1/workflows?page=2&limit=50
```

Response includes metadata:
```json
{
  "meta": {
    "page": 2,
    "limit": 50,
    "total": 150,
    "totalPages": 3
  }
}
```

## Rate Limiting

API requests are rate-limited:

- Default: 100 requests per minute
- Authenticated: 1000 requests per minute

Rate limit headers:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1705329600
```

## Error Codes

| Code | Description |
|------|-------------|
| `VALIDATION_ERROR` | Invalid request data |
| `NOT_FOUND` | Resource not found |
| `UNAUTHORIZED` | Authentication required |
| `FORBIDDEN` | Insufficient permissions |
| `CONFLICT` | Resource conflict |
| `RATE_LIMITED` | Too many requests |
| `INTERNAL_ERROR` | Server error |

## SDK Examples

### Go

```go
import "github.com/m9m/m9m-go-sdk"

client := m9m.NewClient("http://localhost:8080", "your-api-key")

// List workflows
workflows, err := client.Workflows.List(nil)

// Execute workflow
execution, err := client.Workflows.Execute("wf-001", map[string]interface{}{
    "inputData": data,
})
```

### Python

```python
import m9m

client = m9m.Client("http://localhost:8080", api_key="your-api-key")

# List workflows
workflows = client.workflows.list()

# Execute workflow
execution = client.workflows.execute("wf-001", input_data={"key": "value"})
```

### JavaScript

```javascript
const { M9MClient } = require('@m9m/sdk');

const client = new M9MClient({
  baseUrl: 'http://localhost:8080',
  apiKey: 'your-api-key'
});

// List workflows
const workflows = await client.workflows.list();

// Execute workflow
const execution = await client.workflows.execute('wf-001', {
  inputData: { key: 'value' }
});
```

## Next Steps

- [Endpoints Reference](endpoints.md) - Complete endpoint list
- [Authentication](authentication.md) - Auth configuration
- [Webhooks](webhooks.md) - Webhook setup
