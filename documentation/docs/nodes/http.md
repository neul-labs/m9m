# HTTP Node

The HTTP Request node makes HTTP calls to REST APIs and web services.

## HTTP Request Node

### Type

```
n8n-nodes-base.httpRequest
```

### Description

Make HTTP requests to external APIs and services. Supports all HTTP methods, custom headers, request bodies, and authentication.

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `url` | string | Yes | - | Target URL (supports expressions) |
| `method` | string | No | `GET` | HTTP method |
| `headers` | object | No | `{}` | Request headers |
| `body` | string/object | No | - | Request body |
| `authentication` | string | No | `none` | Auth type |

### HTTP Methods

| Method | Description |
|--------|-------------|
| `GET` | Retrieve data |
| `POST` | Create resource |
| `PUT` | Replace resource |
| `PATCH` | Update resource |
| `DELETE` | Delete resource |
| `HEAD` | Get headers only |
| `OPTIONS` | Get allowed methods |

### Examples

#### Simple GET Request

```json
{
  "id": "http-1",
  "name": "Get Users",
  "type": "n8n-nodes-base.httpRequest",
  "position": [450, 300],
  "parameters": {
    "url": "https://api.example.com/users",
    "method": "GET"
  }
}
```

#### GET with Query Parameters

```json
{
  "type": "n8n-nodes-base.httpRequest",
  "parameters": {
    "url": "https://api.example.com/users?page={{ $json.page }}&limit=10",
    "method": "GET"
  }
}
```

#### POST with JSON Body

```json
{
  "type": "n8n-nodes-base.httpRequest",
  "parameters": {
    "url": "https://api.example.com/users",
    "method": "POST",
    "headers": {
      "Content-Type": "application/json"
    },
    "body": {
      "name": "={{ $json.name }}",
      "email": "={{ $json.email }}"
    }
  }
}
```

#### With Custom Headers

```json
{
  "type": "n8n-nodes-base.httpRequest",
  "parameters": {
    "url": "https://api.example.com/data",
    "method": "GET",
    "headers": {
      "Authorization": "Bearer {{ $json.token }}",
      "X-Custom-Header": "custom-value",
      "Accept": "application/json"
    }
  }
}
```

#### PUT Request

```json
{
  "type": "n8n-nodes-base.httpRequest",
  "parameters": {
    "url": "https://api.example.com/users/{{ $json.id }}",
    "method": "PUT",
    "headers": {
      "Content-Type": "application/json"
    },
    "body": {
      "name": "={{ $json.name }}",
      "status": "active"
    }
  }
}
```

#### DELETE Request

```json
{
  "type": "n8n-nodes-base.httpRequest",
  "parameters": {
    "url": "https://api.example.com/users/{{ $json.userId }}",
    "method": "DELETE"
  }
}
```

### Output

The node outputs a single item with the response:

```json
{
  "json": {
    "statusCode": 200,
    "headers": {
      "content-type": "application/json",
      "x-request-id": "abc123"
    },
    "body": "raw response body string",
    "json": {
      "parsed": "response",
      "data": [...]
    }
  }
}
```

| Field | Description |
|-------|-------------|
| `statusCode` | HTTP status code |
| `headers` | Response headers |
| `body` | Raw response body |
| `json` | Parsed JSON (if Content-Type is JSON) |

### Authentication

#### API Key in Header

```json
{
  "type": "n8n-nodes-base.httpRequest",
  "parameters": {
    "url": "https://api.example.com/data",
    "method": "GET",
    "headers": {
      "X-API-Key": "={{ $credentials.apiKey }}"
    }
  }
}
```

#### Bearer Token

```json
{
  "type": "n8n-nodes-base.httpRequest",
  "parameters": {
    "url": "https://api.example.com/data",
    "method": "GET",
    "headers": {
      "Authorization": "Bearer {{ $credentials.token }}"
    }
  }
}
```

#### Basic Auth

```json
{
  "type": "n8n-nodes-base.httpRequest",
  "parameters": {
    "url": "https://api.example.com/data",
    "method": "GET",
    "headers": {
      "Authorization": "Basic {{ Buffer.from($credentials.username + ':' + $credentials.password).toString('base64') }}"
    }
  }
}
```

### Error Handling

Check status codes with a Filter node:

```json
{
  "type": "n8n-nodes-base.filter",
  "parameters": {
    "conditions": [
      {
        "leftValue": "={{ $json.statusCode }}",
        "operator": "equals",
        "rightValue": 200
      }
    ]
  }
}
```

### Common Patterns

#### Pagination Loop

```json
{
  "type": "n8n-nodes-base.httpRequest",
  "parameters": {
    "url": "https://api.example.com/items?page={{ $json.currentPage }}&limit=100",
    "method": "GET"
  }
}
```

#### Retry on Failure

Use the Code node to implement retry logic:

```javascript
const maxRetries = 3;
let attempt = 0;

while (attempt < maxRetries) {
  try {
    // HTTP request logic
    break;
  } catch (error) {
    attempt++;
    if (attempt >= maxRetries) throw error;
  }
}
```

#### Batch Requests

Process multiple URLs:

```json
{
  "type": "n8n-nodes-base.httpRequest",
  "parameters": {
    "url": "={{ $json.endpoint }}",
    "method": "GET"
  }
}
```

With input:
```json
[
  {"json": {"endpoint": "https://api.example.com/item/1"}},
  {"json": {"endpoint": "https://api.example.com/item/2"}}
]
```

### Best Practices

1. **Use expressions for dynamic URLs** - Build URLs from data
2. **Set appropriate headers** - Include Content-Type for POST/PUT
3. **Handle errors** - Check status codes with Filter node
4. **Rate limiting** - Use Split In Batches for bulk requests
5. **Timeout handling** - Set appropriate timeouts for slow APIs

### Related Nodes

- [Webhook](triggers.md#webhook-node) - Receive HTTP requests
- [Filter](transform.md#filter-node) - Check response status
- [Code](transform.md#code-node) - Custom request logic
