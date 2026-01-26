# Queue Configuration

Configure the job queue for workflow execution.

## Overview

The job queue manages asynchronous workflow executions. When you execute a workflow asynchronously, a job is created, queued, and processed by worker threads.

## Supported Queue Types

| Type | Persistence | Use Case |
|------|-------------|----------|
| `memory` | No | Development, testing |
| `sqlite` | Yes | Production (single-node) |

## SQLite Queue (Default)

Persistent queue with SQLite storage:

```yaml
queue:
  type: "sqlite"
  max_workers: 4
  sqlite:
    path: "./data/queue.db"
```

| Setting | Default | Description |
|---------|---------|-------------|
| `sqlite.path` | `~/.m9m/data/queue.db` | Queue database path |

### Benefits

- Jobs survive server restarts
- No external dependencies
- Recovery of failed jobs

### CLI Override

```bash
m9m serve --queue sqlite --queue-db /data/queue.db
```

## Memory Queue

Fast in-memory queue (jobs lost on restart):

```yaml
queue:
  type: "memory"
  max_workers: 4
```

### Benefits

- Fastest performance
- No disk I/O

### Limitations

- Jobs lost on restart
- Not suitable for production

### CLI Override

```bash
m9m serve --queue memory
```

## Worker Configuration

```yaml
queue:
  max_workers: 10
```

| Setting | Default | Description |
|---------|---------|-------------|
| `max_workers` | `4` | Concurrent workflow executions |

### CLI Override

```bash
m9m serve --workers 8
```

### Worker Sizing

| Scenario | Workers |
|----------|---------|
| Development | 2-4 |
| Light production | 4-8 |
| Heavy production | 8-16 |
| CPU-bound workflows | Match CPU cores |
| I/O-bound workflows | 2-4x CPU cores |

## Job Retry Configuration

```yaml
execution:
  retry:
    enabled: true
    max_attempts: 3
    backoff: "exponential"
    initial_interval: "1s"
    max_interval: "5m"
```

| Setting | Default | Description |
|---------|---------|-------------|
| `enabled` | `true` | Enable automatic retry |
| `max_attempts` | `3` | Maximum retry attempts |
| `backoff` | `exponential` | Backoff strategy |
| `initial_interval` | `1s` | First retry delay |
| `max_interval` | `5m` | Maximum retry delay |

### Backoff Strategies

| Strategy | Behavior |
|----------|----------|
| `linear` | Constant delay between retries |
| `exponential` | Doubling delay (1s, 2s, 4s, ...) |

## Environment Variables

```bash
export M9M_QUEUE_TYPE=sqlite
export M9M_QUEUE_PATH=./data/queue.db
export M9M_MAX_WORKERS=8
```

## Job Lifecycle

```
Enqueue → Pending → Running → Completed
                      ↓
                    Failed → Retry → Pending (loop)
                      ↓
                    Failed (max retries)
```

## Monitoring

### Queue Stats API

```bash
curl http://localhost:8080/api/v1/jobs/stats
```

Response:

```json
{
  "pending": 5,
  "running": 2,
  "completed": 1500,
  "failed": 23,
  "workers": 4,
  "queueType": "sqlite"
}
```

### Prometheus Metrics

```
# HELP m9m_queue_pending_jobs Number of pending jobs
m9m_queue_pending_jobs 5

# HELP m9m_queue_running_jobs Number of running jobs
m9m_queue_running_jobs 2

# HELP m9m_queue_workers Number of active workers
m9m_queue_workers 4
```

## Recovery

### SQLite Queue Recovery

On server restart, the SQLite queue:

1. Loads pending jobs from database
2. Requeues jobs that were running (may have been interrupted)
3. Resumes processing

### Manual Recovery

View stuck jobs:

```bash
curl "http://localhost:8080/api/v1/jobs?status=running"
```

## Best Practices

### Production Setup

```yaml
queue:
  type: "sqlite"
  max_workers: 8
  sqlite:
    path: "/data/queue.db"

execution:
  retry:
    enabled: true
    max_attempts: 3
    backoff: "exponential"
```

### High Throughput

```yaml
queue:
  type: "sqlite"
  max_workers: 16
```

### Resource Constrained

```yaml
queue:
  type: "sqlite"
  max_workers: 2
```

## Troubleshooting

### Jobs Not Processing

Check workers are running:

```bash
curl http://localhost:8080/api/v1/jobs/stats
```

If `workers: 0`, restart the server.

### Jobs Stuck in Pending

Check for errors in logs:

```bash
tail -f ~/.m9m/logs/m9m.log | grep -i queue
```

### Queue Database Corruption

Recreate queue database:

```bash
# Stop server
rm ~/.m9m/data/queue.db
# Start server (recreates database)
m9m serve
```

Note: This loses pending jobs.
