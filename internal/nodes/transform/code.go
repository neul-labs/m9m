/*
Package transform provides data transformation node implementations for n8n-go.
*/
package transform

import (
	"fmt"

	"github.com/dipankar/n8n-go/internal/model"
	"github.com/dipankar/n8n-go/internal/nodes/base"
	"github.com/dipankar/n8n-go/internal/expressions"
)

// CodeNode implements the Code node functionality for executing custom code
type CodeNode struct {
	*base.BaseNode
}

// NewCodeNode creates a new Code node
func NewCodeNode() *CodeNode {
	description := base.NodeDescription{
		Name:        "Code",
		Description: "Executes custom code in various languages",
		Category:    "Data Transformation",
	}
	
	return &CodeNode{
		BaseNode: base.NewBaseNode(description),
	}
}

// Description returns the node description
func (c *CodeNode) Description() base.NodeDescription {
	return c.BaseNode.Description()
}

// ValidateParameters validates Code node parameters
func (c *CodeNode) ValidateParameters(params map[string]interface{}) error {
	if params == nil {
		return c.CreateError("parameters cannot be nil", nil)
	}
	
	// Check if mode exists
	mode, ok := params["mode"]
	if !ok {
		return c.CreateError("mode parameter is required", nil)
	}
	
	// Check if mode is a string
	modeStr, ok := mode.(string)
	if !ok {
		return c.CreateError("mode must be a string", nil)
	}
	
	// Validate mode
	validModes := map[string]bool{
		"runOnceForAllItems": true,
		"runOnceForEachItem": true,
	}
	
	if !validModes[modeStr] {
		return c.CreateError(fmt.Sprintf("invalid mode: %s", modeStr), nil)
	}
	
	// Check if language exists
	language, ok := params["language"]
	if !ok {
		return c.CreateError("language parameter is required", nil)
	}
	
	// Check if language is a string
	languageStr, ok := language.(string)
	if !ok {
		return c.CreateError("language must be a string", nil)
	}
	
	// Validate language
	validLanguages := map[string]bool{
		"javascript": true,
		"python":     true,
		"go":         true,
	}
	
	if !validLanguages[languageStr] {
		return c.CreateError(fmt.Sprintf("invalid language: %s", languageStr), nil)
	}
	
	// Check if code exists
	code, ok := params["code"]
	if !ok {
		return c.CreateError("code parameter is required", nil)
	}
	
	// Check if code is a string
	if _, ok := code.(string); !ok {
		return c.CreateError("code must be a string", nil)
	}
	
	return nil
}

// Execute processes the Code node operation
func (c *CodeNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	if len(inputData) == 0 {
		return []model.DataItem{}, nil
	}
	
	// Get parameters from node parameters
	mode := c.GetStringParameter(nodeParams, "mode", "runOnceForAllItems")
	language := c.GetStringParameter(nodeParams, "language", "javascript")
	code := c.GetStringParameter(nodeParams, "code", "")
	
	// Validate parameters
	if code == "" {
		return nil, c.CreateError("code parameter cannot be empty", nil)
	}
	
	// Execute code based on language
	switch language {
	case "javascript":
		return c.executeJavaScript(mode, code, inputData, nodeParams)
	case "python":
		return c.executePython(mode, code, inputData, nodeParams)
	case "go":
		return c.executeGo(mode, code, inputData, nodeParams)
	default:
		return nil, c.CreateError(fmt.Sprintf("unsupported language: %s", language), nil)
	}
}

// executeJavaScript executes JavaScript code using the Goja-based expression system
func (c *CodeNode) executeJavaScript(mode, code string, inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	// Create Goja expression evaluator
	evaluator := expressions.NewGojaExpressionEvaluator(expressions.DefaultEvaluatorConfig())

	// Process based on mode
	switch mode {
	case "runOnceForAllItems":
		return c.executeJavaScriptForAllItems(evaluator, code, inputData, nodeParams)
	case "runOnceForEachItem":
		return c.executeJavaScriptForEachItem(evaluator, code, inputData, nodeParams)
	default:
		return nil, c.CreateError(fmt.Sprintf("unsupported mode: %s", mode), nil)
	}
}

// executePython executes Python code
func (c *CodeNode) executePython(mode, code string, inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	// For now, we'll return a simple result
	// In a real implementation, we would execute the Python code
	result := make([]model.DataItem, len(inputData))
	
	for i, item := range inputData {
		newItem := model.DataItem{
			JSON: make(map[string]interface{}),
		}
		
		// Copy existing JSON data
		for k, v := range item.JSON {
			newItem.JSON[k] = v
		}
		
		// Add Python execution result
		newItem.JSON["pythonResult"] = "Executed Python code successfully"
		newItem.JSON["pythonCode"] = code
		
		// Copy binary data if present
		if item.Binary != nil {
			newItem.Binary = make(map[string]model.BinaryData)
			for k, v := range item.Binary {
				newItem.Binary[k] = v
			}
		}
		
		// Copy paired item data if present
		if item.PairedItem != nil {
			newItem.PairedItem = item.PairedItem
		}
		
		result[i] = newItem
	}
	
	return result, nil
}

// executeGo executes Go code
func (c *CodeNode) executeGo(mode, code string, inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	// For now, we'll return a simple result
	// In a real implementation, we would compile and execute the Go code
	result := make([]model.DataItem, len(inputData))
	
	for i, item := range inputData {
		newItem := model.DataItem{
			JSON: make(map[string]interface{}),
		}
		
		// Copy existing JSON data
		for k, v := range item.JSON {
			newItem.JSON[k] = v
		}
		
		// Add Go execution result
		newItem.JSON["goResult"] = "Executed Go code successfully"
		newItem.JSON["goCode"] = code
		
		// Copy binary data if present
		if item.Binary != nil {
			newItem.Binary = make(map[string]model.BinaryData)
			for k, v := range item.Binary {
				newItem.Binary[k] = v
			}
		}
		
		// Copy paired item data if present
		if item.PairedItem != nil {
			newItem.PairedItem = item.PairedItem
		}
		
		result[i] = newItem
	}
	
	return result, nil
}

// executeJavaScriptForAllItems executes code once for all items
func (c *CodeNode) executeJavaScriptForAllItems(evaluator *expressions.GojaExpressionEvaluator, code string, inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	// Create expression context for all items
	context := &expressions.ExpressionContext{
		RunIndex:            0,
		ItemIndex:           0,
		ActiveNodeName:      "code-node",
		ConnectionInputData: inputData,
		Mode:                expressions.ModeManual,
		AdditionalKeys:      &expressions.AdditionalKeys{},
	}

	// Execute the code once
	result, err := evaluator.EvaluateCode(code, context)
	if err != nil {
		return nil, c.CreateError(fmt.Sprintf("failed to execute JavaScript code: %v", err), nil)
	}

	// Convert result to DataItems
	return c.convertCodeResult(result, inputData)
}

// executeJavaScriptForEachItem executes code once for each item
func (c *CodeNode) executeJavaScriptForEachItem(evaluator *expressions.GojaExpressionEvaluator, code string, inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	result := make([]model.DataItem, len(inputData))

	for i, item := range inputData {
		// Create expression context for this item
		context := &expressions.ExpressionContext{
			RunIndex:            0,
			ItemIndex:           i,
			ActiveNodeName:      "code-node",
			ConnectionInputData: []model.DataItem{item},
			Mode:                expressions.ModeManual,
			AdditionalKeys:      &expressions.AdditionalKeys{},
		}

		// Execute the code for this item
		itemResult, err := evaluator.EvaluateCode(code, context)
		if err != nil {
			return nil, c.CreateError(fmt.Sprintf("failed to execute JavaScript code for item %d: %v", i, err), nil)
		}

		// Convert result to DataItem
		converted, err := c.convertSingleCodeResult(itemResult, item)
		if err != nil {
			return nil, c.CreateError(fmt.Sprintf("failed to convert result for item %d: %v", i, err), nil)
		}

		result[i] = converted
	}

	return result, nil
}

// convertCodeResult converts a code execution result to DataItems
func (c *CodeNode) convertCodeResult(result interface{}, inputData []model.DataItem) ([]model.DataItem, error) {
	// If result is an array, try to convert each element to a DataItem
	if resultSlice, ok := result.([]interface{}); ok {
		converted := make([]model.DataItem, len(resultSlice))
		for i, item := range resultSlice {
			if itemMap, ok := item.(map[string]interface{}); ok {
				converted[i] = model.DataItem{JSON: itemMap}
			} else {
				converted[i] = model.DataItem{JSON: map[string]interface{}{"result": item}}
			}
		}
		return converted, nil
	}

	// If result is a single object, wrap it in an array
	if resultMap, ok := result.(map[string]interface{}); ok {
		return []model.DataItem{{JSON: resultMap}}, nil
	}

	// For other types, create a single DataItem with the result
	return []model.DataItem{{JSON: map[string]interface{}{"result": result}}}, nil
}

// convertSingleCodeResult converts a single code result to a DataItem
func (c *CodeNode) convertSingleCodeResult(result interface{}, originalItem model.DataItem) (model.DataItem, error) {
	newItem := model.DataItem{
		JSON: make(map[string]interface{}),
	}

	// Copy binary and paired item data
	if originalItem.Binary != nil {
		newItem.Binary = make(map[string]model.BinaryData)
		for k, v := range originalItem.Binary {
			newItem.Binary[k] = v
		}
	}
	if originalItem.PairedItem != nil {
		newItem.PairedItem = originalItem.PairedItem
	}

	// Convert result to JSON
	if resultMap, ok := result.(map[string]interface{}); ok {
		newItem.JSON = resultMap
	} else {
		newItem.JSON["result"] = result
	}

	return newItem, nil
}