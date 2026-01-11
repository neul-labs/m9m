# m9m: High-Performance Workflow Automation Platform

m9m is a high-performance, cloud-native workflow automation platform built in Go. It provides a scalable alternative to n8n with enhanced performance, embedded Python runtime, and enterprise-grade features.

## Key Features

### Performance & Architecture
- **5-10x faster execution** compared to Node.js-based alternatives
- **70% lower memory usage** with efficient Go runtime
- **Cloud-native architecture** designed for Kubernetes and containerized environments
- **Embedded Python runtime** - no external Python installation required
- **Horizontal scaling** with Redis, RabbitMQ, and in-memory queue backends

### Enterprise Capabilities
- **Distributed tracing** with OpenTelemetry and Jaeger integration
- **Prometheus metrics** for comprehensive monitoring
- **Git-based workflow versioning** with branch management
- **Queue-based execution** for high-throughput scenarios
- **Multiple authentication methods** including OAuth2, API keys, and service accounts

### Extensive Integrations
- **100+ supported services** through unified LLM interface
- **Advanced database support** including MongoDB, Redis, PostgreSQL, MySQL, Elasticsearch
- **Modern messaging platforms** including Slack, Discord, Telegram, Microsoft Teams
- **Version control systems** with GitHub and GitLab integration
- **Cloud platforms** with AWS, GCP, and Azure native support
- **Productivity tools** including Google Sheets and Microsoft 365

## Claude Code Integration (MCP)

m9m includes a built-in MCP (Model Context Protocol) server that enables Claude Code to orchestrate workflows conversationally. This allows you to create, execute, and manage workflows using natural language.

### Quick Start with Claude Code

```bash
# Build the MCP server
go build -o mcp-server ./cmd/mcp-server

# Add to Claude Code MCP settings (~/.claude/claude_desktop_config.json)
{
  "mcpServers": {
    "m9m": {
      "command": "/path/to/mcp-server",
      "args": ["--data", "./data"]
    }
  }
}
```

### MCP Server Modes

| Mode | Command | Description |
|------|---------|-------------|
| **Local + SQLite** | `./mcp-server` | Default, persists to `./data/m9m.db` |
| **Local + Postgres** | `./mcp-server --postgres "postgres://..."` | Production local setup |
| **Cloud** | `./mcp-server --api-url https://m9m.example.com` | Connect to remote m9m |

### Available MCP Tools (37 tools)

| Category | Tools | Description |
|----------|-------|-------------|
| **Node Discovery** | 4 | List/search available node types |
| **Quick Actions** | 6 | `http_request`, `send_slack`, `ai_openai`, etc. |
| **Workflow Management** | 9 | Create, update, delete, activate workflows |
| **Execution** | 7 | Run, monitor, cancel, retry workflows |
| **Debugging** | 5 | Execution logs, node outputs, performance |
| **Plugins** | 6 | Create JavaScript/REST custom nodes |

### Example Conversations

```
You: "Send a Slack message to #alerts saying 'Deployment complete'"
Claude: [Uses send_slack tool to send the message]

You: "Create a workflow that checks our API health every 5 minutes"
Claude: [Creates workflow with HTTP Request + Slack nodes, schedules it]

You: "My last workflow failed, what went wrong?"
Claude: [Gets execution logs and explains the error]

You: "Create a custom node that formats timestamps"
Claude: [Creates JavaScript plugin node using Goja runtime]
```

See [MCP Documentation](docs/mcp/README.md) for detailed usage.

## Quick Start

### Installation

#### Docker (Recommended)
```bash
docker run -p 8080:8080 m9m/m9m:latest
```

#### Binary Release
```bash
# Download latest release
wget https://github.com/m9m/m9m/releases/latest/download/m9m-linux-amd64
chmod +x m9m-linux-amd64
./m9m-linux-amd64 execute workflow.json
```

#### From Source
```bash
git clone https://github.com/m9m/m9m.git
cd m9m
go build -o m9m cmd/m9m/main.go
./m9m execute workflow.json
```

### Basic Usage

Execute a workflow:
```bash
m9m execute my-workflow.json
```

Start with monitoring:
```bash
m9m serve --metrics-port 9090 --queue redis://localhost:6379
```

## Architecture

m9m implements a modular, plugin-based architecture:

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
         │ Messaging   │ Databases   │ Cloud Ops   │    AI     │
         │ Nodes       │ Nodes       │ Nodes       │  Nodes    │
         └─────────────┴─────────────┴─────────────┴───────────┘
                               │
         ┌─────────────────────────────────────────────────────┐
         │              Queue System                           │
         ├─────────────┬─────────────┬─────────────────────────┤
         │   Memory    │    Redis    │      RabbitMQ           │
         │   Queue     │    Queue    │       Queue             │
         └─────────────┴─────────────┴─────────────────────────┘
```

## Configuration

### Environment Variables
```bash
# Server Configuration
M9M_PORT=8080
M9M_HOST=0.0.0.0

# Queue Configuration
M9M_QUEUE_TYPE=redis
M9M_QUEUE_URL=redis://localhost:6379
M9M_MAX_WORKERS=10

# Monitoring
M9M_METRICS_PORT=9090
M9M_TRACING_ENDPOINT=http://localhost:14268/api/traces

# Database
M9M_DB_TYPE=postgresql
M9M_DB_URL=postgres://user:pass@localhost/n8n_go
```

### Configuration File
```yaml
# config.yaml
server:
  port: 8080
  host: "0.0.0.0"

queue:
  type: "redis"
  url: "redis://localhost:6379"
  max_workers: 10

monitoring:
  metrics_port: 9090
  tracing:
    endpoint: "http://localhost:14268/api/traces"
    service_name: "m9m"

database:
  type: "postgresql"
  url: "postgres://user:pass@localhost/n8n_go"
```

## Node Development

### Creating Custom Nodes

```go
package custom

import (
    "context"
    "github.com/yourusername/m9m/internal/interfaces"
    "github.com/yourusername/m9m/internal/nodes/base"
)

type MyCustomNode struct {
    *base.BaseNode
}

func NewMyCustomNode() interfaces.Node {
    return &MyCustomNode{
        BaseNode: base.NewBaseNode("MyCustom"),
    }
}

func (n *MyCustomNode) GetMetadata() interfaces.NodeMetadata {
    return interfaces.NodeMetadata{
        Name:        "My Custom Node",
        Version:     "1.0.0",
        Description: "Custom node implementation",
        Category:    "Custom",
        Properties: []interfaces.NodeProperty{
            {
                Name:        "message",
                Type:        "string",
                DisplayName: "Message",
                Required:    true,
            },
        },
    }
}

func (n *MyCustomNode) Execute(ctx context.Context, params interfaces.ExecutionParams) (interfaces.NodeOutput, error) {
    message := params.GetString("message")

    return &base.NodeOutput{
        Data: map[string]interface{}{
            "result": "Processed: " + message,
        },
    }, nil
}
```

## SDK & Language Bindings

m9m can be embedded as a library in your applications. We provide native bindings for Go, Python, and Node.js.

### Go SDK

Import m9m directly into your Go applications:

```go
import "github.com/m9m/m9m/pkg/m9m"

func main() {
    // Create engine
    engine := m9m.New()

    // Load workflow
    workflow, _ := m9m.LoadWorkflow("workflow.json")

    // Execute
    result, err := engine.Execute(workflow, []m9m.DataItem{
        {JSON: map[string]interface{}{"input": "data"}},
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(result.Data)
}
```

### Python Bindings

Install and use m9m in Python applications:

```bash
# Install with uv
cd bindings/python
uv pip install -e .

# Or build the shared library first
make cgo-lib
```

```python
from m9m import WorkflowEngine, Workflow

# Create engine
engine = WorkflowEngine()

# Load and execute workflow
workflow = Workflow.from_file("workflow.json")
result = engine.execute(workflow, [{"json": {"input": "data"}}])
print(result.data)

# Register custom nodes with decorators
@engine.node("custom.myNode", name="My Node", category="transform")
def my_node(input_data, params):
    return [{"json": {"result": "processed"}}]
```

### Node.js Bindings

Use m9m in Node.js/TypeScript applications:

```bash
# Install dependencies and build
cd bindings/nodejs
npm install
npm run build
```

```typescript
import { WorkflowEngine, Workflow } from '@m9m/workflow-engine';

const engine = new WorkflowEngine();

// Load workflow from JSON
const workflow = Workflow.fromJSON({
  name: 'My Workflow',
  nodes: [],
  connections: {}
});

// Execute workflow
const result = await engine.execute(workflow, [{ json: { input: 'data' } }]);
console.log(result.data);
```

### Building Native Libraries

```bash
# Build CGO shared library for all platforms
make cgo-lib          # Linux (.so)
make cgo-lib-darwin   # macOS (.dylib)
make cgo-lib-windows  # Windows (.dll)

# Build Python bindings
make python-bindings

# Build Node.js bindings
make nodejs-bindings
```

See [SDK Documentation](docs/sdk/README.md) for detailed API reference and examples.

## Performance Benchmarks

| Metric | n8n | m9m | Improvement |
|--------|-----|--------|-------------|
| Workflow Execution | 500ms | 100ms | 5x faster |
| Memory Usage | 512MB | 150MB | 70% reduction |
| Cold Start Time | 3s | 500ms | 6x faster |
| Concurrent Workflows | 50 | 500 | 10x scale |
| Container Size | 1.2GB | 300MB | 75% smaller |

## Production Deployment

### Kubernetes
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: m9m
spec:
  replicas: 3
  selector:
    matchLabels:
      app: m9m
  template:
    metadata:
      labels:
        app: m9m
    spec:
      containers:
      - name: m9m
        image: m9m/m9m:latest
        ports:
        - containerPort: 8080
        - containerPort: 9090
        env:
        - name: M9M_QUEUE_TYPE
          value: "redis"
        - name: M9M_QUEUE_URL
          value: "redis://redis-service:6379"
```

### Docker Compose
```yaml
version: '3.8'
services:
  m9m:
    image: m9m/m9m:latest
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      - M9M_QUEUE_TYPE=redis
      - M9M_QUEUE_URL=redis://redis:6379
    depends_on:
      - redis
      - postgres

  redis:
    image: redis:7-alpine

  postgres:
    image: postgres:15
    environment:
      - POSTGRES_DB=n8n_go
      - POSTGRES_USER=n8n_go
      - POSTGRES_PASSWORD=password
```

## Monitoring & Observability

### Prometheus Metrics
- Workflow execution metrics
- Node performance metrics
- Queue size and processing rates
- System resource utilization
- HTTP request metrics

### OpenTelemetry Tracing
- End-to-end workflow tracing
- Node execution spans
- Database query tracing
- HTTP request tracing

### Health Endpoints
```bash
# Health check
curl http://localhost:8080/health

# Readiness check
curl http://localhost:8080/ready

# Metrics endpoint
curl http://localhost:9090/metrics
```

## Migration from n8n

### Workflow Compatibility
m9m provides 95% compatibility with existing n8n workflows:

```bash
# Export from n8n
curl -X GET http://n8n:5678/api/v1/workflows/export

# Import to m9m
m9m import workflow.json
```

### Credential Migration
```bash
# Export credentials (encrypted)
m9m migrate-credentials --from-n8n http://n8n:5678 --credentials-key your-key
```

## Contributing

We welcome contributions from the community. Please see our [Contributing Guide](docs/CONTRIBUTING.md) for details.

### Development Setup
```bash
git clone https://github.com/m9m/m9m.git
cd m9m
go mod download
make test
make build
```

## Documentation

- [Documentation Index](docs/README.md)
- [MCP / Claude Code Integration](docs/mcp/README.md)
- [Architecture](docs/architecture/README.md)
- [API Reference](docs/api/API_COMPATIBILITY.md)
- [SDK & Bindings](docs/sdk/README.md)
- [Deployment Guide](docs/deployment/DEPLOYMENT_GUIDE.md)
- [Node Development](docs/nodes/README.md)
- [Migration Guide](docs/migration/from-n8n.md)
- [Contributing](docs/CONTRIBUTING.md)

## License

m9m is released under the Apache 2.0 License. See [LICENSE](LICENSE) for details.

## Support

- **Enterprise Support**: Contact enterprise@m9m.com
- **Community Support**: GitHub Issues and Discussions
- **Documentation**: https://docs.m9m.com
- **Slack Community**: https://slack.m9m.com

## Roadmap

See our [Project Roadmap](docs/roadmap.md) for planned features and improvements.