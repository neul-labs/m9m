# Variables and Environments

The m9m variables and environments system provides centralized configuration management with encryption support, allowing you to manage secrets, environment-specific settings, and workflow-specific variables.

## Overview

The system provides three levels of variable scoping:
- **Global Variables**: Accessible across all workflows and environments
- **Environment Variables**: Scoped to specific environments (dev, staging, production)
- **Workflow Variables**: Specific to individual workflows

Variables are resolved with priority: Workflow > Environment > Global

## Key Features

- 🔐 **AES-256-GCM encryption** for sensitive values
- 🌍 **Multiple environments** support (dev, staging, production, etc.)
- 🎯 **Priority-based resolution** (workflow overrides environment overrides global)
- 🔄 **Hot reload** - changes take effect immediately
- 💾 **Persistent storage** compatible with all backends
- 🏷️ **Tags and metadata** for organization
- 🔍 **Search and filtering** capabilities

## Architecture

### Component Structure

```
internal/variables/
├── variable.go           # Data models and types
├── variable_storage.go   # Storage layer (Memory & Persistent)
├── variable_manager.go   # Business logic and encryption
└── variable_handler.go   # HTTP API endpoints
```

### Data Models

#### Variable
```go
type Variable struct {
    ID          string                 // Unique identifier
    Key         string                 // Variable name (e.g., "API_KEY")
    Value       string                 // Variable value (encrypted if sensitive)
    Type        VariableType           // global, environment, workflow
    Description string                 // Human-readable description
    Encrypted   bool                   // Whether value is encrypted
    Tags        []string               // Categorization tags
    Metadata    map[string]interface{} // Additional metadata
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

#### Environment
```go
type Environment struct {
    ID          string                 // Unique identifier
    Name        string                 // Display name (e.g., "Production")
    Key         string                 // Environment key (e.g., "prod")
    Description string                 // Description
    Variables   map[string]string      // Environment-specific variables
    Active      bool                   // Is this the active environment
    Metadata    map[string]interface{} // Additional metadata
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

## API Reference

### Variables API

#### List Variables
```bash
GET /api/v1/variables?type={type}&search={query}&limit={n}&offset={n}

Response:
{
  "data": [...],
  "total": 10,
  "count": 10,
  "limit": 50,
  "offset": 0
}
```

#### Create Variable
```bash
POST /api/v1/variables

Request:
{
  "key": "DATABASE_URL",
  "value": "postgres://localhost:5432/mydb",
  "type": "global",
  "description": "Database connection string",
  "encrypted": true,
  "tags": ["database", "config"]
}

Response:
{
  "id": "var_1234567890",
  "key": "DATABASE_URL",
  "value": "encrypted_base64_string",
  "type": "global",
  "description": "Database connection string",
  "encrypted": true,
  "tags": ["database", "config"],
  "createdAt": "2025-11-10T14:00:00Z",
  "updatedAt": "2025-11-10T14:00:00Z"
}
```

#### Get Variable
```bash
GET /api/v1/variables/{id}?decrypt=true

Response:
{
  "id": "var_1234567890",
  "key": "DATABASE_URL",
  "value": "postgres://localhost:5432/mydb",  # Decrypted if decrypt=true
  "type": "global",
  ...
}
```

#### Update Variable
```bash
PUT /api/v1/variables/{id}

Request:
{
  "value": "new_value",
  "description": "Updated description",
  "tags": ["updated", "config"]
}

Response:
{
  "id": "var_1234567890",
  ...updated fields...
}
```

#### Delete Variable
```bash
DELETE /api/v1/variables/{id}

Response: 204 No Content
```

### Environments API

#### List Environments
```bash
GET /api/v1/environments

Response:
{
  "data": [
    {
      "id": "env_123",
      "name": "Production",
      "key": "prod",
      "description": "Production environment",
      "variables": {
        "LOG_LEVEL": "error",
        "RATE_LIMIT": "1000"
      },
      "active": true,
      "createdAt": "2025-11-10T14:00:00Z",
      "updatedAt": "2025-11-10T14:00:00Z"
    }
  ],
  "total": 3,
  "count": 3
}
```

#### Create Environment
```bash
POST /api/v1/environments

Request:
{
  "name": "Staging",
  "key": "staging",
  "description": "Staging environment for testing",
  "variables": {
    "LOG_LEVEL": "debug",
    "RATE_LIMIT": "500"
  },
  "active": false
}

Response:
{
  "id": "env_456",
  "name": "Staging",
  "key": "staging",
  ...
}
```

#### Update Environment
```bash
PUT /api/v1/environments/{id}

Request:
{
  "active": true,
  "variables": {
    "LOG_LEVEL": "info",
    "NEW_VAR": "value"
  }
}

Response:
{
  "id": "env_456",
  ...updated fields...
}
```

#### Delete Environment
```bash
DELETE /api/v1/environments/{id}

Response: 204 No Content

Note: Cannot delete the active environment
```

### Workflow Variables API

#### Get Workflow Variables
```bash
GET /api/v1/workflows/{workflowId}/variables

Response:
{
  "workflowId": "workflow_123",
  "variables": {
    "RETRY_COUNT": "3",
    "TIMEOUT": "30"
  }
}
```

#### Save Workflow Variables
```bash
POST /api/v1/workflows/{workflowId}/variables

Request:
{
  "RETRY_COUNT": "5",
  "TIMEOUT": "60",
  "NEW_VAR": "value"
}

Response:
{
  "success": true,
  "workflowId": "workflow_123",
  "variables": {
    "RETRY_COUNT": "5",
    "TIMEOUT": "60",
    "NEW_VAR": "value"
  }
}
```

## Usage Examples

### Example 1: Managing API Keys

**Step 1: Create an encrypted global variable for API key**
```bash
curl -X POST http://localhost:8080/api/v1/variables \
  -H "Content-Type: application/json" \
  -d '{
    "key": "OPENAI_API_KEY",
    "value": "sk-1234567890abcdef",
    "type": "global",
    "description": "OpenAI API key",
    "encrypted": true,
    "tags": ["api", "secret", "ai"]
  }'
```

**Step 2: Use in workflow**
The variable can be referenced as `$vars.OPENAI_API_KEY` in workflows.

### Example 2: Multiple Environments

**Step 1: Create development environment**
```bash
curl -X POST http://localhost:8080/api/v1/environments \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Development",
    "key": "dev",
    "description": "Development environment",
    "active": true,
    "variables": {
      "API_BASE_URL": "http://localhost:3000",
      "LOG_LEVEL": "debug",
      "ENABLE_CACHE": "false"
    }
  }'
```

**Step 2: Create production environment**
```bash
curl -X POST http://localhost:8080/api/v1/environments \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Production",
    "key": "prod",
    "description": "Production environment",
    "active": false,
    "variables": {
      "API_BASE_URL": "https://api.example.com",
      "LOG_LEVEL": "error",
      "ENABLE_CACHE": "true"
    }
  }'
```

**Step 3: Switch environments**
```bash
# Activate production
curl -X PUT http://localhost:8080/api/v1/environments/env_prod_id \
  -H "Content-Type: application/json" \
  -d '{"active": true}'
```

### Example 3: Workflow-Specific Variables

**Set variables for a specific workflow**
```bash
curl -X POST http://localhost:8080/api/v1/workflows/workflow_123/variables \
  -H "Content-Type: application/json" \
  -d '{
    "MAX_RETRIES": "5",
    "TIMEOUT_SECONDS": "30",
    "NOTIFY_ON_FAILURE": "true"
  }'
```

### Example 4: Variable Priority

Given this configuration:
```yaml
Global:
  API_TIMEOUT: "60"
  MAX_RETRIES: "3"

Environment (prod):
  API_TIMEOUT: "30"

Workflow (workflow_123):
  MAX_RETRIES: "5"
```

When workflow_123 runs in prod environment:
- `API_TIMEOUT` = `"30"` (from environment, overrides global)
- `MAX_RETRIES` = `"5"` (from workflow, overrides environment and global)

## Encryption

### How It Works

Variables marked as `encrypted: true` are encrypted using **AES-256-GCM** before storage:

1. **Encryption Key**: 32-byte key (configurable via `--encryption-key` flag)
2. **Algorithm**: AES-256-GCM (authenticated encryption)
3. **Encoding**: Base64-encoded ciphertext
4. **Nonce**: Random nonce generated per encryption

### Configuration

**Set encryption key via CLI:**
```bash
./m9m --encryption-key "your-32-byte-encryption-key-here"
```

**Important**:
- Key must be 32 bytes for AES-256
- Keys are padded/truncated automatically
- **NEVER commit encryption keys to version control**
- Use environment variables or secure key management systems

### Security Best Practices

1. **Always encrypt sensitive data**:
   - API keys
   - Passwords
   - Database credentials
   - OAuth tokens
   - Private keys

2. **Use environment variables for the encryption key**:
   ```bash
   export ENCRYPTION_KEY="$(openssl rand -base64 32)"
   ./m9m --encryption-key "$ENCRYPTION_KEY"
   ```

3. **Rotate encryption keys periodically**:
   - Re-encrypt all encrypted variables with new key
   - Keep old key temporarily for decryption

4. **Limit access to encrypted variables**:
   - Only decrypt when necessary
   - Use `decrypt=false` (default) in API calls when possible

## Variable Resolution

### Resolution Order

When a workflow requests a variable, the system checks in this order:

1. **Workflow Variables** (highest priority)
2. **Environment Variables** (active environment)
3. **Global Variables** (lowest priority)

### Example Resolution

**Configuration:**
```javascript
// Global
{
  "DATABASE_HOST": "global-db.example.com",
  "DATABASE_PORT": "5432",
  "TIMEOUT": "60"
}

// Environment: production
{
  "DATABASE_HOST": "prod-db.example.com",
  "TIMEOUT": "30"
}

// Workflow: critical-workflow
{
  "TIMEOUT": "10"
}
```

**Resolution for critical-workflow in production:**
```javascript
{
  "DATABASE_HOST": "prod-db.example.com",  // From environment
  "DATABASE_PORT": "5432",                 // From global
  "TIMEOUT": "10"                          // From workflow
}
```

## Integration with Workflows

### Using Variables in Node Parameters

Variables can be referenced using the `$vars` prefix:

```json
{
  "nodes": [
    {
      "name": "HTTP Request",
      "type": "n8n-nodes-base.httpRequest",
      "parameters": {
        "url": "{{ $vars.API_BASE_URL }}/users",
        "authentication": "genericCredentialType",
        "genericAuthType": "httpHeaderAuth",
        "httpHeaderAuth": {
          "name": "Authorization",
          "value": "Bearer {{ $vars.API_KEY }}"
        }
      }
    }
  ]
}
```

### Variable Context in Expression Evaluator

The expression evaluator has access to:
```javascript
$vars = {
  ...global_variables,
  ...environment_variables,
  ...workflow_variables
}
```

Example expressions:
```javascript
{{ $vars.API_BASE_URL + '/api/v1/users' }}
{{ $vars.MAX_RETRIES > 0 ? 'retry' : 'skip' }}
{{ $vars.ENABLE_LOGGING === 'true' }}
```

## Storage Backends

The variables system supports all m9m storage backends:

### Memory Storage (Default)
```bash
./m9m --db memory
```
- Fast, ephemeral
- Variables lost on restart
- Good for development/testing

### BadgerDB Storage
```bash
./m9m --db badger --data-dir ./data
```
- Persistent, embedded
- Variables survive restarts
- Good for single-node deployments

### PostgreSQL Storage
```bash
./m9m --db postgres --db-url "postgres://user:pass@localhost/n8n"
```
- Persistent, relational
- Scalable, production-ready
- Good for multi-node deployments

### SQLite Storage
```bash
./m9m --db sqlite --db-url ./n8n.db
```
- Persistent, file-based
- Simple, reliable
- Good for small deployments

## Management Tools

### List All Variables
```bash
# List all
curl http://localhost:8080/api/v1/variables

# Filter by type
curl http://localhost:8080/api/v1/variables?type=global

# Search
curl http://localhost:8080/api/v1/variables?search=API

# Pagination
curl http://localhost:8080/api/v1/variables?limit=10&offset=20
```

### Export Variables
```bash
# Export all variables
curl http://localhost:8080/api/v1/variables | jq '.data' > variables.json
```

### Import Variables
```bash
# Import from JSON
cat variables.json | jq -c '.[]' | while read var; do
  curl -X POST http://localhost:8080/api/v1/variables \
    -H "Content-Type: application/json" \
    -d "$var"
done
```

### Backup and Restore

**Backup:**
```bash
# Backup all variables
curl http://localhost:8080/api/v1/variables | jq '.data' > backup-variables.json

# Backup all environments
curl http://localhost:8080/api/v1/environments | jq '.data' > backup-environments.json
```

**Restore:**
```bash
# Restore variables
cat backup-variables.json | jq -c '.[]' | while read var; do
  curl -X POST http://localhost:8080/api/v1/variables \
    -H "Content-Type: application/json" \
    -d "$var"
done
```

## Best Practices

### 1. Naming Conventions

Use consistent naming:
```bash
# Good
API_BASE_URL
DATABASE_HOST
MAX_RETRY_COUNT
ENABLE_DEBUG_LOGGING

# Avoid
apiUrl
db_host
maxRetries
debug
```

### 2. Variable Organization

Use tags for categorization:
```json
{
  "key": "SLACK_WEBHOOK",
  "tags": ["integration", "notifications", "slack"]
}
```

### 3. Environment Strategy

Create environments for each stage:
- `dev` - Development
- `staging` - Staging/QA
- `prod` - Production
- `local` - Local development

### 4. Encryption Strategy

Encrypt sensitive data:
```bash
# Always encrypt
- API keys
- Passwords
- Tokens
- Credentials

# Usually don't need encryption
- URLs
- Timeouts
- Feature flags
- Log levels
```

### 5. Documentation

Add descriptions to all variables:
```json
{
  "key": "MAX_RETRIES",
  "value": "3",
  "description": "Maximum number of retry attempts for failed HTTP requests"
}
```

## Troubleshooting

### Issue: Variable Not Found
**Cause**: Variable doesn't exist or wrong type specified
**Solution**: Check variable key and type match exactly

### Issue: Decryption Failed
**Cause**: Wrong encryption key or corrupted data
**Solution**: Ensure encryption key matches the one used for encryption

### Issue: Cannot Delete Active Environment
**Cause**: Trying to delete the currently active environment
**Solution**: Activate a different environment first

### Issue: Variable Not Resolving in Workflow
**Cause**: Wrong variable reference syntax
**Solution**: Use `{{ $vars.VARIABLE_NAME }}` syntax

## Performance Considerations

- Variables are cached in memory for fast access
- Encryption/decryption happens on-demand
- Storage operations are optimized with indexing
- Large numbers of variables (1000+) have minimal performance impact

## Security Considerations

1. **Encryption Key Security**
   - Never commit to version control
   - Rotate periodically
   - Use key management systems in production

2. **Access Control**
   - Implement authentication (JWT)
   - Restrict API access to authorized users
   - Audit variable access

3. **Network Security**
   - Use HTTPS in production
   - Encrypt variables at rest
   - Secure storage backend connections

## Future Enhancements

Planned features:
- Variable validation (regex, types)
- Variable inheritance between environments
- Audit logging for variable changes
- Variable import/export CLI tool
- Variable templates
- Environment cloning
- Variable versioning

## API Integration Examples

### JavaScript/Node.js
```javascript
const axios = require('axios');

// Create variable
async function createVariable(key, value, encrypted = false) {
  const response = await axios.post('http://localhost:8080/api/v1/variables', {
    key,
    value,
    type: 'global',
    encrypted
  });
  return response.data;
}

// Get variable (decrypted)
async function getVariable(id) {
  const response = await axios.get(
    `http://localhost:8080/api/v1/variables/${id}?decrypt=true`
  );
  return response.data;
}
```

### Python
```python
import requests

# Create environment
def create_environment(name, key, variables, active=False):
    response = requests.post(
        'http://localhost:8080/api/v1/environments',
        json={
            'name': name,
            'key': key,
            'variables': variables,
            'active': active
        }
    )
    return response.json()

# List all variables
def list_variables(var_type=None):
    params = {'type': var_type} if var_type else {}
    response = requests.get(
        'http://localhost:8080/api/v1/variables',
        params=params
    )
    return response.json()['data']
```

## Additional Resources

- [Workflow API Documentation](./WORKFLOW_API.md)
- [Version Control Documentation](./WORKFLOW_VERSIONS.md)
- [Authentication Documentation](./AUTHENTICATION.md)
- [Webhook Documentation](./WEBHOOKS.md)

## Support

For issues or questions:
- GitHub Issues: https://github.com/neul-labs/m9m/issues
- Documentation: https://github.com/neul-labs/m9m/docs
