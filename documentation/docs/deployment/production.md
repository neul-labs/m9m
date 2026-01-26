# Production Checklist

Essential steps for production deployment.

## Security

### Authentication

- [ ] **Set strong JWT secret**
  ```bash
  M9M_JWT_SECRET=$(openssl rand -hex 32)
  ```

- [ ] **Configure JWT expiration**
  ```yaml
  security:
    jwtExpiration: 24h
    refreshTokenExpiration: 168h
  ```

- [ ] **Enable API key authentication for service accounts**

### Encryption

- [ ] **Set encryption key for credentials**
  ```bash
  M9M_ENCRYPTION_KEY=$(openssl rand -hex 16)
  ```

- [ ] **Enable TLS/HTTPS**
  ```yaml
  server:
    tls:
      enabled: true
      certFile: /etc/m9m/tls.crt
      keyFile: /etc/m9m/tls.key
  ```

### Network

- [ ] **Configure CORS appropriately**
  ```yaml
  security:
    cors:
      allowedOrigins:
        - "https://app.example.com"
      allowedMethods:
        - GET
        - POST
        - PUT
        - DELETE
  ```

- [ ] **Set up firewall rules**
- [ ] **Use private network for database connections**
- [ ] **Enable rate limiting**
  ```yaml
  security:
    rateLimit:
      enabled: true
      requests: 100
      window: 60
  ```

### Access Control

- [ ] **Disable debug endpoints in production**
  ```yaml
  server:
    enablePprof: false
  ```

- [ ] **Use minimal permissions for service accounts**
- [ ] **Audit credential access**

## Database

### PostgreSQL Recommended

- [ ] **Use PostgreSQL for production**
  ```yaml
  database:
    type: postgres
    url: "postgres://user:pass@host:5432/m9m?sslmode=require"
  ```

- [ ] **Enable SSL connections**
- [ ] **Configure connection pooling**
  ```yaml
  database:
    maxOpenConns: 25
    maxIdleConns: 5
    connMaxLifetime: 5m
  ```

### Backups

- [ ] **Set up automated backups**
  ```bash
  pg_dump -h localhost -U m9m m9m > backup.sql
  ```

- [ ] **Test backup restoration**
- [ ] **Store backups in separate location**
- [ ] **Configure retention policy**

### Performance

- [ ] **Add database indexes**
- [ ] **Monitor query performance**
- [ ] **Set up connection pooling (PgBouncer)**

## High Availability

### Multiple Instances

- [ ] **Run at least 2 replicas**
  ```yaml
  replicas: 2
  ```

- [ ] **Configure load balancer**
- [ ] **Enable health checks**

### Queue

- [ ] **Use Redis for distributed queue**
  ```yaml
  queue:
    type: redis
    url: "redis://redis:6379"
  ```

- [ ] **Configure Redis persistence**
- [ ] **Set up Redis Sentinel or Cluster**

### Failover

- [ ] **Configure pod anti-affinity**
- [ ] **Set up PodDisruptionBudget**
- [ ] **Test failover scenarios**

## Monitoring

### Metrics

- [ ] **Enable Prometheus metrics**
  ```yaml
  monitoring:
    enabled: true
    metricsPort: 9090
  ```

- [ ] **Set up Grafana dashboards**
- [ ] **Configure alerting rules**

### Key Metrics to Monitor

| Metric | Alert Threshold |
|--------|-----------------|
| `m9m_execution_errors` | > 5 per minute |
| `m9m_execution_duration` | p99 > 30s |
| `m9m_queue_size` | > 1000 |
| `m9m_active_workflows` | Unexpected changes |
| Memory usage | > 80% |
| CPU usage | > 80% |

### Logging

- [ ] **Use JSON log format**
  ```yaml
  logging:
    format: json
    level: info
  ```

- [ ] **Set up log aggregation** (ELK, Loki)
- [ ] **Configure log retention**
- [ ] **Don't log sensitive data**

### Tracing

- [ ] **Enable distributed tracing**
  ```yaml
  tracing:
    enabled: true
    endpoint: "http://jaeger:14268/api/traces"
  ```

## Performance

### Resource Allocation

- [ ] **Set appropriate resource limits**
  ```yaml
  resources:
    requests:
      memory: "256Mi"
      cpu: "250m"
    limits:
      memory: "512Mi"
      cpu: "1000m"
  ```

### Autoscaling

- [ ] **Configure HPA**
  ```yaml
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 10
    targetCPU: 70
  ```

### Timeouts

- [ ] **Set appropriate timeouts**
  ```yaml
  server:
    readTimeout: 30s
    writeTimeout: 30s
    idleTimeout: 120s
  ```

## Operations

### Deployment

- [ ] **Use container orchestration** (Kubernetes)
- [ ] **Implement blue/green or rolling deployments**
- [ ] **Version container images**
- [ ] **Don't use `latest` tag in production**

### Configuration

- [ ] **Use environment variables or secrets manager**
- [ ] **Don't commit secrets to version control**
- [ ] **Use ConfigMaps for non-sensitive config**

### Updates

- [ ] **Plan maintenance windows**
- [ ] **Test updates in staging first**
- [ ] **Document rollback procedures**
- [ ] **Keep dependencies updated**

## Disaster Recovery

### Backup Strategy

| Data | Frequency | Retention |
|------|-----------|-----------|
| Database | Daily | 30 days |
| Workflows | On change | Indefinite |
| Credentials | Daily | 30 days |
| Config | On change | Version controlled |

### Recovery Plan

- [ ] **Document recovery procedures**
- [ ] **Test recovery regularly**
- [ ] **Define RTO and RPO**
- [ ] **Have runbooks ready**

## Compliance

### Audit Logging

- [ ] **Enable audit logging**
  ```yaml
  audit:
    enabled: true
    logLevel: info
  ```

- [ ] **Log authentication events**
- [ ] **Log workflow modifications**
- [ ] **Log credential access**

### Data Protection

- [ ] **Encrypt data at rest**
- [ ] **Encrypt data in transit (TLS)**
- [ ] **Implement data retention policies**
- [ ] **Handle PII appropriately**

## Pre-Launch Checklist

### Final Verification

- [ ] All secrets are properly configured
- [ ] TLS certificates are valid and not expiring soon
- [ ] Database backups are working
- [ ] Monitoring and alerting are configured
- [ ] Health checks are passing
- [ ] Load testing completed
- [ ] Security scan completed
- [ ] Documentation is up to date
- [ ] Runbooks are ready
- [ ] Support contacts are defined

### Go-Live

- [ ] Announce maintenance window
- [ ] Deploy to production
- [ ] Verify health checks
- [ ] Test critical workflows
- [ ] Monitor metrics
- [ ] Verify alerts work
- [ ] Celebrate!
