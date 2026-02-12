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
	execWorkflowID string
	execLimit      int
)

var executionCmd = &cobra.Command{
	Use:     "execution",
	Aliases: []string{"exec", "executions"},
	Short:   "Manage workflow executions",
	Long: `View and manage workflow executions.

Executions are recorded each time a workflow runs, including:
- Status (success, error, running)
- Input/output data
- Duration and timing
- Error messages if failed`,
}

var executionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workflow executions",
	Long: `List workflow executions with optional filtering.

Examples:
  m9m execution list
  m9m execution list --workflow my-workflow
  m9m execution list --limit 10
  m9m execution list --output json`,
	Run: runExecutionList,
}

var executionGetCmd = &cobra.Command{
	Use:   "get <execution-id>",
	Short: "Get execution details",
	Long: `Get detailed information about a specific execution.

Examples:
  m9m execution get exec_123456
  m9m execution get exec_123456 --output json`,
	Args: cobra.ExactArgs(1),
	Run:  runExecutionGet,
}

var executionWatchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch executions in real-time",
	Long: `Watch workflow executions as they happen.

Examples:
  m9m execution watch
  m9m execution watch --workflow my-workflow`,
	Run: runExecutionWatch,
}

func init() {
	executionListCmd.Flags().StringVar(&execWorkflowID, "workflow", "", "Filter by workflow name or ID")
	executionListCmd.Flags().IntVar(&execLimit, "limit", 20, "Maximum number of executions to show")

	executionWatchCmd.Flags().StringVar(&execWorkflowID, "workflow", "", "Filter by workflow name or ID")

	executionCmd.AddCommand(executionListCmd)
	executionCmd.AddCommand(executionGetCmd)
	executionCmd.AddCommand(executionWatchCmd)
}

func runExecutionList(cmd *cobra.Command, args []string) {
	client := service.NewClient(nil)

	resp, err := client.ListExecutions(workspaceFlag, execWorkflowID, execLimit)
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

	executions, _ := data["executions"].([]interface{})
	total, _ := data["total"].(float64)

	if outputFlag == "json" {
		output, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(output))
		return
	}

	if len(executions) == 0 {
		fmt.Println("No executions found.")
		fmt.Println("\nRun a workflow with: m9m run <workflow>")
		return
	}

	fmt.Printf("Executions (showing %d of %d):\n\n", len(executions), int(total))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tWORKFLOW\tSTATUS\tMODE\tSTARTED\tDURATION")
	fmt.Fprintln(w, "--\t--------\t------\t----\t-------\t--------")

	for _, exec := range executions {
		execMap, ok := exec.(map[string]interface{})
		if !ok {
			continue
		}

		id, _ := execMap["id"].(string)
		workflowID, _ := execMap["workflowId"].(string)
		status, _ := execMap["status"].(string)
		mode, _ := execMap["mode"].(string)
		startedAt, _ := execMap["startedAt"].(string)

		// Truncate ID
		shortID := id
		if len(shortID) > 15 {
			shortID = shortID[:15] + "..."
		}

		// Truncate workflow ID
		shortWF := workflowID
		if len(shortWF) > 15 {
			shortWF = shortWF[:15] + "..."
		}

		// Format date
		shortDate := startedAt
		if len(shortDate) > 19 {
			shortDate = shortDate[:19]
		}

		// Calculate duration if finished
		duration := "-"
		if finishedAt, ok := execMap["finishedAt"].(string); ok && finishedAt != "" {
			duration = "completed"
		}

		// Color status
		statusDisplay := status
		switch status {
		case "success":
			statusDisplay = "✓ success"
		case "error":
			statusDisplay = "✗ error"
		case "running":
			statusDisplay = "● running"
		case "cancelled":
			statusDisplay = "○ cancelled"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			shortID, shortWF, statusDisplay, mode, shortDate, duration)
	}
	w.Flush()
}

func runExecutionGet(cmd *cobra.Command, args []string) {
	executionID := args[0]

	client := service.NewClient(nil)

	resp, err := client.GetExecution(workspaceFlag, executionID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if !resp.Success {
		fmt.Printf("Error: %s\n", resp.Error)
		os.Exit(1)
	}

	execution, ok := resp.Data.(map[string]interface{})
	if !ok {
		fmt.Println("Error: Invalid response format")
		os.Exit(1)
	}

	if outputFlag == "json" {
		output, _ := json.MarshalIndent(execution, "", "  ")
		fmt.Println(string(output))
		return
	}

	// Human-readable output
	id, _ := execution["id"].(string)
	workflowID, _ := execution["workflowId"].(string)
	status, _ := execution["status"].(string)
	mode, _ := execution["mode"].(string)
	startedAt, _ := execution["startedAt"].(string)
	finishedAt, _ := execution["finishedAt"].(string)

	fmt.Printf("Execution: %s\n", id)
	fmt.Printf("============%s\n", repeatChar('=', len(id)))
	fmt.Printf("Workflow ID: %s\n", workflowID)
	fmt.Printf("Status: %s\n", status)
	fmt.Printf("Mode: %s\n", mode)
	fmt.Printf("Started: %s\n", formatDate(startedAt))
	if finishedAt != "" {
		fmt.Printf("Finished: %s\n", formatDate(finishedAt))
	}

	// Show error if present
	if errMsg, ok := execution["error"].(string); ok && errMsg != "" {
		fmt.Printf("\nError:\n  %s\n", errMsg)
	}

	// Show data if present
	if data, ok := execution["data"]; ok && data != nil {
		fmt.Println("\nOutput Data:")
		output, _ := json.MarshalIndent(data, "  ", "  ")
		fmt.Println("  " + string(output))
	}
}

func runExecutionWatch(cmd *cobra.Command, args []string) {
	fmt.Println("Watching executions... (Press Ctrl+C to stop)")
	fmt.Println()

	// For now, just poll and show new executions
	// In the future, this could use WebSocket for real-time updates
	client := service.NewClient(nil)

	lastSeen := ""
	for {
		resp, err := client.ListExecutions(workspaceFlag, execWorkflowID, 5)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		if !resp.Success {
			continue
		}

		data, ok := resp.Data.(map[string]interface{})
		if !ok {
			continue
		}

		executions, _ := data["executions"].([]interface{})
		for _, exec := range executions {
			execMap, ok := exec.(map[string]interface{})
			if !ok {
				continue
			}

			id, _ := execMap["id"].(string)
			if id == lastSeen {
				break
			}

			if lastSeen == "" {
				lastSeen = id
				break
			}

			status, _ := execMap["status"].(string)
			workflowID, _ := execMap["workflowId"].(string)
			startedAt, _ := execMap["startedAt"].(string)

			fmt.Printf("[%s] %s - %s (%s)\n", formatDate(startedAt), workflowID, status, id)
			lastSeen = id
		}

		// Sleep before polling again
		// time.Sleep(2 * time.Second)
		break // For now, just show once - real watch would loop
	}

	fmt.Println("\nNote: Real-time watching requires the server WebSocket. Use 'm9m serve' for live updates.")
}
