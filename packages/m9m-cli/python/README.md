# m9m-cli

> **High-performance workflow automation CLI and SDK for Python** — 5–10x faster than n8n, 70% lower memory, sub-second startup. Execute n8n-compatible workflows programmatically from your Python applications.

[![PyPI version](https://img.shields.io/pypi/v/m9m-cli.svg)](https://pypi.org/project/m9m-cli/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/neul-labs/m9m/blob/main/LICENSE)
[![Python versions](https://img.shields.io/pypi/pyversions/m9m-cli.svg)](https://pypi.org/project/m9m-cli/)
[![GitHub release](https://img.shields.io/github/v/release/neul-labs/m9m)](https://github.com/neul-labs/m9m/releases)

## What is m9m?

**m9m** is a cloud-native workflow automation engine built in Go that delivers **95% feature parity with n8n** while running **5–10x faster** and using **~70% less memory** (~150 MB vs 512 MB). It supports the full n8n workflow JSON format, expression syntax, and node ecosystem — making it a drop-in replacement for teams that need enterprise-grade performance without the overhead.

The `m9m-cli` PyPI package embeds the engine directly into your Python application. On first use, it automatically downloads the correct platform-native binary (macOS, Linux, or Windows; AMD64 and ARM64) from GitHub Releases and caches it locally. No Docker, no Kubernetes, no separate server process required.

## Key Features

- **n8n-Compatible Workflows** — Load and execute workflows exported from n8n without modification
- **Pythonic API** — Clean, intuitive API for workflow execution with full type hints
- **Zero Configuration** — Binary auto-downloads on first use for your OS/architecture
- **Lightweight** — ~15 MB binary, ~150 MB runtime memory footprint
- **Cross-Platform** — Native support for macOS (Intel & Apple Silicon), Linux (AMD64 & ARM64), and Windows
- **Custom Node Registration** — Extend the engine with Python node implementations
- **Context Manager Support** — Use `with` statements for safe resource handling
- **Credential Manager** — Securely manage API keys and credentials within your application

## Installation

```bash
pip install m9m-cli
```

Or with your favorite Python package manager:

```bash
uv add m9m-cli
poetry add m9m-cli
conda install -c conda-forge m9m-cli  # (community)
```

### Manual Binary Download

If the automatic download fails (e.g., behind a corporate proxy), you can download the binary manually:

```bash
# macOS Apple Silicon
curl -L -o ~/.m9m/bin/m9m https://github.com/neul-labs/m9m/releases/latest/download/m9m-darwin-arm64
chmod +x ~/.m9m/bin/m9m

# Linux AMD64
curl -L -o ~/.m9m/bin/m9m https://github.com/neul-labs/m9m/releases/latest/download/m9m-linux-amd64
chmod +x ~/.m9m/bin/m9m
```

## Quick Start

### Execute a Workflow

```python
from m9m import WorkflowEngine, Workflow

# Create an engine
engine = WorkflowEngine()

# Load a workflow from an n8n-compatible JSON file
workflow = Workflow.from_file("./my-workflow.json")

# Execute with input data
result = engine.execute(workflow, [
    {"json": {"email": "user@example.com", "name": "Alice"}}
])

print("Output:", result.data)
if result.error:
    print("Execution failed:", result.error)
```

### Create a Workflow Programmatically

```python
from m9m import WorkflowEngine, Workflow, create_workflow

workflow = create_workflow({
    "name": "Email Notification",
    "nodes": [
        {
            "name": "Start",
            "type": "n8n-nodes-base.start",
            "parameters": {}
        },
        {
            "name": "Send Email",
            "type": "n8n-nodes-base.emailSend",
            "parameters": {
                "to": "={{ $json.email }}",
                "subject": "Welcome!",
                "text": "Hello {{ $json.name }}"
            }
        }
    ],
    "connections": {
        "Start": {
            "main": [[{"node": "Send Email", "type": "main", "index": 0}]]
        }
    }
})

engine = WorkflowEngine()
result = engine.execute(workflow)
```

### Register Custom Nodes

```python
from m9m import WorkflowEngine

engine = WorkflowEngine()

# Register a custom transformation node
@engine.node("custom.uppercase", name="Uppercase")
def uppercase_node(input_data, params):
    return [
        {"json": {"text": item["json"]["text"].upper()}}
        for item in input_data
    ]

# Or register directly
def my_node(input_data, params):
    return input_data

engine.register_node("custom.myNode", my_node)
```

### Use as a Context Manager

```python
from m9m import WorkflowEngine, Workflow

with WorkflowEngine() as engine:
    workflow = Workflow.from_file("workflow.json")
    result = engine.execute(workflow)
    print(result.data)
```

## API Reference

### `WorkflowEngine`

The main entry point for executing workflows.

```python
from m9m import WorkflowEngine, CredentialManager

# Basic engine
engine = WorkflowEngine()

# Engine with credentials
cred_manager = CredentialManager()
engine = WorkflowEngine(credential_manager=cred_manager)

# Execute workflow
result = engine.execute(workflow, input_data)

# Register custom node
engine.register_node("custom.myNode", handler)

# Decorator registration
@engine.node("custom.myNode", name="My Node")
def my_handler(input_data, params):
    return input_data
```

### `Workflow`

Represents a workflow definition.

```python
from m9m import Workflow

# From file
workflow = Workflow.from_file("path/to/workflow.json")

# From JSON string
workflow = Workflow.from_json('{"name": "test", "nodes": [], "connections": {}}')

# From dict
workflow = Workflow.from_dict({
    "name": "My Workflow",
    "nodes": [],
    "connections": {}
})

# Access properties
print(workflow.name)
print(workflow.id)

# Serialize
data = workflow.to_dict()
json_str = workflow.to_json()
```

### `DataItem`

The standard data structure passed between nodes.

```python
from m9m import DataItem

# Create from dict
item = DataItem(json={"key": "value"})

# Convert from dict representation
item = DataItem.from_dict({"json": {"key": "value"}})
```

### `ExecutionResult`

Returned by `engine.execute()`.

```python
from m9m import ExecutionResult

result.data      # List[DataItem]
result.error     # Optional error message
result.success   # Boolean

# Parse from JSON string
result = ExecutionResult.from_json(json_string)
```

### `CredentialManager`

Securely store and retrieve credentials.

```python
from m9m import CredentialManager

cred_mgr = CredentialManager()
cred_mgr.store({
    "id": "api-key-1",
    "name": "My API Key",
    "type": "apiKey",
    "data": {"apiKey": "secret123"}
})
```

### Utility Functions

```python
from m9m import version, get_binary_path, download_binary

# Get the m9m binary version
print(version())  # "0.2.1"

# Manually download the binary for a specific version
path = download_binary("0.2.1")

# Get the path to the cached binary
binary_path = get_binary_path()
```

## Supported Platforms

| OS | Architecture | Status |
|---|---|---|
| macOS | Intel (AMD64) | ✅ Supported |
| macOS | Apple Silicon (ARM64) | ✅ Supported |
| Linux | AMD64 | ✅ Supported |
| Linux | ARM64 | ✅ Supported |
| Windows | AMD64 | ✅ Supported |

The correct binary is automatically selected and downloaded on first use based on your operating system and CPU architecture.

## Supported Node Types

m9m supports 40+ built-in node types including:

- **Core**: `start`, `noOp`, `wait`, `executeWorkflow`
- **Transform**: `set`, `filter`, `code`, `function`, `merge`, `json`, `if`, `loop`
- **HTTP**: `httpRequest`
- **Trigger**: `webhook`, `errorTrigger`
- **Timer**: `cron`
- **Messaging**: `slack`, `discord`, `twilio`, `teams`
- **Database**: `postgres`, `mysql`, `sqlite`, `elasticsearch`, `redis`, `mongodb`
- **Email**: `emailSend`, `sendGrid`
- **AI**: `openAi`, `anthropic`
- **VCS**: `github`, `gitlab`
- **Cloud**: `aws`, `gcp`, `azure`
- **Productivity**: `notion`, `stripe`, `googleSheets`

## Performance

| Metric | m9m | n8n | Improvement |
|---|---|---|---|
| Startup Time | < 500 ms | ~3 s | **6x faster** |
| Memory Usage | ~150 MB | ~512 MB | **70% lower** |
| Execution Speed | Baseline | 5–10x slower | **5–10x faster** |
| Container Size | ~300 MB | ~1.2 GB | **75% smaller** |

## Why m9m?

- **Go-Powered Performance** — Built in Go for maximum concurrency and minimal resource usage
- **n8n Compatibility** — Import existing n8n workflows without changes
- **Cloud-Native** — Stateless architecture ready for Kubernetes, Docker, and serverless
- **Enterprise Observability** — Built-in Prometheus metrics and OpenTelemetry tracing
- **No Node.js Runtime Required** — The engine is a single static binary; no npm install on the server
- **Python-First Design** — Native Python API with type hints, context managers, and decorator support

## Type Hints

The entire `m9m-cli` package is fully typed and includes a `py.typed` marker file. This means your type checker (mypy, pyright, etc.) will have full autocomplete and type checking support:

```python
from m9m import WorkflowEngine, Workflow, DataItem

engine: WorkflowEngine = WorkflowEngine()
workflow: Workflow = Workflow.from_file("workflow.json")
result = engine.execute(workflow, [DataItem(json={"key": "value"})])
```

## Requirements

- **Python**: 3.8 or later
- **Operating System**: macOS, Linux, or Windows
- **Architecture**: AMD64 (x86_64) or ARM64 (aarch64)
- **Network**: Internet connection required for initial binary download

## Development

```bash
# Clone the repository
git clone https://github.com/neul-labs/m9m.git
cd m9m/packages/m9m-cli/python

# Install in editable mode
pip install -e ".[dev]"

# Run tests
pytest tests/ -v
```

## Documentation

- [Full Documentation](https://github.com/neul-labs/m9m/tree/main/docs)
- [SDK Reference](https://github.com/neul-labs/m9m/tree/main/docs/sdk)
- [Workflow Examples](https://github.com/neul-labs/m9m/tree/main/examples)
- [n8n Compatibility Guide](https://github.com/neul-labs/m9m/tree/main/docs/api/API_COMPATIBILITY.md)

## License

MIT License — see [LICENSE](https://github.com/neul-labs/m9m/blob/main/LICENSE) for details.

## Contributing

Contributions are welcome! Please see our [Contributing Guide](https://github.com/neul-labs/m9m/blob/main/docs/CONTRIBUTING.md) and [Code of Conduct](https://github.com/neul-labs/m9m/blob/main/CODE_OF_CONDUCT.md).

## Community

- [GitHub Discussions](https://github.com/neul-labs/m9m/discussions)
- [GitHub Issues](https://github.com/neul-labs/m9m/issues)
- [Release Notes](https://github.com/neul-labs/m9m/releases)

---

**Built with ❤️ by [Neul Labs](https://github.com/neul-labs)**
