---
title: Your first m9m workflow
description: Build, run, and trigger your first m9m workflow in five minutes. Start node, HTTP request, Set transform, webhook trigger, error handling.
keywords: m9m tutorial, workflow tutorial, first workflow, n8n workflow, webhook, expression syntax
---

# Your first m9m workflow

A five-minute walkthrough: fetch JSON from an API, transform the response, return the result. Then add a webhook trigger and basic error handling.

## What you'll build

```
┌─────────┐    ┌─────────────┐    ┌──────────────┐
│  Start  │───▶│ Fetch Posts │───▶│ Format Output│
└─────────┘    └─────────────┘    └──────────────┘
```

A workflow that:

1. Starts via the `Start` node (or a webhook trigger — added later)
2. Fetches a JSON post from `jsonplaceholder.typicode.com`
3. Transforms the response into a compact summary
4. Returns the output

## Prerequisites

- m9m installed ([installation guide](installation.md))
- `curl` for API calls (or the Web UI at `http://localhost:8080`)

## Step 1 — define the workflow

Save as `my-first-workflow.json`:

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
          { "name": "title",   "value": "={{ $json.title }}" },
          { "name": "summary", "value": "Post ID: {{ $json.id }} by User {{ $json.userId }}" }
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

This is the same JSON shape n8n uses. Workflows exported from n8n drop in here.

## Step 2 — run it

### CLI

```bash
m9m exec my-first-workflow.json
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

### REST API

```bash
# Create
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @my-first-workflow.json

# Execute
curl -X POST http://localhost:8080/api/v1/workflows/{workflow-id}/execute
```

## Step 3 — what each node does

### Start node

```json
{ "type": "n8n-nodes-base.start", "parameters": {} }
```

Entry point. Passes input through, or emits a single empty data item if no input is provided.

### HTTP Request node

```json
{
  "type": "n8n-nodes-base.httpRequest",
  "parameters": {
    "url": "https://jsonplaceholder.typicode.com/posts/1",
    "method": "GET"
  }
}
```

Makes an HTTP call. The parsed response body is exposed to downstream nodes as `$json`.

### Set node

```json
{
  "type": "n8n-nodes-base.set",
  "parameters": {
    "assignments": [
      { "name": "title",   "value": "={{ $json.title }}" },
      { "name": "summary", "value": "Post ID: {{ $json.id }} by User {{ $json.userId }}" }
    ]
  }
}
```

Reshapes data with expressions. Use `={{ ... }}` for a value computed entirely by an expression, or interpolate inline with `{{ ... }}` inside a string.

### Connections

```json
{
  "Start":       { "main": [[{ "node": "Fetch Posts",  "type": "main", "index": 0 }]] },
  "Fetch Posts": { "main": [[{ "node": "Format Output","type": "main", "index": 0 }]] }
}
```

Defines the directed graph: which node's output feeds which node's input.

## Step 4 — trigger via webhook

Replace the `Start` node with a webhook:

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

Then trigger it:

```bash
curl -X POST http://localhost:8080/webhook/my-workflow \
  -H "Content-Type: application/json" \
  -d '{"custom": "data"}'
```

The posted body is available as `$json` in downstream nodes.

## Step 5 — add error handling

Insert a filter to short-circuit on non-2xx responses:

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

For richer error handling, see [Workflow patterns › Error handling](../workflows/examples.md).

## Next steps

1. **[Core Concepts](concepts.md)** — workflows, nodes, data flow, expressions
2. **[Nodes Reference](../nodes/index.md)** — all 40+ available node types
3. **[Expressions](../expressions/index.md)** — full expression syntax and built-in functions
4. **[Workflow Examples](../workflows/examples.md)** — common patterns
5. **[Migrate from n8n](../migrate-from-n8n.md)** — import your existing workflows
