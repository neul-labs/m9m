package queue

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestJob(id string) *Job {
	return &Job{
		ID:         id,
		WorkflowID: "wf-test-123",
		Workflow: &model.Workflow{
			ID:   "wf-test-123",
			Name: "Test Workflow",
		},
		InputData: []model.DataItem{
			{JSON: map[string]interface{}{"test": "data"}},
		},
		Priority:   0,
		Status:     JobStatusPending,
		MaxRetries: 3,
		CreatedAt:  time.Now(),
	}
}

// Test Memory Queue
func TestMemoryQueue_EnqueueDequeue(t *testing.T) {
	q := NewMemoryJobQueue(10)
	defer q.Close()

	job := createTestJob("job-1")
	err := q.Enqueue(job)
	require.NoError(t, err)

	assert.Equal(t, 1, q.GetPendingCount())
	assert.Equal(t, 0, q.GetRunningCount())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	dequeued, err := q.Dequeue(ctx)
	require.NoError(t, err)
	assert.Equal(t, job.ID, dequeued.ID)
	assert.Equal(t, JobStatusRunning, dequeued.Status)

	assert.Equal(t, 0, q.GetPendingCount())
	assert.Equal(t, 1, q.GetRunningCount())
}

func TestMemoryQueue_AckNack(t *testing.T) {
	q := NewMemoryJobQueue(10)
	defer q.Close()

	job := createTestJob("job-1")
	err := q.Enqueue(job)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	dequeued, err := q.Dequeue(ctx)
	require.NoError(t, err)

	// Test Ack
	result := &JobResult{
		Data: []model.DataItem{{JSON: map[string]interface{}{"result": "success"}}},
	}
	err = q.Ack(dequeued.ID, result)
	require.NoError(t, err)

	retrieved, err := q.GetJob(dequeued.ID)
	require.NoError(t, err)
	assert.Equal(t, JobStatusCompleted, retrieved.Status)
	assert.Equal(t, 0, q.GetRunningCount())
}

func TestMemoryQueue_Nack(t *testing.T) {
	q := NewMemoryJobQueue(10)
	defer q.Close()

	job := createTestJob("job-1")
	err := q.Enqueue(job)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	dequeued, err := q.Dequeue(ctx)
	require.NoError(t, err)

	// Test Nack
	err = q.Nack(dequeued.ID, assert.AnError)
	require.NoError(t, err)

	// Job should be requeued since retry count < max retries
	assert.Equal(t, 1, q.GetPendingCount())
}

func TestMemoryQueue_ListJobs(t *testing.T) {
	q := NewMemoryJobQueue(10)
	defer q.Close()

	for i := 0; i < 5; i++ {
		job := createTestJob("job-" + string(rune('a'+i)))
		err := q.Enqueue(job)
		require.NoError(t, err)
	}

	status := JobStatusPending
	jobs, err := q.ListJobs(&status, 10)
	require.NoError(t, err)
	assert.Len(t, jobs, 5)
}

func TestMemoryQueue_DequeueTimeout(t *testing.T) {
	q := NewMemoryJobQueue(10)
	defer q.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := q.Dequeue(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

// Test SQLite Queue
func TestSQLiteQueue_EnqueueDequeue(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_queue.db")

	q, err := NewSQLiteJobQueue(dbPath, 10)
	require.NoError(t, err)
	defer q.Close()

	job := createTestJob("job-sqlite-1")
	err = q.Enqueue(job)
	require.NoError(t, err)

	assert.Equal(t, 1, q.GetPendingCount())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	dequeued, err := q.Dequeue(ctx)
	require.NoError(t, err)
	assert.Equal(t, job.ID, dequeued.ID)
	assert.Equal(t, JobStatusRunning, dequeued.Status)
}

func TestSQLiteQueue_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_queue.db")

	// Create queue and add jobs
	q1, err := NewSQLiteJobQueue(dbPath, 10)
	require.NoError(t, err)

	job := createTestJob("job-persist-1")
	err = q1.Enqueue(job)
	require.NoError(t, err)
	q1.Close()

	// Reopen queue and verify job is recovered
	q2, err := NewSQLiteJobQueue(dbPath, 10)
	require.NoError(t, err)
	defer q2.Close()

	assert.Equal(t, 1, q2.GetPendingCount())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	dequeued, err := q2.Dequeue(ctx)
	require.NoError(t, err)
	assert.Equal(t, job.ID, dequeued.ID)
}

func TestSQLiteQueue_AckNack(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_queue.db")

	q, err := NewSQLiteJobQueue(dbPath, 10)
	require.NoError(t, err)
	defer q.Close()

	job := createTestJob("job-ack-1")
	err = q.Enqueue(job)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	dequeued, err := q.Dequeue(ctx)
	require.NoError(t, err)

	result := &JobResult{
		Data: []model.DataItem{{JSON: map[string]interface{}{"result": "success"}}},
	}
	err = q.Ack(dequeued.ID, result)
	require.NoError(t, err)

	retrieved, err := q.GetJob(dequeued.ID)
	require.NoError(t, err)
	assert.Equal(t, JobStatusCompleted, retrieved.Status)
}

func TestSQLiteQueue_ListJobs(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_queue.db")

	q, err := NewSQLiteJobQueue(dbPath, 10)
	require.NoError(t, err)
	defer q.Close()

	for i := 0; i < 5; i++ {
		job := createTestJob("job-list-" + string(rune('a'+i)))
		err := q.Enqueue(job)
		require.NoError(t, err)
	}

	status := JobStatusPending
	jobs, err := q.ListJobs(&status, 10)
	require.NoError(t, err)
	assert.Len(t, jobs, 5)
}

func TestSQLiteQueue_RecoverRunningJobs(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_queue.db")

	// Create queue, enqueue and dequeue a job
	q1, err := NewSQLiteJobQueue(dbPath, 10)
	require.NoError(t, err)

	job := createTestJob("job-running-1")
	err = q1.Enqueue(job)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = q1.Dequeue(ctx)
	require.NoError(t, err)

	assert.Equal(t, 0, q1.GetPendingCount())
	assert.Equal(t, 1, q1.GetRunningCount())

	// Close without ack (simulating crash)
	q1.Close()

	// Reopen - running jobs should be recovered as pending
	q2, err := NewSQLiteJobQueue(dbPath, 10)
	require.NoError(t, err)
	defer q2.Close()

	assert.Equal(t, 1, q2.GetPendingCount())
}

// Cleanup test files
func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}
