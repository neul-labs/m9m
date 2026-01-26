# Webhooks Overview

Trigger workflows via HTTP requests.

## What are Webhooks?

Webhooks allow external systems to trigger workflows:

```
External Service → HTTP POST → m9m Webhook → Workflow Execution
```

## Creating a Webhook

Add a Webhook node as the workflow trigger:

```json
{
  "nodes": [
    {
      "id": "webhook",
      "name": "Webhook Trigger",
      "type": "n8n-nodes-base.webhook",
      "position": [250, 300],
      "parameters": {
        "path": "/my-webhook",
        "httpMethod": "POST"
      }
    }
  ]
}
```

## Webhook URL

After activating the workflow, the webhook is available at:

```
http://localhost:8080/webhook/my-webhook
```

Production URL:

```
https://your-domain.com/webhook/my-webhook
```

## Triggering Webhooks

### POST Request

```bash
curl -X POST http://localhost:8080/webhook/my-webhook \
  -H "Content-Type: application/json" \
  -d '{"event": "user_created", "user_id": 123}'
```

### GET Request

```bash
curl "http://localhost:8080/webhook/my-webhook?event=ping"
```

## Webhook Configuration

### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `path` | string | Required | URL path for webhook |
| `httpMethod` | string | `POST` | HTTP method to accept |
| `responseMode` | string | `onReceived` | When to respond |
| `responseData` | string | `allEntries` | What to return |

### HTTP Methods

```json
{
  "parameters": {
    "httpMethod": "POST"  // or GET, PUT, DELETE
  }
}
```

### Response Mode

| Mode | Description |
|------|-------------|
| `onReceived` | Respond immediately (default) |
| `lastNode` | Respond after workflow completes |

## Accessing Webhook Data

In downstream nodes:

### Request Body

```javascript
{{ $json.body }}           // Full body
{{ $json.body.event }}     // Specific field
```

### Query Parameters

```javascript
{{ $json.query }}          // All query params
{{ $json.query.token }}    // Specific param
```

### Headers

```javascript
{{ $json.headers }}                    // All headers
{{ $json.headers["x-custom-header"] }} // Specific header
```

### Request Info

```javascript
{{ $json.webhookUrl }}     // Full webhook URL
{{ $json.httpMethod }}     // Request method
```

## Response Configuration

### Immediate Response

```json
{
  "parameters": {
    "responseMode": "onReceived",
    "responseCode": 200,
    "responseData": "allEntries"
  }
}
```

### Custom Response

```json
{
  "parameters": {
    "responseMode": "lastNode",
    "responseData": "noData"
  }
}
```

Then add a Set node at the end:

```json
{
  "name": "Response",
  "type": "n8n-nodes-base.set",
  "parameters": {
    "assignments": [
      {"name": "status", "value": "success"},
      {"name": "message", "value": "Processed"}
    ]
  }
}
```

## Webhook Types

### Production Webhook

Active when workflow is activated:

```
POST /webhook/my-webhook
```

### Test Webhook

For testing without activation:

```
POST /webhook-test/my-webhook
```

## Common Use Cases

### GitHub Webhooks

```json
{
  "parameters": {
    "path": "/github-events",
    "httpMethod": "POST"
  }
}
```

Filter by event:

```javascript
{{ $json.headers["x-github-event"] === "push" }}
```

### Stripe Webhooks

```json
{
  "parameters": {
    "path": "/stripe",
    "httpMethod": "POST"
  }
}
```

Access event:

```javascript
{{ $json.body.type }}     // "payment_intent.succeeded"
{{ $json.body.data }}     // Event data
```

### Slack Events

```json
{
  "parameters": {
    "path": "/slack-events",
    "httpMethod": "POST",
    "responseMode": "lastNode"
  }
}
```

Handle challenge:

```javascript
// Return challenge for Slack URL verification
{{ $json.body.challenge }}
```

## Webhook Management

### List Active Webhooks

```bash
curl http://localhost:8080/api/v1/webhooks \
  -H "Authorization: Bearer <token>"
```

### Webhook Status

```bash
curl http://localhost:8080/api/v1/webhooks/my-webhook \
  -H "Authorization: Bearer <token>"
```

## Best Practices

### 1. Use Descriptive Paths

```
/github-push-events     # Good
/webhook1               # Less clear
```

### 2. Validate Input

Add a Filter node after webhook:

```json
{
  "type": "n8n-nodes-base.filter",
  "parameters": {
    "conditions": [
      {
        "leftValue": "={{ $json.body.event }}",
        "operator": "exists"
      }
    ]
  }
}
```

### 3. Respond Quickly

For slow workflows, use immediate response:

```json
{
  "responseMode": "onReceived"
}
```

### 4. Handle Errors

Return appropriate status codes:

```json
{
  "responseCode": 400  // For invalid requests
}
```

### 5. Log Requests

Add a Code node to log:

```javascript
console.log('Webhook received:', JSON.stringify($json.body));
return items;
```

## Error Handling

### Invalid Webhook

```
404 Not Found
Webhook not found or workflow not active
```

### Invalid Method

```
405 Method Not Allowed
Webhook only accepts POST requests
```

### Processing Error

```
500 Internal Server Error
Workflow execution failed
```

## Rate Limiting

Configure rate limits:

```yaml
webhooks:
  rateLimit:
    enabled: true
    requests: 100
    window: 60  # seconds
```

Returns `429 Too Many Requests` when exceeded.

## Next Steps

- [Webhook Authentication](authentication.md) - Secure webhooks
- [API Reference](../api/webhooks.md) - Webhook API
- [Workflows](../workflows/index.md) - Creating workflows
