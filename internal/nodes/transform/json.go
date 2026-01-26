package transform

import (
	"encoding/json"
	"fmt"

	"github.com/neul-labs/m9m/internal/expressions"
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// JSONNode implements the JSON node functionality for parsing and manipulating JSON data
type JSONNode struct {
	*base.BaseNode
	evaluator *expressions.GojaExpressionEvaluator
}

// NewJSONNode creates a new JSON node
func NewJSONNode() *JSONNode {
	description := base.NodeDescription{
		Name:        "JSON",
		Description: "Parse, stringify and manipulate JSON data",
		Category:    "Data Transformation",
	}

	return &JSONNode{
		BaseNode:  base.NewBaseNode(description),
		evaluator: expressions.NewGojaExpressionEvaluator(expressions.DefaultEvaluatorConfig()),
	}
}

// Execute processes the JSON node operation
func (j *JSONNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	if len(inputData) == 0 {
		return []model.DataItem{}, nil
	}

	// Get operation type
	operation, ok := nodeParams["operation"].(string)
	if !ok {
		operation = "parse" // Default operation
	}

	var outputData []model.DataItem

	for index, item := range inputData {
		context := &expressions.ExpressionContext{
			ActiveNodeName:      "JSON",
			RunIndex:           0,
			ItemIndex:          index,
			Mode:               expressions.ModeManual,
			ConnectionInputData: []model.DataItem{item},
			Workflow: &model.Workflow{
				Name: "JSON Processing",
			},
			AdditionalKeys: &expressions.AdditionalKeys{
				ExecutionId: "json-processing",
			},
		}

		switch operation {
		case "parse":
			result, err := j.parseJSON(item, nodeParams, context)
			if err != nil {
				return nil, fmt.Errorf("JSON parse operation failed: %w", err)
			}
			outputData = append(outputData, result...)

		case "stringify":
			result, err := j.stringifyJSON(item, nodeParams, context)
			if err != nil {
				return nil, fmt.Errorf("JSON stringify operation failed: %w", err)
			}
			outputData = append(outputData, result)

		case "extract":
			result, err := j.extractFromJSON(item, nodeParams, context)
			if err != nil {
				return nil, fmt.Errorf("JSON extract operation failed: %w", err)
			}
			outputData = append(outputData, result)

		case "merge":
			result, err := j.mergeJSON(item, nodeParams, context)
			if err != nil {
				return nil, fmt.Errorf("JSON merge operation failed: %w", err)
			}
			outputData = append(outputData, result)

		default:
			return nil, fmt.Errorf("unsupported JSON operation: %s", operation)
		}
	}

	return outputData, nil
}

// parseJSON parses JSON strings into objects
func (j *JSONNode) parseJSON(item model.DataItem, nodeParams map[string]interface{}, context *expressions.ExpressionContext) ([]model.DataItem, error) {
	// Get the JSON string to parse
	jsonPath, _ := nodeParams["jsonPath"].(string)
	if jsonPath == "" {
		jsonPath = "$json" // Default to entire JSON
	}

	// Resolve the expression to get the JSON string
	jsonStringExpr := fmt.Sprintf("{{ %s }}", jsonPath)
	jsonStringValue, err := j.evaluator.EvaluateExpression(jsonStringExpr, context)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve JSON path: %w", err)
	}

	jsonString, ok := jsonStringValue.(string)
	if !ok {
		return nil, fmt.Errorf("JSON path did not resolve to a string, got %T", jsonStringValue)
	}

	// Parse the JSON string
	var parsedData interface{}
	err = json.Unmarshal([]byte(jsonString), &parsedData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Handle arrays differently - create multiple items for array elements
	includeItemIndex, _ := nodeParams["includeItemIndex"].(bool)

	if arr, ok := parsedData.([]interface{}); ok && includeItemIndex {
		var result []model.DataItem
		for idx, element := range arr {
			newItem := model.DataItem{
				JSON: map[string]interface{}{
					"data":      element,
					"itemIndex": idx,
				},
			}
			result = append(result, newItem)
		}
		return result, nil
	}

	// Single item result
	result := model.DataItem{
		JSON: map[string]interface{}{
			"data": parsedData,
		},
	}

	return []model.DataItem{result}, nil
}

// stringifyJSON converts objects to JSON strings
func (j *JSONNode) stringifyJSON(item model.DataItem, nodeParams map[string]interface{}, context *expressions.ExpressionContext) (model.DataItem, error) {
	// Get the data to stringify
	dataPath, _ := nodeParams["dataPath"].(string)
	if dataPath == "" {
		dataPath = "$json" // Default to entire JSON
	}

	// Resolve the expression
	dataExpr := fmt.Sprintf("{{ %s }}", dataPath)
	dataValue, err := j.evaluator.EvaluateExpression(dataExpr, context)
	if err != nil {
		return model.DataItem{}, fmt.Errorf("failed to resolve data path: %w", err)
	}

	// Get formatting options
	indent, _ := nodeParams["indent"].(bool)
	var jsonBytes []byte

	if indent {
		jsonBytes, err = json.MarshalIndent(dataValue, "", "  ")
	} else {
		jsonBytes, err = json.Marshal(dataValue)
	}

	if err != nil {
		return model.DataItem{}, fmt.Errorf("failed to stringify JSON: %w", err)
	}

	// Create result item
	result := model.DataItem{
		JSON: map[string]interface{}{
			"jsonString": string(jsonBytes),
		},
	}

	return result, nil
}

// extractFromJSON extracts specific fields from JSON data
func (j *JSONNode) extractFromJSON(item model.DataItem, nodeParams map[string]interface{}, context *expressions.ExpressionContext) (model.DataItem, error) {
	// Get extraction paths
	extractionFields, ok := nodeParams["extractionFields"].([]interface{})
	if !ok {
		return model.DataItem{}, fmt.Errorf("extraction fields not specified")
	}

	result := make(map[string]interface{})

	for _, field := range extractionFields {
		fieldMap, ok := field.(map[string]interface{})
		if !ok {
			continue
		}

		fieldName, _ := fieldMap["name"].(string)
		fieldPath, _ := fieldMap["path"].(string)

		if fieldName == "" || fieldPath == "" {
			continue
		}

		// Resolve the field expression
		fieldExpr := fmt.Sprintf("{{ %s }}", fieldPath)
		fieldValue, err := j.evaluator.EvaluateExpression(fieldExpr, context)
		if err != nil {
			// Set null for failed extractions
			result[fieldName] = nil
			continue
		}

		result[fieldName] = fieldValue
	}

	return model.DataItem{JSON: result}, nil
}

// mergeJSON merges multiple JSON objects
func (j *JSONNode) mergeJSON(item model.DataItem, nodeParams map[string]interface{}, context *expressions.ExpressionContext) (model.DataItem, error) {
	// Get merge sources
	mergeSources, ok := nodeParams["mergeSources"].([]interface{})
	if !ok {
		return model.DataItem{}, fmt.Errorf("merge sources not specified")
	}

	result := make(map[string]interface{})

	// Start with the base item
	for key, value := range item.JSON {
		result[key] = value
	}

	// Merge each source
	for _, source := range mergeSources {
		sourcePath, ok := source.(string)
		if !ok {
			continue
		}

		// Resolve the source expression
		sourceExpr := fmt.Sprintf("{{ %s }}", sourcePath)
		sourceValue, err := j.evaluator.EvaluateExpression(sourceExpr, context)
		if err != nil {
			continue
		}

		// Merge if it's a map
		if sourceMap, ok := sourceValue.(map[string]interface{}); ok {
			for key, value := range sourceMap {
				result[key] = value
			}
		}
	}

	return model.DataItem{JSON: result}, nil
}

// ValidateParameters validates JSON node parameters
func (j *JSONNode) ValidateParameters(params map[string]interface{}) error {
	operation, ok := params["operation"].(string)
	if !ok {
		return fmt.Errorf("operation parameter is required")
	}

	validOperations := map[string]bool{
		"parse": true, "stringify": true, "extract": true, "merge": true,
	}

	if !validOperations[operation] {
		return fmt.Errorf("invalid operation: %s", operation)
	}

	// Validate operation-specific parameters
	switch operation {
	case "extract":
		if _, ok := params["extractionFields"]; !ok {
			return fmt.Errorf("extractionFields parameter is required for extract operation")
		}
	case "merge":
		if _, ok := params["mergeSources"]; !ok {
			return fmt.Errorf("mergeSources parameter is required for merge operation")
		}
	}

	return nil
}