# Getting Started

Welcome to m9m! This guide will help you get up and running quickly.

## What is m9m?

m9m is a high-performance workflow automation platform that provides:

- **n8n Compatibility** - Run existing n8n workflows without modification
- **Superior Performance** - 5-10x faster execution than n8n
- **Lower Resource Usage** - 70% less memory, 75% smaller containers
- **Cloud Native** - Built for modern infrastructure

## Quick Start

### Option 1: Binary Installation

```bash
# Install with Go
go install github.com/neul-labs/m9m/cmd/m9m@latest

# Start the server
m9m serve
```

### Option 2: Docker

```bash
# Run with Docker
docker run -p 8080:8080 ghcr.io/neul-labs/m9m:latest
```

### Option 3: From Source

```bash
# Clone the repository
git clone https://github.com/neul-labs/m9m.git
cd m9m

# Build
make build

# Run
./m9m serve
```

## Verify Installation

Once running, verify m9m is working:

```bash
# Check health endpoint
curl http://localhost:8080/health

# Expected response:
# {"status":"ok"}
```

## Access Points

After starting m9m, you can access:

| Service | URL | Description |
|---------|-----|-------------|
| Web UI | `http://localhost:8080` | Visual workflow editor |
| REST API | `http://localhost:8080/api/v1` | Programmatic access |
| Health | `http://localhost:8080/health` | Health check endpoint |
| Metrics | `http://localhost:8080/metrics` | Prometheus metrics |

## Next Steps

1. **[Installation](installation.md)** - Detailed installation options
2. **[First Workflow](first-workflow.md)** - Create your first workflow
3. **[Core Concepts](concepts.md)** - Understand workflows, nodes, and data flow
4. **[Agent Usage Guide](agent-usage.md)** - Use m9m from AI agents via MCP or REST API
