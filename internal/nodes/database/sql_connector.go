package database

import (
	"database/sql"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/neul-labs/m9m/internal/expressions"
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"

	// Import SQL drivers
	_ "github.com/lib/pq"              // PostgreSQL
	_ "github.com/go-sql-driver/mysql" // MySQL
	_ "github.com/mattn/go-sqlite3"    // SQLite
)

// validIdentifierRegex validates SQL identifiers (table/column names)
// Only allows alphanumeric characters, underscores, and must start with letter or underscore
var validIdentifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// maxIdentifierLength prevents excessively long identifiers
const maxIdentifierLength = 128

// validateSQLIdentifier validates a SQL identifier (table/column name) to prevent SQL injection
func validateSQLIdentifier(identifier string) error {
	if identifier == "" {
		return fmt.Errorf("identifier cannot be empty")
	}
	if len(identifier) > maxIdentifierLength {
		return fmt.Errorf("identifier exceeds maximum length of %d characters", maxIdentifierLength)
	}
	if !validIdentifierRegex.MatchString(identifier) {
		return fmt.Errorf("invalid identifier '%s': must contain only alphanumeric characters and underscores, and start with a letter or underscore", identifier)
	}
	// Additional check for SQL reserved words that could cause issues
	reservedWords := map[string]bool{
		"SELECT": true, "INSERT": true, "UPDATE": true, "DELETE": true, "DROP": true,
		"CREATE": true, "ALTER": true, "TRUNCATE": true, "GRANT": true, "REVOKE": true,
		"UNION": true, "WHERE": true, "FROM": true, "TABLE": true, "DATABASE": true,
	}
	if reservedWords[strings.ToUpper(identifier)] {
		return fmt.Errorf("identifier '%s' is a SQL reserved word", identifier)
	}
	return nil
}

// quoteIdentifier properly quotes a SQL identifier for the given database type
func quoteIdentifier(identifier, dbType string) string {
	switch dbType {
	case "postgres":
		return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
	case "mysql":
		return "`" + strings.ReplaceAll(identifier, "`", "``") + "`"
	case "sqlite":
		return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
	default:
		return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
	}
}

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

	// SECURITY: Validate table name to prevent SQL injection
	if err := validateSQLIdentifier(table); err != nil {
		return nil, fmt.Errorf("invalid table name: %w", err)
	}

	// Get columns to insert
	columns := make([]string, 0)
	values := make([]interface{}, 0)

	// Use data mapping if specified
	if mapping, ok := nodeParams["columnMapping"].(map[string]interface{}); ok {
		for column, valueExpr := range mapping {
			// SECURITY: Validate column name to prevent SQL injection
			if err := validateSQLIdentifier(column); err != nil {
				return nil, fmt.Errorf("invalid column name '%s': %w", column, err)
			}
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
			// SECURITY: Validate column name to prevent SQL injection
			if err := validateSQLIdentifier(key); err != nil {
				return nil, fmt.Errorf("invalid column name '%s': %w", key, err)
			}
			columns = append(columns, key)
			values = append(values, value)
		}
	}

	if len(columns) == 0 {
		return nil, fmt.Errorf("no columns to insert")
	}

	// Build INSERT query with properly quoted identifiers
	placeholders := make([]string, len(columns))
	quotedColumns := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = n.getPlaceholder(i + 1)
		quotedColumns[i] = quoteIdentifier(columns[i], n.dbConfig.Type)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		quoteIdentifier(table, n.dbConfig.Type),
		strings.Join(quotedColumns, ", "),
		strings.Join(placeholders, ", "))

	return n.executeModifyingQuery(query, values)
}

// executeUpdate performs UPDATE operations
func (n *SQLConnectorNode) executeUpdate(item model.DataItem, nodeParams map[string]interface{}, context *expressions.ExpressionContext) (*QueryResult, error) {
	table, ok := nodeParams["table"].(string)
	if !ok {
		return nil, fmt.Errorf("table parameter is required")
	}

	// SECURITY: Validate table name to prevent SQL injection
	if err := validateSQLIdentifier(table); err != nil {
		return nil, fmt.Errorf("invalid table name: %w", err)
	}

	// SECURITY: For WHERE clauses, we now require parameterized conditions
	// The whereColumn and whereValue parameters are used instead of raw WHERE strings
	whereColumn, hasWhereColumn := nodeParams["whereColumn"].(string)
	whereValueExpr, hasWhereValue := nodeParams["whereValue"].(string)

	// Legacy support: if old "where" parameter is used, validate it more strictly
	if !hasWhereColumn || !hasWhereValue {
		whereExpr, ok := nodeParams["where"].(string)
		if !ok {
			return nil, fmt.Errorf("either (whereColumn and whereValue) or where parameter is required for updates")
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

		// SECURITY WARNING: Legacy WHERE string is being used
		// This is less secure than parameterized WHERE. Log a warning.
		// For production, consider requiring parameterized WHERE only.
		return n.executeUpdateWithLegacyWhere(item, table, whereStr, nodeParams, context)
	}

	// SECURITY: Validate WHERE column name
	if err := validateSQLIdentifier(whereColumn); err != nil {
		return nil, fmt.Errorf("invalid whereColumn name: %w", err)
	}

	// Evaluate WHERE value
	whereValue, err := n.evaluator.EvaluateExpression(whereValueExpr, context)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate whereValue: %w", err)
	}

	// Get columns to update
	var setParts []string
	var values []interface{}

	if mapping, ok := nodeParams["columnMapping"].(map[string]interface{}); ok {
		for column, valueExpr := range mapping {
			// SECURITY: Validate column name
			if err := validateSQLIdentifier(column); err != nil {
				return nil, fmt.Errorf("invalid column name '%s': %w", column, err)
			}
			if valueExprStr, ok := valueExpr.(string); ok {
				value, err := n.evaluator.EvaluateExpression(valueExprStr, context)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate column %s: %w", column, err)
				}
				setParts = append(setParts, fmt.Sprintf("%s = %s", quoteIdentifier(column, n.dbConfig.Type), n.getPlaceholder(len(values)+1)))
				values = append(values, value)
			}
		}
	} else {
		// Use all JSON fields from the item
		for key, value := range item.JSON {
			// SECURITY: Validate column name
			if err := validateSQLIdentifier(key); err != nil {
				return nil, fmt.Errorf("invalid column name '%s': %w", key, err)
			}
			setParts = append(setParts, fmt.Sprintf("%s = %s", quoteIdentifier(key, n.dbConfig.Type), n.getPlaceholder(len(values)+1)))
			values = append(values, value)
		}
	}

	if len(setParts) == 0 {
		return nil, fmt.Errorf("no columns to update")
	}

	// Add WHERE value as the last parameter
	values = append(values, whereValue)

	// Build UPDATE query with parameterized WHERE
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = %s",
		quoteIdentifier(table, n.dbConfig.Type),
		strings.Join(setParts, ", "),
		quoteIdentifier(whereColumn, n.dbConfig.Type),
		n.getPlaceholder(len(values)))

	return n.executeModifyingQuery(query, values)
}

// executeUpdateWithLegacyWhere handles legacy WHERE string (less secure)
// DEPRECATED: Use whereColumn/whereValue parameters instead
func (n *SQLConnectorNode) executeUpdateWithLegacyWhere(item model.DataItem, table, whereStr string, nodeParams map[string]interface{}, context *expressions.ExpressionContext) (*QueryResult, error) {
	// Get columns to update
	var setParts []string
	var values []interface{}

	if mapping, ok := nodeParams["columnMapping"].(map[string]interface{}); ok {
		for column, valueExpr := range mapping {
			// SECURITY: Validate column name
			if err := validateSQLIdentifier(column); err != nil {
				return nil, fmt.Errorf("invalid column name '%s': %w", column, err)
			}
			if valueExprStr, ok := valueExpr.(string); ok {
				value, err := n.evaluator.EvaluateExpression(valueExprStr, context)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate column %s: %w", column, err)
				}
				setParts = append(setParts, fmt.Sprintf("%s = %s", quoteIdentifier(column, n.dbConfig.Type), n.getPlaceholder(len(values)+1)))
				values = append(values, value)
			}
		}
	} else {
		// Use all JSON fields from the item
		for key, value := range item.JSON {
			// SECURITY: Validate column name
			if err := validateSQLIdentifier(key); err != nil {
				return nil, fmt.Errorf("invalid column name '%s': %w", key, err)
			}
			setParts = append(setParts, fmt.Sprintf("%s = %s", quoteIdentifier(key, n.dbConfig.Type), n.getPlaceholder(len(values)+1)))
			values = append(values, value)
		}
	}

	if len(setParts) == 0 {
		return nil, fmt.Errorf("no columns to update")
	}

	// Build UPDATE query - WARNING: whereStr is not parameterized
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		quoteIdentifier(table, n.dbConfig.Type),
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

	// SECURITY: Validate table name to prevent SQL injection
	if err := validateSQLIdentifier(table); err != nil {
		return nil, fmt.Errorf("invalid table name: %w", err)
	}

	// SECURITY: For WHERE clauses, we now require parameterized conditions
	// The whereColumn and whereValue parameters are used instead of raw WHERE strings
	whereColumn, hasWhereColumn := nodeParams["whereColumn"].(string)
	whereValueExpr, hasWhereValue := nodeParams["whereValue"].(string)

	// Legacy support: if old "where" parameter is used
	if !hasWhereColumn || !hasWhereValue {
		whereExpr, ok := nodeParams["where"].(string)
		if !ok {
			return nil, fmt.Errorf("either (whereColumn and whereValue) or where parameter is required for deletes")
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

		// SECURITY WARNING: Legacy WHERE string - less secure
		// Build DELETE query - WARNING: whereStr is not parameterized
		query := fmt.Sprintf("DELETE FROM %s WHERE %s", quoteIdentifier(table, n.dbConfig.Type), whereStr)
		return n.executeModifyingQuery(query, nil)
	}

	// SECURITY: Validate WHERE column name
	if err := validateSQLIdentifier(whereColumn); err != nil {
		return nil, fmt.Errorf("invalid whereColumn name: %w", err)
	}

	// Evaluate WHERE value
	whereValue, err := n.evaluator.EvaluateExpression(whereValueExpr, context)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate whereValue: %w", err)
	}

	// Build DELETE query with parameterized WHERE
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = %s",
		quoteIdentifier(table, n.dbConfig.Type),
		quoteIdentifier(whereColumn, n.dbConfig.Type),
		n.getPlaceholder(1))

	return n.executeModifyingQuery(query, []interface{}{whereValue})
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
		// SECURITY: Use URL format with proper escaping to handle special characters in credentials
		// Build connection URL: postgres://user:password@host:port/database?sslmode=require
		u := &url.URL{
			Scheme: "postgres",
			User:   url.UserPassword(config.Username, config.Password),
			Host:   fmt.Sprintf("%s:%d", config.Host, config.Port),
			Path:   "/" + config.Database,
		}

		// Build query parameters
		q := u.Query()

		// SECURITY: Default to sslmode=require for encrypted connections
		// Only allow sslmode=disable if explicitly set (and warn in logs)
		if config.SSLMode != "" {
			q.Set("sslmode", config.SSLMode)
		} else {
			q.Set("sslmode", "require") // SECURITY: Changed from "disable" to "require"
		}

		// Add additional options with proper escaping
		for key, value := range config.Options {
			// SECURITY: Validate option keys to prevent injection
			if !validIdentifierRegex.MatchString(key) {
				return "", fmt.Errorf("invalid option key '%s': must contain only alphanumeric characters and underscores", key)
			}
			q.Set(key, value)
		}

		u.RawQuery = q.Encode()
		return u.String(), nil

	case "mysql":
		// SECURITY: Properly escape username and password for MySQL DSN
		// MySQL DSN format: user:password@tcp(host:port)/database?params
		// Use url.QueryEscape for credentials to handle special characters
		escapedUser := url.QueryEscape(config.Username)
		escapedPass := url.QueryEscape(config.Password)

		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			escapedUser, escapedPass, config.Host, config.Port, config.Database)

		// Add options with proper escaping
		params := make([]string, 0, len(config.Options)+1)

		// SECURITY: Enable TLS by default for MySQL
		if _, hasTLS := config.Options["tls"]; !hasTLS {
			params = append(params, "tls=preferred")
		}

		for key, value := range config.Options {
			// SECURITY: Validate option keys
			if !validIdentifierRegex.MatchString(key) {
				return "", fmt.Errorf("invalid option key '%s': must contain only alphanumeric characters and underscores", key)
			}
			params = append(params, fmt.Sprintf("%s=%s", url.QueryEscape(key), url.QueryEscape(value)))
		}

		if len(params) > 0 {
			dsn += "?" + strings.Join(params, "&")
		}

		return dsn, nil

	case "sqlite":
		// SECURITY: Validate SQLite database path
		if strings.Contains(config.Database, "..") {
			return "", fmt.Errorf("invalid database path: path traversal not allowed")
		}
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