package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/neul-labs/m9m/internal/service"
)

var (
	runInput    string
	runInputRaw bool
)

var runCmd = &cobra.Command{
	Use:   "run <workflow>",
	Short: "Execute a workflow",
	Long: `Execute a workflow by name or ID.

The workflow is executed with optional input data.

Examples:
  m9m run hello-world
  m9m run my-workflow --input '{"key": "value"}'
  m9m run my-workflow --input @data.json
  m9m run workflow-id --output json`,
	Args: cobra.ExactArgs(1),
	Run:  runRun,
}

func init() {
	runCmd.Flags().StringVar(&runInput, "input", "", "Input data as JSON or @file.json")
	runCmd.Flags().BoolVar(&runInputRaw, "raw", false, "Output raw result data only")
}

func runRun(cmd *cobra.Command, args []string) {
	workflowName := args[0]

	// Parse input data
	var inputData map[string]interface{}
	if runInput != "" {
		var inputJSON string

		if strings.HasPrefix(runInput, "@") {
			// Read from file
			filePath := strings.TrimPrefix(runInput, "@")
			data, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Printf("Error: Cannot read input file: %v\n", err)
				os.Exit(1)
			}
			inputJSON = string(data)
		} else {
			inputJSON = runInput
		}

		if err := json.Unmarshal([]byte(inputJSON), &inputData); err != nil {
			fmt.Printf("Error: Invalid input JSON: %v\n", err)
			os.Exit(1)
		}
	}

	client := service.NewClient(nil)

	if verboseFlag {
		fmt.Printf("Executing workflow: %s\n", workflowName)
		if inputData != nil {
			fmt.Printf("Input: %v\n", inputData)
		}
		fmt.Println()
	}

	resp, err := client.RunWorkflow(workspaceFlag, workflowName, inputData)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("\nMake sure a workspace is initialized and the workflow exists.")
		os.Exit(1)
	}

	if !resp.Success {
		fmt.Printf("Error: Workflow execution failed\n")
		fmt.Printf("Details: %s\n", resp.Error)
		os.Exit(1)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		fmt.Println("Error: Invalid response format")
		os.Exit(1)
	}

	if outputFlag == "json" {
		output, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(output))
		return
	}

	if runInputRaw {
		// Output just the result data
		if resultData, ok := data["data"]; ok {
			output, _ := json.MarshalIndent(resultData, "", "  ")
			fmt.Println(string(output))
		}
		return
	}

	// Human-readable output
	executionID, _ := data["executionId"].(string)
	status, _ := data["status"].(string)
	duration, _ := data["duration"].(string)

	fmt.Println("Workflow executed successfully!")
	fmt.Println()
	fmt.Printf("Execution ID: %s\n", executionID)
	fmt.Printf("Status: %s\n", status)
	fmt.Printf("Duration: %s\n", duration)

	if resultData, ok := data["data"]; ok {
		fmt.Println("\nResult:")
		output, _ := json.MarshalIndent(resultData, "", "  ")
		fmt.Println(string(output))
	}
}
