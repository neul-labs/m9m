/*
Package transform provides data transformation node implementations for n8n-go.
*/
package transform

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	
	"github.com/yourusername/n8n-go/internal/model"
	"github.com/yourusername/n8n-go/internal/nodes/base"
)

// FilterNode implements the Filter node functionality for filtering data items
type FilterNode struct {
	*base.BaseNode
}

// NewFilterNode creates a new Filter node
func NewFilterNode() *FilterNode {
	description := base.NodeDescription{
		Name:        "Filter",
		Description: "Filters items based on conditions",
		Category:    "Data Transformation",
	}
	
	return &FilterNode{
		BaseNode: base.NewBaseNode(description),
	}
}

// Description returns the node description
func (f *FilterNode) Description() base.NodeDescription {
	return f.BaseNode.Description()
}

// ValidateParameters validates Filter node parameters
func (f *FilterNode) ValidateParameters(params map[string]interface{}) error {
	if params == nil {
		return f.CreateError("parameters cannot be nil", nil)
	}
	
	// Check if conditions exist
	conditions, ok := params["conditions"]
	if !ok {
		return f.CreateError("conditions parameter is required", nil)
	}
	
	// Check if conditions is an array
	conditionsArr, ok := conditions.([]interface{})
	if !ok {
		return f.CreateError("conditions must be an array", nil)
	}
	
	// Validate each condition
	for i, condition := range conditionsArr {
		conditionMap, ok := condition.(map[string]interface{})
		if !ok {
			return f.CreateError(fmt.Sprintf("condition %d must be an object", i), nil)
		}
		
		// Check required fields
		if _, ok := conditionMap["leftValue"]; !ok {
			return f.CreateError(fmt.Sprintf("condition %d missing 'leftValue' field", i), nil)
		}
		
		if _, ok := conditionMap["rightValue"]; !ok {
			return f.CreateError(fmt.Sprintf("condition %d missing 'rightValue' field", i), nil)
		}
		
		if _, ok := conditionMap["operator"]; !ok {
			return f.CreateError(fmt.Sprintf("condition %d missing 'operator' field", i), nil)
		}
		
		// Validate operator
		operator, ok := conditionMap["operator"].(string)
		if !ok {
			return f.CreateError(fmt.Sprintf("condition %d operator must be a string", i), nil)
		}
		
		validOperators := map[string]bool{
			"equals":           true,
			"notEquals":        true,
			"contains":         true,
			"notContains":      true,
			"startsWith":       true,
			"endsWith":         true,
			"regex":            true,
			"exists":           true,
			"notExists":        true,
			"greaterThan":      true,
			"lessThan":         true,
			"greaterThanOrEqual": true,
			"lessThanOrEqual":    true,
			"between":          true,
			"empty":            true,
			"notEmpty":         true,
		}
		
		if !validOperators[operator] {
			return f.CreateError(fmt.Sprintf("condition %d has invalid operator: %s", i, operator), nil)
		}
	}
	
	// Check combiner
	combiner := f.GetStringParameter(params, "combiner", "and")
	if combiner != "and" && combiner != "or" {
		return f.CreateError("combiner must be 'and' or 'or'", nil)
	}
	
	return nil
}

// Execute processes the Filter node operation
func (f *FilterNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	if len(inputData) == 0 {
		return []model.DataItem{}, nil
	}
	
	// Get conditions from node parameters
	conditions, ok := nodeParams["conditions"]
	if !ok {
		return nil, f.CreateError("conditions parameter is required", nil)
	}
	
	conditionsArr, ok := conditions.([]interface{})
	if !ok {
		return nil, f.CreateError("conditions must be an array", nil)
	}
	
	// Get combiner from node parameters
	combiner := f.GetStringParameter(nodeParams, "combiner", "and")
	
	// Process each input data item
	var result []model.DataItem
	
	for _, item := range inputData {
		// Evaluate conditions for this item
		passesFilter := f.evaluateConditions(item, conditionsArr, combiner)
		
		// Add to result if it passes the filter
		if passesFilter {
			result = append(result, item)
		}
	}
	
	return result, nil
}

// evaluateConditions evaluates conditions for a data item
func (f *FilterNode) evaluateConditions(item model.DataItem, conditions []interface{}, combiner string) bool {
	if len(conditions) == 0 {
		return true // No conditions means pass all items
	}
	
	// Evaluate each condition
	results := make([]bool, len(conditions))
	
	for i, condition := range conditions {
		conditionMap, ok := condition.(map[string]interface{})
		if !ok {
			results[i] = false
			continue
		}
		
		results[i] = f.evaluateCondition(item, conditionMap)
	}
	
	// Combine results based on combiner
	if combiner == "and" {
		// All conditions must be true
		for _, result := range results {
			if !result {
				return false
			}
		}
		return true
	} else {
		// At least one condition must be true
		for _, result := range results {
			if result {
				return true
			}
		}
		return false
	}
}

// evaluateCondition evaluates a single condition for a data item
func (f *FilterNode) evaluateCondition(item model.DataItem, condition map[string]interface{}) bool {
	leftValue := condition["leftValue"]
	rightValue := condition["rightValue"]
	operator := f.GetStringParameter(condition, "operator", "equals")
	
	// Resolve left value from item JSON
	leftResolved := f.resolveValue(item.JSON, leftValue)
	
	// Resolve right value
	rightResolved := rightValue
	
	// Handle special operators that don't need right value
	switch operator {
	case "exists":
		return leftResolved != nil
	case "notExists":
		return leftResolved == nil
	case "empty":
		return f.isEmpty(leftResolved)
	case "notEmpty":
		return !f.isEmpty(leftResolved)
	}
	
	// For other operators, we need both values
	if leftResolved == nil || rightResolved == nil {
		return false
	}
	
	// Compare values based on operator
	switch operator {
	case "equals":
		return f.equals(leftResolved, rightResolved)
	case "notEquals":
		return !f.equals(leftResolved, rightResolved)
	case "contains":
		return f.contains(leftResolved, rightResolved)
	case "notContains":
		return !f.contains(leftResolved, rightResolved)
	case "startsWith":
		return f.startsWith(leftResolved, rightResolved)
	case "endsWith":
		return f.endsWith(leftResolved, rightResolved)
	case "regex":
		return f.regexMatch(leftResolved, rightResolved)
	case "greaterThan":
		return f.greaterThan(leftResolved, rightResolved)
	case "lessThan":
		return f.lessThan(leftResolved, rightResolved)
	case "greaterThanOrEqual":
		return f.greaterThanOrEqual(leftResolved, rightResolved)
	case "lessThanOrEqual":
		return f.lessThanOrEqual(leftResolved, rightResolved)
	case "between":
		return f.between(leftResolved, rightResolved)
	default:
		return false
	}
}

// resolveValue resolves a value from JSON data
func (f *FilterNode) resolveValue(data map[string]interface{}, value interface{}) interface{} {
	// If value is a string that starts with $json., resolve it from data
	if strValue, ok := value.(string); ok && strings.HasPrefix(strValue, "$json.") {
		path := strings.TrimPrefix(strValue, "$json.")
		return f.getValueAtPath(data, path)
	}
	
	// Return the value as-is
	return value
}

// getValueAtPath gets a value at a dot-separated path in a map
func (f *FilterNode) getValueAtPath(data map[string]interface{}, path string) interface{} {
	if path == "" {
		return data
	}
	
	// Split the path by dots
	parts := strings.Split(path, ".")
	current := data
	
	// Navigate through the path
	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part - return the value
			return current[part]
		} else {
			// Intermediate part - should be a map
			if next, ok := current[part].(map[string]interface{}); ok {
				current = next
			} else {
				return nil
			}
		}
	}
	
	return nil
}

// isEmpty checks if a value is empty
func (f *FilterNode) isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}
	
	switch v := value.(type) {
	case string:
		return v == ""
	case []interface{}:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	case float64:
		return v == 0
	case bool:
		return !v
	default:
		return false
	}
}

// equals checks if two values are equal
func (f *FilterNode) equals(left, right interface{}) bool {
	// Convert both values to strings for comparison
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)
	
	return leftStr == rightStr
}

// contains checks if left value contains right value
func (f *FilterNode) contains(left, right interface{}) bool {
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)
	
	return strings.Contains(leftStr, rightStr)
}

// startsWith checks if left value starts with right value
func (f *FilterNode) startsWith(left, right interface{}) bool {
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)
	
	return strings.HasPrefix(leftStr, rightStr)
}

// endsWith checks if left value ends with right value
func (f *FilterNode) endsWith(left, right interface{}) bool {
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)
	
	return strings.HasSuffix(leftStr, rightStr)
}

// regexMatch checks if left value matches right regex pattern
func (f *FilterNode) regexMatch(left, right interface{}) bool {
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)
	
	match, err := regexp.MatchString(rightStr, leftStr)
	if err != nil {
		return false
	}
	
	return match
}

// greaterThan checks if left value is greater than right value
func (f *FilterNode) greaterThan(left, right interface{}) bool {
	leftNum, leftOk := f.toFloat64(left)
	rightNum, rightOk := f.toFloat64(right)
	
	if !leftOk || !rightOk {
		return false
	}
	
	return leftNum > rightNum
}

// lessThan checks if left value is less than right value
func (f *FilterNode) lessThan(left, right interface{}) bool {
	leftNum, leftOk := f.toFloat64(left)
	rightNum, rightOk := f.toFloat64(right)
	
	if !leftOk || !rightOk {
		return false
	}
	
	return leftNum < rightNum
}

// greaterThanOrEqual checks if left value is greater than or equal to right value
func (f *FilterNode) greaterThanOrEqual(left, right interface{}) bool {
	leftNum, leftOk := f.toFloat64(left)
	rightNum, rightOk := f.toFloat64(right)
	
	if !leftOk || !rightOk {
		return false
	}
	
	return leftNum >= rightNum
}

// lessThanOrEqual checks if left value is less than or equal to right value
func (f *FilterNode) lessThanOrEqual(left, right interface{}) bool {
	leftNum, leftOk := f.toFloat64(left)
	rightNum, rightOk := f.toFloat64(right)
	
	if !leftOk || !rightOk {
		return false
	}
	
	return leftNum <= rightNum
}

// between checks if left value is between two values (assumes right is a range)
func (f *FilterNode) between(left, right interface{}) bool {
	// For simplicity, we'll assume right is a string with format "min,max"
	rightStr := fmt.Sprintf("%v", right)
	parts := strings.Split(rightStr, ",")
	
	if len(parts) != 2 {
		return false
	}
	
	min, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	max, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	
	if err1 != nil || err2 != nil {
		return false
	}
	
	leftNum, leftOk := f.toFloat64(left)
	if !leftOk {
		return false
	}
	
	return leftNum >= min && leftNum <= max
}

// toFloat64 converts a value to float64
func (f *FilterNode) toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case float64:
		return v, true
	case string:
		if num, err := strconv.ParseFloat(v, 64); err == nil {
			return num, true
		}
	}
	
	return 0, false
}