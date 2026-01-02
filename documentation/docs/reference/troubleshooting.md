# Troubleshooting

Common issues and solutions for m9m.

## Quick Diagnostics

```bash
# Check server health
curl http://localhost:8080/health

# Check readiness
curl http://localhost:8080/ready

# View logs
m9m logs --tail 100

# Check version
m9m version
```

## Startup Issues

### Server Won't Start

**Symptom**: Server exits immediately or hangs.

**Solutions**:

1. **Port already in use**
   ```bash
   # Check what's using the port
   lsof -i :8080

   # Use different port
   m9m serve --port 3000
   ```

2. **Database connection failed**
   ```bash
   # Test database connection
   psql "$M9M_DB_URL" -c "SELECT 1"

   # Check connection string
   echo $M9M_DB_URL
   ```

3. **Missing configuration**
   ```bash
   # Verify config file
   m9m serve --config /path/to/config.yaml
   ```

### Database Errors

**Symptom**: `connection refused` or `authentication failed`

**Solutions**:

1. Verify database is running:
   ```bash
   docker ps | grep postgres
   ```

2. Check connection URL format:
   ```
   postgres://user:password@host:5432/database?sslmode=disable
   ```

3. Verify credentials:
   ```bash
   psql -h localhost -U m9m -d m9m
   ```

### Queue Connection Failed

**Symptom**: `redis: connection refused`

**Solutions**:

1. Check Redis is running:
   ```bash
   redis-cli ping
   ```

2. Verify URL format:
   ```
   redis://:password@localhost:6379/0
   ```

## Execution Issues

### Workflow Timeout

**Symptom**: `execution timeout exceeded`

**Solutions**:

1. Increase timeout:
   ```yaml
   execution:
     timeout: 600s
   ```

2. Check for infinite loops in workflow

3. Optimize slow nodes (add caching, reduce data)

### Node Execution Failed

**Symptom**: Specific node fails with error

**Solutions**:

1. **HTTP Request errors**
   ```
   Enable retry:
   retryOnFail: true
   maxRetries: 3
   ```

2. **Credential errors**
   ```bash
   # Test credential
   m9m credentials test <id>
   ```

3. **Expression errors**
   - Check for undefined variables
   - Use optional chaining: `$json.field?.nested`

### Out of Memory

**Symptom**: `out of memory` or process killed

**Solutions**:

1. Increase memory limit:
   ```yaml
   resources:
     limits:
       memory: 2Gi
   ```

2. Process data in batches:
   ```json
   {
     "type": "n8n-nodes-base.splitInBatches",
     "parameters": {"batchSize": 100}
   }
   ```

3. Clear large data between nodes

## Webhook Issues

### Webhook Not Responding

**Symptom**: 404 or no response from webhook

**Solutions**:

1. Verify workflow is active:
   ```bash
   m9m workflow list | grep <name>
   ```

2. Check webhook path:
   ```bash
   curl -X POST http://localhost:8080/webhook/your-path
   ```

3. Check logs for errors:
   ```bash
   m9m logs --filter webhook
   ```

### Webhook Timeout

**Symptom**: Gateway timeout (504)

**Solutions**:

1. Use `onReceived` response mode:
   ```json
   {
     "parameters": {
       "responseMode": "onReceived",
       "responseCode": 202
     }
   }
   ```

2. Process asynchronously via queue

## Performance Issues

### Slow Workflow Execution

**Symptom**: Workflows take longer than expected

**Solutions**:

1. Enable profiling:
   ```bash
   M9M_ENABLE_PPROF=true m9m serve
   ```

2. Check bottlenecks:
   ```bash
   curl http://localhost:8080/debug/pprof/profile?seconds=30 > profile.out
   go tool pprof profile.out
   ```

3. Optimize:
   - Cache HTTP responses
   - Reduce data transformations
   - Use batch operations

### High Memory Usage

**Symptom**: Memory grows over time

**Solutions**:

1. Check for memory leaks:
   ```bash
   curl http://localhost:8080/debug/pprof/heap > heap.out
   go tool pprof heap.out
   ```

2. Clear execution history:
   ```bash
   m9m executions prune --older-than 7d
   ```

3. Reduce saved execution data:
   ```yaml
   execution:
     saveSuccessData: summary
   ```

### Queue Backlog

**Symptom**: Queue depth growing

**Solutions**:

1. Increase workers:
   ```yaml
   queue:
     maxWorkers: 50
   ```

2. Scale horizontally:
   ```bash
   kubectl scale deployment/m9m-workers --replicas=5
   ```

3. Check for failing workflows blocking queue

## Connection Issues

### Database Connection Pool Exhausted

**Symptom**: `too many connections`

**Solutions**:

1. Increase pool size:
   ```yaml
   database:
     maxConnections: 100
   ```

2. Use connection pooler (PgBouncer):
   ```yaml
   database:
     url: "postgres://pgbouncer:6432/m9m"
   ```

3. Check for connection leaks

### Redis Connection Errors

**Symptom**: `connection reset by peer`

**Solutions**:

1. Increase timeout:
   ```yaml
   queue:
     connectionTimeout: 10s
   ```

2. Check Redis memory:
   ```bash
   redis-cli info memory
   ```

3. Enable keepalive:
   ```yaml
   queue:
     keepAlive: true
   ```

## Authentication Issues

### API Key Invalid

**Symptom**: `401 Unauthorized`

**Solutions**:

1. Verify key format:
   ```bash
   curl -H "X-API-Key: m9m_sk_..." http://localhost:8080/api/v1/workflows
   ```

2. Check key exists:
   ```bash
   m9m apikey list
   ```

3. Regenerate key:
   ```bash
   m9m apikey create --name "New Key"
   ```

### Token Expired

**Symptom**: `401 Token expired`

**Solutions**:

1. Refresh token:
   ```bash
   curl -X POST http://localhost:8080/api/v1/auth/refresh \
     -d '{"refreshToken": "..."}'
   ```

2. Increase token expiry:
   ```yaml
   auth:
     jwt:
       accessTokenExpiry: 24h
   ```

## Logging

### Enable Debug Logging

```bash
M9M_LOG_LEVEL=debug m9m serve
```

### Log to File

```yaml
logging:
  output: /var/log/m9m/m9m.log
```

### Filter Logs

```bash
# Workflow errors only
m9m logs --filter error --workflow <id>

# Specific node
m9m logs --filter "node=HTTP Request"
```

## Getting Help

### Gather Diagnostics

```bash
#!/bin/bash
# diagnostics.sh

echo "=== m9m Diagnostics ==="
echo ""
echo "Version:"
m9m version
echo ""
echo "Health:"
curl -s http://localhost:8080/health | jq
echo ""
echo "Config:"
m9m config show
echo ""
echo "Recent Errors:"
m9m logs --filter error --limit 20
echo ""
echo "System Resources:"
free -h
df -h
echo ""
echo "Processes:"
ps aux | grep m9m
```

### Report Issues

Include in bug reports:

1. m9m version (`m9m version`)
2. Configuration (sanitized)
3. Error logs
4. Steps to reproduce
5. Expected vs actual behavior

Submit issues at: https://github.com/m9m/m9m/issues

## Next Steps

- [Configuration](configuration.md) - Configuration options
- [CLI Reference](cli.md) - Command help
- [Production](../deployment/production.md) - Production setup
