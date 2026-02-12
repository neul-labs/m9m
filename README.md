# m9m

**Workflow Automation at Ludicrous Speed**

A blazing-fast, cloud-native workflow engine built in Go. 5-10x faster than Node.js alternatives.

[![Build Status](https://img.shields.io/github/actions/workflow/status/neul-labs/m9m/ci.yml?branch=main&style=flat-square&logo=github)](https://github.com/neul-labs/m9m/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/neul-labs/m9m?style=flat-square)](https://goreportcard.com/report/github.com/neul-labs/m9m)
[![Coverage](https://img.shields.io/codecov/c/github/neul-labs/m9m?style=flat-square&logo=codecov)](https://codecov.io/gh/neul-labs/m9m)
[![Go Reference](https://img.shields.io/badge/go.dev-reference-007d9c?style=flat-square&logo=go)](https://pkg.go.dev/github.com/neul-labs/m9m)
[![Release](https://img.shields.io/github/v/release/neul-labs/m9m?style=flat-square&logo=github)](https://github.com/neul-labs/m9m/releases)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square)](https://opensource.org/licenses/Apache-2.0)

---

## Why m9m?

| Metric | Node.js Alternatives | m9m | Improvement |
|--------|---------------------|-----|-------------|
| **Workflow Execution** | 500ms | 100ms | **5x faster** |
| **Memory Usage** | 512MB | 150MB | **70% less** |
| **Cold Start** | 3s | 500ms | **6x faster** |
| **Concurrent Workflows** | 50 | 500 | **10x more** |
| **Container Size** | 1.2GB | 300MB | **75% smaller** |

---

## Features

### Performance & Architecture
- **5-10x faster execution** with Go's compiled efficiency
- **70% lower memory footprint** - run more workflows on the same hardware
- **Sub-second cold starts** - perfect for serverless and edge deployments
- **Horizontal scaling** with Redis, RabbitMQ, or in-memory queues

### CLI Agent Orchestration
- **Run AI coding agents** (Claude Code, Codex, Aider) in secure sandboxes
- **Linux sandboxing** with bubblewrap - namespace isolation, cgroups, seccomp
- **Streaming and one-shot execution** modes for interactive CLI tools
- **Resource limits** - memory, CPU, timeout enforcement

```bash
# List CLI execution nodes
m9m node list --category cli

# Search for AI agent support
m9m node list --search "claude"
```

### Enterprise Ready
- **OpenTelemetry tracing** with Jaeger integration
- **Prometheus metrics** for comprehensive monitoring
- **Git-based workflow versioning** with branch management
- **OAuth2, API keys, and service accounts** for authentication

### 30+ Integrations
- **Databases**: PostgreSQL, MySQL, SQLite
- **Cloud**: AWS (S3, Lambda), GCP, Azure
- **AI/LLM**: OpenAI, Anthropic Claude
- **CLI Agents**: Claude Code, Codex, Aider (sandboxed execution)
- **Messaging**: Slack, Discord
- **VCS**: GitHub, GitLab

### Claude Code Integration (MCP)

m9m includes a built-in MCP server for AI-powered workflow orchestration:

```json
{
  "mcpServers": {
    "m9m": {
      "command": "/path/to/mcp-server",
      "args": ["--data", "./data"]
    }
  }
}
```

**37 MCP tools** for natural language workflow management:
- Create and execute workflows conversationally
- Debug executions with detailed logs
- Build custom nodes with JavaScript

---

## Quick Start

### Docker (Recommended)

```bash
docker run -p 8080:8080 neul-labs/m9m:latest
```

### Binary

```bash
# Download latest release
curl -fsSL https://github.com/neul-labs/m9m/releases/latest/download/m9m-linux-amd64 -o m9m
chmod +x m9m
./m9m serve
```

### From Source

```bash
git clone https://github.com/neul-labs/m9m.git
cd m9m
make build
./m9m serve
```

### Execute a Workflow

```bash
# Run a workflow file
m9m run workflow.json

# Start with monitoring
m9m serve --metrics-port 9090

# List available nodes
m9m node list
m9m node categories
```

---

## Running CLI AI Agents

Execute AI coding assistants like Claude Code, Codex, or Aider within workflows:

```json
{
  "name": "Run Claude Code",
  "type": "n8n-nodes-base.cliExecute",
  "parameters": {
    "command": "claude",
    "args": ["--print", "--prompt", "Analyze this codebase"],
    "sandboxEnabled": true,
    "isolationLevel": "standard",
    "networkAccess": "host",
    "timeout": 300,
    "maxMemoryMB": 2048,
    "additionalMounts": [
      {"source": "/home/user/project", "destination": "/workspace", "readWrite": false}
    ]
  }
}
```

**Supported isolation levels:**
- `none` - No isolation (development only)
- `minimal` - Basic filesystem isolation
- `standard` - PID namespace + resource limits (recommended)
- `strict` - Network isolation + seccomp filtering
- `paranoid` - No host filesystem access

**Prerequisites (Linux):**
```bash
# Install bubblewrap for sandboxing
sudo apt install bubblewrap  # Debian/Ubuntu
sudo dnf install bubblewrap  # Fedora/RHEL
```

---

## SDK & Language Bindings

Embed m9m directly in your applications:

**Go:**
```go
import "github.com/neul-labs/m9m/pkg/m9m"

engine := m9m.New()
result, _ := engine.Execute(workflow, inputData)
```

**Python:**
```python
from m9m import WorkflowEngine

engine = WorkflowEngine()
result = engine.execute(workflow, input_data)
```

**Node.js:**
```typescript
import { WorkflowEngine } from '@m9m/workflow-engine';

const engine = new WorkflowEngine();
const result = await engine.execute(workflow);
```

---

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web UI        │    │   REST API      │    │   CLI Interface │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │
         ┌─────────────────────────────────────────────────────┐
         │              Workflow Engine                        │
         └─────────────────────┬───────────────────────────────┘
                               │
         ┌─────────────────────────────────────────────────────┐
         │                Node Registry                        │
         ├─────────────┬─────────────┬─────────────┬───────────┤
         │ Messaging   │ Databases   │ CLI Agents  │    AI     │
         └─────────────┴─────────────┴─────────────┴───────────┘
                               │
         ┌─────────────────────────────────────────────────────┐
         │              Queue System                           │
         ├─────────────┬─────────────┬─────────────────────────┤
         │   Memory    │    Redis    │      RabbitMQ           │
         └─────────────┴─────────────┴─────────────────────────┘
```

---

## Workflow Compatibility

m9m provides **95% compatibility** with n8n workflow definitions. Import and run your workflows with minimal changes:

```bash
# Import existing workflow
m9m import workflow.json

# Validate compatibility
m9m validate workflow.json
```

---

## Configuration

**Zero dependencies by default** - runs with SQLite + in-memory queue out of the box.

```yaml
# config.yaml
server:
  port: 8080
  host: "0.0.0.0"

database:
  type: "sqlite"  # or postgres, mysql

queue:
  type: "memory"  # or redis, rabbitmq
  max_workers: 10
```

For production, switch to Redis/PostgreSQL:
```bash
M9M_DATABASE_TYPE=postgres
M9M_DATABASE_URL=postgres://user:pass@localhost/m9m
M9M_QUEUE_TYPE=redis
M9M_QUEUE_URL=redis://localhost:6379
```

---

## Custom Node Development

```go
type MyNode struct {
    *base.BaseNode
}

func (n *MyNode) Execute(inputData []model.DataItem, params map[string]interface{}) ([]model.DataItem, error) {
    message := params["message"].(string)
    return []model.DataItem{
        {JSON: map[string]interface{}{"result": "Processed: " + message}},
    }, nil
}
```

---

## Monitoring & Observability

- **Prometheus metrics** at `/metrics`
- **OpenTelemetry tracing** with Jaeger/Zipkin
- **Health endpoints**: `/health`, `/ready`

```bash
# Check health
curl http://localhost:8080/health

# View metrics
curl http://localhost:9090/metrics
```

---

## Documentation

| Resource | Description |
|----------|-------------|
| [Getting Started](docs/README.md) | Quick start guide |
| [Architecture](docs/architecture/README.md) | System design |
| [API Reference](docs/api/API_COMPATIBILITY.md) | REST API documentation |
| [Node Development](docs/nodes/README.md) | Building custom nodes |
| [CLI Nodes](documentation/docs/nodes/cli.md) | CLI agent orchestration |
| [Deployment](docs/deployment/DEPLOYMENT_GUIDE.md) | Production deployment |
| [MCP Integration](docs/mcp/README.md) | Claude Code integration |

---

## Contributing

We welcome contributions! See our [Contributing Guide](docs/CONTRIBUTING.md) for details.

```bash
git clone https://github.com/neul-labs/m9m.git
cd m9m
make deps
make test
make build
```

---

## License

m9m is released under the [Apache 2.0 License](LICENSE).

---

**Built with Go for performance-obsessed engineers**

[GitHub](https://github.com/neul-labs/m9m) | [Documentation](https://docs.neullabs.com/m9m)
