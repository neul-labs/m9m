# Expression Functions

Built-in functions available in expressions.

## String Functions

### toUpperCase()

Convert to uppercase:

```javascript
{{ $json.name.toUpperCase() }}  // "JOHN"
```

### toLowerCase()

Convert to lowercase:

```javascript
{{ $json.name.toLowerCase() }}  // "john"
```

### trim()

Remove whitespace:

```javascript
{{ $json.text.trim() }}  // "hello" (from "  hello  ")
```

### replace()

Replace text:

```javascript
{{ $json.text.replace("old", "new") }}
{{ $json.text.replace(/\s+/g, "-") }}  // Regex
```

### split()

Split into array:

```javascript
{{ $json.tags.split(",") }}  // ["tag1", "tag2", "tag3"]
```

### substring()

Extract portion:

```javascript
{{ $json.text.substring(0, 10) }}  // First 10 chars
```

### includes()

Check if contains:

```javascript
{{ $json.email.includes("@gmail.com") }}  // true/false
```

### startsWith() / endsWith()

Check prefix/suffix:

```javascript
{{ $json.url.startsWith("https://") }}
{{ $json.file.endsWith(".pdf") }}
```

### padStart() / padEnd()

Pad string:

```javascript
{{ String($json.id).padStart(5, "0") }}  // "00042"
```

## Number Functions

### Math.round()

Round to nearest integer:

```javascript
{{ Math.round($json.price) }}  // 10 (from 9.7)
```

### Math.floor() / Math.ceil()

Round down/up:

```javascript
{{ Math.floor($json.value) }}  // 9 (from 9.7)
{{ Math.ceil($json.value) }}   // 10 (from 9.2)
```

### Math.abs()

Absolute value:

```javascript
{{ Math.abs($json.difference) }}  // 5 (from -5)
```

### Math.min() / Math.max()

Find minimum/maximum:

```javascript
{{ Math.min($json.a, $json.b) }}
{{ Math.max(...$json.values) }}  // From array
```

### toFixed()

Format decimal places:

```javascript
{{ $json.price.toFixed(2) }}  // "9.99"
```

### parseInt() / parseFloat()

Parse numbers:

```javascript
{{ parseInt($json.stringNum) }}     // 42
{{ parseFloat($json.stringDec) }}   // 3.14
```

## Array Functions

### length

Get array length:

```javascript
{{ $json.items.length }}  // 5
```

### join()

Join elements:

```javascript
{{ $json.tags.join(", ") }}  // "tag1, tag2, tag3"
```

### map()

Transform elements:

```javascript
{{ $json.users.map(u => u.name) }}  // ["John", "Jane"]
```

### filter()

Filter elements:

```javascript
{{ $json.items.filter(i => i.active) }}
```

### find()

Find first match:

```javascript
{{ $json.users.find(u => u.id === 123) }}
```

### reduce()

Aggregate values:

```javascript
{{ $json.items.reduce((sum, i) => sum + i.price, 0) }}
```

### includes()

Check if contains:

```javascript
{{ $json.roles.includes("admin") }}  // true/false
```

### indexOf()

Find position:

```javascript
{{ $json.items.indexOf("target") }}  // -1 if not found
```

### slice()

Extract portion:

```javascript
{{ $json.items.slice(0, 5) }}  // First 5 items
```

### sort()

Sort array:

```javascript
{{ $json.numbers.sort((a, b) => a - b) }}  // Ascending
{{ $json.names.sort() }}  // Alphabetical
```

### reverse()

Reverse order:

```javascript
{{ $json.items.reverse() }}
```

## Object Functions

### Object.keys()

Get property names:

```javascript
{{ Object.keys($json.data) }}  // ["name", "age", "email"]
```

### Object.values()

Get property values:

```javascript
{{ Object.values($json.data) }}  // ["John", 30, "john@example.com"]
```

### Object.entries()

Get key-value pairs:

```javascript
{{ Object.entries($json.data) }}  // [["name", "John"], ["age", 30]]
```

### JSON.stringify()

Convert to JSON string:

```javascript
{{ JSON.stringify($json.data) }}
{{ JSON.stringify($json.data, null, 2) }}  // Pretty print
```

### JSON.parse()

Parse JSON string:

```javascript
{{ JSON.parse($json.jsonString) }}
```

## Date Functions

### new Date()

Create date:

```javascript
{{ new Date() }}                    // Current date/time
{{ new Date($json.timestamp) }}     // From timestamp
{{ new Date($json.dateString) }}    // From string
```

### Date Methods

```javascript
{{ new Date().toISOString() }}        // "2024-01-15T10:30:00.000Z"
{{ new Date().toLocaleDateString() }} // "1/15/2024"
{{ new Date().toLocaleTimeString() }} // "10:30:00 AM"
{{ new Date().getFullYear() }}        // 2024
{{ new Date().getMonth() }}           // 0 (January)
{{ new Date().getDate() }}            // 15
{{ new Date().getDay() }}             // 1 (Monday)
{{ new Date().getTime() }}            // Unix timestamp ms
```

### Date Arithmetic

```javascript
// Add days
{{ new Date(new Date().getTime() + 7 * 24 * 60 * 60 * 1000).toISOString() }}

// Days between dates
{{ Math.floor((new Date($json.end) - new Date($json.start)) / (1000 * 60 * 60 * 24)) }}
```

## Type Functions

### typeof

Check type:

```javascript
{{ typeof $json.value }}  // "string", "number", "object", etc.
```

### Array.isArray()

Check if array:

```javascript
{{ Array.isArray($json.items) }}  // true/false
```

### String()

Convert to string:

```javascript
{{ String($json.number) }}  // "42"
```

### Number()

Convert to number:

```javascript
{{ Number($json.string) }}  // 42
```

### Boolean()

Convert to boolean:

```javascript
{{ Boolean($json.value) }}  // true/false
```

## Utility Functions

### encodeURIComponent()

URL encode:

```javascript
{{ encodeURIComponent($json.query) }}
```

### decodeURIComponent()

URL decode:

```javascript
{{ decodeURIComponent($json.encoded) }}
```

### btoa()

Base64 encode:

```javascript
{{ btoa($json.text) }}
```

### atob()

Base64 decode:

```javascript
{{ atob($json.encoded) }}
```

## Common Patterns

### Format Currency

```javascript
{{ "$" + $json.price.toFixed(2) }}  // "$9.99"
```

### Slugify String

```javascript
{{ $json.title.toLowerCase().replace(/\s+/g, "-").replace(/[^a-z0-9-]/g, "") }}
```

### Extract Domain

```javascript
{{ new URL($json.url).hostname }}  // "example.com"
```

### Format Date

```javascript
{{ new Date($json.date).toLocaleDateString("en-US", {
  year: "numeric",
  month: "long",
  day: "numeric"
}) }}  // "January 15, 2024"
```

### Safe Access

```javascript
{{ $json.data?.nested?.field ?? "default" }}
```

### Unique Array

```javascript
{{ [...new Set($json.items)] }}
```

### Group Count

```javascript
{{ $input.all.filter(i => i.json.status === "active").length }}
```
