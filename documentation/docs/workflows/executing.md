# Executing Workflows

Learn how to run workflows and manage executions.

## Execution Methods

| Method | Description |
|--------|-------------|
| CLI | Run from command line |
| API | Execute via REST API |
| Webhook | Trigger via HTTP request |
| Schedule | Automatic cron-based |

## CLI Execution

### Run from File

```bash
m9m run workflow.json
```

### Run by ID

```bash
m9m run 550e8400-e29b-41d4-a716-446655440000
```

### Run by Name

```bash
m9m run "Daily Report"
```

### With Input Data

```bash
# Inline JSON
m9m run workflow.json --input '{"name": "John"}'

# From file
m9m run workflow.json --input @input.json
```

### Raw Output

```bash
m9m run workflow.json --raw
```

## API Execution

### Synchronous

Wait for completion:

```bash
curl -X POST http://localhost:8080/api/v1/workflows/{id}/execute \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"inputData": [{"json": {"key": "value"}}]}'
```

Response:

```json
{
  "id": "exec-123",
  "status": "success",
  "data": [{"json": {"result": "..."}}]
}
```

### Asynchronous

Return immediately:

```bash
curl -X POST http://localhost:8080/api/v1/workflows/{id}/execute-async \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"inputData": [{"json": {"key": "value"}}]}'
```

Response:

```json
{
  "jobId": "job-456",
  "status": "pending"
}
```

Then check status:

```bash
curl http://localhost:8080/api/v1/jobs/job-456
```

## Webhook Execution

Configure a webhook node and call it:

```bash
curl -X POST http://localhost:8080/webhook/my-endpoint \
  -H "Content-Type: application/json" \
  -d '{"event": "user_signup", "data": {...}}'
```

## Scheduled Execution

Workflows with Cron nodes run automatically:

```json
{
  "type": "n8n-nodes-base.cron",
  "parameters": {
    "cronExpression": "0 9 * * MON-FRI"
  }
}
```

Manage via API:

```bash
# Create schedule
curl -X POST http://localhost:8080/api/v1/schedules \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "workflowId": "wf-123",
    "cronExpression": "0 9 * * *"
  }'
```

## Execution Status

| Status | Description |
|--------|-------------|
| `pending` | Queued, waiting |
| `running` | Currently executing |
| `success` | Completed successfully |
| `failed` | Completed with error |
| `cancelled` | Manually stopped |

## Monitoring Executions

### List Executions

```bash
m9m execution list
```

Or via API:

```bash
curl http://localhost:8080/api/v1/executions
```

### Get Execution Details

```bash
m9m execution get exec-123
```

### Watch in Real-Time

```bash
m9m execution watch
```

## Handling Failures

### View Error Details

```bash
m9m execution get exec-123
```

Shows:

```
Status: failed
Error:
  Node: HTTP Request
  Message: Connection timeout
```

### Retry Failed Execution

```bash
m9m execution retry exec-123
```

Or via API:

```bash
curl -X POST http://localhost:8080/api/v1/executions/exec-123/retry
```

## Cancel Running Execution

```bash
m9m execution cancel exec-123
```

Or via API:

```bash
curl -X POST http://localhost:8080/api/v1/executions/exec-123/cancel
```

## Input Data

### Structure

```json
{
  "inputData": [
    {
      "json": {
        "field1": "value1",
        "field2": 123
      }
    }
  ]
}
```

### Multiple Items

```json
{
  "inputData": [
    {"json": {"id": 1}},
    {"json": {"id": 2}},
    {"json": {"id": 3}}
  ]
}
```

### Access in Workflow

First node receives input data:

```
{{ $json.field1 }}  // "value1"
```

## Execution Output

### Success Output

```json
{
  "id": "exec-123",
  "status": "success",
  "data": [
    {
      "json": {
        "result": "processed",
        "count": 5
      }
    }
  ],
  "duration": 1234
}
```

### Failed Output

```json
{
  "id": "exec-124",
  "status": "failed",
  "error": {
    "message": "HTTP request failed",
    "node": "Fetch Data"
  }
}
```

## Best Practices

1. **Test with sample data** before production
2. **Use async** for long-running workflows
3. **Monitor failures** and set up alerts
4. **Add timeouts** for external calls
5. **Log important data** for debugging
