# Core Nodes

Core nodes provide fundamental workflow control functionality.

## Start Node

The Start node is the entry point for workflows that don't use trigger nodes.

### Type

```
n8n-nodes-base.start
```

### Description

The Start node initiates workflow execution. It:

- Passes through any input data provided
- Creates an empty item if no input is given
- Must be the first node in non-triggered workflows

### Parameters

The Start node has no configurable parameters.

### Example

```json
{
  "id": "start-1",
  "name": "Start",
  "type": "n8n-nodes-base.start",
  "position": [250, 300],
  "parameters": {}
}
```

### Input

- **None required** - Creates empty item `[{json: {}}]`
- **Optional** - Passes through provided input data

### Output

```json
[
  {
    "json": {}
  }
]
```

Or with input data:

```json
[
  {
    "json": {
      "inputField": "inputValue"
    }
  }
]
```

### Usage

#### Basic Workflow Entry

```json
{
  "nodes": [
    {
      "id": "start",
      "name": "Start",
      "type": "n8n-nodes-base.start",
      "position": [250, 300],
      "parameters": {}
    },
    {
      "id": "next",
      "name": "Next Step",
      "type": "n8n-nodes-base.set",
      "position": [450, 300],
      "parameters": {
        "assignments": [
          {"name": "message", "value": "Workflow started!"}
        ]
      }
    }
  ],
  "connections": {
    "Start": {
      "main": [[{"node": "Next Step", "type": "main", "index": 0}]]
    }
  }
}
```

#### With Input Data (CLI)

```bash
m9m run workflow.json --input '{"name": "John"}'
```

The Start node passes through `{"name": "John"}` to connected nodes.

#### With Input Data (API)

```bash
curl -X POST http://localhost:8080/api/v1/workflows/{id}/execute \
  -H "Content-Type: application/json" \
  -d '{"inputData": [{"json": {"name": "John"}}]}'
```

### When to Use

| Scenario | Use Start Node? |
|----------|-----------------|
| Manual execution | Yes |
| Webhook trigger | No - use Webhook node |
| Scheduled execution | No - use Cron node |
| Subworkflow | Yes |

### Best Practices

1. **One Start node per workflow** - Only one entry point needed
2. **Position first** - Place at the left of your workflow
3. **Clear naming** - Keep the default "Start" name for clarity

### Related Nodes

- [Webhook](triggers.md#webhook-node) - HTTP trigger
- [Cron](triggers.md#cron-node) - Scheduled trigger
