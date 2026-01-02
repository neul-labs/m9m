# Credentials

Credentials securely store authentication information for connecting to external services.

## Overview

m9m encrypts all credentials at rest and provides:

- Secure storage with AES-256 encryption
- Fine-grained access control
- Credential sharing across workflows
- Multiple credential types per service

## Credential Types

### API Key

Simple key-based authentication:

```json
{
  "type": "apiKey",
  "data": {
    "apiKey": "your-api-key",
    "headerName": "X-API-Key"
  }
}
```

### OAuth2

OAuth 2.0 authentication flow:

```json
{
  "type": "oauth2",
  "data": {
    "clientId": "your-client-id",
    "clientSecret": "your-client-secret",
    "accessToken": "obtained-access-token",
    "refreshToken": "obtained-refresh-token"
  }
}
```

### Basic Auth

Username and password:

```json
{
  "type": "basicAuth",
  "data": {
    "username": "user",
    "password": "password"
  }
}
```

### Database

Database connection credentials:

```json
{
  "type": "postgres",
  "data": {
    "host": "localhost",
    "port": 5432,
    "database": "mydb",
    "user": "dbuser",
    "password": "dbpassword",
    "ssl": true
  }
}
```

## Creating Credentials

### Via Web UI

1. Navigate to **Settings** → **Credentials**
2. Click **Add Credential**
3. Select the credential type
4. Fill in the required fields
5. Click **Save**

### Via CLI

```bash
# Create from inline JSON
m9m credentials create my-api \
  --type apiKey \
  --data '{"apiKey": "abc123"}'

# Create from file
m9m credentials create my-database \
  --type postgres \
  --file credentials.json
```

### Via API

```bash
curl -X POST http://localhost:8080/api/v1/credentials \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My API Key",
    "type": "apiKey",
    "data": {
      "apiKey": "your-key-here"
    }
  }'
```

## Using Credentials in Workflows

Reference credentials in node configuration:

```json
{
  "id": "http-request",
  "type": "n8n-nodes-base.httpRequest",
  "credentials": {
    "httpHeaderAuth": {
      "id": "1",
      "name": "My API Key"
    }
  },
  "parameters": {
    "url": "https://api.example.com/data"
  }
}
```

## Service-Specific Credentials

### Slack

```json
{
  "type": "slackApi",
  "data": {
    "accessToken": "xoxb-your-bot-token"
  }
}
```

### GitHub

```json
{
  "type": "githubApi",
  "data": {
    "accessToken": "ghp_your-personal-access-token"
  }
}
```

### AWS

```json
{
  "type": "aws",
  "data": {
    "accessKeyId": "AKIAIOSFODNN7EXAMPLE",
    "secretAccessKey": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
    "region": "us-east-1"
  }
}
```

### OpenAI

```json
{
  "type": "openAiApi",
  "data": {
    "apiKey": "sk-your-openai-key"
  }
}
```

### PostgreSQL

```json
{
  "type": "postgres",
  "data": {
    "host": "db.example.com",
    "port": 5432,
    "database": "production",
    "user": "app_user",
    "password": "secure-password",
    "ssl": true,
    "sslCertificate": "-----BEGIN CERTIFICATE-----..."
  }
}
```

## Managing Credentials

### List Credentials

```bash
m9m credentials list
```

```
ID    NAME              TYPE        CREATED
1     My Slack          slackApi    2024-01-10
2     Production DB     postgres    2024-01-08
3     GitHub Token      githubApi   2024-01-05
```

### Update Credentials

```bash
m9m credentials update 1 \
  --data '{"accessToken": "new-token"}'
```

### Delete Credentials

```bash
m9m credentials delete 1
```

### Test Credentials

```bash
m9m credentials test 1
```

## OAuth2 Setup

For OAuth2 credentials, m9m provides a callback URL:

### 1. Configure OAuth App

In the service's developer console:

- **Redirect URI**: `https://your-m9m-host/api/v1/oauth2/callback`

### 2. Create Credential

```bash
m9m credentials create my-oauth \
  --type oauth2 \
  --data '{
    "clientId": "your-client-id",
    "clientSecret": "your-client-secret",
    "authorizationUrl": "https://service.com/oauth/authorize",
    "tokenUrl": "https://service.com/oauth/token",
    "scope": "read write"
  }'
```

### 3. Authorize

```bash
m9m credentials authorize my-oauth
```

This opens a browser for the OAuth flow.

## Security Best Practices

### Environment Variables

Store sensitive values in environment variables:

```bash
export SLACK_TOKEN="xoxb-your-token"
```

Reference in credentials:

```json
{
  "type": "slackApi",
  "data": {
    "accessToken": "={{ $env.SLACK_TOKEN }}"
  }
}
```

### Encryption Key

Set a custom encryption key:

```bash
export M9M_ENCRYPTION_KEY="your-32-character-encryption-key"
```

### Credential Rotation

Regularly rotate credentials:

```bash
# Update with new value
m9m credentials update 1 --data '{"apiKey": "new-key"}'

# Or rotate automatically
m9m credentials rotate 1
```

### Access Control

Limit credential access by workflow:

```json
{
  "name": "Production DB",
  "type": "postgres",
  "allowedWorkflows": ["wf-001", "wf-002"]
}
```

## Importing and Exporting

### Export (Encrypted)

```bash
m9m credentials export --output credentials.enc
```

### Import

```bash
m9m credentials import --input credentials.enc
```

!!! warning "Security"
    Never export credentials to unencrypted files or commit them to version control.

## Troubleshooting

### Connection Failed

1. Verify credential values are correct
2. Check network connectivity
3. Confirm service is accessible
4. Review firewall rules

### Authentication Error

1. Check token hasn't expired
2. Verify scopes are sufficient
3. Confirm credentials match environment (prod vs dev)

### OAuth Token Expired

```bash
# Refresh OAuth token
m9m credentials refresh 1
```

## Next Steps

- [Variables](variables.md) - Environment configuration
- [Error Handling](error-handling.md) - Handle authentication failures
- [Deployment](../deployment/production.md) - Production credential management
