# First Workflow

This guide walks you through creating and running your first m9m workflow.

## What You'll Build

A simple workflow that:

1. Starts with a trigger
2. Makes an HTTP request to fetch data
3. Transforms the response
4. Outputs the result

## Prerequisites

- m9m installed and running (`m9m serve`)
- `curl` for API calls (or use the Web UI)

## Step 1: Create the Workflow

Create a file named `my-first-workflow.json`:

```json
{
  "name": "My First Workflow",
  "nodes": [
    {
      "id": "start",
      "name": "Start",
      "type": "n8n-nodes-base.start",
      "position": [250, 300],
      "parameters": {}
    },
    {
      "id": "http",
      "name": "Fetch Posts",
      "type": "n8n-nodes-base.httpRequest",
      "position": [450, 300],
      "parameters": {
        "url": "https://jsonplaceholder.typicode.com/posts/1",
        "method": "GET"
      }
    },
    {
      "id": "set",
      "name": "Format Output",
      "type": "n8n-nodes-base.set",
      "position": [650, 300],
      "parameters": {
        "assignments": [
          {
            "name": "title",
            "value": "={{ $json.title }}"
          },
          {
            "name": "summary",
            "value": "Post ID: {{ $json.id }} by User {{ $json.userId }}"
          }
        ]
      }
    }
  ],
  "connections": {
    "Start": {
      "main": [[{"node": "Fetch Posts", "type": "main", "index": 0}]]
    },
    "Fetch Posts": {
      "main": [[{"node": "Format Output", "type": "main", "index": 0}]]
    }
  }
}
```

## Step 2: Run the Workflow

### Using the CLI

```bash
m9m run my-first-workflow.json
```

Expected output:

```json
{
  "status": "success",
  "data": [
    {
      "title": "sunt aut facere repellat provident occaecati excepturi optio reprehenderit",
      "summary": "Post ID: 1 by User 1"
    }
  ]
}
```

### Using the API

First, create the workflow:

```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @my-first-workflow.json
```

Then execute it:

```bash
curl -X POST http://localhost:8080/api/v1/workflows/{workflow-id}/execute
```

## Step 3: Understand the Workflow

Let's break down what each node does:

### Start Node

```json
{
  "type": "n8n-nodes-base.start",
  "parameters": {}
}
```

The Start node is the entry point. It passes through any input data or creates an empty item if none is provided.

### HTTP Request Node

```json
{
  "type": "n8n-nodes-base.httpRequest",
  "parameters": {
    "url": "https://jsonplaceholder.typicode.com/posts/1",
    "method": "GET"
  }
}
```

Makes an HTTP GET request and returns the response. The response JSON is available as `$json` in downstream nodes.

### Set Node

```json
{
  "type": "n8n-nodes-base.set",
  "parameters": {
    "assignments": [
      {"name": "title", "value": "={{ $json.title }}"},
      {"name": "summary", "value": "Post ID: {{ $json.id }} by User {{ $json.userId }}"}
    ]
  }
}
```

Transforms data by setting new fields. Uses expressions (`={{ }}` or `{{ }}`) to reference data from previous nodes.

### Connections

```json
{
  "Start": {
    "main": [[{"node": "Fetch Posts", "type": "main", "index": 0}]]
  },
  "Fetch Posts": {
    "main": [[{"node": "Format Output", "type": "main", "index": 0}]]
  }
}
```

Defines the data flow between nodes. Each connection specifies:

- Source node name
- Connection type (`main` for data flow)
- Target node and input index

## Adding a Webhook Trigger

Replace the Start node with a Webhook to trigger the workflow via HTTP:

```json
{
  "id": "webhook",
  "name": "Webhook",
  "type": "n8n-nodes-base.webhook",
  "position": [250, 300],
  "parameters": {
    "path": "/my-workflow",
    "httpMethod": "POST"
  }
}
```

Now you can trigger the workflow with:

```bash
curl -X POST http://localhost:8080/webhook/my-workflow \
  -H "Content-Type: application/json" \
  -d '{"custom": "data"}'
```

## Adding Error Handling

Wrap operations in a filter to handle errors:

```json
{
  "id": "check",
  "name": "Check Success",
  "type": "n8n-nodes-base.filter",
  "position": [550, 300],
  "parameters": {
    "conditions": [
      {
        "leftValue": "={{ $json.statusCode }}",
        "operator": "equals",
        "rightValue": 200
      }
    ]
  }
}
```

## Next Steps

Now that you've created your first workflow:

1. **[Core Concepts](concepts.md)** - Understand workflows, nodes, and data
2. **[Nodes Reference](../nodes/index.md)** - Explore all available nodes
3. **[Expressions](../expressions/index.md)** - Learn the expression syntax
4. **[Workflow Examples](../workflows/examples.md)** - See more complex workflows
