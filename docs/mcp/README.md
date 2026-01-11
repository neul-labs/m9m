# MCP Integration - Claude Code for Workflow Automation

m9m includes a built-in MCP (Model Context Protocol) server that enables Claude Code to orchestrate workflows conversationally. This transforms workflow automation into a natural language interface.

## Overview

The MCP server exposes m9m's full workflow automation capabilities through 37 tools across 7 categories, allowing Claude Code to:

- **Discover** available node types and their capabilities
- **Execute** quick actions (HTTP requests, Slack messages, AI completions)
- **Create** complex multi-node workflows
- **Monitor** execution status and debug failures
- **Extend** the platform with custom JavaScript or REST-based nodes

## Installation

### Building the MCP Server

```bash
# From the m9m repository root
go build -o mcp-server ./cmd/mcp-server

# Verify the build
./mcp-server --version
# Output: m9m-mcp-server version 1.0.0
```

### Configuring Claude Code

Add the MCP server to your Claude Code configuration:

**macOS/Linux**: `~/.claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "m9m": {
      "command": "/absolute/path/to/mcp-server",
      "args": ["--data", "/path/to/data/directory"]
    }
  }
}
```

Restart Claude Code to load the MCP server.

## Server Modes

The MCP server supports two operational modes:

### Local Mode (Default)

Runs a complete m9m instance locally with persistent storage.

```bash
# SQLite storage (default)
./mcp-server
# Creates ./data/m9m.db automatically

# Explicit SQLite path
./mcp-server --db /path/to/m9m.db

# PostgreSQL storage
./mcp-server --postgres "postgres://user:pass@localhost:5432/m9m"

# Custom data directory
./mcp-server --data /var/lib/m9m
```

**Use Local Mode when:**
- Developing and testing workflows
- Running on a single machine
- You want a self-contained setup

### Cloud Mode

Connects to a remote m9m API server, delegating all operations.

```bash
# Connect to cloud m9m instance
./mcp-server --api-url https://m9m.example.com

# Connect to local m9m API server
./mcp-server --api-url http://localhost:8080
```

**Use Cloud Mode when:**
- Connecting to a shared team instance
- Working with production workflows
- The m9m server is running elsewhere (Kubernetes, Docker, etc.)

## Command-Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `--db` | SQLite database path | `./data/m9m.db` |
| `--postgres` | PostgreSQL connection URL | - |
| `--api-url` | Remote m9m API URL (cloud mode) | - |
| `--data` | Data directory for storage and plugins | `./data` |
| `--plugins` | Plugin directory | `{data}/plugins` |
| `--version` | Show version and exit | - |

## Available Tools

### 1. Node Discovery (4 tools)

Explore available node types and their capabilities.

| Tool | Description |
|------|-------------|
| `node_types_list` | List all available node types grouped by category |
| `node_type_get` | Get detailed schema for a specific node type |
| `node_categories_list` | List all node categories |
| `node_search` | Search nodes by name or description |

**Example:**
```
You: "What nodes are available for sending messages?"
Claude: [Uses node_search to find messaging nodes like Slack, Discord, Email]
```

### 2. Quick Actions (6 tools)

Execute single operations without creating a full workflow.

| Tool | Description |
|------|-------------|
| `http_request` | Make HTTP requests (GET, POST, PUT, DELETE) |
| `send_slack` | Send messages to Slack channels |
| `send_discord` | Send messages to Discord channels |
| `ai_openai` | Get completions from OpenAI models |
| `ai_anthropic` | Get completions from Anthropic models |
| `transform_data` | Transform data using expressions |

**Example:**
```
You: "Fetch the latest weather from api.weather.com"
Claude: [Uses http_request to GET the weather API]
```

### 3. Workflow Management (9 tools)

Full CRUD operations for workflows.

| Tool | Description |
|------|-------------|
| `workflow_list` | List workflows with filters |
| `workflow_get` | Get workflow by ID |
| `workflow_create` | Create new workflow |
| `workflow_update` | Update existing workflow |
| `workflow_delete` | Delete a workflow |
| `workflow_activate` | Activate a workflow |
| `workflow_deactivate` | Deactivate a workflow |
| `workflow_duplicate` | Duplicate a workflow |
| `workflow_validate` | Validate workflow structure |

**Example:**
```
You: "Create a workflow that fetches user data from our API and sends a summary to Slack"
Claude: [Uses workflow_create with HTTP Request and Slack nodes]
```

### 4. Execution (7 tools)

Run and monitor workflow executions.

| Tool | Description |
|------|-------------|
| `execution_run` | Execute workflow synchronously |
| `execution_run_async` | Execute asynchronously, return execution ID |
| `execution_get` | Get execution details |
| `execution_list` | List executions with filters |
| `execution_cancel` | Cancel running execution |
| `execution_retry` | Retry failed execution |
| `execution_wait` | Wait for execution to complete |

**Example:**
```
You: "Run my data-sync workflow and wait for it to complete"
Claude: [Uses execution_run_async, then execution_wait to monitor progress]
```

### 5. Debugging (5 tools)

Investigate execution issues and performance.

| Tool | Description |
|------|-------------|
| `debug_execution_logs` | Get detailed execution logs |
| `debug_node_output` | Get output from a specific node |
| `debug_list_events` | List audit events |
| `debug_performance` | Get performance metrics |
| `debug_live_status` | Get real-time execution status |

**Log Detail Levels:**
- `summary` - High-level execution flow
- `detailed` - Node inputs/outputs
- `verbose` - Full data + expression evaluations

**Example:**
```
You: "My workflow failed, show me what went wrong"
Claude: [Uses debug_execution_logs with level="detailed" to find the error]
```

### 6. Plugin Management (6 tools)

Create and manage custom nodes.

| Tool | Description |
|------|-------------|
| `plugin_create_js` | Create JavaScript plugin node (Goja runtime) |
| `plugin_create_rest` | Create REST API wrapper node |
| `plugin_list` | List installed plugins |
| `plugin_get` | Get plugin details and source |
| `plugin_reload` | Hot-reload a plugin |
| `plugin_delete` | Delete a plugin |

**Example:**
```
You: "Create a custom node that formats phone numbers"
Claude: [Uses plugin_create_js with formatting logic]
```

## Custom Node Development

### JavaScript Plugins

Create custom nodes using JavaScript (Goja runtime):

```javascript
module.exports = {
  description: {
    name: "Phone Formatter",
    description: "Formats phone numbers to E.164 format",
    category: "transform"
  },
  execute: function(inputData, params) {
    return inputData.map(item => ({
      json: {
        ...item.json,
        phone: formatE164(item.json[params.field])
      }
    }));
  }
};
```

**Available in JavaScript context:**
- `console.log()` - Logging
- `$json` - Current item's JSON data
- `$node` - Node information
- `$parameter` - Access to node parameters

### REST Wrapper Plugins

Wrap external REST APIs as reusable nodes:

```yaml
name: weather-api
description: Get weather data from WeatherAPI
category: integration
endpoint: https://api.weatherapi.com/v1/current.json
method: GET
headers:
  Content-Type: application/json
timeout: 10s
authType: apiKey
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Claude Code                              │
│                    (MCP Client)                              │
└─────────────────────────┬───────────────────────────────────┘
                          │ JSON-RPC over stdio
┌─────────────────────────▼───────────────────────────────────┐
│                   MCP Server                                 │
│              (cmd/mcp-server)                                │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ Tool        │  │ Transport   │  │ Mode Selection      │  │
│  │ Registry    │  │ (stdio)     │  │ Local / Cloud       │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
└─────────────────────────┬───────────────────────────────────┘
                          │
         ┌────────────────┴────────────────┐
         │                                 │
┌────────▼────────┐              ┌────────▼────────┐
│   Local Mode    │              │   Cloud Mode    │
├─────────────────┤              ├─────────────────┤
│ SQLite/Postgres │              │  HTTP Client    │
│ Embedded Engine │              │  Remote API     │
│ Plugin Registry │              │                 │
└─────────────────┘              └─────────────────┘
```

## Example Workflows

### 1. API Health Check with Alerting

```
You: "Create a workflow that checks our API at https://api.example.com/health
      every 5 minutes. If it returns anything other than 200, send an alert
      to #ops-alerts on Slack."

Claude: I'll create a workflow with the following structure:
1. Cron trigger (every 5 minutes)
2. HTTP Request to your health endpoint
3. IF node to check status code
4. Slack node for alerts

[Creates workflow with workflow_create]
[Activates with workflow_activate]

Your health check workflow is now active and will alert #ops-alerts
if the API returns a non-200 status.
```

### 2. Data Pipeline with Transformation

```
You: "Build a workflow that fetches user data from our API, filters for
      active users, and sends a daily summary to the team Slack channel."

Claude: [Creates multi-node workflow]
- Start node (manual or scheduled trigger)
- HTTP Request to fetch users
- Filter node for active users
- Set node to format the summary
- Slack node to post the message
```

### 3. Custom Integration

```
You: "We use an internal API at internal.example.com/api/tickets.
      Create a custom node that can create tickets with title and description."

Claude: [Uses plugin_create_rest]
Created 'internal-tickets' node. You can now use it in workflows to
create tickets. Parameters: title (required), description (optional).
```

## Troubleshooting

### MCP Server Not Connecting

1. Verify the path in `claude_desktop_config.json` is absolute
2. Ensure the binary has execute permissions: `chmod +x mcp-server`
3. Check logs: `./mcp-server 2>&1 | head -20`

### Database Errors

```bash
# Reset SQLite database
rm ./data/m9m.db
./mcp-server  # Creates fresh database

# Test PostgreSQL connection
psql "postgres://user:pass@localhost/m9m" -c "SELECT 1"
```

### Cloud Mode Connection Issues

```bash
# Test API connectivity
curl https://m9m.example.com/health

# Verify authentication if required
curl -H "Authorization: Bearer TOKEN" https://m9m.example.com/api/v1/workflows
```

## Security Considerations

- **Local Mode**: Data stored in SQLite/Postgres on your machine
- **Cloud Mode**: All data handled by remote m9m server
- **Credentials**: Managed through m9m's credential system, never exposed in MCP responses
- **Plugins**: JavaScript runs in sandboxed Goja runtime

## See Also

- [Plugin System](../PLUGIN_SYSTEM.md) - Detailed plugin development guide
- [API Reference](../api/API_COMPATIBILITY.md) - REST API documentation
- [Deployment Guide](../deployment/DEPLOYMENT_GUIDE.md) - Production setup
