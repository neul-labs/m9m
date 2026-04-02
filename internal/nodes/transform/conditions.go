package transform

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/neul-labs/m9m/internal/model"
)

// ConditionEvaluator provides shared condition evaluation logic for Filter and IF nodes.
type ConditionEvaluator struct{}

// ValidOperators is the set of supported condition operators.
var ValidOperators = map[string]bool{
	"equals":             true,
	"notEquals":          true,
	"contains":           true,
	"notContains":        true,
	"startsWith":         true,
	"endsWith":           true,
	"regex":              true,
	"exists":             true,
	"notExists":          true,
	"greaterThan":        true,
	"lessThan":           true,
	"greaterThanOrEqual": true,
	"lessThanOrEqual":    true,
	"between":            true,
	"empty":              true,
	"notEmpty":           true,
}

// ValidateConditions validates the conditions array and combiner.
func ValidateConditions(conditions interface{}, combiner string) error {
	conditionsArr, ok := conditions.([]interface{})
	if !ok {
		return fmt.Errorf("conditions must be an array")
	}

	for i, condition := range conditionsArr {
		conditionMap, ok := condition.(map[string]interface{})
		if !ok {
			return fmt.Errorf("condition %d must be an object", i)
		}

		if _, ok := conditionMap["leftValue"]; !ok {
			return fmt.Errorf("condition %d missing 'leftValue' field", i)
		}
		if _, ok := conditionMap["rightValue"]; !ok {
			return fmt.Errorf("condition %d missing 'rightValue' field", i)
		}
		if _, ok := conditionMap["operator"]; !ok {
			return fmt.Errorf("condition %d missing 'operator' field", i)
		}

		operator, ok := conditionMap["operator"].(string)
		if !ok {
			return fmt.Errorf("condition %d operator must be a string", i)
		}
		if !ValidOperators[operator] {
			return fmt.Errorf("condition %d has invalid operator: %s", i, operator)
		}
	}

	if combiner != "and" && combiner != "or" {
		return fmt.Errorf("combiner must be 'and' or 'or'")
	}

	return nil
}

// EvaluateConditions evaluates all conditions against a data item.
func EvaluateConditions(item model.DataItem, conditions []interface{}, combiner string) bool {
	if len(conditions) == 0 {
		return true
	}

	for _, condition := range conditions {
		conditionMap, ok := condition.(map[string]interface{})
		if !ok {
			if combiner == "and" {
				return false
			}
			continue
		}

		result := evaluateCondition(item, conditionMap)

		if combiner == "and" && !result {
			return false
		}
		if combiner == "or" && result {
			return true
		}
	}

	return combiner == "and"
}

func evaluateCondition(item model.DataItem, condition map[string]interface{}) bool {
	leftValue := condition["leftValue"]
	rightValue := condition["rightValue"]
	operator, _ := condition["operator"].(string)
	if operator == "" {
		operator = "equals"
	}

	leftResolved := resolveValue(item.JSON, leftValue)
	rightResolved := rightValue

	switch operator {
	case "exists":
		return leftResolved != nil
	case "notExists":
		return leftResolved == nil
	case "empty":
		return condIsEmpty(leftResolved)
	case "notEmpty":
		return !condIsEmpty(leftResolved)
	}

	if leftResolved == nil || rightResolved == nil {
		return false
	}

	switch operator {
	case "equals":
		return condEquals(leftResolved, rightResolved)
	case "notEquals":
		return !condEquals(leftResolved, rightResolved)
	case "contains":
		return condContains(leftResolved, rightResolved)
	case "notContains":
		return !condContains(leftResolved, rightResolved)
	case "startsWith":
		return strings.HasPrefix(fmt.Sprintf("%v", leftResolved), fmt.Sprintf("%v", rightResolved))
	case "endsWith":
		return strings.HasSuffix(fmt.Sprintf("%v", leftResolved), fmt.Sprintf("%v", rightResolved))
	case "regex":
		match, err := regexp.MatchString(fmt.Sprintf("%v", rightResolved), fmt.Sprintf("%v", leftResolved))
		return err == nil && match
	case "greaterThan":
		return condCompare(leftResolved, rightResolved, func(l, r float64) bool { return l > r })
	case "lessThan":
		return condCompare(leftResolved, rightResolved, func(l, r float64) bool { return l < r })
	case "greaterThanOrEqual":
		return condCompare(leftResolved, rightResolved, func(l, r float64) bool { return l >= r })
	case "lessThanOrEqual":
		return condCompare(leftResolved, rightResolved, func(l, r float64) bool { return l <= r })
	case "between":
		return condBetween(leftResolved, rightResolved)
	default:
		return false
	}
}

func resolveValue(data map[string]interface{}, value interface{}) interface{} {
	strValue, ok := value.(string)
	if !ok || !strings.HasPrefix(strValue, "$json.") {
		return value
	}
	path := strings.TrimPrefix(strValue, "$json.")
	return getValueAtPath(data, path)
}

func getValueAtPath(data map[string]interface{}, path string) interface{} {
	if path == "" {
		return data
	}
	parts := strings.Split(path, ".")
	current := data
	for i, part := range parts {
		if i == len(parts)-1 {
			return current[part]
		}
		next, ok := current[part].(map[string]interface{})
		if !ok {
			return nil
		}
		current = next
	}
	return nil
}

func condIsEmpty(value interface{}) bool {
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

func condEquals(left, right interface{}) bool {
	return fmt.Sprintf("%v", left) == fmt.Sprintf("%v", right)
}

func condContains(left, right interface{}) bool {
	return strings.Contains(fmt.Sprintf("%v", left), fmt.Sprintf("%v", right))
}

func condCompare(left, right interface{}, cmp func(float64, float64) bool) bool {
	l, lok := condToFloat64(left)
	r, rok := condToFloat64(right)
	if !lok || !rok {
		return false
	}
	return cmp(l, r)
}

func condToFloat64(value interface{}) (float64, bool) {
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

func condBetween(left, right interface{}) bool {
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
	leftNum, ok := condToFloat64(left)
	if !ok {
		return false
	}
	return leftNum >= min && leftNum <= max
}
