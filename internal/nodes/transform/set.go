/*
Package transform provides data transformation node implementations for n8n-go.
*/
package transform

import (
	"fmt"
	"strings"
	
	"github.com/neul-labs/m9m/internal/expressions"
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// SetNode implements the Set node functionality for assigning values to fields
type SetNode struct {
	*base.BaseNode
	evaluator *expressions.GojaExpressionEvaluator
}

// NewSetNode creates a new Set node
func NewSetNode() *SetNode {
	description := base.NodeDescription{
		Name:        "Set",
		Description: "Sets values on items",
		Category:    "Data Transformation",
	}
	
	return &SetNode{
		BaseNode:  base.NewBaseNode(description),
		evaluator: expressions.NewGojaExpressionEvaluator(expressions.DefaultEvaluatorConfig()),
	}
}

// Description returns the node description
func (s *SetNode) Description() base.NodeDescription {
	return s.BaseNode.Description()
}

// ValidateParameters validates Set node parameters
func (s *SetNode) ValidateParameters(params map[string]interface{}) error {
	if params == nil {
		return s.CreateError("parameters cannot be nil", nil)
	}
	
	// Check if assignments exist
	assignments, ok := params["assignments"]
	if !ok {
		return s.CreateError("assignments parameter is required", nil)
	}
	
	// Check if assignments is an array
	assignmentsArr, ok := assignments.([]interface{})
	if !ok {
		return s.CreateError("assignments must be an array", nil)
	}
	
	// Validate each assignment
	for i, assignment := range assignmentsArr {
		assignmentMap, ok := assignment.(map[string]interface{})
		if !ok {
			return s.CreateError(fmt.Sprintf("assignment %d must be an object", i), nil)
		}
		
		// Check required fields
		if _, ok := assignmentMap["name"]; !ok {
			return s.CreateError(fmt.Sprintf("assignment %d missing 'name' field", i), nil)
		}
		
		if _, ok := assignmentMap["value"]; !ok {
			return s.CreateError(fmt.Sprintf("assignment %d missing 'value' field", i), nil)
		}
	}
	
	return nil
}

// Execute processes the Set node operation
func (s *SetNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	if len(inputData) == 0 {
		return []model.DataItem{}, nil
	}

	// Get assignments
	assignments, ok := nodeParams["assignments"].([]interface{})
	if !ok {
		return nil, s.CreateError("assignments parameter is required", nil)
	}

	// Process each input data item
	result := make([]model.DataItem, len(inputData))

	for i, item := range inputData {
		// Create expression context for current item
		context := &expressions.ExpressionContext{
			ActiveNodeName:      "Set",
			RunIndex:           0,
			ItemIndex:          i,
			Mode:               expressions.ModeManual,
			ConnectionInputData: []model.DataItem{item},
			Workflow: &model.Workflow{
				Name: "Set Processing",
			},
			AdditionalKeys: &expressions.AdditionalKeys{
				ExecutionId: "set-processing",
			},
		}

		// Copy the original item
		newItem := model.DataItem{
			JSON: make(map[string]interface{}),
		}

		// Copy existing JSON data
		for k, v := range item.JSON {
			newItem.JSON[k] = v
		}
		
		// Apply each assignment
		for _, assignment := range assignments {
			assignmentMap, ok := assignment.(map[string]interface{})
			if !ok {
				continue
			}

			name, nameOk := assignmentMap["name"].(string)
			value := assignmentMap["value"]

			if !nameOk {
				continue
			}

			// Check if the value is a string that might contain expressions
			if valueStr, ok := value.(string); ok {
				// Handle n8n-style expressions (starting with =)
				if strings.HasPrefix(valueStr, "=") {
					// Process n8n-style expression - remove the = and wrap in {{ }}
					expressionToEvaluate := "{{ " + strings.TrimSpace(valueStr[1:]) + " }}"

					// Evaluate the expression
					evaluatedValue, err := s.evaluator.EvaluateExpression(expressionToEvaluate, context)
					if err != nil {
						return nil, s.CreateError(fmt.Sprintf("failed to evaluate expression '%s': %v", valueStr, err), nil)
					}
					newItem.JSON[name] = evaluatedValue
				} else {
					// Check if it's already a proper n8n expression {{ }}
					if strings.HasPrefix(valueStr, "{{") && strings.HasSuffix(valueStr, "}}") {
						// Evaluate the expression
						evaluatedValue, err := s.evaluator.EvaluateExpression(valueStr, context)
						if err != nil {
							return nil, s.CreateError(fmt.Sprintf("failed to evaluate expression '%s': %v", valueStr, err), nil)
						}
						newItem.JSON[name] = evaluatedValue
					} else {
						// Use the literal value
						newItem.JSON[name] = value
					}
				}
			} else {
				// Use the literal value
				newItem.JSON[name] = value
			}
		}
		
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