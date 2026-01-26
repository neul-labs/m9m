# Transform Nodes

Transform nodes manipulate and process data flowing through workflows.

## Set Node

Assign values to fields on data items.

### Type

```
n8n-nodes-base.set
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `assignments` | array | Yes | List of field assignments |
| `assignments[].name` | string | Yes | Field name to set |
| `assignments[].value` | string | Yes | Value (supports expressions) |

### Example

```json
{
  "type": "n8n-nodes-base.set",
  "parameters": {
    "assignments": [
      {"name": "fullName", "value": "={{ $json.firstName }} {{ $json.lastName }}"},
      {"name": "status", "value": "active"},
      {"name": "timestamp", "value": "={{ $now }}"}
    ]
  }
}
```

### Input

```json
[{"json": {"firstName": "John", "lastName": "Doe"}}]
```

### Output

```json
[{"json": {"firstName": "John", "lastName": "Doe", "fullName": "John Doe", "status": "active", "timestamp": "2024-01-26T10:00:00Z"}}]
```

---

## Filter Node

Filter items based on conditions.

### Type

```
n8n-nodes-base.filter
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `conditions` | array | Yes | List of conditions to evaluate |
| `combiner` | string | No | `and` (default) or `or` |

### Condition Structure

| Field | Type | Description |
|-------|------|-------------|
| `leftValue` | any | Left side of comparison |
| `operator` | string | Comparison operator |
| `rightValue` | any | Right side of comparison |

### Operators

| Operator | Description |
|----------|-------------|
| `equals` | Equal to |
| `notEquals` | Not equal to |
| `contains` | String contains |
| `notContains` | String doesn't contain |
| `startsWith` | String starts with |
| `endsWith` | String ends with |
| `regex` | Matches regex pattern |
| `exists` | Field exists |
| `notExists` | Field doesn't exist |
| `greaterThan` | Greater than |
| `lessThan` | Less than |
| `greaterThanOrEqual` | Greater than or equal |
| `lessThanOrEqual` | Less than or equal |
| `between` | Between two values |
| `empty` | Is empty |
| `notEmpty` | Is not empty |

### Example

```json
{
  "type": "n8n-nodes-base.filter",
  "parameters": {
    "conditions": [
      {
        "leftValue": "={{ $json.status }}",
        "operator": "equals",
        "rightValue": "active"
      },
      {
        "leftValue": "={{ $json.age }}",
        "operator": "greaterThan",
        "rightValue": 18
      }
    ],
    "combiner": "and"
  }
}
```

---

## Code Node

Execute custom code in JavaScript or Python.

### Type

```
n8n-nodes-base.code
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `language` | string | Yes | `javascript` or `python` |
| `mode` | string | No | `runOnceForAllItems` or `runOnceForEachItem` |
| `code` | string | Yes | Code to execute |

### JavaScript Example

```json
{
  "type": "n8n-nodes-base.code",
  "parameters": {
    "language": "javascript",
    "mode": "runOnceForAllItems",
    "code": "return items.map(item => ({ json: { ...item.json, processed: true } }));"
  }
}
```

### Python Example

Python requires `python3` to be installed on the host system. m9m creates an isolated virtual environment and executes Python code via subprocess.

```json
{
  "type": "n8n-nodes-base.code",
  "parameters": {
    "language": "python",
    "mode": "runOnceForEachItem",
    "code": "output = {'json': {'original': $json, 'doubled': $json.get('value', 0) * 2}}"
  }
}
```

**Python Requirements:**
- Python 3.x installed on host
- Pre-installed packages: `numpy`, `pandas`, `requests`
- Additional allowed packages: `beautifulsoup4`, `matplotlib`, `scipy`, `scikit-learn`, `pillow`, `openpyxl`, `pyyaml`

### Available Variables

| Variable | Description |
|----------|-------------|
| `items` | All input items (runOnceForAllItems) |
| `item` | Current item (runOnceForEachItem) |
| `$input` | Input data reference |

---

## Function Node

Execute JavaScript code (n8n compatible).

### Type

```
n8n-nodes-base.function
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `functionCode` | string | Yes | JavaScript code |

### Example

```json
{
  "type": "n8n-nodes-base.function",
  "parameters": {
    "functionCode": "for (const item of items) {\n  item.json.processed = true;\n}\nreturn items;"
  }
}
```

---

## Merge Node

Combine data from multiple input connections.

### Type

```
n8n-nodes-base.merge
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `mode` | string | Yes | Merge strategy |

### Modes

| Mode | Description |
|------|-------------|
| `append` | Combine all items into single array |
| `merge` | Merge items by index |
| `multiplex` | Create all combinations |

### Example

```json
{
  "type": "n8n-nodes-base.merge",
  "parameters": {
    "mode": "append"
  }
}
```

---

## JSON Node

Parse and manipulate JSON data.

### Type

```
n8n-nodes-base.json
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `operation` | string | Yes | `parse`, `stringify`, or `transform` |
| `jsonInput` | string | Depends | JSON string to parse |

### Example - Parse JSON String

```json
{
  "type": "n8n-nodes-base.json",
  "parameters": {
    "operation": "parse",
    "jsonInput": "={{ $json.jsonString }}"
  }
}
```

---

## Switch Node

Route data to different outputs based on conditions.

### Type

```
n8n-nodes-base.switch
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `rules` | array | Yes | Routing rules |
| `fallbackOutput` | number | No | Default output index |

### Example

```json
{
  "type": "n8n-nodes-base.switch",
  "parameters": {
    "rules": [
      {
        "conditions": [{"leftValue": "={{ $json.type }}", "operator": "equals", "rightValue": "order"}],
        "output": 0
      },
      {
        "conditions": [{"leftValue": "={{ $json.type }}", "operator": "equals", "rightValue": "refund"}],
        "output": 1
      }
    ],
    "fallbackOutput": 2
  }
}
```

---

## Split In Batches Node

Process items in batches.

### Type

```
n8n-nodes-base.splitInBatches
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `batchSize` | number | Yes | Items per batch |

### Example

```json
{
  "type": "n8n-nodes-base.splitInBatches",
  "parameters": {
    "batchSize": 10
  }
}
```

---

## Item Lists Node

Combine or split item arrays.

### Type

```
n8n-nodes-base.itemLists
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `mode` | string | Yes | `combine` or `split` |

### Example - Combine Items

```json
{
  "type": "n8n-nodes-base.itemLists",
  "parameters": {
    "mode": "combine"
  }
}
```

**Input:**
```json
[{"json": {"a": 1}}, {"json": {"b": 2}}]
```

**Output:**
```json
[{"json": {"items": [{"a": 1}, {"b": 2}]}}]
```

---

## Quick Reference

| Node | Purpose | Key Parameter |
|------|---------|---------------|
| Set | Assign fields | `assignments` |
| Filter | Conditional pass | `conditions` |
| Code | Custom code | `language`, `code` |
| Function | JavaScript | `functionCode` |
| Merge | Combine inputs | `mode` |
| JSON | JSON operations | `operation` |
| Switch | Conditional routing | `rules` |
| Split In Batches | Batch processing | `batchSize` |
| Item Lists | Array operations | `mode` |
