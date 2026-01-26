/*
Package transform provides data transformation node implementations for m9m.
*/
package transform

import (
	"fmt"
	
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// ItemListsNode implements the Item Lists node functionality
type ItemListsNode struct {
	*base.BaseNode
}

// NewItemListsNode creates a new Item Lists node
func NewItemListsNode() *ItemListsNode {
	description := base.NodeDescription{
		Name:        "Item Lists",
		Description: "Splits or combines items in a list",
		Category:    "Data Transformation",
	}
	
	return &ItemListsNode{
		BaseNode: base.NewBaseNode(description),
	}
}

// Description returns the node description
func (i *ItemListsNode) Description() base.NodeDescription {
	return i.BaseNode.Description()
}

// ValidateParameters validates Item Lists node parameters
func (i *ItemListsNode) ValidateParameters(params map[string]interface{}) error {
	// Nil params are allowed (defaults to combine mode)
	if params == nil {
		return nil
	}
	
	// Check if mode exists
	mode, ok := params["mode"]
	if !ok {
		// Mode is optional, default to "combine"
		return nil
	}
	
	// Check if mode is a string
	modeStr, ok := mode.(string)
	if !ok {
		return i.CreateError("mode must be a string", nil)
	}
	
	// Validate mode
	validModes := map[string]bool{
		"combine": true,
		"split":   true,
	}
	
	if !validModes[modeStr] {
		return i.CreateError(fmt.Sprintf("invalid mode: %s", modeStr), nil)
	}
	
	return nil
}

// Execute processes the Item Lists node operation
func (i *ItemListsNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	if len(inputData) == 0 {
		return []model.DataItem{}, nil
	}
	
	// Get mode from node parameters, default to "combine"
	mode := i.GetStringParameter(nodeParams, "mode", "combine")
	
	switch mode {
	case "combine":
		return i.combineMode(inputData), nil
	case "split":
		return i.splitMode(inputData), nil
	default:
		// Default to combine mode
		return i.combineMode(inputData), nil
	}
}

// combineMode combines multiple items into a single item with an array
func (i *ItemListsNode) combineMode(inputData []model.DataItem) []model.DataItem {
	if len(inputData) == 0 {
		return []model.DataItem{}
	}
	
	// Create items array
	items := make([]interface{}, len(inputData))
	for j, item := range inputData {
		items[j] = item.JSON
	}
	
	// Create combined item
	combinedItem := model.DataItem{
		JSON: map[string]interface{}{
			"items": items,
		},
	}
	
	// Copy binary data if present
	if len(inputData) > 0 && inputData[0].Binary != nil {
		combinedItem.Binary = make(map[string]model.BinaryData)
		for k, v := range inputData[0].Binary {
			combinedItem.Binary[k] = v
		}
	}
	
	// Copy paired item data if present
	if len(inputData) > 0 && inputData[0].PairedItem != nil {
		combinedItem.PairedItem = inputData[0].PairedItem
	}
	
	return []model.DataItem{combinedItem}
}

// splitMode splits items based on array fields
func (i *ItemListsNode) splitMode(inputData []model.DataItem) []model.DataItem {
	if len(inputData) == 0 {
		return []model.DataItem{}
	}
	
	// For each input item, check if it has an 'items' field
	var result []model.DataItem
	
	for _, item := range inputData {
		// Check if there's an 'items' field with an array
		if items, ok := item.JSON["items"]; ok {
			if itemsArr, ok := items.([]interface{}); ok {
				// Create a new item for each element in the array
				for _, arrayItem := range itemsArr {
					if itemMap, ok := arrayItem.(map[string]interface{}); ok {
						newItem := model.DataItem{
							JSON: itemMap,
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
						
						result = append(result, newItem)
					}
				}
			} else {
				// If items is not an array, pass the item through
				result = append(result, item)
			}
		} else {
			// If no 'items' field, pass the item through
			result = append(result, item)
		}
	}
	
	return result
}