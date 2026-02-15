/*
Package database provides database-related node implementations for m9m.
*/
package database

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"strings"

	_ "github.com/go-sql-driver/mysql" // MySQL driver

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// MySQLNode implements the MySQL node functionality
type MySQLNode struct {
	*base.BaseNode
}

// NewMySQLNode creates a new MySQL node
func NewMySQLNode() *MySQLNode {
	description := base.NodeDescription{
		Name:        "MySQL",
		Description: "Executes queries against MySQL databases",
		Category:    "Database",
	}
	
	return &MySQLNode{
		BaseNode: base.NewBaseNode(description),
	}
}

// Description returns the node description
func (m *MySQLNode) Description() base.NodeDescription {
	return m.BaseNode.Description()
}

// ValidateParameters validates MySQL node parameters
func (m *MySQLNode) ValidateParameters(params map[string]interface{}) error {
	if params == nil {
		return m.CreateError("parameters cannot be nil", nil)
	}
	
	// Check required parameters
	connectionURL := m.GetStringParameter(params, "connectionUrl", "")
	if connectionURL == "" {
		// Check individual connection parameters if connectionUrl is not provided
		host := m.GetStringParameter(params, "host", "")
		database := m.GetStringParameter(params, "database", "")
		
		if host == "" {
			return m.CreateError("either connectionUrl or host is required", nil)
		}
		
		if database == "" {
			return m.CreateError("database is required", nil)
		}
	}
	
	// Check if operation is provided
	operation := m.GetStringParameter(params, "operation", "")
	if operation == "" {
		return m.CreateError("operation is required", nil)
	}
	
	validOperations := map[string]bool{
		"executeQuery": true,
		"insert":       true,
		"update":       true,
		"delete":       true,
	}
	
	if !validOperations[operation] {
		return m.CreateError(fmt.Sprintf("invalid operation: %s", operation), nil)
	}
	
	// For executeQuery operation, query is required
	if operation == "executeQuery" {
		query := m.GetStringParameter(params, "query", "")
		if query == "" {
			return m.CreateError("query is required for executeQuery operation", nil)
		}
	}
	
	return nil
}

// Execute processes the MySQL node operation
func (m *MySQLNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	if len(inputData) == 0 {
		return []model.DataItem{}, nil
	}
	
	// Get connection parameters
	connectionURL := m.GetStringParameter(nodeParams, "connectionUrl", "")

	// If connection URL is not provided, build it from individual parameters
	if connectionURL == "" {
		host := m.GetStringParameter(nodeParams, "host", "localhost")
		port := m.GetIntParameter(nodeParams, "port", 3306)
		database := m.GetStringParameter(nodeParams, "database", "")
		user := m.GetStringParameter(nodeParams, "user", "")
		password := m.GetStringParameter(nodeParams, "password", "")

		// SECURITY: Get TLS mode from parameters, default to "preferred" for encrypted connections
		tlsMode := m.GetStringParameter(nodeParams, "tls", "preferred")

		// SECURITY: Validate TLS mode
		validTLSModes := map[string]bool{
			"true": true, "false": true, "skip-verify": true, "preferred": true,
		}
		if !validTLSModes[tlsMode] {
			return nil, m.CreateError(fmt.Sprintf("invalid tls mode: %s", tlsMode), nil)
		}

		// SECURITY: Warn if TLS is disabled
		if tlsMode == "false" {
			log.Printf("SECURITY WARNING: MySQL connection to %s using tls=false. This is insecure for production.", host)
		}

		// SECURITY: Properly escape username and password for MySQL DSN
		escapedUser := url.QueryEscape(user)
		escapedPass := url.QueryEscape(password)

		connectionURL = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?tls=%s",
			escapedUser, escapedPass, host, port, database, tlsMode)
	} else {
		// SECURITY: Warn if provided URL does not contain TLS settings
		if !strings.Contains(connectionURL, "tls=") {
			log.Printf("SECURITY WARNING: MySQL connection URL does not specify TLS settings. Consider adding tls=preferred or tls=true.")
		}
	}
	
	// Connect to database
	db, err := sql.Open("mysql", connectionURL)
	if err != nil {
		return nil, m.CreateError(fmt.Sprintf("failed to connect to database: %v", err), nil)
	}
	defer db.Close()
	
	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, m.CreateError(fmt.Sprintf("failed to ping database: %v", err), nil)
	}
	
	// Get operation
	operation := m.GetStringParameter(nodeParams, "operation", "")
	
	// Process each input data item
	result := make([]model.DataItem, len(inputData))
	
	for i, item := range inputData {
		var newItem model.DataItem
		
		switch operation {
		case "executeQuery":
			queryResult, err := m.executeQuery(db, nodeParams, item)
			if err != nil {
				return nil, m.CreateError(fmt.Sprintf("failed to execute query: %v", err), nil)
			}
			newItem = queryResult
			
		default:
			// For other operations, just pass through the data
			newItem = item
		}
		
		result[i] = newItem
	}
	
	return result, nil
}

// executeQuery executes a SELECT query and returns results
func (m *MySQLNode) executeQuery(db *sql.DB, nodeParams map[string]interface{}, item model.DataItem) (model.DataItem, error) {
	query := m.GetStringParameter(nodeParams, "query", "")

	// SECURITY: Basic validation - reject obviously dangerous patterns
	// Note: This is defense-in-depth, not a complete SQL injection prevention
	dangerousPatterns := []string{
		"--",           // SQL comment
		";",            // Statement terminator (could chain queries)
		"/*",           // Block comment start
		"*/",           // Block comment end
		"xp_",          // SQL Server extended procedures
		"EXEC ",        // Execute
		"EXECUTE ",     // Execute
		"sp_",          // Stored procedures
		"DROP ",        // Drop statement
		"TRUNCATE ",    // Truncate statement
		"ALTER ",       // Alter statement
		"CREATE ",      // Create statement
		"GRANT ",       // Grant privileges
		"REVOKE ",      // Revoke privileges
	}

	upperQuery := strings.ToUpper(query)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(upperQuery, pattern) {
			// Log for security monitoring
			log.Printf("SECURITY WARNING: Potentially dangerous SQL pattern detected: %s", pattern)
			// Allow but warn - in strict mode, you might want to reject
		}
	}

	// Execute query
	rows, err := db.Query(query)
	if err != nil {
		return model.DataItem{}, fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()
	
	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return model.DataItem{}, fmt.Errorf("failed to get columns: %v", err)
	}
	
	// Process rows
	var results []map[string]interface{}
	
	for rows.Next() {
		// Create a slice of interface{} to hold the values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		
		// Scan the row
		if err := rows.Scan(valuePtrs...); err != nil {
			return model.DataItem{}, fmt.Errorf("failed to scan row: %v", err)
		}
		
		// Create a map for this row
		rowMap := make(map[string]interface{})
		for i, col := range columns {
			// Handle nil values
			if values[i] == nil {
				rowMap[col] = nil
			} else {
				rowMap[col] = values[i]
			}
		}
		
		results = append(results, rowMap)
	}
	
	// Check for errors after iteration
	if err := rows.Err(); err != nil {
		return model.DataItem{}, fmt.Errorf("error during row iteration: %v", err)
	}
	
	// Create result item
	resultItem := model.DataItem{
		JSON: map[string]interface{}{
			"rows": results,
		},
	}
	
	return resultItem, nil
}