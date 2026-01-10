# m9m Python Bindings

Python bindings for the m9m high-performance workflow automation engine.

## Installation

```bash
# From source
cd bindings/python
pip install -e .
```

Before using, ensure the m9m shared library is built:

```bash
cd cgo
make shared
```

## Quick Start

```python
from m9m import WorkflowEngine, Workflow

# Create an engine
engine = WorkflowEngine()

# Load and execute a workflow
workflow = Workflow.from_file("workflow.json")
result = engine.execute(workflow)

print(f"Success: {result.success}")
for item in result.data:
    print(item.json)
```

## Custom Nodes

Register custom nodes using the decorator:

```python
@engine.node("custom.uppercase", name="Uppercase")
def uppercase_node(input_data, params):
    return [
        {"json": {"text": item["json"]["text"].upper()}}
        for item in input_data
    ]
```

Or register directly:

```python
def my_node(input_data, params):
    return input_data

engine.register_node("custom.myNode", my_node)
```

## Credentials

Use the credential manager for secure credential storage:

```python
from m9m import WorkflowEngine, CredentialManager

cred_mgr = CredentialManager()
cred_mgr.store({
    "id": "api-key-1",
    "name": "My API Key",
    "type": "apiKey",
    "data": {"apiKey": "secret123"}
})

engine = WorkflowEngine(credential_manager=cred_mgr)
```

## API Reference

### WorkflowEngine

- `execute(workflow, input_data=None)` - Execute a workflow
- `load_workflow(path)` - Load workflow from file
- `parse_workflow(json_str)` - Parse workflow from JSON
- `register_node(node_type, fn)` - Register a custom node
- `node(node_type, ...)` - Decorator to register nodes

### Workflow

- `from_file(path)` - Load from JSON file
- `from_json(json_str)` - Parse from JSON string
- `to_json()` - Serialize to JSON
- `id`, `name`, `active`, `nodes` - Properties

### ExecutionResult

- `data` - List of output DataItems
- `error` - Error message if failed
- `success` - Boolean success flag

## License

MIT License
