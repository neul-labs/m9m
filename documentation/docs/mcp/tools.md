# MCP Tools Reference

The m9m MCP server provides **37 tools** organized into 7 categories.

## Node Discovery

Explore available node types and capabilities.

### node_types_list

List all available node types grouped by category.

**Parameters:** None

**Example Response:**
```json
{
  "categories": {
    "messaging": ["slack", "discord", "email"],
    "database": ["postgres", "mysql", "mongodb"],
    "ai": ["openai", "anthropic"],
    "transform": ["set", "filter", "code", "merge"]
  }
}
```

---

### node_type_get

Get detailed schema for a specific node type.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `nodeType` | string | Yes | Node type identifier (e.g., `n8n-nodes-base.slack`) |

---

### node_categories_list

List all available node categories.

---

### node_search

Search nodes by name or description.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `query` | string | Yes | Search term |

---

## Quick Actions

Execute single operations without creating workflows.

### http_request

Make HTTP requests to any URL.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `url` | string | Yes | Target URL |
| `method` | string | No | HTTP method (GET, POST, PUT, DELETE). Default: GET |
| `headers` | object | No | Request headers |
| `body` | any | No | Request body (for POST/PUT) |
| `timeout` | string | No | Request timeout (e.g., "30s") |

**Example:**
```
You: "GET https://api.github.com/users/octocat"
```

---

### send_slack

Send messages to Slack channels.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `channel` | string | Yes | Channel name or ID |
| `text` | string | Yes | Message text |
| `credentialId` | string | No | Slack credential ID |

---

### send_discord

Send messages to Discord channels.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `channelId` | string | Yes | Discord channel ID |
| `content` | string | Yes | Message content |
| `credentialId` | string | No | Discord credential ID |

---

### ai_openai

Get completions from OpenAI models.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `prompt` | string | Yes | The prompt to send |
| `model` | string | No | Model name (default: gpt-4) |
| `maxTokens` | integer | No | Maximum tokens |
| `credentialId` | string | No | OpenAI credential ID |

---

### ai_anthropic

Get completions from Anthropic models.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `prompt` | string | Yes | The prompt to send |
| `model` | string | No | Model name (default: claude-3-sonnet) |
| `maxTokens` | integer | No | Maximum tokens |
| `credentialId` | string | No | Anthropic credential ID |

---

### transform_data

Transform data using expressions.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `data` | any | Yes | Input data |
| `expression` | string | Yes | Transformation expression |

---

## Workflow Management

Full CRUD operations for workflows.

### workflow_list

List workflows with optional filters.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `active` | boolean | No | Filter by active status |
| `search` | string | No | Search in workflow names |
| `tags` | array | No | Filter by tags |
| `limit` | integer | No | Max results (default: 50) |
| `offset` | integer | No | Pagination offset |

---

### workflow_get

Get a workflow by ID.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `workflowId` | string | Yes | Workflow ID |

---

### workflow_create

Create a new workflow.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | Yes | Workflow name |
| `description` | string | No | Workflow description |
| `nodes` | array | Yes | Array of node definitions |
| `connections` | object | Yes | Node connections |
| `settings` | object | No | Workflow settings |

---

### workflow_update

Update an existing workflow.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `workflowId` | string | Yes | Workflow ID |
| `name` | string | No | New name |
| `nodes` | array | No | Updated nodes |
| `connections` | object | No | Updated connections |

---

### workflow_delete

Delete a workflow.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `workflowId` | string | Yes | Workflow ID |

---

### workflow_activate / workflow_deactivate

Activate or deactivate a workflow.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `workflowId` | string | Yes | Workflow ID |

---

### workflow_duplicate

Create a copy of a workflow.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `workflowId` | string | Yes | Workflow ID to duplicate |
| `name` | string | No | Name for the copy |

---

### workflow_validate

Validate workflow structure without saving.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `nodes` | array | Yes | Nodes to validate |
| `connections` | object | Yes | Connections to validate |

---

## Execution

Run and monitor workflow executions.

### execution_run

Execute a workflow synchronously (waits for completion).

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `workflowId` | string | Yes | Workflow ID |
| `inputData` | array | No | Input data items |

---

### execution_run_async

Execute a workflow asynchronously.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `workflowId` | string | Yes | Workflow ID |
| `inputData` | array | No | Input data items |

**Returns:** Execution ID for tracking

---

### execution_get

Get execution details.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `executionId` | string | Yes | Execution ID |

---

### execution_list

List executions with filters.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `workflowId` | string | No | Filter by workflow |
| `status` | string | No | Filter by status |
| `limit` | integer | No | Max results |

---

### execution_cancel

Cancel a running execution.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `executionId` | string | Yes | Execution ID |

---

### execution_retry

Retry a failed execution.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `executionId` | string | Yes | Execution ID |

---

### execution_wait

Wait for an execution to complete.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `executionId` | string | Yes | Execution ID |
| `timeout` | string | No | Max wait time (e.g., "5m") |

---

## Debugging

Investigate execution issues.

### debug_execution_logs

Get detailed execution logs.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `executionId` | string | Yes | Execution ID |
| `level` | string | No | Detail level: `summary`, `detailed`, `verbose` |
| `nodeFilter` | string | No | Filter logs for specific node |

---

### debug_node_output

Get output from a specific node.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `executionId` | string | Yes | Execution ID |
| `nodeName` | string | Yes | Node name |

---

### debug_list_events

List audit events.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `eventType` | string | No | Filter by type |
| `workflowId` | string | No | Filter by workflow |
| `limit` | integer | No | Max results |

---

### debug_performance

Get performance metrics.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `workflowId` | string | No | Workflow to analyze |
| `limit` | integer | No | Executions to analyze |

---

### debug_live_status

Get real-time status of running execution.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `executionId` | string | Yes | Execution ID |

---

## Plugin Management

Create and manage custom nodes.

### plugin_create_js

Create a JavaScript plugin node.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | Yes | Node name |
| `description` | string | No | Node description |
| `category` | string | No | Node category |
| `code` | string | Yes | JavaScript code |
| `parameters` | array | No | Parameter definitions |

---

### plugin_create_rest

Create a REST API wrapper node.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | Yes | Node name |
| `description` | string | No | Node description |
| `endpoint` | string | Yes | API endpoint URL |
| `method` | string | No | HTTP method |
| `headers` | object | No | Default headers |
| `timeout` | string | No | Request timeout |
| `authType` | string | No | Auth type: `none`, `bearer`, `basic`, `apiKey` |

---

### plugin_list

List all installed plugins.

---

### plugin_get

Get plugin details and source code.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | Yes | Plugin name |

---

### plugin_reload

Hot-reload a plugin without restart.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | No | Plugin to reload (all if empty) |

---

### plugin_delete

Delete a plugin.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | Yes | Plugin name |
