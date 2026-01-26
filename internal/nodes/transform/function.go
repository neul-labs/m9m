/*
Package transform provides data transformation node implementations for m9m.
*/
package transform

import (
	"fmt"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
	"github.com/neul-labs/m9m/internal/expressions"
)

// FunctionNode implements the Function node functionality for executing custom JavaScript code
type FunctionNode struct {
	*base.BaseNode
}

// NewFunctionNode creates a new Function node
func NewFunctionNode() *FunctionNode {
	description := base.NodeDescription{
		Name:        "Function",
		Description: "Executes custom JavaScript code",
		Category:    "Data Transformation",
	}
	
	return &FunctionNode{
		BaseNode: base.NewBaseNode(description),
	}
}

// Description returns the node description
func (f *FunctionNode) Description() base.NodeDescription {
	return f.BaseNode.Description()
}

// ValidateParameters validates Function node parameters
func (f *FunctionNode) ValidateParameters(params map[string]interface{}) error {
	if params == nil {
		return f.CreateError("parameters cannot be nil", nil)
	}
	
	// Check if jsCode exists
	jsCode, ok := params["jsCode"]
	if !ok {
		return f.CreateError("jsCode parameter is required", nil)
	}
	
	// Check if jsCode is a string
	jsCodeStr, ok := jsCode.(string)
	if !ok {
		return f.CreateError("jsCode must be a string", nil)
	}
	
	// Check if jsCode is empty
	if jsCodeStr == "" {
		return f.CreateError("jsCode parameter cannot be empty", nil)
	}
	
	return nil
}

// Execute processes the Function node operation
func (f *FunctionNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	if len(inputData) == 0 {
		return []model.DataItem{}, nil
	}

	// Get JavaScript code from node parameters
	jsCode, ok := nodeParams["jsCode"].(string)
	if !ok {
		return nil, f.CreateError("jsCode parameter must be a string", nil)
	}

	if jsCode == "" {
		return nil, f.CreateError("jsCode parameter cannot be empty", nil)
	}

	// Create Goja expression evaluator
	evaluator := expressions.NewGojaExpressionEvaluator(expressions.DefaultEvaluatorConfig())

	// Process each input data item
	result := make([]model.DataItem, len(inputData))

	for i, item := range inputData {
		// Create expression context for this item
		context := &expressions.ExpressionContext{
			RunIndex:            0,
			ItemIndex:           i,
			ActiveNodeName:      "function-node",
			ConnectionInputData: []model.DataItem{item},
			Mode:                expressions.ModeManual,
			AdditionalKeys:      &expressions.AdditionalKeys{},
		}

		// Execute the JavaScript code
		value, err := evaluator.EvaluateCode(jsCode, context)
		if err != nil {
			return nil, f.CreateError(fmt.Sprintf("failed to execute JavaScript code: %v", err), nil)
		}

		// Convert the result to a DataItem
		newItem, err := f.convertJavaScriptResult(value, item)
		if err != nil {
			return nil, f.CreateError(fmt.Sprintf("failed to convert JavaScript result: %v", err), nil)
		}

		result[i] = newItem
	}

	return result, nil
}

// convertJavaScriptResult converts a JavaScript result to a DataItem
func (f *FunctionNode) convertJavaScriptResult(value interface{}, originalItem model.DataItem) (model.DataItem, error) {
	// Create a new DataItem based on the JavaScript result
	newItem := model.DataItem{
		JSON: make(map[string]interface{}),
	}

	// Copy binary data if present
	if originalItem.Binary != nil {
		newItem.Binary = make(map[string]model.BinaryData)
		for k, v := range originalItem.Binary {
			newItem.Binary[k] = v
		}
	}

	// Copy paired item data if present
	if originalItem.PairedItem != nil {
		newItem.PairedItem = originalItem.PairedItem
	}

	// Convert JavaScript result to JSON
	if valueMap, ok := value.(map[string]interface{}); ok {
		// If result is an object, use it as the new JSON
		newItem.JSON = valueMap
	} else {
		// For other types, store as result property
		newItem.JSON["result"] = value
	}

	return newItem, nil
}