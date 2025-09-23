package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/yourusername/n8n-go/internal/model"
)

// QueueType represents the type of queue backend
type QueueType string

const (
	QueueTypeMemory QueueType = "memory"
	QueueTypeRedis  QueueType = "redis"
	QueueTypeRabbitMQ QueueType = "rabbitmq"
	QueueTypeKafka  QueueType = "kafka"
)

// JobStatus represents the status of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusRetrying  JobStatus = "retrying"
)

// Job represents a workflow execution job
type Job struct {
	ID           string                 `json:"id"`
	WorkflowID   string                 `json:"workflowId"`
	WorkflowName string                 `json:"workflowName"`
	ExecutionID  string                 `json:"executionId"`
	Priority     int                    `json:"priority"`
	Status       JobStatus              `json:"status"`
	Payload      map[string]interface{} `json:"payload"`
	RetryCount   int                    `json:"retryCount"`
	MaxRetries   int                    `json:"maxRetries"`
	CreatedAt    time.Time              `json:"createdAt"`
	StartedAt    *time.Time             `json:"startedAt,omitempty"`
	CompletedAt  *time.Time             `json:"completedAt,omitempty"`
	Error        string                 `json:"error,omitempty"`
	WorkerID     string                 `json:"workerId,omitempty"`
}

// QueueBackend defines the interface for queue implementations
type QueueBackend interface {
	// Job operations
	Enqueue(ctx context.Context, job *Job) error
	Dequeue(ctx context.Context, workerID string) (*Job, error)
	Acknowledge(ctx context.Context, jobID string) error
	Reject(ctx context.Context, jobID string, reason string) error
	Retry(ctx context.Context, jobID string) error

	// Status operations
	GetJob(ctx context.Context, jobID string) (*Job, error)
	UpdateJobStatus(ctx context.Context, jobID string, status JobStatus) error
	ListJobs(ctx context.Context, status JobStatus, limit int) ([]*Job, error)

	// Queue management
	GetQueueSize(ctx context.Context) (int, error)
	PurgeQueue(ctx context.Context) error
	Close() error
}

// QueueManager manages job queues for horizontal scaling
type QueueManager struct {
	backend       QueueBackend
	workerPool    *WorkerPool
	config        QueueConfig
	mu            sync.RWMutex
	shutdownCh    chan struct{}
	isShutdown    bool
	stats         *QueueStats
}

// QueueConfig holds queue configuration
type QueueConfig struct {
	Type              QueueType
	ConnectionString  string
	MaxWorkers        int
	MaxQueueSize      int
	DefaultPriority   int
	DefaultMaxRetries int
	RetryDelay        time.Duration
	PollInterval      time.Duration
	VisibilityTimeout time.Duration
}

// QueueStats tracks queue statistics
type QueueStats struct {
	mu               sync.RWMutex
	EnqueuedJobs     int64
	DequeuedJobs     int64
	CompletedJobs    int64
	FailedJobs       int64
	RetriedJobs      int64
	CurrentQueueSize int
	ActiveWorkers    int
}

// NewQueueManager creates a new queue manager
func NewQueueManager(config QueueConfig) (*QueueManager, error) {
	var backend QueueBackend
	var err error

	switch config.Type {
	case QueueTypeMemory:
		backend = NewMemoryQueue(config.MaxQueueSize)
	case QueueTypeRedis:
		backend, err = NewRedisQueue(config.ConnectionString)
	case QueueTypeRabbitMQ:
		backend, err = NewRabbitMQQueue(config.ConnectionString)
	default:
		return nil, fmt.Errorf("unsupported queue type: %s", config.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create queue backend: %w", err)
	}

	qm := &QueueManager{
		backend:    backend,
		config:     config,
		shutdownCh: make(chan struct{}),
		stats:      &QueueStats{},
	}

	// Create worker pool
	qm.workerPool = NewWorkerPool(config.MaxWorkers, qm)

	return qm, nil
}

// Start starts the queue manager
func (qm *QueueManager) Start(ctx context.Context) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	if qm.isShutdown {
		return fmt.Errorf("queue manager is shutdown")
	}

	// Start worker pool
	if err := qm.workerPool.Start(ctx); err != nil {
		return fmt.Errorf("failed to start worker pool: %w", err)
	}

	// Start statistics collector
	go qm.collectStats(ctx)

	return nil
}

// Stop stops the queue manager
func (qm *QueueManager) Stop(ctx context.Context) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	if qm.isShutdown {
		return nil
	}

	// Signal shutdown
	close(qm.shutdownCh)
	qm.isShutdown = true

	// Stop worker pool
	if err := qm.workerPool.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop worker pool: %w", err)
	}

	// Close backend
	if err := qm.backend.Close(); err != nil {
		return fmt.Errorf("failed to close queue backend: %w", err)
	}

	return nil
}

// EnqueueWorkflow enqueues a workflow for execution
func (qm *QueueManager) EnqueueWorkflow(ctx context.Context, workflow *model.Workflow, inputData []model.DataItem, options ...JobOption) (*Job, error) {
	job := &Job{
		ID:           generateJobID(),
		WorkflowID:   workflow.ID,
		WorkflowName: workflow.Name,
		ExecutionID:  generateExecutionID(),
		Priority:     qm.config.DefaultPriority,
		Status:       JobStatusPending,
		MaxRetries:   qm.config.DefaultMaxRetries,
		CreatedAt:    time.Now(),
		Payload: map[string]interface{}{
			"workflow":  workflow,
			"inputData": inputData,
		},
	}

	// Apply options
	for _, opt := range options {
		opt(job)
	}

	// Enqueue job
	if err := qm.backend.Enqueue(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to enqueue job: %w", err)
	}

	// Update stats
	qm.stats.mu.Lock()
	qm.stats.EnqueuedJobs++
	qm.stats.mu.Unlock()

	return job, nil
}

// GetJob retrieves a job by ID
func (qm *QueueManager) GetJob(ctx context.Context, jobID string) (*Job, error) {
	return qm.backend.GetJob(ctx, jobID)
}

// ListJobs lists jobs by status
func (qm *QueueManager) ListJobs(ctx context.Context, status JobStatus, limit int) ([]*Job, error) {
	return qm.backend.ListJobs(ctx, status, limit)
}

// GetStats returns queue statistics
func (qm *QueueManager) GetStats() QueueStats {
	qm.stats.mu.RLock()
	defer qm.stats.mu.RUnlock()
	return *qm.stats
}

// collectStats periodically collects queue statistics
func (qm *QueueManager) collectStats(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-qm.shutdownCh:
			return
		case <-ticker.C:
			size, _ := qm.backend.GetQueueSize(ctx)
			qm.stats.mu.Lock()
			qm.stats.CurrentQueueSize = size
			qm.stats.ActiveWorkers = qm.workerPool.ActiveWorkers()
			qm.stats.mu.Unlock()
		}
	}
}

// JobOption is a function that configures a job
type JobOption func(*Job)

// WithPriority sets the job priority
func WithPriority(priority int) JobOption {
	return func(j *Job) {
		j.Priority = priority
	}
}

// WithMaxRetries sets the maximum retries for a job
func WithMaxRetries(maxRetries int) JobOption {
	return func(j *Job) {
		j.MaxRetries = maxRetries
	}
}

// WithExecutionID sets a specific execution ID
func WithExecutionID(executionID string) JobOption {
	return func(j *Job) {
		j.ExecutionID = executionID
	}
}

func generateJobID() string {
	return fmt.Sprintf("job_%d_%d", time.Now().Unix(), time.Now().Nanosecond())
}

func generateExecutionID() string {
	return fmt.Sprintf("exec_%d_%d", time.Now().Unix(), time.Now().Nanosecond())
}