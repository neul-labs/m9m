package queue

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryJobQueue implements JobQueue with in-memory storage
// Jobs are lost on restart - use SQLiteJobQueue for persistence
type MemoryJobQueue struct {
	jobs      chan *Job
	pending   map[string]*Job
	running   map[string]*Job
	completed map[string]*Job
	mu        sync.RWMutex
	closed    bool
	closeCh   chan struct{}
}

// NewMemoryJobQueue creates a new in-memory job queue
func NewMemoryJobQueue(bufferSize int) *MemoryJobQueue {
	if bufferSize <= 0 {
		bufferSize = 1000
	}

	return &MemoryJobQueue{
		jobs:      make(chan *Job, bufferSize),
		pending:   make(map[string]*Job),
		running:   make(map[string]*Job),
		completed: make(map[string]*Job),
		closeCh:   make(chan struct{}),
	}
}

// Enqueue adds a job to the queue
func (q *MemoryJobQueue) Enqueue(job *Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return fmt.Errorf("queue is closed")
	}

	if job.ID == "" {
		return fmt.Errorf("job ID is required")
	}

	if _, exists := q.pending[job.ID]; exists {
		return fmt.Errorf("job %s already exists", job.ID)
	}

	job.Status = JobStatusPending
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	if job.MaxRetries == 0 {
		job.MaxRetries = 3
	}

	q.pending[job.ID] = job

	select {
	case q.jobs <- job:
		return nil
	default:
		delete(q.pending, job.ID)
		return fmt.Errorf("queue is full")
	}
}

// Dequeue retrieves the next job from the queue
func (q *MemoryJobQueue) Dequeue(ctx context.Context) (*Job, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-q.closeCh:
		return nil, fmt.Errorf("queue is closed")
	case job := <-q.jobs:
		q.mu.Lock()
		defer q.mu.Unlock()

		// Move from pending to running
		delete(q.pending, job.ID)
		job.Status = JobStatusRunning
		now := time.Now()
		job.StartedAt = &now
		q.running[job.ID] = job

		return job, nil
	}
}

// Ack marks a job as completed
func (q *MemoryJobQueue) Ack(jobID string, result *JobResult) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	job, exists := q.running[jobID]
	if !exists {
		return fmt.Errorf("job %s not found in running jobs", jobID)
	}

	delete(q.running, jobID)
	job.Status = JobStatusCompleted
	now := time.Now()
	job.CompletedAt = &now

	if result != nil {
		job.ResultData = result.Data
		if result.Error != nil {
			job.Error = result.Error.Error()
		}
	}

	q.completed[jobID] = job
	return nil
}

// Nack marks a job as failed, optionally requeueing
func (q *MemoryJobQueue) Nack(jobID string, err error) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	job, exists := q.running[jobID]
	if !exists {
		return fmt.Errorf("job %s not found in running jobs", jobID)
	}

	delete(q.running, jobID)
	job.RetryCount++

	if err != nil {
		job.Error = err.Error()
	}

	// Retry if under limit
	if job.RetryCount < job.MaxRetries {
		job.Status = JobStatusPending
		job.StartedAt = nil
		q.pending[jobID] = job

		select {
		case q.jobs <- job:
			return nil
		default:
			// Queue full, mark as failed
			job.Status = JobStatusFailed
			now := time.Now()
			job.CompletedAt = &now
			q.completed[jobID] = job
			delete(q.pending, jobID)
			return fmt.Errorf("queue full, cannot retry job %s", jobID)
		}
	}

	// Max retries exceeded
	job.Status = JobStatusFailed
	now := time.Now()
	job.CompletedAt = &now
	q.completed[jobID] = job
	return nil
}

// GetJob retrieves a job by ID
func (q *MemoryJobQueue) GetJob(jobID string) (*Job, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if job, exists := q.pending[jobID]; exists {
		return job, nil
	}
	if job, exists := q.running[jobID]; exists {
		return job, nil
	}
	if job, exists := q.completed[jobID]; exists {
		return job, nil
	}

	return nil, fmt.Errorf("job %s not found", jobID)
}

// GetPendingCount returns the number of pending jobs
func (q *MemoryJobQueue) GetPendingCount() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.pending)
}

// GetRunningCount returns the number of running jobs
func (q *MemoryJobQueue) GetRunningCount() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.running)
}

// ListJobs returns jobs matching the given status
func (q *MemoryJobQueue) ListJobs(status *JobStatus, limit int) ([]*Job, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var jobs []*Job

	addJobs := func(m map[string]*Job, s JobStatus) {
		if status != nil && *status != s {
			return
		}
		for _, job := range m {
			if limit > 0 && len(jobs) >= limit {
				return
			}
			jobs = append(jobs, job)
		}
	}

	addJobs(q.pending, JobStatusPending)
	addJobs(q.running, JobStatusRunning)
	addJobs(q.completed, JobStatusCompleted)

	return jobs, nil
}

// Close gracefully shuts down the queue
func (q *MemoryJobQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return nil
	}

	q.closed = true
	close(q.closeCh)
	return nil
}
