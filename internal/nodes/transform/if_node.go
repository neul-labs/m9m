package transform

import (
	"fmt"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// IfNode routes items based on conditions, tagging each with the evaluation result.
type IfNode struct {
	*base.BaseNode
}

// NewIfNode creates a new IF node.
func NewIfNode() *IfNode {
	return &IfNode{
		BaseNode: base.NewBaseNode(base.NodeDescription{
			Name:        "IF",
			Description: "Routes items based on conditions",
			Category:    "Data Transformation",
		}),
	}
}

// Execute evaluates conditions for each input item.
// Items that pass conditions are returned. If returnBothBranches is true,
// all items are returned with an _ifResult metadata field.
func (n *IfNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	if len(inputData) == 0 {
		return []model.DataItem{}, nil
	}

	conditions, ok := nodeParams["conditions"]
	if !ok {
		return nil, n.CreateError("conditions parameter is required", nil)
	}

	conditionsArr, ok := conditions.([]interface{})
	if !ok {
		return nil, n.CreateError("conditions must be an array", nil)
	}

	combiner := n.GetStringParameter(nodeParams, "combiner", "and")
	returnBoth := n.GetBoolParameter(nodeParams, "returnBothBranches", false)

	var trueItems, falseItems []model.DataItem

	for _, item := range inputData {
		passes := EvaluateConditions(item, conditionsArr, combiner)
		if passes {
			trueItems = append(trueItems, item)
		} else {
			falseItems = append(falseItems, item)
		}
	}

	if returnBoth {
		var result []model.DataItem
		for _, item := range trueItems {
			tagged := copyDataItem(item)
			tagged.JSON["_ifResult"] = true
			result = append(result, tagged)
		}
		for _, item := range falseItems {
			tagged := copyDataItem(item)
			tagged.JSON["_ifResult"] = false
			result = append(result, tagged)
		}
		return result, nil
	}

	return trueItems, nil
}

// ValidateParameters validates IF node parameters.
func (n *IfNode) ValidateParameters(params map[string]interface{}) error {
	if params == nil {
		return n.CreateError("parameters cannot be nil", nil)
	}

	conditions, ok := params["conditions"]
	if !ok {
		return n.CreateError("conditions parameter is required", nil)
	}

	combiner := n.GetStringParameter(params, "combiner", "and")
	if err := ValidateConditions(conditions, combiner); err != nil {
		return fmt.Errorf("node IF error: %w", err)
	}

	return nil
}

func copyDataItem(item model.DataItem) model.DataItem {
	newJSON := make(map[string]interface{}, len(item.JSON))
	for k, v := range item.JSON {
		newJSON[k] = v
	}
	return model.DataItem{JSON: newJSON}
}
