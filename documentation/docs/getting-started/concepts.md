# Core Concepts

This guide explains the fundamental concepts in m9m workflow automation.

## Workflows

A **workflow** is a directed graph of nodes that defines an automated process. Workflows consist of:

- **Nodes** - Individual processing steps
- **Connections** - Data flow paths between nodes
- **Settings** - Workflow-level configuration

### Workflow Structure

```json
{
  "name": "Example Workflow",
  "nodes": [...],
  "connections": {...},
  "settings": {...},
  "active": true
}
```

### Workflow States

| State | Description |
|-------|-------------|
| `active` | Workflow is enabled and can be triggered |
| `inactive` | Workflow is disabled |

## Nodes

**Nodes** are the building blocks of workflows. Each node performs a specific operation.

### Node Structure

```json
{
  "id": "unique-id",
  "name": "Display Name",
  "type": "n8n-nodes-base.httpRequest",
  "position": [x, y],
  "parameters": {...},
  "disabled": false
}
```

### Node Categories

| Category | Description | Examples |
|----------|-------------|----------|
| **Core** | Workflow control | Start |
| **Trigger** | Start workflows | Webhook, Cron |
| **Transform** | Data manipulation | Set, Filter, Code |
| **HTTP** | Web requests | HTTP Request |
| **Database** | Data storage | PostgreSQL, MySQL |
| **Messaging** | Communication | Slack, Discord |
| **AI** | AI/ML operations | OpenAI, Anthropic |
| **Cloud** | Cloud services | AWS S3, Azure Blob |
| **VCS** | Version control | GitHub, GitLab |

### Node Parameters

Parameters configure node behavior. They support:

- **Static values**: `"url": "https://api.example.com"`
- **Expressions**: `"url": "={{ $json.endpoint }}"`

## Data Items

Data flows through workflows as **items**. Each item is a JSON object.

### Item Structure

```json
{
  "json": {
    "field1": "value1",
    "field2": 123
  },
  "binary": {
    "data": "base64-encoded..."
  }
}
```

### Data Flow

```
┌─────────┐     ┌─────────┐     ┌─────────┐
│  Node A │────▶│  Node B │────▶│  Node C │
│         │     │         │     │         │
│ Output: │     │ Input:  │     │ Input:  │
│ [item1] │     │ [item1] │     │ [item2] │
│ [item2] │     │ [item2] │     │         │
└─────────┘     │ Output: │     └─────────┘
                │ [item2] │
                └─────────┘
```

- Nodes receive **all items** from connected upstream nodes
- Nodes can output **zero, one, or many items**
- Data is passed by reference (not copied)

## Connections

**Connections** define how data flows between nodes.

### Connection Structure

```json
{
  "Source Node Name": {
    "main": [
      [
        {"node": "Target Node", "type": "main", "index": 0}
      ]
    ]
  }
}
```

### Connection Types

| Type | Description |
|------|-------------|
| `main` | Primary data flow |

### Multiple Outputs

Some nodes have multiple outputs (e.g., Switch node):

```json
{
  "Switch": {
    "main": [
      [{"node": "Path A", "type": "main", "index": 0}],
      [{"node": "Path B", "type": "main", "index": 0}]
    ]
  }
}
```

## Expressions

**Expressions** allow dynamic values based on input data.

### Syntax

```javascript
// Full expression
={{ $json.fieldName }}

// Template string
Hello {{ $json.name }}!

// With operations
={{ $json.price * 1.1 }}
```

### Available Variables

| Variable | Description |
|----------|-------------|
| `$json` | Current item's JSON data |
| `$item` | Current item index |
| `$now` | Current timestamp |
| `$env` | Environment variables |
| `$input` | All input items |

See [Expressions](../expressions/index.md) for complete documentation.

## Executions

An **execution** is a single run of a workflow.

### Execution States

| State | Description |
|-------|-------------|
| `pending` | Queued, waiting to run |
| `running` | Currently executing |
| `success` | Completed successfully |
| `failed` | Completed with errors |
| `cancelled` | Manually cancelled |

### Execution Data

Each execution stores:

- Input data
- Output data from each node
- Error messages (if failed)
- Timing information
- Execution mode (manual, scheduled, webhook)

## Credentials

**Credentials** securely store authentication data for external services.

### Credential Structure

```json
{
  "name": "My API Key",
  "type": "apiKey",
  "data": {
    "apiKey": "secret-value"
  }
}
```

### Credential Types

| Type | Used For |
|------|----------|
| `apiKey` | API key authentication |
| `oauth2` | OAuth 2.0 flows |
| `basicAuth` | HTTP Basic Auth |
| `httpHeader` | Custom headers |

Credentials are encrypted at rest and injected into node parameters at runtime.

## Triggers

**Triggers** start workflow execution.

### Trigger Types

| Trigger | Description |
|---------|-------------|
| **Manual** | Started via API or CLI |
| **Webhook** | HTTP request to endpoint |
| **Schedule** | Cron-based timing |

### Webhook Triggers

```json
{
  "type": "n8n-nodes-base.webhook",
  "parameters": {
    "path": "/my-webhook",
    "httpMethod": "POST"
  }
}
```

Endpoint: `http://localhost:8080/webhook/my-webhook`

### Scheduled Triggers

```json
{
  "type": "n8n-nodes-base.cron",
  "parameters": {
    "cronExpression": "0 9 * * MON"
  }
}
```

## Job Queue

m9m uses a **job queue** for workflow execution.

### Queue Behavior

1. Workflow execution request → Job created
2. Job queued with status `pending`
3. Worker picks up job → status `running`
4. Execution completes → status `success` or `failed`

### Queue Types

| Type | Persistence | Use Case |
|------|-------------|----------|
| `memory` | No | Development, testing |
| `sqlite` | Yes | Production (single node) |

## Summary

| Concept | Description |
|---------|-------------|
| **Workflow** | Directed graph of automated steps |
| **Node** | Single processing operation |
| **Connection** | Data flow path between nodes |
| **Item** | Unit of data (JSON object) |
| **Expression** | Dynamic value using input data |
| **Execution** | Single workflow run |
| **Credential** | Secure authentication storage |
| **Trigger** | Workflow start mechanism |
| **Queue** | Execution job management |

## Next Steps

- [Create workflows](../workflows/creating.md)
- [Explore nodes](../nodes/index.md)
- [Learn expressions](../expressions/index.md)
