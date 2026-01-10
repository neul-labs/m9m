# SDK Overview

m9m can be embedded as a library in your applications. We provide native bindings for **Go**, **Python**, and **Node.js**.

## Supported Languages

| Language | Package | Build Requirement |
|----------|---------|-------------------|
| Go | `pkg/m9m` | None (direct import) |
| Python | `bindings/python` | CGO shared library |
| Node.js | `bindings/nodejs` | CGO shared library + N-API |

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Your Application                     │
├─────────────┬─────────────┬─────────────────────────────┤
│   Go SDK    │   Python    │         Node.js             │
│  (pkg/m9m)  │  (ctypes)   │         (N-API)             │
├─────────────┴─────────────┴─────────────────────────────┤
│                  CGO Shared Library                     │
│                    (libm9m.so)                          │
├─────────────────────────────────────────────────────────┤
│                  m9m Core Engine                        │
└─────────────────────────────────────────────────────────┘
```

## Quick Start

=== "Go"

    ```go
    import "github.com/m9m/m9m/pkg/m9m"

    engine := m9m.New()
    workflow, _ := m9m.LoadWorkflow("workflow.json")
    result, _ := engine.Execute(workflow, nil)
    ```

=== "Python"

    ```python
    from m9m import WorkflowEngine, Workflow

    engine = WorkflowEngine()
    workflow = Workflow.from_file("workflow.json")
    result = engine.execute(workflow)
    ```

=== "Node.js"

    ```typescript
    import { WorkflowEngine, Workflow } from '@m9m/workflow-engine';

    const engine = new WorkflowEngine();
    const workflow = Workflow.fromFile('workflow.json');
    const result = await engine.execute(workflow);
    ```

## Building the CGO Library

Python and Node.js bindings require the CGO shared library:

```bash
# Linux
make cgo-lib

# macOS
make cgo-lib-darwin

# Windows
make cgo-lib-windows
```

## Use Cases

- **Embedded Automation**: Add workflow capabilities to existing applications
- **Custom Integrations**: Build specialized workflow tools
- **Testing**: Programmatically test workflows
- **CI/CD Pipelines**: Execute workflows in build pipelines
- **Microservices**: Embed workflow engine in service architectures
