# Expressions

Expressions allow you to dynamically reference and transform data within workflows.

## Expression Syntax

Expressions are enclosed in double curly braces with an equals sign:

```
{{ expression }}
```

Or the legacy format:

```
={{ expression }}
```

## Accessing Data

### Current Node Data

Access data from the current execution context:

```javascript
// Current item's JSON data
{{ $json.fieldName }}

// Nested fields
{{ $json.user.email }}

// Array access
{{ $json.items[0].name }}
```

### Input Data

Access data from the input item:

```javascript
// Input item data
{{ $input.item.json.fieldName }}

// First input item
{{ $input.first().json.fieldName }}

// All input items
{{ $input.all() }}
```

### Previous Node Data

Reference data from specific nodes:

```javascript
// Data from named node
{{ $node["HTTP Request"].json.statusCode }}

// First item from node
{{ $node["Get Users"].first().json.email }}

// All items from node
{{ $node["Database Query"].all() }}
```

## Built-in Variables

### Workflow Variables

```javascript
// Current workflow
{{ $workflow.id }}
{{ $workflow.name }}
{{ $workflow.active }}

// Current execution
{{ $execution.id }}
{{ $execution.mode }}
{{ $execution.resumeUrl }}
```

### Item Variables

```javascript
// Current item index
{{ $itemIndex }}

// Run index (for loops)
{{ $runIndex }}

// Position in output
{{ $position }}
```

### Date and Time

```javascript
// Current timestamp
{{ $now }}

// Formatted date
{{ $now.format('YYYY-MM-DD') }}

// ISO string
{{ $now.toISO() }}

// Unix timestamp
{{ $now.toMillis() }}

// Date manipulation
{{ $now.plus({days: 7}).toISO() }}
{{ $now.minus({hours: 2}).format('HH:mm') }}
```

### Environment

```javascript
// Environment variables
{{ $env.MY_VARIABLE }}

// Timezone
{{ $env.TIMEZONE }}
```

## Data Transformation

### String Operations

```javascript
// Uppercase
{{ $json.name.toUpperCase() }}

// Lowercase
{{ $json.email.toLowerCase() }}

// Trim whitespace
{{ $json.input.trim() }}

// Replace
{{ $json.text.replace('old', 'new') }}

// Split
{{ $json.tags.split(',') }}

// Substring
{{ $json.code.substring(0, 3) }}
```

### Number Operations

```javascript
// Arithmetic
{{ $json.price * 1.1 }}
{{ $json.total + $json.tax }}

// Rounding
{{ Math.round($json.value) }}
{{ Math.floor($json.amount) }}
{{ $json.price.toFixed(2) }}

// Parsing
{{ parseInt($json.id) }}
{{ parseFloat($json.rate) }}
```

### Array Operations

```javascript
// Length
{{ $json.items.length }}

// Map
{{ $json.users.map(u => u.name) }}

// Filter
{{ $json.items.filter(i => i.active) }}

// Find
{{ $json.users.find(u => u.id === 123) }}

// Join
{{ $json.tags.join(', ') }}

// Includes
{{ $json.roles.includes('admin') }}
```

### Object Operations

```javascript
// Keys
{{ Object.keys($json.data) }}

// Values
{{ Object.values($json.config) }}

// Merge
{{ {...$json.defaults, ...$json.overrides} }}

// Property access
{{ $json.users['user-id-123'] }}
```

## Conditional Expressions

### Ternary Operator

```javascript
{{ $json.status === 'active' ? 'Yes' : 'No' }}
```

### Nullish Coalescing

```javascript
// Default value if null/undefined
{{ $json.name ?? 'Unknown' }}
```

### Optional Chaining

```javascript
// Safe property access
{{ $json.user?.profile?.avatar }}
```

### Logical Operators

```javascript
// AND
{{ $json.active && $json.verified }}

// OR
{{ $json.nickname || $json.username || 'Anonymous' }}
```

## JSON Operations

### Parse JSON

```javascript
{{ JSON.parse($json.jsonString) }}
```

### Stringify

```javascript
{{ JSON.stringify($json.data) }}
{{ JSON.stringify($json.data, null, 2) }}
```

## Practical Examples

### Building URLs

```javascript
{{ 'https://api.example.com/users/' + $json.userId + '/profile' }}
```

### Formatting Messages

```javascript
{{ `Hello ${$json.firstName}, your order #${$json.orderId} is ready!` }}
```

### Conditional Values

```javascript
{{ $json.type === 'premium' ? $json.price * 0.9 : $json.price }}
```

### Data Extraction

```javascript
// Extract domain from email
{{ $json.email.split('@')[1] }}

// Get file extension
{{ $json.filename.split('.').pop() }}
```

### Date Formatting

```javascript
// Custom format
{{ $now.format('dddd, MMMM D, YYYY') }}

// Relative time
{{ $now.minus({days: $json.daysAgo}).toRelative() }}
```

### Array Processing

```javascript
// Sum values
{{ $json.items.reduce((sum, i) => sum + i.price, 0) }}

// Get unique values
{{ [...new Set($json.tags)] }}

// Sort items
{{ $json.users.sort((a, b) => a.name.localeCompare(b.name)) }}
```

## Expression Editor

The Web UI includes an expression editor with:

- Syntax highlighting
- Auto-completion
- Preview of results
- Error detection

### Using the Editor

1. Click the expression icon (=) next to a field
2. Type your expression
3. View the result preview
4. Click **Apply**

## Debugging Expressions

### Test in Editor

Use the expression editor's preview to test:

```javascript
// Add to see intermediate values
{{ console.log($json.data) || $json.data.value }}
```

### Common Errors

**"Cannot read property of undefined"**
```javascript
// Problem
{{ $json.user.name }}

// Solution: Use optional chaining
{{ $json.user?.name }}
```

**"is not a function"**
```javascript
// Problem: Wrong type
{{ $json.count.map(...) }}

// Solution: Ensure it's an array
{{ Array.isArray($json.count) ? $json.count.map(...) : [] }}
```

## Best Practices

1. **Use optional chaining** for potentially missing data
2. **Provide defaults** with `??` or `||`
3. **Keep expressions simple** - use Code nodes for complex logic
4. **Test thoroughly** with different input scenarios
5. **Document complex expressions** in node notes

## Next Steps

- [Credentials](credentials.md) - Manage authentication
- [Variables](variables.md) - Environment configuration
- [Nodes Reference](../nodes/overview.md) - Available nodes
