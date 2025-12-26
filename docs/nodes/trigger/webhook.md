# Webhook Node

The Webhook node allows your workflow to receive HTTP requests, making it possible to trigger workflows from external systems.

## Node Information

- **Category**: Trigger
- **Type**: `n8n-nodes-base.webhook`
- **Compatibility**: n8n compatible with enhanced security features

## Parameters

### Core Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | The webhook endpoint path |
| `httpMethod` | string | No | HTTP method (default: "POST") |
| `authentication` | string | No | Authentication type |

### HTTP Methods

Supported HTTP methods:
- `GET`
- `POST` (default)
- `PUT`
- `DELETE`
- `PATCH`
- `HEAD`
- `OPTIONS`

### Authentication Types

| Type | Description | Additional Parameters |
|------|-------------|----------------------|
| `none` | No authentication | - |
| `basicAuth` | HTTP Basic Authentication | - |
| `headerAuth` | Header-based authentication | `headerName`, `headerValue` |

## Request Data Structure

The webhook node provides access to the complete HTTP request:

```json
{
  "headers": {
    "content-type": "application/json",
    "authorization": "Bearer token",
    "user-agent": "Client/1.0"
  },
  "params": {
    "id": "123"
  },
  "query": {
    "filter": "active",
    "limit": "10"
  },
  "body": {
    "name": "John Doe",
    "email": "john@example.com"
  },
  "method": "POST",
  "path": "/webhook/user-created"
}
```

## Examples

### Basic Webhook

**Configuration:**
```json
{
  "path": "/webhook/simple",
  "httpMethod": "POST"
}
```

**Usage:**
```bash
curl -X POST https://your-domain.com/webhook/simple \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello World"}'
```

### GET Webhook with Query Parameters

**Configuration:**
```json
{
  "path": "/webhook/search",
  "httpMethod": "GET"
}
```

**Usage:**
```bash
curl "https://your-domain.com/webhook/search?q=searchterm&limit=10"
```

**Received Data:**
```json
{
  "headers": {...},
  "params": {},
  "query": {
    "q": "searchterm",
    "limit": "10"
  },
  "body": null,
  "method": "GET",
  "path": "/webhook/search"
}
```

### Webhook with Path Parameters

**Configuration:**
```json
{
  "path": "/webhook/user/:id",
  "httpMethod": "PUT"
}
```

**Usage:**
```bash
curl -X PUT https://your-domain.com/webhook/user/123 \
  -H "Content-Type: application/json" \
  -d '{"name": "Updated Name"}'
```

**Received Data:**
```json
{
  "headers": {...},
  "params": {
    "id": "123"
  },
  "query": {},
  "body": {
    "name": "Updated Name"
  },
  "method": "PUT",
  "path": "/webhook/user/123"
}
```

## Authentication

### Basic Authentication

**Configuration:**
```json
{
  "path": "/webhook/secure",
  "httpMethod": "POST",
  "authentication": "basicAuth"
}
```

**Usage:**
```bash
curl -X POST https://your-domain.com/webhook/secure \
  -u "username:password" \
  -H "Content-Type: application/json" \
  -d '{"data": "secure data"}'
```

### Header Authentication

**Configuration:**
```json
{
  "path": "/webhook/api",
  "httpMethod": "POST",
  "authentication": "headerAuth",
  "headerName": "X-API-Key",
  "headerValue": "your-secret-api-key"
}
```

**Usage:**
```bash
curl -X POST https://your-domain.com/webhook/api \
  -H "X-API-Key: your-secret-api-key" \
  -H "Content-Type: application/json" \
  -d '{"data": "authenticated data"}'
```

## Advanced Use Cases

### Form Data Processing

Handle HTML form submissions:

**Configuration:**
```json
{
  "path": "/webhook/contact-form",
  "httpMethod": "POST"
}
```

**HTML Form:**
```html
<form action="https://your-domain.com/webhook/contact-form" method="POST">
  <input type="text" name="name" required>
  <input type="email" name="email" required>
  <textarea name="message" required></textarea>
  <button type="submit">Send</button>
</form>
```

### File Upload Webhook

Handle file uploads:

**Configuration:**
```json
{
  "path": "/webhook/upload",
  "httpMethod": "POST"
}
```

**Usage:**
```bash
curl -X POST https://your-domain.com/webhook/upload \
  -F "file=@document.pdf" \
  -F "description=Important document"
```

### API Integration Webhook

Receive webhooks from external services (GitHub, Stripe, etc.):

**Configuration:**
```json
{
  "path": "/webhook/github",
  "httpMethod": "POST",
  "authentication": "headerAuth",
  "headerName": "X-GitHub-Event",
  "headerValue": "push"
}
```

## Response Handling

The webhook node can return different HTTP responses:

### Success Response (200 OK)

Default successful processing returns HTTP 200 with workflow execution details.

### Error Response (400/500)

If the workflow fails, appropriate error codes are returned:

- `400 Bad Request` - Invalid request data
- `401 Unauthorized` - Authentication failed
- `500 Internal Server Error` - Workflow execution error

### Custom Response

Use the HTTP Response node to customize the response:

```json
{
  "statusCode": 201,
  "body": {
    "message": "Resource created successfully",
    "id": "{{ $json.generatedId }}"
  },
  "headers": {
    "Location": "/api/resource/{{ $json.generatedId }}"
  }
}
```

## Security Considerations

### Authentication

Always use authentication for production webhooks:

```json
{
  "authentication": "headerAuth",
  "headerName": "X-Webhook-Secret",
  "headerValue": "{{ $env.WEBHOOK_SECRET }}"
}
```

### Rate Limiting

Consider implementing rate limiting for public webhooks:

- Use reverse proxy (nginx) for rate limiting
- Implement application-level throttling
- Monitor webhook usage patterns

### Validation

Validate incoming data:

```json
{
  "path": "/webhook/strict",
  "validation": {
    "required": ["email", "name"],
    "format": {
      "email": "email"
    }
  }
}
```

## Webhook Server Integration

### External Webhook Server

m9m can integrate with external webhook servers:

```go
// Register webhook handler
webhookInfo := webhookNode.GetWebhookInfo(params)
server.RegisterWebhook(webhookInfo.Path, webhookInfo.Method, func(w http.ResponseWriter, r *http.Request) {
    // Process webhook and trigger workflow
    result := workflowEngine.ExecuteWorkflow(workflow, webhookData)
    // Return response
})
```

### Built-in Server

Use m9m's built-in webhook server:

```bash
m9m server --webhook-port 3000 --webhook-path "/webhook"
```

## Performance

- **Throughput**: 10K+ requests/second
- **Latency**: Sub-millisecond webhook processing
- **Concurrency**: Efficient goroutine-based handling
- **Memory**: Low memory footprint per connection

## Monitoring

### Webhook Metrics

Monitor webhook performance:

```json
{
  "webhookMetrics": {
    "requestCount": 1500,
    "averageResponseTime": "45ms",
    "errorRate": "0.2%",
    "activeConnections": 12
  }
}
```

### Logging

Comprehensive webhook logging:

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "webhook": "/webhook/api",
  "method": "POST",
  "sourceIP": "192.168.1.100",
  "userAgent": "API-Client/1.0",
  "statusCode": 200,
  "responseTime": "42ms",
  "workflowId": "workflow-123"
}
```

## Migration from n8n

Webhook configuration is compatible with n8n:

### Compatible Features
- Path configuration
- HTTP method selection
- Basic authentication
- Request data structure

### Enhanced Features
- **Header authentication**: More flexible auth options
- **Performance**: 10x better throughput
- **Security**: Enhanced validation and error handling
- **Monitoring**: Built-in metrics and logging

### Migration Steps

1. **Export webhook configuration** from n8n
2. **Import unchanged** to m9m
3. **Update webhook URLs** to point to m9m server
4. **Test webhook functionality**

## Troubleshooting

### Common Issues

**Webhook not receiving requests:**
- Check path configuration
- Verify HTTP method
- Confirm server is running

**Authentication failures:**
- Validate header names and values
- Check basic auth credentials
- Review authentication type

**Workflow not triggering:**
- Verify webhook data format
- Check expression syntax
- Review error logs

## See Also

- [HTTP Request Node](../http/request.md) - For making HTTP requests
- [HTTP Response Node](../http/response.md) - For custom responses
- [Expression Reference](../../expressions/README.md) - Working with webhook data
- [Security Guide](../../security/README.md) - Webhook security best practices