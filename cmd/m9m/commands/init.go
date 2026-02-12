package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/neul-labs/m9m/internal/workspace"
)

var (
	initNoInject bool
	initStorage  string
)

var initCmd = &cobra.Command{
	Use:   "init [workspace]",
	Short: "Initialize a workspace",
	Long: `Initialize a new m9m workspace and optionally inject agent instructions into CLAUDE.md.

If no workspace name is provided, uses the current directory name.

Examples:
  m9m init                    Initialize with directory name
  m9m init my-project         Initialize workspace "my-project"
  m9m init my-project --no-inject  Skip CLAUDE.md injection`,
	Args: cobra.MaximumNArgs(1),
	Run:  runInit,
}

func init() {
	initCmd.Flags().BoolVar(&initNoInject, "no-inject", false, "Skip CLAUDE.md injection")
	initCmd.Flags().StringVar(&initStorage, "storage", "sqlite", "Storage type: sqlite, postgres")
}

func runInit(cmd *cobra.Command, args []string) {
	// Determine workspace name
	var name string
	if len(args) > 0 {
		name = args[0]
	} else {
		// Use current directory name
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		name = filepath.Base(cwd)
	}

	// Validate name
	if name == "" || name == "." || name == ".." {
		fmt.Println("Error: Invalid workspace name")
		os.Exit(1)
	}

	mgr, err := workspace.NewManager()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Check if workspace already exists
	if mgr.WorkspaceExists(name) {
		// Just switch to it
		if err := mgr.SetCurrent(name); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Workspace '%s' already exists. Switched to it.\n", name)

		// Still inject CLAUDE.md if not disabled
		if !initNoInject {
			injectClaudeMd(name)
		}
		return
	}

	// Create workspace
	config := &workspace.WorkspaceConfig{
		StorageType:   initStorage,
		IdleTimeout:   300, // 5 minutes
		MaxExecutions: 1000,
		DefaultTags:   []string{},
	}

	ws, err := mgr.Create(name, config)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Set as current workspace
	if err := mgr.SetCurrent(name); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created workspace '%s'\n", ws.Name)
	fmt.Printf("  Path: %s\n", ws.Path)
	fmt.Printf("  Storage: %s\n", ws.Config.StorageType)

	// Inject CLAUDE.md
	if !initNoInject {
		injectClaudeMd(name)
	}

	fmt.Println("\nWorkspace ready! Try these commands:")
	fmt.Println("  m9m node list       - List available node types")
	fmt.Println("  m9m status          - Check workspace status")
}

// injectClaudeMd injects m9m instructions into CLAUDE.md
func injectClaudeMd(workspaceName string) {
	claudeMdPath := "CLAUDE.md"

	// Read existing CLAUDE.md if it exists
	var existingContent string
	if data, err := os.ReadFile(claudeMdPath); err == nil {
		existingContent = string(data)
	}

	// Check if m9m section already exists
	if strings.Contains(existingContent, "<!-- m9m:begin -->") {
		// Update existing section
		start := strings.Index(existingContent, "<!-- m9m:begin -->")
		end := strings.Index(existingContent, "<!-- m9m:end -->")
		if start != -1 && end != -1 {
			end += len("<!-- m9m:end -->")
			newContent := existingContent[:start] + getClaudeMdContent(workspaceName) + existingContent[end:]
			if err := os.WriteFile(claudeMdPath, []byte(newContent), 0644); err != nil {
				fmt.Printf("Warning: Failed to update CLAUDE.md: %v\n", err)
				return
			}
			fmt.Println("Updated CLAUDE.md with m9m instructions")
			return
		}
	}

	// Append to CLAUDE.md
	var newContent string
	if existingContent != "" {
		newContent = existingContent + "\n\n" + getClaudeMdContent(workspaceName)
	} else {
		newContent = getClaudeMdContent(workspaceName)
	}

	if err := os.WriteFile(claudeMdPath, []byte(newContent), 0644); err != nil {
		fmt.Printf("Warning: Failed to create CLAUDE.md: %v\n", err)
		return
	}

	fmt.Println("Added m9m instructions to CLAUDE.md")
}

func getClaudeMdContent(workspaceName string) string {
	return fmt.Sprintf(`<!-- m9m:begin -->
## m9m Workflow Engine

This project uses m9m for workflow automation via CLI. Current workspace: **%s**

### Quick Reference

| Command | Description |
|---------|-------------|
| `+"`m9m list`"+` | List all workflows |
| `+"`m9m run <name>`"+` | Execute a workflow |
| `+"`m9m get <name>`"+` | Get workflow details |
| `+"`m9m create --from file.json`"+` | Create workflow from JSON |
| `+"`m9m validate file.json`"+` | Validate workflow JSON |
| `+"`m9m execution list`"+` | List workflow executions |
| `+"`m9m execution get <id>`"+` | Get execution details/status |
| `+"`m9m node list`"+` | List available node types |
| `+"`m9m node info <type>`"+` | Get node parameters |
| `+"`m9m node test <type>`"+` | Test a node |
| `+"`m9m status`"+` | Check workspace/service status |

### Workflow Management

`+"```bash"+`
# Create and run a workflow
m9m validate workflow.json                    # Always validate first
m9m create --name "My Workflow" --from workflow.json
m9m run "My Workflow"
m9m run "My Workflow" --input '{"key": "value"}'
m9m run "My Workflow" --input @data.json      # Input from file

# View workflow details
m9m list                                      # List all workflows
m9m get "My Workflow"                         # Get workflow details
m9m get "My Workflow" --output json           # JSON output
`+"```"+`

### Execution Monitoring

`+"```bash"+`
# View executions and their state
m9m execution list                            # List all executions
m9m execution list --workflow "My Workflow"   # Filter by workflow
m9m execution list --limit 10                 # Limit results
m9m execution get exec_123456                 # Get execution details
m9m execution get exec_123456 --output json   # Full JSON with data

# Execution statuses: success, error, running, cancelled
`+"```"+`

### Node Discovery

`+"```bash"+`
# List and search nodes
m9m node list                          # List all 30+ node types
m9m node list --category ai            # Filter: ai, database, transform, http, etc.
m9m node list --search "http"          # Search by name/description
m9m node categories                    # List all categories

# Get node details
m9m node info n8n-nodes-base.httpRequest
m9m node info n8n-nodes-base.openAi

# Test a node
m9m node test n8n-nodes-base.set --input '{"name": "test"}'
`+"```"+`

### Creating Custom Nodes

`+"```bash"+`
m9m node create --from mynode.js
`+"```"+`

`+"```javascript"+`
// mynode.js - Custom JavaScript node
module.exports = {
  name: "myCustomNode",
  description: "My custom processing node",
  category: "transform",
  execute: function(input, params) {
    return input.map(item => ({
      ...item,
      processed: true,
      timestamp: new Date().toISOString()
    }));
  }
};
`+"```"+`

**Scripting Limitations (Goja runtime):**
- ES5.1+ only - NO async/await
- NO real HTTP - use httpRequest node instead
- Mocked npm: lodash (partial), moment (partial), uuid
- 30s timeout per execution

### Workflow JSON Structure

`+"```json"+`
{
  "name": "Example Workflow",
  "nodes": [
    {"name": "Start", "type": "n8n-nodes-base.start", "parameters": {}, "position": [250, 300]},
    {"name": "HTTP", "type": "n8n-nodes-base.httpRequest", "parameters": {"url": "https://api.example.com", "method": "GET"}, "position": [450, 300]},
    {"name": "Transform", "type": "n8n-nodes-base.set", "parameters": {"assignments": [{"name": "result", "value": "={{$json.data}}"}]}, "position": [650, 300]}
  ],
  "connections": {
    "Start": {"main": [[{"node": "HTTP", "type": "main", "index": 0}]]},
    "HTTP": {"main": [[{"node": "Transform", "type": "main", "index": 0}]]}
  }
}
`+"```"+`

### Workspace Management

`+"```bash"+`
# Workspace commands
m9m workspace list                     # List all workspaces
m9m workspace current                  # Show current workspace
m9m workspace use other-project        # Switch workspace
m9m workspace create new-project       # Create new workspace

# Override workspace per-command
m9m list --workspace other-project
m9m run "Workflow" -w other-project

# Or use environment variable
export M9M_WORKSPACE=other-project
`+"```"+`

### Common Node Types

| Node | Description |
|------|-------------|
| `+"`n8n-nodes-base.httpRequest`"+` | HTTP requests (GET, POST, PUT, DELETE) |
| `+"`n8n-nodes-base.code`"+` | Execute JavaScript/Python code |
| `+"`n8n-nodes-base.set`"+` | Set/transform data fields |
| `+"`n8n-nodes-base.filter`"+` | Filter items by conditions |
| `+"`n8n-nodes-base.merge`"+` | Merge data from multiple inputs |
| `+"`n8n-nodes-base.switch`"+` | Route items by conditions |
| `+"`n8n-nodes-base.slack`"+` | Send Slack messages |
| `+"`n8n-nodes-base.openAi`"+` | OpenAI API (GPT, embeddings) |
| `+"`n8n-nodes-base.anthropic`"+` | Anthropic Claude API |
| `+"`n8n-nodes-base.postgres`"+` | PostgreSQL queries |
| `+"`n8n-nodes-base.webhook`"+` | Receive HTTP webhooks |
| `+"`n8n-nodes-base.cron`"+` | Schedule workflows |

Run `+"`m9m node list`"+` for all 30+ available nodes.

### Service Management

`+"```bash"+`
m9m status                             # Check service status
m9m service start                      # Start background daemon
m9m service stop                       # Stop daemon
m9m service status                     # Daemon status
m9m serve --port 8080                  # Full server mode with REST API
`+"```"+`

The daemon starts automatically when needed and exits after 5 min idle.
<!-- m9m:end -->
`, workspaceName)
}
