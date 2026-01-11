# Custom Plugins via MCP

Create custom nodes on-the-fly using Claude Code. The MCP server supports two types of plugins:

- **JavaScript Plugins** - Full logic using Goja runtime
- **REST Plugins** - Wrap external APIs as nodes

## JavaScript Plugins

JavaScript plugins run in the Goja runtime (ES5.1 compatible) and have full access to input data and parameters.

### Basic Structure

```javascript
module.exports = {
  description: {
    name: "My Plugin",
    description: "What it does",
    category: "transform"  // transform, trigger, action, integration
  },
  execute: function(inputData, params) {
    // Process each item
    return inputData.map(item => ({
      json: {
        ...item.json,
        processed: true
      }
    }));
  },
  validateParameters: function(params) {
    // Optional: validate parameters
    if (!params.requiredField) {
      throw new Error("requiredField is required");
    }
  }
};
```

### Available Context

| Variable | Description |
|----------|-------------|
| `console.log()` | Log to stderr for debugging |
| `$json` | Current item's JSON data |
| `$node` | Node metadata |
| `$parameter(name)` | Get parameter value |
| `inputData` | Array of all input items |
| `params` | Object with all parameter values |

### Example: Phone Formatter

```
You: "Create a plugin that formats US phone numbers to (XXX) XXX-XXXX format"

Claude: [Uses plugin_create_js]
```

```javascript
module.exports = {
  description: {
    name: "US Phone Formatter",
    description: "Formats phone numbers to (XXX) XXX-XXXX",
    category: "transform"
  },
  execute: function(inputData, params) {
    var field = params.field || 'phone';

    return inputData.map(function(item) {
      var phone = item.json[field];
      if (!phone) return item;

      // Remove non-digits
      var digits = String(phone).replace(/\D/g, '');

      // Format if 10 digits
      if (digits.length === 10) {
        var formatted = '(' + digits.slice(0,3) + ') ' +
                        digits.slice(3,6) + '-' +
                        digits.slice(6);
        var result = {};
        for (var key in item.json) {
          result[key] = item.json[key];
        }
        result[field] = formatted;
        return { json: result };
      }

      return item;
    });
  }
};
```

### Example: Data Validator

```javascript
module.exports = {
  description: {
    name: "Email Validator",
    description: "Validates email addresses and adds isValid field",
    category: "transform"
  },
  execute: function(inputData, params) {
    var emailField = params.emailField || 'email';
    var emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

    return inputData.map(function(item) {
      var email = item.json[emailField];
      var result = {};

      for (var key in item.json) {
        result[key] = item.json[key];
      }

      result.isValidEmail = emailRegex.test(email || '');

      return { json: result };
    });
  }
};
```

### Defining Parameters

```javascript
// When creating the plugin via MCP:
{
  "name": "my-plugin",
  "code": "...",
  "parameters": [
    {
      "name": "field",
      "type": "string",
      "required": true,
      "default": "email"
    },
    {
      "name": "strictMode",
      "type": "boolean",
      "required": false,
      "default": false
    }
  ]
}
```

---

## REST Plugins

Wrap external REST APIs as reusable workflow nodes without writing code.

### Basic Structure

```yaml
name: my-api
description: Call My External API
category: integration
endpoint: https://api.example.com/v1/resource
method: POST
headers:
  Content-Type: application/json
  X-Custom-Header: value
timeout: 30s
authType: bearer  # none, bearer, basic, apiKey
```

### Example: Weather API

```
You: "Create a node that fetches weather from OpenWeatherMap"

Claude: [Uses plugin_create_rest]
```

Creates:
```yaml
name: openweathermap
description: Get current weather data
category: integration
endpoint: https://api.openweathermap.org/data/2.5/weather
method: GET
timeout: 10s
authType: apiKey
```

### Example: Internal Ticketing System

```yaml
name: internal-tickets
description: Create support tickets in internal system
category: action
endpoint: https://internal.example.com/api/tickets
method: POST
headers:
  Content-Type: application/json
timeout: 30s
authType: bearer
```

### Authentication Types

| Type | Description | Credential Data |
|------|-------------|-----------------|
| `none` | No authentication | - |
| `bearer` | Bearer token in Authorization header | `{ "token": "..." }` |
| `basic` | Basic auth | `{ "username": "...", "password": "..." }` |
| `apiKey` | API key in header or query | `{ "key": "...", "headerName": "X-API-Key" }` |

---

## Managing Plugins

### List Installed Plugins

```
You: "What custom plugins are installed?"
Claude: [Uses plugin_list]
```

### View Plugin Source

```
You: "Show me the code for the phone-formatter plugin"
Claude: [Uses plugin_get with name="phone-formatter"]
```

### Hot Reload

After modifying a plugin file directly:

```
You: "Reload the phone-formatter plugin"
Claude: [Uses plugin_reload]
```

### Delete Plugin

```
You: "Remove the old-plugin node"
Claude: [Uses plugin_delete with name="old-plugin"]
```

---

## Plugin Storage

Plugins are stored in the plugins directory:

```
{data}/plugins/
├── phone-formatter.js      # JavaScript plugin
├── weather-api.yaml        # REST plugin
└── custom-transform.js
```

The default location is `./data/plugins/`, configurable with `--plugins` flag.

---

## Best Practices

### JavaScript Plugins

1. **Use ES5 syntax** - Goja doesn't support ES6+ features like arrow functions
2. **Handle missing data** - Always check if fields exist before accessing
3. **Return proper structure** - Always return `{ json: {...} }` objects
4. **Log for debugging** - Use `console.log()` during development

### REST Plugins

1. **Set appropriate timeouts** - Don't use overly long timeouts
2. **Use credentials** - Never hardcode API keys in the plugin
3. **Choose correct auth type** - Match the API's authentication method

### General

1. **Use descriptive names** - Plugin names become node type identifiers
2. **Document parameters** - Add descriptions to parameter definitions
3. **Test incrementally** - Create simple plugins first, then add complexity
