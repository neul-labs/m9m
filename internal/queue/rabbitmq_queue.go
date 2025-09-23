package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/streadway/amqp"
)

// RabbitMQQueue is a RabbitMQ-based implementation of QueueBackend
type RabbitMQQueue struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	queueName    string
	exchangeName string
	jobsTable    map[string]*Job // In-memory storage for job details
}

// NewRabbitMQQueue creates a new RabbitMQ queue
func NewRabbitMQQueue(connectionString string) (*RabbitMQQueue, error) {
	conn, err := amqp.Dial(connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	rq := &RabbitMQQueue{
		conn:         conn,
		channel:      channel,
		queueName:    "n8n_go_workflow_queue",
		exchangeName: "n8n_go_exchange",
		jobsTable:    make(map[string]*Job),
	}

	// Setup exchange and queue
	if err := rq.setup(); err != nil {
		return nil, fmt.Errorf("failed to setup RabbitMQ: %w", err)
	}

	return rq, nil
}

func (rq *RabbitMQQueue) setup() error {
	// Declare exchange
	err := rq.channel.ExchangeDeclare(
		rq.exchangeName,
		"direct",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare queue
	_, err = rq.channel.QueueDeclare(
		rq.queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	err = rq.channel.QueueBind(
		rq.queueName,
		"workflow", // routing key
		rq.exchangeName,
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	return nil
}

// Enqueue adds a job to the queue
func (rq *RabbitMQQueue) Enqueue(ctx context.Context, job *Job) error {
	// Store job details
	rq.jobsTable[job.ID] = job

	// Create message
	message := map[string]interface{}{
		"jobId":    job.ID,
		"priority": job.Priority,
	}

	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Publish message
	err = rq.channel.Publish(
		rq.exchangeName,
		"workflow", // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // persistent
			Priority:     uint8(job.Priority),
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// Dequeue retrieves the next job from the queue
func (rq *RabbitMQQueue) Dequeue(ctx context.Context, workerID string) (*Job, error) {
	// Get message from queue
	delivery, ok, err := rq.channel.Get(rq.queueName, false) // auto-ack = false
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	if !ok {
		return nil, nil // No messages available
	}

	// Parse message
	var message map[string]interface{}
	if err := json.Unmarshal(delivery.Body, &message); err != nil {
		delivery.Nack(false, true) // requeue
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	jobID, ok := message["jobId"].(string)
	if !ok {
		delivery.Nack(false, false) // don't requeue
		return nil, fmt.Errorf("invalid job ID in message")
	}

	// Get job details
	job, exists := rq.jobsTable[jobID]
	if !exists {
		delivery.Nack(false, false) // don't requeue
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	// Update job status
	job.Status = JobStatusRunning
	job.WorkerID = workerID
	now := time.Now()
	job.StartedAt = &now

	// Store delivery tag for later acknowledgment
	job.Payload["deliveryTag"] = delivery.DeliveryTag

	return job, nil
}

// Acknowledge marks a job as completed
func (rq *RabbitMQQueue) Acknowledge(ctx context.Context, jobID string) error {
	job, exists := rq.jobsTable[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	// Get delivery tag
	deliveryTag, ok := job.Payload["deliveryTag"].(uint64)
	if !ok {
		return fmt.Errorf("delivery tag not found for job: %s", jobID)
	}

	// Acknowledge message
	if err := rq.channel.Ack(deliveryTag, false); err != nil {
		return fmt.Errorf("failed to acknowledge message: %w", err)
	}

	// Update job status
	job.Status = JobStatusCompleted
	now := time.Now()
	job.CompletedAt = &now

	return nil
}

// Reject marks a job as failed
func (rq *RabbitMQQueue) Reject(ctx context.Context, jobID string, reason string) error {
	job, exists := rq.jobsTable[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	// Get delivery tag
	deliveryTag, ok := job.Payload["deliveryTag"].(uint64)
	if !ok {
		return fmt.Errorf("delivery tag not found for job: %s", jobID)
	}

	// Reject message (don't requeue)
	if err := rq.channel.Nack(deliveryTag, false, false); err != nil {
		return fmt.Errorf("failed to reject message: %w", err)
	}

	// Update job status
	job.Status = JobStatusFailed
	job.Error = reason
	now := time.Now()
	job.CompletedAt = &now

	return nil
}

// Retry re-queues a job for retry
func (rq *RabbitMQQueue) Retry(ctx context.Context, jobID string) error {
	job, exists := rq.jobsTable[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	// Get delivery tag
	deliveryTag, ok := job.Payload["deliveryTag"].(uint64)
	if !ok {
		return fmt.Errorf("delivery tag not found for job: %s", jobID)
	}

	// Reject and requeue message
	if err := rq.channel.Nack(deliveryTag, false, true); err != nil {
		return fmt.Errorf("failed to requeue message: %w", err)
	}

	// Update job for retry
	job.Status = JobStatusPending
	job.StartedAt = nil
	job.WorkerID = ""
	job.RetryCount++
	// Remove delivery tag
	delete(job.Payload, "deliveryTag")

	return nil
}

// GetJob retrieves a job by ID
func (rq *RabbitMQQueue) GetJob(ctx context.Context, jobID string) (*Job, error) {
	job, exists := rq.jobsTable[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	return job, nil
}

// UpdateJobStatus updates the status of a job
func (rq *RabbitMQQueue) UpdateJobStatus(ctx context.Context, jobID string, status JobStatus) error {
	job, exists := rq.jobsTable[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	job.Status = status
	return nil
}

// ListJobs lists jobs by status
func (rq *RabbitMQQueue) ListJobs(ctx context.Context, status JobStatus, limit int) ([]*Job, error) {
	var jobs []*Job
	for _, job := range rq.jobsTable {
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
func (rq *RabbitMQQueue) GetQueueSize(ctx context.Context) (int, error) {
	// Get queue info
	queue, err := rq.channel.QueueInspect(rq.queueName)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect queue: %w", err)
	}

	return queue.Messages, nil
}

// PurgeQueue removes all jobs from the queue
func (rq *RabbitMQQueue) PurgeQueue(ctx context.Context) error {
	// Purge queue
	_, err := rq.channel.QueuePurge(rq.queueName, false)
	if err != nil {
		return fmt.Errorf("failed to purge queue: %w", err)
	}

	// Clear job table
	rq.jobsTable = make(map[string]*Job)

	return nil
}

// Close closes the RabbitMQ connection
func (rq *RabbitMQQueue) Close() error {
	if rq.channel != nil {
		rq.channel.Close()
	}
	if rq.conn != nil {
		return rq.conn.Close()
	}
	return nil
}