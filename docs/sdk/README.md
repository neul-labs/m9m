# m9m SDK & Language Bindings

m9m can be embedded as a library in your applications. This documentation covers the Go SDK and native bindings for Python and Node.js.

## Overview

| Language | Package | Status |
|----------|---------|--------|
| Go | `pkg/m9m` | Stable |
| Python | `bindings/python` | Stable |
| Node.js | `bindings/nodejs` | Stable |

## Go SDK

The Go SDK provides direct access to the m9m workflow engine.

### Installation

```go
import "github.com/m9m/m9m/pkg/m9m"
```

### Quick Start

```go
package main

import (
    "fmt"
    "log"
    "github.com/m9m/m9m/pkg/m9m"
)

func main() {
    // Create a new engine
    engine := m9m.New()

    // Load workflow from file
    workflow, err := m9m.LoadWorkflow("workflow.json")
    if err != nil {
        log.Fatal(err)
    }

    // Execute with input data
    result, err := engine.Execute(workflow, []m9m.DataItem{
        {JSON: map[string]interface{}{"message": "Hello"}},
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Result: %+v\n", result.Data)
}
```

### API Reference

#### Engine

```go
// Create new engine
engine := m9m.New()

// Execute workflow
result, err := engine.Execute(workflow, inputData)

// Register custom node
engine.RegisterNode("custom.myNode", myExecutor)

// Set credential manager
engine.SetCredentialManager(credManager)
```

#### Workflow

```go
// Load from file
workflow, err := m9m.LoadWorkflow("path/to/workflow.json")

// Parse from JSON
workflow, err := m9m.ParseWorkflow(jsonBytes)

// Create programmatically
workflow := &m9m.Workflow{
    Name: "My Workflow",
    Nodes: []m9m.WorkflowNode{...},
    Connections: map[string]m9m.NodeConnections{...},
}
```

#### Custom Nodes

```go
type MyNode struct {
    m9m.BaseNode
}

func (n *MyNode) Execute(input []m9m.DataItem, params map[string]interface{}) ([]m9m.DataItem, error) {
    // Process input
    return []m9m.DataItem{
        {JSON: map[string]interface{}{"result": "processed"}},
    }, nil
}

func (n *MyNode) Description() m9m.NodeDescription {
    return m9m.NodeDescription{
        Name:        "My Node",
        Description: "Custom processing node",
        Category:    "transform",
    }
}
```

---

## Python Bindings

Python bindings use ctypes to call the CGO shared library.

### Installation

```bash
# Build the shared library first
cd /path/to/m9m
make cgo-lib

# Install Python package
cd bindings/python
uv pip install -e .
```

### Quick Start

```python
from m9m import WorkflowEngine, Workflow

# Create engine
engine = WorkflowEngine()

# Load workflow
workflow = Workflow.from_file("workflow.json")

# Execute
result = engine.execute(workflow, [{"json": {"input": "data"}}])
print(result.data)
```

### API Reference

#### WorkflowEngine

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
@engine.node("custom.myNode", name="My Node", category="transform")
def my_node(input_data, params):
    return [{"json": {"result": "processed"}}]
```

#### Workflow

```python
from m9m import Workflow

# From file
workflow = Workflow.from_file("workflow.json")

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

# Convert to dict
data = workflow.to_dict()
```

#### Data Types

```python
from m9m import DataItem, ExecutionResult

# DataItem
item = DataItem(json={"key": "value"})
item = DataItem.from_dict({"json": {"key": "value"}})

# ExecutionResult
result.data      # List[DataItem]
result.error     # Optional error message
result.success   # Boolean
```

---

## Node.js Bindings

Node.js bindings use N-API for native addon support.

### Installation

```bash
# Build the shared library
cd /path/to/m9m
make cgo-lib

# Install and build Node.js bindings
cd bindings/nodejs
npm install
npm run build
```

### Quick Start

```typescript
import { WorkflowEngine, Workflow } from '@m9m/workflow-engine';

const engine = new WorkflowEngine();

// Load workflow from JSON object
const workflow = Workflow.fromJSON({
  name: 'My Workflow',
  nodes: [],
  connections: {}
});

// Execute
const result = await engine.execute(workflow, [{ json: { input: 'data' } }]);
console.log(result.data);
```

### API Reference

#### WorkflowEngine

```typescript
import { WorkflowEngine, CredentialManager } from '@m9m/workflow-engine';

// Basic engine
const engine = new WorkflowEngine();

// Engine with credentials
const credManager = new CredentialManager();
const engine = new WorkflowEngine(credManager);

// Execute workflow
const result = await engine.execute(workflow, inputData);
```

#### Workflow

```typescript
import { Workflow } from '@m9m/workflow-engine';

// From JSON object
const workflow = Workflow.fromJSON({
  name: 'Test',
  nodes: [],
  connections: {}
});

// From file
const workflow = Workflow.fromFile('workflow.json');

// Access properties
console.log(workflow.name);
console.log(workflow.id);

// Convert to JSON
const json = workflow.toJSON();
```

#### Types

```typescript
interface DataItem {
  json: Record<string, unknown>;
  binary?: Record<string, BinaryData>;
  pairedItem?: PairedItemInfo;
  error?: ExecutionError;
}

interface ExecutionResult {
  data: DataItem[];
  error?: string;
}

interface WorkflowData {
  name: string;
  nodes: NodeData[];
  connections: Record<string, NodeConnections>;
  active?: boolean;
  settings?: WorkflowSettings;
}
```

---

## Building from Source

### Prerequisites

- Go 1.21+ with CGO enabled
- GCC/Clang for CGO compilation
- Python 3.8+ (for Python bindings)
- Node.js 16+ (for Node.js bindings)

### Build Commands

```bash
# Build CGO shared library
make cgo-lib

# Platform-specific builds
make cgo-lib-linux    # Linux (.so)
make cgo-lib-darwin   # macOS (.dylib)
make cgo-lib-windows  # Windows (.dll)

# Build all bindings
make python-bindings
make nodejs-bindings
```

### Running Tests

```bash
# Go SDK tests
cd pkg/m9m && go test -v ./...

# Python tests
cd bindings/python && uv run pytest tests/ -v

# Node.js tests
cd bindings/nodejs && npm test
```

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Application                          │
├─────────────┬─────────────┬─────────────────────────────┤
│   Go SDK    │   Python    │         Node.js             │
│  (pkg/m9m)  │  (ctypes)   │         (N-API)             │
├─────────────┴─────────────┴─────────────────────────────┤
│                  CGO Shared Library                     │
│                    (libm9m.so)                          │
├─────────────────────────────────────────────────────────┤
│                  m9m Core Engine                        │
│               (internal/engine)                         │
└─────────────────────────────────────────────────────────┘
```

## See Also

- [Go SDK Examples](../../examples/sdk/)
- [Node Development](../nodes/README.md)
- [API Reference](../api/API_COMPATIBILITY.md)
