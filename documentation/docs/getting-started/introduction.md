# Introduction

## What is m9m?

m9m is a high-performance, cloud-native workflow automation platform built in Go. It enables you to automate tasks, integrate services, and build complex workflows with minimal resource usage and maximum performance.

## Why m9m?

### Performance First

m9m was built from the ground up in Go to deliver exceptional performance:

- **5-10x faster** workflow execution compared to Node.js alternatives
- **70% lower memory** footprint (~150MB vs 512MB)
- **Sub-second startup** times (500ms vs 3s)
- **75% smaller** container images (300MB vs 1.2GB)

### Cloud Native

Designed for modern infrastructure:

- Kubernetes-native deployment
- Horizontal scaling with Redis/RabbitMQ queues
- Prometheus metrics and OpenTelemetry tracing
- Health checks and graceful shutdown

### n8n Compatible

Migrate without friction:

- 95% workflow compatibility with n8n
- Same expression syntax and functions
- Compatible credential management
- Easy workflow import/export

### Extensive Integrations

Connect to 100+ services:

- **Databases**: PostgreSQL, MySQL, MongoDB, Redis, Elasticsearch
- **Cloud**: AWS, GCP, Azure
- **Messaging**: Slack, Discord, Telegram, Microsoft Teams
- **AI/LLM**: OpenAI, Anthropic, local models
- **Version Control**: GitHub, GitLab
- **Productivity**: Google Sheets, Microsoft 365

## Use Cases

### Data Integration

Connect and synchronize data between different systems:

```
CRM → Transform → Database → Notification
```

### Workflow Automation

Automate repetitive business processes:

```
Form Submission → Validation → Database → Email → Slack
```

### Event Processing

React to events from various sources:

```
Webhook → Process → Multiple Destinations
```

### AI Pipelines

Build intelligent automation with AI:

```
Input → LLM Analysis → Decision → Action
```

## Architecture

m9m follows a modular, plugin-based architecture:

```
┌─────────────────────────────────────────────────────────────┐
│                     User Interfaces                          │
│  ┌──────────┐    ┌──────────┐    ┌──────────────────────┐   │
│  │  Web UI  │    │   CLI    │    │      REST API        │   │
│  └────┬─────┘    └────┬─────┘    └──────────┬───────────┘   │
└───────┼───────────────┼─────────────────────┼───────────────┘
        │               │                     │
        └───────────────┴──────────┬──────────┘
                                   │
┌──────────────────────────────────┴──────────────────────────┐
│                      Workflow Engine                         │
│  ┌──────────────┐  ┌─────────────────┐  ┌────────────────┐  │
│  │ Node Registry│  │Expression Engine│  │Credential Mgr  │  │
│  └──────────────┘  └─────────────────┘  └────────────────┘  │
└──────────────────────────────┬──────────────────────────────┘
                               │
┌──────────────────────────────┴──────────────────────────────┐
│                       Queue System                           │
│  ┌──────────┐    ┌──────────┐    ┌──────────────────────┐   │
│  │  Memory  │    │  Redis   │    │      RabbitMQ        │   │
│  └──────────┘    └──────────┘    └──────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Key Concepts

### Workflows

A workflow is a collection of nodes connected together to perform a task. Workflows define the logic of your automation.

### Nodes

Nodes are the building blocks of workflows. Each node performs a specific action:

- **Trigger Nodes**: Start workflow execution (webhooks, schedules, events)
- **Action Nodes**: Perform operations (HTTP requests, database queries)
- **Transform Nodes**: Manipulate data (filter, map, merge)
- **AI Nodes**: Integrate with LLMs and AI services

### Expressions

Expressions allow you to dynamically reference data within workflows:

```
{{ $json.customer.name }}
{{ $input.item.email }}
{{ $now.format('YYYY-MM-DD') }}
```

### Credentials

Credentials securely store authentication information for external services. m9m encrypts credentials at rest and provides fine-grained access control.

## Next Steps

Ready to get started?

1. [Install m9m](installation.md) on your system
2. Follow the [Quick Start](quickstart.md) guide
3. Build [Your First Workflow](first-workflow.md)
