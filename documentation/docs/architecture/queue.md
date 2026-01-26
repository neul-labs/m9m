# Job Queue System

Understanding the job queue architecture.

## Overview

The job queue manages asynchronous workflow execution:

```
API Request → Job Created → Queue → Worker → Execute → Result
```

## Queue Backends

| Backend | Use Case | Persistence | Distribution |
|---------|----------|-------------|--------------|
| Memory | Development | No | Single instance |
| SQLite | Single server | Yes | Single instance |
| Redis | Production | Yes | Multi-instance |

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                    Producers                        │
│  (API handlers, webhooks, schedulers)               │
└──────────────────────┬──────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────┐
│                   Job Queue                         │
│  ┌─────────────────────────────────────────────┐   │
│  │  Pending  │  Running  │  Completed │  Failed │   │
│  └─────────────────────────────────────────────┘   │
└──────────────────────┬──────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────┐
│                  Worker Pool                        │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐           │
│  │ Worker 1 │ │ Worker 2 │ │ Worker N │           │
│  └──────────┘ └──────────┘ └──────────┘           │
└─────────────────────────────────────────────────────┘
```

## Job Structure

```go
type Job struct {
    ID          string                 // Unique identifier
    WorkflowID  string                 // Workflow to execute
    InputData   []DataItem             // Input data
    Status      JobStatus              // pending, running, completed, failed
    Priority    int                    // Higher = sooner
    CreatedAt   time.Time
    StartedAt   *time.Time
    CompletedAt *time.Time
    Result      *ExecutionResult
    Error       string
    RetryCount  int
    MaxRetries  int
}
```

## Job Lifecycle

```
     Created
        │
        ▼
     Pending ←─────┐
        │          │
        ▼          │
     Running ──────┤ (retry)
        │          │
   ┌────┴────┐     │
   ▼         ▼     │
Completed  Failed ─┘
```

## Queue Interface

```go
type JobQueue interface {
    // Add job to queue
    Enqueue(ctx context.Context, job *Job) error

    // Get next job for processing
    Dequeue(ctx context.Context) (*Job, error)

    // Update job status
    UpdateStatus(ctx context.Context, jobID string, status JobStatus) error

    // Get job by ID
    Get(ctx context.Context, jobID string) (*Job, error)

    // List jobs with filters
    List(ctx context.Context, filters JobFilters) ([]*Job, error)

    // Cancel a pending job
    Cancel(ctx context.Context, jobID string) error
}
```

## Memory Queue

In-memory queue for development:

```go
type MemoryQueue struct {
    jobs     map[string]*Job
    pending  chan *Job
    mu       sync.RWMutex
}
```

### Configuration

```yaml
queue:
  type: memory
  size: 1000  # Max pending jobs
```

### Characteristics

- Fast, no I/O overhead
- Jobs lost on restart
- Single instance only

## SQLite Queue

Persistent queue using SQLite:

```sql
CREATE TABLE jobs (
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL,
    input_data TEXT,
    status TEXT DEFAULT 'pending',
    priority INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    result TEXT,
    error TEXT,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3
);

CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_priority ON jobs(priority DESC, created_at ASC);
```

### Configuration

```yaml
queue:
  type: sqlite
  path: /data/queue.db
```

### Characteristics

- Persistent across restarts
- Single instance
- Good for moderate workloads

## Redis Queue

Distributed queue using Redis:

```
┌─────────────────────────────────────┐
│              Redis                  │
│  ┌─────────────────────────────┐   │
│  │ m9m:jobs:pending (sorted set)│   │
│  └─────────────────────────────┘   │
│  ┌─────────────────────────────┐   │
│  │ m9m:jobs:running (set)      │   │
│  └─────────────────────────────┘   │
│  ┌─────────────────────────────┐   │
│  │ m9m:job:{id} (hash)         │   │
│  └─────────────────────────────┘   │
└─────────────────────────────────────┘
```

### Configuration

```yaml
queue:
  type: redis
  url: "redis://localhost:6379"
  prefix: "m9m"
  database: 0
```

### Characteristics

- Distributed across instances
- Persistent (with AOF/RDB)
- High throughput
- Atomic operations

## Worker Pool

Workers process jobs from the queue:

```go
type WorkerPool struct {
    queue    JobQueue
    engine   WorkflowEngine
    workers  int
    wg       sync.WaitGroup
    ctx      context.Context
    cancel   context.CancelFunc
}

func (p *WorkerPool) Start() {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go p.worker(i)
    }
}

func (p *WorkerPool) worker(id int) {
    defer p.wg.Done()

    for {
        select {
        case <-p.ctx.Done():
            return
        default:
            job, err := p.queue.Dequeue(p.ctx)
            if err != nil {
                time.Sleep(100 * time.Millisecond)
                continue
            }
            p.processJob(job)
        }
    }
}
```

### Configuration

```yaml
queue:
  workers: 5           # Number of workers
  pollInterval: 100ms  # Queue polling interval
```

## Job Priority

Higher priority jobs execute first:

```go
// Priority levels
const (
    PriorityLow    = 0
    PriorityNormal = 5
    PriorityHigh   = 10
    PriorityUrgent = 20
)

// Enqueue with priority
queue.Enqueue(ctx, &Job{
    WorkflowID: "wf-123",
    Priority:   PriorityHigh,
})
```

## Retry Mechanism

Failed jobs are automatically retried:

```go
func (p *WorkerPool) processJob(job *Job) {
    result, err := p.engine.Execute(ctx, job.Workflow, job.InputData)

    if err != nil && job.RetryCount < job.MaxRetries {
        job.RetryCount++
        job.Status = JobStatusPending
        p.queue.Enqueue(ctx, job)
        return
    }

    if err != nil {
        job.Status = JobStatusFailed
        job.Error = err.Error()
    } else {
        job.Status = JobStatusCompleted
        job.Result = result
    }

    p.queue.Update(ctx, job)
}
```

### Retry Configuration

```yaml
queue:
  maxRetries: 3
  retryDelay: 5s      # Initial delay
  retryBackoff: 2.0   # Exponential backoff multiplier
```

### Retry Delays

| Attempt | Delay |
|---------|-------|
| 1 | 5s |
| 2 | 10s |
| 3 | 20s |

## Monitoring

### Queue Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `queue_size` | Gauge | Pending jobs |
| `queue_processing` | Gauge | Running jobs |
| `queue_processed` | Counter | Completed jobs |
| `queue_failed` | Counter | Failed jobs |
| `queue_wait_time` | Histogram | Time in queue |
| `queue_process_time` | Histogram | Processing time |

### Health Check

```go
func (q *Queue) Health() HealthStatus {
    return HealthStatus{
        Healthy:   q.isConnected(),
        Pending:   q.pendingCount(),
        Running:   q.runningCount(),
        Workers:   q.activeWorkers(),
    }
}
```

## Dead Letter Queue

Jobs that exceed max retries go to DLQ:

```yaml
queue:
  deadLetterQueue:
    enabled: true
    retention: 7d  # Keep for 7 days
```

### Reviewing Failed Jobs

```bash
# List failed jobs
m9m job list --status failed

# Retry a failed job
m9m job retry job-123

# Delete failed job
m9m job delete job-123
```

## Best Practices

### Scaling Workers

```yaml
# Development
queue:
  workers: 2

# Production
queue:
  workers: 10
```

### Queue Sizing

Monitor queue depth and adjust:

- If queue grows consistently → Add workers
- If workers often idle → Reduce workers
- If jobs timeout → Increase timeout or optimize workflows

### High Availability

For production:

1. Use Redis queue
2. Run multiple m9m instances
3. Configure Redis Sentinel/Cluster
4. Monitor queue health

### Job Timeout

Prevent jobs from running forever:

```yaml
queue:
  jobTimeout: 5m  # Cancel after 5 minutes
```
