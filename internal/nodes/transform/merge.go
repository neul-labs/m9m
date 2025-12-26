package transform

import (
	"fmt"

	"github.com/dipankar/m9m/internal/expressions"
	"github.com/dipankar/m9m/internal/model"
	"github.com/dipankar/m9m/internal/nodes/base"
)

// MergeNode implements the Merge node functionality for combining data from multiple inputs
type MergeNode struct {
	*base.BaseNode
	evaluator *expressions.GojaExpressionEvaluator
}

// NewMergeNode creates a new Merge node
func NewMergeNode() *MergeNode {
	description := base.NodeDescription{
		Name:        "Merge",
		Description: "Merge data from multiple inputs",
		Category:    "Data Transformation",
	}

	return &MergeNode{
		BaseNode:  base.NewBaseNode(description),
		evaluator: expressions.NewGojaExpressionEvaluator(expressions.DefaultEvaluatorConfig()),
	}
}

// Execute processes the Merge node operation
func (m *MergeNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	if len(inputData) == 0 {
		return []model.DataItem{}, nil
	}

	// Get merge mode
	mode, ok := nodeParams["mode"].(string)
	if !ok {
		mode = "append" // Default mode
	}

	switch mode {
	case "append":
		return m.appendMerge(inputData, nodeParams)
	case "merge":
		return m.objectMerge(inputData, nodeParams)
	case "combine":
		return m.combineMerge(inputData, nodeParams)
	case "chooseBranch":
		return m.chooseBranchMerge(inputData, nodeParams)
	default:
		return nil, fmt.Errorf("unsupported merge mode: %s", mode)
	}
}

// appendMerge simply appends all input items
func (m *MergeNode) appendMerge(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	// In append mode, we just pass through all items
	return inputData, nil
}

// objectMerge merges all items into a single object
func (m *MergeNode) objectMerge(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	if len(inputData) == 0 {
		return []model.DataItem{}, nil
	}

	mergedData := make(map[string]interface{})

	// Get merge options
	overwriteExisting, _ := nodeParams["overwriteExisting"].(bool)

	for _, item := range inputData {
		for key, value := range item.JSON {
			// Check if key already exists and if we should overwrite
			if _, exists := mergedData[key]; exists && !overwriteExisting {
				continue
			}
			mergedData[key] = value
		}
	}

	result := model.DataItem{
		JSON: mergedData,
	}

	return []model.DataItem{result}, nil
}

// combineMerge combines items by creating arrays of values
func (m *MergeNode) combineMerge(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	if len(inputData) == 0 {
		return []model.DataItem{}, nil
	}

	combinedData := make(map[string][]interface{})

	// Collect all unique keys
	allKeys := make(map[string]bool)
	for _, item := range inputData {
		for key := range item.JSON {
			allKeys[key] = true
		}
	}

	// Create arrays for each key
	for key := range allKeys {
		combinedData[key] = make([]interface{}, 0)
	}

	// Fill arrays with values from each item
	for _, item := range inputData {
		for key := range allKeys {
			if value, exists := item.JSON[key]; exists {
				combinedData[key] = append(combinedData[key], value)
			} else {
				combinedData[key] = append(combinedData[key], nil)
			}
		}
	}

	// Convert []interface{} to interface{} for JSON compatibility
	result := make(map[string]interface{})
	for key, values := range combinedData {
		result[key] = values
	}

	return []model.DataItem{{JSON: result}}, nil
}

// chooseBranchMerge selects items based on conditions
func (m *MergeNode) chooseBranchMerge(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	// Get branch selection criteria
	branchCondition, _ := nodeParams["branchCondition"].(string)
	if branchCondition == "" {
		// If no condition, return all items
		return inputData, nil
	}

	var result []model.DataItem

	for index, item := range inputData {
		context := &expressions.ExpressionContext{
			ActiveNodeName:      "Merge",
			RunIndex:           0,
			ItemIndex:          index,
			Mode:               expressions.ModeManual,
			ConnectionInputData: []model.DataItem{item},
			Workflow: &model.Workflow{
				Name: "Merge Processing",
			},
			AdditionalKeys: &expressions.AdditionalKeys{
				ExecutionId: "merge-processing",
			},
		}

		// Evaluate the condition
		conditionResult, err := m.evaluator.EvaluateExpression(branchCondition, context)
		if err != nil {
			// If evaluation fails, skip the item
			continue
		}

		// Check if condition is truthy
		if isTruthy(conditionResult) {
			result = append(result, item)
		}
	}

	return result, nil
}

// isTruthy checks if a value is considered true in JavaScript context
func isTruthy(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v != ""
	case int, int64, float64:
		return v != 0
	case []interface{}:
		return len(v) > 0
	case map[string]interface{}:
		return len(v) > 0
	case nil:
		return false
	default:
		return true
	}
}

// ValidateParameters validates Merge node parameters
func (m *MergeNode) ValidateParameters(params map[string]interface{}) error {
	mode, ok := params["mode"].(string)
	if !ok {
		return fmt.Errorf("mode parameter is required")
	}

	validModes := map[string]bool{
		"append": true, "merge": true, "combine": true, "chooseBranch": true,
	}

	if !validModes[mode] {
		return fmt.Errorf("invalid merge mode: %s", mode)
	}

	// Validate mode-specific parameters
	if mode == "chooseBranch" {
		if _, ok := params["branchCondition"]; !ok {
			return fmt.Errorf("branchCondition parameter is required for chooseBranch mode")
		}
	}

	return nil
}