# Node Commands

Commands for exploring and testing available node types.

## node list

List all available node types.

### Synopsis

```bash
m9m node list [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--category` | - | Filter by category |
| `--search` | - | Search by name or description |

### Categories

| Category | Description |
|----------|-------------|
| `core` | Workflow control nodes |
| `transform` | Data transformation |
| `trigger` | Workflow triggers |
| `http` | HTTP requests |
| `database` | Database operations |
| `messaging` | Chat platforms |
| `ai` | AI/LLM services |
| `cli` | CLI execution & AI agents |
| `cloud` | Cloud storage |
| `vcs` | Version control |
| `email` | Email operations |
| `file` | File operations |

### Examples

```bash
# List all nodes
m9m node list

# Filter by category
m9m node list --category ai

# Search by name
m9m node list --search "http"

# JSON output
m9m node list --output json
```

### Output

```
TYPE                              NAME              CATEGORY    DESCRIPTION
n8n-nodes-base.start              Start             core        Workflow entry point
n8n-nodes-base.httpRequest        HTTP Request      http        Make HTTP requests
n8n-nodes-base.set                Set               transform   Set field values
n8n-nodes-base.filter             Filter            transform   Filter items by condition
n8n-nodes-base.code               Code              transform   Execute custom code
n8n-nodes-base.cliExecute         CLI Execute       cli         Execute CLI commands and AI agents
n8n-nodes-base.webhook            Webhook           trigger     HTTP webhook trigger
n8n-nodes-base.cron               Cron              trigger     Scheduled trigger
n8n-nodes-base.slack              Slack             messaging   Send Slack messages
n8n-nodes-base.openAi             OpenAI            ai          OpenAI API integration
...
```

---

## node categories

List all node categories with counts.

### Synopsis

```bash
m9m node categories
```

### Examples

```bash
m9m node categories
```

### Output

```
CATEGORY    COUNT  DESCRIPTION
core        1      Workflow control nodes
transform   9      Data transformation
trigger     2      Workflow triggers
http        1      HTTP requests
database    3      Database operations
messaging   2      Chat platforms
ai          2      AI/LLM services
cli         1      CLI execution & AI agents
cloud       4      Cloud storage
vcs         2      Version control
email       1      Email operations
file        2      File operations
───────────────────────────────
TOTAL       30
```

---

## node info

Get detailed information about a node type.

### Synopsis

```bash
m9m node info <node-type>
```

### Examples

```bash
# Get node info
m9m node info n8n-nodes-base.httpRequest

# Shorthand (without prefix)
m9m node info httpRequest
```

### Output

```
Node: HTTP Request
Type: n8n-nodes-base.httpRequest
Category: http
Version: 1.0.0

Description:
  Make HTTP requests to REST APIs and web services. Supports all HTTP
  methods, custom headers, request bodies, and various authentication
  options.

Parameters:
  NAME          TYPE      REQUIRED  DEFAULT   DESCRIPTION
  url           string    yes       -         Target URL
  method        string    no        GET       HTTP method
  headers       object    no        {}        Request headers
  body          any       no        -         Request body
  authentication string   no        none      Auth type

Methods:
  GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS

Example:
  {
    "type": "n8n-nodes-base.httpRequest",
    "parameters": {
      "url": "https://api.example.com/data",
      "method": "GET"
    }
  }

See also:
  - Documentation: https://docs.neullabs.com/m9m/nodes/http
```

---

## node test

Test a node with sample data.

### Synopsis

```bash
m9m node test <node-type> [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--input` | - | Input data (JSON or @file) |
| `--params` | - | Node parameters (JSON or @file) |

### Examples

```bash
# Test with default sample data
m9m node test n8n-nodes-base.set

# Test with custom input
m9m node test n8n-nodes-base.filter --input '{"status": "active"}'

# Test with parameters
m9m node test n8n-nodes-base.set --params '{"assignments": [{"name": "test", "value": "hello"}]}'

# Input from file
m9m node test n8n-nodes-base.httpRequest --params @params.json
```

### Output

```
Testing node: n8n-nodes-base.set

Input:
  [{"json": {"name": "John"}}]

Parameters:
  {
    "assignments": [
      {"name": "greeting", "value": "Hello, {{ $json.name }}!"}
    ]
  }

Output:
  [{"json": {"name": "John", "greeting": "Hello, John!"}}]

Status: Success
Duration: 2ms
```

---

## node create

Create a custom node from a script file.

### Synopsis

```bash
m9m node create --from <script-file> [flags]
```

### Flags

| Flag | Required | Description |
|------|----------|-------------|
| `--from` | Yes | Script file path (.js or .py) |
| `--name` | No | Override node name |
| `--category` | No | Node category |

### Script Requirements

The script must export:

- `name` - Node display name
- `description` - Node description
- `category` - Node category
- `execute` - Execution function

### JavaScript Example

```javascript
// custom-node.js
module.exports = {
  name: "My Custom Node",
  description: "Does custom processing",
  category: "transform",

  execute: function(items, params) {
    return items.map(item => ({
      json: {
        ...item.json,
        processed: true,
        timestamp: new Date().toISOString()
      }
    }));
  }
};
```

### Examples

```bash
# Create from JavaScript
m9m node create --from custom-node.js

# Override name
m9m node create --from custom-node.js --name "Custom Processor"

# Set category
m9m node create --from custom-node.py --category transform
```

### Output

```
Custom node created successfully!

Type: custom.myCustomNode
Name: My Custom Node
Category: transform

Use in workflows:
  {
    "type": "custom.myCustomNode",
    "parameters": {}
  }
```

---

## See Also

- [Nodes Reference](../nodes/index.md) - Full node documentation
- [Transform Nodes](../nodes/transform.md) - Data transformation nodes
