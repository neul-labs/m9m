/*
Package m9m provides a public Go SDK for embedding the m9m workflow automation engine.

m9m is a high-performance workflow automation platform that provides 95% backend
feature parity with n8n while offering significant performance improvements.

# Quick Start

Create an engine, register nodes, and execute workflows:

	import "github.com/neul-labs/m9m/pkg/m9m"

	// Create a new workflow engine
	engine := m9m.New()

	// Register a custom node
	engine.RegisterNode("custom.myNode", &MyCustomNode{})

	// Load and execute a workflow
	workflow, err := m9m.LoadWorkflow("workflow.json")
	if err != nil {
		log.Fatal(err)
	}

	result, err := engine.Execute(workflow, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Execution result: %+v\n", result.Data)

# Custom Nodes

Implement the NodeExecutor interface to create custom nodes:

	type MyCustomNode struct {
		*m9m.BaseNode
	}

	func NewMyCustomNode() *MyCustomNode {
		return &MyCustomNode{
			BaseNode: m9m.NewBaseNode(m9m.NodeDescription{
				Name:        "My Custom Node",
				Description: "Does custom processing",
				Category:    "transform",
			}),
		}
	}

	func (n *MyCustomNode) Execute(input []m9m.DataItem, params map[string]interface{}) ([]m9m.DataItem, error) {
		// Custom execution logic
		return input, nil
	}

# Credential Management

Set up credential management for secure API access:

	credManager, err := m9m.NewCredentialManager()
	if err != nil {
		log.Fatal(err)
	}

	engine.SetCredentialManager(credManager)

# Thread Safety

The m9m engine is thread-safe and can execute multiple workflows concurrently:

	results, err := engine.ExecuteParallel(workflows, inputs)

# Performance

m9m offers significant performance improvements over similar solutions:
  - 5-10x faster execution
  - 70% lower memory usage (~150MB vs 512MB)
  - Sub-second startup time

For more information, see the project documentation at https://github.com/neul-labs/m9m
*/
package m9m
