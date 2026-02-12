# Frequently Asked Questions

Common questions about m9m.

## General

### What is m9m?

m9m is a high-performance workflow automation platform written in Go. It provides n8n-compatible workflow execution with significantly better performance characteristics.

### How does m9m compare to n8n?

| Feature | m9m | n8n |
|---------|-----|-----|
| Execution speed | 5-10x faster | Baseline |
| Memory usage | ~150 MB | ~512 MB |
| Container size | ~300 MB | ~1.2 GB |
| Startup time | ~500 ms | ~3 s |
| Workflow format | Compatible | Original |
| Language | Go | TypeScript |

### Is m9m compatible with n8n workflows?

Yes. m9m can execute n8n workflows without modification. Simply export your workflow from n8n and run it with m9m.

### Is m9m open source?

Yes, m9m is open source and available on GitHub.

## Installation

### What are the system requirements?

**Minimum**:
- 1 CPU core
- 256 MB RAM
- 1 GB disk

**Recommended**:
- 2+ CPU cores
- 512 MB+ RAM
- 10 GB+ SSD

### Which databases are supported?

- SQLite (default, single instance)
- PostgreSQL (production, scalable)

### Can I run m9m on Windows?

m9m is primarily tested on Linux. For Windows, we recommend using Docker or WSL2.

## Workflows

### How do I create a workflow?

Three options:

1. **JSON file** - Write workflow JSON directly
2. **API** - Create via REST API
3. **n8n export** - Export from n8n and import

### Can I use n8n's web editor?

m9m focuses on headless execution. You can:
- Use n8n's editor to design workflows, then export
- Use the API to manage workflows programmatically
- Write JSON workflows directly

### How many workflows can I run?

There's no hard limit. Performance depends on:
- Hardware resources
- Workflow complexity
- Execution frequency

### What node types are supported?

34+ node types including:
- HTTP requests
- Database operations (PostgreSQL, MySQL, SQLite)
- Messaging (Slack, Discord)
- AI/LLM (OpenAI, Anthropic)
- Cloud services (AWS, Azure, GCP)
- And more

See [Nodes documentation](../nodes/index.md) for full list.

## Execution

### How do I trigger a workflow?

1. **Manual** - CLI or API call
2. **Webhook** - HTTP request to webhook URL
3. **Schedule** - Cron expression

### What happens if a workflow fails?

- Execution is marked as "failed"
- Error details are logged
- Configured retries are attempted
- Alerting can be configured

### Can workflows run in parallel?

Yes. Multiple workflows can execute simultaneously. Configure the number of workers:

```yaml
queue:
  workers: 10
```

### How long can a workflow run?

Default timeout is 5 minutes. Configure per-workflow:

```yaml
execution:
  timeout: 30m
```

## Performance

### Why is m9m faster than n8n?

- **Go vs JavaScript** - Compiled language, better performance
- **Efficient memory** - Less garbage collection overhead
- **Optimized execution** - Parallel node execution where possible
- **Native concurrency** - Goroutines for efficient parallelism

### How do I optimize workflow performance?

1. Minimize external API calls
2. Use caching where possible
3. Process data in batches
4. Use async execution for long workflows
5. Monitor and profile executions

### What's the maximum data size?

There's no hard limit, but large datasets may impact performance. For large data:
- Use streaming where possible
- Process in batches
- Increase memory allocation

## Security

### How are credentials stored?

Credentials are encrypted at rest using AES-256. The encryption key should be stored securely (environment variable, secrets manager).

### Does m9m support SSO?

Currently, m9m supports JWT and API key authentication. SSO integration is on the roadmap.

### Is data encrypted in transit?

Configure TLS for HTTPS:

```yaml
server:
  tls:
    enabled: true
    certFile: /path/to/cert.pem
    keyFile: /path/to/key.pem
```

## Deployment

### Should I use Docker or Kubernetes?

| Use Case | Recommendation |
|----------|----------------|
| Development | Docker |
| Single server | Docker Compose |
| Production | Binary / package manager |
| Edge deployment | Binary |
| Kubernetes | Experimental |

### How do I scale m9m?

1. Run multiple instances behind load balancer
2. Use Redis for distributed queue
3. Use PostgreSQL for shared storage
4. Use orchestration only if you are prepared to run experimental Kubernetes manifests

### How do I backup m9m?

Backup the database:

```bash
# SQLite
cp /data/m9m.db /backup/m9m-$(date +%Y%m%d).db

# PostgreSQL
pg_dump m9m > /backup/m9m-$(date +%Y%m%d).sql
```

## Troubleshooting

### Where are the logs?

```bash
# CLI
m9m serve  # Outputs to stdout

# Docker
docker logs m9m

# Kubernetes
kubectl logs deployment/m9m
```

### Why is my webhook not working?

1. Is the workflow activated?
2. Is the path correct?
3. Is the HTTP method correct?
4. Check server logs for errors

### Why is my execution stuck?

Check for:
- Long-running external calls
- Infinite loops in Code nodes
- Resource exhaustion

Cancel stuck execution:
```bash
m9m execution cancel exec-123
```

## Migration

### How do I migrate from n8n?

1. Export workflows from n8n (JSON format)
2. Import into m9m: `m9m create --from workflow.json`
3. Create credentials in m9m
4. Test workflows
5. Activate workflows

### Are all n8n nodes supported?

Most common nodes are supported. Check the [nodes documentation](../nodes/index.md) for the full list.

### Can I run n8n and m9m together?

Yes, you can run both during migration:
- Keep n8n for unsupported workflows
- Migrate supported workflows to m9m
- Gradually transition

## Support

### How do I report a bug?

Open an issue on GitHub with:
- m9m version
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs

### How do I request a feature?

Open a feature request on GitHub describing:
- Use case
- Proposed solution
- Alternatives considered

### Is commercial support available?

Contact Neul Labs for commercial support options.
