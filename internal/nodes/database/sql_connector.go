package database

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dipankar/m9m/internal/expressions"
	"github.com/dipankar/m9m/internal/model"
	"github.com/dipankar/m9m/internal/nodes/base"

	// Import SQL drivers
	_ "github.com/lib/pq"           // PostgreSQL
	_ "github.com/go-sql-driver/mysql" // MySQL
	_ "github.com/mattn/go-sqlite3"    // SQLite
)

// SQLConnectorNode provides database connectivity for SQL databases
type SQLConnectorNode struct {
	*base.BaseNode
	evaluator *expressions.GojaExpressionEvaluator
	db        *sql.DB
	dbConfig  *DatabaseConfig
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Type     string `json:"type"`     // postgres, mysql, sqlite
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	SSLMode  string `json:"sslMode,omitempty"`
	Options  map[string]string `json:"options,omitempty"`
}

// QueryResult represents the result of a database query
type QueryResult struct {
	Rows         []map[string]interface{} `json:"rows"`
	RowCount     int                      `json:"rowCount"`
	AffectedRows int64                    `json:"affectedRows,omitempty"`
	LastInsertID int64                    `json:"lastInsertId,omitempty"`
	ExecutionTime time.Duration           `json:"executionTime"`
	Query        string                   `json:"query"`
}

// NewSQLConnectorNode creates a new SQL connector node
func NewSQLConnectorNode() *SQLConnectorNode {
	return &SQLConnectorNode{
		BaseNode:  base.NewBaseNode(base.NodeDescription{Name: "SQL Database", Description: "n8n-nodes-base.sqlDatabase", Category: "core"}),
		evaluator: expressions.NewGojaExpressionEvaluator(expressions.DefaultEvaluatorConfig()),
	}
}

// Execute performs database operations based on the operation type
func (n *SQLConnectorNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	var results []model.DataItem

	// Get database configuration
	dbConfig, err := n.parseDBConfig(nodeParams)
	if err != nil {
		return nil, n.CreateError("invalid database configuration", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Connect to database if not already connected
	if n.db == nil || n.dbConfig == nil || !n.isSameConfig(dbConfig) {
		if err := n.connect(dbConfig); err != nil {
			return nil, n.CreateError("failed to connect to database", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// Get operation type
	operation, _ := nodeParams["operation"].(string)
	if operation == "" {
		operation = "select" // Default operation
	}

	for index, item := range inputData {
		// Create expression context
		context := &expressions.ExpressionContext{
			ActiveNodeName:      "SQL Database",
			RunIndex:           0,
			ItemIndex:          index,
			Mode:               expressions.ModeManual,
			ConnectionInputData: []model.DataItem{item},
			AdditionalKeys: &expressions.AdditionalKeys{},
		}

		var result *QueryResult
		switch strings.ToLower(operation) {
		case "select", "query":
			result, err = n.executeQuery(nodeParams, context)
		case "insert":
			result, err = n.executeInsert(item, nodeParams, context)
		case "update":
			result, err = n.executeUpdate(item, nodeParams, context)
		case "delete":
			result, err = n.executeDelete(nodeParams, context)
		case "raw":
			result, err = n.executeRawSQL(nodeParams, context)
		default:
			return nil, n.CreateError("unsupported operation", map[string]interface{}{
				"operation": operation,
			})
		}

		if err != nil {
			return nil, n.CreateError("database operation failed", map[string]interface{}{
				"operation": operation,
				"error":     err.Error(),
			})
		}

		// Create result items
		if operation == "select" || operation == "query" {
			// For select queries, create one item per row
			for _, row := range result.Rows {
				resultItem := model.DataItem{
					JSON: row,
				}
				results = append(results, resultItem)
			}
		} else {
			// For other operations, create a single result item
			resultItem := model.DataItem{
				JSON: map[string]interface{}{
					"success":       true,
					"operation":     operation,
					"affectedRows":  result.AffectedRows,
					"lastInsertId":  result.LastInsertID,
					"executionTime": result.ExecutionTime.String(),
					"query":         result.Query,
				},
			}
			results = append(results, resultItem)
		}
	}

	return results, nil
}

// executeQuery performs SELECT queries
func (n *SQLConnectorNode) executeQuery(nodeParams map[string]interface{}, context *expressions.ExpressionContext) (*QueryResult, error) {
	queryExpr, ok := nodeParams["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter is required")
	}

	// Evaluate query expression
	query, err := n.evaluator.EvaluateExpression(queryExpr, context)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate query: %w", err)
	}

	queryStr, ok := query.(string)
	if !ok {
		return nil, fmt.Errorf("query must be a string")
	}

	startTime := time.Now()

	// Execute query
	rows, err := n.db.Query(queryStr)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Process results
	var results []map[string]interface{}
	for rows.Next() {
		// Create scan destinations
		values := make([]interface{}, len(columns))
		scanArgs := make([]interface{}, len(columns))
		for i := range values {
			scanArgs[i] = &values[i]
		}

		// Scan row
		if err := rows.Scan(scanArgs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert to map
		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = n.convertSQLValue(values[i])
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return &QueryResult{
		Rows:          results,
		RowCount:      len(results),
		ExecutionTime: time.Since(startTime),
		Query:         queryStr,
	}, nil
}

// executeInsert performs INSERT operations
func (n *SQLConnectorNode) executeInsert(item model.DataItem, nodeParams map[string]interface{}, context *expressions.ExpressionContext) (*QueryResult, error) {
	table, ok := nodeParams["table"].(string)
	if !ok {
		return nil, fmt.Errorf("table parameter is required")
	}

	// Get columns to insert
	columns := make([]string, 0)
	values := make([]interface{}, 0)

	// Use data mapping if specified
	if mapping, ok := nodeParams["columnMapping"].(map[string]interface{}); ok {
		for column, valueExpr := range mapping {
			if valueExprStr, ok := valueExpr.(string); ok {
				value, err := n.evaluator.EvaluateExpression(valueExprStr, context)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate column %s: %w", column, err)
				}
				columns = append(columns, column)
				values = append(values, value)
			}
		}
	} else {
		// Use all JSON fields from the item
		for key, value := range item.JSON {
			columns = append(columns, key)
			values = append(values, value)
		}
	}

	if len(columns) == 0 {
		return nil, fmt.Errorf("no columns to insert")
	}

	// Build INSERT query
	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = n.getPlaceholder(i + 1)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	return n.executeModifyingQuery(query, values)
}

// executeUpdate performs UPDATE operations
func (n *SQLConnectorNode) executeUpdate(item model.DataItem, nodeParams map[string]interface{}, context *expressions.ExpressionContext) (*QueryResult, error) {
	table, ok := nodeParams["table"].(string)
	if !ok {
		return nil, fmt.Errorf("table parameter is required")
	}

	whereExpr, ok := nodeParams["where"].(string)
	if !ok {
		return nil, fmt.Errorf("where parameter is required for updates")
	}

	// Evaluate WHERE clause
	whereClause, err := n.evaluator.EvaluateExpression(whereExpr, context)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate where clause: %w", err)
	}

	whereStr, ok := whereClause.(string)
	if !ok {
		return nil, fmt.Errorf("where clause must be a string")
	}

	// Get columns to update
	var setParts []string
	var values []interface{}

	if mapping, ok := nodeParams["columnMapping"].(map[string]interface{}); ok {
		for column, valueExpr := range mapping {
			if valueExprStr, ok := valueExpr.(string); ok {
				value, err := n.evaluator.EvaluateExpression(valueExprStr, context)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate column %s: %w", column, err)
				}
				setParts = append(setParts, fmt.Sprintf("%s = %s", column, n.getPlaceholder(len(values)+1)))
				values = append(values, value)
			}
		}
	} else {
		// Use all JSON fields from the item
		for key, value := range item.JSON {
			setParts = append(setParts, fmt.Sprintf("%s = %s", key, n.getPlaceholder(len(values)+1)))
			values = append(values, value)
		}
	}

	if len(setParts) == 0 {
		return nil, fmt.Errorf("no columns to update")
	}

	// Build UPDATE query
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		table,
		strings.Join(setParts, ", "),
		whereStr)

	return n.executeModifyingQuery(query, values)
}

// executeDelete performs DELETE operations
func (n *SQLConnectorNode) executeDelete(nodeParams map[string]interface{}, context *expressions.ExpressionContext) (*QueryResult, error) {
	table, ok := nodeParams["table"].(string)
	if !ok {
		return nil, fmt.Errorf("table parameter is required")
	}

	whereExpr, ok := nodeParams["where"].(string)
	if !ok {
		return nil, fmt.Errorf("where parameter is required for deletes")
	}

	// Evaluate WHERE clause
	whereClause, err := n.evaluator.EvaluateExpression(whereExpr, context)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate where clause: %w", err)
	}

	whereStr, ok := whereClause.(string)
	if !ok {
		return nil, fmt.Errorf("where clause must be a string")
	}

	// Build DELETE query
	query := fmt.Sprintf("DELETE FROM %s WHERE %s", table, whereStr)

	return n.executeModifyingQuery(query, nil)
}

// executeRawSQL executes raw SQL statements
func (n *SQLConnectorNode) executeRawSQL(nodeParams map[string]interface{}, context *expressions.ExpressionContext) (*QueryResult, error) {
	queryExpr, ok := nodeParams["rawQuery"].(string)
	if !ok {
		return nil, fmt.Errorf("rawQuery parameter is required")
	}

	// Evaluate query expression
	query, err := n.evaluator.EvaluateExpression(queryExpr, context)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate query: %w", err)
	}

	queryStr, ok := query.(string)
	if !ok {
		return nil, fmt.Errorf("query must be a string")
	}

	// Determine if this is a SELECT query or modifying query
	trimmed := strings.TrimSpace(strings.ToUpper(queryStr))
	if strings.HasPrefix(trimmed, "SELECT") {
		// Execute as query
		return n.executeQuery(map[string]interface{}{"query": queryStr}, context)
	} else {
		// Execute as modifying query
		return n.executeModifyingQuery(queryStr, nil)
	}
}

// executeModifyingQuery executes INSERT, UPDATE, DELETE queries
func (n *SQLConnectorNode) executeModifyingQuery(query string, args []interface{}) (*QueryResult, error) {
	startTime := time.Now()

	result, err := n.db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	affectedRows, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()

	return &QueryResult{
		AffectedRows:  affectedRows,
		LastInsertID:  lastInsertID,
		ExecutionTime: time.Since(startTime),
		Query:         query,
	}, nil
}

// connect establishes database connection
func (n *SQLConnectorNode) connect(config *DatabaseConfig) error {
	// Close existing connection if any
	if n.db != nil {
		n.db.Close()
	}

	// Build connection string
	dsn, err := n.buildDSN(config)
	if err != nil {
		return fmt.Errorf("failed to build DSN: %w", err)
	}

	// Open database connection
	db, err := sql.Open(config.Type, dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	n.db = db
	n.dbConfig = config

	return nil
}

// buildDSN builds database connection string
func (n *SQLConnectorNode) buildDSN(config *DatabaseConfig) (string, error) {
	switch config.Type {
	case "postgres":
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s",
			config.Host, config.Port, config.Username, config.Password, config.Database)

		if config.SSLMode != "" {
			dsn += " sslmode=" + config.SSLMode
		} else {
			dsn += " sslmode=disable"
		}

		// Add additional options
		for key, value := range config.Options {
			dsn += fmt.Sprintf(" %s=%s", key, value)
		}

		return dsn, nil

	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			config.Username, config.Password, config.Host, config.Port, config.Database)

		// Add options
		if len(config.Options) > 0 {
			params := make([]string, 0, len(config.Options))
			for key, value := range config.Options {
				params = append(params, fmt.Sprintf("%s=%s", key, value))
			}
			dsn += "?" + strings.Join(params, "&")
		}

		return dsn, nil

	case "sqlite":
		return config.Database, nil

	default:
		return "", fmt.Errorf("unsupported database type: %s", config.Type)
	}
}

// parseDBConfig parses database configuration from node parameters
func (n *SQLConnectorNode) parseDBConfig(nodeParams map[string]interface{}) (*DatabaseConfig, error) {
	config := &DatabaseConfig{
		Options: make(map[string]string),
	}

	// Get database type
	dbType, ok := nodeParams["databaseType"].(string)
	if !ok {
		return nil, fmt.Errorf("databaseType is required")
	}
	config.Type = dbType

	// Get connection parameters
	if host, ok := nodeParams["host"].(string); ok {
		config.Host = host
	}

	if port, ok := nodeParams["port"]; ok {
		if portStr, ok := port.(string); ok {
			if portInt, err := strconv.Atoi(portStr); err == nil {
				config.Port = portInt
			}
		} else if portInt, ok := port.(int); ok {
			config.Port = portInt
		}
	}

	if database, ok := nodeParams["database"].(string); ok {
		config.Database = database
	}

	if username, ok := nodeParams["username"].(string); ok {
		config.Username = username
	}

	if password, ok := nodeParams["password"].(string); ok {
		config.Password = password
	}

	if sslMode, ok := nodeParams["sslMode"].(string); ok {
		config.SSLMode = sslMode
	}

	// Validate required fields
	if config.Type == "" {
		return nil, fmt.Errorf("database type is required")
	}

	if config.Type != "sqlite" {
		if config.Host == "" {
			return nil, fmt.Errorf("host is required for %s", config.Type)
		}
		if config.Database == "" {
			return nil, fmt.Errorf("database name is required")
		}
	} else {
		if config.Database == "" {
			return nil, fmt.Errorf("database file path is required for SQLite")
		}
	}

	return config, nil
}

// isSameConfig checks if the new config is the same as current
func (n *SQLConnectorNode) isSameConfig(config *DatabaseConfig) bool {
	if n.dbConfig == nil {
		return false
	}

	return n.dbConfig.Type == config.Type &&
		n.dbConfig.Host == config.Host &&
		n.dbConfig.Port == config.Port &&
		n.dbConfig.Database == config.Database &&
		n.dbConfig.Username == config.Username &&
		n.dbConfig.Password == config.Password
}

// getPlaceholder returns the correct placeholder for the database type
func (n *SQLConnectorNode) getPlaceholder(index int) string {
	switch n.dbConfig.Type {
	case "postgres":
		return fmt.Sprintf("$%d", index)
	case "mysql", "sqlite":
		return "?"
	default:
		return "?"
	}
}

// convertSQLValue converts SQL null values and types to Go types
func (n *SQLConnectorNode) convertSQLValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case []byte:
		// Convert byte arrays to strings
		return string(v)
	case time.Time:
		// Convert time to ISO string
		return v.Format(time.RFC3339)
	default:
		return v
	}
}

// ValidateParameters validates the node parameters
func (n *SQLConnectorNode) ValidateParameters(params map[string]interface{}) error {
	// Validate database configuration
	_, err := n.parseDBConfig(params)
	if err != nil {
		return fmt.Errorf("invalid database configuration: %w", err)
	}

	// Validate operation
	operation, _ := params["operation"].(string)
	validOperations := []string{"select", "query", "insert", "update", "delete", "raw"}
	if operation != "" {
		valid := false
		for _, validOp := range validOperations {
			if operation == validOp {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid operation: %s. Valid operations: %v", operation, validOperations)
		}
	}

	return nil
}

// Close closes the database connection
func (n *SQLConnectorNode) Close() error {
	if n.db != nil {
		return n.db.Close()
	}
	return nil
}