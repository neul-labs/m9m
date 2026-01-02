# HTTP & API Nodes

HTTP nodes enable communication with external APIs and web services.

## HTTP Request Node

The primary node for making HTTP calls.

### Basic GET Request

```json
{
  "type": "n8n-nodes-base.httpRequest",
  "parameters": {
    "url": "https://api.example.com/users",
    "method": "GET"
  }
}
```

### POST with JSON Body

```json
{
  "parameters": {
    "url": "https://api.example.com/users",
    "method": "POST",
    "bodyType": "json",
    "jsonBody": {
      "name": "={{ $json.name }}",
      "email": "={{ $json.email }}"
    }
  }
}
```

### With Headers

```json
{
  "parameters": {
    "url": "https://api.example.com/data",
    "method": "GET",
    "headers": {
      "Accept": "application/json",
      "X-Custom-Header": "value"
    }
  }
}
```

### Query Parameters

```json
{
  "parameters": {
    "url": "https://api.example.com/search",
    "method": "GET",
    "queryParameters": {
      "q": "={{ $json.searchTerm }}",
      "limit": "100",
      "offset": "={{ $json.page * 100 }}"
    }
  }
}
```

## Authentication

### API Key

```json
{
  "parameters": {
    "url": "https://api.example.com/data",
    "authentication": "headerAuth"
  },
  "credentials": {
    "httpHeaderAuth": {"id": "1", "name": "API Key"}
  }
}
```

Credential:
```json
{
  "type": "httpHeaderAuth",
  "data": {
    "name": "X-API-Key",
    "value": "your-api-key"
  }
}
```

### Basic Auth

```json
{
  "parameters": {
    "authentication": "basicAuth"
  },
  "credentials": {
    "httpBasicAuth": {"id": "1", "name": "Basic Auth"}
  }
}
```

### Bearer Token

```json
{
  "parameters": {
    "authentication": "bearerToken"
  },
  "credentials": {
    "httpBearerAuth": {"id": "1", "name": "Bearer Token"}
  }
}
```

### OAuth2

```json
{
  "parameters": {
    "authentication": "oAuth2"
  },
  "credentials": {
    "oAuth2Api": {"id": "1", "name": "OAuth2"}
  }
}
```

## Request Options

### Timeout

```json
{
  "parameters": {
    "url": "https://slow-api.example.com/data",
    "timeout": 60000
  }
}
```

### Follow Redirects

```json
{
  "parameters": {
    "options": {
      "redirect": {
        "followRedirects": true,
        "maxRedirects": 5
      }
    }
  }
}
```

### Ignore SSL Errors

```json
{
  "parameters": {
    "options": {
      "response": {
        "rejectUnauthorized": false
      }
    }
  }
}
```

!!! warning "Security"
    Only disable SSL verification in development environments.

### Custom Response Handling

```json
{
  "parameters": {
    "options": {
      "response": {
        "response": {
          "neverError": true,
          "fullResponse": true
        }
      }
    }
  }
}
```

## Request Body Types

### JSON

```json
{
  "parameters": {
    "bodyType": "json",
    "jsonBody": {
      "key": "value"
    }
  }
}
```

### Form Data

```json
{
  "parameters": {
    "bodyType": "form",
    "formData": {
      "field1": "value1",
      "field2": "value2"
    }
  }
}
```

### Multipart (File Upload)

```json
{
  "parameters": {
    "bodyType": "multipart",
    "multipartData": [
      {
        "name": "file",
        "binaryData": true,
        "inputDataFieldName": "data"
      },
      {
        "name": "description",
        "value": "My file"
      }
    ]
  }
}
```

### Raw Body

```json
{
  "parameters": {
    "bodyType": "raw",
    "rawBody": "<xml>Custom body</xml>",
    "contentType": "application/xml"
  }
}
```

## Response Handling

### Access Response Data

```javascript
// Response body
{{ $json.data }}

// Specific field
{{ $json.users[0].name }}

// Status code (with fullResponse)
{{ $json.statusCode }}

// Headers (with fullResponse)
{{ $json.headers['content-type'] }}
```

### Binary Response

For file downloads:

```json
{
  "parameters": {
    "responseFormat": "file",
    "outputPropertyName": "data"
  }
}
```

## Pagination

### Offset-Based

```json
{
  "nodes": [
    {
      "id": "init",
      "type": "n8n-nodes-base.set",
      "parameters": {
        "values": {"number": [{"name": "offset", "value": 0}]}
      }
    },
    {
      "id": "fetch",
      "type": "n8n-nodes-base.httpRequest",
      "parameters": {
        "url": "https://api.example.com/items?limit=100&offset={{ $json.offset }}"
      }
    },
    {
      "id": "check-more",
      "type": "n8n-nodes-base.if",
      "parameters": {
        "conditions": {
          "number": [{"value1": "={{ $json.items.length }}", "operation": "equal", "value2": 100}]
        }
      }
    },
    {
      "id": "increment",
      "type": "n8n-nodes-base.set",
      "parameters": {
        "values": {"number": [{"name": "offset", "value": "={{ $json.offset + 100 }}"}]}
      }
    }
  ]
}
```

### Cursor-Based

```javascript
// In Code node
let allItems = [];
let cursor = null;

do {
  const url = cursor
    ? `https://api.example.com/items?cursor=${cursor}`
    : 'https://api.example.com/items';

  const response = await fetch(url);
  const data = await response.json();

  allItems = allItems.concat(data.items);
  cursor = data.nextCursor;
} while (cursor);

return allItems.map(item => ({ json: item }));
```

## GraphQL

### Query

```json
{
  "type": "n8n-nodes-base.graphql",
  "parameters": {
    "endpoint": "https://api.example.com/graphql",
    "query": "query GetUsers($limit: Int!) {\n  users(limit: $limit) {\n    id\n    name\n    email\n  }\n}",
    "variables": {
      "limit": 100
    }
  }
}
```

### Mutation

```json
{
  "parameters": {
    "query": "mutation CreateUser($input: UserInput!) {\n  createUser(input: $input) {\n    id\n    name\n  }\n}",
    "variables": {
      "input": {
        "name": "={{ $json.name }}",
        "email": "={{ $json.email }}"
      }
    }
  }
}
```

## Retry & Error Handling

### Automatic Retry

```json
{
  "retryOnFail": true,
  "maxRetries": 3,
  "retryInterval": 1000,
  "retryConditions": {
    "statusCodes": [429, 500, 502, 503, 504]
  }
}
```

### Handle Rate Limits

```json
{
  "parameters": {
    "options": {
      "response": {
        "response": {
          "neverError": true
        }
      }
    }
  }
}
```

Then check status code:
```javascript
if ($json.statusCode === 429) {
  // Wait and retry
  await new Promise(r => setTimeout(r, $json.headers['retry-after'] * 1000));
}
```

## Webhook Response Node

Send responses to incoming webhooks:

```json
{
  "type": "n8n-nodes-base.respondToWebhook",
  "parameters": {
    "respondWith": "json",
    "responseBody": {
      "success": true,
      "id": "={{ $json.createdId }}"
    },
    "responseCode": 201
  }
}
```

## Best Practices

1. **Use credentials** for authentication - never hardcode tokens
2. **Set appropriate timeouts** for slow APIs
3. **Handle rate limits** with retry logic
4. **Validate responses** before processing
5. **Log requests** for debugging
6. **Use HTTPS** for all production requests

## Next Steps

- [AI Nodes](ai.md) - LLM integrations
- [Transform Nodes](transform.md) - Process API responses
- [Error Handling](../user-guide/error-handling.md) - Handle API failures
