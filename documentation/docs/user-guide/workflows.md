# Workflows

Workflows are the core concept in m9m. They define automated processes as a series of connected nodes.

## Workflow Structure

A workflow consists of:

- **Nodes**: Individual processing steps
- **Connections**: Links between nodes defining data flow
- **Settings**: Configuration for the workflow itself

### Basic Workflow JSON

```json
{
  "name": "My Workflow",
  "active": true,
  "nodes": [
    {
      "id": "node-1",
      "type": "n8n-nodes-base.start",
      "position": [250, 300],
      "parameters": {}
    }
  ],
  "connections": {},
  "settings": {
    "executionOrder": "v1"
  }
}
```

## Creating Workflows

### Via Web UI

1. Open the m9m dashboard
2. Click **New Workflow**
3. Drag nodes from the node palette
4. Connect nodes by dragging between ports
5. Configure parameters in the side panel
6. Save the workflow

### Via CLI

```bash
# Create from JSON file
m9m workflow create my-workflow.json

# Create from template
m9m workflow create --template http-request
```

### Via API

```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @workflow.json
```

## Node Types

### Trigger Nodes

Start workflow execution:

| Node | Description |
|------|-------------|
| `cron` | Schedule-based triggers |
| `webhook` | HTTP webhook triggers |
| `start` | Manual execution trigger |

### Action Nodes

Perform operations:

| Node | Description |
|------|-------------|
| `httpRequest` | Make HTTP calls |
| `function` | Run custom JavaScript |
| `code` | Execute Python/JavaScript code |

### Transform Nodes

Manipulate data:

| Node | Description |
|------|-------------|
| `set` | Set field values |
| `filter` | Filter items |
| `merge` | Combine data streams |
| `splitInBatches` | Process in chunks |

## Connections

Connections define how data flows between nodes.

### Single Connection

```json
{
  "connections": {
    "node-1": {
      "main": [
        [
          {
            "node": "node-2",
            "type": "main",
            "index": 0
          }
        ]
      ]
    }
  }
}
```

### Multiple Outputs

Some nodes have multiple output ports:

```json
{
  "connections": {
    "if-node": {
      "main": [
        [{"node": "true-branch", "type": "main", "index": 0}],
        [{"node": "false-branch", "type": "main", "index": 0}]
      ]
    }
  }
}
```

### Parallel Execution

Connect one node to multiple downstream nodes:

```json
{
  "connections": {
    "start": {
      "main": [
        [
          {"node": "branch-1", "type": "main", "index": 0},
          {"node": "branch-2", "type": "main", "index": 0}
        ]
      ]
    }
  }
}
```

## Workflow Settings

### Execution Order

Control how nodes execute:

```json
{
  "settings": {
    "executionOrder": "v1"
  }
}
```

- `v0`: Legacy breadth-first execution
- `v1`: Optimized depth-first execution (recommended)

### Error Handling

Configure workflow-level error behavior:

```json
{
  "settings": {
    "errorWorkflow": "error-handler-workflow-id",
    "saveDataSuccessExecution": "all",
    "saveDataErrorExecution": "all"
  }
}
```

### Timeout

Set maximum execution time:

```json
{
  "settings": {
    "executionTimeout": 300
  }
}
```

## Managing Workflows

### List Workflows

```bash
m9m workflow list
```

```
ID        NAME                  STATUS    LAST RUN
wf-001    Site Monitor          active    2 min ago
wf-002    Data Sync             active    1 hour ago
wf-003    Report Generator      inactive  3 days ago
```

### Activate/Deactivate

```bash
# Activate
m9m workflow activate wf-001

# Deactivate
m9m workflow deactivate wf-001
```

### Delete

```bash
m9m workflow delete wf-001
```

### Export

```bash
m9m workflow export wf-001 > workflow.json
```

## Execution

### Manual Execution

```bash
# Execute workflow
m9m execute workflow.json

# With input data
m9m execute workflow.json --input '{"key": "value"}'

# Specific node only
m9m execute workflow.json --start-node "process-data"
```

### Execution History

View past executions:

```bash
m9m executions list --workflow wf-001 --limit 10
```

```
ID          STATUS    DURATION    STARTED
exec-100    success   1.2s        2024-01-15 10:30:00
exec-099    success   0.8s        2024-01-15 10:25:00
exec-098    error     5.1s        2024-01-15 10:20:00
```

### Execution Details

```bash
m9m executions get exec-100
```

## Versioning

m9m supports workflow versioning for change tracking.

### Save Version

```bash
m9m workflow version save wf-001 --message "Added error handling"
```

### List Versions

```bash
m9m workflow versions wf-001
```

### Restore Version

```bash
m9m workflow version restore wf-001 --version 3
```

## Best Practices

### Naming Conventions

- Use descriptive workflow names
- Prefix with category: `sync-`, `notify-`, `process-`
- Include target system: `sync-salesforce-to-postgres`

### Error Handling

- Add error handlers for critical workflows
- Use try/catch in code nodes
- Set up notification for failures

### Performance

- Use `splitInBatches` for large datasets
- Avoid unnecessary data transformations
- Cache frequently accessed data

### Organization

- Group related workflows in folders
- Document workflow purpose in settings
- Tag workflows for filtering

## Next Steps

- [Expressions](expressions.md) - Use dynamic data in workflows
- [Credentials](credentials.md) - Manage authentication
- [Error Handling](error-handling.md) - Handle failures gracefully
