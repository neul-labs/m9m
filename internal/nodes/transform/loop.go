package transform

import (
	"fmt"

	"github.com/dop251/goja"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

const defaultMaxIterations = 1000

// LoopNode iterates over items or a fixed count, producing output items.
type LoopNode struct {
	*base.BaseNode
}

// NewLoopNode creates a new Loop node.
func NewLoopNode() *LoopNode {
	return &LoopNode{
		BaseNode: base.NewBaseNode(base.NodeDescription{
			Name:        "Loop",
			Description: "Iterates over items, a count, or a condition",
			Category:    "Data Transformation",
		}),
	}
}

// Execute runs the loop in the configured mode.
func (n *LoopNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	mode := n.GetStringParameter(nodeParams, "mode", "each")

	switch mode {
	case "each":
		return n.executeEach(inputData, nodeParams)
	case "times":
		return n.executeTimes(inputData, nodeParams)
	case "while":
		return n.executeWhile(inputData, nodeParams)
	default:
		return nil, n.CreateError(fmt.Sprintf("invalid mode: %s (expected each, times, or while)", mode), nil)
	}
}

func (n *LoopNode) executeEach(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	if len(inputData) == 0 {
		return []model.DataItem{}, nil
	}

	var result []model.DataItem
	for i, item := range inputData {
		out := make(map[string]interface{}, len(item.JSON)+2)
		for k, v := range item.JSON {
			out[k] = v
		}
		out["$index"] = i
		out["$item"] = item.JSON
		result = append(result, model.DataItem{JSON: out})
	}
	return result, nil
}

func (n *LoopNode) executeTimes(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	times := n.GetIntParameter(nodeParams, "times", 1)
	if times < 0 {
		return nil, n.CreateError("times must be non-negative", nil)
	}
	if times > defaultMaxIterations {
		times = defaultMaxIterations
	}

	result := make([]model.DataItem, 0, times)
	for i := 0; i < times; i++ {
		result = append(result, model.DataItem{
			JSON: map[string]interface{}{
				"$index": i,
			},
		})
	}
	return result, nil
}

func (n *LoopNode) executeWhile(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	conditionExpr := n.GetStringParameter(nodeParams, "condition", "")
	if conditionExpr == "" {
		return nil, n.CreateError("condition expression is required for while mode", nil)
	}

	maxIter := n.GetIntParameter(nodeParams, "maxIterations", defaultMaxIterations)
	if maxIter <= 0 || maxIter > defaultMaxIterations {
		maxIter = defaultMaxIterations
	}

	vm := goja.New()

	// Seed initial $item from first input item
	initialItem := map[string]interface{}{}
	if len(inputData) > 0 {
		initialItem = inputData[0].JSON
	}

	var result []model.DataItem
	for i := 0; i < maxIter; i++ {
		_ = vm.Set("$index", i)
		_ = vm.Set("$item", initialItem)

		val, err := vm.RunString(conditionExpr)
		if err != nil {
			return nil, n.CreateError(fmt.Sprintf("condition evaluation error at iteration %d: %v", i, err), nil)
		}

		if !val.ToBoolean() {
			break
		}

		out := map[string]interface{}{
			"$index": i,
		}
		for k, v := range initialItem {
			out[k] = v
		}
		result = append(result, model.DataItem{JSON: out})
	}

	return result, nil
}

// ValidateParameters validates Loop node parameters.
func (n *LoopNode) ValidateParameters(params map[string]interface{}) error {
	if params == nil {
		return nil
	}

	mode := n.GetStringParameter(params, "mode", "each")
	switch mode {
	case "each":
		// no extra params required
	case "times":
		times := n.GetIntParameter(params, "times", 1)
		if times < 0 {
			return n.CreateError("times must be non-negative", nil)
		}
	case "while":
		cond := n.GetStringParameter(params, "condition", "")
		if cond == "" {
			return n.CreateError("condition is required for while mode", nil)
		}
	default:
		return n.CreateError(fmt.Sprintf("invalid mode: %s", mode), nil)
	}

	return nil
}
