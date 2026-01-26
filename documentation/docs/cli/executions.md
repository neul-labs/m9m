# Execution Commands

Commands for managing workflow executions.

## execution list

List workflow executions.

### Synopsis

```bash
m9m execution list [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--workflow` | - | Filter by workflow name/ID |
| `--status` | - | Filter by status |
| `--limit` | `20` | Maximum executions to show |
| `--since` | - | Show executions since time |

### Status Values

| Status | Description |
|--------|-------------|
| `pending` | Queued, waiting to run |
| `running` | Currently executing |
| `success` | Completed successfully |
| `failed` | Completed with errors |
| `cancelled` | Manually cancelled |

### Examples

```bash
# List recent executions
m9m execution list

# Filter by workflow
m9m execution list --workflow "Daily Report"

# Filter by status
m9m execution list --status failed

# Limit results
m9m execution list --limit 50

# Since a specific time
m9m execution list --since "2024-01-25"

# Combine filters
m9m execution list --workflow "Daily Report" --status success --limit 10
```

### Output

```
ID           WORKFLOW        STATUS   MODE      STARTED              DURATION
exec-001     Daily Report    success  manual    2024-01-26 10:00:00  1.2s
exec-002     Data Sync       failed   schedule  2024-01-26 09:00:00  0.5s
exec-003     Webhook Handler success  webhook   2024-01-26 08:45:00  0.3s
exec-004     Daily Report    running  manual    2024-01-26 10:05:00  -
```

---

## execution get

Get detailed execution information.

### Synopsis

```bash
m9m execution get <execution-id> [flags]
```

### Examples

```bash
# Get execution details
m9m execution get exec-001

# JSON output
m9m execution get exec-001 --output json
```

### Output

```
Execution: exec-001
Workflow:  Daily Report (550e8400-e29b-41d4-a716-446655440000)
Status:    success
Mode:      manual
Started:   2024-01-26T10:00:00Z
Finished:  2024-01-26T10:00:01Z
Duration:  1.234s

Node Results:
  ✓ Start                 0.001s
  ✓ Fetch Data            0.823s
  ✓ Transform             0.102s
  ✓ Filter                0.008s
  ✓ Send Report           0.300s

Output Data:
{
  "reportSent": true,
  "itemsProcessed": 42
}
```

### Failed Execution Output

```
Execution: exec-002
Workflow:  Data Sync
Status:    failed
Mode:      schedule
Started:   2024-01-26T09:00:00Z
Finished:  2024-01-26T09:00:00Z
Duration:  0.5s

Error:
  Node: Fetch Data
  Message: HTTP request failed: connection timeout

Node Results:
  ✓ Start                 0.001s
  ✗ Fetch Data            0.499s (failed)
```

---

## execution watch

Watch executions in real-time.

### Synopsis

```bash
m9m execution watch [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--workflow` | - | Filter to specific workflow |
| `--interval` | `2s` | Refresh interval |

### Examples

```bash
# Watch all executions
m9m execution watch

# Watch specific workflow
m9m execution watch --workflow "Daily Report"

# Custom refresh interval
m9m execution watch --interval 5s
```

### Output

```
Watching executions... (Ctrl+C to stop)

[10:05:00] exec-005 Daily Report    started   manual
[10:05:01] exec-005 Daily Report    success   manual    1.2s
[10:05:15] exec-006 Webhook Handler started   webhook
[10:05:15] exec-006 Webhook Handler success   webhook   0.3s
```

---

## execution retry

Retry a failed execution.

### Synopsis

```bash
m9m execution retry <execution-id> [flags]
```

### Examples

```bash
# Retry failed execution
m9m execution retry exec-002
```

### Output

```
Retrying execution exec-002...

New Execution ID: exec-007
Status: pending

Check status with:
  m9m execution get exec-007
```

---

## execution cancel

Cancel a running execution.

### Synopsis

```bash
m9m execution cancel <execution-id> [flags]
```

### Examples

```bash
# Cancel running execution
m9m execution cancel exec-004
```

### Output

```
Cancelling execution exec-004...
Execution cancelled successfully.
```

---

## execution delete

Delete an execution record.

### Synopsis

```bash
m9m execution delete <execution-id> [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--force` | `false` | Skip confirmation |

### Examples

```bash
# Delete with confirmation
m9m execution delete exec-001

# Force delete
m9m execution delete exec-001 --force

# Delete multiple (with shell expansion)
m9m execution delete exec-001 exec-002 exec-003 --force
```

---

## execution logs

View execution logs.

### Synopsis

```bash
m9m execution logs <execution-id> [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--follow` | `false` | Follow log output |
| `--tail` | `100` | Number of lines to show |

### Examples

```bash
# View logs
m9m execution logs exec-001

# Follow logs (for running executions)
m9m execution logs exec-004 --follow

# Last 50 lines
m9m execution logs exec-001 --tail 50
```

### Output

```
[2024-01-26 10:00:00.000] [INFO] Starting execution exec-001
[2024-01-26 10:00:00.001] [INFO] Node "Start" completed (1ms)
[2024-01-26 10:00:00.824] [INFO] Node "Fetch Data" completed (823ms)
[2024-01-26 10:00:00.926] [INFO] Node "Transform" completed (102ms)
[2024-01-26 10:00:00.934] [INFO] Node "Filter" completed (8ms)
[2024-01-26 10:00:01.234] [INFO] Node "Send Report" completed (300ms)
[2024-01-26 10:00:01.234] [INFO] Execution completed successfully
```

---

## See Also

- [Workflow Commands](workflows.md) - Workflow management
- [API Reference](../api/executions.md) - Execution API endpoints
