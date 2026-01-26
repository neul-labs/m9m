# Scheduling Overview

Automate workflow execution with time-based triggers.

## Scheduling Methods

| Method | Description | Use Case |
|--------|-------------|----------|
| Cron node | In-workflow trigger | Workflow-specific schedules |
| API schedules | External schedule management | Dynamic scheduling |
| CLI schedules | Command-line scheduling | Development/testing |

## Cron Node

Add a Cron node as the workflow trigger:

```json
{
  "nodes": [
    {
      "id": "trigger",
      "name": "Run Daily",
      "type": "n8n-nodes-base.cron",
      "position": [250, 300],
      "parameters": {
        "cronExpression": "0 9 * * *"
      }
    }
  ]
}
```

When the workflow is activated, it runs on schedule.

## Schedule Management

### Create Schedule via API

```bash
curl -X POST http://localhost:8080/api/v1/schedules \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "workflowId": "wf-123",
    "cronExpression": "0 9 * * *",
    "enabled": true,
    "timezone": "America/New_York"
  }'
```

### List Schedules

```bash
curl http://localhost:8080/api/v1/schedules \
  -H "Authorization: Bearer <token>"
```

### Update Schedule

```bash
curl -X PUT http://localhost:8080/api/v1/schedules/sched-123 \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "cronExpression": "0 10 * * *",
    "enabled": true
  }'
```

### Delete Schedule

```bash
curl -X DELETE http://localhost:8080/api/v1/schedules/sched-123 \
  -H "Authorization: Bearer <token>"
```

## Schedule Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `workflowId` | string | Yes | Workflow to execute |
| `cronExpression` | string | Yes | Cron expression |
| `enabled` | boolean | No | Enable/disable (default: true) |
| `timezone` | string | No | Timezone (default: UTC) |
| `inputData` | array | No | Input data for execution |

## Common Schedules

| Schedule | Cron Expression | Description |
|----------|-----------------|-------------|
| Every minute | `* * * * *` | Run every minute |
| Hourly | `0 * * * *` | Run at top of hour |
| Daily at 9 AM | `0 9 * * *` | Run at 9:00 AM |
| Weekly Monday | `0 9 * * 1` | Monday at 9:00 AM |
| Monthly 1st | `0 0 1 * *` | 1st of month at midnight |
| Quarterly | `0 0 1 */3 *` | Every 3 months |

## Timezone Support

Specify timezone for schedule:

```json
{
  "cronExpression": "0 9 * * *",
  "timezone": "America/New_York"
}
```

Common timezones:

| Timezone | Offset |
|----------|--------|
| UTC | +00:00 |
| America/New_York | -05:00 |
| America/Los_Angeles | -08:00 |
| Europe/London | +00:00 |
| Europe/Berlin | +01:00 |
| Asia/Tokyo | +09:00 |

## Execution Behavior

### Missed Executions

If the server is down during a scheduled time:

- **Default**: Skip missed executions
- **Catch up**: Run missed executions on startup

Configure catch-up:

```yaml
scheduling:
  catchUpMissed: true
  maxCatchUpExecutions: 10
```

### Overlapping Executions

If a scheduled run starts while previous is running:

| Policy | Behavior |
|--------|----------|
| `skip` | Skip new execution (default) |
| `queue` | Queue for later |
| `parallel` | Run in parallel |

Configure:

```yaml
scheduling:
  overlapPolicy: skip
```

## Monitoring Schedules

### View Schedule Status

```bash
curl http://localhost:8080/api/v1/schedules/sched-123 \
  -H "Authorization: Bearer <token>"
```

Response:

```json
{
  "id": "sched-123",
  "workflowId": "wf-456",
  "cronExpression": "0 9 * * *",
  "enabled": true,
  "timezone": "UTC",
  "lastRun": "2024-01-15T09:00:00Z",
  "nextRun": "2024-01-16T09:00:00Z",
  "lastStatus": "success"
}
```

### View Schedule History

```bash
curl http://localhost:8080/api/v1/schedules/sched-123/history \
  -H "Authorization: Bearer <token>"
```

## Best Practices

### 1. Use Descriptive Names

```json
{
  "name": "Daily Report - 9 AM EST"
}
```

### 2. Set Appropriate Intervals

- Don't schedule more frequently than needed
- Consider rate limits of external APIs
- Allow time for execution to complete

### 3. Handle Failures

- Set up alerting for failed scheduled runs
- Implement retry logic in workflows
- Log execution details for debugging

### 4. Test Schedules

```bash
# Test that cron expression is valid
m9m schedule validate "0 9 * * *"

# Preview next 5 runs
m9m schedule preview "0 9 * * *" --count 5
```

### 5. Monitor Resource Usage

- Stagger schedules to avoid load spikes
- Monitor execution times
- Set up alerts for long-running jobs

## Troubleshooting

### Schedule Not Running

1. Check if workflow is activated
2. Verify cron expression is valid
3. Check schedule is enabled
4. Review server logs

### Incorrect Timing

1. Verify timezone setting
2. Check system clock
3. Confirm cron expression syntax

### Too Many Executions

1. Review overlap policy
2. Check cron expression (e.g., `*` vs `0`)
3. Look for duplicate schedules

## Next Steps

- [Cron Expressions](cron.md) - Detailed cron syntax
- [Workflows](../workflows/index.md) - Creating workflows
- [API Reference](../api/schedules.md) - Schedule API
