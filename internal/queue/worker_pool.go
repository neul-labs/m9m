package queue

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/neul-labs/m9m/internal/engine"
	"github.com/neul-labs/m9m/internal/model"
)

// WorkerPool processes jobs from a JobQueue using multiple workers
type WorkerPool struct {
	queue      JobQueue
	engine     engine.WorkflowEngine
	numWorkers int
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	started    bool
	mu         sync.Mutex
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(queue JobQueue, eng engine.WorkflowEngine, numWorkers int) *WorkerPool {
	if numWorkers <= 0 {
		numWorkers = 1
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		queue:      queue,
		engine:     eng,
		numWorkers: numWorkers,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start begins processing jobs with the worker pool
func (p *WorkerPool) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return
	}
	p.started = true

	log.Printf("Starting worker pool with %d workers", p.numWorkers)

	for i := 0; i < p.numWorkers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// Stop gracefully stops the worker pool
func (p *WorkerPool) Stop() {
	p.mu.Lock()
	if !p.started {
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()

	log.Println("Stopping worker pool...")
	p.cancel()
	p.wg.Wait()
	log.Println("Worker pool stopped")
}

// worker is a single worker goroutine that processes jobs
func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()

	log.Printf("Worker %d started", id)

	for {
		select {
		case <-p.ctx.Done():
			log.Printf("Worker %d shutting down", id)
			return
		default:
			// Try to get a job with a timeout
			ctx, cancel := context.WithTimeout(p.ctx, 1*time.Second)
			job, err := p.queue.Dequeue(ctx)
			cancel()

			if err != nil {
				if err == context.DeadlineExceeded {
					// No jobs available, loop and try again
					continue
				}
				if p.ctx.Err() != nil {
					// Context cancelled, shutting down
					return
				}
				log.Printf("Worker %d: error dequeuing job: %v", id, err)
				continue
			}

			p.processJob(id, job)
		}
	}
}

// processJob executes a single job
func (p *WorkerPool) processJob(workerID int, job *Job) {
	log.Printf("Worker %d: processing job %s (workflow: %s)", workerID, job.ID, job.WorkflowID)
	startTime := time.Now()

	// Execute the workflow
	result, err := p.engine.ExecuteWorkflow(job.Workflow, job.InputData)

	duration := time.Since(startTime)

	if err != nil {
		log.Printf("Worker %d: job %s failed after %v: %v", workerID, job.ID, duration, err)
		if nackErr := p.queue.Nack(job.ID, err); nackErr != nil {
			log.Printf("Worker %d: failed to nack job %s: %v", workerID, job.ID, nackErr)
		}
		return
	}

	log.Printf("Worker %d: job %s completed in %v", workerID, job.ID, duration)

	// Acknowledge successful completion
	jobResult := &JobResult{
		Data: result.Data,
	}
	if result.Error != nil {
		jobResult.Error = result.Error
	}

	if ackErr := p.queue.Ack(job.ID, jobResult); ackErr != nil {
		log.Printf("Worker %d: failed to ack job %s: %v", workerID, job.ID, ackErr)
	}
}

// GetStats returns worker pool statistics
func (p *WorkerPool) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"workers":       p.numWorkers,
		"pending_jobs":  p.queue.GetPendingCount(),
		"running_jobs":  p.queue.GetRunningCount(),
	}
}

// EnqueueWorkflow is a helper to enqueue a workflow execution
func (p *WorkerPool) EnqueueWorkflow(workflow *model.Workflow, inputData []model.DataItem) (string, error) {
	jobID := fmt.Sprintf("job_%d", time.Now().UnixNano())

	job := &Job{
		ID:         jobID,
		WorkflowID: workflow.ID,
		Workflow:   workflow,
		InputData:  inputData,
		Priority:   0,
		MaxRetries: 3,
		CreatedAt:  time.Now(),
	}

	if err := p.queue.Enqueue(job); err != nil {
		return "", err
	}

	return jobID, nil
}
