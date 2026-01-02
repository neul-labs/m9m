# CLI Reference

Complete reference for the m9m command-line interface.

## Global Options

```bash
m9m [global options] command [command options] [arguments...]
```

| Option | Description |
|--------|-------------|
| `--config, -c` | Path to config file |
| `--log-level` | Log level (debug, info, warn, error) |
| `--help, -h` | Show help |
| `--version, -v` | Show version |

## Commands

### serve

Start the m9m server.

```bash
m9m serve [options]
```

| Option | Default | Description |
|--------|---------|-------------|
| `--port, -p` | `8080` | HTTP port |
| `--host` | `0.0.0.0` | Bind address |
| `--metrics-port` | `9090` | Metrics port |
| `--config, -c` | `config.yaml` | Config file |

Examples:
```bash
# Start with defaults
m9m serve

# Custom port
m9m serve --port 3000

# With config file
m9m serve --config /etc/m9m/config.yaml

# Development mode
m9m serve --log-level debug
```

### execute

Execute a workflow.

```bash
m9m execute [options] <workflow-file>
```

| Option | Description |
|--------|-------------|
| `--input, -i` | Input data (JSON) |
| `--input-file` | Input data file |
| `--start-node` | Start from specific node |
| `--skip-trigger` | Skip trigger nodes |
| `--output, -o` | Output file |
| `--format` | Output format (json, table) |

Examples:
```bash
# Execute workflow
m9m execute workflow.json

# With input data
m9m execute workflow.json --input '{"key": "value"}'

# From input file
m9m execute workflow.json --input-file data.json

# Start from specific node
m9m execute workflow.json --start-node "process-data"

# Save output
m9m execute workflow.json --output result.json
```

### validate

Validate a workflow definition.

```bash
m9m validate <workflow-file>
```

Examples:
```bash
# Validate workflow
m9m validate workflow.json

# Validate multiple
m9m validate workflows/*.json
```

### workflow

Manage workflows.

```bash
m9m workflow <subcommand>
```

#### workflow list

```bash
m9m workflow list [options]
```

| Option | Description |
|--------|-------------|
| `--active` | Show only active |
| `--inactive` | Show only inactive |
| `--format` | Output format |

#### workflow get

```bash
m9m workflow get <id>
```

#### workflow create

```bash
m9m workflow create <file>
```

#### workflow update

```bash
m9m workflow update <id> <file>
```

#### workflow delete

```bash
m9m workflow delete <id>
```

#### workflow activate

```bash
m9m workflow activate <id>
```

#### workflow deactivate

```bash
m9m workflow deactivate <id>
```

#### workflow export

```bash
m9m workflow export <id> [--output file.json]
```

#### workflow import

```bash
m9m workflow import <file>
```

### executions

Manage workflow executions.

```bash
m9m executions <subcommand>
```

#### executions list

```bash
m9m executions list [options]
```

| Option | Description |
|--------|-------------|
| `--workflow` | Filter by workflow ID |
| `--status` | Filter by status |
| `--limit` | Number of results |

#### executions get

```bash
m9m executions get <id>
```

#### executions stop

```bash
m9m executions stop <id>
```

#### executions retry

```bash
m9m executions retry <id>
```

### credentials

Manage credentials.

```bash
m9m credentials <subcommand>
```

#### credentials list

```bash
m9m credentials list
```

#### credentials create

```bash
m9m credentials create <name> --type <type> --data '<json>'
```

Examples:
```bash
# API key credential
m9m credentials create my-api --type apiKey --data '{"apiKey": "secret"}'

# From file
m9m credentials create my-db --type postgres --file creds.json
```

#### credentials update

```bash
m9m credentials update <id> --data '<json>'
```

#### credentials delete

```bash
m9m credentials delete <id>
```

#### credentials test

```bash
m9m credentials test <id>
```

### variables

Manage variables.

```bash
m9m variables <subcommand>
```

#### variables list

```bash
m9m variables list
```

#### variables get

```bash
m9m variables get <key>
```

#### variables set

```bash
m9m variables set <key> <value> [--secret]
```

#### variables delete

```bash
m9m variables delete <key>
```

### nodes

Node information.

```bash
m9m nodes <subcommand>
```

#### nodes list

```bash
m9m nodes list [--category <category>]
```

#### nodes info

```bash
m9m nodes info <node-type>
```

### apikey

Manage API keys.

```bash
m9m apikey <subcommand>
```

#### apikey create

```bash
m9m apikey create --name <name> [--scopes <scopes>]
```

#### apikey list

```bash
m9m apikey list
```

#### apikey revoke

```bash
m9m apikey revoke <key>
```

### worker

Run as worker only.

```bash
m9m worker [options]
```

| Option | Default | Description |
|--------|---------|-------------|
| `--workers` | `10` | Number of workers |
| `--queue` | - | Queue URL |

Examples:
```bash
# Start worker
m9m worker --workers 20 --queue redis://localhost:6379
```

### migrate

Database migrations.

```bash
m9m migrate <subcommand>
```

#### migrate up

```bash
m9m migrate up
```

#### migrate down

```bash
m9m migrate down [--steps n]
```

#### migrate status

```bash
m9m migrate status
```

### version

Show version information.

```bash
m9m version
```

Output:
```
m9m version 1.0.0
Build: 2024-01-15T10:30:00Z
Go: go1.21.5
OS/Arch: linux/amd64
```

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid arguments |
| 3 | Configuration error |
| 4 | Execution error |
| 5 | Connection error |

## Shell Completion

### Bash

```bash
# Add to ~/.bashrc
source <(m9m completion bash)
```

### Zsh

```bash
# Add to ~/.zshrc
source <(m9m completion zsh)
```

### Fish

```bash
m9m completion fish | source
```

## Configuration File

The CLI looks for configuration in:

1. Path specified by `--config`
2. `./config.yaml`
3. `~/.m9m/config.yaml`
4. `/etc/m9m/config.yaml`

## Environment Variables

| Variable | Description |
|----------|-------------|
| `M9M_CONFIG` | Config file path |
| `M9M_API_URL` | API server URL |
| `M9M_API_KEY` | API key for authentication |

## Examples

### Common Workflows

```bash
# Start server and run workflow
m9m serve &
m9m execute my-workflow.json

# Deploy workflow from CI/CD
m9m workflow import production-workflow.json
m9m workflow activate <id>

# Monitor executions
watch m9m executions list --status running

# Backup workflows
m9m workflow list --format json | jq '.[] | .id' | xargs -I {} m9m workflow export {}
```

### Scripting

```bash
#!/bin/bash
# Execute workflow and check result

RESULT=$(m9m execute workflow.json --format json)
STATUS=$(echo $RESULT | jq -r '.status')

if [ "$STATUS" = "success" ]; then
    echo "Workflow completed successfully"
    exit 0
else
    echo "Workflow failed: $(echo $RESULT | jq -r '.error')"
    exit 1
fi
```

## Next Steps

- [Configuration](configuration.md) - Full configuration reference
- [Troubleshooting](troubleshooting.md) - Common issues
- [API Reference](../api/rest-api.md) - REST API
