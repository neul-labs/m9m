# Webhooks

Webhooks allow external services to trigger m9m workflows via HTTP requests.

## Webhook URLs

### Production Webhooks

```
https://your-m9m-host/webhook/<path>
```

### Test Webhooks

For development and testing:

```
https://your-m9m-host/webhook-test/<path>
```

Test webhooks don't require the workflow to be active.

## Creating Webhooks

### In Workflow

Add a Webhook node:

```json
{
  "id": "webhook-trigger",
  "type": "n8n-nodes-base.webhook",
  "parameters": {
    "path": "my-webhook",
    "httpMethod": "POST",
    "responseMode": "onReceived"
  }
}
```

### Webhook Parameters

| Parameter | Description |
|-----------|-------------|
| `path` | URL path (e.g., `my-webhook` → `/webhook/my-webhook`) |
| `httpMethod` | HTTP method (GET, POST, PUT, DELETE, PATCH) |
| `responseMode` | When to respond (see below) |
| `responseData` | What to return |

## Response Modes

### Immediately (`onReceived`)

Respond immediately upon receiving the request:

```json
{
  "parameters": {
    "responseMode": "onReceived",
    "responseCode": 202,
    "responseData": "allEntries"
  }
}
```

### After Workflow (`lastNode`)

Wait for workflow to complete:

```json
{
  "parameters": {
    "responseMode": "lastNode"
  }
}
```

### Custom Response (`responseNode`)

Use a Respond to Webhook node:

```json
{
  "id": "respond",
  "type": "n8n-nodes-base.respondToWebhook",
  "parameters": {
    "respondWith": "json",
    "responseBody": {
      "success": true,
      "id": "={{ $json.createdId }}"
    }
  }
}
```

## Accessing Webhook Data

### Query Parameters

```javascript
{{ $json.query.paramName }}
```

### Request Body

```javascript
{{ $json.body.fieldName }}
{{ $json.body }}  // entire body
```

### Headers

```javascript
{{ $json.headers['content-type'] }}
{{ $json.headers.authorization }}
```

### Path Parameters

For path `/webhook/users/:userId`:

```javascript
{{ $json.params.userId }}
```

## Authentication

### Header Authentication

```json
{
  "parameters": {
    "authentication": "headerAuth",
    "headerName": "X-Webhook-Secret",
    "headerValue": "={{ $env.WEBHOOK_SECRET }}"
  }
}
```

### Basic Auth

```json
{
  "parameters": {
    "authentication": "basicAuth"
  },
  "credentials": {
    "httpBasicAuth": {"id": "1", "name": "Webhook Auth"}
  }
}
```

### HMAC Signature Verification

```json
{
  "parameters": {
    "authentication": "hmacSignature",
    "signatureHeader": "X-Hub-Signature-256",
    "algorithm": "sha256"
  },
  "credentials": {
    "hmacAuth": {"id": "1", "name": "HMAC Secret"}
  }
}
```

## Webhook Examples

### GitHub Webhook

```json
{
  "id": "github-webhook",
  "type": "n8n-nodes-base.webhook",
  "parameters": {
    "path": "github-events",
    "httpMethod": "POST",
    "authentication": "hmacSignature",
    "signatureHeader": "X-Hub-Signature-256"
  }
}
```

Process events:
```javascript
const event = $json.headers['x-github-event'];

if (event === 'push') {
  return { action: 'build', branch: $json.body.ref };
} else if (event === 'pull_request') {
  return { action: 'review', pr: $json.body.number };
}
```

### Stripe Webhook

```json
{
  "parameters": {
    "path": "stripe-events",
    "httpMethod": "POST"
  }
}
```

Verify signature in Code node:
```javascript
const stripe = require('stripe');
const endpointSecret = $env.STRIPE_WEBHOOK_SECRET;

const sig = $json.headers['stripe-signature'];
const event = stripe.webhooks.constructEvent(
  $json.rawBody,
  sig,
  endpointSecret
);

return { event: event.type, data: event.data.object };
```

### Slack Webhook

```json
{
  "parameters": {
    "path": "slack-events",
    "httpMethod": "POST"
  }
}
```

Handle URL verification:
```javascript
// Slack URL verification
if ($json.body.type === 'url_verification') {
  return { challenge: $json.body.challenge };
}

// Process events
return $json.body.event;
```

## Binary Data

### Receive File Upload

```json
{
  "parameters": {
    "path": "upload",
    "httpMethod": "POST",
    "options": {
      "rawBody": true,
      "binaryData": true
    }
  }
}
```

Access uploaded file:
```javascript
const file = $binary.data;
const filename = $json.headers['x-filename'];
```

### Send File Response

```json
{
  "id": "respond",
  "type": "n8n-nodes-base.respondToWebhook",
  "parameters": {
    "respondWith": "binary",
    "inputDataFieldName": "data",
    "options": {
      "headers": {
        "Content-Disposition": "attachment; filename=\"report.pdf\""
      }
    }
  }
}
```

## Error Handling

### Custom Error Response

```json
{
  "nodes": [
    {
      "id": "webhook",
      "type": "n8n-nodes-base.webhook",
      "parameters": {
        "responseMode": "responseNode"
      }
    },
    {
      "id": "validate",
      "type": "n8n-nodes-base.if",
      "continueOnFail": true
    },
    {
      "id": "error-response",
      "type": "n8n-nodes-base.respondToWebhook",
      "parameters": {
        "respondWith": "json",
        "responseCode": 400,
        "responseBody": {
          "error": "Validation failed",
          "message": "={{ $json.error }}"
        }
      }
    }
  ]
}
```

### Timeout Configuration

```yaml
webhooks:
  timeout: 30s
  maxBodySize: 10MB
```

## Security Best Practices

### IP Whitelisting

```yaml
webhooks:
  allowedIps:
    - "192.30.252.0/22"  # GitHub
    - "54.240.0.0/18"    # AWS SNS
```

### Rate Limiting

```yaml
webhooks:
  rateLimit:
    enabled: true
    requests: 100
    period: 1m
    byPath: true
```

### Request Validation

Always validate incoming requests:

```javascript
// Validate required fields
if (!$json.body.orderId) {
  throw new Error('orderId is required');
}

// Validate data types
if (typeof $json.body.amount !== 'number') {
  throw new Error('amount must be a number');
}

// Validate ranges
if ($json.body.amount < 0) {
  throw new Error('amount must be positive');
}
```

### Idempotency

Handle duplicate webhook deliveries:

```javascript
const eventId = $json.headers['x-event-id'];

// Check if already processed
const processed = await redis.get(`event:${eventId}`);
if (processed) {
  return { skipped: true, reason: 'duplicate' };
}

// Mark as processed
await redis.set(`event:${eventId}`, '1', 'EX', 86400);

// Process event
return processEvent($json.body);
```

## Monitoring Webhooks

### List Active Webhooks

```bash
m9m webhooks list
```

```
PATH                  METHOD    WORKFLOW          STATUS
/webhook/github       POST      CI/CD Pipeline    active
/webhook/stripe       POST      Payment Handler   active
/webhook/slack        POST      Slack Bot         active
```

### Webhook Metrics

Monitor via Prometheus:

```
m9m_webhook_requests_total{path="/github", status="200"}
m9m_webhook_duration_seconds{path="/github"}
m9m_webhook_errors_total{path="/github", error="timeout"}
```

## Debugging

### Test Webhook

```bash
curl -X POST http://localhost:8080/webhook-test/my-webhook \
  -H "Content-Type: application/json" \
  -d '{"test": true}'
```

### View Webhook Logs

```bash
m9m logs --filter webhook --tail
```

## Next Steps

- [REST API](rest-api.md) - API reference
- [Authentication](authentication.md) - Secure webhooks
- [Error Handling](../user-guide/error-handling.md) - Handle failures
