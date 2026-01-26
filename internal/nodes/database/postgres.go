/*
Package database provides database-related node implementations for n8n-go.
*/
package database

import (
	"database/sql"
	"fmt"
	
	_ "github.com/lib/pq" // PostgreSQL driver
	
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// PostgresNode implements the PostgreSQL node functionality
type PostgresNode struct {
	*base.BaseNode
}

// NewPostgresNode creates a new PostgreSQL node
func NewPostgresNode() *PostgresNode {
	description := base.NodeDescription{
		Name:        "PostgreSQL",
		Description: "Executes queries against PostgreSQL databases",
		Category:    "Database",
	}
	
	return &PostgresNode{
		BaseNode: base.NewBaseNode(description),
	}
}

// Description returns the node description
func (p *PostgresNode) Description() base.NodeDescription {
	return p.BaseNode.Description()
}

// ValidateParameters validates PostgreSQL node parameters
func (p *PostgresNode) ValidateParameters(params map[string]interface{}) error {
	if params == nil {
		return p.CreateError("parameters cannot be nil", nil)
	}
	
	// Check required parameters
	connectionURL := p.GetStringParameter(params, "connectionUrl", "")
	if connectionURL == "" {
		// Check individual connection parameters if connectionUrl is not provided
		host := p.GetStringParameter(params, "host", "")
		database := p.GetStringParameter(params, "database", "")
		
		if host == "" {
			return p.CreateError("either connectionUrl or host is required", nil)
		}
		
		if database == "" {
			return p.CreateError("database is required", nil)
		}
	}
	
	// Check if operation is provided
	operation := p.GetStringParameter(params, "operation", "")
	if operation == "" {
		return p.CreateError("operation is required", nil)
	}
	
	validOperations := map[string]bool{
		"executeQuery": true,
		"insert":       true,
		"update":       true,
		"delete":       true,
	}
	
	if !validOperations[operation] {
		return p.CreateError(fmt.Sprintf("invalid operation: %s", operation), nil)
	}
	
	// For executeQuery operation, query is required
	if operation == "executeQuery" {
		query := p.GetStringParameter(params, "query", "")
		if query == "" {
			return p.CreateError("query is required for executeQuery operation", nil)
		}
	}
	
	return nil
}

// Execute processes the PostgreSQL node operation
func (p *PostgresNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	if len(inputData) == 0 {
		return []model.DataItem{}, nil
	}
	
	// Get connection parameters
	connectionURL := p.GetStringParameter(nodeParams, "connectionUrl", "")
	
	// If connection URL is not provided, build it from individual parameters
	if connectionURL == "" {
		host := p.GetStringParameter(nodeParams, "host", "localhost")
		port := p.GetIntParameter(nodeParams, "port", 5432)
		database := p.GetStringParameter(nodeParams, "database", "")
		user := p.GetStringParameter(nodeParams, "user", "")
		password := p.GetStringParameter(nodeParams, "password", "")
		
		connectionURL = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", user, password, host, port, database)
	}
	
	// Connect to database
	db, err := sql.Open("postgres", connectionURL)
	if err != nil {
		return nil, p.CreateError(fmt.Sprintf("failed to connect to database: %v", err), nil)
	}
	defer db.Close()
	
	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, p.CreateError(fmt.Sprintf("failed to ping database: %v", err), nil)
	}
	
	// Get operation
	operation := p.GetStringParameter(nodeParams, "operation", "")
	
	// Process each input data item
	result := make([]model.DataItem, len(inputData))
	
	for i, item := range inputData {
		var newItem model.DataItem
		
		switch operation {
		case "executeQuery":
			queryResult, err := p.executeQuery(db, nodeParams, item)
			if err != nil {
				return nil, p.CreateError(fmt.Sprintf("failed to execute query: %v", err), nil)
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
func (p *PostgresNode) executeQuery(db *sql.DB, nodeParams map[string]interface{}, item model.DataItem) (model.DataItem, error) {
	query := p.GetStringParameter(nodeParams, "query", "")
	
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