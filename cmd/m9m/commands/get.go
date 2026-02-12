package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/neul-labs/m9m/internal/service"
)

var getCmd = &cobra.Command{
	Use:   "get <workflow>",
	Short: "Get workflow details",
	Long: `Get detailed information about a workflow by name or ID.

Examples:
  m9m get my-workflow
  m9m get workflow-id
  m9m get my-workflow --output json`,
	Args: cobra.ExactArgs(1),
	Run:  runGet,
}

func runGet(cmd *cobra.Command, args []string) {
	workflowName := args[0]

	client := service.NewClient(nil)

	resp, err := client.GetWorkflow(workspaceFlag, workflowName)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("\nMake sure a workspace is initialized. Run 'm9m init' first.")
		os.Exit(1)
	}

	if !resp.Success {
		fmt.Printf("Error: %s\n", resp.Error)
		os.Exit(1)
	}

	workflow, ok := resp.Data.(map[string]interface{})
	if !ok {
		fmt.Println("Error: Invalid response format")
		os.Exit(1)
	}

	if outputFlag == "json" {
		output, _ := json.MarshalIndent(workflow, "", "  ")
		fmt.Println(string(output))
		return
	}

	// Human-readable output
	name, _ := workflow["name"].(string)
	id, _ := workflow["id"].(string)
	description, _ := workflow["description"].(string)
	active, _ := workflow["active"].(bool)
	createdAt, _ := workflow["createdAt"].(string)
	updatedAt, _ := workflow["updatedAt"].(string)

	fmt.Printf("Workflow: %s\n", name)
	fmt.Printf("==========%s\n", repeatChar('=', len(name)))
	fmt.Printf("ID: %s\n", id)
	if description != "" {
		fmt.Printf("Description: %s\n", description)
	}
	fmt.Printf("Active: %v\n", active)
	fmt.Printf("Created: %s\n", formatDate(createdAt))
	fmt.Printf("Updated: %s\n", formatDate(updatedAt))

	// Nodes
	if nodes, ok := workflow["nodes"].([]interface{}); ok {
		fmt.Printf("\nNodes (%d):\n", len(nodes))
		for i, node := range nodes {
			nodeMap, ok := node.(map[string]interface{})
			if !ok {
				continue
			}
			nodeName, _ := nodeMap["name"].(string)
			nodeType, _ := nodeMap["type"].(string)
			fmt.Printf("  %d. %s (%s)\n", i+1, nodeName, nodeType)
		}
	}

	// Connections summary
	if connections, ok := workflow["connections"].(map[string]interface{}); ok {
		fmt.Printf("\nConnections:\n")
		for source, connData := range connections {
			if connMap, ok := connData.(map[string]interface{}); ok {
				if main, ok := connMap["main"].([]interface{}); ok {
					for _, outputs := range main {
						if outputArr, ok := outputs.([]interface{}); ok {
							for _, conn := range outputArr {
								if connObj, ok := conn.(map[string]interface{}); ok {
									target, _ := connObj["node"].(string)
									fmt.Printf("  %s -> %s\n", source, target)
								}
							}
						}
					}
				}
			}
		}
	}
}

func repeatChar(char rune, count int) string {
	result := make([]rune, count)
	for i := range result {
		result[i] = char
	}
	return string(result)
}

func formatDate(dateStr string) string {
	if len(dateStr) > 19 {
		return dateStr[:19]
	}
	return dateStr
}
