# Expressions Overview

Expressions allow dynamic values in node parameters using data from previous nodes.

## Syntax

Expressions use double curly braces:

```
{{ expression }}
```

## Basic Examples

### Access JSON Data

```javascript
{{ $json.fieldName }}
```

### String Interpolation

```javascript
Hello, {{ $json.name }}! You have {{ $json.count }} messages.
```

### Mathematical Operations

```javascript
{{ $json.price * $json.quantity }}
```

## Expression Context

Every expression has access to:

| Variable | Description |
|----------|-------------|
| `$json` | Current item's JSON data |
| `$item` | Current item index (0-based) |
| `$node` | Access other nodes' output |
| `$input` | Input data helpers |
| `$env` | Environment variables |
| `$now` | Current timestamp |

## Where to Use Expressions

Expressions work in most node parameter fields:

```json
{
  "type": "n8n-nodes-base.httpRequest",
  "parameters": {
    "url": "https://api.example.com/users/{{ $json.userId }}",
    "method": "GET"
  }
}
```

## Expression Types

### Simple Reference

Direct field access:

```javascript
{{ $json.email }}
```

### Nested Access

Access nested objects:

```javascript
{{ $json.user.address.city }}
```

### Array Access

Access array elements:

```javascript
{{ $json.items[0].name }}
```

### Conditional

Ternary expressions:

```javascript
{{ $json.status === "active" ? "Yes" : "No" }}
```

### Function Calls

Use built-in functions:

```javascript
{{ $json.name.toUpperCase() }}
```

## Data Flow Example

```
Node A outputs: { "user": "John", "score": 85 }
     ↓
Node B receives and uses: {{ $json.user }} scored {{ $json.score }}
     ↓
Result: "John scored 85"
```

## Error Handling

### Undefined Values

Use optional chaining:

```javascript
{{ $json.user?.name ?? "Unknown" }}
```

### Default Values

Provide fallbacks:

```javascript
{{ $json.count || 0 }}
```

## Debugging Expressions

### Code Node

Test expressions in a Code node:

```javascript
return items.map(item => ({
  json: {
    original: item.json,
    computed: item.json.value * 2
  }
}));
```

### Log Output

Log values for debugging:

```javascript
console.log($json);
return items;
```

## Best Practices

1. **Use meaningful field names** - Makes expressions readable
2. **Handle missing data** - Use defaults and null checks
3. **Keep expressions simple** - Complex logic belongs in Code nodes
4. **Test with sample data** - Verify before production

## Next Steps

- [Variables Reference](variables.md) - All available variables
- [Functions Reference](functions.md) - Built-in functions
