package transform

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/n8n-go/n8n-go/internal/expressions"
	"github.com/n8n-go/n8n-go/internal/nodes/base"
	"github.com/n8n-go/n8n-go/pkg/model"
)

type DataTransformerNode struct {
	*base.BaseNode
	evaluator *expressions.GojaExpressionEvaluator
}

type TransformationConfig struct {
	Transformations []Transformation `json:"transformations"`
	OutputMode      string           `json:"output_mode"`     // "replace", "merge", "new_field"
	FieldMappings   map[string]string `json:"field_mappings"`  // Old field name -> New field name
	FilterCondition string           `json:"filter_condition"` // Expression to filter items
	GroupBy         []string         `json:"group_by"`        // Fields to group by
	SortBy          []SortField      `json:"sort_by"`         // Fields to sort by
	Pagination      *PaginationConfig `json:"pagination"`      // Pagination settings
}

type Transformation struct {
	Field     string                 `json:"field"`      // Target field name
	Operation string                 `json:"operation"`  // Transformation operation
	Value     interface{}            `json:"value"`      // Value for the operation
	Options   map[string]interface{} `json:"options"`    // Additional options
	Condition string                 `json:"condition"`  // Conditional transformation
}

type SortField struct {
	Field string `json:"field"`
	Order string `json:"order"` // "asc" or "desc"
}

type PaginationConfig struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

type TransformationResult struct {
	TransformedItems []model.DataItem       `json:"transformed_items"`
	Statistics       *TransformationStats   `json:"statistics"`
	Errors           []TransformationError  `json:"errors"`
}

type TransformationStats struct {
	TotalItems       int                    `json:"total_items"`
	TransformedItems int                    `json:"transformed_items"`
	FilteredItems    int                    `json:"filtered_items"`
	ErrorItems       int                    `json:"error_items"`
	FieldStatistics  map[string]FieldStats  `json:"field_statistics"`
}

type FieldStats struct {
	Type         string      `json:"type"`
	UniqueValues int         `json:"unique_values"`
	NullValues   int         `json:"null_values"`
	MinValue     interface{} `json:"min_value"`
	MaxValue     interface{} `json:"max_value"`
	AvgValue     interface{} `json:"avg_value"`
}

type TransformationError struct {
	ItemIndex int    `json:"item_index"`
	Field     string `json:"field"`
	Operation string `json:"operation"`
	Message   string `json:"message"`
}

func NewDataTransformerNode() *DataTransformerNode {
	return &DataTransformerNode{
		BaseNode:  base.NewBaseNode("DataTransformer", "Transform and manipulate data", "1.0.0"),
		evaluator: expressions.NewGojaExpressionEvaluator(),
	}
}

func (n *DataTransformerNode) Execute(input *model.NodeExecutionInput) (*model.NodeExecutionOutput, error) {
	var config TransformationConfig
	if err := json.Unmarshal(input.Config, &config); err != nil {
		return nil, fmt.Errorf("failed to parse transformation config: %w", err)
	}

	result := &TransformationResult{
		TransformedItems: []model.DataItem{},
		Statistics: &TransformationStats{
			TotalItems:      len(input.Items),
			FieldStatistics: make(map[string]FieldStats),
		},
		Errors: []TransformationError{},
	}

	// Filter items if condition is provided
	filteredItems := n.filterItems(input.Items, config.FilterCondition, result)

	// Apply transformations
	transformedItems := n.applyTransformations(filteredItems, config, result)

	// Apply field mappings
	if len(config.FieldMappings) > 0 {
		transformedItems = n.applyFieldMappings(transformedItems, config.FieldMappings)
	}

	// Group by if specified
	if len(config.GroupBy) > 0 {
		transformedItems = n.groupItems(transformedItems, config.GroupBy)
	}

	// Sort if specified
	if len(config.SortBy) > 0 {
		transformedItems = n.sortItems(transformedItems, config.SortBy)
	}

	// Apply pagination if specified
	if config.Pagination != nil {
		transformedItems = n.paginateItems(transformedItems, config.Pagination)
	}

	result.TransformedItems = transformedItems
	result.Statistics.TransformedItems = len(transformedItems)

	// Calculate field statistics
	n.calculateFieldStatistics(transformedItems, result.Statistics)

	return &model.NodeExecutionOutput{
		Items: transformedItems,
		Metadata: map[string]interface{}{
			"transformation_stats": result.Statistics,
			"transformation_errors": result.Errors,
		},
	}, nil
}

func (n *DataTransformerNode) filterItems(items []model.DataItem, condition string, result *TransformationResult) []model.DataItem {
	if condition == "" {
		return items
	}

	var filtered []model.DataItem
	for i, item := range items {
		context := expressions.NewExpressionContext()
		context.SetVariable("item", item.JSON)
		context.SetVariable("index", i)

		match, err := n.evaluator.Evaluate(condition, context)
		if err != nil {
			result.Errors = append(result.Errors, TransformationError{
				ItemIndex: i,
				Field:     "filter",
				Operation: "condition",
				Message:   err.Error(),
			})
			continue
		}

		if n.isTruthy(match) {
			filtered = append(filtered, item)
		} else {
			result.Statistics.FilteredItems++
		}
	}

	return filtered
}

func (n *DataTransformerNode) applyTransformations(items []model.DataItem, config TransformationConfig, result *TransformationResult) []model.DataItem {
	var transformed []model.DataItem

	for i, item := range items {
		newItem := model.DataItem{
			JSON: make(map[string]interface{}),
		}

		// Copy original data based on output mode
		switch config.OutputMode {
		case "replace":
			// Start with empty object
		case "merge", "":
			// Copy original data
			for k, v := range item.JSON {
				newItem.JSON[k] = v
			}
		case "new_field":
			// Copy original data and add transformed fields with prefix
			for k, v := range item.JSON {
				newItem.JSON[k] = v
			}
		}

		// Apply each transformation
		hasErrors := false
		for _, transformation := range config.Transformations {
			if transformation.Condition != "" {
				context := expressions.NewExpressionContext()
				context.SetVariable("item", item.JSON)
				context.SetVariable("index", i)

				shouldApply, err := n.evaluator.Evaluate(transformation.Condition, context)
				if err != nil || !n.isTruthy(shouldApply) {
					continue
				}
			}

			err := n.applyTransformation(&newItem, &transformation, item, i)
			if err != nil {
				result.Errors = append(result.Errors, TransformationError{
					ItemIndex: i,
					Field:     transformation.Field,
					Operation: transformation.Operation,
					Message:   err.Error(),
				})
				hasErrors = true
			}
		}

		if hasErrors {
			result.Statistics.ErrorItems++
		}

		transformed = append(transformed, newItem)
	}

	return transformed
}

func (n *DataTransformerNode) applyTransformation(newItem *model.DataItem, transformation *Transformation, originalItem model.DataItem, itemIndex int) error {
	sourceValue := n.getFieldValue(originalItem.JSON, transformation.Field)

	var result interface{}
	var err error

	switch transformation.Operation {
	case "set":
		result = transformation.Value
	case "copy":
		sourceField := transformation.Value.(string)
		result = n.getFieldValue(originalItem.JSON, sourceField)
	case "concat":
		result, err = n.concatOperation(sourceValue, transformation.Value, transformation.Options)
	case "substring":
		result, err = n.substringOperation(sourceValue, transformation.Options)
	case "replace":
		result, err = n.replaceOperation(sourceValue, transformation.Options)
	case "uppercase":
		result = n.uppercaseOperation(sourceValue)
	case "lowercase":
		result = n.lowercaseOperation(sourceValue)
	case "trim":
		result = n.trimOperation(sourceValue)
	case "add":
		result, err = n.addOperation(sourceValue, transformation.Value)
	case "subtract":
		result, err = n.subtractOperation(sourceValue, transformation.Value)
	case "multiply":
		result, err = n.multiplyOperation(sourceValue, transformation.Value)
	case "divide":
		result, err = n.divideOperation(sourceValue, transformation.Value)
	case "round":
		result, err = n.roundOperation(sourceValue, transformation.Options)
	case "abs":
		result, err = n.absOperation(sourceValue)
	case "date_format":
		result, err = n.dateFormatOperation(sourceValue, transformation.Options)
	case "date_add":
		result, err = n.dateAddOperation(sourceValue, transformation.Options)
	case "date_parse":
		result, err = n.dateParseOperation(sourceValue, transformation.Options)
	case "array_join":
		result, err = n.arrayJoinOperation(sourceValue, transformation.Options)
	case "array_split":
		result, err = n.arraySplitOperation(sourceValue, transformation.Options)
	case "array_unique":
		result = n.arrayUniqueOperation(sourceValue)
	case "array_sort":
		result = n.arraySortOperation(sourceValue, transformation.Options)
	case "json_parse":
		result, err = n.jsonParseOperation(sourceValue)
	case "json_stringify":
		result, err = n.jsonStringifyOperation(sourceValue, transformation.Options)
	case "regex_extract":
		result, err = n.regexExtractOperation(sourceValue, transformation.Options)
	case "regex_match":
		result, err = n.regexMatchOperation(sourceValue, transformation.Options)
	case "hash":
		result, err = n.hashOperation(sourceValue, transformation.Options)
	case "encode_base64":
		result = n.encodeBase64Operation(sourceValue)
	case "decode_base64":
		result, err = n.decodeBase64Operation(sourceValue)
	case "expression":
		result, err = n.expressionOperation(transformation.Value.(string), originalItem, itemIndex)
	default:
		return fmt.Errorf("unknown transformation operation: %s", transformation.Operation)
	}

	if err != nil {
		return err
	}

	n.setFieldValue(newItem.JSON, transformation.Field, result)
	return nil
}

func (n *DataTransformerNode) concatOperation(sourceValue, value interface{}, options map[string]interface{}) (interface{}, error) {
	separator := ""
	if options != nil {
		if sep, ok := options["separator"]; ok {
			separator = fmt.Sprintf("%v", sep)
		}
	}

	parts := []string{}
	if sourceValue != nil {
		parts = append(parts, fmt.Sprintf("%v", sourceValue))
	}

	switch v := value.(type) {
	case string:
		parts = append(parts, v)
	case []interface{}:
		for _, item := range v {
			parts = append(parts, fmt.Sprintf("%v", item))
		}
	default:
		parts = append(parts, fmt.Sprintf("%v", v))
	}

	return strings.Join(parts, separator), nil
}

func (n *DataTransformerNode) substringOperation(sourceValue interface{}, options map[string]interface{}) (interface{}, error) {
	str := fmt.Sprintf("%v", sourceValue)
	start := 0
	end := len(str)

	if options != nil {
		if s, ok := options["start"]; ok {
			if startInt, ok := s.(float64); ok {
				start = int(startInt)
			}
		}
		if e, ok := options["end"]; ok {
			if endInt, ok := e.(float64); ok {
				end = int(endInt)
			}
		}
		if l, ok := options["length"]; ok {
			if lengthInt, ok := l.(float64); ok {
				end = start + int(lengthInt)
			}
		}
	}

	if start < 0 {
		start = 0
	}
	if end > len(str) {
		end = len(str)
	}
	if start > end {
		start = end
	}

	return str[start:end], nil
}

func (n *DataTransformerNode) replaceOperation(sourceValue interface{}, options map[string]interface{}) (interface{}, error) {
	str := fmt.Sprintf("%v", sourceValue)

	if options == nil {
		return str, nil
	}

	find, findOk := options["find"]
	replace, replaceOk := options["replace"]

	if !findOk || !replaceOk {
		return str, fmt.Errorf("replace operation requires 'find' and 'replace' options")
	}

	findStr := fmt.Sprintf("%v", find)
	replaceStr := fmt.Sprintf("%v", replace)

	// Check if it's a regex replace
	if useRegex, ok := options["regex"]; ok && useRegex.(bool) {
		re, err := regexp.Compile(findStr)
		if err != nil {
			return nil, err
		}
		return re.ReplaceAllString(str, replaceStr), nil
	}

	return strings.ReplaceAll(str, findStr, replaceStr), nil
}

func (n *DataTransformerNode) uppercaseOperation(sourceValue interface{}) interface{} {
	return strings.ToUpper(fmt.Sprintf("%v", sourceValue))
}

func (n *DataTransformerNode) lowercaseOperation(sourceValue interface{}) interface{} {
	return strings.ToLower(fmt.Sprintf("%v", sourceValue))
}

func (n *DataTransformerNode) trimOperation(sourceValue interface{}) interface{} {
	return strings.TrimSpace(fmt.Sprintf("%v", sourceValue))
}

func (n *DataTransformerNode) addOperation(sourceValue, value interface{}) (interface{}, error) {
	num1, err := n.toNumber(sourceValue)
	if err != nil {
		return nil, err
	}
	num2, err := n.toNumber(value)
	if err != nil {
		return nil, err
	}
	return num1 + num2, nil
}

func (n *DataTransformerNode) subtractOperation(sourceValue, value interface{}) (interface{}, error) {
	num1, err := n.toNumber(sourceValue)
	if err != nil {
		return nil, err
	}
	num2, err := n.toNumber(value)
	if err != nil {
		return nil, err
	}
	return num1 - num2, nil
}

func (n *DataTransformerNode) multiplyOperation(sourceValue, value interface{}) (interface{}, error) {
	num1, err := n.toNumber(sourceValue)
	if err != nil {
		return nil, err
	}
	num2, err := n.toNumber(value)
	if err != nil {
		return nil, err
	}
	return num1 * num2, nil
}

func (n *DataTransformerNode) divideOperation(sourceValue, value interface{}) (interface{}, error) {
	num1, err := n.toNumber(sourceValue)
	if err != nil {
		return nil, err
	}
	num2, err := n.toNumber(value)
	if err != nil {
		return nil, err
	}
	if num2 == 0 {
		return nil, fmt.Errorf("division by zero")
	}
	return num1 / num2, nil
}

func (n *DataTransformerNode) roundOperation(sourceValue interface{}, options map[string]interface{}) (interface{}, error) {
	num, err := n.toNumber(sourceValue)
	if err != nil {
		return nil, err
	}

	precision := 0
	if options != nil {
		if p, ok := options["precision"]; ok {
			if pInt, ok := p.(float64); ok {
				precision = int(pInt)
			}
		}
	}

	multiplier := 1.0
	for i := 0; i < precision; i++ {
		multiplier *= 10
	}

	return float64(int(num*multiplier+0.5)) / multiplier, nil
}

func (n *DataTransformerNode) absOperation(sourceValue interface{}) (interface{}, error) {
	num, err := n.toNumber(sourceValue)
	if err != nil {
		return nil, err
	}
	if num < 0 {
		return -num, nil
	}
	return num, nil
}

func (n *DataTransformerNode) dateFormatOperation(sourceValue interface{}, options map[string]interface{}) (interface{}, error) {
	date, err := n.toTime(sourceValue)
	if err != nil {
		return nil, err
	}

	format := "2006-01-02 15:04:05"
	if options != nil {
		if f, ok := options["format"]; ok {
			format = fmt.Sprintf("%v", f)
		}
	}

	return date.Format(format), nil
}

func (n *DataTransformerNode) dateAddOperation(sourceValue interface{}, options map[string]interface{}) (interface{}, error) {
	date, err := n.toTime(sourceValue)
	if err != nil {
		return nil, err
	}

	if options == nil {
		return date, nil
	}

	duration := time.Duration(0)
	if years, ok := options["years"]; ok {
		if y, ok := years.(float64); ok {
			duration += time.Duration(y) * 365 * 24 * time.Hour
		}
	}
	if months, ok := options["months"]; ok {
		if m, ok := months.(float64); ok {
			duration += time.Duration(m) * 30 * 24 * time.Hour
		}
	}
	if days, ok := options["days"]; ok {
		if d, ok := days.(float64); ok {
			duration += time.Duration(d) * 24 * time.Hour
		}
	}
	if hours, ok := options["hours"]; ok {
		if h, ok := hours.(float64); ok {
			duration += time.Duration(h) * time.Hour
		}
	}
	if minutes, ok := options["minutes"]; ok {
		if m, ok := minutes.(float64); ok {
			duration += time.Duration(m) * time.Minute
		}
	}
	if seconds, ok := options["seconds"]; ok {
		if s, ok := seconds.(float64); ok {
			duration += time.Duration(s) * time.Second
		}
	}

	return date.Add(duration), nil
}

func (n *DataTransformerNode) dateParseOperation(sourceValue interface{}, options map[string]interface{}) (interface{}, error) {
	str := fmt.Sprintf("%v", sourceValue)

	format := time.RFC3339
	if options != nil {
		if f, ok := options["format"]; ok {
			format = fmt.Sprintf("%v", f)
		}
	}

	return time.Parse(format, str)
}

func (n *DataTransformerNode) arrayJoinOperation(sourceValue interface{}, options map[string]interface{}) (interface{}, error) {
	arr, ok := sourceValue.([]interface{})
	if !ok {
		return nil, fmt.Errorf("value is not an array")
	}

	separator := ","
	if options != nil {
		if sep, ok := options["separator"]; ok {
			separator = fmt.Sprintf("%v", sep)
		}
	}

	parts := make([]string, len(arr))
	for i, item := range arr {
		parts[i] = fmt.Sprintf("%v", item)
	}

	return strings.Join(parts, separator), nil
}

func (n *DataTransformerNode) arraySplitOperation(sourceValue interface{}, options map[string]interface{}) (interface{}, error) {
	str := fmt.Sprintf("%v", sourceValue)

	separator := ","
	if options != nil {
		if sep, ok := options["separator"]; ok {
			separator = fmt.Sprintf("%v", sep)
		}
	}

	parts := strings.Split(str, separator)
	result := make([]interface{}, len(parts))
	for i, part := range parts {
		result[i] = strings.TrimSpace(part)
	}

	return result, nil
}

func (n *DataTransformerNode) arrayUniqueOperation(sourceValue interface{}) interface{} {
	arr, ok := sourceValue.([]interface{})
	if !ok {
		return sourceValue
	}

	seen := make(map[string]bool)
	var unique []interface{}

	for _, item := range arr {
		key := fmt.Sprintf("%v", item)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, item)
		}
	}

	return unique
}

func (n *DataTransformerNode) arraySortOperation(sourceValue interface{}, options map[string]interface{}) interface{} {
	arr, ok := sourceValue.([]interface{})
	if !ok {
		return sourceValue
	}

	// Create a copy to avoid modifying the original
	sorted := make([]interface{}, len(arr))
	copy(sorted, arr)

	reverse := false
	if options != nil {
		if r, ok := options["reverse"]; ok {
			reverse = r.(bool)
		}
	}

	sort.Slice(sorted, func(i, j int) bool {
		str1 := fmt.Sprintf("%v", sorted[i])
		str2 := fmt.Sprintf("%v", sorted[j])

		if reverse {
			return str1 > str2
		}
		return str1 < str2
	})

	return sorted
}

func (n *DataTransformerNode) jsonParseOperation(sourceValue interface{}) (interface{}, error) {
	str := fmt.Sprintf("%v", sourceValue)

	var result interface{}
	err := json.Unmarshal([]byte(str), &result)
	return result, err
}

func (n *DataTransformerNode) jsonStringifyOperation(sourceValue interface{}, options map[string]interface{}) (interface{}, error) {
	pretty := false
	if options != nil {
		if p, ok := options["pretty"]; ok {
			pretty = p.(bool)
		}
	}

	if pretty {
		data, err := json.MarshalIndent(sourceValue, "", "  ")
		return string(data), err
	}

	data, err := json.Marshal(sourceValue)
	return string(data), err
}

func (n *DataTransformerNode) regexExtractOperation(sourceValue interface{}, options map[string]interface{}) (interface{}, error) {
	str := fmt.Sprintf("%v", sourceValue)

	if options == nil {
		return nil, fmt.Errorf("regex_extract requires 'pattern' option")
	}

	pattern, ok := options["pattern"]
	if !ok {
		return nil, fmt.Errorf("regex_extract requires 'pattern' option")
	}

	re, err := regexp.Compile(fmt.Sprintf("%v", pattern))
	if err != nil {
		return nil, err
	}

	matches := re.FindStringSubmatch(str)
	if len(matches) == 0 {
		return nil, nil
	}

	if len(matches) == 1 {
		return matches[0], nil
	}

	// Return all matches as array
	result := make([]interface{}, len(matches))
	for i, match := range matches {
		result[i] = match
	}
	return result, nil
}

func (n *DataTransformerNode) regexMatchOperation(sourceValue interface{}, options map[string]interface{}) (interface{}, error) {
	str := fmt.Sprintf("%v", sourceValue)

	if options == nil {
		return false, fmt.Errorf("regex_match requires 'pattern' option")
	}

	pattern, ok := options["pattern"]
	if !ok {
		return false, fmt.Errorf("regex_match requires 'pattern' option")
	}

	re, err := regexp.Compile(fmt.Sprintf("%v", pattern))
	if err != nil {
		return false, err
	}

	return re.MatchString(str), nil
}

func (n *DataTransformerNode) hashOperation(sourceValue interface{}, options map[string]interface{}) (interface{}, error) {
	// This would implement various hashing algorithms
	// For now, we'll return a simple hash placeholder
	str := fmt.Sprintf("%v", sourceValue)
	return fmt.Sprintf("hash_%s", str), nil
}

func (n *DataTransformerNode) encodeBase64Operation(sourceValue interface{}) interface{} {
	str := fmt.Sprintf("%v", sourceValue)
	// This is a placeholder - real implementation would use base64 encoding
	return fmt.Sprintf("base64_%s", str)
}

func (n *DataTransformerNode) decodeBase64Operation(sourceValue interface{}) (interface{}, error) {
	str := fmt.Sprintf("%v", sourceValue)
	// This is a placeholder - real implementation would use base64 decoding
	if strings.HasPrefix(str, "base64_") {
		return strings.TrimPrefix(str, "base64_"), nil
	}
	return str, nil
}

func (n *DataTransformerNode) expressionOperation(expression string, originalItem model.DataItem, itemIndex int) (interface{}, error) {
	context := expressions.NewExpressionContext()
	context.SetVariable("item", originalItem.JSON)
	context.SetVariable("index", itemIndex)

	return n.evaluator.Evaluate(expression, context)
}

// Helper methods

func (n *DataTransformerNode) getFieldValue(data map[string]interface{}, field string) interface{} {
	if strings.Contains(field, ".") {
		parts := strings.Split(field, ".")
		current := data

		for i, part := range parts {
			if i == len(parts)-1 {
				return current[part]
			}

			if next, ok := current[part].(map[string]interface{}); ok {
				current = next
			} else {
				return nil
			}
		}
	}

	return data[field]
}

func (n *DataTransformerNode) setFieldValue(data map[string]interface{}, field string, value interface{}) {
	if strings.Contains(field, ".") {
		parts := strings.Split(field, ".")
		current := data

		for i, part := range parts {
			if i == len(parts)-1 {
				current[part] = value
				return
			}

			if _, ok := current[part]; !ok {
				current[part] = make(map[string]interface{})
			}

			if next, ok := current[part].(map[string]interface{}); ok {
				current = next
			} else {
				current[part] = make(map[string]interface{})
				current = current[part].(map[string]interface{})
			}
		}
	} else {
		data[field] = value
	}
}

func (n *DataTransformerNode) toNumber(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %v to number", value)
	}
}

func (n *DataTransformerNode) toTime(value interface{}) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v, nil
	case string:
		return time.Parse(time.RFC3339, v)
	case int64:
		return time.Unix(v, 0), nil
	case float64:
		return time.Unix(int64(v), 0), nil
	default:
		return time.Time{}, fmt.Errorf("cannot convert %v to time", value)
	}
}

func (n *DataTransformerNode) isTruthy(value interface{}) bool {
	if value == nil {
		return false
	}

	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v != ""
	case int, int32, int64:
		return v != 0
	case float32, float64:
		return v != 0.0
	case []interface{}:
		return len(v) > 0
	case map[string]interface{}:
		return len(v) > 0
	default:
		return true
	}
}

func (n *DataTransformerNode) applyFieldMappings(items []model.DataItem, mappings map[string]string) []model.DataItem {
	var result []model.DataItem

	for _, item := range items {
		newItem := model.DataItem{
			JSON: make(map[string]interface{}),
		}

		for oldField, newField := range mappings {
			if value := n.getFieldValue(item.JSON, oldField); value != nil {
				n.setFieldValue(newItem.JSON, newField, value)
			}
		}

		// Copy fields that aren't being mapped
		for field, value := range item.JSON {
			if _, isMapped := mappings[field]; !isMapped {
				newItem.JSON[field] = value
			}
		}

		result = append(result, newItem)
	}

	return result
}

func (n *DataTransformerNode) groupItems(items []model.DataItem, groupBy []string) []model.DataItem {
	groups := make(map[string][]model.DataItem)

	for _, item := range items {
		var keyParts []string
		for _, field := range groupBy {
			value := n.getFieldValue(item.JSON, field)
			keyParts = append(keyParts, fmt.Sprintf("%v", value))
		}
		key := strings.Join(keyParts, "|")
		groups[key] = append(groups[key], item)
	}

	var result []model.DataItem
	for _, group := range groups {
		groupData := make(map[string]interface{})
		groupData["items"] = group
		groupData["count"] = len(group)

		// Add group key fields
		if len(group) > 0 {
			for _, field := range groupBy {
				groupData[field] = n.getFieldValue(group[0].JSON, field)
			}
		}

		result = append(result, model.DataItem{JSON: groupData})
	}

	return result
}

func (n *DataTransformerNode) sortItems(items []model.DataItem, sortBy []SortField) []model.DataItem {
	// Create a copy to avoid modifying the original
	sorted := make([]model.DataItem, len(items))
	copy(sorted, items)

	sort.Slice(sorted, func(i, j int) bool {
		for _, sortField := range sortBy {
			val1 := n.getFieldValue(sorted[i].JSON, sortField.Field)
			val2 := n.getFieldValue(sorted[j].JSON, sortField.Field)

			str1 := fmt.Sprintf("%v", val1)
			str2 := fmt.Sprintf("%v", val2)

			if str1 == str2 {
				continue
			}

			if sortField.Order == "desc" {
				return str1 > str2
			}
			return str1 < str2
		}
		return false
	})

	return sorted
}

func (n *DataTransformerNode) paginateItems(items []model.DataItem, pagination *PaginationConfig) []model.DataItem {
	if pagination.PageSize <= 0 {
		return items
	}

	start := (pagination.Page - 1) * pagination.PageSize
	if start < 0 {
		start = 0
	}
	if start >= len(items) {
		return []model.DataItem{}
	}

	end := start + pagination.PageSize
	if end > len(items) {
		end = len(items)
	}

	return items[start:end]
}

func (n *DataTransformerNode) calculateFieldStatistics(items []model.DataItem, stats *TransformationStats) {
	fieldValues := make(map[string][]interface{})

	// Collect all field values
	for _, item := range items {
		for field, value := range item.JSON {
			fieldValues[field] = append(fieldValues[field], value)
		}
	}

	// Calculate statistics for each field
	for field, values := range fieldValues {
		fieldStat := FieldStats{}

		// Count unique values and nulls
		unique := make(map[string]bool)
		nullCount := 0

		for _, value := range values {
			if value == nil {
				nullCount++
			} else {
				unique[fmt.Sprintf("%v", value)] = true
			}
		}

		fieldStat.UniqueValues = len(unique)
		fieldStat.NullValues = nullCount

		// Determine field type and calculate min/max/avg for numeric fields
		if len(values) > 0 && values[0] != nil {
			fieldStat.Type = reflect.TypeOf(values[0]).String()

			// For numeric fields, calculate min/max/avg
			if fieldStat.Type == "float64" || fieldStat.Type == "int" {
				var sum float64
				var min, max float64
				validCount := 0

				for i, value := range values {
					if value != nil {
						if num, err := n.toNumber(value); err == nil {
							if validCount == 0 {
								min = num
								max = num
							} else {
								if num < min {
									min = num
								}
								if num > max {
									max = num
								}
							}
							sum += num
							validCount++
						}
					}
				}

				if validCount > 0 {
					fieldStat.MinValue = min
					fieldStat.MaxValue = max
					fieldStat.AvgValue = sum / float64(validCount)
				}
			}
		}

		stats.FieldStatistics[field] = fieldStat
	}
}