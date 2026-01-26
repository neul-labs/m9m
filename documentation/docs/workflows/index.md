# Workflows Overview

Workflows are the core of m9m - automated processes that connect nodes to accomplish tasks.

## What is a Workflow?

A workflow is a directed graph of nodes that:

1. **Starts** with a trigger (manual, webhook, schedule)
2. **Processes** data through connected nodes
3. **Outputs** results or performs actions

```
┌─────────┐     ┌─────────┐     ┌─────────┐     ┌─────────┐
│ Trigger │────▶│  Fetch  │────▶│Transform│────▶│  Send   │
│         │     │  Data   │     │  Data   │     │  Alert  │
└─────────┘     └─────────┘     └─────────┘     └─────────┘
```

## Workflow Structure

```json
{
  "name": "My Workflow",
  "nodes": [...],
  "connections": {...},
  "settings": {},
  "active": true
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Workflow name |
| `nodes` | array | Yes | List of nodes |
| `connections` | object | Yes | Node connections |
| `settings` | object | No | Workflow settings |
| `active` | boolean | No | Enable triggers |

## Workflow Lifecycle

```
Create → Edit → Test → Activate → Execute → Monitor
                 ↑                              ↓
                 └──────────────────────────────┘
                         (iterate)
```

## Key Concepts

### Nodes

Building blocks that perform operations:

- **Triggers**: Start workflows (Webhook, Cron)
- **Actions**: Perform tasks (HTTP, Database, Email)
- **Transform**: Modify data (Set, Filter, Code)

### Connections

Define data flow between nodes:

```json
{
  "Source Node": {
    "main": [[{"node": "Target Node", "type": "main", "index": 0}]]
  }
}
```

### Data Items

Data flows as items - JSON objects with optional binary:

```json
{
  "json": {"field": "value"},
  "binary": {}
}
```

### Expressions

Dynamic values using input data:

```
{{ $json.fieldName }}
```

## Workflow Types

### Manual Workflows

Started via API or CLI:

```bash
m9m run my-workflow.json
```

### Webhook Workflows

Triggered by HTTP requests:

```
POST http://localhost:8080/webhook/my-endpoint
```

### Scheduled Workflows

Run on a schedule:

```json
{
  "type": "n8n-nodes-base.cron",
  "parameters": {
    "cronExpression": "0 9 * * *"
  }
}
```

## Best Practices

1. **Clear naming** - Use descriptive node and workflow names
2. **Error handling** - Add filters to check for errors
3. **Modular design** - Break complex logic into smaller workflows
4. **Test thoroughly** - Test with sample data before activating
5. **Monitor executions** - Watch for failures and optimize

## Next Steps

- [Creating Workflows](creating.md)
- [Executing Workflows](executing.md)
- [Data Flow](data-flow.md)
- [Examples](examples.md)
