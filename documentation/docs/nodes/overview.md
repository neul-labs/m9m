# Nodes Overview

Nodes are the building blocks of m9m workflows. Each node performs a specific action or transformation.

## Node Categories

| Category | Description | Examples |
|----------|-------------|----------|
| **Triggers** | Start workflow execution | Webhook, Cron, Start |
| **Transform** | Manipulate data | Set, Filter, Merge, Code |
| **HTTP/API** | Make HTTP requests | HTTP Request, GraphQL |
| **Database** | Database operations | PostgreSQL, MySQL, MongoDB |
| **Messaging** | Send messages | Slack, Discord, Email |
| **AI** | AI/LLM operations | OpenAI, Anthropic |
| **Cloud** | Cloud services | AWS S3, GCP Storage |
| **VCS** | Version control | GitHub, GitLab |

## Node Structure

Every node has:

```json
{
  "id": "unique-node-id",
  "type": "n8n-nodes-base.nodeType",
  "position": [x, y],
  "parameters": {},
  "credentials": {}
}
```

### Properties

| Property | Description |
|----------|-------------|
| `id` | Unique identifier within the workflow |
| `type` | Node type (e.g., `n8n-nodes-base.httpRequest`) |
| `position` | Visual position in editor `[x, y]` |
| `parameters` | Configuration options |
| `credentials` | Authentication reference |

## Data Flow

Nodes process data as **items**. Each item contains:

```json
{
  "json": {
    "field1": "value1",
    "field2": "value2"
  },
  "binary": {
    "data": "<base64-encoded>"
  }
}
```

### Input and Output

- Nodes receive items from connected upstream nodes
- Process each item according to their configuration
- Output items to connected downstream nodes

```
[Input Items] → [Node Processing] → [Output Items]
```

### Multiple Items

Most nodes process each input item independently:

```
Input: [item1, item2, item3]
       ↓       ↓       ↓
      [Node Processing]
       ↓       ↓       ↓
Output: [result1, result2, result3]
```

## Common Parameters

### All Nodes

| Parameter | Description |
|-----------|-------------|
| `continueOnFail` | Continue workflow on error |
| `retryOnFail` | Retry on failure |
| `maxRetries` | Maximum retry attempts |

### Expression Support

Most parameters support expressions:

```json
{
  "parameters": {
    "url": "={{ $env.API_URL }}/users/{{ $json.userId }}"
  }
}
```

## Trigger Nodes

Trigger nodes start workflow execution.

### Start

Manual execution trigger:

```json
{
  "type": "n8n-nodes-base.start",
  "parameters": {}
}
```

### Webhook

HTTP endpoint trigger:

```json
{
  "type": "n8n-nodes-base.webhook",
  "parameters": {
    "path": "my-webhook",
    "httpMethod": "POST"
  }
}
```

### Cron

Schedule-based trigger:

```json
{
  "type": "n8n-nodes-base.cron",
  "parameters": {
    "triggerTimes": {
      "item": [{"mode": "everyHour"}]
    }
  }
}
```

## Action Nodes

Action nodes perform operations.

### HTTP Request

Make HTTP calls:

```json
{
  "type": "n8n-nodes-base.httpRequest",
  "parameters": {
    "url": "https://api.example.com/data",
    "method": "GET",
    "headers": {
      "Accept": "application/json"
    }
  }
}
```

### Code

Execute custom code:

```json
{
  "type": "n8n-nodes-base.code",
  "parameters": {
    "language": "javascript",
    "code": "return items.map(item => ({ ...item.json, processed: true }));"
  }
}
```

## Transform Nodes

Transform nodes manipulate data.

### Set

Set field values:

```json
{
  "type": "n8n-nodes-base.set",
  "parameters": {
    "values": {
      "string": [
        {"name": "fullName", "value": "={{ $json.firstName }} {{ $json.lastName }}"}
      ]
    }
  }
}
```

### Filter

Filter items based on conditions:

```json
{
  "type": "n8n-nodes-base.filter",
  "parameters": {
    "conditions": {
      "number": [
        {"value1": "={{ $json.age }}", "operation": "larger", "value2": 18}
      ]
    }
  }
}
```

### Merge

Combine data from multiple sources:

```json
{
  "type": "n8n-nodes-base.merge",
  "parameters": {
    "mode": "mergeByKey",
    "propertyName1": "id",
    "propertyName2": "userId"
  }
}
```

## Node Connections

### Single Output

Connect to one downstream node:

```json
{
  "connections": {
    "source-node": {
      "main": [[{"node": "target-node", "type": "main", "index": 0}]]
    }
  }
}
```

### Multiple Outputs

Some nodes have multiple output branches:

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

### Parallel Connections

Send to multiple nodes simultaneously:

```json
{
  "connections": {
    "source": {
      "main": [
        [
          {"node": "branch-a", "type": "main", "index": 0},
          {"node": "branch-b", "type": "main", "index": 0}
        ]
      ]
    }
  }
}
```

## Listing Nodes

### CLI

```bash
# List all available nodes
m9m nodes list

# Filter by category
m9m nodes list --category transform

# Search nodes
m9m nodes search "database"
```

### API

```bash
curl http://localhost:8080/api/v1/nodes
```

## Node Information

### CLI

```bash
m9m nodes info n8n-nodes-base.httpRequest
```

Output:
```
Name: HTTP Request
Type: n8n-nodes-base.httpRequest
Category: http
Description: Make HTTP requests

Parameters:
  - url (string, required): The URL to request
  - method (options): HTTP method (GET, POST, PUT, DELETE)
  - headers (object): Request headers
  - body (object): Request body
```

## Performance Considerations

### Batch Processing

Use `splitInBatches` for large datasets:

```json
{
  "type": "n8n-nodes-base.splitInBatches",
  "parameters": {
    "batchSize": 100
  }
}
```

### Parallel Execution

m9m executes independent branches in parallel automatically.

### Memory Usage

For large binary data, use streaming nodes when available.

## Next Steps

- [Transform Nodes](transform.md) - Data manipulation
- [Trigger Nodes](triggers.md) - Workflow triggers
- [Database Nodes](databases.md) - Database integrations
- [Custom Nodes](custom-nodes.md) - Build your own
