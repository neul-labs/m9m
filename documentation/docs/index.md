# m9m Documentation

## Workflow Automation at Ludicrous Speed

m9m is a high-performance workflow automation platform built in Go. It provides **95% backend feature parity with n8n** while delivering **5-10x faster execution** and **70% lower memory usage**.

## Why m9m?

| Metric | m9m | n8n | Improvement |
|--------|-----|-----|-------------|
| Execution Speed | ~100ms avg | ~500ms avg | **5-10x faster** |
| Memory Usage | ~150MB | ~512MB | **70% lower** |
| Container Size | ~300MB | ~1.2GB | **75% smaller** |
| Startup Time | ~500ms | ~3s | **6x faster** |

## Key Features

- **n8n Compatible** - Run existing n8n workflows without modification
- **CLI Agent Orchestration** - Run AI coding agents (Claude Code, Codex, Aider) in secure sandboxes
- **35+ Node Types** - HTTP, databases, AI/LLM, cloud storage, messaging, CLI execution, and more
- **Linux Sandboxing** - Bubblewrap-based isolation for secure command execution
- **Expression Engine** - Full n8n expression syntax support
- **Multiple Storage Backends** - SQLite, PostgreSQL, or in-memory
- **Job Queue** - Persistent job queue with SQLite or in-memory options
- **REST API** - Complete API for workflow management
- **CLI Tool** - Powerful command-line interface
- **Cloud Native** - Built for containers and bare-metal deployment

## Quick Start

```bash
# Install m9m (official single-binary path)
curl -fsSL https://raw.githubusercontent.com/neul-labs/m9m/main/install.sh | bash

# Start the server
m9m serve

# Or run with Docker (optional)
docker run -p 8080:8080 ghcr.io/neul-labs/m9m:latest
```

The server starts at `http://localhost:8080` with:

- **Web UI**: `http://localhost:8080`
- **API**: `http://localhost:8080/api/v1`
- **Health**: `http://localhost:8080/health`

## Create Your First Workflow

```bash
# Create a simple workflow
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

# Run the workflow
m9m run hello-world.json
```

## Documentation Sections

<div class="grid cards" markdown>

-   :material-rocket-launch: **[Getting Started](getting-started/index.md)**

    ---

    Install m9m, create your first workflow, and learn core concepts

-   :material-cog: **[Configuration](configuration/index.md)**

    ---

    Configure server, database, queue, and security settings

-   :material-console: **[CLI Reference](cli/index.md)**

    ---

    Complete command-line interface documentation

-   :material-api: **[API Reference](api/index.md)**

    ---

    REST API endpoints for workflow management

-   :material-cube-outline: **[Nodes](nodes/index.md)**

    ---

    Documentation for all 34+ available node types

-   :material-sitemap: **[Workflows](workflows/index.md)**

    ---

    Learn how to create, execute, and manage workflows

-   :material-function: **[Expressions](expressions/index.md)**

    ---

    Expression syntax, variables, and built-in functions

-   :material-cloud-upload: **[Deployment](deployment/index.md)**

    ---

    Deploy with single binaries, Docker, or bare metal

</div>

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                         m9m Server                          │
├─────────────────────────────────────────────────────────────┤
│  REST API  │  Web UI  │  Webhooks  │  Scheduler            │
├─────────────────────────────────────────────────────────────┤
│                    Workflow Engine                          │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐       │
│  │  Nodes  │  │ Express │  │ Connect │  │ Credent │       │
│  │ Registry│  │  Eval   │  │ Router  │  │ Manager │       │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘       │
├─────────────────────────────────────────────────────────────┤
│              Job Queue (Memory / SQLite)                    │
├─────────────────────────────────────────────────────────────┤
│           Storage (SQLite / PostgreSQL)                     │
└─────────────────────────────────────────────────────────────┘
```

## License

m9m is open source software licensed under the [MIT License](https://github.com/neul-labs/m9m/blob/main/LICENSE).

## Support

- **GitHub Issues**: [Report bugs and request features](https://github.com/neul-labs/m9m/issues)
- **Documentation**: [docs.neullabs.com/m9m](https://docs.neullabs.com/m9m)
