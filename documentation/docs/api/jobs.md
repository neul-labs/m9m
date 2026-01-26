# Jobs API

API endpoints for managing the job queue.

## Overview

Jobs represent workflow executions in the job queue. When you execute a workflow asynchronously, a job is created and processed by worker threads.

## Job Lifecycle

```
┌─────────┐     ┌─────────┐     ┌───────────┐
│ pending │────▶│ running │────▶│ completed │
└─────────┘     └─────────┘     └───────────┘
                     │
                     ▼
                ┌─────────┐
                │ failed  │
                └─────────┘
```

## Job Status

| Status | Description |
|--------|-------------|
| `pending` | Queued, waiting for worker |
| `running` | Being processed by worker |
| `completed` | Successfully finished |
| `failed` | Failed with error |

---

## List Jobs

Retrieve jobs from the queue.

```http
GET /api/v1/jobs
```

### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `status` | string | - | Filter by status |
| `limit` | integer | 100 | Maximum jobs to return |

### Example Request

```bash
curl "http://localhost:8080/api/v1/jobs?status=pending&limit=50" \
  -H "Authorization: Bearer <token>"
```

### Response

```json
{
  "data": [
    {
      "id": "job-123456",
      "workflowId": "550e8400-e29b-41d4-a716-446655440000",
      "workflowName": "Data Sync",
      "status": "pending",
      "priority": 0,
      "retryCount": 0,
      "maxRetries": 3,
      "createdAt": "2024-01-26T16:00:00Z"
    },
    {
      "id": "job-123457",
      "workflowId": "660e8400-e29b-41d4-a716-446655440001",
      "workflowName": "Report Generator",
      "status": "running",
      "priority": 0,
      "retryCount": 0,
      "maxRetries": 3,
      "createdAt": "2024-01-26T15:59:00Z",
      "startedAt": "2024-01-26T16:00:01Z"
    }
  ],
  "total": 2,
  "pendingCount": 1,
  "runningCount": 1
}
```

---

## Get Job

Retrieve detailed job information.

```http
GET /api/v1/jobs/{id}
```

### Example Request

```bash
curl http://localhost:8080/api/v1/jobs/job-123456 \
  -H "Authorization: Bearer <token>"
```

### Response (Pending)

```json
{
  "id": "job-123456",
  "workflowId": "550e8400-e29b-41d4-a716-446655440000",
  "workflowName": "Data Sync",
  "status": "pending",
  "priority": 0,
  "retryCount": 0,
  "maxRetries": 3,
  "inputData": [
    {
      "json": {
        "source": "api"
      }
    }
  ],
  "createdAt": "2024-01-26T16:00:00Z"
}
```

### Response (Running)

```json
{
  "id": "job-123457",
  "workflowId": "660e8400-e29b-41d4-a716-446655440001",
  "workflowName": "Report Generator",
  "status": "running",
  "priority": 0,
  "retryCount": 0,
  "maxRetries": 3,
  "createdAt": "2024-01-26T15:59:00Z",
  "startedAt": "2024-01-26T16:00:01Z",
  "workerId": 2
}
```

### Response (Completed)

```json
{
  "id": "job-123458",
  "workflowId": "770e8400-e29b-41d4-a716-446655440002",
  "workflowName": "Email Campaign",
  "status": "completed",
  "priority": 0,
  "retryCount": 0,
  "maxRetries": 3,
  "createdAt": "2024-01-26T15:50:00Z",
  "startedAt": "2024-01-26T15:50:01Z",
  "completedAt": "2024-01-26T15:50:05Z",
  "duration": 4000,
  "result": {
    "data": [
      {
        "json": {
          "emailsSent": 150
        }
      }
    ]
  }
}
```

### Response (Failed)

```json
{
  "id": "job-123459",
  "workflowId": "880e8400-e29b-41d4-a716-446655440003",
  "workflowName": "Data Import",
  "status": "failed",
  "priority": 0,
  "retryCount": 3,
  "maxRetries": 3,
  "createdAt": "2024-01-26T15:40:00Z",
  "startedAt": "2024-01-26T15:45:00Z",
  "completedAt": "2024-01-26T15:45:02Z",
  "error": "Database connection timeout after 3 retries"
}
```

---

## Queue Statistics

Get job queue statistics.

```http
GET /api/v1/jobs/stats
```

### Example Request

```bash
curl http://localhost:8080/api/v1/jobs/stats \
  -H "Authorization: Bearer <token>"
```

### Response

```json
{
  "pending": 5,
  "running": 2,
  "completed": 1500,
  "failed": 23,
  "workers": 4,
  "queueType": "sqlite",
  "avgProcessingTime": 2340,
  "throughput": {
    "lastMinute": 12,
    "lastHour": 450,
    "lastDay": 8500
  }
}
```

---

## Usage with Async Execution

When you execute a workflow asynchronously:

```bash
# 1. Queue workflow execution
curl -X POST http://localhost:8080/api/v1/workflows/wf-123/execute-async \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"inputData": [{"json": {"key": "value"}}]}'

# Response:
# {
#   "jobId": "job-123456",
#   "status": "pending"
# }

# 2. Poll job status
curl http://localhost:8080/api/v1/jobs/job-123456 \
  -H "Authorization: Bearer <token>"

# 3. When completed, get execution details
curl http://localhost:8080/api/v1/executions/exec-from-job \
  -H "Authorization: Bearer <token>"
```

---

## Job Priority

Jobs can have priority levels (higher = processed first):

| Priority | Use Case |
|----------|----------|
| 0 | Normal (default) |
| 1-5 | High priority |
| -1 to -5 | Low priority |

Set priority when executing async:

```bash
curl -X POST http://localhost:8080/api/v1/workflows/wf-123/execute-async \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "inputData": [{"json": {}}],
    "priority": 5
  }'
```

---

## Retry Behavior

Failed jobs are automatically retried up to `maxRetries` times:

| Retry | Delay |
|-------|-------|
| 1st | Immediate |
| 2nd | 10 seconds |
| 3rd | 60 seconds |

After max retries, job status becomes `failed`.

---

## Error Responses

### 404 Not Found

```json
{
  "error": "Job not found",
  "code": "NOT_FOUND"
}
```

---

## See Also

- [Workflows API](workflows.md) - Execute workflows
- [Executions API](executions.md) - Execution results
- [Queue Configuration](../configuration/queue.md) - Configure job queue
