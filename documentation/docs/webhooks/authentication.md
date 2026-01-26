# Webhook Authentication

Secure your webhooks with authentication.

## Authentication Methods

| Method | Description | Use Case |
|--------|-------------|----------|
| None | No authentication | Internal/testing |
| Header | Custom header validation | Simple API keys |
| Basic Auth | Username/password | Standard HTTP auth |
| HMAC Signature | Cryptographic verification | Third-party services |
| JWT | Token-based | Advanced security |

## No Authentication

Default - accepts all requests:

```json
{
  "type": "n8n-nodes-base.webhook",
  "parameters": {
    "path": "/public-webhook",
    "authentication": "none"
  }
}
```

Only use for:

- Internal services
- Development/testing
- Public endpoints (with validation)

## Header Authentication

Validate a custom header:

```json
{
  "type": "n8n-nodes-base.webhook",
  "parameters": {
    "path": "/secure-webhook",
    "authentication": "headerAuth",
    "headerName": "X-API-Key",
    "headerValue": "={{ $env.WEBHOOK_API_KEY }}"
  }
}
```

Caller must include:

```bash
curl -X POST http://localhost:8080/webhook/secure-webhook \
  -H "X-API-Key: your-secret-key" \
  -H "Content-Type: application/json" \
  -d '{"data": "test"}'
```

## Basic Authentication

HTTP Basic auth:

```json
{
  "type": "n8n-nodes-base.webhook",
  "parameters": {
    "path": "/basic-auth-webhook",
    "authentication": "basicAuth"
  },
  "credentials": {
    "httpBasicAuth": {
      "id": "cred-123"
    }
  }
}
```

Caller:

```bash
curl -X POST http://localhost:8080/webhook/basic-auth-webhook \
  -u username:password \
  -H "Content-Type: application/json" \
  -d '{"data": "test"}'
```

## HMAC Signature Verification

Verify webhook signatures from services like GitHub, Stripe:

### GitHub Webhooks

```json
{
  "type": "n8n-nodes-base.webhook",
  "parameters": {
    "path": "/github",
    "authentication": "hmac",
    "hmacSecret": "={{ $env.GITHUB_WEBHOOK_SECRET }}",
    "hmacHeader": "X-Hub-Signature-256",
    "hmacAlgorithm": "sha256"
  }
}
```

### Stripe Webhooks

```json
{
  "type": "n8n-nodes-base.webhook",
  "parameters": {
    "path": "/stripe",
    "authentication": "hmac",
    "hmacSecret": "={{ $env.STRIPE_WEBHOOK_SECRET }}",
    "hmacHeader": "Stripe-Signature"
  }
}
```

### Custom HMAC

Generate signature on caller side:

```python
import hmac
import hashlib
import json

body = json.dumps({"event": "test"})
secret = "your-secret"
signature = hmac.new(
    secret.encode(),
    body.encode(),
    hashlib.sha256
).hexdigest()
```

Include in request:

```bash
curl -X POST http://localhost:8080/webhook/my-webhook \
  -H "X-Signature: sha256=abc123..." \
  -H "Content-Type: application/json" \
  -d '{"event": "test"}'
```

## JWT Authentication

Validate JWT tokens:

```json
{
  "type": "n8n-nodes-base.webhook",
  "parameters": {
    "path": "/jwt-webhook",
    "authentication": "jwt",
    "jwtSecret": "={{ $env.JWT_SECRET }}",
    "jwtHeader": "Authorization"
  }
}
```

Caller:

```bash
curl -X POST http://localhost:8080/webhook/jwt-webhook \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -H "Content-Type: application/json" \
  -d '{"data": "test"}'
```

Access JWT claims:

```javascript
{{ $json.jwt.sub }}      // Subject
{{ $json.jwt.exp }}      // Expiration
{{ $json.jwt.claims }}   // Custom claims
```

## IP Allowlist

Restrict by source IP:

```json
{
  "type": "n8n-nodes-base.webhook",
  "parameters": {
    "path": "/ip-restricted",
    "ipAllowlist": [
      "192.168.1.0/24",
      "10.0.0.1"
    ]
  }
}
```

Or configure globally:

```yaml
webhooks:
  ipAllowlist:
    - "192.168.1.0/24"
    - "10.0.0.0/8"
```

## Custom Validation

Add a Filter or Code node after webhook:

### Filter Node

```json
{
  "type": "n8n-nodes-base.filter",
  "parameters": {
    "conditions": [
      {
        "leftValue": "={{ $json.headers['x-custom-token'] }}",
        "operator": "equals",
        "rightValue": "expected-value"
      }
    ]
  }
}
```

### Code Node

```javascript
// Custom validation logic
const token = $json.headers['x-api-key'];
const validTokens = ['token1', 'token2', 'token3'];

if (!validTokens.includes(token)) {
  throw new Error('Invalid API key');
}

return items;
```

## Service-Specific Authentication

### GitHub

```json
{
  "parameters": {
    "authentication": "hmac",
    "hmacSecret": "={{ $env.GITHUB_SECRET }}",
    "hmacHeader": "X-Hub-Signature-256",
    "hmacAlgorithm": "sha256"
  }
}
```

### Stripe

```json
{
  "parameters": {
    "authentication": "stripe"
  },
  "credentials": {
    "stripeWebhook": {
      "id": "cred-stripe"
    }
  }
}
```

### Slack

Validate Slack signing secret:

```json
{
  "parameters": {
    "authentication": "hmac",
    "hmacSecret": "={{ $env.SLACK_SIGNING_SECRET }}",
    "hmacHeader": "X-Slack-Signature"
  }
}
```

### Twilio

```json
{
  "parameters": {
    "authentication": "hmac",
    "hmacSecret": "={{ $env.TWILIO_AUTH_TOKEN }}",
    "hmacHeader": "X-Twilio-Signature"
  }
}
```

## Security Best Practices

### 1. Always Authenticate Production Webhooks

Never expose unauthenticated webhooks in production.

### 2. Use Strong Secrets

```bash
# Generate strong secret
openssl rand -hex 32
```

### 3. Store Secrets Securely

Use environment variables or credential manager:

```bash
export WEBHOOK_SECRET="generated-secret"
```

### 4. Use HTTPS

Always use HTTPS in production:

```
https://your-domain.com/webhook/secure
```

### 5. Validate Timestamps

Prevent replay attacks:

```javascript
const timestamp = $json.headers['x-timestamp'];
const now = Date.now();
const fiveMinutes = 5 * 60 * 1000;

if (Math.abs(now - timestamp) > fiveMinutes) {
  throw new Error('Request too old');
}
```

### 6. Rate Limit

Prevent abuse:

```yaml
webhooks:
  rateLimit:
    enabled: true
    requests: 100
    window: 60
```

### 7. Log and Monitor

Track webhook activity:

```javascript
console.log({
  path: $json.webhookUrl,
  method: $json.httpMethod,
  ip: $json.headers['x-forwarded-for'],
  timestamp: new Date().toISOString()
});
```

## Error Responses

| Status | Description |
|--------|-------------|
| 401 | Authentication failed |
| 403 | IP not allowed |
| 429 | Rate limit exceeded |

## Testing Authentication

### Test Header Auth

```bash
# Should succeed
curl -X POST http://localhost:8080/webhook/secure \
  -H "X-API-Key: correct-key"

# Should fail (401)
curl -X POST http://localhost:8080/webhook/secure \
  -H "X-API-Key: wrong-key"
```

### Test HMAC

```bash
# Generate signature
BODY='{"test": "data"}'
SIGNATURE=$(echo -n "$BODY" | openssl dgst -sha256 -hmac "your-secret" | cut -d' ' -f2)

# Send request
curl -X POST http://localhost:8080/webhook/hmac \
  -H "X-Signature: sha256=$SIGNATURE" \
  -H "Content-Type: application/json" \
  -d "$BODY"
```
