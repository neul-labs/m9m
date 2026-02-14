package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/neul-labs/m9m/internal/model"
)

// PostgresStorage provides PostgreSQL-backed workflow storage
type PostgresStorage struct {
	db *sql.DB
}

// NewPostgresStorage creates a new PostgreSQL storage instance
func NewPostgresStorage(connectionURL string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", connectionURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	storage := &PostgresStorage{db: db}

	// Initialize schema
	if err := storage.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return storage, nil
}

// initSchema creates the necessary database tables
func (s *PostgresStorage) initSchema() error {
	schema := `
		CREATE TABLE IF NOT EXISTS workflows (
			id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			nodes JSONB NOT NULL,
			connections JSONB NOT NULL,
			settings JSONB,
			active BOOLEAN DEFAULT FALSE,
			tags JSONB,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			created_by VARCHAR(255)
		);

		CREATE TABLE IF NOT EXISTS executions (
			id VARCHAR(255) PRIMARY KEY,
			workflow_id VARCHAR(255) NOT NULL,
			status VARCHAR(50) NOT NULL,
			mode VARCHAR(50) NOT NULL,
			started_at TIMESTAMP NOT NULL,
			finished_at TIMESTAMP,
			data JSONB,
			error TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS credentials (
			id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			type VARCHAR(255) NOT NULL,
			data JSONB NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS tags (
			id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			color VARCHAR(50),
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_workflows_active ON workflows(active);
		CREATE INDEX IF NOT EXISTS idx_workflows_name ON workflows(name);
		CREATE INDEX IF NOT EXISTS idx_executions_workflow_id ON executions(workflow_id);
		CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Workflow operations

func (s *PostgresStorage) SaveWorkflow(workflow *model.Workflow) error {
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
		INSERT INTO workflows (id, name, description, nodes, connections, settings, active, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			nodes = EXCLUDED.nodes,
			connections = EXCLUDED.connections,
			settings = EXCLUDED.settings,
			active = EXCLUDED.active,
			tags = EXCLUDED.tags,
			updated_at = EXCLUDED.updated_at
	`

	_, err := s.db.Exec(query, workflow.ID, workflow.Name, workflow.Description,
		nodesJSON, connectionsJSON, settingsJSON, workflow.Active, tagsJSON,
		workflow.CreatedAt, workflow.UpdatedAt)

	return err
}

func (s *PostgresStorage) GetWorkflow(id string) (*model.Workflow, error) {
	query := `
		SELECT id, name, description, nodes, connections, settings, active, tags, created_at, updated_at
		FROM workflows WHERE id = $1
	`

	var workflow model.Workflow
	var nodesJSON, connectionsJSON, settingsJSON, tagsJSON []byte

	err := s.db.QueryRow(query, id).Scan(
		&workflow.ID, &workflow.Name, &workflow.Description,
		&nodesJSON, &connectionsJSON, &settingsJSON,
		&workflow.Active, &tagsJSON, &workflow.CreatedAt, &workflow.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workflow not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(nodesJSON, &workflow.Nodes)
	json.Unmarshal(connectionsJSON, &workflow.Connections)
	json.Unmarshal(settingsJSON, &workflow.Settings)
	json.Unmarshal(tagsJSON, &workflow.Tags)

	return &workflow, nil
}

func (s *PostgresStorage) ListWorkflows(filters WorkflowFilters) ([]*model.Workflow, int, error) {
	var conditions []string
	var args []interface{}
	argCount := 1

	if filters.Active != nil {
		conditions = append(conditions, fmt.Sprintf("active = $%d", argCount))
		args = append(args, *filters.Active)
		argCount++
	}

	if filters.Search != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(name) LIKE $%d", argCount))
		args = append(args, "%"+strings.ToLower(filters.Search)+"%")
		argCount++
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
		LIMIT $%d OFFSET $%d
	`, whereClause, argCount, argCount+1)

	args = append(args, filters.Limit, filters.Offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var workflows []*model.Workflow
	for rows.Next() {
		var workflow model.Workflow
		var nodesJSON, connectionsJSON, settingsJSON, tagsJSON []byte

		err := rows.Scan(
			&workflow.ID, &workflow.Name, &workflow.Description,
			&nodesJSON, &connectionsJSON, &settingsJSON,
			&workflow.Active, &tagsJSON, &workflow.CreatedAt, &workflow.UpdatedAt,
		)
		if err != nil {
			continue
		}

		json.Unmarshal(nodesJSON, &workflow.Nodes)
		json.Unmarshal(connectionsJSON, &workflow.Connections)
		json.Unmarshal(settingsJSON, &workflow.Settings)
		json.Unmarshal(tagsJSON, &workflow.Tags)

		workflows = append(workflows, &workflow)
	}

	return workflows, total, nil
}

func (s *PostgresStorage) UpdateWorkflow(id string, workflow *model.Workflow) error {
	workflow.ID = id
	workflow.UpdatedAt = time.Now()
	return s.SaveWorkflow(workflow)
}

func (s *PostgresStorage) DeleteWorkflow(id string) error {
	query := "DELETE FROM workflows WHERE id = $1"
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

func (s *PostgresStorage) ActivateWorkflow(id string) error {
	query := "UPDATE workflows SET active = TRUE, updated_at = $1 WHERE id = $2"
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

func (s *PostgresStorage) DeactivateWorkflow(id string) error {
	query := "UPDATE workflows SET active = FALSE, updated_at = $1 WHERE id = $2"
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

// Execution operations (simplified implementations)

func (s *PostgresStorage) SaveExecution(execution *model.WorkflowExecution) error {
	if execution.ID == "" {
		execution.ID = generateID("exec")
	}
	if execution.StartedAt.IsZero() {
		execution.StartedAt = time.Now()
	}

	dataJSON, _ := json.Marshal(execution.Data)
	errorText := ""
	if execution.Error != nil {
		errorText = execution.Error.Error()
	}
	createdAt := time.Now()

	query := `
		INSERT INTO executions (id, workflow_id, status, mode, started_at, finished_at, data, error, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			workflow_id = EXCLUDED.workflow_id,
			status = EXCLUDED.status,
			mode = EXCLUDED.mode,
			started_at = EXCLUDED.started_at,
			finished_at = EXCLUDED.finished_at,
			data = EXCLUDED.data,
			error = EXCLUDED.error
	`

	_, err := s.db.Exec(query, execution.ID, execution.WorkflowID, execution.Status,
		execution.Mode, execution.StartedAt, execution.FinishedAt, dataJSON, errorText, createdAt)

	return err
}

func (s *PostgresStorage) GetExecution(id string) (*model.WorkflowExecution, error) {
	query := `
		SELECT id, workflow_id, status, mode, started_at, finished_at, data, error
		FROM executions WHERE id = $1
	`

	var execution model.WorkflowExecution
	var dataJSON []byte
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

	json.Unmarshal(dataJSON, &execution.Data)

	if errorText.Valid && errorText.String != "" {
		execution.Error = fmt.Errorf("%s", errorText.String)
	}

	return &execution, nil
}

func (s *PostgresStorage) ListExecutions(filters ExecutionFilters) ([]*model.WorkflowExecution, int, error) {
	var conditions []string
	var args []interface{}
	argCount := 1

	if filters.WorkflowID != "" {
		conditions = append(conditions, fmt.Sprintf("workflow_id = $%d", argCount))
		args = append(args, filters.WorkflowID)
		argCount++
	}

	if filters.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argCount))
		args = append(args, filters.Status)
		argCount++
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
		LIMIT $%d OFFSET $%d
	`, whereClause, argCount, argCount+1)

	args = append(args, filters.Limit, filters.Offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var executions []*model.WorkflowExecution
	for rows.Next() {
		var execution model.WorkflowExecution
		var dataJSON []byte
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

		json.Unmarshal(dataJSON, &execution.Data)

		if errorText.Valid && errorText.String != "" {
			execution.Error = fmt.Errorf("%s", errorText.String)
		}

		executions = append(executions, &execution)
	}

	return executions, total, nil
}

func (s *PostgresStorage) DeleteExecution(id string) error {
	query := "DELETE FROM executions WHERE id = $1"
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

// Credential operations (simplified implementations)

func (s *PostgresStorage) SaveCredential(credential *Credential) error {
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
		INSERT INTO credentials (id, name, type, data, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			type = EXCLUDED.type,
			data = EXCLUDED.data,
			updated_at = EXCLUDED.updated_at
	`

	_, err := s.db.Exec(query, credential.ID, credential.Name, credential.Type,
		dataJSON, credential.CreatedAt, credential.UpdatedAt)

	return err
}

func (s *PostgresStorage) GetCredential(id string) (*Credential, error) {
	query := "SELECT id, name, type, data, created_at, updated_at FROM credentials WHERE id = $1"

	var credential Credential
	var dataJSON []byte

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

	json.Unmarshal(dataJSON, &credential.Data)

	return &credential, nil
}

func (s *PostgresStorage) ListCredentials() ([]*Credential, error) {
	query := "SELECT id, name, type, data, created_at, updated_at FROM credentials ORDER BY name"

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var credentials []*Credential
	for rows.Next() {
		var credential Credential
		var dataJSON []byte

		err := rows.Scan(
			&credential.ID, &credential.Name, &credential.Type,
			&dataJSON, &credential.CreatedAt, &credential.UpdatedAt,
		)
		if err != nil {
			continue
		}

		json.Unmarshal(dataJSON, &credential.Data)
		credentials = append(credentials, &credential)
	}

	return credentials, nil
}

func (s *PostgresStorage) UpdateCredential(id string, credential *Credential) error {
	credential.ID = id
	credential.UpdatedAt = time.Now()
	return s.SaveCredential(credential)
}

func (s *PostgresStorage) DeleteCredential(id string) error {
	query := "DELETE FROM credentials WHERE id = $1"
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

// Tag operations (simplified implementations)

func (s *PostgresStorage) SaveTag(tag *Tag) error {
	if tag.ID == "" {
		tag.ID = generateID("tag")
	}

	now := time.Now()
	if tag.CreatedAt.IsZero() {
		tag.CreatedAt = now
	}
	tag.UpdatedAt = now

	query := `
		INSERT INTO tags (id, name, color, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			color = EXCLUDED.color,
			updated_at = EXCLUDED.updated_at
	`

	_, err := s.db.Exec(query, tag.ID, tag.Name, tag.Color, tag.CreatedAt, tag.UpdatedAt)
	return err
}

func (s *PostgresStorage) GetTag(id string) (*Tag, error) {
	query := "SELECT id, name, color, created_at, updated_at FROM tags WHERE id = $1"

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

func (s *PostgresStorage) ListTags() ([]*Tag, error) {
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

func (s *PostgresStorage) UpdateTag(id string, tag *Tag) error {
	tag.ID = id
	tag.UpdatedAt = time.Now()
	return s.SaveTag(tag)
}

func (s *PostgresStorage) DeleteTag(id string) error {
	query := "DELETE FROM tags WHERE id = $1"
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

func (s *PostgresStorage) SaveRaw(key string, value []byte) error {
	_, err := s.db.Exec(`
		INSERT INTO raw_data (key, value, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()
	`, key, value)

	return err
}

func (s *PostgresStorage) GetRaw(key string) ([]byte, error) {
	var value []byte
	err := s.db.QueryRow("SELECT value FROM raw_data WHERE key = $1", key).Scan(&value)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	return value, err
}

func (s *PostgresStorage) ListKeys(prefix string) ([]string, error) {
	rows, err := s.db.Query("SELECT key FROM raw_data WHERE key LIKE $1", prefix+"%")
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

func (s *PostgresStorage) DeleteRaw(key string) error {
	_, err := s.db.Exec("DELETE FROM raw_data WHERE key = $1", key)
	return err
}

// Close closes the database connection
func (s *PostgresStorage) Close() error {
	return s.db.Close()
}
