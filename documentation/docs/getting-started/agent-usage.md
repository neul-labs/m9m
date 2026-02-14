# Agent Usage Guide

This guide shows how an AI agent can use m9m as an orchestration backend.

## Recommended Integration Modes

1. MCP mode (recommended for LLM agents)
2. REST API mode (recommended for deterministic service-to-service control)

`m9m` supports both local and cloud operation:
- Local: run `mcp-server` or `m9m serve` on the same machine
- Cloud: point agents to a remote `m9m` API server

## Public Launch Notes

- Official launch path: single binary distribution (package manager/releases)
- Kubernetes manifests are experimental reference material
- Unauthenticated API exposure should be treated as internal/experimental only

## Option 1: MCP (Best For AI Assistants)

MCP gives agents a structured tool interface (workflow, execution, debug, plugin tools).

### Start MCP Server

```bash
go build -o mcp-server ./cmd/mcp-server
./mcp-server --db ./data/m9m.db
```

### Configure Client (example)

```json
{
  "mcpServers": {
    "m9m": {
      "command": "/absolute/path/to/mcp-server",
      "args": ["--db", "/absolute/path/to/data/m9m.db"]
    }
  }
}
```

### Core Agent Flow (MCP tools)

1. Discover capabilities: `node_types_list`, `node_search`
2. Create or update workflows: `workflow_create`, `workflow_update`
3. Execute: `execution_run` or `execution_run_async`
4. Observe: `execution_wait`, `execution_get`, `debug_execution_logs`
5. Recover: `execution_retry` or `execution_cancel`

## Option 2: REST API (Best For Backend Agents)

Base URL:

```text
http://localhost:8080/api/v1
```

### 1) Discover Workflows

```bash
curl "http://localhost:8080/api/v1/workflows?limit=20"
```

### 2) Execute Existing Workflow (Sync)

Request body supports either:
- raw array: `[{ "json": {...} }]`
- envelope: `{ "inputData": [{ "json": {...} }] }`

```bash
curl -X POST http://localhost:8080/api/v1/workflows/<workflow-id>/execute \
  -H "Content-Type: application/json" \
  -d '{"inputData":[{"json":{"task":"summarize"}}]}'
```

### 3) Execute Inline Workflow Definition

Use this when your agent composes workflows dynamically:

```bash
curl -X POST http://localhost:8080/api/v1/workflows/run \
  -H "Content-Type: application/json" \
  -d '{
    "workflow": {
      "name": "Inline Run",
      "nodes": [
        {"id":"start","name":"Start","type":"n8n-nodes-base.start","parameters":{}}
      ],
      "connections": {}
    },
    "inputData": [{"json":{"requestId":"abc-123"}}]
  }'
```

### 4) Run Async + Poll

```bash
curl -X POST http://localhost:8080/api/v1/workflows/<workflow-id>/execute-async \
  -H "Content-Type: application/json" \
  -d '{"inputData":[{"json":{"batchId":"b-42"}}]}'
```

Then:

```bash
curl "http://localhost:8080/api/v1/jobs/<job-id>"
curl "http://localhost:8080/api/v1/executions?workflowId=<workflow-id>&limit=10"
```

### 5) Cancel / Retry

Cancel:

```bash
curl -X POST http://localhost:8080/api/v1/executions/<execution-id>/cancel
```

Possible behavior:
- `202 Accepted`: cancellation requested (`status: "cancel_requested"`)
- `409 Conflict`: runtime cannot cancel this execution instance

Retry failed execution:

```bash
curl -X POST http://localhost:8080/api/v1/executions/<execution-id>/retry
```

## Practical Agent Pattern

Use a simple control loop:

1. Plan: choose existing workflow or construct inline workflow
2. Execute: run sync for short tasks, async for long tasks
3. Observe: poll executions/jobs and inspect logs
4. Recover: retry on transient failures, cancel when superseded
5. Persist: save useful workflows for future reuse

## Security and Reliability Guidance

- Prefer authenticated API access in production
- Use least-privilege credentials per workflow
- For CLI agent nodes, keep `sandboxEnabled=true`
- Set explicit timeouts/memory limits for long-running agent tasks
- Track execution IDs in your agent state for robust resume/recovery

