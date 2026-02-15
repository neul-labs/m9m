# m9m Agent Examples

These workflows are designed for use with the `m9m exec` command, which provides
a simple, agent-friendly interface for workflow execution.

## Quick Start

```bash
# Execute a workflow
m9m exec simple-transform.json

# Execute with input data
m9m exec simple-transform.json --input '{"name": "test"}'

# Pipe data from another command
echo '{"data": "value"}' | m9m exec simple-transform.json --stdin

# Get only the result data (no metadata)
m9m exec simple-transform.json --quiet

# Pretty-print output
m9m exec simple-transform.json --pretty
```

## Example Workflows

### simple-transform.json
Basic data transformation that sets greeting and processed fields.

```bash
m9m exec simple-transform.json --pretty
```

### http-request.json
Makes an HTTP GET request to httpbin.org.

```bash
m9m exec http-request.json --pretty
```

### filter-data.json
Filters items based on an "active" field.

```bash
m9m exec filter-data.json --input '[{"name":"a","active":true},{"name":"b","active":false}]' --quiet
```

## Output Format

The default output is JSON with metadata:

```json
{
  "success": true,
  "data": [...],
  "duration": "15.2ms",
  "nodeCount": 1
}
```

Use `--quiet` for just the data array:

```json
[{"field": "value"}]
```

## Error Handling

Errors are returned as JSON:

```json
{
  "success": false,
  "error": "Error description"
}
```

Exit codes:
- 0: Success
- 1: Error (workflow not found, execution failed, etc.)

## For Agents

The `m9m exec` command is designed for programmatic use:

1. **No daemon required**: Direct execution without background services
2. **Structured JSON output**: Easy to parse
3. **Stdin support**: Chain with other commands
4. **Quiet mode**: Get just the data you need
5. **Exit codes**: Proper error signaling
