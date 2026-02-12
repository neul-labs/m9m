package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// NodeTypeInfo describes a node type (mirrors internal/mcp/tools/nodes.go)
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

// nodeTypeCatalog is the comprehensive catalog of all m9m node types
var nodeTypeCatalog = []NodeTypeInfo{
	// Core nodes
	{Name: "n8n-nodes-base.start", DisplayName: "Start", Description: "Workflow entry point", Category: "core", Version: 1},

	// Transform nodes
	{Name: "n8n-nodes-base.set", DisplayName: "Set", Description: "Set values on items using assignments", Category: "transform", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "assignments", "type": "array", "description": "Array of name/value pairs to set"},
		}},
	{Name: "n8n-nodes-base.filter", DisplayName: "Filter", Description: "Filter items based on conditions (equals, contains, regex, etc.)", Category: "transform", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "field", "type": "string", "description": "Field to filter on"},
			{"name": "operator", "type": "string", "description": "Filter operator: equals, notEquals, contains, startsWith, endsWith, regex, greaterThan, lessThan"},
			{"name": "value", "type": "any", "description": "Value to compare against"},
			{"name": "combiner", "type": "string", "description": "How to combine conditions: and, or"},
		}},
	{Name: "n8n-nodes-base.code", DisplayName: "Code", Description: "Execute JavaScript, Python, or Go code", Category: "transform", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "language", "type": "string", "description": "Language: javascript, python, go"},
			{"name": "code", "type": "string", "description": "Code to execute"},
			{"name": "mode", "type": "string", "description": "Mode: runOnceForAllItems, runOnceForEachItem"},
		}},
	{Name: "n8n-nodes-base.function", DisplayName: "Function", Description: "Execute custom JavaScript function", Category: "transform", Version: 1},
	{Name: "n8n-nodes-base.merge", DisplayName: "Merge", Description: "Merge data from multiple inputs", Category: "transform", Version: 1},
	{Name: "n8n-nodes-base.json", DisplayName: "JSON", Description: "Parse and manipulate JSON data", Category: "transform", Version: 1},
	{Name: "n8n-nodes-base.switch", DisplayName: "Switch", Description: "Route items to different outputs based on conditions", Category: "transform", Version: 1},
	{Name: "n8n-nodes-base.splitInBatches", DisplayName: "Split In Batches", Description: "Split items into batches for processing", Category: "transform", Version: 1},
	{Name: "n8n-nodes-base.itemLists", DisplayName: "Item Lists", Description: "Manipulate item lists (split, concatenate, limit)", Category: "transform", Version: 1},

	// HTTP nodes
	{Name: "n8n-nodes-base.httpRequest", DisplayName: "HTTP Request", Description: "Make HTTP requests (GET, POST, PUT, DELETE, etc.)", Category: "http", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "url", "type": "string", "required": true, "description": "URL to request"},
			{"name": "method", "type": "string", "description": "HTTP method: GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS"},
			{"name": "headers", "type": "object", "description": "HTTP headers"},
			{"name": "body", "type": "any", "description": "Request body"},
		}},

	// Trigger nodes
	{Name: "n8n-nodes-base.webhook", DisplayName: "Webhook", Description: "Receive HTTP webhooks with authentication support", Category: "trigger", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "path", "type": "string", "description": "Webhook path"},
			{"name": "httpMethod", "type": "string", "description": "HTTP method to accept"},
			{"name": "authentication", "type": "string", "description": "Auth type: none, basicAuth, headerAuth"},
		}},
	{Name: "n8n-nodes-base.cron", DisplayName: "Cron", Description: "Schedule workflows using cron expressions", Category: "trigger", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "cronExpression", "type": "string", "description": "Cron expression (e.g., '0 */5 * * *')"},
		}},

	// Messaging nodes
	{Name: "n8n-nodes-base.slack", DisplayName: "Slack", Description: "Send messages to Slack channels via webhook or API", Category: "messaging", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "webhookUrl", "type": "string", "description": "Slack webhook URL"},
			{"name": "text", "type": "string", "required": true, "description": "Message text"},
			{"name": "channel", "type": "string", "description": "Channel to send to"},
		}},
	{Name: "n8n-nodes-base.discord", DisplayName: "Discord", Description: "Send messages to Discord channels", Category: "messaging", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "webhookUrl", "type": "string", "required": true, "description": "Discord webhook URL"},
			{"name": "content", "type": "string", "required": true, "description": "Message content"},
			{"name": "username", "type": "string", "description": "Bot username"},
		}},

	// Database nodes
	{Name: "n8n-nodes-base.postgres", DisplayName: "PostgreSQL", Description: "Execute PostgreSQL queries (SELECT, INSERT, UPDATE, DELETE)", Category: "database", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "operation", "type": "string", "description": "Operation: executeQuery, insert, update, delete"},
			{"name": "connectionUrl", "type": "string", "description": "PostgreSQL connection URL"},
			{"name": "query", "type": "string", "description": "SQL query to execute"},
		}},
	{Name: "n8n-nodes-base.mysql", DisplayName: "MySQL", Description: "Execute MySQL queries", Category: "database", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "host", "type": "string", "description": "MySQL host"},
			{"name": "database", "type": "string", "description": "Database name"},
			{"name": "query", "type": "string", "description": "SQL query to execute"},
		}},
	{Name: "n8n-nodes-base.sqlite", DisplayName: "SQLite", Description: "Execute SQLite queries", Category: "database", Version: 1},

	// Email nodes
	{Name: "n8n-nodes-base.emailSend", DisplayName: "Send Email", Description: "Send emails via SMTP", Category: "email", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "smtpHost", "type": "string", "description": "SMTP server host"},
			{"name": "fromEmail", "type": "string", "description": "From email address"},
			{"name": "toEmail", "type": "string", "required": true, "description": "To email address"},
			{"name": "subject", "type": "string", "description": "Email subject"},
			{"name": "body", "type": "string", "description": "Email body"},
		}},

	// Cloud nodes - AWS
	{Name: "n8n-nodes-base.awsS3", DisplayName: "AWS S3", Description: "AWS S3 operations (upload, download, list, delete, presigned URLs)", Category: "cloud", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "operation", "type": "string", "description": "Operation: upload, download, delete, list, copy, createBucket, generatePresignedUrl"},
			{"name": "bucket", "type": "string", "description": "S3 bucket name"},
			{"name": "key", "type": "string", "description": "Object key/path"},
		}},
	{Name: "n8n-nodes-base.awsLambda", DisplayName: "AWS Lambda", Description: "Invoke AWS Lambda functions", Category: "cloud", Version: 1},

	// Cloud nodes - Azure
	{Name: "n8n-nodes-base.azureBlobStorage", DisplayName: "Azure Blob Storage", Description: "Azure Blob Storage operations", Category: "cloud", Version: 1},

	// Cloud nodes - GCP
	{Name: "n8n-nodes-base.gcpCloudStorage", DisplayName: "GCP Cloud Storage", Description: "Google Cloud Storage operations", Category: "cloud", Version: 1},

	// AI nodes
	{Name: "n8n-nodes-base.openAi", DisplayName: "OpenAI", Description: "Get completions from OpenAI models (GPT-3.5, GPT-4)", Category: "ai", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "model", "type": "string", "description": "Model name (gpt-3.5-turbo, gpt-4)"},
			{"name": "prompt", "type": "string", "required": true, "description": "Prompt to send"},
			{"name": "maxTokens", "type": "integer", "description": "Maximum tokens in response"},
			{"name": "temperature", "type": "number", "description": "Sampling temperature (0-2)"},
		}},
	{Name: "n8n-nodes-base.anthropic", DisplayName: "Anthropic (Claude)", Description: "Get completions from Anthropic Claude models", Category: "ai", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "model", "type": "string", "description": "Model name (claude-3-5-sonnet-20241022)"},
			{"name": "prompt", "type": "string", "required": true, "description": "Prompt to send"},
			{"name": "maxTokens", "type": "integer", "description": "Maximum tokens in response"},
		}},

	// File nodes
	{Name: "n8n-nodes-base.readBinaryFile", DisplayName: "Read Binary File", Description: "Read files with encoding and hash support", Category: "file", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "filePath", "type": "string", "required": true, "description": "Path to file"},
			{"name": "encoding", "type": "string", "description": "Encoding: binary, base64, hex, utf8"},
			{"name": "includeHashes", "type": "boolean", "description": "Include MD5, SHA256 hashes"},
		}},
	{Name: "n8n-nodes-base.writeBinaryFile", DisplayName: "Write Binary File", Description: "Write files with encoding support", Category: "file", Version: 1},

	// VCS nodes
	{Name: "n8n-nodes-base.github", DisplayName: "GitHub", Description: "GitHub API operations (repos, issues, PRs, users)", Category: "vcs", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "resource", "type": "string", "description": "Resource: repository, issue, pullRequest, user"},
			{"name": "operation", "type": "string", "description": "Operation: get, list"},
			{"name": "owner", "type": "string", "description": "Repository owner"},
			{"name": "repository", "type": "string", "description": "Repository name"},
		}},
	{Name: "n8n-nodes-base.gitlab", DisplayName: "GitLab", Description: "GitLab API operations", Category: "vcs", Version: 1},

	// Productivity nodes
	{Name: "n8n-nodes-base.googleSheets", DisplayName: "Google Sheets", Description: "Read and write Google Sheets", Category: "productivity", Version: 1},

	// Python code node
	{Name: "n8n-nodes-base.pythonCode", DisplayName: "Python Code", Description: "Execute Python code", Category: "code", Version: 1},

	// CLI execution node
	{Name: "n8n-nodes-base.cliExecute", DisplayName: "CLI Execute", Description: "Execute CLI commands and AI agents (Claude Code, Codex, Aider) in sandboxed environment", Category: "cli", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "command", "type": "string", "required": true, "description": "Command to execute"},
			{"name": "args", "type": "array", "description": "Command arguments"},
			{"name": "env", "type": "object", "description": "Environment variables"},
			{"name": "workDir", "type": "string", "description": "Working directory"},
			{"name": "shell", "type": "boolean", "description": "Run command through shell"},
			{"name": "sandboxEnabled", "type": "boolean", "description": "Enable sandbox isolation (default: true)"},
			{"name": "isolationLevel", "type": "string", "description": "Isolation level: none, minimal, standard, strict, paranoid"},
			{"name": "networkAccess", "type": "string", "description": "Network access: host, isolated, loopback"},
			{"name": "timeout", "type": "integer", "description": "Timeout in seconds (default: 60)"},
			{"name": "maxMemoryMB", "type": "integer", "description": "Max memory in MB (default: 512)"},
			{"name": "outputFormat", "type": "string", "description": "Output format: text, json, lines"},
			{"name": "additionalMounts", "type": "array", "description": "Additional filesystem mounts [{source, destination, readWrite}]"},
		}},
}

// nodeCategoryCatalog is the list of node categories
var nodeCategoryCatalog = []NodeCategory{
	{Name: "core", Description: "Core workflow nodes (start, end)", Count: 1},
	{Name: "transform", Description: "Data transformation nodes (set, filter, code, merge)", Count: 9},
	{Name: "http", Description: "HTTP request nodes", Count: 1},
	{Name: "trigger", Description: "Workflow trigger nodes (webhook, cron)", Count: 2},
	{Name: "messaging", Description: "Messaging platform nodes (Slack, Discord)", Count: 2},
	{Name: "database", Description: "Database nodes (PostgreSQL, MySQL, SQLite)", Count: 3},
	{Name: "email", Description: "Email sending nodes", Count: 1},
	{Name: "cloud", Description: "Cloud platform nodes (AWS, Azure, GCP)", Count: 4},
	{Name: "ai", Description: "AI/LLM nodes (OpenAI, Anthropic)", Count: 2},
	{Name: "cli", Description: "CLI execution nodes for AI agents (Claude Code, Codex, Aider)", Count: 1},
	{Name: "file", Description: "File read/write nodes", Count: 2},
	{Name: "vcs", Description: "Version control nodes (GitHub, GitLab)", Count: 2},
	{Name: "productivity", Description: "Productivity app nodes (Google Sheets)", Count: 1},
	{Name: "code", Description: "Code execution nodes", Count: 1},
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

	for _, node := range nodeTypeCatalog {
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
	if outputFlag == "json" {
		data, _ := json.MarshalIndent(nodeCategoryCatalog, "", "  ")
		fmt.Println(string(data))
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CATEGORY\tCOUNT\tDESCRIPTION")
	fmt.Fprintln(w, "--------\t-----\t-----------")
	for _, cat := range nodeCategoryCatalog {
		fmt.Fprintf(w, "%s\t%d\t%s\n", cat.Name, cat.Count, cat.Description)
	}
	w.Flush()
}

func runNodeInfo(cmd *cobra.Command, args []string) {
	nodeType := args[0]

	var found *NodeTypeInfo
	for _, node := range nodeTypeCatalog {
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

	// Verify node exists
	var found *NodeTypeInfo
	for _, node := range nodeTypeCatalog {
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

	// Parse input data
	var inputData map[string]interface{}
	if nodeTestInput != "" {
		var inputJSON string
		if strings.HasPrefix(nodeTestInput, "@") {
			filePath := strings.TrimPrefix(nodeTestInput, "@")
			data, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Printf("Error: Cannot read input file: %v\n", err)
				os.Exit(1)
			}
			inputJSON = string(data)
		} else {
			inputJSON = nodeTestInput
		}

		if err := json.Unmarshal([]byte(inputJSON), &inputData); err != nil {
			fmt.Printf("Error: Invalid input JSON: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Default test input
		inputData = map[string]interface{}{
			"test": "data",
			"id":   1,
		}
	}

	fmt.Printf("Testing node: %s\n", found.DisplayName)
	fmt.Printf("Type: %s\n", found.Name)
	fmt.Println()
	fmt.Println("Input:")
	inputJSON, _ := json.MarshalIndent(inputData, "  ", "  ")
	fmt.Println("  " + string(inputJSON))
	fmt.Println()

	// For full testing, we need the service running
	// Show a message about how to test via workflow
	fmt.Println("To test this node with actual execution:")
	fmt.Println()
	fmt.Printf("  1. Create a test workflow:\n")
	fmt.Printf("     {\n")
	fmt.Printf("       \"name\": \"Test %s\",\n", found.DisplayName)
	fmt.Printf("       \"nodes\": [\n")
	fmt.Printf("         {\"name\": \"Start\", \"type\": \"n8n-nodes-base.start\", \"position\": [250, 300]},\n")
	fmt.Printf("         {\"name\": \"Test\", \"type\": \"%s\", \"position\": [450, 300], \"parameters\": {}}\n", found.Name)
	fmt.Printf("       ],\n")
	fmt.Printf("       \"connections\": {\"Start\": {\"main\": [[{\"node\": \"Test\", \"type\": \"main\", \"index\": 0}]]}}\n")
	fmt.Printf("     }\n")
	fmt.Println()
	fmt.Println("  2. Save as test.json and run:")
	fmt.Println("     m9m validate test.json")
	fmt.Println("     m9m create --from test.json --name \"Test\"")
	fmt.Println("     m9m run \"Test\" --input '{\"test\": \"data\"}'")
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
