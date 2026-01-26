# Expression Variables

All variables available in expressions.

## Current Item Variables

### $json

The current item's JSON data:

```javascript
// Item: { "name": "John", "age": 30 }

{{ $json.name }}      // "John"
{{ $json.age }}       // 30
{{ $json }}           // { "name": "John", "age": 30 }
```

### $item

Current item index (0-based):

```javascript
// Processing 3 items

{{ $item }}  // 0, 1, 2 for each item
```

### $binary

Binary data attached to the item:

```javascript
{{ $binary.data.fileName }}    // "document.pdf"
{{ $binary.data.mimeType }}    // "application/pdf"
{{ $binary.data.fileSize }}    // 1024
```

## Input Data Variables

### $input

Access all input data:

```javascript
{{ $input.all }}              // Array of all input items
{{ $input.first }}            // First input item
{{ $input.last }}             // Last input item
{{ $input.item }}             // Current input item
```

### $input.all Example

```javascript
// Sum all values
{{ $input.all.reduce((sum, item) => sum + item.json.value, 0) }}
```

## Node Access Variables

### $node

Access output from any node by name:

```javascript
{{ $node["Node Name"].json.field }}
```

### Examples

```javascript
// Access HTTP Request output
{{ $node["Fetch Users"].json.users }}

// Access Set node output
{{ $node["Format Data"].json.message }}

// Access first item from node
{{ $node["Get Data"].json }}
```

### Multiple Items from Node

```javascript
// Get all items from a node
{{ $node["Fetch Data"].all }}

// Get specific item
{{ $node["Fetch Data"].item(2).json.field }}
```

## Environment Variables

### $env

Access environment variables:

```javascript
{{ $env.API_KEY }}
{{ $env.DATABASE_URL }}
{{ $env.NODE_ENV }}
```

### Usage Example

```json
{
  "parameters": {
    "url": "{{ $env.API_BASE_URL }}/users",
    "headers": {
      "Authorization": "Bearer {{ $env.API_TOKEN }}"
    }
  }
}
```

## Date/Time Variables

### $now

Current timestamp (ISO 8601):

```javascript
{{ $now }}  // "2024-01-15T10:30:00.000Z"
```

### $today

Current date (YYYY-MM-DD):

```javascript
{{ $today }}  // "2024-01-15"
```

## Workflow Variables

### $workflow

Workflow metadata:

```javascript
{{ $workflow.id }}          // Workflow ID
{{ $workflow.name }}        // Workflow name
{{ $workflow.active }}      // Is active
```

### $execution

Current execution info:

```javascript
{{ $execution.id }}         // Execution ID
{{ $execution.mode }}       // "manual" or "trigger"
```

## Parameter Variables

### $parameter

Access node parameters:

```javascript
{{ $parameter.url }}
{{ $parameter.method }}
```

## Position Variables

### $position

Node position in workflow:

```javascript
{{ $position }}  // Node index in execution order
```

## Variable Scope

| Variable | Scope | Description |
|----------|-------|-------------|
| `$json` | Per item | Current item data |
| `$item` | Per item | Item index |
| `$binary` | Per item | Binary data |
| `$input` | Per node | All input items |
| `$node` | Global | Any node's output |
| `$env` | Global | Environment vars |
| `$now` | Global | Current time |
| `$workflow` | Global | Workflow info |
| `$execution` | Global | Execution info |

## Complex Examples

### Conditional Logic

```javascript
{{ $json.status === "active" ? $json.name : "Inactive: " + $json.name }}
```

### Array Operations

```javascript
// Filter and map
{{ $input.all.filter(i => i.json.active).map(i => i.json.name).join(", ") }}
```

### String Building

```javascript
{{ `User ${$json.name} (ID: ${$json.id}) - ${$json.role}` }}
```

### Date Formatting

```javascript
{{ new Date($json.createdAt).toLocaleDateString() }}
```

### Aggregation

```javascript
// Total from all items
{{ $input.all.reduce((sum, i) => sum + i.json.amount, 0) }}
```

## Best Practices

1. **Use descriptive node names** - Makes `$node["Name"]` readable
2. **Check for undefined** - Use `?.` operator
3. **Provide defaults** - Use `??` or `||` operators
4. **Keep expressions simple** - Complex logic goes in Code nodes

## Common Patterns

### Safe Field Access

```javascript
{{ $json.user?.profile?.email ?? "no-email@example.com" }}
```

### Dynamic Field Names

```javascript
{{ $json[$node["Config"].json.fieldName] }}
```

### JSON Stringification

```javascript
{{ JSON.stringify($json.data) }}
```

### Parsing JSON

```javascript
{{ JSON.parse($json.jsonString) }}
```
