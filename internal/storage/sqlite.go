package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/dipankar/n8n-go/internal/model"
)

// SQLiteStorage provides SQLite-backed workflow storage
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite storage instance
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	storage := &SQLiteStorage{db: db}

	// Initialize schema
	if err := storage.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return storage, nil
}

// initSchema creates the necessary database tables
func (s *SQLiteStorage) initSchema() error {
	schema := `
		CREATE TABLE IF NOT EXISTS workflows (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			nodes TEXT NOT NULL,
			connections TEXT NOT NULL,
			settings TEXT,
			active INTEGER DEFAULT 0,
			tags TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			created_by TEXT
		);

		CREATE TABLE IF NOT EXISTS executions (
			id TEXT PRIMARY KEY,
			workflow_id TEXT NOT NULL,
			status TEXT NOT NULL,
			mode TEXT NOT NULL,
			started_at DATETIME NOT NULL,
			finished_at DATETIME,
			data TEXT,
			error TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS credentials (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			data TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS tags (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			color TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_workflows_active ON workflows(active);
		CREATE INDEX IF NOT EXISTS idx_workflows_name ON workflows(name);
		CREATE INDEX IF NOT EXISTS idx_executions_workflow_id ON executions(workflow_id);
		CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Workflow operations (similar to PostgreSQL but adapted for SQLite)

func (s *SQLiteStorage) SaveWorkflow(workflow *model.Workflow) error {
	if workflow.ID == "" {
		workflow.ID = generateID("workflow")
	}

	now := time.Now()
	if workflow.CreatedAt.IsZero() {
		workflow.CreatedAt = now
	}
	workflow.UpdatedAt = now

	nodesJSON, _ := json.Marshal(workflow.Nodes)
	connectionsJSON, _ := json.Marshal(workflow.Connections)
	settingsJSON, _ := json.Marshal(workflow.Settings)
	tagsJSON, _ := json.Marshal(workflow.Tags)

	query := `
		INSERT OR REPLACE INTO workflows
		(id, name, description, nodes, connections, settings, active, tags, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	activeInt := 0
	if workflow.Active {
		activeInt = 1
	}

	_, err := s.db.Exec(query, workflow.ID, workflow.Name, workflow.Description,
		string(nodesJSON), string(connectionsJSON), string(settingsJSON), activeInt,
		string(tagsJSON), workflow.CreatedAt, workflow.UpdatedAt)

	return err
}

func (s *SQLiteStorage) GetWorkflow(id string) (*model.Workflow, error) {
	query := `
		SELECT id, name, description, nodes, connections, settings, active, tags, created_at, updated_at
		FROM workflows WHERE id = ?
	`

	var workflow model.Workflow
	var nodesJSON, connectionsJSON, settingsJSON, tagsJSON string
	var activeInt int

	err := s.db.QueryRow(query, id).Scan(
		&workflow.ID, &workflow.Name, &workflow.Description,
		&nodesJSON, &connectionsJSON, &settingsJSON,
		&activeInt, &tagsJSON, &workflow.CreatedAt, &workflow.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workflow not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	workflow.Active = activeInt == 1

	json.Unmarshal([]byte(nodesJSON), &workflow.Nodes)
	json.Unmarshal([]byte(connectionsJSON), &workflow.Connections)
	json.Unmarshal([]byte(settingsJSON), &workflow.Settings)
	json.Unmarshal([]byte(tagsJSON), &workflow.Tags)

	return &workflow, nil
}

func (s *SQLiteStorage) ListWorkflows(filters WorkflowFilters) ([]*model.Workflow, int, error) {
	var conditions []string
	var args []interface{}

	if filters.Active != nil {
		activeInt := 0
		if *filters.Active {
			activeInt = 1
		}
		conditions = append(conditions, "active = ?")
		args = append(args, activeInt)
	}

	if filters.Search != "" {
		conditions = append(conditions, "LOWER(name) LIKE ?")
		args = append(args, "%"+strings.ToLower(filters.Search)+"%")
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM workflows %s", whereClause)
	var total int
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get workflows with pagination
	query := fmt.Sprintf(`
		SELECT id, name, description, nodes, connections, settings, active, tags, created_at, updated_at
		FROM workflows %s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, filters.Limit, filters.Offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var workflows []*model.Workflow
	for rows.Next() {
		var workflow model.Workflow
		var nodesJSON, connectionsJSON, settingsJSON, tagsJSON string
		var activeInt int

		err := rows.Scan(
			&workflow.ID, &workflow.Name, &workflow.Description,
			&nodesJSON, &connectionsJSON, &settingsJSON,
			&activeInt, &tagsJSON, &workflow.CreatedAt, &workflow.UpdatedAt,
		)
		if err != nil {
			continue
		}

		workflow.Active = activeInt == 1

		json.Unmarshal([]byte(nodesJSON), &workflow.Nodes)
		json.Unmarshal([]byte(connectionsJSON), &workflow.Connections)
		json.Unmarshal([]byte(settingsJSON), &workflow.Settings)
		json.Unmarshal([]byte(tagsJSON), &workflow.Tags)

		workflows = append(workflows, &workflow)
	}

	return workflows, total, nil
}

func (s *SQLiteStorage) UpdateWorkflow(id string, workflow *model.Workflow) error {
	workflow.ID = id
	workflow.UpdatedAt = time.Now()
	return s.SaveWorkflow(workflow)
}

func (s *SQLiteStorage) DeleteWorkflow(id string) error {
	query := "DELETE FROM workflows WHERE id = ?"
	result, err := s.db.Exec(query, id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("workflow not found: %s", id)
	}

	return nil
}

func (s *SQLiteStorage) ActivateWorkflow(id string) error {
	query := "UPDATE workflows SET active = 1, updated_at = ? WHERE id = ?"
	result, err := s.db.Exec(query, time.Now(), id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("workflow not found: %s", id)
	}

	return nil
}

func (s *SQLiteStorage) DeactivateWorkflow(id string) error {
	query := "UPDATE workflows SET active = 0, updated_at = ? WHERE id = ?"
	result, err := s.db.Exec(query, time.Now(), id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("workflow not found: %s", id)
	}

	return nil
}

// Execution operations

func (s *SQLiteStorage) SaveExecution(execution *model.WorkflowExecution) error {
	if execution.ID == "" {
		execution.ID = generateID("exec")
	}

	dataJSON, _ := json.Marshal(execution.Data)
	errorText := ""
	if execution.Error != nil {
		errorText = execution.Error.Error()
	}

	query := `
		INSERT INTO executions (id, workflow_id, status, mode, started_at, finished_at, data, error, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query, execution.ID, execution.WorkflowID, execution.Status,
		execution.Mode, execution.StartedAt, execution.FinishedAt, string(dataJSON), errorText, time.Now())

	return err
}

func (s *SQLiteStorage) GetExecution(id string) (*model.WorkflowExecution, error) {
	query := `
		SELECT id, workflow_id, status, mode, started_at, finished_at, data, error
		FROM executions WHERE id = ?
	`

	var execution model.WorkflowExecution
	var dataJSON string
	var errorText sql.NullString
	var finishedAt sql.NullTime

	err := s.db.QueryRow(query, id).Scan(
		&execution.ID, &execution.WorkflowID, &execution.Status, &execution.Mode,
		&execution.StartedAt, &finishedAt, &dataJSON, &errorText,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("execution not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	if finishedAt.Valid {
		execution.FinishedAt = &finishedAt.Time
	}

	json.Unmarshal([]byte(dataJSON), &execution.Data)

	if errorText.Valid && errorText.String != "" {
		execution.Error = fmt.Errorf("%s", errorText.String)
	}

	return &execution, nil
}

func (s *SQLiteStorage) ListExecutions(filters ExecutionFilters) ([]*model.WorkflowExecution, int, error) {
	var conditions []string
	var args []interface{}

	if filters.WorkflowID != "" {
		conditions = append(conditions, "workflow_id = ?")
		args = append(args, filters.WorkflowID)
	}

	if filters.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, filters.Status)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM executions %s", whereClause)
	var total int
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get executions with pagination
	query := fmt.Sprintf(`
		SELECT id, workflow_id, status, mode, started_at, finished_at, data, error
		FROM executions %s
		ORDER BY started_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, filters.Limit, filters.Offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var executions []*model.WorkflowExecution
	for rows.Next() {
		var execution model.WorkflowExecution
		var dataJSON string
		var errorText sql.NullString
		var finishedAt sql.NullTime

		err := rows.Scan(
			&execution.ID, &execution.WorkflowID, &execution.Status, &execution.Mode,
			&execution.StartedAt, &finishedAt, &dataJSON, &errorText,
		)
		if err != nil {
			continue
		}

		if finishedAt.Valid {
			execution.FinishedAt = &finishedAt.Time
		}

		json.Unmarshal([]byte(dataJSON), &execution.Data)

		if errorText.Valid && errorText.String != "" {
			execution.Error = fmt.Errorf("%s", errorText.String)
		}

		executions = append(executions, &execution)
	}

	return executions, total, nil
}

func (s *SQLiteStorage) DeleteExecution(id string) error {
	query := "DELETE FROM executions WHERE id = ?"
	result, err := s.db.Exec(query, id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("execution not found: %s", id)
	}

	return nil
}

// Credential operations

func (s *SQLiteStorage) SaveCredential(credential *Credential) error {
	if credential.ID == "" {
		credential.ID = generateID("cred")
	}

	now := time.Now()
	if credential.CreatedAt.IsZero() {
		credential.CreatedAt = now
	}
	credential.UpdatedAt = now

	dataJSON, _ := json.Marshal(credential.Data)

	query := `
		INSERT OR REPLACE INTO credentials (id, name, type, data, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query, credential.ID, credential.Name, credential.Type,
		string(dataJSON), credential.CreatedAt, credential.UpdatedAt)

	return err
}

func (s *SQLiteStorage) GetCredential(id string) (*Credential, error) {
	query := "SELECT id, name, type, data, created_at, updated_at FROM credentials WHERE id = ?"

	var credential Credential
	var dataJSON string

	err := s.db.QueryRow(query, id).Scan(
		&credential.ID, &credential.Name, &credential.Type,
		&dataJSON, &credential.CreatedAt, &credential.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("credential not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(dataJSON), &credential.Data)

	return &credential, nil
}

func (s *SQLiteStorage) ListCredentials() ([]*Credential, error) {
	query := "SELECT id, name, type, data, created_at, updated_at FROM credentials ORDER BY name"

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var credentials []*Credential
	for rows.Next() {
		var credential Credential
		var dataJSON string

		err := rows.Scan(
			&credential.ID, &credential.Name, &credential.Type,
			&dataJSON, &credential.CreatedAt, &credential.UpdatedAt,
		)
		if err != nil {
			continue
		}

		json.Unmarshal([]byte(dataJSON), &credential.Data)
		credentials = append(credentials, &credential)
	}

	return credentials, nil
}

func (s *SQLiteStorage) UpdateCredential(id string, credential *Credential) error {
	credential.ID = id
	credential.UpdatedAt = time.Now()
	return s.SaveCredential(credential)
}

func (s *SQLiteStorage) DeleteCredential(id string) error {
	query := "DELETE FROM credentials WHERE id = ?"
	result, err := s.db.Exec(query, id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("credential not found: %s", id)
	}

	return nil
}

// Tag operations

func (s *SQLiteStorage) SaveTag(tag *Tag) error {
	if tag.ID == "" {
		tag.ID = generateID("tag")
	}

	now := time.Now()
	if tag.CreatedAt.IsZero() {
		tag.CreatedAt = now
	}
	tag.UpdatedAt = now

	query := `
		INSERT OR REPLACE INTO tags (id, name, color, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query, tag.ID, tag.Name, tag.Color, tag.CreatedAt, tag.UpdatedAt)
	return err
}

func (s *SQLiteStorage) GetTag(id string) (*Tag, error) {
	query := "SELECT id, name, color, created_at, updated_at FROM tags WHERE id = ?"

	var tag Tag
	err := s.db.QueryRow(query, id).Scan(&tag.ID, &tag.Name, &tag.Color, &tag.CreatedAt, &tag.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tag not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	return &tag, nil
}

func (s *SQLiteStorage) ListTags() ([]*Tag, error) {
	query := "SELECT id, name, color, created_at, updated_at FROM tags ORDER BY name"

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*Tag
	for rows.Next() {
		var tag Tag
		err := rows.Scan(&tag.ID, &tag.Name, &tag.Color, &tag.CreatedAt, &tag.UpdatedAt)
		if err != nil {
			continue
		}
		tags = append(tags, &tag)
	}

	return tags, nil
}

func (s *SQLiteStorage) UpdateTag(id string, tag *Tag) error {
	tag.ID = id
	tag.UpdatedAt = time.Now()
	return s.SaveTag(tag)
}

func (s *SQLiteStorage) DeleteTag(id string) error {
	query := "DELETE FROM tags WHERE id = ?"
	result, err := s.db.Exec(query, id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("tag not found: %s", id)
	}

	return nil
}

// Raw key-value operations

func (s *SQLiteStorage) SaveRaw(key string, value []byte) error {
	_, err := s.db.Exec(`
		INSERT INTO raw_data (key, value, created_at, updated_at)
		VALUES (?, ?, datetime('now'), datetime('now'))
		ON CONFLICT (key) DO UPDATE SET value = ?, updated_at = datetime('now')
	`, key, value, value)

	return err
}

func (s *SQLiteStorage) GetRaw(key string) ([]byte, error) {
	var value []byte
	err := s.db.QueryRow("SELECT value FROM raw_data WHERE key = ?", key).Scan(&value)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	return value, err
}

func (s *SQLiteStorage) ListKeys(prefix string) ([]string, error) {
	rows, err := s.db.Query("SELECT key FROM raw_data WHERE key LIKE ?", prefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}

	return keys, rows.Err()
}

func (s *SQLiteStorage) DeleteRaw(key string) error {
	_, err := s.db.Exec("DELETE FROM raw_data WHERE key = ?", key)
	return err
}

// Close closes the database connection
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
