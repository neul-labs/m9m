# Migration from n8n

Migrate your workflows and data from n8n to m9m.

## Compatibility

m9m provides **95% compatibility** with n8n:

| Feature | Compatibility |
|---------|---------------|
| Workflows | Full |
| Expressions | Full |
| Nodes | 95%+ |
| Credentials | Full |
| Webhooks | Full |
| Variables | Full |

## Migration Steps

### 1. Export from n8n

#### Export Workflows

```bash
# Export all workflows
curl -X GET http://n8n-host:5678/api/v1/workflows \
  -H "X-N8N-API-KEY: your-api-key" \
  -o workflows.json

# Export specific workflow
curl -X GET http://n8n-host:5678/api/v1/workflows/<id> \
  -H "X-N8N-API-KEY: your-api-key" \
  -o workflow.json
```

Or via n8n UI:
1. Go to **Workflows**
2. Select workflow → **...** → **Download**

#### Export Credentials

```bash
curl -X GET http://n8n-host:5678/api/v1/credentials \
  -H "X-N8N-API-KEY: your-api-key" \
  -o credentials.json
```

!!! warning "Credential Security"
    Credential secrets are not exported by default. You'll need to re-enter secrets in m9m.

### 2. Prepare m9m

```bash
# Start m9m
m9m serve

# Or with Docker
docker run -d -p 8080:8080 m9m/m9m:latest
```

### 3. Import Workflows

```bash
# Import single workflow
m9m workflow import workflow.json

# Import multiple workflows
for f in workflows/*.json; do
  m9m workflow import "$f"
done
```

Or via API:
```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @workflow.json
```

### 4. Configure Credentials

Re-create credentials with your secrets:

```bash
m9m credentials create "Slack Bot" \
  --type slackApi \
  --data '{"accessToken": "xoxb-your-token"}'
```

### 5. Test Workflows

```bash
# Validate workflow
m9m validate workflow.json

# Test execution
m9m execute workflow.json --skip-trigger
```

### 6. Activate Workflows

```bash
m9m workflow activate <id>
```

## Node Compatibility

### Fully Compatible Nodes

- **Core**: Start, Set, IF, Switch, Merge, Split In Batches
- **Transform**: Filter, Sort, Rename Keys, Item Lists
- **HTTP**: HTTP Request, Webhook
- **Code**: Code (JavaScript), Function
- **Database**: PostgreSQL, MySQL, MongoDB, Redis
- **Messaging**: Slack, Discord, Email (SMTP)
- **Cloud**: AWS S3, AWS Lambda, GCP Storage
- **AI**: OpenAI, Anthropic

### Nodes with Differences

| Node | Compatibility | Notes |
|------|---------------|-------|
| Execute Command | Partial | Different execution environment |
| SSH | Partial | Requires additional setup |
| FTP | Partial | Use SFTP for better support |

### Unsupported Nodes

Some specialized n8n nodes may not be available. Use alternatives:

| n8n Node | m9m Alternative |
|----------|-----------------|
| n8n-nodes-custom | Create custom Go node |

## Expression Migration

Expressions are fully compatible:

```javascript
// These work identically in m9m
{{ $json.field }}
{{ $input.first().json.data }}
{{ $node["HTTP Request"].json.statusCode }}
{{ $now.format('YYYY-MM-DD') }}
```

## Credential Migration

### Automatic Migration

```bash
m9m migrate-credentials \
  --from-n8n http://n8n-host:5678 \
  --api-key your-n8n-api-key
```

This imports credential metadata. Secrets must be re-entered.

### Manual Migration

1. List n8n credentials
2. Create equivalent in m9m
3. Update workflows to reference new credential IDs

## Webhook Migration

Webhook URLs change format:

| n8n | m9m |
|-----|-----|
| `n8n-host:5678/webhook/<path>` | `m9m-host:8080/webhook/<path>` |
| `n8n-host:5678/webhook-test/<path>` | `m9m-host:8080/webhook-test/<path>` |

Update your integrations with new URLs.

## Database Migration

### Execution History

Export execution history if needed:

```bash
# Export from n8n
curl -X GET "http://n8n-host:5678/api/v1/executions?limit=1000" \
  -H "X-N8N-API-KEY: your-api-key" \
  -o executions.json

# Import to m9m (optional)
m9m executions import executions.json
```

### Workflow Data

Workflow definitions are fully compatible:

```bash
# Direct import works
m9m workflow import n8n-workflow.json
```

## Configuration Mapping

### Environment Variables

| n8n | m9m |
|-----|-----|
| `N8N_PORT` | `M9M_PORT` |
| `N8N_HOST` | `M9M_HOST` |
| `DB_TYPE` | `M9M_DB_TYPE` |
| `DB_POSTGRESDB_HOST` | `M9M_DB_URL` |
| `EXECUTIONS_DATA_SAVE_ON_SUCCESS` | `M9M_SAVE_SUCCESS_DATA` |
| `QUEUE_BULL_REDIS_HOST` | `M9M_QUEUE_URL` |
| `N8N_ENCRYPTION_KEY` | `M9M_ENCRYPTION_KEY` |

### Queue Migration

| n8n | m9m |
|-----|-----|
| Bull (Redis) | Redis |
| - | RabbitMQ (additional option) |

## Validation Script

Use this script to validate migration:

```bash
#!/bin/bash
# migration-validate.sh

echo "Validating m9m migration..."

# Check workflows
echo "Checking workflows..."
WORKFLOWS=$(m9m workflow list --format json | jq length)
echo "  Imported workflows: $WORKFLOWS"

# Check credentials
echo "Checking credentials..."
CREDS=$(m9m credentials list --format json | jq length)
echo "  Imported credentials: $CREDS"

# Test sample workflow
echo "Testing sample workflow..."
m9m execute test-workflow.json --skip-trigger
if [ $? -eq 0 ]; then
    echo "  Sample workflow: PASSED"
else
    echo "  Sample workflow: FAILED"
fi

# Check webhooks
echo "Checking webhook endpoints..."
curl -s http://localhost:8080/webhook/test > /dev/null
if [ $? -eq 0 ]; then
    echo "  Webhooks: ACTIVE"
else
    echo "  Webhooks: INACTIVE"
fi

echo "Migration validation complete."
```

## Rollback Plan

If migration fails:

1. Keep n8n running during migration
2. Test m9m thoroughly before switching
3. Use DNS/load balancer to switch traffic
4. Keep n8n available for 30 days as fallback

```bash
# Quick rollback
# 1. Update DNS/load balancer to point to n8n
# 2. Deactivate m9m workflows
m9m workflow list | xargs -I {} m9m workflow deactivate {}
```

## Performance Comparison

After migration, you should see:

| Metric | Improvement |
|--------|-------------|
| Startup time | 6x faster |
| Execution speed | 5-10x faster |
| Memory usage | 70% less |
| Container size | 75% smaller |

## Troubleshooting

### Workflow Import Fails

```bash
# Validate JSON
jq . workflow.json

# Check for unsupported nodes
m9m validate workflow.json
```

### Credentials Not Working

1. Verify credential type matches
2. Re-enter secrets
3. Test credential connection

### Expression Errors

1. Check for n8n-specific functions
2. Verify variable references
3. Test expressions individually

## Next Steps

- [Configuration](configuration.md) - Configure m9m
- [Troubleshooting](troubleshooting.md) - Common issues
- [Production](../deployment/production.md) - Production deployment
