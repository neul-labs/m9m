# Workflow Commands

Commands for managing workflows from the command line.

## init

Initialize a new workspace.

### Synopsis

```bash
m9m init [workspace-name] [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--no-inject` | `false` | Skip CLAUDE.md injection |
| `--storage` | `sqlite` | Storage backend (sqlite, postgres) |

### Examples

```bash
# Initialize in current directory
m9m init

# Initialize named workspace
m9m init my-project

# Skip CLAUDE.md injection
m9m init --no-inject
```

---

## list

List all workflows in the workspace.

### Synopsis

```bash
m9m list [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--search` | - | Search by name |
| `--limit` | `50` | Maximum workflows to show |
| `--active` | - | Filter by active status |

### Examples

```bash
# List all workflows
m9m list

# Search by name
m9m list --search "daily"

# Show only active workflows
m9m list --active true

# Limit results
m9m list --limit 10

# JSON output
m9m list --output json
```

### Output

```
ID                                    NAME              NODES  ACTIVE  UPDATED
550e8400-e29b-41d4-a716-446655440000  Daily Report      5      true    2024-01-26
660e8400-e29b-41d4-a716-446655440001  Webhook Handler   3      true    2024-01-25
770e8400-e29b-41d4-a716-446655440002  Data Sync         8      false   2024-01-24
```

---

## get

Get detailed workflow information.

### Synopsis

```bash
m9m get <workflow-id-or-name> [flags]
```

### Examples

```bash
# Get by ID
m9m get 550e8400-e29b-41d4-a716-446655440000

# Get by name
m9m get "Daily Report"

# JSON output
m9m get "Daily Report" --output json
```

### Output

```
Workflow: Daily Report
ID:       550e8400-e29b-41d4-a716-446655440000
Active:   true
Created:  2024-01-20T10:00:00Z
Updated:  2024-01-26T15:30:00Z

Nodes (5):
  - Start (n8n-nodes-base.start)
  - Fetch Data (n8n-nodes-base.httpRequest)
  - Transform (n8n-nodes-base.set)
  - Filter (n8n-nodes-base.filter)
  - Send Report (n8n-nodes-base.slack)

Connections:
  Start → Fetch Data → Transform → Filter → Send Report
```

---

## create

Create a workflow from a JSON file.

### Synopsis

```bash
m9m create --from <file.json> [flags]
```

### Flags

| Flag | Required | Description |
|------|----------|-------------|
| `--from` | Yes | JSON file path |
| `--name` | No | Override workflow name |
| `--description` | No | Add description |
| `--skip-validate` | No | Skip validation |

### Examples

```bash
# Create from file
m9m create --from workflow.json

# Override name
m9m create --from workflow.json --name "Production Workflow"

# Add description
m9m create --from workflow.json --description "Handles daily reports"

# Skip validation
m9m create --from workflow.json --skip-validate
```

### Output

```
Workflow created successfully!
ID:   550e8400-e29b-41d4-a716-446655440000
Name: My Workflow
```

---

## validate

Validate workflow JSON without creating it.

### Synopsis

```bash
m9m validate <workflow.json> [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--verbose` | `false` | Show detailed validation |
| `--check-nodes` | `false` | Verify node types exist |

### Examples

```bash
# Basic validation
m9m validate workflow.json

# Verbose output
m9m validate workflow.json --verbose

# Check node types
m9m validate workflow.json --check-nodes
```

### Output (Success)

```
✓ Workflow is valid

Summary:
  - Name: My Workflow
  - Nodes: 5
  - Connections: 4
  - No circular dependencies
```

### Output (Error)

```
✗ Validation failed

Errors:
  - Missing required field: nodes[2].type
  - Invalid connection: "Unknown Node" does not exist
  - Circular dependency detected: A → B → C → A
```

---

## run

Execute a workflow.

### Synopsis

```bash
m9m run <workflow-id-or-file> [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--input` | - | Input data (JSON or @file.json) |
| `--raw` | `false` | Output raw result only |
| `--async` | `false` | Run asynchronously |

### Examples

```bash
# Run workflow file
m9m run workflow.json

# Run by ID
m9m run 550e8400-e29b-41d4-a716-446655440000

# Run by name
m9m run "Daily Report"

# With input data
m9m run workflow.json --input '{"name": "John"}'

# Input from file
m9m run workflow.json --input @input.json

# Raw output only
m9m run workflow.json --raw

# Async execution
m9m run workflow.json --async
```

### Output

```
Executing workflow: My Workflow
Execution ID: exec-123456

Status: success
Duration: 1.234s

Output:
{
  "result": "processed",
  "items": 5
}
```

### Async Output

```bash
m9m run workflow.json --async
```

```
Job queued successfully!
Job ID: job-789012

Check status with:
  m9m execution get job-789012
```

---

## activate / deactivate

Enable or disable workflow triggers.

### Synopsis

```bash
m9m activate <workflow-id>
m9m deactivate <workflow-id>
```

### Examples

```bash
# Activate workflow
m9m activate 550e8400-e29b-41d4-a716-446655440000

# Deactivate workflow
m9m deactivate "Daily Report"
```

---

## delete

Delete a workflow.

### Synopsis

```bash
m9m delete <workflow-id> [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--force` | `false` | Skip confirmation |

### Examples

```bash
# Delete with confirmation
m9m delete 550e8400-e29b-41d4-a716-446655440000

# Force delete
m9m delete "Old Workflow" --force
```

---

## See Also

- [Execution Commands](executions.md) - Manage workflow executions
- [API Reference](../api/workflows.md) - Workflow API endpoints
