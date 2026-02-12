package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/neul-labs/m9m/internal/service"
)

var (
	listSearch string
	listLimit  int
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all workflows",
	Long: `List all workflows in the current workspace.

Examples:
  m9m list
  m9m list --search "my-workflow"
  m9m list --output json
  m9m list --workspace my-project`,
	Run: runList,
}

func init() {
	listCmd.Flags().StringVar(&listSearch, "search", "", "Search workflows by name")
	listCmd.Flags().IntVar(&listLimit, "limit", 50, "Maximum number of workflows to list")
}

func runList(cmd *cobra.Command, args []string) {
	client := service.NewClient(nil)

	resp, err := client.ListWorkflows(workspaceFlag, listSearch, listLimit)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("\nMake sure a workspace is initialized. Run 'm9m init' first.")
		os.Exit(1)
	}

	if !resp.Success {
		fmt.Printf("Error: %s\n", resp.Error)
		os.Exit(1)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		fmt.Println("Error: Invalid response format")
		os.Exit(1)
	}

	workflows, _ := data["workflows"].([]interface{})
	total, _ := data["total"].(float64)

	if outputFlag == "json" {
		output, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(output))
		return
	}

	if len(workflows) == 0 {
		fmt.Println("No workflows found.")
		fmt.Println("\nTo create a workflow:")
		fmt.Println("  m9m create --name \"My Workflow\" --from workflow.json")
		return
	}

	fmt.Printf("Workflows (showing %d of %d):\n\n", len(workflows), int(total))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tNODES\tACTIVE\tUPDATED")
	fmt.Fprintln(w, "--\t----\t-----\t------\t-------")

	for _, wf := range workflows {
		wfMap, ok := wf.(map[string]interface{})
		if !ok {
			continue
		}

		id, _ := wfMap["id"].(string)
		name, _ := wfMap["name"].(string)
		nodeCount, _ := wfMap["nodeCount"].(float64)
		active, _ := wfMap["active"].(bool)
		updatedAt, _ := wfMap["updatedAt"].(string)

		activeStr := "no"
		if active {
			activeStr = "yes"
		}

		// Truncate ID for display
		shortID := id
		if len(shortID) > 12 {
			shortID = shortID[:12] + "..."
		}

		// Truncate updated date
		shortDate := updatedAt
		if len(shortDate) > 10 {
			shortDate = shortDate[:10]
		}

		fmt.Fprintf(w, "%s\t%s\t%.0f\t%s\t%s\n", shortID, name, nodeCount, activeStr, shortDate)
	}
	w.Flush()
}
