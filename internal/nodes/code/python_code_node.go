package code

import (
	"context"
	"fmt"
	"strings"

	"github.com/dipankar/m9m/internal/model"
	"github.com/dipankar/m9m/internal/nodes/base"
	"github.com/dipankar/m9m/internal/runtime"
)

// PythonCodeNode executes Python code with n8n compatibility
type PythonCodeNode struct {
	*base.BaseNode
	pythonRuntime *runtime.PythonRuntime
}

// NewPythonCodeNode creates a new Python code execution node
func NewPythonCodeNode() *PythonCodeNode {
	node := &PythonCodeNode{
		BaseNode: base.NewBaseNode(base.NodeDescription{
			Name:        "PythonCode",
			Description: "Execute Python code with n8n compatibility",
			Category:    "code",
		}),
	}

	// Initialize Python runtime
	pyRuntime, err := runtime.NewPythonRuntime()
	if err != nil {
		// Log error but continue - will handle in Execute
		node.pythonRuntime = nil
	} else {
		node.pythonRuntime = pyRuntime
	}

	return node
}

// Execute runs the Python code
func (n *PythonCodeNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	if n.pythonRuntime == nil {
		// Try to initialize runtime if it failed during construction
		pyRuntime, err := runtime.NewPythonRuntime()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Python runtime: %w", err)
		}
		n.pythonRuntime = pyRuntime

		// Initialize the runtime
		if err := n.pythonRuntime.Initialize(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to initialize Python runtime: %w", err)
		}
	}

	// Get parameters
	pythonCode, _ := nodeParams["pythonCode"].(string)
	if pythonCode == "" {
		pythonCode = "return $input"
	}

	mode, _ := nodeParams["mode"].(string)
	if mode == "" {
		mode = "runOnce"
	}

	continueOnFail, _ := nodeParams["continueOnFail"].(bool)

	var outputItems []model.DataItem

	if mode == "runOnce" {
		// Run once for all items
		allItems := make([]interface{}, len(inputData))
		for i, item := range inputData {
			allItems[i] = item.JSON
		}

		// Execute Python code
		result, err := n.pythonRuntime.Execute(pythonCode, allItems, map[string]interface{}{
			"mode": mode,
		})

		if err != nil {
			if continueOnFail {
				// Return error as data
				outputItems = append(outputItems, model.DataItem{
					JSON: map[string]interface{}{
						"error": err.Error(),
					},
				})
			} else {
				return nil, fmt.Errorf("Python execution failed: %w", err)
			}
		} else if result.Type == "error" {
			if continueOnFail {
				outputItems = append(outputItems, model.DataItem{
					JSON: map[string]interface{}{
						"error": result.Error,
					},
				})
			} else {
				return nil, fmt.Errorf("Python code error: %s", result.Error)
			}
		} else {
			// Process result
			if result.Output != nil {
				switch v := result.Output.(type) {
				case []interface{}:
					// Multiple items returned
					for _, item := range v {
						if itemMap, ok := item.(map[string]interface{}); ok {
							outputItems = append(outputItems, model.DataItem{JSON: itemMap})
						} else {
							outputItems = append(outputItems, model.DataItem{
								JSON: map[string]interface{}{"value": item},
							})
						}
					}
				case map[string]interface{}:
					// Single object returned
					outputItems = append(outputItems, model.DataItem{JSON: v})
				default:
					// Primitive value returned
					outputItems = append(outputItems, model.DataItem{
						JSON: map[string]interface{}{"value": v},
					})
				}
			} else {
				// No output, pass through input
				outputItems = inputData
			}
		}
	} else {
		// Run once for each item
		for i, item := range inputData {
			result, err := n.pythonRuntime.Execute(pythonCode, item.JSON, map[string]interface{}{
				"mode":  mode,
				"index": i,
			})

			if err != nil {
				if continueOnFail {
					outputItems = append(outputItems, model.DataItem{
						JSON: map[string]interface{}{
							"error":        err.Error(),
							"originalItem": item.JSON,
						},
					})
				} else {
					return nil, fmt.Errorf("Python execution failed for item %d: %w", i, err)
				}
			} else if result.Type == "error" {
				if continueOnFail {
					outputItems = append(outputItems, model.DataItem{
						JSON: map[string]interface{}{
							"error":        result.Error,
							"originalItem": item.JSON,
						},
					})
				} else {
					return nil, fmt.Errorf("Python code error for item %d: %s", i, result.Error)
				}
			} else {
				// Process result
				if result.Output != nil {
					if outputMap, ok := result.Output.(map[string]interface{}); ok {
						outputItems = append(outputItems, model.DataItem{JSON: outputMap})
					} else {
						outputItems = append(outputItems, model.DataItem{
							JSON: map[string]interface{}{"value": result.Output},
						})
					}
				} else {
					// No output, pass through item
					outputItems = append(outputItems, item)
				}
			}
		}
	}

	return outputItems, nil
}

// ValidateParameters validates node parameters
func (n *PythonCodeNode) ValidateParameters(params map[string]interface{}) error {
	pythonCode, ok := params["pythonCode"].(string)
	if !ok || strings.TrimSpace(pythonCode) == "" {
		return fmt.Errorf("Python code is required")
	}

	mode, ok := params["mode"].(string)
	if ok && mode != "runOnce" && mode != "runForEach" {
		return fmt.Errorf("invalid mode: %s", mode)
	}

	return nil
}

// Cleanup cleans up resources
func (n *PythonCodeNode) Cleanup() error {
	if n.pythonRuntime != nil {
		return n.pythonRuntime.Cleanup()
	}
	return nil
}
