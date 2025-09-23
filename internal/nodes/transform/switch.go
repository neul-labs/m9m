package transform

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/yourusername/n8n-go/internal/expressions"
	"github.com/yourusername/n8n-go/internal/model"
	"github.com/yourusername/n8n-go/internal/nodes/base"
)

// SwitchNode implements the Switch node functionality for conditional routing
type SwitchNode struct {
	*base.BaseNode
	evaluator *expressions.GojaExpressionEvaluator
}

// NewSwitchNode creates a new Switch node
func NewSwitchNode() *SwitchNode {
	description := base.NodeDescription{
		Name:        "Switch",
		Description: "Route data based on conditions",
		Category:    "Flow Control",
	}

	return &SwitchNode{
		BaseNode:  base.NewBaseNode(description),
		evaluator: expressions.NewGojaExpressionEvaluator(expressions.DefaultEvaluatorConfig()),
	}
}

// Execute processes the Switch node operation
func (s *SwitchNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	if len(inputData) == 0 {
		return []model.DataItem{}, nil
	}

	// Get routing rules
	rules, ok := nodeParams["rules"].([]interface{})
	if !ok || len(rules) == 0 {
		return nil, fmt.Errorf("switch rules are required")
	}

	var outputData []model.DataItem

	for index, item := range inputData {
		context := &expressions.ExpressionContext{
			ActiveNodeName:      "Switch",
			RunIndex:           0,
			ItemIndex:          index,
			Mode:               expressions.ModeManual,
			ConnectionInputData: []model.DataItem{item},
			Workflow: &model.Workflow{
				Name: "Switch Processing",
			},
			AdditionalKeys: &expressions.AdditionalKeys{
				ExecutionId: "switch-processing",
			},
		}

		// Check each rule in order
		matched := false
		for ruleIndex, rule := range rules {
			ruleMap, ok := rule.(map[string]interface{})
			if !ok {
				continue
			}

			matches, err := s.evaluateRule(ruleMap, context)
			if err != nil {
				return nil, fmt.Errorf("error evaluating rule %d: %w", ruleIndex, err)
			}

			if matches {
				// Add output index to indicate which rule matched
				outputItem := model.DataItem{
					JSON: item.JSON,
				}

				// Add metadata about which rule matched
				if outputItem.JSON == nil {
					outputItem.JSON = make(map[string]interface{})
				}
				outputItem.JSON["_switchRuleIndex"] = ruleIndex

				outputData = append(outputData, outputItem)
				matched = true

				// Check if we should stop after first match
				stopAfterFirstMatch, _ := nodeParams["stopAfterFirstMatch"].(bool)
				if stopAfterFirstMatch {
					break
				}
			}
		}

		// Handle fallback for unmatched items
		if !matched {
			fallbackToLast, _ := nodeParams["fallbackToLast"].(bool)
			if fallbackToLast {
				outputItem := model.DataItem{
					JSON: item.JSON,
				}
				if outputItem.JSON == nil {
					outputItem.JSON = make(map[string]interface{})
				}
				outputItem.JSON["_switchRuleIndex"] = len(rules) // Indicates fallback
				outputData = append(outputData, outputItem)
			}
		}
	}

	return outputData, nil
}

// evaluateRule evaluates a single switch rule
func (s *SwitchNode) evaluateRule(rule map[string]interface{}, context *expressions.ExpressionContext) (bool, error) {
	// Get rule properties
	field, _ := rule["field"].(string)
	operation, _ := rule["operation"].(string)
	value, _ := rule["value"]

	if field == "" || operation == "" {
		return false, fmt.Errorf("field and operation are required for switch rule")
	}

	// Resolve the field value
	fieldExpr := fmt.Sprintf("{{ %s }}", field)
	fieldValue, err := s.evaluator.EvaluateExpression(fieldExpr, context)
	if err != nil {
		return false, fmt.Errorf("failed to resolve field %s: %w", field, err)
	}

	// Perform the comparison based on operation
	return s.compareValues(fieldValue, operation, value)
}

// compareValues compares two values based on the specified operation
func (s *SwitchNode) compareValues(fieldValue interface{}, operation string, expectedValue interface{}) (bool, error) {
	switch operation {
	case "equal":
		return s.isEqual(fieldValue, expectedValue), nil
	case "notEqual":
		return !s.isEqual(fieldValue, expectedValue), nil
	case "contains":
		return s.contains(fieldValue, expectedValue), nil
	case "notContains":
		return !s.contains(fieldValue, expectedValue), nil
	case "startsWith":
		return s.startsWith(fieldValue, expectedValue), nil
	case "endsWith":
		return s.endsWith(fieldValue, expectedValue), nil
	case "regex":
		return s.matchesRegex(fieldValue, expectedValue)
	case "greater":
		return s.isGreater(fieldValue, expectedValue), nil
	case "greaterEqual":
		return s.isGreaterOrEqual(fieldValue, expectedValue), nil
	case "smaller":
		return s.isSmaller(fieldValue, expectedValue), nil
	case "smallerEqual":
		return s.isSmallerOrEqual(fieldValue, expectedValue), nil
	case "isEmpty":
		return s.isEmpty(fieldValue), nil
	case "isNotEmpty":
		return !s.isEmpty(fieldValue), nil
	default:
		return false, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// Helper functions for comparisons
func (s *SwitchNode) isEqual(a, b interface{}) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func (s *SwitchNode) contains(haystack, needle interface{}) bool {
	haystackStr := fmt.Sprintf("%v", haystack)
	needleStr := fmt.Sprintf("%v", needle)
	return strings.Contains(strings.ToLower(haystackStr), strings.ToLower(needleStr))
}

func (s *SwitchNode) startsWith(value, prefix interface{}) bool {
	valueStr := fmt.Sprintf("%v", value)
	prefixStr := fmt.Sprintf("%v", prefix)
	return strings.HasPrefix(strings.ToLower(valueStr), strings.ToLower(prefixStr))
}

func (s *SwitchNode) endsWith(value, suffix interface{}) bool {
	valueStr := fmt.Sprintf("%v", value)
	suffixStr := fmt.Sprintf("%v", suffix)
	return strings.HasSuffix(strings.ToLower(valueStr), strings.ToLower(suffixStr))
}

func (s *SwitchNode) matchesRegex(value, pattern interface{}) (bool, error) {
	valueStr := fmt.Sprintf("%v", value)
	patternStr := fmt.Sprintf("%v", pattern)

	regex, err := regexp.Compile(patternStr)
	if err != nil {
		return false, fmt.Errorf("invalid regex pattern: %w", err)
	}

	return regex.MatchString(valueStr), nil
}

func (s *SwitchNode) isGreater(a, b interface{}) bool {
	aNum, aErr := s.toNumber(a)
	bNum, bErr := s.toNumber(b)
	if aErr != nil || bErr != nil {
		return false
	}
	return aNum > bNum
}

func (s *SwitchNode) isGreaterOrEqual(a, b interface{}) bool {
	aNum, aErr := s.toNumber(a)
	bNum, bErr := s.toNumber(b)
	if aErr != nil || bErr != nil {
		return false
	}
	return aNum >= bNum
}

func (s *SwitchNode) isSmaller(a, b interface{}) bool {
	aNum, aErr := s.toNumber(a)
	bNum, bErr := s.toNumber(b)
	if aErr != nil || bErr != nil {
		return false
	}
	return aNum < bNum
}

func (s *SwitchNode) isSmallerOrEqual(a, b interface{}) bool {
	aNum, aErr := s.toNumber(a)
	bNum, bErr := s.toNumber(b)
	if aErr != nil || bErr != nil {
		return false
	}
	return aNum <= bNum
}

func (s *SwitchNode) isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}

	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	case []interface{}:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	default:
		return false
	}
}

func (s *SwitchNode) toNumber(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to number", value)
	}
}

// ValidateParameters validates Switch node parameters
func (s *SwitchNode) ValidateParameters(params map[string]interface{}) error {
	rules, ok := params["rules"].([]interface{})
	if !ok {
		return fmt.Errorf("rules parameter is required")
	}

	if len(rules) == 0 {
		return fmt.Errorf("at least one rule is required")
	}

	// Validate each rule
	for i, rule := range rules {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			return fmt.Errorf("rule %d must be an object", i)
		}

		field, ok := ruleMap["field"].(string)
		if !ok || field == "" {
			return fmt.Errorf("rule %d: field is required", i)
		}

		operation, ok := ruleMap["operation"].(string)
		if !ok || operation == "" {
			return fmt.Errorf("rule %d: operation is required", i)
		}

		validOperations := map[string]bool{
			"equal": true, "notEqual": true, "contains": true, "notContains": true,
			"startsWith": true, "endsWith": true, "regex": true,
			"greater": true, "greaterEqual": true, "smaller": true, "smallerEqual": true,
			"isEmpty": true, "isNotEmpty": true,
		}

		if !validOperations[operation] {
			return fmt.Errorf("rule %d: invalid operation %s", i, operation)
		}

		// Value is required for most operations
		needsValue := map[string]bool{
			"isEmpty": false, "isNotEmpty": false,
		}
		if needsValue[operation] == false && operation != "isEmpty" && operation != "isNotEmpty" {
			if _, ok := ruleMap["value"]; !ok {
				return fmt.Errorf("rule %d: value is required for operation %s", i, operation)
			}
		}
	}

	return nil
}