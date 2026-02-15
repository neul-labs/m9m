package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/neul-labs/m9m/internal/engine"
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/ai"
	"github.com/neul-labs/m9m/internal/nodes/cli"
	"github.com/neul-labs/m9m/internal/nodes/code"
	"github.com/neul-labs/m9m/internal/nodes/database"
	"github.com/neul-labs/m9m/internal/nodes/file"
	httpnode "github.com/neul-labs/m9m/internal/nodes/http"
	"github.com/neul-labs/m9m/internal/nodes/messaging"
	"github.com/neul-labs/m9m/internal/nodes/timer"
	"github.com/neul-labs/m9m/internal/nodes/transform"
	"github.com/neul-labs/m9m/internal/nodes/trigger"
)

var (
	execInput  string
	execStdin  bool
	execQuiet  bool
	execPretty bool
)

var execCmd = &cobra.Command{
	Use:   "exec <workflow.json>",
	Short: "Execute a workflow file directly (agent-friendly)",
	Long: `Execute a workflow file directly without requiring a daemon or server.

This command is designed for agent and script usage:
- Direct execution: No daemon or server required
- JSON output: Easy to parse programmatically
- Stdin support: Pipe data directly into workflows
- Zero config: Works immediately with no setup

Examples:
  # Execute a workflow file
  m9m exec workflow.json

  # Execute with input data
  m9m exec workflow.json --input '{"name": "test"}'

  # Execute with input from file
  m9m exec workflow.json --input @data.json

  # Pipe data from stdin
  echo '{"message": "hello"}' | m9m exec workflow.json --stdin

  # Quiet mode (only output result data)
  m9m exec workflow.json --quiet

  # Pretty-print JSON output
  m9m exec workflow.json --pretty`,
	Args: cobra.ExactArgs(1),
	Run:  runExec,
}

func init() {
	execCmd.Flags().StringVarP(&execInput, "input", "i", "", "Input data as JSON or @file.json")
	execCmd.Flags().BoolVar(&execStdin, "stdin", false, "Read input data from stdin")
	execCmd.Flags().BoolVarP(&execQuiet, "quiet", "q", false, "Output only the result data (no metadata)")
	execCmd.Flags().BoolVarP(&execPretty, "pretty", "p", false, "Pretty-print JSON output")
}

// ExecResult is the structured output for exec command
type ExecResult struct {
	Success   bool                     `json:"success"`
	Data      []map[string]interface{} `json:"data,omitempty"`
	Error     string                   `json:"error,omitempty"`
	Duration  string                   `json:"duration,omitempty"`
	NodeCount int                      `json:"nodeCount,omitempty"`
}

func runExec(cmd *cobra.Command, args []string) {
	workflowFile := args[0]
	startTime := time.Now()

	// Read workflow file
	workflowData, err := os.ReadFile(workflowFile)
	if err != nil {
		outputExecError(fmt.Sprintf("Cannot read workflow file: %v", err))
		os.Exit(1)
	}

	// Parse workflow
	var workflow model.Workflow
	if err := json.Unmarshal(workflowData, &workflow); err != nil {
		outputExecError(fmt.Sprintf("Invalid workflow JSON: %v", err))
		os.Exit(1)
	}

	// Parse input data
	inputData, err := parseExecInput()
	if err != nil {
		outputExecError(fmt.Sprintf("Invalid input: %v", err))
		os.Exit(1)
	}

	// Create engine and register nodes
	eng := engine.NewWorkflowEngine()
	registerExecNodes(eng)

	// Convert input to DataItems
	var dataItems []model.DataItem
	if inputData != nil {
		dataItems = []model.DataItem{{JSON: inputData}}
	} else {
		dataItems = []model.DataItem{{JSON: make(map[string]interface{})}}
	}

	// Execute workflow
	result, err := eng.ExecuteWorkflow(&workflow, dataItems)
	if err != nil {
		outputExecError(fmt.Sprintf("Execution failed: %v", err))
		os.Exit(1)
	}

	// Check for execution errors in result
	if result.Error != nil {
		outputExecError(fmt.Sprintf("Workflow error: %v", result.Error))
		os.Exit(1)
	}

	// Build output
	duration := time.Since(startTime)
	outputExecResult(result, &workflow, duration)
}

func parseExecInput() (map[string]interface{}, error) {
	var inputJSON string

	if execStdin {
		// Read from stdin
		reader := bufio.NewReader(os.Stdin)
		var builder strings.Builder
		for {
			line, err := reader.ReadString('\n')
			builder.WriteString(line)
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("error reading stdin: %w", err)
			}
		}
		inputJSON = strings.TrimSpace(builder.String())
	} else if execInput != "" {
		if strings.HasPrefix(execInput, "@") {
			// Read from file
			filePath := strings.TrimPrefix(execInput, "@")
			data, err := os.ReadFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("cannot read input file: %w", err)
			}
			inputJSON = string(data)
		} else {
			inputJSON = execInput
		}
	}

	if inputJSON == "" {
		return nil, nil
	}

	var inputData map[string]interface{}
	if err := json.Unmarshal([]byte(inputJSON), &inputData); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	return inputData, nil
}

func outputExecError(message string) {
	result := ExecResult{
		Success: false,
		Error:   message,
	}
	outputJSON(result)
}

func outputExecResult(result *engine.ExecutionResult, workflow *model.Workflow, duration time.Duration) {
	// Convert DataItems to maps
	var data []map[string]interface{}
	for _, item := range result.Data {
		data = append(data, item.JSON)
	}

	if execQuiet {
		// Quiet mode: only output the data array
		outputJSON(data)
	} else {
		// Full result with metadata
		execResult := ExecResult{
			Success:   true,
			Data:      data,
			Duration:  duration.String(),
			NodeCount: len(workflow.Nodes),
		}
		outputJSON(execResult)
	}
}

func outputJSON(v interface{}) {
	var output []byte
	var err error

	if execPretty {
		output, err = json.MarshalIndent(v, "", "  ")
	} else {
		output, err = json.Marshal(v)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, `{"success":false,"error":"JSON encoding error: %v"}`, err)
		os.Exit(1)
	}

	fmt.Println(string(output))
}

// registerExecNodes registers all available node types for direct execution
func registerExecNodes(eng engine.WorkflowEngine) {
	// Transform nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.set", transform.NewSetNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.filter", transform.NewFilterNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.code", transform.NewCodeNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.merge", transform.NewMergeNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.splitInBatches", transform.NewSplitInBatchesNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.itemLists", transform.NewItemListsNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.switch", transform.NewSwitchNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.function", transform.NewFunctionNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.json", transform.NewJSONNode())

	// HTTP nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.httpRequest", httpnode.NewHTTPRequestNode())

	// Trigger nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.webhook", trigger.NewWebhookNode())

	// Timer/Schedule nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.cron", timer.NewCronNode())

	// Code execution nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.executeCommand", cli.NewExecuteNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.pythonCode", code.NewPythonCodeNode())

	// File nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.readBinaryFile", file.NewReadBinaryFileNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.writeBinaryFile", file.NewWriteBinaryFileNode())

	// Database nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.postgres", database.NewPostgresNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.mySql", database.NewMySQLNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.sqlite", database.NewSQLiteNode())

	// Messaging nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.slack", messaging.NewSlackNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.discord", messaging.NewDiscordNode())

	// AI nodes
	eng.RegisterNodeExecutor("@n8n/n8n-nodes-langchain.openAi", ai.NewOpenAINode())
	eng.RegisterNodeExecutor("@n8n/n8n-nodes-langchain.anthropic", ai.NewAnthropicNode())
}
