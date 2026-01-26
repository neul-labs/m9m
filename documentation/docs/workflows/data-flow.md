# Data Flow

Understanding how data moves through workflows.

## Data Items

Data flows through workflows as **items**. Each item is:

```json
{
  "json": {
    "field": "value"
  },
  "binary": {}
}
```

| Property | Description |
|----------|-------------|
| `json` | Main data payload |
| `binary` | Binary data (files, images) |

## Node Input/Output

Every node:

1. **Receives** items from connected upstream nodes
2. **Processes** the items
3. **Outputs** zero, one, or many items

```
Input: [item1, item2]
         ↓
    ┌─────────┐
    │  Node   │
    └─────────┘
         ↓
Output: [item3]
```

## Data Flow Patterns

### Sequential Flow

Each node passes to the next:

```
A → B → C → D
```

### Branching

One node feeds multiple:

```
    ┌→ B
A ──┤
    └→ C
```

Both B and C receive A's output.

### Merging

Multiple nodes feed one:

```
A ──┐
    ├→ C
B ──┘
```

C receives items from both A and B.

### Conditional (Switch)

Different paths based on conditions:

```
      ┌→ B (if condition)
A ────┤
      └→ C (else)
```

## Item Transformation

### 1:1 Transformation (Set)

Each input item produces one output:

```
Input:  [{json: {a: 1}}, {json: {a: 2}}]
                  ↓
              Set Node
                  ↓
Output: [{json: {a: 1, b: 10}}, {json: {a: 2, b: 20}}]
```

### Filtering (Filter)

Remove items that don't match:

```
Input:  [{json: {status: "active"}}, {json: {status: "inactive"}}]
                           ↓
                    Filter (status=active)
                           ↓
Output: [{json: {status: "active"}}]
```

### Aggregation (Combine)

Multiple items to one:

```
Input:  [{json: {n: 1}}, {json: {n: 2}}, {json: {n: 3}}]
                           ↓
                    Item Lists (combine)
                           ↓
Output: [{json: {items: [{n: 1}, {n: 2}, {n: 3}]}}]
```

### Expansion (Split)

One item to many:

```
Input:  [{json: {items: [1, 2, 3]}}]
                    ↓
             Item Lists (split)
                    ↓
Output: [{json: {value: 1}}, {json: {value: 2}}, {json: {value: 3}}]
```

## Accessing Data

### Current Item

```javascript
{{ $json.fieldName }}
```

### Item Index

```javascript
{{ $item }}  // 0, 1, 2, ...
```

### All Input Items

```javascript
{{ $input.all }}
```

### Previous Node Output

```javascript
{{ $node["Node Name"].json.field }}
```

## Data Types

### JSON Data

Most data is JSON:

```json
{
  "json": {
    "string": "text",
    "number": 42,
    "boolean": true,
    "array": [1, 2, 3],
    "object": {"nested": "value"}
  }
}
```

### Binary Data

Files, images, etc.:

```json
{
  "json": {"filename": "document.pdf"},
  "binary": {
    "data": {
      "data": "base64-encoded-content",
      "mimeType": "application/pdf",
      "fileName": "document.pdf"
    }
  }
}
```

## Multiple Outputs

Some nodes have multiple outputs (e.g., Switch):

```json
{
  "Switch": {
    "main": [
      [{"node": "Path A"}],  // Output 0
      [{"node": "Path B"}]   // Output 1
    ]
  }
}
```

## Data Persistence

Data is stored in:

- **Execution records** - Full input/output for debugging
- **Node results** - Per-node output data

## Example: Complete Flow

```
Start
  ↓
[{json: {}}]  (empty item)
  ↓
HTTP Request (GET /users)
  ↓
[{json: {users: [{id: 1, name: "John"}, {id: 2, name: "Jane"}]}}]
  ↓
Item Lists (split on users)
  ↓
[{json: {id: 1, name: "John"}}, {json: {id: 2, name: "Jane"}}]
  ↓
Filter (id > 1)
  ↓
[{json: {id: 2, name: "Jane"}}]
  ↓
Set (add greeting)
  ↓
[{json: {id: 2, name: "Jane", greeting: "Hello, Jane!"}}]
```

## Debugging Data Flow

### Log Output

Use Code node to log:

```javascript
console.log(JSON.stringify(items, null, 2));
return items;
```

### Check Execution

View execution details:

```bash
m9m execution get exec-123
```

Shows data at each node.
