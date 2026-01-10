# Python Bindings

Python bindings use ctypes to interface with the CGO shared library.

## Prerequisites

- Python 3.8+
- CGO shared library (`libm9m.so`, `libm9m.dylib`, or `m9m.dll`)

## Installation

```bash
# Build the shared library first
cd /path/to/m9m
make cgo-lib

# Install Python package
cd bindings/python
uv pip install -e .
```

## Quick Start

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

## API Reference

### WorkflowEngine

#### Creating an Engine

```python
from m9m import WorkflowEngine, CredentialManager

# Basic engine
engine = WorkflowEngine()

# Engine with credential manager
cred_manager = CredentialManager()
engine = WorkflowEngine(credential_manager=cred_manager)
```

#### Executing Workflows

```python
# Basic execution
result = engine.execute(workflow)

# With input data
result = engine.execute(workflow, [
    {"json": {"key": "value"}},
    {"json": {"key": "value2"}}
])

# Check results
if result.success:
    for item in result.data:
        print(item.json)
else:
    print(f"Error: {result.error}")
```

#### Registering Custom Nodes

```python
@engine.node("custom.myNode", name="My Node", category="transform")
def my_node(input_data: list, params: dict) -> list:
    """Process input data and return results."""
    return [{"json": {"processed": True}}]
```

### Workflow

#### Loading Workflows

```python
from m9m import Workflow

# From file
workflow = Workflow.from_file("workflow.json")

# From JSON string
workflow = Workflow.from_json('{"name": "test", "nodes": [], "connections": {}}')

# From dictionary
workflow = Workflow.from_dict({
    "name": "My Workflow",
    "nodes": [],
    "connections": {}
})
```

#### Workflow Properties

```python
workflow.name      # Workflow name
workflow.id        # Unique identifier

# Convert to dict
data = workflow.to_dict()
```

### CredentialManager

```python
from m9m import CredentialManager

cred_manager = CredentialManager()

# Store credentials
cred_manager.store({
    "id": "my-api-key",
    "name": "API Key",
    "type": "apiKey",
    "data": {"apiKey": "secret-key"}
})
```

### Data Types

#### DataItem

```python
from m9m import DataItem

# Create from dict
item = DataItem.from_dict({"json": {"key": "value"}})

# Access data
print(item.json)       # {"key": "value"}
print(item.binary)     # Optional binary data
print(item.error)      # Optional error info
```

#### ExecutionResult

```python
result = engine.execute(workflow)

result.data      # List[DataItem] - output items
result.error     # Optional[str] - error message
result.success   # bool - True if no error
```

## Custom Nodes

### Using Decorators

```python
@engine.node(
    "custom.transform",
    name="Transform Data",
    category="transform",
    description="Transforms input data"
)
def transform_node(input_data: list, params: dict) -> list:
    multiplier = params.get("multiplier", 1)
    results = []

    for item in input_data:
        value = item.get("json", {}).get("value", 0)
        results.append({
            "json": {"result": value * multiplier}
        })

    return results
```

### Manual Registration

```python
def my_processor(input_data: list, params: dict) -> list:
    return [{"json": {"processed": True}}]

engine.register_node(
    "custom.processor",
    my_processor,
    name="Processor",
    category="transform"
)
```

## Error Handling

```python
from m9m import M9MError

try:
    workflow = Workflow.from_file("missing.json")
except M9MError as e:
    print(f"m9m error: {e}")
except FileNotFoundError:
    print("Workflow file not found")

# Check execution errors
result = engine.execute(workflow)
if not result.success:
    print(f"Execution failed: {result.error}")
```

## Environment Setup

The library automatically searches for the shared library in:

1. `bindings/python/m9m/` directory
2. System library paths
3. `LD_LIBRARY_PATH` (Linux) / `DYLD_LIBRARY_PATH` (macOS)

```bash
# Set library path if needed
export LD_LIBRARY_PATH=/path/to/m9m/cgo:$LD_LIBRARY_PATH
```

## Testing

```bash
cd bindings/python
uv run pytest tests/ -v
```
