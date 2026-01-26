# Workspace Commands

Commands for managing m9m workspaces.

## Overview

Workspaces allow you to organize workflows into separate environments. Each workspace has its own:

- Workflows
- Executions
- Credentials
- Configuration

## workspace list

List all workspaces.

### Synopsis

```bash
m9m workspace list
```

### Examples

```bash
m9m workspace list
```

### Output

```
NAME          LOCATION                        WORKFLOWS  ACTIVE
default       ~/.m9m/workspaces/default       12         *
production    ~/.m9m/workspaces/production    8
development   ~/.m9m/workspaces/development   15
testing       ./my-project                    3
```

---

## workspace current

Show the current active workspace.

### Synopsis

```bash
m9m workspace current
```

### Examples

```bash
m9m workspace current
```

### Output

```
Current workspace: default
Location: /home/user/.m9m/workspaces/default
Workflows: 12
Storage: sqlite (/home/user/.m9m/workspaces/default/m9m.db)
```

---

## workspace use

Switch to a different workspace.

### Synopsis

```bash
m9m workspace use <name>
```

### Examples

```bash
# Switch workspace
m9m workspace use production

# Switch to local project workspace
m9m workspace use ./my-project
```

### Output

```
Switched to workspace: production
Location: /home/user/.m9m/workspaces/production
```

---

## workspace create

Create a new workspace.

### Synopsis

```bash
m9m workspace create <name> [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--path` | - | Custom location for workspace |
| `--storage` | `sqlite` | Storage backend |
| `--switch` | `true` | Switch to new workspace |

### Examples

```bash
# Create workspace
m9m workspace create staging

# Create with custom path
m9m workspace create myproject --path ./projects/myproject

# Create without switching
m9m workspace create testing --switch=false

# Create with PostgreSQL storage
m9m workspace create production --storage postgres
```

### Output

```
Workspace created successfully!

Name: staging
Location: /home/user/.m9m/workspaces/staging
Storage: sqlite

Switched to workspace: staging
```

---

## workspace delete

Delete a workspace.

### Synopsis

```bash
m9m workspace delete <name> [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--force` | `false` | Skip confirmation |
| `--keep-data` | `false` | Keep workflow data |

### Examples

```bash
# Delete with confirmation
m9m workspace delete testing

# Force delete
m9m workspace delete old-project --force

# Keep data files
m9m workspace delete staging --keep-data
```

### Output

```
Are you sure you want to delete workspace "testing"?
This will delete all workflows and executions. (y/N): y

Workspace "testing" deleted.
```

---

## workspace info

Show detailed workspace information.

### Synopsis

```bash
m9m workspace info [name]
```

### Examples

```bash
# Current workspace info
m9m workspace info

# Specific workspace info
m9m workspace info production
```

### Output

```
Workspace: production

Location:     /home/user/.m9m/workspaces/production
Storage:      sqlite
Database:     /home/user/.m9m/workspaces/production/m9m.db

Statistics:
  Workflows:   8
  Active:      5
  Executions:  1,234
  Credentials: 3

Last Activity: 2024-01-26T15:30:00Z
```

---

## workspace export

Export workspace workflows.

### Synopsis

```bash
m9m workspace export [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--output` | `-` | Output file (- for stdout) |
| `--format` | `json` | Export format (json, yaml) |
| `--include-credentials` | `false` | Include credentials |

### Examples

```bash
# Export to file
m9m workspace export --output backup.json

# Export to stdout
m9m workspace export

# YAML format
m9m workspace export --format yaml --output backup.yaml

# Include credentials (encrypted)
m9m workspace export --include-credentials --output full-backup.json
```

---

## workspace import

Import workflows into workspace.

### Synopsis

```bash
m9m workspace import <file> [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--overwrite` | `false` | Overwrite existing workflows |
| `--skip-validation` | `false` | Skip workflow validation |

### Examples

```bash
# Import workflows
m9m workspace import backup.json

# Overwrite existing
m9m workspace import backup.json --overwrite

# Import from URL
m9m workspace import https://example.com/workflows.json
```

### Output

```
Importing workflows...

Imported: 8 workflows
  ✓ Daily Report
  ✓ Data Sync
  ✓ Webhook Handler
  ✓ Email Notifications
  ⊘ Duplicate: Alert System (skipped, use --overwrite to replace)
  ...

Import completed!
```

---

## Using Workspaces

### Global Flag

Use any workspace with the `-w` flag:

```bash
# List workflows in production workspace
m9m list -w production

# Run workflow in staging
m9m run workflow.json -w staging
```

### Project-Local Workspaces

Initialize a workspace in your project:

```bash
cd my-project
m9m init

# This creates:
# my-project/
#   .m9m/
#     config.yaml
#     m9m.db
```

### Environment Variable

Set default workspace:

```bash
export M9M_WORKSPACE=production
m9m list  # Uses production workspace
```

---

## See Also

- [Configuration](../configuration/index.md) - Workspace configuration
- [Deployment](../deployment/index.md) - Multi-environment deployments
