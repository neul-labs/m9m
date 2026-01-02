# Variables & Environments

Variables allow you to configure workflows differently across environments without changing the workflow definition.

## Variable Types

### Environment Variables

System-level variables accessible via `$env`:

```javascript
{{ $env.API_URL }}
{{ $env.MAX_RETRIES }}
```

### Workflow Variables

Variables scoped to a specific workflow:

```javascript
{{ $vars.batchSize }}
{{ $vars.notificationEmail }}
```

### Global Variables

Variables available across all workflows:

```javascript
{{ $globals.companyName }}
{{ $globals.supportEmail }}
```

## Setting Variables

### Environment Variables

Set via shell environment:

```bash
export M9M_VAR_API_URL="https://api.example.com"
export M9M_VAR_MAX_RETRIES="3"

m9m serve
```

Or in configuration file:

```yaml
# config.yaml
variables:
  API_URL: "https://api.example.com"
  MAX_RETRIES: 3
  DEBUG: false
```

### Global Variables (CLI)

```bash
m9m variables set companyName "Acme Corp"
m9m variables set supportEmail "support@acme.com"
```

### Global Variables (API)

```bash
curl -X PUT http://localhost:8080/api/v1/variables/companyName \
  -H "Content-Type: application/json" \
  -d '{"value": "Acme Corp"}'
```

### Workflow Variables

Define in workflow settings:

```json
{
  "name": "My Workflow",
  "settings": {
    "variables": {
      "batchSize": 100,
      "retryCount": 3,
      "targetEmail": "team@example.com"
    }
  }
}
```

## Environment Profiles

Create different configurations for different environments.

### Define Profiles

```yaml
# environments/development.yaml
variables:
  API_URL: "http://localhost:3000"
  DATABASE_URL: "postgres://localhost/dev"
  LOG_LEVEL: "debug"
  SEND_EMAILS: false

# environments/production.yaml
variables:
  API_URL: "https://api.example.com"
  DATABASE_URL: "postgres://prod-db/app"
  LOG_LEVEL: "info"
  SEND_EMAILS: true
```

### Use Profiles

```bash
# Development
m9m serve --env development

# Production
m9m serve --env production
```

## Using Variables in Workflows

### In Expressions

```javascript
// Environment variable
{{ $env.API_URL }}/users

// Workflow variable
{{ $vars.batchSize }}

// With defaults
{{ $env.TIMEOUT ?? 30000 }}
```

### In Node Parameters

```json
{
  "type": "n8n-nodes-base.httpRequest",
  "parameters": {
    "url": "={{ $env.API_URL }}/data",
    "timeout": "={{ $vars.timeout ?? 5000 }}"
  }
}
```

### In Credentials

```json
{
  "type": "postgres",
  "data": {
    "host": "={{ $env.DB_HOST }}",
    "password": "={{ $env.DB_PASSWORD }}"
  }
}
```

## Variable Precedence

Variables are resolved in this order (later overrides earlier):

1. Default values in workflow
2. Global variables
3. Environment variables
4. Workflow-specific variables
5. Runtime overrides

## Secret Management

### Marking Secrets

Mark sensitive variables as secrets:

```yaml
variables:
  API_KEY:
    value: "sk-secret-key"
    secret: true
```

Secrets are:
- Masked in logs
- Encrypted at rest
- Not exposed in API responses

### External Secret Managers

Integrate with external secret managers:

```yaml
secrets:
  provider: vault
  config:
    address: "https://vault.example.com"
    token: "${VAULT_TOKEN}"

variables:
  API_KEY:
    source: vault
    path: "secret/data/api-keys"
    key: "primary"
```

## Variable Validation

Define validation rules:

```yaml
variables:
  PORT:
    value: 8080
    type: integer
    min: 1024
    max: 65535

  EMAIL:
    value: "admin@example.com"
    type: string
    pattern: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"

  LOG_LEVEL:
    value: "info"
    type: string
    enum: ["debug", "info", "warn", "error"]
```

## Managing Variables

### List Variables

```bash
m9m variables list
```

```
NAME              VALUE                    SCOPE
API_URL           https://api.example.com  environment
MAX_RETRIES       3                        environment
companyName       Acme Corp                global
```

### Get Variable

```bash
m9m variables get API_URL
```

### Set Variable

```bash
m9m variables set KEY "value"
m9m variables set KEY "value" --secret
```

### Delete Variable

```bash
m9m variables delete KEY
```

### Export Variables

```bash
m9m variables export > variables.yaml
```

### Import Variables

```bash
m9m variables import variables.yaml
```

## Dynamic Variables

### Computed Variables

Define variables that compute from others:

```yaml
variables:
  BASE_URL: "https://api.example.com"
  API_VERSION: "v2"
  FULL_API_URL:
    computed: true
    expression: "${BASE_URL}/${API_VERSION}"
```

### Runtime Variables

Set variables at execution time:

```bash
m9m execute workflow.json \
  --var batchSize=500 \
  --var targetEnv=staging
```

## Best Practices

### Naming Conventions

- Use SCREAMING_SNAKE_CASE for environment variables
- Use camelCase for workflow variables
- Prefix related variables: `DB_HOST`, `DB_PORT`, `DB_NAME`

### Organization

```yaml
# Group related variables
database:
  host: localhost
  port: 5432
  name: app

api:
  url: https://api.example.com
  timeout: 30000

features:
  enableCache: true
  debugMode: false
```

### Documentation

Document variables in your workflow:

```json
{
  "settings": {
    "variables": {
      "batchSize": {
        "value": 100,
        "description": "Number of items to process per batch"
      }
    }
  }
}
```

### Security

- Never hardcode secrets in workflows
- Use secret managers for production
- Rotate credentials regularly
- Audit variable access

## Troubleshooting

### Variable Not Found

```
Error: Variable 'API_URL' not found
```

**Solutions:**
- Verify variable is defined
- Check variable scope
- Ensure environment profile is loaded

### Type Mismatch

```
Error: Expected integer for 'PORT', got string
```

**Solutions:**
- Use correct type in expression: `{{ parseInt($env.PORT) }}`
- Fix variable definition type

## Next Steps

- [Error Handling](error-handling.md) - Handle configuration errors
- [Deployment](../deployment/production.md) - Production configuration
- [Configuration Reference](../reference/configuration.md) - Full config options
