package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/neul-labs/m9m/internal/engine"
	"github.com/neul-labs/m9m/internal/model"
)

// NodeTypeInfo describes a node type
type NodeTypeInfo struct {
	Name        string                   `json:"name"`
	DisplayName string                   `json:"displayName"`
	Description string                   `json:"description"`
	Category    string                   `json:"category"`
	Version     int                      `json:"version"`
	Properties  []map[string]interface{} `json:"properties,omitempty"`
}

// NodeCategory describes a category of nodes
type NodeCategory struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Count       int    `json:"count"`
}

// getNodeCatalog returns the live node catalog from the engine registry.
func getNodeCatalog() []NodeTypeInfo {
	eng := engine.NewWorkflowEngine()
	RegisterAllNodes(eng)

	registered := eng.GetRegisteredNodeTypes()
	catalog := make([]NodeTypeInfo, 0, len(registered))
	for _, nt := range registered {
		info := NodeTypeInfo{
			Name:        nt.TypeID,
			DisplayName: nt.DisplayName,
			Description: nt.Description,
			Category:    nt.Category,
			Version:     nt.Version,
		}
		if len(nt.Properties) > 0 {
			props := make([]map[string]interface{}, 0, len(nt.Properties))
			for _, p := range nt.Properties {
				props = append(props, map[string]interface{}{
					"name":        p.Name,
					"displayName": p.DisplayName,
					"type":        p.Type,
					"description": p.Description,
					"required":    p.Required,
				})
			}
			info.Properties = props
		}
		catalog = append(catalog, info)
	}

	sort.Slice(catalog, func(i, j int) bool {
		if catalog[i].Category != catalog[j].Category {
			return catalog[i].Category < catalog[j].Category
		}
		return catalog[i].Name < catalog[j].Name
	})

	return catalog
}

// getCategoryCatalog builds categories from the live node catalog.
func getCategoryCatalog() []NodeCategory {
	catalog := getNodeCatalog()
	catMap := make(map[string]int)
	for _, n := range catalog {
		cat := n.Category
		if cat == "" {
			cat = "other"
		}
		catMap[cat]++
	}

	var categories []NodeCategory
	for name, count := range catMap {
		categories = append(categories, NodeCategory{
			Name:  name,
			Count: count,
		})
	}
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Name < categories[j].Name
	})
	return categories
}

var (
	nodeCategory string
	nodeSearch   string
)

var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Manage node types",
	Long: `Manage and discover available node types.

Node types are the building blocks of workflows. Use these commands
to discover what nodes are available and their parameters.`,
}

var nodeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available node types",
	Long: `List all available node types, optionally filtered by category.

Examples:
  m9m node list                    List all nodes
  m9m node list --category ai      List AI nodes
  m9m node list --search http      Search for nodes`,
	Run: runNodeList,
}

var nodeCategoriesCmd = &cobra.Command{
	Use:   "categories",
	Short: "List node categories",
	Run:   runNodeCategories,
}

var nodeInfoCmd = &cobra.Command{
	Use:   "info <node-type>",
	Short: "Get detailed info about a node type",
	Long: `Get detailed information about a specific node type.

Examples:
  m9m node info n8n-nodes-base.httpRequest
  m9m node info n8n-nodes-base.openAi`,
	Args: cobra.ExactArgs(1),
	Run:  runNodeInfo,
}

var (
	nodeCreateFrom string
	nodeTestInput  string
)

var nodeCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a custom node from a script",
	Long: `Create a custom node from a JavaScript or Python script.

Custom nodes can be used in workflows just like built-in nodes.
The script must export a module with name, description, and execute function.

JavaScript example (mynode.js):
  module.exports = {
    name: "myCustomNode",
    description: "My custom processing node",
    category: "transform",
    execute: function(input, params) {
      return input.map(item => ({
        ...item,
        processed: true
      }));
    }
  };

Examples:
  m9m node create --from mynode.js
  m9m node create --from processor.py`,
	Run: runNodeCreate,
}

var nodeTestCmd = &cobra.Command{
	Use:   "test <node-type>",
	Short: "Test a node with sample data",
	Long: `Test a node type with sample input data.

This executes the node in isolation to verify it works correctly.

Examples:
  m9m node test n8n-nodes-base.set --input '{"name": "test"}'
  m9m node test n8n-nodes-base.filter --input @data.json`,
	Args: cobra.ExactArgs(1),
	Run:  runNodeTest,
}

func init() {
	nodeListCmd.Flags().StringVar(&nodeCategory, "category", "", "Filter by category")
	nodeListCmd.Flags().StringVar(&nodeSearch, "search", "", "Search by name or description")

	nodeCreateCmd.Flags().StringVar(&nodeCreateFrom, "from", "", "Script file to create node from (required)")
	nodeCreateCmd.MarkFlagRequired("from")

	nodeTestCmd.Flags().StringVar(&nodeTestInput, "input", "", "Input data as JSON or @file.json")

	nodeCmd.AddCommand(nodeListCmd)
	nodeCmd.AddCommand(nodeCategoriesCmd)
	nodeCmd.AddCommand(nodeInfoCmd)
	nodeCmd.AddCommand(nodeCreateCmd)
	nodeCmd.AddCommand(nodeTestCmd)
}

func runNodeList(cmd *cobra.Command, args []string) {
	var filtered []NodeTypeInfo

	for _, node := range getNodeCatalog() {
		// Category filter
		if nodeCategory != "" && node.Category != nodeCategory {
			continue
		}

		// Search filter
		if nodeSearch != "" {
			query := strings.ToLower(nodeSearch)
			if !strings.Contains(strings.ToLower(node.Name), query) &&
				!strings.Contains(strings.ToLower(node.DisplayName), query) &&
				!strings.Contains(strings.ToLower(node.Description), query) {
				continue
			}
		}

		filtered = append(filtered, node)
	}

	if outputFlag == "json" {
		data, _ := json.MarshalIndent(filtered, "", "  ")
		fmt.Println(string(data))
		return
	}

	// Table output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tDISPLAY NAME\tCATEGORY\tDESCRIPTION")
	fmt.Fprintln(w, "----\t------------\t--------\t-----------")
	for _, node := range filtered {
		desc := node.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", node.Name, node.DisplayName, node.Category, desc)
	}
	w.Flush()

	fmt.Printf("\nTotal: %d nodes\n", len(filtered))
}

func runNodeCategories(cmd *cobra.Command, args []string) {
	categories := getCategoryCatalog()

	if outputFlag == "json" {
		data, _ := json.MarshalIndent(categories, "", "  ")
		fmt.Println(string(data))
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CATEGORY\tCOUNT")
	fmt.Fprintln(w, "--------\t-----")
	for _, cat := range categories {
		fmt.Fprintf(w, "%s\t%d\n", cat.Name, cat.Count)
	}
	w.Flush()
}

func runNodeInfo(cmd *cobra.Command, args []string) {
	nodeType := args[0]

	var found *NodeTypeInfo
	for _, node := range getNodeCatalog() {
		if node.Name == nodeType {
			found = &node
			break
		}
	}

	if found == nil {
		fmt.Printf("Error: Node type not found: %s\n", nodeType)
		fmt.Println("\nUse 'm9m node list' to see available node types.")
		os.Exit(1)
	}

	if outputFlag == "json" {
		data, _ := json.MarshalIndent(found, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Printf("Node Type: %s\n", found.Name)
	fmt.Printf("Display Name: %s\n", found.DisplayName)
	fmt.Printf("Category: %s\n", found.Category)
	fmt.Printf("Version: %d\n", found.Version)
	fmt.Printf("Description: %s\n", found.Description)

	if len(found.Properties) > 0 {
		fmt.Println("\nProperties:")
		for _, prop := range found.Properties {
			name := prop["name"].(string)
			propType := prop["type"].(string)
			desc := ""
			if d, ok := prop["description"].(string); ok {
				desc = d
			}
			required := ""
			if r, ok := prop["required"].(bool); ok && r {
				required = " [required]"
			}
			fmt.Printf("  - %s (%s)%s\n", name, propType, required)
			if desc != "" {
				fmt.Printf("    %s\n", desc)
			}
		}
	}
}

func runNodeCreate(cmd *cobra.Command, args []string) {
	// Read the script file
	scriptData, err := os.ReadFile(nodeCreateFrom)
	if err != nil {
		fmt.Printf("Error: Cannot read script file: %v\n", err)
		os.Exit(1)
	}

	// Determine script type from extension
	scriptType := "javascript"
	if strings.HasSuffix(nodeCreateFrom, ".py") {
		scriptType = "python"
	}

	// For now, show what would be created
	// Full implementation requires the plugin registry
	fmt.Printf("Creating custom node from: %s\n", nodeCreateFrom)
	fmt.Printf("Script type: %s\n", scriptType)
	fmt.Printf("Script size: %d bytes\n", len(scriptData))
	fmt.Println()

	// Parse the script to extract node info (basic extraction)
	content := string(scriptData)

	// Try to find node name from the script
	nodeName := extractFromScript(content, "name")
	nodeDesc := extractFromScript(content, "description")
	nodeCat := extractFromScript(content, "category")

	if nodeName == "" {
		// Use filename as node name
		base := strings.TrimSuffix(nodeCreateFrom, ".js")
		base = strings.TrimSuffix(base, ".py")
		nodeName = "custom." + base
	}

	fmt.Printf("Node Name: %s\n", nodeName)
	if nodeDesc != "" {
		fmt.Printf("Description: %s\n", nodeDesc)
	}
	if nodeCat != "" {
		fmt.Printf("Category: %s\n", nodeCat)
	}

	fmt.Println()
	fmt.Println("To use this node:")
	fmt.Println("  1. Save the script to ~/.m9m/plugins/")
	fmt.Println("  2. Restart the m9m service")
	fmt.Println("  3. Use the node type in workflows")
	fmt.Println()
	fmt.Println("Note: Full custom node support requires the m9m service.")
	fmt.Println("      Start with: m9m serve")
}

func runNodeTest(cmd *cobra.Command, args []string) {
	nodeType := args[0]

	// Create engine and register nodes
	eng := engine.NewWorkflowEngine()
	RegisterAllNodes(eng)

	// Verify node exists in registry
	executor, err := eng.GetNodeExecutor(nodeType)
	if err != nil {
		fmt.Printf("Error: Node type not found: %s\n", nodeType)
		fmt.Println("\nUse 'm9m node list' to see available node types.")
		os.Exit(1)
	}

	desc := executor.Description()

	// Parse input data
	var inputData map[string]interface{}
	if nodeTestInput != "" {
		var inputJSON string
		if strings.HasPrefix(nodeTestInput, "@") {
			filePath := strings.TrimPrefix(nodeTestInput, "@")
			data, readErr := os.ReadFile(filePath)
			if readErr != nil {
				fmt.Printf("Error: Cannot read input file: %v\n", readErr)
				os.Exit(1)
			}
			inputJSON = string(data)
		} else {
			inputJSON = nodeTestInput
		}

		if jsonErr := json.Unmarshal([]byte(inputJSON), &inputData); jsonErr != nil {
			fmt.Printf("Error: Invalid input JSON: %v\n", jsonErr)
			os.Exit(1)
		}
	} else {
		inputData = map[string]interface{}{
			"test": "data",
			"id":   1,
		}
	}

	fmt.Printf("Testing node: %s (%s)\n", desc.Name, nodeType)
	fmt.Println()

	inputJSON, _ := json.MarshalIndent(inputData, "  ", "  ")
	fmt.Println("Input:")
	fmt.Println("  " + string(inputJSON))
	fmt.Println()

	// Execute the node directly
	items := []model.DataItem{{JSON: inputData}}
	result, execErr := executor.Execute(items, map[string]interface{}{})

	if execErr != nil {
		fmt.Printf("Error: %v\n", execErr)
		os.Exit(1)
	}

	fmt.Printf("Output: %d items\n", len(result))
	for i, item := range result {
		outJSON, _ := json.MarshalIndent(item.JSON, "  ", "  ")
		fmt.Printf("  [%d] %s\n", i, string(outJSON))
	}
}

// extractFromScript tries to extract a value from a JavaScript/Python script
func extractFromScript(content, field string) string {
	// Simple regex-like extraction for common patterns
	// Look for: name: "value" or name = "value"
	patterns := []string{
		field + `: "`,
		field + ": '",
		field + ` = "`,
		field + " = '",
	}

	for _, pattern := range patterns {
		idx := strings.Index(content, pattern)
		if idx != -1 {
			start := idx + len(pattern)
			// Find closing quote
			for i, char := range content[start:] {
				if char == '"' || char == '\'' {
					return content[start : start+i]
				}
			}
		}
	}
	return ""
}
