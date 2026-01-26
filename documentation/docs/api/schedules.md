# Schedules API

API endpoints for managing scheduled workflow executions.

## Overview

Schedules allow workflows to run automatically based on cron expressions.

---

## List Schedules

Retrieve all schedules.

```http
GET /api/v1/schedules
```

### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `workflowId` | string | - | Filter by workflow |
| `enabled` | boolean | - | Filter by enabled status |

### Example Request

```bash
curl http://localhost:8080/api/v1/schedules \
  -H "Authorization: Bearer <token>"
```

### Response

```json
{
  "data": [
    {
      "id": "sched-123",
      "workflowId": "550e8400-e29b-41d4-a716-446655440000",
      "workflowName": "Daily Report",
      "cronExpression": "0 9 * * MON-FRI",
      "timezone": "America/New_York",
      "enabled": true,
      "nextRun": "2024-01-27T09:00:00-05:00",
      "lastRun": "2024-01-26T09:00:00-05:00",
      "lastStatus": "success",
      "createdAt": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 5
}
```

---

## Get Schedule

Retrieve a single schedule.

```http
GET /api/v1/schedules/{id}
```

### Example Request

```bash
curl http://localhost:8080/api/v1/schedules/sched-123 \
  -H "Authorization: Bearer <token>"
```

### Response

```json
{
  "id": "sched-123",
  "workflowId": "550e8400-e29b-41d4-a716-446655440000",
  "workflowName": "Daily Report",
  "cronExpression": "0 9 * * MON-FRI",
  "timezone": "America/New_York",
  "enabled": true,
  "inputData": [
    {
      "json": {
        "reportType": "daily"
      }
    }
  ],
  "nextRun": "2024-01-27T09:00:00-05:00",
  "lastRun": "2024-01-26T09:00:00-05:00",
  "lastStatus": "success",
  "runCount": 25,
  "successCount": 24,
  "failureCount": 1,
  "createdAt": "2024-01-01T00:00:00Z",
  "updatedAt": "2024-01-26T09:00:00Z"
}
```

---

## Create Schedule

Create a new schedule.

```http
POST /api/v1/schedules
```

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `workflowId` | string | Yes | Workflow to schedule |
| `cronExpression` | string | Yes | Cron expression |
| `timezone` | string | No | IANA timezone (default: UTC) |
| `enabled` | boolean | No | Enable immediately (default: true) |
| `inputData` | array | No | Input data for each run |

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

### Example Request

```bash
curl -X POST http://localhost:8080/api/v1/schedules \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "workflowId": "550e8400-e29b-41d4-a716-446655440000",
    "cronExpression": "0 9 * * MON-FRI",
    "timezone": "America/New_York",
    "enabled": true,
    "inputData": [{"json": {"reportType": "daily"}}]
  }'
```

### Response

```json
{
  "id": "sched-456",
  "workflowId": "550e8400-e29b-41d4-a716-446655440000",
  "cronExpression": "0 9 * * MON-FRI",
  "timezone": "America/New_York",
  "enabled": true,
  "nextRun": "2024-01-27T09:00:00-05:00",
  "createdAt": "2024-01-26T16:00:00Z"
}
```

---

## Update Schedule

Update an existing schedule.

```http
PUT /api/v1/schedules/{id}
```

### Example Request

```bash
curl -X PUT http://localhost:8080/api/v1/schedules/sched-123 \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "cronExpression": "0 8 * * MON-FRI",
    "timezone": "Europe/London"
  }'
```

---

## Delete Schedule

Delete a schedule.

```http
DELETE /api/v1/schedules/{id}
```

### Example Request

```bash
curl -X DELETE http://localhost:8080/api/v1/schedules/sched-123 \
  -H "Authorization: Bearer <token>"
```

### Response

```
204 No Content
```

---

## Enable Schedule

Enable a disabled schedule.

```http
POST /api/v1/schedules/{id}/enable
```

### Example Request

```bash
curl -X POST http://localhost:8080/api/v1/schedules/sched-123/enable \
  -H "Authorization: Bearer <token>"
```

### Response

```json
{
  "id": "sched-123",
  "enabled": true,
  "nextRun": "2024-01-27T09:00:00-05:00"
}
```

---

## Disable Schedule

Disable a schedule.

```http
POST /api/v1/schedules/{id}/disable
```

### Example Request

```bash
curl -X POST http://localhost:8080/api/v1/schedules/sched-123/disable \
  -H "Authorization: Bearer <token>"
```

### Response

```json
{
  "id": "sched-123",
  "enabled": false,
  "nextRun": null
}
```

---

## Get Schedule History

Get execution history for a schedule.

```http
GET /api/v1/schedules/{id}/history
```

### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | integer | 20 | Maximum entries |

### Example Request

```bash
curl "http://localhost:8080/api/v1/schedules/sched-123/history?limit=10" \
  -H "Authorization: Bearer <token>"
```

### Response

```json
{
  "data": [
    {
      "executionId": "exec-100",
      "scheduledTime": "2024-01-26T09:00:00-05:00",
      "startedAt": "2024-01-26T09:00:00.500-05:00",
      "finishedAt": "2024-01-26T09:00:02.500-05:00",
      "status": "success",
      "duration": 2000
    },
    {
      "executionId": "exec-99",
      "scheduledTime": "2024-01-25T09:00:00-05:00",
      "startedAt": "2024-01-25T09:00:00.500-05:00",
      "finishedAt": "2024-01-25T09:00:01.000-05:00",
      "status": "failed",
      "duration": 500,
      "error": "HTTP request timeout"
    }
  ],
  "total": 25
}
```

---

## Common Cron Examples

| Expression | Description |
|------------|-------------|
| `* * * * *` | Every minute |
| `*/5 * * * *` | Every 5 minutes |
| `0 * * * *` | Every hour |
| `0 9 * * *` | Daily at 9 AM |
| `0 9 * * MON-FRI` | Weekdays at 9 AM |
| `0 0 * * *` | Daily at midnight |
| `0 0 1 * *` | First of each month |
| `0 0 * * 0` | Every Sunday |

---

## Error Responses

### 400 Bad Request

```json
{
  "error": "Invalid cron expression",
  "code": "VALIDATION_ERROR",
  "details": {
    "cronExpression": "Invalid format"
  }
}
```

### 404 Not Found

```json
{
  "error": "Schedule not found",
  "code": "NOT_FOUND"
}
```

### 409 Conflict

```json
{
  "error": "Schedule already exists for this workflow",
  "code": "CONFLICT"
}
```

---

## See Also

- [Scheduling Guide](../scheduling/index.md) - Scheduling concepts
- [Cron Expressions](../scheduling/cron.md) - Cron syntax reference
