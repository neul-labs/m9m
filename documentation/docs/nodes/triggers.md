# Trigger Nodes

Trigger nodes start workflow execution based on events, schedules, or manual activation.

## Start Node

Manual trigger for testing and development.

```json
{
  "type": "n8n-nodes-base.start",
  "parameters": {}
}
```

Execute manually via CLI:
```bash
m9m execute workflow.json
```

## Webhook Node

HTTP endpoint that triggers workflows.

### Basic Webhook

```json
{
  "type": "n8n-nodes-base.webhook",
  "parameters": {
    "path": "my-webhook",
    "httpMethod": "POST"
  }
}
```

Webhook URL: `https://your-m9m-host/webhook/my-webhook`

### With Authentication

```json
{
  "parameters": {
    "path": "secure-webhook",
    "httpMethod": "POST",
    "authentication": "headerAuth",
    "headerName": "X-API-Key",
    "headerValue": "={{ $env.WEBHOOK_SECRET }}"
  }
}
```

### Response Configuration

```json
{
  "parameters": {
    "path": "webhook",
    "httpMethod": "POST",
    "responseMode": "responseNode",
    "responseData": "allEntries"
  }
}
```

### HTTP Methods

| Method | Use Case |
|--------|----------|
| GET | Retrieve data, health checks |
| POST | Submit data, most common |
| PUT | Update resources |
| DELETE | Remove resources |

### Query Parameters

Access query parameters:
```javascript
{{ $json.query.paramName }}
```

### Request Body

Access request body:
```javascript
{{ $json.body.fieldName }}
```

### Headers

Access headers:
```javascript
{{ $json.headers['content-type'] }}
```

## Cron Node

Schedule-based triggers.

### Every Minute

```json
{
  "type": "n8n-nodes-base.cron",
  "parameters": {
    "triggerTimes": {
      "item": [
        {"mode": "everyMinute"}
      ]
    }
  }
}
```

### Every Hour

```json
{
  "parameters": {
    "triggerTimes": {
      "item": [
        {"mode": "everyHour", "minute": 0}
      ]
    }
  }
}
```

### Daily at Specific Time

```json
{
  "parameters": {
    "triggerTimes": {
      "item": [
        {
          "mode": "everyDay",
          "hour": 9,
          "minute": 0
        }
      ]
    }
  }
}
```

### Weekly

```json
{
  "parameters": {
    "triggerTimes": {
      "item": [
        {
          "mode": "everyWeek",
          "weekday": "monday",
          "hour": 9,
          "minute": 0
        }
      ]
    }
  }
}
```

### Monthly

```json
{
  "parameters": {
    "triggerTimes": {
      "item": [
        {
          "mode": "everyMonth",
          "dayOfMonth": 1,
          "hour": 0,
          "minute": 0
        }
      ]
    }
  }
}
```

### Cron Expression

For complex schedules:

```json
{
  "parameters": {
    "triggerTimes": {
      "item": [
        {
          "mode": "custom",
          "cronExpression": "0 */15 9-17 * * MON-FRI"
        }
      ]
    }
  }
}
```

Cron format: `second minute hour day-of-month month day-of-week`

### Timezone

```json
{
  "parameters": {
    "triggerTimes": {
      "item": [{"mode": "everyDay", "hour": 9, "minute": 0}]
    },
    "timezone": "America/New_York"
  }
}
```

## Interval Node

Fixed interval triggers.

```json
{
  "type": "n8n-nodes-base.interval",
  "parameters": {
    "interval": 5,
    "unit": "minutes"
  }
}
```

### Units

- `seconds`
- `minutes`
- `hours`

## Error Trigger

Triggers when another workflow fails.

```json
{
  "type": "n8n-nodes-base.errorTrigger",
  "parameters": {}
}
```

Receives error data:
```javascript
{{ $json.workflow.name }}
{{ $json.error.message }}
{{ $json.execution.id }}
```

## Manual Trigger

Explicit trigger for testing.

```json
{
  "type": "n8n-nodes-base.manualTrigger",
  "parameters": {}
}
```

## Service-Specific Triggers

### Slack Event

```json
{
  "type": "n8n-nodes-base.slackTrigger",
  "parameters": {
    "events": ["message.channels", "reaction_added"]
  },
  "credentials": {
    "slackApi": {"id": "1", "name": "Slack"}
  }
}
```

### GitHub Webhook

```json
{
  "type": "n8n-nodes-base.githubTrigger",
  "parameters": {
    "events": ["push", "pull_request"]
  },
  "credentials": {
    "githubApi": {"id": "1", "name": "GitHub"}
  }
}
```

### Email (IMAP)

```json
{
  "type": "n8n-nodes-base.imapTrigger",
  "parameters": {
    "mailbox": "INBOX",
    "action": "read"
  },
  "credentials": {
    "imap": {"id": "1", "name": "Email"}
  }
}
```

## Multiple Triggers

A workflow can have multiple triggers:

```json
{
  "nodes": [
    {
      "id": "cron-trigger",
      "type": "n8n-nodes-base.cron",
      "parameters": {
        "triggerTimes": {"item": [{"mode": "everyHour"}]}
      }
    },
    {
      "id": "webhook-trigger",
      "type": "n8n-nodes-base.webhook",
      "parameters": {
        "path": "manual-run",
        "httpMethod": "POST"
      }
    },
    {
      "id": "process",
      "type": "n8n-nodes-base.set"
    }
  ],
  "connections": {
    "cron-trigger": {
      "main": [[{"node": "process", "type": "main", "index": 0}]]
    },
    "webhook-trigger": {
      "main": [[{"node": "process", "type": "main", "index": 0}]]
    }
  }
}
```

## Trigger Configuration

### Activate/Deactivate

Triggers only run when the workflow is active:

```bash
# Activate workflow
m9m workflow activate wf-001

# Deactivate
m9m workflow deactivate wf-001
```

### Test Triggers

Test without activating:

```bash
m9m execute workflow.json --trigger-data '{"body": {"test": true}}'
```

## Webhook Security

### HMAC Signature

Verify webhook signatures:

```json
{
  "parameters": {
    "path": "secure",
    "httpMethod": "POST",
    "authentication": "hmacSignature",
    "signatureHeader": "X-Hub-Signature-256",
    "algorithm": "sha256"
  }
}
```

### IP Whitelist

Configure in server settings:

```yaml
webhooks:
  allowedIps:
    - "192.168.1.0/24"
    - "10.0.0.1"
```

### Rate Limiting

```yaml
webhooks:
  rateLimit:
    requests: 100
    period: "1m"
```

## Best Practices

1. **Use meaningful webhook paths** - `/orders/new` not `/webhook1`
2. **Secure webhooks** with authentication
3. **Set appropriate cron intervals** - don't poll too frequently
4. **Handle duplicate events** - webhooks may retry
5. **Log trigger events** for debugging

## Next Steps

- [Transform Nodes](transform.md) - Process triggered data
- [HTTP Nodes](http.md) - Make outgoing requests
- [Error Handling](../user-guide/error-handling.md) - Handle trigger failures
