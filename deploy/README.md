# m9m Deployment Guide

Deploy m9m - the Agent-Native Workflow Automation Platform - with one click.

## One-Click Deploy

### Railway
[![Deploy on Railway](https://railway.app/button.svg)](https://railway.app/template/m9m)

```bash
railway login
railway init
railway up
```

### Fly.io
[![Deploy on Fly.io](https://fly.io/button.svg)](https://fly.io/docs/speedrun/)

```bash
fly auth login
fly launch
fly deploy
```

### Render
[![Deploy to Render](https://render.com/images/deploy-to-render-button.svg)](https://render.com/deploy)

Connect your GitHub repo and Render will automatically deploy.

### DigitalOcean
[![Deploy to DO](https://www.deploytodo.com/do-btn-blue.svg)](https://cloud.digitalocean.com/apps/new)

Use the App Platform with our Dockerfile.

## Docker

```bash
# Pull and run
docker pull ghcr.io/neul-labs/m9m:latest
docker run -d -p 8080:8080 ghcr.io/neul-labs/m9m:latest

# With docker-compose
docker-compose up -d
```

## Kubernetes

```bash
# Apply manifests
kubectl apply -f deploy/kubernetes.yaml

# Check status
kubectl get pods -n m9m
kubectl get svc -n m9m
```

## Helm Chart (Coming Soon)

```bash
helm repo add m9m https://charts.m9m.io
helm install m9m m9m/m9m
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `M9M_PORT` | HTTP server port | `8080` |
| `M9M_HOST` | Bind address | `0.0.0.0` |
| `M9M_LOG_LEVEL` | Log level (debug/info/warn/error) | `info` |
| `M9M_DB_TYPE` | Database type (sqlite/postgres/badger) | `sqlite` |
| `M9M_DB_POSTGRES_URL` | PostgreSQL connection URL | - |
| `M9M_QUEUE_TYPE` | Queue type (memory/redis/rabbitmq) | `memory` |
| `M9M_QUEUE_URL` | Queue connection URL | - |
| `M9M_COPILOT_PROVIDER` | AI provider (openai/anthropic/ollama) | - |
| `M9M_COPILOT_API_KEY` | AI API key | - |
| `M9M_COPILOT_MODEL` | AI model name | `gpt-4` |

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Load Balancer                         │
└─────────────────────────────────────────────────────────┘
                          │
          ┌───────────────┼───────────────┐
          ▼               ▼               ▼
    ┌──────────┐    ┌──────────┐    ┌──────────┐
    │ Control  │    │ Control  │    │ Control  │
    │  Plane   │    │  Plane   │    │  Plane   │
    └──────────┘    └──────────┘    └──────────┘
          │               │               │
          └───────────────┼───────────────┘
                          │
          ┌───────────────┼───────────────┐
          ▼               ▼               ▼
    ┌──────────┐    ┌──────────┐    ┌──────────┐
    │  Worker  │    │  Worker  │    │  Worker  │
    └──────────┘    └──────────┘    └──────────┘
          │               │               │
          └───────────────┼───────────────┘
                          │
                    ┌─────┴─────┐
                    │ PostgreSQL│
                    │   Redis   │
                    └───────────┘
```

## Production Checklist

- [ ] Use PostgreSQL for storage
- [ ] Enable Redis for queue/caching
- [ ] Configure SSL/TLS
- [ ] Set up monitoring (Prometheus/Grafana)
- [ ] Enable audit logging
- [ ] Configure backups
- [ ] Set resource limits
- [ ] Enable horizontal scaling

## Support

- Documentation: https://docs.m9m.io
- GitHub Issues: https://github.com/neul-labs/m9m/issues
- Discord: https://discord.gg/m9m
