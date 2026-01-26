package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/neul-labs/m9m/internal/model"
	_ "github.com/mattn/go-sqlite3"
)

// SQLiteJobQueue implements JobQueue with SQLite persistence
// Jobs are kept in memory for fast access and persisted to SQLite for durability
type SQLiteJobQueue struct {
	db        *sql.DB
	jobs      chan *Job          // In-memory channel for fast dequeue
	pending   map[string]*Job    // Track pending jobs in memory
	running   map[string]*Job    // Track running jobs in memory
	mu        sync.RWMutex
	closed    bool
	closeCh   chan struct{}
	dbPath    string
}

// NewSQLiteJobQueue creates a new SQLite-backed job queue
func NewSQLiteJobQueue(dbPath string, bufferSize int) (*SQLiteJobQueue, error) {
	if bufferSize <= 0 {
		bufferSize = 1000
	}

	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	q := &SQLiteJobQueue{
		db:      db,
		jobs:    make(chan *Job, bufferSize),
		pending: make(map[string]*Job),
		running: make(map[string]*Job),
		closeCh: make(chan struct{}),
		dbPath:  dbPath,
	}

	if err := q.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	if err := q.recoverJobs(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to recover jobs: %w", err)
	}

	return q, nil
}

// initSchema creates the jobs table if it doesn't exist
func (q *SQLiteJobQueue) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS jobs (
		id TEXT PRIMARY KEY,
		workflow_id TEXT NOT NULL,
		workflow_json TEXT NOT NULL,
		input_data_json TEXT,
		priority INTEGER DEFAULT 0,
		status TEXT DEFAULT 'pending',
		retry_count INTEGER DEFAULT 0,
		max_retries INTEGER DEFAULT 3,
		result_data_json TEXT,
		error TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		started_at DATETIME,
		completed_at DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
	CREATE INDEX IF NOT EXISTS idx_jobs_created ON jobs(created_at);
	CREATE INDEX IF NOT EXISTS idx_jobs_priority ON jobs(priority DESC, created_at ASC);
	`

	_, err := q.db.Exec(schema)
	return err
}

// recoverJobs loads pending and running jobs from SQLite on startup
func (q *SQLiteJobQueue) recoverJobs() error {
	// Reset any running jobs back to pending (crashed during execution)
	_, err := q.db.Exec(`UPDATE jobs SET status = 'pending', started_at = NULL WHERE status = 'running'`)
	if err != nil {
		return fmt.Errorf("failed to reset running jobs: %w", err)
	}

	// Load pending jobs ordered by priority (high first) then creation time
	rows, err := q.db.Query(`
		SELECT id, workflow_id, workflow_json, input_data_json, priority,
		       status, retry_count, max_retries, created_at
		FROM jobs
		WHERE status = 'pending'
		ORDER BY priority DESC, created_at ASC
	`)
	if err != nil {
		return fmt.Errorf("failed to query pending jobs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		job, err := q.scanJob(rows)
		if err != nil {
			return fmt.Errorf("failed to scan job: %w", err)
		}

		q.pending[job.ID] = job
		select {
		case q.jobs <- job:
		default:
			// Channel full, job will remain in pending map
		}
	}

	return rows.Err()
}

// scanJob scans a job from a database row
func (q *SQLiteJobQueue) scanJob(rows *sql.Rows) (*Job, error) {
	var (
		id            string
		workflowID    string
		workflowJSON  string
		inputDataJSON sql.NullString
		priority      int
		status        string
		retryCount    int
		maxRetries    int
		createdAt     time.Time
	)

	err := rows.Scan(&id, &workflowID, &workflowJSON, &inputDataJSON, &priority,
		&status, &retryCount, &maxRetries, &createdAt)
	if err != nil {
		return nil, err
	}

	var workflow model.Workflow
	if err := json.Unmarshal([]byte(workflowJSON), &workflow); err != nil {
		return nil, fmt.Errorf("failed to unmarshal workflow: %w", err)
	}

	var inputData []model.DataItem
	if inputDataJSON.Valid && inputDataJSON.String != "" {
		if err := json.Unmarshal([]byte(inputDataJSON.String), &inputData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal input data: %w", err)
		}
	}

	return &Job{
		ID:         id,
		WorkflowID: workflowID,
		Workflow:   &workflow,
		InputData:  inputData,
		Priority:   priority,
		Status:     JobStatus(status),
		RetryCount: retryCount,
		MaxRetries: maxRetries,
		CreatedAt:  createdAt,
	}, nil
}

// Enqueue adds a job to the queue
func (q *SQLiteJobQueue) Enqueue(job *Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return fmt.Errorf("queue is closed")
	}

	if job.ID == "" {
		return fmt.Errorf("job ID is required")
	}

	job.Status = JobStatusPending
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	if job.MaxRetries == 0 {
		job.MaxRetries = 3
	}

	// Persist to SQLite first
	workflowJSON, err := json.Marshal(job.Workflow)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}

	var inputDataJSON []byte
	if len(job.InputData) > 0 {
		inputDataJSON, err = json.Marshal(job.InputData)
		if err != nil {
			return fmt.Errorf("failed to marshal input data: %w", err)
		}
	}

	_, err = q.db.Exec(`
		INSERT INTO jobs (id, workflow_id, workflow_json, input_data_json, priority,
		                  status, retry_count, max_retries, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, job.ID, job.WorkflowID, string(workflowJSON), string(inputDataJSON),
		job.Priority, string(job.Status), job.RetryCount, job.MaxRetries, job.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert job: %w", err)
	}

	// Add to in-memory structures
	q.pending[job.ID] = job

	select {
	case q.jobs <- job:
		return nil
	default:
		// Channel full but job is persisted, it will be picked up on recovery
		return nil
	}
}

// Dequeue retrieves the next job from the queue
func (q *SQLiteJobQueue) Dequeue(ctx context.Context) (*Job, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-q.closeCh:
		return nil, fmt.Errorf("queue is closed")
	case job := <-q.jobs:
		q.mu.Lock()
		defer q.mu.Unlock()

		// Update status in SQLite
		now := time.Now()
		_, err := q.db.Exec(`UPDATE jobs SET status = 'running', started_at = ? WHERE id = ?`,
			now, job.ID)
		if err != nil {
			// Put job back in channel
			select {
			case q.jobs <- job:
			default:
			}
			return nil, fmt.Errorf("failed to update job status: %w", err)
		}

		// Move from pending to running in memory
		delete(q.pending, job.ID)
		job.Status = JobStatusRunning
		job.StartedAt = &now
		q.running[job.ID] = job

		return job, nil
	}
}

// Ack marks a job as completed
func (q *SQLiteJobQueue) Ack(jobID string, result *JobResult) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	job, exists := q.running[jobID]
	if !exists {
		return fmt.Errorf("job %s not found in running jobs", jobID)
	}

	now := time.Now()
	var resultDataJSON []byte
	var errStr string

	if result != nil {
		if len(result.Data) > 0 {
			var err error
			resultDataJSON, err = json.Marshal(result.Data)
			if err != nil {
				return fmt.Errorf("failed to marshal result data: %w", err)
			}
		}
		if result.Error != nil {
			errStr = result.Error.Error()
		}
	}

	_, err := q.db.Exec(`
		UPDATE jobs SET status = 'completed', completed_at = ?, result_data_json = ?, error = ?
		WHERE id = ?
	`, now, string(resultDataJSON), errStr, jobID)
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	delete(q.running, jobID)
	job.Status = JobStatusCompleted
	job.CompletedAt = &now
	if result != nil {
		job.ResultData = result.Data
		if result.Error != nil {
			job.Error = result.Error.Error()
		}
	}

	return nil
}

// Nack marks a job as failed, optionally requeueing
func (q *SQLiteJobQueue) Nack(jobID string, err error) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	job, exists := q.running[jobID]
	if !exists {
		return fmt.Errorf("job %s not found in running jobs", jobID)
	}

	delete(q.running, jobID)
	job.RetryCount++

	var errStr string
	if err != nil {
		errStr = err.Error()
		job.Error = errStr
	}

	// Retry if under limit
	if job.RetryCount < job.MaxRetries {
		_, dbErr := q.db.Exec(`
			UPDATE jobs SET status = 'pending', started_at = NULL, retry_count = ?, error = ?
			WHERE id = ?
		`, job.RetryCount, errStr, jobID)
		if dbErr != nil {
			return fmt.Errorf("failed to update job for retry: %w", dbErr)
		}

		job.Status = JobStatusPending
		job.StartedAt = nil
		q.pending[jobID] = job

		select {
		case q.jobs <- job:
		default:
			// Channel full, job is persisted and will be picked up on recovery
		}
		return nil
	}

	// Max retries exceeded
	now := time.Now()
	_, dbErr := q.db.Exec(`
		UPDATE jobs SET status = 'failed', completed_at = ?, retry_count = ?, error = ?
		WHERE id = ?
	`, now, job.RetryCount, errStr, jobID)
	if dbErr != nil {
		return fmt.Errorf("failed to update failed job: %w", dbErr)
	}

	job.Status = JobStatusFailed
	job.CompletedAt = &now
	return nil
}

// GetJob retrieves a job by ID
func (q *SQLiteJobQueue) GetJob(jobID string) (*Job, error) {
	q.mu.RLock()
	if job, exists := q.pending[jobID]; exists {
		q.mu.RUnlock()
		return job, nil
	}
	if job, exists := q.running[jobID]; exists {
		q.mu.RUnlock()
		return job, nil
	}
	q.mu.RUnlock()

	// Query from database for completed/failed jobs
	row := q.db.QueryRow(`
		SELECT id, workflow_id, workflow_json, input_data_json, priority,
		       status, retry_count, max_retries, result_data_json, error,
		       created_at, started_at, completed_at
		FROM jobs WHERE id = ?
	`, jobID)

	var (
		id             string
		workflowID     string
		workflowJSON   string
		inputDataJSON  sql.NullString
		priority       int
		status         string
		retryCount     int
		maxRetries     int
		resultDataJSON sql.NullString
		errStr         sql.NullString
		createdAt      time.Time
		startedAt      sql.NullTime
		completedAt    sql.NullTime
	)

	err := row.Scan(&id, &workflowID, &workflowJSON, &inputDataJSON, &priority,
		&status, &retryCount, &maxRetries, &resultDataJSON, &errStr,
		&createdAt, &startedAt, &completedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job %s not found", jobID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query job: %w", err)
	}

	var workflow model.Workflow
	if err := json.Unmarshal([]byte(workflowJSON), &workflow); err != nil {
		return nil, fmt.Errorf("failed to unmarshal workflow: %w", err)
	}

	job := &Job{
		ID:         id,
		WorkflowID: workflowID,
		Workflow:   &workflow,
		Priority:   priority,
		Status:     JobStatus(status),
		RetryCount: retryCount,
		MaxRetries: maxRetries,
		CreatedAt:  createdAt,
	}

	if inputDataJSON.Valid {
		json.Unmarshal([]byte(inputDataJSON.String), &job.InputData)
	}
	if resultDataJSON.Valid {
		json.Unmarshal([]byte(resultDataJSON.String), &job.ResultData)
	}
	if errStr.Valid {
		job.Error = errStr.String
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}

	return job, nil
}

// GetPendingCount returns the number of pending jobs
func (q *SQLiteJobQueue) GetPendingCount() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.pending)
}

// GetRunningCount returns the number of running jobs
func (q *SQLiteJobQueue) GetRunningCount() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.running)
}

// ListJobs returns jobs matching the given status
func (q *SQLiteJobQueue) ListJobs(status *JobStatus, limit int) ([]*Job, error) {
	if limit <= 0 {
		limit = 100
	}

	var query string
	var args []interface{}

	if status != nil {
		query = `
			SELECT id, workflow_id, workflow_json, input_data_json, priority,
			       status, retry_count, max_retries, result_data_json, error,
			       created_at, started_at, completed_at
			FROM jobs WHERE status = ?
			ORDER BY created_at DESC
			LIMIT ?
		`
		args = []interface{}{string(*status), limit}
	} else {
		query = `
			SELECT id, workflow_id, workflow_json, input_data_json, priority,
			       status, retry_count, max_retries, result_data_json, error,
			       created_at, started_at, completed_at
			FROM jobs
			ORDER BY created_at DESC
			LIMIT ?
		`
		args = []interface{}{limit}
	}

	rows, err := q.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		var (
			id             string
			workflowID     string
			workflowJSON   string
			inputDataJSON  sql.NullString
			priority       int
			statusStr      string
			retryCount     int
			maxRetries     int
			resultDataJSON sql.NullString
			errStr         sql.NullString
			createdAt      time.Time
			startedAt      sql.NullTime
			completedAt    sql.NullTime
		)

		err := rows.Scan(&id, &workflowID, &workflowJSON, &inputDataJSON, &priority,
			&statusStr, &retryCount, &maxRetries, &resultDataJSON, &errStr,
			&createdAt, &startedAt, &completedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}

		var workflow model.Workflow
		json.Unmarshal([]byte(workflowJSON), &workflow)

		job := &Job{
			ID:         id,
			WorkflowID: workflowID,
			Workflow:   &workflow,
			Priority:   priority,
			Status:     JobStatus(statusStr),
			RetryCount: retryCount,
			MaxRetries: maxRetries,
			CreatedAt:  createdAt,
		}

		if inputDataJSON.Valid {
			json.Unmarshal([]byte(inputDataJSON.String), &job.InputData)
		}
		if resultDataJSON.Valid {
			json.Unmarshal([]byte(resultDataJSON.String), &job.ResultData)
		}
		if errStr.Valid {
			job.Error = errStr.String
		}
		if startedAt.Valid {
			job.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			job.CompletedAt = &completedAt.Time
		}

		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

// Close gracefully shuts down the queue
func (q *SQLiteJobQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return nil
	}

	q.closed = true
	close(q.closeCh)
	return q.db.Close()
}
