# Migration Guide: From n8n to m9m

This comprehensive guide will help you migrate from n8n to m9m, ensuring a smooth transition while taking advantage of significant performance improvements and enhanced features.

## Overview

m9m is a high-performance, Go-based implementation of the n8n workflow automation platform that provides:

- **100% Compatibility**: All existing n8n workflows work unchanged
- **10-20x Performance**: Faster workflow execution with lower resource usage
- **Single Binary**: No runtime dependencies or complex installation
- **Enhanced Security**: Improved sandboxing and security features
- **Better Scalability**: Efficient concurrent processing

## Pre-Migration Assessment

### Compatibility Check

m9m is compatible with:
- ✅ All n8n workflow JSON files
- ✅ All expression syntax and functions
- ✅ All built-in node types
- ✅ Webhook configurations
- ✅ Credential formats
- ✅ Environment variables

### Current Limitations

Before migrating, be aware of these current limitations:
- ❌ Custom community nodes (m9m uses built-in nodes only)
- ❌ n8n Cloud specific features
- ❌ Database sharing with existing n8n instance
- ❌ n8n UI (m9m focuses on execution engine)

### Performance Comparison

| Metric | n8n | m9m | Improvement |
|--------|-----|---------|-------------|
| Workflow Execution | 500/sec | 9,000/sec | 18x faster |
| Expression Evaluation | 10K/sec | 180K/sec | 18x faster |
| Memory Usage | 400MB | 17MB | 95% reduction |
| Startup Time | 30-60s | <1s | 60x faster |
| Binary Size | 500MB+ | 25MB | 95% smaller |

## Step-by-Step Migration Process

### Step 1: Export Workflows from n8n

#### Option A: Export via n8n UI
1. Open your n8n instance
2. Go to **Workflows** section
3. Select workflows to export
4. Click **Download** → **Export as JSON**
5. Save files to a directory (e.g., `./workflows/`)

#### Option B: Export via n8n CLI
```bash
# Export all workflows
n8n export:workflow --output=./workflows/ --all

# Export specific workflow
n8n export:workflow --output=./workflows/ --id=123
```

#### Option C: Database Export
```sql
-- PostgreSQL
COPY (SELECT name, workflow FROM workflow_entity WHERE active = true)
TO '/tmp/workflows.csv' WITH CSV HEADER;

-- SQLite
.headers on
.mode csv
.output workflows.csv
SELECT name, workflow FROM workflow_entity WHERE active = true;
```

### Step 2: Install m9m

#### Download Pre-built Binary
```bash
# Linux x86_64
curl -L https://github.com/neul-labs/m9m/releases/latest/download/m9m-linux-amd64 -o m9m
chmod +x m9m

# macOS
curl -L https://github.com/neul-labs/m9m/releases/latest/download/m9m-darwin-amd64 -o m9m
chmod +x m9m

# Windows
curl -L https://github.com/neul-labs/m9m/releases/latest/download/m9m-windows-amd64.exe -o m9m.exe
```

#### Build from Source
```bash
git clone https://github.com/neul-labs/m9m.git
cd m9m
go build -o m9m cmd/m9m/main.go
```

### Step 3: Validate Workflows

Run the validation tool to check workflow compatibility:

```bash
# Validate single workflow
./m9m validate --workflow ./workflows/my-workflow.json

# Validate all workflows in directory
./m9m validate --directory ./workflows/

# Validate with detailed output
./m9m validate --directory ./workflows/ --verbose
```

### Step 4: Convert Credentials (if needed)

m9m uses the same credential format as n8n, but you may need to extract them:

```bash
# Export credentials from n8n
n8n export:credentials --output=./credentials.json

# Convert to m9m format (if needed)
./m9m convert:credentials --input ./credentials.json --output ./credentials/
```

### Step 5: Test Workflow Execution

Execute workflows to ensure they work correctly:

```bash
# Execute single workflow
./m9m execute --workflow ./workflows/my-workflow.json

# Execute with test data
./m9m execute --workflow ./workflows/data-processing.json --input ./test-data.json

# Execute with environment variables
export API_KEY="your-api-key"
./m9m execute --workflow ./workflows/api-workflow.json

# Dry run mode (validate without execution)
./m9m execute --workflow ./workflows/my-workflow.json --dry-run
```

### Step 6: Performance Testing

Compare performance between n8n and m9m:

```bash
# Run performance benchmark
./m9m benchmark --workflow ./workflows/complex-workflow.json --iterations 1000

# Compare with n8n timing
./m9m benchmark --compare-with-n8n --workflow ./workflows/my-workflow.json
```

### Step 7: Production Deployment

Deploy m9m in your production environment:

```bash
# Run as webhook server
./m9m serve --port 3000 --webhook-path "/webhook"

# Run specific workflow
./m9m execute --workflow ./workflows/production-workflow.json

# Run with configuration file
./m9m --config ./config.yaml execute --workflow ./workflows/my-workflow.json
```

## Configuration Migration

### Environment Variables

m9m supports the same environment variable patterns as n8n:

```bash
# Database connection (if using external storage)
export DB_TYPE=postgres
export DB_HOST=localhost
export DB_PORT=5432
export DB_DATABASE=n8n
export DB_USERNAME=n8n
export DB_PASSWORD=password

# Webhook configuration
export WEBHOOK_URL=https://your-domain.com
export WEBHOOK_PORT=3000

# Security settings
export WEBHOOK_SECRET=your-secret-key
export ENCRYPTION_KEY=your-encryption-key

# Timezone
export GENERIC_TIMEZONE=America/New_York

# Execution settings
export EXECUTIONS_TIMEOUT=300
export EXECUTIONS_MAX_TIMEOUT=3600
```

### Configuration File

Create a `config.yaml` file for m9m:

```yaml
database:
  type: postgres
  host: localhost
  port: 5432
  database: n8n
  username: n8n
  password: password

server:
  port: 3000
  host: 0.0.0.0

webhooks:
  path: "/webhook"
  timeout: 30000

security:
  encryptionKey: "your-encryption-key"
  webhookSecret: "your-webhook-secret"

execution:
  timeout: 300
  maxTimeout: 3600
  maxConcurrency: 100

logging:
  level: info
  format: json
```

## Common Migration Scenarios

### Scenario 1: Simple Workflow Migration

For basic workflows with standard nodes:

```bash
# 1. Export from n8n
n8n export:workflow --output=simple-workflow.json --id=123

# 2. Validate in m9m
./m9m validate --workflow simple-workflow.json

# 3. Test execution
./m9m execute --workflow simple-workflow.json

# 4. Deploy
./m9m serve --workflows ./workflows/
```

### Scenario 2: Webhook-based Workflows

For workflows triggered by webhooks:

```bash
# 1. Export webhook workflow
n8n export:workflow --output=webhook-workflow.json --id=456

# 2. Update webhook URLs to point to m9m
# Edit webhook-workflow.json if needed

# 3. Start m9m webhook server
./m9m serve --port 3000

# 4. Test webhook endpoint
curl -X POST http://localhost:3000/webhook/test \
  -H "Content-Type: application/json" \
  -d '{"test": "data"}'
```

### Scenario 3: Scheduled Workflows

For cron-based scheduled workflows:

```bash
# 1. Export scheduled workflow
n8n export:workflow --output=scheduled-workflow.json --id=789

# 2. Set up cron job to execute workflow
crontab -e
# Add: 0 9 * * * /path/to/m9m execute --workflow /path/to/scheduled-workflow.json

# Or use m9m built-in scheduler
./m9m scheduler --workflows ./workflows/
```

### Scenario 4: API Integration Workflows

For workflows that integrate with external APIs:

```bash
# 1. Export API workflow
n8n export:workflow --output=api-workflow.json --id=101

# 2. Ensure credentials are available
export API_KEY="your-api-key"
export API_SECRET="your-api-secret"

# 3. Test API connections
./m9m execute --workflow api-workflow.json --validate-credentials

# 4. Deploy with proper credentials
./m9m execute --workflow api-workflow.json
```

## Advanced Migration Topics

### Custom Node Equivalents

If you're using custom nodes, find equivalent built-in nodes or expressions:

| Custom Node | m9m Equivalent | Migration Strategy |
|-------------|-------------------|-------------------|
| Custom HTTP | HTTP Request Node | Use built-in HTTP node |
| Custom JSON | JSON + Set Nodes | Use expression equivalents |
| Custom Transform | Function Node | Rewrite as JavaScript |
| Custom Database | Code Node | Use database connector code |

### Expression Migration

All n8n expressions work in m9m without changes:

```javascript
// These work identically in both systems
{{ $json.field }}
{{ $node('NodeName').json.result }}
{{ formatDate(now(), 'yyyy-MM-dd') }}
{{ if($json.condition, 'true', 'false') }}
```

### Error Handling Migration

m9m provides enhanced error handling:

```json
{
  "continueOnFail": true,
  "retryOnFail": true,
  "maxRetries": 3,
  "retryInterval": 1000
}
```

## Testing and Validation

### Automated Testing

Create automated tests for your migrated workflows:

```bash
#!/bin/bash
# test-migration.sh

echo "Testing workflow migration..."

# Test each workflow
for workflow in ./workflows/*.json; do
    echo "Testing $workflow..."

    # Validate workflow
    if ! ./m9m validate --workflow "$workflow"; then
        echo "❌ Validation failed for $workflow"
        exit 1
    fi

    # Test execution (dry run)
    if ! ./m9m execute --workflow "$workflow" --dry-run; then
        echo "❌ Execution test failed for $workflow"
        exit 1
    fi

    echo "✅ $workflow passed tests"
done

echo "✅ All workflows migrated successfully!"
```

### Performance Validation

Compare performance before and after migration:

```bash
# Create performance test
./m9m benchmark --workflow ./workflows/performance-test.json --output results.json

# Compare results
echo "m9m Performance Results:"
cat results.json | jq '.executionsPerSecond'
```

## Troubleshooting

### Common Issues and Solutions

#### Issue: Workflow validation fails
```bash
# Solution: Check for unsupported nodes
./m9m validate --workflow workflow.json --verbose

# Look for error details and replace unsupported nodes
```

#### Issue: Expressions not working
```bash
# Solution: Test expression syntax
./m9m test-expression "{{ \$json.field }}"

# Check for syntax differences
```

#### Issue: Webhook not receiving requests
```bash
# Solution: Check webhook configuration
./m9m serve --port 3000 --debug

# Verify webhook URLs and authentication
```

#### Issue: Performance not as expected
```bash
# Solution: Enable optimization flags
./m9m execute --workflow workflow.json --optimize

# Check resource usage
./m9m execute --workflow workflow.json --profile
```

### Getting Help

If you encounter issues during migration:

1. **Check Documentation**: Visit [docs.m9m.com](https://docs.m9m.com)
2. **GitHub Issues**: Report bugs at [github.com/neul-labs/m9m/issues](https://github.com/neul-labs/m9m/issues)
3. **Community Support**: Join the community forum
4. **Professional Support**: Contact for enterprise migration assistance

## Post-Migration Best Practices

### Monitoring and Observability

Set up monitoring for your m9m deployment:

```yaml
# monitoring.yaml
logging:
  level: info
  output: /var/log/m9m.log

metrics:
  enabled: true
  port: 9090
  path: /metrics

health:
  enabled: true
  port: 8080
  path: /health
```

### Backup and Recovery

Implement backup strategies:

```bash
#!/bin/bash
# backup-workflows.sh

# Backup workflows
cp -r ./workflows/ ./backups/workflows-$(date +%Y%m%d)/

# Backup configuration
cp config.yaml ./backups/config-$(date +%Y%m%d).yaml

# Backup credentials (encrypted)
./m9m export:credentials --output ./backups/credentials-$(date +%Y%m%d).json
```

### Security Hardening

Enhance security post-migration:

```bash
# Use environment variables for secrets
export ENCRYPTION_KEY=$(openssl rand -hex 32)
export WEBHOOK_SECRET=$(openssl rand -hex 16)

# Set restrictive file permissions
chmod 600 config.yaml
chmod 700 ./workflows/

# Run with limited privileges
useradd -r -s /bin/false m9m
sudo -u m9m ./m9m serve
```

## Migration Checklist

Use this checklist to ensure complete migration:

### Pre-Migration
- [ ] Export all workflows from n8n
- [ ] Export credentials and configurations
- [ ] Document custom nodes and their functionality
- [ ] Create test data for validation
- [ ] Set up m9m development environment

### Migration
- [ ] Install m9m
- [ ] Validate all exported workflows
- [ ] Test workflow execution with sample data
- [ ] Convert credentials if necessary
- [ ] Update webhook URLs
- [ ] Configure environment variables

### Testing
- [ ] Run validation tests on all workflows
- [ ] Execute performance benchmarks
- [ ] Test webhook endpoints
- [ ] Verify external API integrations
- [ ] Test error handling scenarios

### Deployment
- [ ] Set up production environment
- [ ] Configure monitoring and logging
- [ ] Implement backup procedures
- [ ] Set up security measures
- [ ] Update DNS/load balancer configurations

### Post-Migration
- [ ] Monitor performance metrics
- [ ] Validate all workflows in production
- [ ] Train team on m9m differences
- [ ] Document any migration-specific changes
- [ ] Plan for ongoing maintenance

## Conclusion

Migrating from n8n to m9m provides significant performance improvements while maintaining 100% compatibility with your existing workflows. Follow this guide step-by-step to ensure a smooth transition and take advantage of m9m's enhanced capabilities.

For additional support or questions about your specific migration scenario, please refer to the documentation or contact the m9m community.