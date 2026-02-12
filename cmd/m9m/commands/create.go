package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/neul-labs/m9m/internal/service"
)

var (
	createName        string
	createFrom        string
	createDescription string
	createSkipValidate bool
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new workflow",
	Long: `Create a new workflow from a JSON file.

The workflow JSON should include nodes and connections.
By default, the workflow is validated before creation.

Examples:
  m9m create --name "My Workflow" --from workflow.json
  m9m create --from workflow.json  # Uses name from JSON
  m9m create --name "Test" --from workflow.json --skip-validate`,
	Run: runCreate,
}

func init() {
	createCmd.Flags().StringVar(&createName, "name", "", "Workflow name (overrides name in JSON)")
	createCmd.Flags().StringVar(&createFrom, "from", "", "JSON file containing workflow definition (required)")
	createCmd.Flags().StringVar(&createDescription, "description", "", "Workflow description")
	createCmd.Flags().BoolVar(&createSkipValidate, "skip-validate", false, "Skip validation before creating")
	createCmd.MarkFlagRequired("from")
}

func runCreate(cmd *cobra.Command, args []string) {
	// Read workflow file
	data, err := os.ReadFile(createFrom)
	if err != nil {
		fmt.Printf("Error: Cannot read file: %v\n", err)
		os.Exit(1)
	}

	// Parse JSON
	var workflow map[string]interface{}
	if err := json.Unmarshal(data, &workflow); err != nil {
		fmt.Printf("Error: Invalid JSON: %v\n", err)
		os.Exit(1)
	}

	// Override name if provided
	if createName != "" {
		workflow["name"] = createName
	}

	// Add description if provided
	if createDescription != "" {
		workflow["description"] = createDescription
	}

	// Validate name
	name, ok := workflow["name"].(string)
	if !ok || name == "" {
		fmt.Println("Error: Workflow must have a name")
		fmt.Println("  Use --name flag or include 'name' in the JSON")
		os.Exit(1)
	}

	// Validate workflow structure (unless skipped)
	if !createSkipValidate {
		result := validateWorkflow(workflow)
		if !result.Valid {
			fmt.Printf("Validation failed for %s\n", createFrom)
			fmt.Println("\nErrors:")
			for _, e := range result.Errors {
				fmt.Printf("  - [%s] %s\n", e.Field, e.Message)
			}
			fmt.Println("\nUse --skip-validate to bypass validation")
			os.Exit(1)
		}

		if verboseFlag && len(result.Warnings) > 0 {
			fmt.Println("Warnings:")
			for _, w := range result.Warnings {
				fmt.Printf("  - [%s] %s\n", w.Field, w.Message)
			}
			fmt.Println()
		}
	}

	// Create workflow via service
	client := service.NewClient(nil)

	resp, err := client.CreateWorkflow(workspaceFlag, workflow)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("\nMake sure a workspace is initialized. Run 'm9m init' first.")
		os.Exit(1)
	}

	if !resp.Success {
		fmt.Printf("Error: %s\n", resp.Error)
		os.Exit(1)
	}

	data2, ok := resp.Data.(map[string]interface{})
	if !ok {
		fmt.Println("Error: Invalid response format")
		os.Exit(1)
	}

	if outputFlag == "json" {
		output, _ := json.MarshalIndent(data2, "", "  ")
		fmt.Println(string(output))
		return
	}

	id, _ := data2["id"].(string)
	fmt.Printf("Created workflow '%s'\n", name)
	fmt.Printf("  ID: %s\n", id)
	fmt.Println("\nTo run this workflow:")
	fmt.Printf("  m9m run \"%s\"\n", name)
}
