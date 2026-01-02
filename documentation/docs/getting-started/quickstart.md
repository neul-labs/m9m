# Quick Start

Get m9m running and execute your first workflow in under 5 minutes.

## Start the Server

Start m9m with default settings:

```bash
m9m serve
```

You should see:

```
INFO  Starting m9m server
INFO  Server listening on http://0.0.0.0:8080
INFO  Metrics available at http://0.0.0.0:9090/metrics
INFO  Ready to execute workflows
```

## Create a Simple Workflow

Create a file called `hello-world.json`:

```json
{
  "name": "Hello World",
  "nodes": [
    {
      "id": "start",
      "type": "n8n-nodes-base.start",
      "position": [250, 300],
      "parameters": {}
    },
    {
      "id": "set",
      "type": "n8n-nodes-base.set",
      "position": [450, 300],
      "parameters": {
        "values": {
          "string": [
            {
              "name": "message",
              "value": "Hello from m9m!"
            },
            {
              "name": "timestamp",
              "value": "={{ $now.toISO() }}"
            }
          ]
        }
      }
    }
  ],
  "connections": {
    "start": {
      "main": [
        [
          {
            "node": "set",
            "type": "main",
            "index": 0
          }
        ]
      ]
    }
  }
}
```

## Execute the Workflow

### Via CLI

Execute directly from command line:

```bash
m9m execute hello-world.json
```

Output:

```json
{
  "success": true,
  "data": [
    {
      "message": "Hello from m9m!",
      "timestamp": "2024-01-15T10:30:00.000Z"
    }
  ],
  "executionTime": "12ms"
}
```

### Via REST API

Execute via the API:

```bash
curl -X POST http://localhost:8080/api/v1/workflows/execute \
  -H "Content-Type: application/json" \
  -d @hello-world.json
```

## A More Practical Example

Create a workflow that fetches data from an API:

```json
{
  "name": "API Fetch Example",
  "nodes": [
    {
      "id": "start",
      "type": "n8n-nodes-base.start",
      "position": [250, 300],
      "parameters": {}
    },
    {
      "id": "http",
      "type": "n8n-nodes-base.httpRequest",
      "position": [450, 300],
      "parameters": {
        "url": "https://jsonplaceholder.typicode.com/posts/1",
        "method": "GET"
      }
    },
    {
      "id": "transform",
      "type": "n8n-nodes-base.set",
      "position": [650, 300],
      "parameters": {
        "values": {
          "string": [
            {
              "name": "title",
              "value": "={{ $json.title }}"
            },
            {
              "name": "summary",
              "value": "Post by user {{ $json.userId }}"
            }
          ]
        }
      }
    }
  ],
  "connections": {
    "start": {
      "main": [[{"node": "http", "type": "main", "index": 0}]]
    },
    "http": {
      "main": [[{"node": "transform", "type": "main", "index": 0}]]
    }
  }
}
```

Execute it:

```bash
m9m execute api-fetch.json
```

## Using the Web UI

Open [http://localhost:8080](http://localhost:8080) in your browser to access the visual workflow editor.

### Create a Workflow

1. Click **New Workflow**
2. Drag nodes from the sidebar
3. Connect nodes by clicking and dragging between ports
4. Configure node parameters in the right panel
5. Click **Execute** to run

### Import Existing Workflows

1. Click **Import**
2. Paste workflow JSON or upload a file
3. Click **Import**

## Health Check

Verify the server is healthy:

```bash
curl http://localhost:8080/health
```

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": "5m30s"
}
```

## View Metrics

Access Prometheus metrics:

```bash
curl http://localhost:9090/metrics
```

Key metrics include:

- `m9m_workflow_executions_total` - Total workflow executions
- `m9m_workflow_execution_duration_seconds` - Execution time histogram
- `m9m_node_executions_total` - Per-node execution counts
- `m9m_queue_size` - Current queue depth

## Common Commands

```bash
# Start server
m9m serve

# Execute workflow file
m9m execute workflow.json

# Execute with input data
m9m execute workflow.json --input '{"key": "value"}'

# Validate workflow
m9m validate workflow.json

# List available nodes
m9m nodes list

# Show node info
m9m nodes info n8n-nodes-base.httpRequest
```

## Next Steps

Now that you have m9m running:

1. [Build Your First Workflow](first-workflow.md) - Complete tutorial
2. [Learn Expressions](../user-guide/expressions.md) - Dynamic data references
3. [Explore Nodes](../nodes/overview.md) - Available integrations
4. [Set Up Credentials](../user-guide/credentials.md) - Connect to services
