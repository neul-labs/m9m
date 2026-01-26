# Troubleshooting

Common issues and how to resolve them.

## Startup Issues

### Server Won't Start

**Symptom**: Server exits immediately or hangs.

**Check logs**:
```bash
m9m serve --log-level debug
```

**Common causes**:

1. **Port in use**
   ```bash
   # Check port
   lsof -i :8080

   # Use different port
   m9m serve --port 8081
   ```

2. **Database connection failed**
   ```bash
   # Test database connection
   psql $DATABASE_URL -c "SELECT 1"
   ```

3. **Missing configuration**
   ```bash
   # Check config file exists
   cat /etc/m9m/config.yaml
   ```

### Database Connection Errors

**Symptom**: "connection refused" or "authentication failed"

**PostgreSQL**:
```bash
# Check PostgreSQL is running
systemctl status postgresql

# Test connection
psql -h localhost -U m9m -d m9m

# Check pg_hba.conf allows connections
```

**SQLite**:
```bash
# Check file permissions
ls -la /data/m9m.db

# Check directory exists
mkdir -p /data
chmod 755 /data
```

## Workflow Execution Issues

### Workflow Fails Immediately

**Check**:
1. Workflow JSON is valid
2. All node types exist
3. Connections are correct

**Validate workflow**:
```bash
m9m validate workflow.json
```

### Node Execution Fails

**Get execution details**:
```bash
m9m execution get exec-123
```

**Common node errors**:

| Error | Cause | Solution |
|-------|-------|----------|
| "connection timeout" | External service slow | Increase timeout |
| "401 Unauthorized" | Invalid credentials | Update credentials |
| "404 Not Found" | Wrong URL | Check URL |
| "expression error" | Invalid expression | Fix expression syntax |

### Expression Errors

**Symptom**: "Cannot read property 'x' of undefined"

**Cause**: Accessing missing field

**Solution**: Use safe access
```javascript
// Before
{{ $json.user.name }}

// After
{{ $json.user?.name ?? "default" }}
```

### Credentials Not Working

**Check credentials**:
```bash
# Test credential
m9m credential test cred-123

# View credential (redacted)
m9m credential get cred-123
```

**Common issues**:
- Expired tokens (OAuth2)
- Wrong API endpoint
- Missing permissions

## Performance Issues

### Slow Execution

**Profile workflow**:
```bash
m9m run workflow.json --profile
```

**Check**:
1. External API response times
2. Database query performance
3. Large data volumes

**Solutions**:
- Add caching
- Paginate large datasets
- Use async execution

### High Memory Usage

**Check memory**:
```bash
curl http://localhost:8080/metrics | grep memory
```

**Causes**:
- Large workflows
- Many concurrent executions
- Memory leaks

**Solutions**:
```yaml
# Limit concurrent executions
queue:
  workers: 5

# Set execution timeout
execution:
  timeout: 5m
```

### Queue Backing Up

**Check queue**:
```bash
curl http://localhost:8080/api/v1/jobs?status=pending | jq '.count'
```

**Solutions**:
1. Add more workers
2. Increase instance count
3. Optimize workflows

## Connection Issues

### Can't Connect to API

**Check**:
```bash
# Is server running?
curl http://localhost:8080/health

# Check firewall
sudo ufw status

# Check listening address
netstat -tlnp | grep 8080
```

### Webhook Not Triggering

**Check**:
1. Workflow is activated
2. Webhook path is correct
3. HTTP method matches

**Test webhook**:
```bash
curl -X POST http://localhost:8080/webhook/my-endpoint \
  -H "Content-Type: application/json" \
  -d '{"test": true}'
```

### Redis Connection Failed

**Check Redis**:
```bash
# Is Redis running?
redis-cli ping

# Test connection
redis-cli -h localhost -p 6379 info
```

## Authentication Issues

### JWT Token Invalid

**Symptoms**:
- 401 Unauthorized
- "token expired"
- "invalid signature"

**Solutions**:
```bash
# Get new token
m9m auth login

# Check token expiration
m9m auth status

# Verify JWT_SECRET is consistent across instances
```

### API Key Not Working

**Check**:
```bash
# List API keys
m9m apikey list

# Test API key
curl -H "X-API-Key: your-key" http://localhost:8080/api/v1/workflows
```

## Docker Issues

### Container Keeps Restarting

**Check logs**:
```bash
docker logs m9m
docker logs --tail 100 m9m
```

**Check health**:
```bash
docker inspect m9m | jq '.[0].State'
```

### Volume Permissions

**Symptom**: "permission denied"

**Fix**:
```bash
# Check volume permissions
docker exec m9m ls -la /data

# Fix permissions
docker exec m9m chown -R m9m:m9m /data
```

## Kubernetes Issues

### Pod CrashLoopBackOff

**Check**:
```bash
kubectl describe pod m9m-xxx -n m9m
kubectl logs m9m-xxx -n m9m --previous
```

### Service Not Accessible

**Check**:
```bash
# Service exists
kubectl get svc -n m9m

# Endpoints exist
kubectl get endpoints m9m -n m9m

# Pod is ready
kubectl get pods -n m9m -l app=m9m
```

### Ingress Not Working

**Check**:
```bash
# Ingress status
kubectl describe ingress m9m -n m9m

# Ingress controller logs
kubectl logs -n ingress-nginx deployment/ingress-nginx-controller
```

## Logging

### Enable Debug Logging

```bash
# CLI
m9m serve --log-level debug

# Environment
M9M_LOG_LEVEL=debug m9m serve

# Config
logging:
  level: debug
```

### Log Locations

| Deployment | Location |
|------------|----------|
| Binary | stdout |
| Docker | `docker logs m9m` |
| Kubernetes | `kubectl logs` |
| Systemd | `journalctl -u m9m` |

### Important Log Messages

| Message | Meaning |
|---------|---------|
| "starting server" | Server is starting |
| "listening on" | Server ready |
| "workflow execution started" | Job picked up |
| "workflow execution completed" | Job finished |
| "error executing node" | Node failed |

## Getting Help

### Gather Information

Before asking for help:

```bash
# Version
m9m version

# Health
curl http://localhost:8080/health

# Recent logs
m9m serve --log-level debug 2>&1 | tail -100

# Execution details
m9m execution get exec-123
```

### Community Resources

- GitHub Issues: Report bugs and feature requests
- Documentation: https://docs.neullabs.com/m9m

## Next Steps

- [FAQ](faq.md) - Frequently asked questions
