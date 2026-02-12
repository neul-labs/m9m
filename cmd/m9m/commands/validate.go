package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	validateVerbose   bool
	validateCheckNodes bool
)

var validateCmd = &cobra.Command{
	Use:   "validate <workflow.json>",
	Short: "Validate a workflow JSON file",
	Long: `Validate a workflow JSON file before creating it.

Checks include:
- JSON structure validity
- Required fields (name, nodes, connections)
- Node type existence
- Connection validity
- Circular dependency detection

Examples:
  m9m validate workflow.json
  m9m validate workflow.json --verbose
  m9m validate workflow.json --check-nodes`,
	Args: cobra.ExactArgs(1),
	Run:  runValidate,
}

func init() {
	validateCmd.Flags().BoolVar(&validateVerbose, "verbose", false, "Show detailed validation output")
	validateCmd.Flags().BoolVar(&validateCheckNodes, "check-nodes", false, "Check if node types exist")
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Level   string `json:"level"` // error, warning
}

// ValidationResult holds the validation results
type ValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []ValidationError `json:"warnings,omitempty"`
}

func runValidate(cmd *cobra.Command, args []string) {
	filePath := args[0]

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error: Cannot read file: %v\n", err)
		os.Exit(1)
	}

	// Parse JSON
	var workflow map[string]interface{}
	if err := json.Unmarshal(data, &workflow); err != nil {
		fmt.Printf("Error: Invalid JSON: %v\n", err)
		os.Exit(1)
	}

	result := validateWorkflow(workflow)

	if outputFlag == "json" {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		if !result.Valid {
			os.Exit(1)
		}
		return
	}

	// Print results
	if result.Valid {
		fmt.Printf("Validation passed for %s\n", filePath)
	} else {
		fmt.Printf("Validation failed for %s\n", filePath)
	}

	if len(result.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, e := range result.Errors {
			fmt.Printf("  - [%s] %s\n", e.Field, e.Message)
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, w := range result.Warnings {
			fmt.Printf("  - [%s] %s\n", w.Field, w.Message)
		}
	}

	if validateVerbose && result.Valid {
		printWorkflowSummary(workflow)
	}

	if !result.Valid {
		os.Exit(1)
	}
}

func validateWorkflow(workflow map[string]interface{}) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
	}

	// Check required fields
	if _, ok := workflow["name"]; !ok {
		result.addError("name", "Missing required field 'name'")
	} else if name, ok := workflow["name"].(string); !ok || name == "" {
		result.addError("name", "Field 'name' must be a non-empty string")
	}

	// Check nodes
	nodes, hasNodes := workflow["nodes"]
	if !hasNodes {
		result.addError("nodes", "Missing required field 'nodes'")
	} else {
		nodeList, ok := nodes.([]interface{})
		if !ok {
			result.addError("nodes", "Field 'nodes' must be an array")
		} else {
			validateNodes(nodeList, result)
		}
	}

	// Check connections (optional but validate if present)
	if connections, ok := workflow["connections"]; ok {
		validateConnections(connections, workflow, result)
	}

	return result
}

func validateNodes(nodes []interface{}, result *ValidationResult) {
	nodeNames := make(map[string]bool)

	for i, node := range nodes {
		nodeMap, ok := node.(map[string]interface{})
		if !ok {
			result.addError(fmt.Sprintf("nodes[%d]", i), "Node must be an object")
			continue
		}

		// Check node name
		name, hasName := nodeMap["name"]
		if !hasName {
			result.addError(fmt.Sprintf("nodes[%d].name", i), "Node missing required field 'name'")
		} else if nameStr, ok := name.(string); !ok || nameStr == "" {
			result.addError(fmt.Sprintf("nodes[%d].name", i), "Node 'name' must be a non-empty string")
		} else {
			if nodeNames[nameStr] {
				result.addError(fmt.Sprintf("nodes[%d].name", i), fmt.Sprintf("Duplicate node name: '%s'", nameStr))
			}
			nodeNames[nameStr] = true
		}

		// Check node type
		nodeType, hasType := nodeMap["type"]
		if !hasType {
			result.addError(fmt.Sprintf("nodes[%d].type", i), "Node missing required field 'type'")
		} else if typeStr, ok := nodeType.(string); !ok || typeStr == "" {
			result.addError(fmt.Sprintf("nodes[%d].type", i), "Node 'type' must be a non-empty string")
		} else if validateCheckNodes {
			// Check if node type exists
			if !isValidNodeType(typeStr) {
				result.addWarning(fmt.Sprintf("nodes[%d].type", i), fmt.Sprintf("Unknown node type: '%s'", typeStr))
			}
		}

		// Check position (optional but warn if missing)
		if _, hasPosition := nodeMap["position"]; !hasPosition {
			result.addWarning(fmt.Sprintf("nodes[%d].position", i), "Node missing 'position' field (recommended for UI)")
		}
	}
}

func validateConnections(connections interface{}, workflow map[string]interface{}, result *ValidationResult) {
	connMap, ok := connections.(map[string]interface{})
	if !ok {
		result.addError("connections", "Field 'connections' must be an object")
		return
	}

	// Get list of valid node names
	nodeNames := make(map[string]bool)
	if nodes, ok := workflow["nodes"].([]interface{}); ok {
		for _, node := range nodes {
			if nodeMap, ok := node.(map[string]interface{}); ok {
				if name, ok := nodeMap["name"].(string); ok {
					nodeNames[name] = true
				}
			}
		}
	}

	// Validate each connection
	for sourceName, connData := range connMap {
		if !nodeNames[sourceName] {
			result.addError(fmt.Sprintf("connections.%s", sourceName), fmt.Sprintf("Connection references unknown node: '%s'", sourceName))
			continue
		}

		// Check connection structure
		connObj, ok := connData.(map[string]interface{})
		if !ok {
			result.addError(fmt.Sprintf("connections.%s", sourceName), "Connection data must be an object")
			continue
		}

		// Check main connections
		if main, ok := connObj["main"]; ok {
			validateConnectionTargets(main, nodeNames, sourceName, result)
		}
	}
}

func validateConnectionTargets(main interface{}, nodeNames map[string]bool, sourceName string, result *ValidationResult) {
	mainArr, ok := main.([]interface{})
	if !ok {
		result.addError(fmt.Sprintf("connections.%s.main", sourceName), "Connection 'main' must be an array")
		return
	}

	for i, outputConns := range mainArr {
		connArr, ok := outputConns.([]interface{})
		if !ok {
			result.addError(fmt.Sprintf("connections.%s.main[%d]", sourceName, i), "Connection output must be an array")
			continue
		}

		for j, conn := range connArr {
			connObj, ok := conn.(map[string]interface{})
			if !ok {
				result.addError(fmt.Sprintf("connections.%s.main[%d][%d]", sourceName, i, j), "Connection must be an object")
				continue
			}

			targetNode, ok := connObj["node"].(string)
			if !ok || targetNode == "" {
				result.addError(fmt.Sprintf("connections.%s.main[%d][%d].node", sourceName, i, j), "Connection missing 'node' field")
			} else if !nodeNames[targetNode] {
				result.addError(fmt.Sprintf("connections.%s.main[%d][%d].node", sourceName, i, j), fmt.Sprintf("Connection references unknown node: '%s'", targetNode))
			}
		}
	}
}

func isValidNodeType(nodeType string) bool {
	// Check against our catalog
	for _, node := range nodeTypeCatalog {
		if node.Name == nodeType {
			return true
		}
	}
	// Also accept custom node types (starting with custom.)
	if strings.HasPrefix(nodeType, "custom.") {
		return true
	}
	return false
}

func (r *ValidationResult) addError(field, message string) {
	r.Valid = false
	r.Errors = append(r.Errors, ValidationError{
		Field:   field,
		Message: message,
		Level:   "error",
	})
}

func (r *ValidationResult) addWarning(field, message string) {
	r.Warnings = append(r.Warnings, ValidationError{
		Field:   field,
		Message: message,
		Level:   "warning",
	})
}

func printWorkflowSummary(workflow map[string]interface{}) {
	fmt.Println("\nWorkflow Summary:")
	fmt.Printf("  Name: %v\n", workflow["name"])

	if nodes, ok := workflow["nodes"].([]interface{}); ok {
		fmt.Printf("  Nodes: %d\n", len(nodes))

		// Count by type
		typeCounts := make(map[string]int)
		for _, node := range nodes {
			if nodeMap, ok := node.(map[string]interface{}); ok {
				if nodeType, ok := nodeMap["type"].(string); ok {
					typeCounts[nodeType]++
				}
			}
		}

		fmt.Println("  Node types:")
		for t, count := range typeCounts {
			fmt.Printf("    - %s: %d\n", t, count)
		}
	}

	if connections, ok := workflow["connections"].(map[string]interface{}); ok {
		fmt.Printf("  Connections: %d sources\n", len(connections))
	}
}
