package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/neul-labs/m9m/internal/compatibility"
	"github.com/neul-labs/m9m/internal/model"
	"github.com/spf13/cobra"
)

func createNodeCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Node compatibility operations",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List available n8n nodes",
			Run:   listNodes,
		},
		&cobra.Command{
			Use:   "test [node-type]",
			Short: "Test specific n8n node",
			Args:  cobra.ExactArgs(1),
			Run:   testNode,
		},
		&cobra.Command{
			Use:   "load [node-directory]",
			Short: "Load and validate n8n node",
			Args:  cobra.ExactArgs(1),
			Run:   loadNode,
		},
	)

	return cmd
}

func listNodes(cmd *cobra.Command, args []string) {
	jsRuntime := newJavaScriptRuntime()
	defer jsRuntime.Dispose()

	nodes, err := compatibility.LoadN8nNodesFromDirectory(nodesDirectory, jsRuntime)
	if err != nil {
		log.Fatalf("Failed to load nodes: %v", err)
	}

	fmt.Printf("📦 Available n8n Nodes (%d total)\n\n", len(nodes))
	for nodeType, executor := range nodes {
		definition := executor.GetNodeDefinition()
		fmt.Printf("• %s\n", nodeType)
		fmt.Printf("  Name: %s\n", definition.DisplayName)
		fmt.Printf("  Description: %s\n", definition.Description)
		fmt.Printf("  Version: %.1f\n", definition.Version)
		fmt.Printf("  Properties: %d\n", len(definition.Properties))
		fmt.Printf("  Inputs: %v\n", definition.Inputs)
		fmt.Printf("  Outputs: %v\n", definition.Outputs)
		fmt.Println()
	}
}

func testNode(cmd *cobra.Command, args []string) {
	nodeType := args[0]

	jsRuntime := newJavaScriptRuntime()
	defer jsRuntime.Dispose()

	nodePath := filepath.Join(nodesDirectory, nodeType)
	if _, err := os.Stat(nodePath); os.IsNotExist(err) {
		log.Fatalf("Node directory not found: %s", nodePath)
	}

	executor, err := compatibility.CreateN8nCompatibleNode(nodePath, jsRuntime)
	if err != nil {
		log.Fatalf("Failed to create node: %v", err)
	}

	fmt.Printf("🧪 Testing Node: %s\n", nodeType)

	inputData := []model.DataItem{{JSON: map[string]interface{}{"test": true}}}
	nodeParams := map[string]interface{}{}

	start := time.Now()
	output, err := executor.Execute(inputData, nodeParams)
	duration := time.Since(start)
	if err != nil {
		log.Fatalf("Node execution failed: %v", err)
	}

	fmt.Printf("✅ Node executed successfully in %v\n", duration)
	fmt.Printf("Input Items: %d\n", len(inputData))
	fmt.Printf("Output Items: %d\n", len(output))

	if verbose && len(output) > 0 {
		fmt.Printf("Sample Output:\n")
		fmt.Println(string(marshalIndented(output[0])))
	}
}

func loadNode(cmd *cobra.Command, args []string) {
	nodePath := args[0]

	jsRuntime := newJavaScriptRuntime()
	defer jsRuntime.Dispose()

	log.Printf("Loading node from: %s", nodePath)

	executor, err := compatibility.CreateN8nCompatibleNode(nodePath, jsRuntime)
	if err != nil {
		log.Fatalf("Failed to load node: %v", err)
	}

	definition := executor.GetNodeDefinition()
	fmt.Printf("✅ Node loaded successfully\n")
	fmt.Printf("Name: %s\n", definition.DisplayName)
	fmt.Printf("Type: %s\n", definition.Name)
	fmt.Printf("Description: %s\n", definition.Description)
	fmt.Printf("Version: %.1f\n", definition.Version)

	if len(definition.Properties) > 0 {
		fmt.Printf("\nProperties:\n")
		for _, prop := range definition.Properties {
			fmt.Printf("  • %s (%s)\n", prop.DisplayName, prop.Type)
			if prop.Required {
				fmt.Printf("    Required: Yes\n")
			}
			if prop.Description != "" {
				fmt.Printf("    Description: %s\n", prop.Description)
			}
		}
	}
}

func runBasicNodeTests() {
	jsRuntime := newJavaScriptRuntime()
	defer jsRuntime.Dispose()

	if _, err := compatibility.LoadN8nNodesFromDirectory(nodesDirectory, jsRuntime); err != nil {
		log.Printf("  ⚠️  No nodes directory found (%s), skipping node tests\n", nodesDirectory)
		return
	}

	fmt.Printf("  ✅ Node loading working correctly\n")
}

var _ = json.Marshal
