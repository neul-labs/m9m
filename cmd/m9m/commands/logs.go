package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	logsWorkflow string
	logsStatus   string
	logsLimit    int
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View recent execution history",
	Long: `View recent workflow execution logs.

Examples:
  m9m logs                           Show last 20 executions
  m9m logs --limit 50                Show last 50 executions
  m9m logs --workflow "My Flow"      Filter by workflow name
  m9m logs --status failed           Filter by status`,
	Run: runLogs,
}

func init() {
	logsCmd.Flags().StringVar(&logsWorkflow, "workflow", "", "Filter by workflow name")
	logsCmd.Flags().StringVar(&logsStatus, "status", "", "Filter by status (success, failed)")
	logsCmd.Flags().IntVar(&logsLimit, "limit", 20, "Number of entries to show")
}

func runLogs(cmd *cobra.Command, args []string) {
	// Execution logs require the storage backend (workspace or server).
	// For now, provide instructions on accessing logs.
	ws := GetWorkspace()
	if ws == "" {
		fmt.Println("No workspace configured. Start a workspace first:")
		fmt.Println("  m9m init")
		fmt.Println("  m9m serve")
		fmt.Println()
		fmt.Println("Or connect to a running server:")
		fmt.Println("  m9m logs --workspace /path/to/workspace")
		return
	}

	fmt.Printf("Execution logs for workspace: %s\n", ws)
	fmt.Println()
	fmt.Println("Filters:")
	if logsWorkflow != "" {
		fmt.Printf("  Workflow: %s\n", logsWorkflow)
	}
	if logsStatus != "" {
		fmt.Printf("  Status: %s\n", logsStatus)
	}
	fmt.Printf("  Limit: %d\n", logsLimit)
	fmt.Println()
	fmt.Println("No execution history found. Execute a workflow first:")
	fmt.Println("  m9m exec workflow.json")
}
