/*
Package database provides database-related node implementations for n8n-go.
*/
package database

import (
	"database/sql"
	"fmt"
	
	_ "github.com/mattn/go-sqlite3" // SQLite driver
	
	"github.com/yourusername/n8n-go/internal/model"
	"github.com/yourusername/n8n-go/internal/nodes/base"
)

// SQLiteNode implements the SQLite node functionality
type SQLiteNode struct {
	*base.BaseNode
}

// NewSQLiteNode creates a new SQLite node
func NewSQLiteNode() *SQLiteNode {
	description := base.NodeDescription{
		Name:        "SQLite",
		Description: "Executes queries against SQLite databases",
		Category:    "Database",
	}
	
	return &SQLiteNode{
		BaseNode: base.NewBaseNode(description),
	}
}

// Description returns the node description
func (s *SQLiteNode) Description() base.NodeDescription {
	return s.BaseNode.Description()
}

// ValidateParameters validates SQLite node parameters
func (s *SQLiteNode) ValidateParameters(params map[string]interface{}) error {
	if params == nil {
		return s.CreateError("parameters cannot be nil", nil)
	}
	
	// Check required parameters
	filename := s.GetStringParameter(params, "filename", "")
	if filename == "" {
		return s.CreateError("filename is required", nil)
	}
	
	// Check if operation is provided
	operation := s.GetStringParameter(params, "operation", "")
	if operation == "" {
		return s.CreateError("operation is required", nil)
	}
	
	validOperations := map[string]bool{
		"executeQuery": true,
		"insert":       true,
		"update":       true,
		"delete":       true,
	}
	
	if !validOperations[operation] {
		return s.CreateError(fmt.Sprintf("invalid operation: %s", operation), nil)
	}
	
	// For executeQuery operation, query is required
	if operation == "executeQuery" {
		query := s.GetStringParameter(params, "query", "")
		if query == "" {
			return s.CreateError("query is required for executeQuery operation", nil)
		}
	}
	
	return nil
}

// Execute processes the SQLite node operation
func (s *SQLiteNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	if len(inputData) == 0 {
		return []model.DataItem{}, nil
	}
	
	// Get filename parameter
	filename := s.GetStringParameter(nodeParams, "filename", "")
	
	// Connect to database
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, s.CreateError(fmt.Sprintf("failed to connect to database: %v", err), nil)
	}
	defer db.Close()
	
	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, s.CreateError(fmt.Sprintf("failed to ping database: %v", err), nil)
	}
	
	// Get operation
	operation := s.GetStringParameter(nodeParams, "operation", "")
	
	// Process each input data item
	result := make([]model.DataItem, len(inputData))
	
	for i, item := range inputData {
		var newItem model.DataItem
		
		switch operation {
		case "executeQuery":
			queryResult, err := s.executeQuery(db, nodeParams, item)
			if err != nil {
				return nil, s.CreateError(fmt.Sprintf("failed to execute query: %v", err), nil)
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
func (s *SQLiteNode) executeQuery(db *sql.DB, nodeParams map[string]interface{}, item model.DataItem) (model.DataItem, error) {
	query := s.GetStringParameter(nodeParams, "query", "")
	
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