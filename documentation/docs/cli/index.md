# CLI Reference

The m9m command-line interface provides full control over workflows, executions, and server management.

## Installation

```bash
go install github.com/neul-labs/m9m/cmd/m9m@latest
```

## Global Flags

These flags are available on all commands:

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--workspace` | `-w` | current | Workspace to use |
| `--output` | `-o` | `table` | Output format (table, json, yaml) |
| `--verbose` | `-v` | `false` | Enable verbose output |

## Commands Overview

| Command | Description |
|---------|-------------|
| [`serve`](serve.md) | Start the m9m server |
| [`init`](workflows.md#init) | Initialize a workspace |
| [`list`](workflows.md#list) | List workflows |
| [`get`](workflows.md#get) | Get workflow details |
| [`create`](workflows.md#create) | Create a workflow |
| [`run`](workflows.md#run) | Execute a workflow |
| [`validate`](workflows.md#validate) | Validate workflow JSON |
| [`execution`](executions.md) | Manage executions |
| [`node`](nodes.md) | Explore node types |
| [`workspace`](workspace.md) | Manage workspaces |
| `status` | Check service status |
| `version` | Show version info |

## Quick Examples

```bash
# Start the server
m9m serve

# List all workflows
m9m list

# Run a workflow
m9m run my-workflow.json

# Check execution status
m9m execution list

# Explore available nodes
m9m node list
```

## Output Formats

### Table (Default)

```bash
m9m list
```

```
ID                                    NAME              NODES  ACTIVE  UPDATED
550e8400-e29b-41d4-a716-446655440000  My Workflow       5      true    2024-01-26
```

### JSON

```bash
m9m list --output json
```

```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "My Workflow",
    "nodes": 5,
    "active": true,
    "updatedAt": "2024-01-26T10:00:00Z"
  }
]
```

### YAML

```bash
m9m list --output yaml
```

```yaml
- id: 550e8400-e29b-41d4-a716-446655440000
  name: My Workflow
  nodes: 5
  active: true
  updatedAt: 2024-01-26T10:00:00Z
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `M9M_WORKSPACE` | Default workspace |
| `M9M_CONFIG` | Config file path |
| `M9M_LOG_LEVEL` | Log level (debug, info, warn, error) |

## Configuration File

m9m looks for configuration in:

1. `./config.yaml`
2. `~/.m9m/config/config.yaml`
3. `/etc/m9m/config.yaml`

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid arguments |
| 3 | Workflow not found |
| 4 | Execution failed |

## Getting Help

```bash
# General help
m9m --help

# Command-specific help
m9m serve --help
m9m execution --help
```
