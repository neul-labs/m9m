package queue

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// MemoryQueue is an in-memory implementation of QueueBackend
type MemoryQueue struct {
	mu          sync.RWMutex
	jobs        map[string]*Job
	pendingJobs []*Job
	runningJobs map[string]*Job
	maxSize     int
}

// NewMemoryQueue creates a new in-memory queue
func NewMemoryQueue(maxSize int) *MemoryQueue {
	return &MemoryQueue{
		jobs:        make(map[string]*Job),
		pendingJobs: make([]*Job, 0),
		runningJobs: make(map[string]*Job),
		maxSize:     maxSize,
	}
}

// Enqueue adds a job to the queue
func (mq *MemoryQueue) Enqueue(ctx context.Context, job *Job) error {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if len(mq.jobs) >= mq.maxSize {
		return fmt.Errorf("queue is full")
	}

	mq.jobs[job.ID] = job
	mq.pendingJobs = append(mq.pendingJobs, job)

	// Sort by priority (higher priority first)
	sort.Slice(mq.pendingJobs, func(i, j int) bool {
		return mq.pendingJobs[i].Priority > mq.pendingJobs[j].Priority
	})

	return nil
}

// Dequeue retrieves the next job from the queue
func (mq *MemoryQueue) Dequeue(ctx context.Context, workerID string) (*Job, error) {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if len(mq.pendingJobs) == 0 {
		return nil, nil
	}

	// Get the highest priority job
	job := mq.pendingJobs[0]
	mq.pendingJobs = mq.pendingJobs[1:]

	// Mark as running
	job.Status = JobStatusRunning
	job.WorkerID = workerID
	now := time.Now()
	job.StartedAt = &now
	mq.runningJobs[job.ID] = job

	return job, nil
}

// Acknowledge marks a job as completed
func (mq *MemoryQueue) Acknowledge(ctx context.Context, jobID string) error {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	job, exists := mq.runningJobs[jobID]
	if !exists {
		return fmt.Errorf("job not found in running jobs")
	}

	delete(mq.runningJobs, jobID)
	job.Status = JobStatusCompleted
	now := time.Now()
	job.CompletedAt = &now

	return nil
}

// Reject marks a job as failed
func (mq *MemoryQueue) Reject(ctx context.Context, jobID string, reason string) error {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	job, exists := mq.runningJobs[jobID]
	if !exists {
		return fmt.Errorf("job not found in running jobs")
	}

	delete(mq.runningJobs, jobID)
	job.Status = JobStatusFailed
	job.Error = reason
	now := time.Now()
	job.CompletedAt = &now

	return nil
}

// Retry re-queues a job for retry
func (mq *MemoryQueue) Retry(ctx context.Context, jobID string) error {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	job, exists := mq.runningJobs[jobID]
	if !exists {
		return fmt.Errorf("job not found in running jobs")
	}

	delete(mq.runningJobs, jobID)
	job.Status = JobStatusPending
	job.StartedAt = nil
	job.WorkerID = ""

	// Add back to pending queue with slightly lower priority
	job.Priority = job.Priority - 1
	mq.pendingJobs = append(mq.pendingJobs, job)

	// Re-sort
	sort.Slice(mq.pendingJobs, func(i, j int) bool {
		return mq.pendingJobs[i].Priority > mq.pendingJobs[j].Priority
	})

	return nil
}

// GetJob retrieves a job by ID
func (mq *MemoryQueue) GetJob(ctx context.Context, jobID string) (*Job, error) {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	job, exists := mq.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found")
	}

	return job, nil
}

// UpdateJobStatus updates the status of a job
func (mq *MemoryQueue) UpdateJobStatus(ctx context.Context, jobID string, status JobStatus) error {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	job, exists := mq.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found")
	}

	job.Status = status
	return nil
}

// ListJobs lists jobs by status
func (mq *MemoryQueue) ListJobs(ctx context.Context, status JobStatus, limit int) ([]*Job, error) {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	var jobs []*Job
	for _, job := range mq.jobs {
		if job.Status == status {
			jobs = append(jobs, job)
			if limit > 0 && len(jobs) >= limit {
				break
			}
		}
	}

	return jobs, nil
}

// GetQueueSize returns the current queue size
func (mq *MemoryQueue) GetQueueSize(ctx context.Context) (int, error) {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	return len(mq.pendingJobs), nil
}

// PurgeQueue removes all jobs from the queue
func (mq *MemoryQueue) PurgeQueue(ctx context.Context) error {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	mq.jobs = make(map[string]*Job)
	mq.pendingJobs = make([]*Job, 0)
	mq.runningJobs = make(map[string]*Job)

	return nil
}

// Close closes the queue
func (mq *MemoryQueue) Close() error {
	// Nothing to close for memory queue
	return nil
}