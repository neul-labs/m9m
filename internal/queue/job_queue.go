/*
Package queue provides job queue implementations for workflow execution.

This file defines the JobQueue interface and Job struct for local workflow
execution with optional persistence. Unlike the NNG-based WorkQueue (for
distributed workers), JobQueue is for single-node execution with persistence.
*/
package queue

import (
	"context"
	"time"

	"github.com/neul-labs/m9m/internal/model"
)

// JobStatus represents the status of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

// Job represents a workflow execution job
type Job struct {
	ID          string           `json:"id"`
	WorkflowID  string           `json:"workflow_id"`
	Workflow    *model.Workflow  `json:"workflow"`
	InputData   []model.DataItem `json:"input_data"`
	Priority    int              `json:"priority"` // Higher = more urgent
	Status      JobStatus        `json:"status"`
	RetryCount  int              `json:"retry_count"`
	MaxRetries  int              `json:"max_retries"`
	ResultData  []model.DataItem `json:"result_data,omitempty"`
	Error       string           `json:"error,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	StartedAt   *time.Time       `json:"started_at,omitempty"`
	CompletedAt *time.Time       `json:"completed_at,omitempty"`
}

// JobResult contains the result of a job execution
type JobResult struct {
	Data  []model.DataItem
	Error error
}

// JobQueue defines the interface for job queue implementations
type JobQueue interface {
	// Enqueue adds a job to the queue
	Enqueue(job *Job) error

	// Dequeue retrieves and removes the next job from the queue
	// Blocks until a job is available or context is cancelled
	Dequeue(ctx context.Context) (*Job, error)

	// Ack marks a job as successfully completed
	Ack(jobID string, result *JobResult) error

	// Nack marks a job as failed, optionally requeueing for retry
	Nack(jobID string, err error) error

	// GetJob retrieves a job by ID
	GetJob(jobID string) (*Job, error)

	// GetPendingCount returns the number of pending jobs
	GetPendingCount() int

	// GetRunningCount returns the number of running jobs
	GetRunningCount() int

	// ListJobs returns jobs matching the given status (nil for all)
	ListJobs(status *JobStatus, limit int) ([]*Job, error)

	// Close gracefully shuts down the queue
	Close() error
}
