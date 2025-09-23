package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisQueue is a Redis-based implementation of QueueBackend
type RedisQueue struct {
	client       *redis.Client
	queueKey     string
	processingKey string
	jobsKey      string
}

// NewRedisQueue creates a new Redis queue
func NewRedisQueue(connectionString string) (*RedisQueue, error) {
	opts, err := redis.ParseURL(connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisQueue{
		client:        client,
		queueKey:      "n8n_go:queue:pending",
		processingKey: "n8n_go:queue:processing",
		jobsKey:       "n8n_go:jobs",
	}, nil
}

// Enqueue adds a job to the queue
func (rq *RedisQueue) Enqueue(ctx context.Context, job *Job) error {
	// Serialize job
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	// Store job data
	if err := rq.client.HSet(ctx, rq.jobsKey, job.ID, jobData).Err(); err != nil {
		return fmt.Errorf("failed to store job: %w", err)
	}

	// Add to priority queue
	if err := rq.client.ZAdd(ctx, rq.queueKey, redis.Z{
		Score:  float64(job.Priority),
		Member: job.ID,
	}).Err(); err != nil {
		return fmt.Errorf("failed to enqueue job: %w", err)
	}

	return nil
}

// Dequeue retrieves the next job from the queue
func (rq *RedisQueue) Dequeue(ctx context.Context, workerID string) (*Job, error) {
	// Get highest priority job (ZREVRANGE gets highest scores first)
	result, err := rq.client.ZRevRangeWithScores(ctx, rq.queueKey, 0, 0).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	if len(result) == 0 {
		return nil, nil // No jobs available
	}

	jobID := result[0].Member.(string)

	// Move job from pending to processing queue atomically
	pipe := rq.client.TxPipeline()
	pipe.ZRem(ctx, rq.queueKey, jobID)
	pipe.ZAdd(ctx, rq.processingKey, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: jobID,
	})
	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to move job to processing: %w", err)
	}

	// Get job data
	jobData, err := rq.client.HGet(ctx, rq.jobsKey, jobID).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("job data not found")
		}
		return nil, fmt.Errorf("failed to get job data: %w", err)
	}

	// Deserialize job
	var job Job
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	// Update job status
	job.Status = JobStatusRunning
	job.WorkerID = workerID
	now := time.Now()
	job.StartedAt = &now

	// Update job in Redis
	updatedJobData, err := json.Marshal(&job)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated job: %w", err)
	}

	if err := rq.client.HSet(ctx, rq.jobsKey, job.ID, updatedJobData).Err(); err != nil {
		return nil, fmt.Errorf("failed to update job: %w", err)
	}

	return &job, nil
}

// Acknowledge marks a job as completed
func (rq *RedisQueue) Acknowledge(ctx context.Context, jobID string) error {
	// Remove from processing queue
	if err := rq.client.ZRem(ctx, rq.processingKey, jobID).Err(); err != nil {
		return fmt.Errorf("failed to remove job from processing queue: %w", err)
	}

	return nil
}

// Reject marks a job as failed
func (rq *RedisQueue) Reject(ctx context.Context, jobID string, reason string) error {
	// Remove from processing queue
	if err := rq.client.ZRem(ctx, rq.processingKey, jobID).Err(); err != nil {
		return fmt.Errorf("failed to remove job from processing queue: %w", err)
	}

	// Update job status
	jobData, err := rq.client.HGet(ctx, rq.jobsKey, jobID).Result()
	if err != nil {
		return fmt.Errorf("failed to get job data: %w", err)
	}

	var job Job
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		return fmt.Errorf("failed to unmarshal job: %w", err)
	}

	job.Status = JobStatusFailed
	job.Error = reason
	now := time.Now()
	job.CompletedAt = &now

	updatedJobData, err := json.Marshal(&job)
	if err != nil {
		return fmt.Errorf("failed to marshal updated job: %w", err)
	}

	if err := rq.client.HSet(ctx, rq.jobsKey, job.ID, updatedJobData).Err(); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	return nil
}

// Retry re-queues a job for retry
func (rq *RedisQueue) Retry(ctx context.Context, jobID string) error {
	// Get job data
	jobData, err := rq.client.HGet(ctx, rq.jobsKey, jobID).Result()
	if err != nil {
		return fmt.Errorf("failed to get job data: %w", err)
	}

	var job Job
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		return fmt.Errorf("failed to unmarshal job: %w", err)
	}

	// Update job for retry
	job.Status = JobStatusPending
	job.StartedAt = nil
	job.WorkerID = ""
	job.RetryCount++
	// Reduce priority slightly for retry
	job.Priority = job.Priority - 1

	// Update job in Redis
	updatedJobData, err := json.Marshal(&job)
	if err != nil {
		return fmt.Errorf("failed to marshal updated job: %w", err)
	}

	if err := rq.client.HSet(ctx, rq.jobsKey, job.ID, updatedJobData).Err(); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	// Move job from processing back to pending queue
	pipe := rq.client.TxPipeline()
	pipe.ZRem(ctx, rq.processingKey, jobID)
	pipe.ZAdd(ctx, rq.queueKey, redis.Z{
		Score:  float64(job.Priority),
		Member: jobID,
	})
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to retry job: %w", err)
	}

	return nil
}

// GetJob retrieves a job by ID
func (rq *RedisQueue) GetJob(ctx context.Context, jobID string) (*Job, error) {
	jobData, err := rq.client.HGet(ctx, rq.jobsKey, jobID).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("job not found")
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	var job Job
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

// UpdateJobStatus updates the status of a job
func (rq *RedisQueue) UpdateJobStatus(ctx context.Context, jobID string, status JobStatus) error {
	// Get current job data
	jobData, err := rq.client.HGet(ctx, rq.jobsKey, jobID).Result()
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	var job Job
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		return fmt.Errorf("failed to unmarshal job: %w", err)
	}

	// Update status
	job.Status = status

	// Marshal and store
	updatedJobData, err := json.Marshal(&job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	if err := rq.client.HSet(ctx, rq.jobsKey, jobID, updatedJobData).Err(); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	return nil
}

// ListJobs lists jobs by status
func (rq *RedisQueue) ListJobs(ctx context.Context, status JobStatus, limit int) ([]*Job, error) {
	// Get all job IDs
	jobData, err := rq.client.HGetAll(ctx, rq.jobsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs: %w", err)
	}

	var jobs []*Job
	for _, data := range jobData {
		var job Job
		if err := json.Unmarshal([]byte(data), &job); err != nil {
			continue // Skip invalid jobs
		}

		if job.Status == status {
			jobs = append(jobs, &job)
			if limit > 0 && len(jobs) >= limit {
				break
			}
		}
	}

	return jobs, nil
}

// GetQueueSize returns the current queue size
func (rq *RedisQueue) GetQueueSize(ctx context.Context) (int, error) {
	size, err := rq.client.ZCard(ctx, rq.queueKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get queue size: %w", err)
	}

	return int(size), nil
}

// PurgeQueue removes all jobs from the queue
func (rq *RedisQueue) PurgeQueue(ctx context.Context) error {
	pipe := rq.client.TxPipeline()
	pipe.Del(ctx, rq.queueKey)
	pipe.Del(ctx, rq.processingKey)
	pipe.Del(ctx, rq.jobsKey)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to purge queue: %w", err)
	}

	return nil
}

// Close closes the Redis connection
func (rq *RedisQueue) Close() error {
	return rq.client.Close()
}

// CleanupStaleJobs removes stale jobs from the processing queue
func (rq *RedisQueue) CleanupStaleJobs(ctx context.Context, timeout time.Duration) error {
	// Get jobs that have been processing for too long
	cutoff := time.Now().Add(-timeout).Unix()
	staleJobs, err := rq.client.ZRangeByScore(ctx, rq.processingKey, &redis.ZRangeBy{
		Min: "0",
		Max: strconv.FormatInt(cutoff, 10),
	}).Result()
	if err != nil {
		return fmt.Errorf("failed to get stale jobs: %w", err)
	}

	// Re-queue stale jobs
	for _, jobID := range staleJobs {
		if err := rq.Retry(ctx, jobID); err != nil {
			// Log error but continue with other jobs
			fmt.Printf("Failed to retry stale job %s: %v\n", jobID, err)
		}
	}

	return nil
}

// StartCleanupWorker starts a background worker to clean up stale jobs
func (rq *RedisQueue) StartCleanupWorker(ctx context.Context, interval, timeout time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := rq.CleanupStaleJobs(ctx, timeout); err != nil {
					fmt.Printf("Failed to cleanup stale jobs: %v\n", err)
				}
			}
		}
	}()
}