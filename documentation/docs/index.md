---
title: m9m — The n8n alternative without the bugs
description: m9m is a drop-in n8n alternative in Go. 5–10× faster, 70% lower memory, deterministic execution, zero npm dependencies. Run n8n workflows unchanged.
keywords: n8n alternative, n8n replacement, workflow automation, Go, MCP server, AI agents, Claude Code, workflow engine, iPaaS
---

# m9m

**The n8n alternative without the bugs — faster, more reliable workflow automation.**

m9m is an open-source workflow automation platform written in Go. It runs n8n workflow JSON unchanged, executes 5–10× faster, uses 70% less memory, and ships as a single 30 MB binary with zero runtime dependencies. No Node.js, no npm tree, no event-loop stalls.

## Why m9m? (vs n8n)

| | m9m | n8n |
|---|---|---|
| **Cold start** | ~500 ms | ~3 s |
| **Memory (idle)** | ~150 MB | ~512 MB |
| **Container size** | ~300 MB | ~1.2 GB |
| **Workflow execution** | Baseline | 5–10× slower |
| **Concurrent workflows** | 500 | 50 |
| **Runtime** | Single static Go binary | Node.js + ~1,000 npm packages |
| **Deterministic execution** | Yes | No (event-loop ordering) |
| **n8n workflow JSON** | Runs unchanged | Native |
| **MCP server for Claude Code** | Built in (37 tools) | Not available |
| **License** | MIT | Sustainable Use License |

[Full comparison and benchmark methodology →](why-m9m.md)

## 30-second quickstart

```bash
# 1. Install
curl -fsSL https://raw.githubusercontent.com/neul-labs/m9m/main/install.sh | bash

# 2. Start the server
m9m serve

# 3. Run the bundled demo
m9m demo
```

The server starts at `http://localhost:8080`:

- **Web UI** — `http://localhost:8080`
- **API** — `http://localhost:8080/api/v1`
- **Health check** — `http://localhost:8080/health`

[Full installation guide →](getting-started/installation.md)

## Run your first workflow

```bash
cat > hello-world.json << 'EOF'
{
  "name": "Hello World",
  "nodes": [
    {
      "id": "start",
      "name": "Start",
      "type": "n8n-nodes-base.start",
      "position": [250, 300],
      "parameters": {}
    },
    {
      "id": "set",
      "name": "Set Message",
      "type": "n8n-nodes-base.set",
      "position": [450, 300],
      "parameters": {
        "assignments": [
          {"name": "message", "value": "Hello from m9m!"}
        ]
      }
    }
  ],
  "connections": {
    "Start": {
      "main": [[{"node": "Set Message", "type": "main", "index": 0}]]
    }
  }
}
EOF

m9m exec hello-world.json
```

[Step-by-step walkthrough →](getting-started/first-workflow.md)

## Key features

- **Drop-in n8n compatibility** — runs n8n workflow JSON, expression syntax, and credentials unchanged.
- **40+ built-in nodes** — HTTP, databases, AI/LLM, cloud storage, messaging, CLI execution, scheduling, more.
- **MCP server for AI agents** — 37 tools for Claude Code, Cursor, and other MCP clients. [Learn more](https://github.com/neul-labs/m9m/blob/main/docs/mcp/README.md).
- **CLI agent sandboxing** — run Claude Code, Codex, Aider in bubblewrap-isolated environments.
- **Expression engine** — full n8n expression syntax (`{{ $json.field }}`, `{{ $node["x"].data }}`).
- **Storage backends** — SQLite, PostgreSQL, or in-memory.
- **Job queue** — persistent (SQLite), in-memory, or external (Redis / RabbitMQ).
- **Observability** — Prometheus metrics + OpenTelemetry tracing built in.
- **REST API** — n8n-compatible surface for workflow management.

## Documentation

<div class="grid cards" markdown>

-   :material-rocket-launch: **[Getting Started](getting-started/index.md)**

    ---

    Install m9m, run your first workflow, learn core concepts.

-   :material-swap-horizontal: **[Migrate from n8n](migrate-from-n8n.md)**

    ---

    One command to run existing n8n workflows. What's compatible, what's not.

-   :material-help-circle: **[FAQ](faq.md)**

    ---

    Common questions about performance, compatibility, and reliability.

-   :material-trending-up: **[Why m9m?](why-m9m.md)**

    ---

    Detailed comparison vs n8n, benchmarks, and the reliability story.

-   :material-cog: **[Configuration](configuration/index.md)**

    ---

    Server, database, queue, security, environment variables.

-   :material-console: **[CLI Reference](cli/index.md)**

    ---

    Complete command-line interface documentation.

-   :material-api: **[API Reference](api/index.md)**

    ---

    REST API endpoints for workflow management.

-   :material-cube-outline: **[Nodes](nodes/index.md)**

    ---

    All 40+ node types — parameters, examples, output schemas.

-   :material-cloud-upload: **[Deployment](deployment/index.md)**

    ---

    Single-binary, Docker, Kubernetes, production.

</div>

## License

m9m is open source software licensed under the [MIT License](https://github.com/neul-labs/m9m/blob/main/LICENSE).

## Community

- **GitHub Issues** — [report bugs, request features](https://github.com/neul-labs/m9m/issues)
- **GitHub Discussions** — [questions, design proposals](https://github.com/neul-labs/m9m/discussions)
- **Release Notes** — [changelog](https://github.com/neul-labs/m9m/releases)
