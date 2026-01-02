# Error Handling

Learn how to handle errors gracefully in your m9m workflows.

## Error Types

### Node Errors

Errors that occur during node execution:

- HTTP request failures
- Database connection errors
- API authentication failures
- Data validation errors

### Expression Errors

Errors in expression evaluation:

- Undefined variables
- Type mismatches
- Syntax errors

### Workflow Errors

System-level errors:

- Timeout exceeded
- Memory limits
- Queue failures

## Node-Level Error Handling

### Continue on Fail

Allow workflow to continue even if a node fails:

```json
{
  "id": "http-request",
  "type": "n8n-nodes-base.httpRequest",
  "continueOnFail": true,
  "parameters": {
    "url": "https://api.example.com/data"
  }
}
```

When enabled, failed nodes output error data instead of stopping:

```json
{
  "error": {
    "message": "Request failed with status code 500",
    "name": "HTTPError",
    "statusCode": 500
  }
}
```

### Error Output

Some nodes have dedicated error outputs:

```json
{
  "connections": {
    "try-request": {
      "main": [
        [{"node": "process-success", "type": "main", "index": 0}],
        [{"node": "handle-error", "type": "main", "index": 0}]
      ]
    }
  }
}
```

## Retry Mechanism

### Automatic Retry

Configure retry behavior for transient failures:

```json
{
  "id": "http-request",
  "type": "n8n-nodes-base.httpRequest",
  "retryOnFail": true,
  "maxRetries": 3,
  "retryInterval": 1000,
  "parameters": {
    "url": "https://api.example.com/data"
  }
}
```

### Exponential Backoff

```json
{
  "retryOnFail": true,
  "maxRetries": 5,
  "retryStrategy": "exponential",
  "retryBaseInterval": 1000,
  "retryMaxInterval": 30000
}
```

Retry intervals: 1s, 2s, 4s, 8s, 16s (capped at 30s)

### Retry Conditions

Only retry on specific errors:

```json
{
  "retryOnFail": true,
  "retryConditions": {
    "statusCodes": [429, 500, 502, 503, 504],
    "errorTypes": ["ETIMEDOUT", "ECONNREFUSED"]
  }
}
```

## Error Workflows

### Define Error Workflow

Create a dedicated error-handling workflow:

```json
{
  "name": "Error Handler",
  "nodes": [
    {
      "id": "error-trigger",
      "type": "n8n-nodes-base.errorTrigger",
      "position": [250, 300]
    },
    {
      "id": "notify",
      "type": "n8n-nodes-base.slack",
      "position": [450, 300],
      "parameters": {
        "channel": "#alerts",
        "text": "Workflow failed: {{ $json.workflow.name }}"
      }
    }
  ]
}
```

### Link Error Workflow

Associate with your workflow:

```json
{
  "name": "Main Workflow",
  "settings": {
    "errorWorkflow": "error-handler-workflow-id"
  }
}
```

### Error Data Structure

Error workflows receive:

```json
{
  "workflow": {
    "id": "wf-001",
    "name": "Data Sync"
  },
  "execution": {
    "id": "exec-123",
    "url": "https://m9m.example.com/execution/exec-123"
  },
  "error": {
    "message": "Connection refused",
    "name": "Error",
    "stack": "Error: Connection refused\n    at..."
  },
  "node": {
    "id": "database-query",
    "name": "Query Database",
    "type": "n8n-nodes-base.postgres"
  },
  "timestamp": "2024-01-15T10:30:00.000Z"
}
```

## Try/Catch Pattern

Use the IF node to catch and handle errors:

```json
{
  "nodes": [
    {
      "id": "api-call",
      "type": "n8n-nodes-base.httpRequest",
      "continueOnFail": true
    },
    {
      "id": "check-error",
      "type": "n8n-nodes-base.if",
      "parameters": {
        "conditions": {
          "boolean": [
            {
              "value1": "={{ $json.error !== undefined }}",
              "value2": true
            }
          ]
        }
      }
    },
    {
      "id": "handle-success",
      "type": "n8n-nodes-base.set"
    },
    {
      "id": "handle-error",
      "type": "n8n-nodes-base.set"
    }
  ]
}
```

## Code Node Error Handling

### JavaScript

```javascript
try {
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
  }
  return { data: await response.json() };
} catch (error) {
  // Log error
  console.error('Request failed:', error.message);

  // Return error data
  return {
    error: true,
    message: error.message,
    timestamp: new Date().toISOString()
  };
}
```

### Python

```python
try:
    response = requests.get(url)
    response.raise_for_status()
    return {"data": response.json()}
except requests.RequestException as e:
    return {
        "error": True,
        "message": str(e),
        "timestamp": datetime.now().isoformat()
    }
```

## Validation Errors

### Input Validation

Validate data before processing:

```javascript
// In Set node or Code node
const email = $json.email;

if (!email || !email.includes('@')) {
  throw new Error('Invalid email format');
}

if ($json.amount < 0) {
  throw new Error('Amount cannot be negative');
}

return $json;
```

### Schema Validation

Use JSON Schema validation:

```javascript
const Ajv = require('ajv');
const ajv = new Ajv();

const schema = {
  type: 'object',
  required: ['name', 'email'],
  properties: {
    name: { type: 'string', minLength: 1 },
    email: { type: 'string', format: 'email' }
  }
};

const validate = ajv.compile(schema);
if (!validate($json)) {
  throw new Error(JSON.stringify(validate.errors));
}
```

## Timeout Handling

### Node Timeout

Set timeout for individual nodes:

```json
{
  "id": "http-request",
  "type": "n8n-nodes-base.httpRequest",
  "parameters": {
    "url": "https://api.example.com/slow-endpoint",
    "timeout": 30000
  }
}
```

### Workflow Timeout

Set maximum execution time:

```json
{
  "settings": {
    "executionTimeout": 300
  }
}
```

## Logging and Debugging

### Enable Debug Logging

```bash
M9M_LOG_LEVEL=debug m9m serve
```

### Log in Code Nodes

```javascript
console.log('Processing item:', $json.id);
console.error('Validation failed:', error);
console.warn('Using default value for missing field');
```

### Execution History

View error details in execution history:

```bash
m9m executions get exec-123 --include-error-details
```

## Alerting

### Slack Notification

```json
{
  "id": "alert",
  "type": "n8n-nodes-base.slack",
  "parameters": {
    "channel": "#alerts",
    "text": "Workflow Error",
    "attachments": [
      {
        "color": "danger",
        "fields": [
          {"title": "Workflow", "value": "={{ $json.workflow.name }}"},
          {"title": "Error", "value": "={{ $json.error.message }}"},
          {"title": "Node", "value": "={{ $json.node.name }}"}
        ]
      }
    ]
  }
}
```

### Email Alert

```json
{
  "id": "email-alert",
  "type": "n8n-nodes-base.emailSend",
  "parameters": {
    "to": "ops@example.com",
    "subject": "Workflow Error: {{ $json.workflow.name }}",
    "text": "Error: {{ $json.error.message }}\n\nExecution: {{ $json.execution.url }}"
  }
}
```

## Best Practices

1. **Always use `continueOnFail`** for non-critical operations
2. **Implement retry logic** for external API calls
3. **Set up error workflows** for critical workflows
4. **Log meaningful error context**
5. **Validate inputs early** in the workflow
6. **Set appropriate timeouts**
7. **Monitor error rates** via metrics

## Next Steps

- [Workflows](workflows.md) - Workflow management
- [Expressions](expressions.md) - Dynamic data handling
- [Configuration](../reference/configuration.md) - Logging configuration
