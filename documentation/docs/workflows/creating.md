# Creating Workflows

Learn how to create workflows from scratch.

## Workflow JSON Structure

Every workflow has this structure:

```json
{
  "name": "Workflow Name",
  "nodes": [],
  "connections": {},
  "settings": {},
  "active": false
}
```

## Adding Nodes

Nodes are the building blocks:

```json
{
  "nodes": [
    {
      "id": "node-1",
      "name": "Start",
      "type": "n8n-nodes-base.start",
      "position": [250, 300],
      "parameters": {}
    },
    {
      "id": "node-2",
      "name": "HTTP Request",
      "type": "n8n-nodes-base.httpRequest",
      "position": [450, 300],
      "parameters": {
        "url": "https://api.example.com/data",
        "method": "GET"
      }
    }
  ]
}
```

### Node Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique identifier |
| `name` | string | Yes | Display name |
| `type` | string | Yes | Node type |
| `position` | [x, y] | Yes | Visual position |
| `parameters` | object | Yes | Node configuration |
| `disabled` | boolean | No | Skip this node |

## Defining Connections

Connect nodes to define data flow:

```json
{
  "connections": {
    "Start": {
      "main": [
        [
          {
            "node": "HTTP Request",
            "type": "main",
            "index": 0
          }
        ]
      ]
    }
  }
}
```

### Connection Format

```
"Source Node Name": {
  "main": [
    [  // Output index 0
      {
        "node": "Target Node Name",
        "type": "main",
        "index": 0  // Target input index
      }
    ]
  ]
}
```

### Multiple Outputs

Some nodes (like Switch) have multiple outputs:

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

## Complete Example

A workflow that fetches data and sends a Slack notification:

```json
{
  "name": "Data Notification",
  "nodes": [
    {
      "id": "start",
      "name": "Start",
      "type": "n8n-nodes-base.start",
      "position": [250, 300],
      "parameters": {}
    },
    {
      "id": "fetch",
      "name": "Fetch Data",
      "type": "n8n-nodes-base.httpRequest",
      "position": [450, 300],
      "parameters": {
        "url": "https://api.example.com/status",
        "method": "GET"
      }
    },
    {
      "id": "format",
      "name": "Format Message",
      "type": "n8n-nodes-base.set",
      "position": [650, 300],
      "parameters": {
        "assignments": [
          {
            "name": "message",
            "value": "Status: {{ $json.status }}"
          }
        ]
      }
    },
    {
      "id": "notify",
      "name": "Send Slack",
      "type": "n8n-nodes-base.slack",
      "position": [850, 300],
      "parameters": {
        "webhookUrl": "https://hooks.slack.com/...",
        "text": "={{ $json.message }}"
      }
    }
  ],
  "connections": {
    "Start": {
      "main": [[{"node": "Fetch Data", "type": "main", "index": 0}]]
    },
    "Fetch Data": {
      "main": [[{"node": "Format Message", "type": "main", "index": 0}]]
    },
    "Format Message": {
      "main": [[{"node": "Send Slack", "type": "main", "index": 0}]]
    }
  },
  "settings": {},
  "active": false
}
```

## Creating via API

```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d @workflow.json
```

## Creating via CLI

```bash
m9m create --from workflow.json
```

## Validating Workflows

Check before creating:

```bash
m9m validate workflow.json
```

Validates:

- Required fields present
- Valid node types
- No circular dependencies
- Connection targets exist

## Tips

### Node IDs

Use meaningful IDs:

```json
{"id": "fetch-user-data"}  // Good
{"id": "node-1"}           // Less clear
```

### Node Positioning

Space nodes for readability:

```json
{"position": [250, 300]}   // First node
{"position": [450, 300]}   // 200px right
{"position": [450, 500]}   // 200px down (branch)
```

### Comments

Add description to workflow:

```json
{
  "name": "My Workflow",
  "description": "Fetches data daily and sends report"
}
```
