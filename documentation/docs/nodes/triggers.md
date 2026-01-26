# Trigger Nodes

Trigger nodes start workflow execution based on external events or schedules.

## Webhook Node

Receive HTTP requests to trigger workflows.

### Type

```
n8n-nodes-base.webhook
```

### Description

Creates an HTTP endpoint that triggers workflow execution when called. Supports various HTTP methods and authentication options.

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `path` | string | Yes | - | Webhook endpoint path |
| `httpMethod` | string | No | `POST` | HTTP method to accept |
| `authentication` | string | No | `none` | Auth type |
| `headerName` | string | No | - | Header name for auth |
| `headerValue` | string | No | - | Expected header value |

### HTTP Methods

| Method | Use Case |
|--------|----------|
| `GET` | Simple triggers, health checks |
| `POST` | Data submission (most common) |
| `PUT` | Resource updates |
| `DELETE` | Resource deletion |
| `PATCH` | Partial updates |

### Example

```json
{
  "id": "webhook-1",
  "name": "Webhook",
  "type": "n8n-nodes-base.webhook",
  "position": [250, 300],
  "parameters": {
    "path": "/my-workflow",
    "httpMethod": "POST"
  }
}
```

### Webhook URL

After creating a workflow with this webhook, the endpoint is:

```
http://localhost:8080/webhook/my-workflow
```

### Triggering the Webhook

```bash
curl -X POST http://localhost:8080/webhook/my-workflow \
  -H "Content-Type: application/json" \
  -d '{"name": "John", "action": "signup"}'
```

### Output

The webhook node outputs request data:

```json
{
  "json": {
    "headers": {
      "content-type": "application/json",
      "user-agent": "curl/7.68.0"
    },
    "params": {},
    "query": {},
    "body": {
      "name": "John",
      "action": "signup"
    }
  }
}
```

| Field | Description |
|-------|-------------|
| `headers` | HTTP request headers |
| `params` | URL path parameters |
| `query` | Query string parameters |
| `body` | Request body (parsed) |

### Authentication Options

#### No Authentication

```json
{
  "type": "n8n-nodes-base.webhook",
  "parameters": {
    "path": "/public-webhook",
    "httpMethod": "POST",
    "authentication": "none"
  }
}
```

#### Header Authentication

```json
{
  "type": "n8n-nodes-base.webhook",
  "parameters": {
    "path": "/secure-webhook",
    "httpMethod": "POST",
    "authentication": "headerAuth",
    "headerName": "X-API-Key",
    "headerValue": "your-secret-key"
  }
}
```

Call with:
```bash
curl -X POST http://localhost:8080/webhook/secure-webhook \
  -H "X-API-Key: your-secret-key" \
  -d '{"data": "value"}'
```

#### Basic Authentication

```json
{
  "type": "n8n-nodes-base.webhook",
  "parameters": {
    "path": "/basic-auth-webhook",
    "httpMethod": "POST",
    "authentication": "basicAuth"
  }
}
```

### Query Parameters

Access query string values:

```
POST /webhook/my-workflow?userId=123&action=update
```

Output:
```json
{
  "json": {
    "query": {
      "userId": "123",
      "action": "update"
    }
  }
}
```

### Use Cases

| Use Case | Configuration |
|----------|---------------|
| GitHub webhooks | POST, path: `/github` |
| Stripe events | POST, path: `/stripe`, header auth |
| Slack commands | POST, path: `/slack` |
| Form submissions | POST, path: `/form` |
| API endpoints | GET/POST, path: `/api/{resource}` |

---

## Cron Node

Trigger workflows on a schedule using cron expressions.

### Type

```
n8n-nodes-base.cron
```

### Description

Executes workflows at scheduled intervals using standard cron syntax.

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `cronExpression` | string | Yes | Cron schedule expression |

### Cron Expression Format

```
┌───────────── minute (0 - 59)
│ ┌───────────── hour (0 - 23)
│ │ ┌───────────── day of month (1 - 31)
│ │ │ ┌───────────── month (1 - 12)
│ │ │ │ ┌───────────── day of week (0 - 6) (Sunday = 0)
│ │ │ │ │
* * * * *
```

### Common Patterns

| Expression | Description |
|------------|-------------|
| `* * * * *` | Every minute |
| `*/5 * * * *` | Every 5 minutes |
| `0 * * * *` | Every hour |
| `0 9 * * *` | Daily at 9 AM |
| `0 9 * * MON` | Every Monday at 9 AM |
| `0 9 * * 1-5` | Weekdays at 9 AM |
| `0 0 1 * *` | First of each month |
| `0 0 * * 0` | Every Sunday at midnight |

### Example

```json
{
  "id": "cron-1",
  "name": "Daily Report",
  "type": "n8n-nodes-base.cron",
  "position": [250, 300],
  "parameters": {
    "cronExpression": "0 9 * * MON-FRI"
  }
}
```

### Output

The Cron node outputs trigger metadata:

```json
{
  "json": {
    "timestamp": "2024-01-26T09:00:00Z",
    "triggered": true
  }
}
```

### Examples

#### Every Hour

```json
{
  "type": "n8n-nodes-base.cron",
  "parameters": {
    "cronExpression": "0 * * * *"
  }
}
```

#### Daily at Midnight

```json
{
  "type": "n8n-nodes-base.cron",
  "parameters": {
    "cronExpression": "0 0 * * *"
  }
}
```

#### Every 15 Minutes (Business Hours)

```json
{
  "type": "n8n-nodes-base.cron",
  "parameters": {
    "cronExpression": "*/15 9-17 * * 1-5"
  }
}
```

#### Monthly on the 1st

```json
{
  "type": "n8n-nodes-base.cron",
  "parameters": {
    "cronExpression": "0 0 1 * *"
  }
}
```

### Use Cases

| Use Case | Cron Expression |
|----------|-----------------|
| Backup database | `0 2 * * *` (2 AM daily) |
| Send daily digest | `0 9 * * *` (9 AM daily) |
| Cleanup old data | `0 0 * * 0` (Sunday midnight) |
| Check API status | `*/5 * * * *` (every 5 min) |
| Monthly report | `0 9 1 * *` (1st of month) |

### Timezone Handling

Schedules run in the server's timezone by default. Configure timezone in the scheduler settings.

---

## Quick Reference

| Node | Type | Trigger |
|------|------|---------|
| Webhook | `n8n-nodes-base.webhook` | HTTP request |
| Cron | `n8n-nodes-base.cron` | Time schedule |
