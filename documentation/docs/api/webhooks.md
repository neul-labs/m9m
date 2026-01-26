# Webhooks API

API endpoints for configuring webhook triggers.

## Overview

Webhooks allow external services to trigger workflow execution via HTTP requests.

---

## Webhook URL Format

```
http://localhost:8080/webhook/{path}
```

Where `{path}` is defined in the Webhook node configuration.

---

## List Webhooks

Retrieve all configured webhooks.

```http
GET /api/v1/webhooks
```

### Example Request

```bash
curl http://localhost:8080/api/v1/webhooks \
  -H "Authorization: Bearer <token>"
```

### Response

```json
{
  "data": [
    {
      "id": "wh-123",
      "workflowId": "550e8400-e29b-41d4-a716-446655440000",
      "workflowName": "GitHub Events",
      "nodeId": "webhook-1",
      "path": "/github",
      "method": "POST",
      "active": true,
      "url": "http://localhost:8080/webhook/github",
      "authentication": "none",
      "createdAt": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 3
}
```

---

## Get Webhook

Retrieve webhook details.

```http
GET /api/v1/webhooks/{id}
```

### Example Request

```bash
curl http://localhost:8080/api/v1/webhooks/wh-123 \
  -H "Authorization: Bearer <token>"
```

### Response

```json
{
  "id": "wh-123",
  "workflowId": "550e8400-e29b-41d4-a716-446655440000",
  "workflowName": "GitHub Events",
  "nodeId": "webhook-1",
  "path": "/github",
  "method": "POST",
  "active": true,
  "url": "http://localhost:8080/webhook/github",
  "authentication": "headerAuth",
  "authConfig": {
    "headerName": "X-Hub-Signature-256"
  },
  "responseMode": "lastNode",
  "responseData": "firstEntryJson",
  "stats": {
    "totalCalls": 150,
    "successCalls": 148,
    "failedCalls": 2,
    "lastCall": "2024-01-26T15:30:00Z"
  },
  "createdAt": "2024-01-01T00:00:00Z"
}
```

---

## Trigger Webhook

Call a webhook endpoint to trigger workflow execution.

```http
POST /webhook/{path}
GET /webhook/{path}
```

The HTTP method must match the webhook configuration.

### Example Request

```bash
curl -X POST http://localhost:8080/webhook/github \
  -H "Content-Type: application/json" \
  -H "X-Hub-Signature-256: sha256=abc123..." \
  -d '{
    "action": "push",
    "repository": {
      "name": "my-repo"
    }
  }'
```

### Response Modes

#### onReceived (Default)

Returns immediately with 200 OK:

```json
{
  "message": "Workflow triggered"
}
```

#### lastNode

Waits for workflow completion and returns last node output:

```json
{
  "result": "processed",
  "items": 5
}
```

#### responseNode

Returns output from a specific response node.

---

## Webhook Authentication

### No Authentication

```json
{
  "authentication": "none"
}
```

### Header Authentication

```json
{
  "authentication": "headerAuth",
  "authConfig": {
    "headerName": "X-API-Key",
    "headerValue": "expected-value"
  }
}
```

Request must include:

```bash
curl -X POST http://localhost:8080/webhook/my-webhook \
  -H "X-API-Key: expected-value" \
  -d '{"data": "value"}'
```

### Basic Authentication

```json
{
  "authentication": "basicAuth",
  "authConfig": {
    "username": "user",
    "password": "pass"
  }
}
```

Request:

```bash
curl -X POST http://localhost:8080/webhook/my-webhook \
  -u "user:pass" \
  -d '{"data": "value"}'
```

---

## Webhook Execution History

Get webhook call history.

```http
GET /api/v1/webhooks/{id}/history
```

### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | integer | 20 | Maximum entries |
| `status` | string | - | Filter by status |

### Example Request

```bash
curl "http://localhost:8080/api/v1/webhooks/wh-123/history?limit=10" \
  -H "Authorization: Bearer <token>"
```

### Response

```json
{
  "data": [
    {
      "id": "call-100",
      "timestamp": "2024-01-26T15:30:00Z",
      "status": "success",
      "executionId": "exec-500",
      "requestMethod": "POST",
      "requestHeaders": {
        "content-type": "application/json"
      },
      "requestBody": "{...}",
      "responseStatus": 200,
      "responseBody": "{...}",
      "duration": 1234
    }
  ],
  "total": 150
}
```

---

## Test Webhook

Test a webhook without triggering execution.

```http
POST /api/v1/webhooks/{id}/test
```

### Request Body

```json
{
  "method": "POST",
  "headers": {
    "Content-Type": "application/json"
  },
  "body": {
    "test": "data"
  }
}
```

### Response

```json
{
  "valid": true,
  "message": "Webhook configuration is valid",
  "parsedBody": {
    "test": "data"
  }
}
```

---

## Webhook Data in Workflows

The Webhook node outputs:

```json
{
  "json": {
    "headers": {
      "content-type": "application/json",
      "x-forwarded-for": "1.2.3.4"
    },
    "params": {},
    "query": {
      "action": "test"
    },
    "body": {
      "data": "value"
    }
  }
}
```

Access in expressions:

```javascript
// Request body
{{ $json.body.data }}

// Query parameters
{{ $json.query.action }}

// Headers
{{ $json.headers['content-type'] }}
```

---

## Common Webhook Patterns

### GitHub Webhooks

```json
{
  "path": "/github",
  "method": "POST",
  "authentication": "headerAuth",
  "authConfig": {
    "headerName": "X-Hub-Signature-256"
  }
}
```

### Stripe Webhooks

```json
{
  "path": "/stripe",
  "method": "POST",
  "authentication": "headerAuth",
  "authConfig": {
    "headerName": "Stripe-Signature"
  }
}
```

### Slack Commands

```json
{
  "path": "/slack-command",
  "method": "POST",
  "responseMode": "onReceived"
}
```

---

## Error Responses

### 401 Unauthorized

```json
{
  "error": "Authentication failed",
  "code": "UNAUTHORIZED"
}
```

### 404 Not Found

```json
{
  "error": "Webhook not found",
  "code": "NOT_FOUND"
}
```

### 405 Method Not Allowed

```json
{
  "error": "Method not allowed",
  "code": "METHOD_NOT_ALLOWED",
  "allowed": ["POST"]
}
```

### 503 Workflow Inactive

```json
{
  "error": "Workflow is not active",
  "code": "WORKFLOW_INACTIVE"
}
```

---

## See Also

- [Webhooks Guide](../webhooks/index.md) - Webhook concepts
- [Webhook Authentication](../webhooks/authentication.md) - Auth methods
- [Webhook Node](../nodes/triggers.md#webhook-node) - Node configuration
