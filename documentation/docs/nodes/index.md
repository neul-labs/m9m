# Nodes Overview

Nodes are the building blocks of m9m workflows. Each node performs a specific operation on data.

## Available Node Categories

| Category | Nodes | Description |
|----------|-------|-------------|
| [Core](core.md) | 1 | Workflow control nodes |
| [Transform](transform.md) | 9 | Data transformation |
| [HTTP](http.md) | 1 | Web requests |
| [Triggers](triggers.md) | 2 | Workflow triggers |
| [Database](database.md) | 3 | Database operations |
| [Messaging](messaging.md) | 2 | Chat platforms |
| [AI & LLM](ai.md) | 2 | AI services |
| [CLI Execution](cli.md) | 1 | Sandboxed CLI commands & AI agents |
| [Cloud Storage](cloud.md) | 4 | Cloud providers |
| [Version Control](vcs.md) | 2 | Git platforms |
| [Email](email.md) | 1 | Email operations |
| [File Operations](file.md) | 2 | File system |
| **Total** | **30+** | |

## Node Structure

Every node follows this JSON structure:

```json
{
  "id": "unique-identifier",
  "name": "Display Name",
  "type": "n8n-nodes-base.nodeType",
  "position": [x, y],
  "parameters": {
    "param1": "value1",
    "param2": "={{ $json.dynamicValue }}"
  },
  "disabled": false
}
```

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier within workflow |
| `name` | string | Display name (used in connections) |
| `type` | string | Node type identifier |
| `position` | [x, y] | Position in visual editor |
| `parameters` | object | Node-specific configuration |

### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `disabled` | boolean | `false` | Skip this node during execution |

## Node Types

All m9m nodes use the `n8n-nodes-base.*` prefix for n8n compatibility:

```
n8n-nodes-base.start
n8n-nodes-base.httpRequest
n8n-nodes-base.set
n8n-nodes-base.filter
...
```

## Data Flow

Nodes receive input data as an array of items and produce output data:

```
Input: [{json: {...}}, {json: {...}}]
           │
           ▼
    ┌─────────────┐
    │    Node     │
    │  (process)  │
    └─────────────┘
           │
           ▼
Output: [{json: {...}}]
```

### Input Data

- Nodes receive **all items** from connected upstream nodes
- Each item has a `json` property with the data
- Items may also have `binary` data for files

### Output Data

- Nodes can output **zero, one, or many items**
- Output items are passed to all connected downstream nodes

## Expression Support

Most node parameters support expressions:

```json
{
  "parameters": {
    "url": "https://api.example.com/users/{{ $json.userId }}",
    "body": "={{ JSON.stringify($json) }}"
  }
}
```

See [Expressions](../expressions/index.md) for full documentation.

## Credential Injection

Nodes that require authentication can reference credentials:

```json
{
  "type": "n8n-nodes-base.slack",
  "parameters": {
    "token": "={{ $credentials.slackApi.token }}"
  }
}
```

## Error Handling

By default, node errors stop workflow execution. Use the Filter node to handle errors gracefully:

```json
{
  "type": "n8n-nodes-base.filter",
  "parameters": {
    "conditions": [
      {
        "leftValue": "={{ $json.error }}",
        "operator": "notExists"
      }
    ]
  }
}
```

## Quick Reference

### Most Used Nodes

| Node | Type | Use Case |
|------|------|----------|
| Start | `n8n-nodes-base.start` | Workflow entry point |
| HTTP Request | `n8n-nodes-base.httpRequest` | API calls |
| Set | `n8n-nodes-base.set` | Set/modify fields |
| Filter | `n8n-nodes-base.filter` | Conditional filtering |
| Code | `n8n-nodes-base.code` | Custom logic |
| CLI Execute | `n8n-nodes-base.cliExecute` | Run CLI tools & AI agents |
| Webhook | `n8n-nodes-base.webhook` | HTTP triggers |

### By Use Case

| Use Case | Recommended Nodes |
|----------|-------------------|
| API Integration | HTTP Request |
| Data Transformation | Set, Filter, Code |
| Scheduled Tasks | Cron |
| External Triggers | Webhook |
| Notifications | Slack, Discord, Email |
| AI Processing | OpenAI, Anthropic |
| AI Coding Agents | CLI Execute (Claude Code, Codex, Aider) |
| File Storage | AWS S3, Azure Blob, GCP Storage |
