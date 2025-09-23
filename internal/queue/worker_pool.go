package queue

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yourusername/n8n-go/internal/engine"
	"github.com/yourusername/n8n-go/internal/model"
)

// Worker represents a workflow execution worker
type Worker struct {
	ID              string
	queueManager    *QueueManager
	workflowEngine  engine.WorkflowEngine
	shutdownCh      chan struct{}
	isRunning       atomic.Bool
	currentJob      atomic.Pointer[Job]
	processedJobs   atomic.Int64
	failedJobs      atomic.Int64
}

// WorkerPool manages a pool of workers
type WorkerPool struct {
	workers      []*Worker
	maxWorkers   int
	queueManager *QueueManager
	mu           sync.RWMutex
	wg           sync.WaitGroup
	shutdownCh   chan struct{}
	isRunning    bool
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(maxWorkers int, queueManager *QueueManager) *WorkerPool {
	return &WorkerPool{
		workers:      make([]*Worker, 0, maxWorkers),
		maxWorkers:   maxWorkers,
		queueManager: queueManager,
		shutdownCh:   make(chan struct{}),
	}
}

// Start starts the worker pool
func (wp *WorkerPool) Start(ctx context.Context) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.isRunning {
		return fmt.Errorf("worker pool already running")
	}

	// Create and start workers
	for i := 0; i < wp.maxWorkers; i++ {
		worker := wp.createWorker(i)
		wp.workers = append(wp.workers, worker)
		wp.wg.Add(1)
		go worker.run(ctx, &wp.wg)
	}

	wp.isRunning = true
	return nil
}

// Stop stops the worker pool
func (wp *WorkerPool) Stop(ctx context.Context) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if !wp.isRunning {
		return nil
	}

	// Signal shutdown to all workers
	close(wp.shutdownCh)

	// Wait for all workers to complete with timeout
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All workers stopped
	case <-ctx.Done():
		return fmt.Errorf("timeout waiting for workers to stop")
	}

	wp.isRunning = false
	wp.workers = nil
	return nil
}

// ActiveWorkers returns the number of active workers
func (wp *WorkerPool) ActiveWorkers() int {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	active := 0
	for _, worker := range wp.workers {
		if worker.isRunning.Load() {
			active++
		}
	}
	return active
}

// GetWorkerStats returns statistics for all workers
func (wp *WorkerPool) GetWorkerStats() []WorkerStats {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	stats := make([]WorkerStats, 0, len(wp.workers))
	for _, worker := range wp.workers {
		stats = append(stats, worker.getStats())
	}
	return stats
}

// ScaleWorkers adjusts the number of workers
func (wp *WorkerPool) ScaleWorkers(ctx context.Context, newWorkerCount int) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if !wp.isRunning {
		return fmt.Errorf("worker pool not running")
	}

	currentCount := len(wp.workers)

	if newWorkerCount > currentCount {
		// Add workers
		for i := currentCount; i < newWorkerCount; i++ {
			worker := wp.createWorker(i)
			wp.workers = append(wp.workers, worker)
			wp.wg.Add(1)
			go worker.run(ctx, &wp.wg)
		}
	} else if newWorkerCount < currentCount {
		// Remove workers
		workersToStop := wp.workers[newWorkerCount:]
		wp.workers = wp.workers[:newWorkerCount]

		for _, worker := range workersToStop {
			worker.stop()
		}
	}

	wp.maxWorkers = newWorkerCount
	return nil
}

func (wp *WorkerPool) createWorker(index int) *Worker {
	return &Worker{
		ID:             fmt.Sprintf("worker_%d", index),
		queueManager:   wp.queueManager,
		workflowEngine: engine.NewWorkflowEngine(),
		shutdownCh:     make(chan struct{}),
	}
}

// Worker methods

func (w *Worker) run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	w.isRunning.Store(true)
	defer w.isRunning.Store(false)

	pollTicker := time.NewTicker(w.queueManager.config.PollInterval)
	defer pollTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.shutdownCh:
			return
		case <-w.queueManager.shutdownCh:
			return
		case <-pollTicker.C:
			w.processNextJob(ctx)
		}
	}
}

func (w *Worker) processNextJob(ctx context.Context) {
	// Dequeue a job
	job, err := w.queueManager.backend.Dequeue(ctx, w.ID)
	if err != nil {
		// No job available or error
		return
	}

	if job == nil {
		return
	}

	// Store current job
	w.currentJob.Store(job)
	defer w.currentJob.Store(nil)

	// Update job status
	now := time.Now()
	job.StartedAt = &now
	job.Status = JobStatusRunning
	job.WorkerID = w.ID
	w.queueManager.backend.UpdateJobStatus(ctx, job.ID, JobStatusRunning)

	// Update stats
	w.queueManager.stats.mu.Lock()
	w.queueManager.stats.DequeuedJobs++
	w.queueManager.stats.mu.Unlock()

	// Execute the workflow
	if err := w.executeWorkflow(ctx, job); err != nil {
		w.handleJobFailure(ctx, job, err)
	} else {
		w.handleJobSuccess(ctx, job)
	}
}

func (w *Worker) executeWorkflow(ctx context.Context, job *Job) error {
	// Extract workflow and input data from payload
	workflowData, ok := job.Payload["workflow"]
	if !ok {
		return fmt.Errorf("workflow not found in job payload")
	}

	inputData, ok := job.Payload["inputData"]
	if !ok {
		return fmt.Errorf("input data not found in job payload")
	}

	// Convert to proper types
	workflow, ok := workflowData.(*model.Workflow)
	if !ok {
		return fmt.Errorf("invalid workflow type in payload")
	}

	input, ok := inputData.([]model.DataItem)
	if !ok {
		return fmt.Errorf("invalid input data type in payload")
	}

	// Execute workflow with timeout
	execCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	result, err := w.workflowEngine.ExecuteWorkflow(workflow, input)
	if err != nil {
		return err
	}

	// Store result in job payload
	job.Payload["result"] = result

	return nil
}

func (w *Worker) handleJobSuccess(ctx context.Context, job *Job) {
	now := time.Now()
	job.CompletedAt = &now
	job.Status = JobStatusCompleted

	// Acknowledge job
	w.queueManager.backend.Acknowledge(ctx, job.ID)
	w.queueManager.backend.UpdateJobStatus(ctx, job.ID, JobStatusCompleted)

	// Update stats
	w.processedJobs.Add(1)
	w.queueManager.stats.mu.Lock()
	w.queueManager.stats.CompletedJobs++
	w.queueManager.stats.mu.Unlock()
}

func (w *Worker) handleJobFailure(ctx context.Context, job *Job, err error) {
	job.Error = err.Error()
	job.RetryCount++

	if job.RetryCount < job.MaxRetries {
		// Retry the job
		job.Status = JobStatusRetrying
		w.queueManager.backend.Retry(ctx, job.ID)
		w.queueManager.backend.UpdateJobStatus(ctx, job.ID, JobStatusRetrying)

		// Update stats
		w.queueManager.stats.mu.Lock()
		w.queueManager.stats.RetriedJobs++
		w.queueManager.stats.mu.Unlock()
	} else {
		// Mark as failed
		now := time.Now()
		job.CompletedAt = &now
		job.Status = JobStatusFailed
		w.queueManager.backend.Reject(ctx, job.ID, err.Error())
		w.queueManager.backend.UpdateJobStatus(ctx, job.ID, JobStatusFailed)

		// Update stats
		w.failedJobs.Add(1)
		w.queueManager.stats.mu.Lock()
		w.queueManager.stats.FailedJobs++
		w.queueManager.stats.mu.Unlock()
	}
}

func (w *Worker) stop() {
	close(w.shutdownCh)
}

func (w *Worker) getStats() WorkerStats {
	var currentJobID string
	if job := w.currentJob.Load(); job != nil {
		currentJobID = job.ID
	}

	return WorkerStats{
		WorkerID:      w.ID,
		IsRunning:     w.isRunning.Load(),
		CurrentJobID:  currentJobID,
		ProcessedJobs: w.processedJobs.Load(),
		FailedJobs:    w.failedJobs.Load(),
	}
}

// WorkerStats represents worker statistics
type WorkerStats struct {
	WorkerID      string `json:"workerId"`
	IsRunning     bool   `json:"isRunning"`
	CurrentJobID  string `json:"currentJobId,omitempty"`
	ProcessedJobs int64  `json:"processedJobs"`
	FailedJobs    int64  `json:"failedJobs"`
}