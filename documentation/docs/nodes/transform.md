# Transform Nodes

Transform nodes manipulate, filter, and reshape data within workflows.

## Set Node

Set or modify field values.

### Basic Usage

```json
{
  "type": "n8n-nodes-base.set",
  "parameters": {
    "values": {
      "string": [
        {"name": "greeting", "value": "Hello, World!"}
      ]
    }
  }
}
```

### With Expressions

```json
{
  "parameters": {
    "values": {
      "string": [
        {"name": "fullName", "value": "={{ $json.firstName }} {{ $json.lastName }}"},
        {"name": "email", "value": "={{ $json.email.toLowerCase() }}"}
      ],
      "number": [
        {"name": "total", "value": "={{ $json.price * $json.quantity }}"}
      ],
      "boolean": [
        {"name": "isActive", "value": "={{ $json.status === 'active' }}"}
      ]
    }
  }
}
```

### Keep Only Set Fields

```json
{
  "parameters": {
    "keepOnlySet": true,
    "values": {
      "string": [
        {"name": "id", "value": "={{ $json.id }}"},
        {"name": "name", "value": "={{ $json.name }}"}
      ]
    }
  }
}
```

## Filter Node

Filter items based on conditions.

### Simple Condition

```json
{
  "type": "n8n-nodes-base.filter",
  "parameters": {
    "conditions": {
      "string": [
        {
          "value1": "={{ $json.status }}",
          "operation": "equals",
          "value2": "active"
        }
      ]
    }
  }
}
```

### Multiple Conditions (AND)

```json
{
  "parameters": {
    "conditions": {
      "number": [
        {"value1": "={{ $json.age }}", "operation": "larger", "value2": 18},
        {"value1": "={{ $json.score }}", "operation": "largerEqual", "value2": 50}
      ]
    }
  }
}
```

### OR Logic

Use multiple filter nodes in parallel branches, then merge.

### Available Operations

| Type | Operations |
|------|------------|
| String | equals, notEquals, contains, notContains, startsWith, endsWith, regex |
| Number | equals, notEquals, larger, smaller, largerEqual, smallerEqual |
| Boolean | equals, notEquals |

## IF Node

Conditional branching.

### Basic IF

```json
{
  "type": "n8n-nodes-base.if",
  "parameters": {
    "conditions": {
      "boolean": [
        {
          "value1": "={{ $json.approved }}",
          "value2": true
        }
      ]
    }
  }
}
```

Connections:
- Output 0: Condition is true
- Output 1: Condition is false

### Complex Conditions

```json
{
  "parameters": {
    "conditions": {
      "number": [
        {"value1": "={{ $json.amount }}", "operation": "larger", "value2": 1000}
      ],
      "string": [
        {"value1": "={{ $json.currency }}", "operation": "equals", "value2": "USD"}
      ]
    },
    "combineOperation": "all"
  }
}
```

## Switch Node

Multi-way branching.

```json
{
  "type": "n8n-nodes-base.switch",
  "parameters": {
    "dataType": "string",
    "value1": "={{ $json.status }}",
    "rules": [
      {"value": "pending", "output": 0},
      {"value": "processing", "output": 1},
      {"value": "completed", "output": 2}
    ],
    "fallbackOutput": 3
  }
}
```

## Merge Node

Combine data from multiple inputs.

### Merge by Position

```json
{
  "type": "n8n-nodes-base.merge",
  "parameters": {
    "mode": "mergeByPosition",
    "join": "inner"
  }
}
```

### Merge by Key

```json
{
  "parameters": {
    "mode": "mergeByKey",
    "propertyName1": "id",
    "propertyName2": "userId"
  }
}
```

### Append

Combine all items from all inputs:

```json
{
  "parameters": {
    "mode": "append"
  }
}
```

### Merge Modes

| Mode | Description |
|------|-------------|
| `append` | Concatenate all items |
| `mergeByPosition` | Merge items at same index |
| `mergeByKey` | Merge items with matching key |
| `keepKeyMatches` | Keep only matching keys |
| `removeKeyMatches` | Remove matching keys |

## Split In Batches

Process items in chunks.

```json
{
  "type": "n8n-nodes-base.splitInBatches",
  "parameters": {
    "batchSize": 100
  }
}
```

Use case: Rate-limited APIs, batch database inserts.

## Item Lists

Work with arrays within items.

### Split Out

Convert array to separate items:

```json
{
  "type": "n8n-nodes-base.itemLists",
  "parameters": {
    "operation": "splitOutItems",
    "fieldToSplitOut": "tags"
  }
}
```

Input:
```json
{"name": "Product", "tags": ["electronics", "sale", "new"]}
```

Output:
```json
{"name": "Product", "tags": "electronics"}
{"name": "Product", "tags": "sale"}
{"name": "Product", "tags": "new"}
```

### Aggregate

Combine items into arrays:

```json
{
  "parameters": {
    "operation": "aggregateItems",
    "fieldsToAggregate": ["name", "price"],
    "destinationFieldName": "products"
  }
}
```

### Remove Duplicates

```json
{
  "parameters": {
    "operation": "removeDuplicates",
    "compare": "allFields"
  }
}
```

### Sort

```json
{
  "parameters": {
    "operation": "sort",
    "sortFieldsUi": {
      "sortField": [
        {"fieldName": "createdAt", "order": "descending"}
      ]
    }
  }
}
```

## Code Node

Execute custom JavaScript or Python.

### JavaScript

```json
{
  "type": "n8n-nodes-base.code",
  "parameters": {
    "language": "javascript",
    "code": "return items.map(item => {\n  return {\n    json: {\n      ...item.json,\n      processed: true,\n      timestamp: new Date().toISOString()\n    }\n  };\n});"
  }
}
```

### Python

```json
{
  "type": "n8n-nodes-base.code",
  "parameters": {
    "language": "python",
    "code": "for item in items:\n    item['processed'] = True\nreturn items"
  }
}
```

### Accessing Data

JavaScript:
```javascript
// Current item
const data = $json;

// All input items
const allItems = items;

// Previous node data
const previousData = $node['Previous Node'].json;

// Environment variable
const apiUrl = $env.API_URL;
```

## Function Node

Legacy JavaScript execution.

```json
{
  "type": "n8n-nodes-base.function",
  "parameters": {
    "functionCode": "return items.map(item => {\n  item.json.modified = true;\n  return item;\n});"
  }
}
```

## JSON Node

JSON operations.

### Parse JSON

```json
{
  "type": "n8n-nodes-base.json",
  "parameters": {
    "operation": "parse",
    "jsonInput": "={{ $json.jsonString }}"
  }
}
```

### Stringify

```json
{
  "parameters": {
    "operation": "stringify",
    "dataToStringify": "={{ $json }}"
  }
}
```

## Date & Time Node

Date manipulation.

```json
{
  "type": "n8n-nodes-base.dateTime",
  "parameters": {
    "action": "format",
    "value": "={{ $json.createdAt }}",
    "format": "YYYY-MM-DD HH:mm:ss",
    "timezone": "America/New_York"
  }
}
```

### Operations

| Operation | Description |
|-----------|-------------|
| `format` | Format date to string |
| `parse` | Parse string to date |
| `add` | Add time interval |
| `subtract` | Subtract time interval |
| `difference` | Calculate time difference |

## Rename Keys Node

Rename object properties.

```json
{
  "type": "n8n-nodes-base.renameKeys",
  "parameters": {
    "keys": [
      {"currentName": "firstName", "newName": "first_name"},
      {"currentName": "lastName", "newName": "last_name"}
    ]
  }
}
```

## Best Practices

1. **Use Set node** for simple transformations
2. **Use Code node** for complex logic
3. **Chain transforms** for clarity
4. **Use Filter early** to reduce processing
5. **Batch large datasets** to avoid memory issues

## Next Steps

- [Trigger Nodes](triggers.md) - Start workflows
- [Database Nodes](databases.md) - Data persistence
- [Custom Nodes](custom-nodes.md) - Build your own
