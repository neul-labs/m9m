# Architecture Overview

Understanding m9m's architecture and design.

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Clients                             │
│  (CLI, API, Webhooks, Scheduled Triggers)                   │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                      REST API Layer                         │
│  ┌─────────┐  ┌──────────┐  ┌──────────┐  ┌─────────────┐  │
│  │Auth     │  │Workflows │  │Executions│  │Webhooks     │  │
│  │Middleware│ │Handlers  │  │Handlers  │  │Handlers     │  │
│  └─────────┘  └──────────┘  └──────────┘  └─────────────┘  │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                    Workflow Engine                          │
│  ┌──────────────┐  ┌─────────────┐  ┌──────────────────┐   │
│  │ Orchestrator │  │ Expression  │  │ Credential       │   │
│  │              │  │ Evaluator   │  │ Manager          │   │
│  └──────────────┘  └─────────────┘  └──────────────────┘   │
│  ┌──────────────┐  ┌─────────────┐  ┌──────────────────┐   │
│  │ Node         │  │ Data        │  │ Error            │   │
│  │ Registry     │  │ Transformer │  │ Handler          │   │
│  └──────────────┘  └─────────────┘  └──────────────────┘   │
└─────────────────────┬───────────────────────────────────────┘
                      │
          ┌───────────┴───────────┐
          ▼                       ▼
┌─────────────────┐     ┌─────────────────┐
│   Job Queue     │     │   Storage       │
│  ┌───────────┐  │     │  ┌───────────┐  │
│  │  Memory   │  │     │  │  SQLite   │  │
│  │  Redis    │  │     │  │  Postgres │  │
│  │  RabbitMQ │  │     │  └───────────┘  │
│  └───────────┘  │     └─────────────────┘
└─────────────────┘
```

## Core Components

### REST API Layer

Handles all incoming HTTP requests:

- **Authentication middleware** - JWT/API key validation
- **Rate limiting** - Prevents abuse
- **Request routing** - Maps endpoints to handlers
- **Response formatting** - Consistent JSON responses

### Workflow Engine

The heart of m9m:

| Component | Responsibility |
|-----------|----------------|
| Orchestrator | Manages workflow execution flow |
| Expression Evaluator | Processes `{{ }}` expressions |
| Node Registry | Stores available node types |
| Data Transformer | Handles data flow between nodes |
| Credential Manager | Securely provides credentials |
| Error Handler | Manages failures and retries |

### Node System

Nodes are pluggable execution units:

```go
type NodeExecutor interface {
    Execute(inputData []DataItem, params map[string]interface{}) ([]DataItem, error)
    Description() NodeDescription
    ValidateParameters(params map[string]interface{}) error
}
```

### Job Queue

Manages asynchronous workflow execution:

- **Memory** - In-process queue for single instance
- **SQLite** - Persistent queue for single instance
- **Redis** - Distributed queue for multiple instances

### Storage Layer

Persists workflows and executions:

- **SQLite** - Default, file-based
- **PostgreSQL** - Production, scalable

## Data Flow

### Workflow Execution Flow

```
1. Trigger (Manual/Webhook/Cron)
       │
       ▼
2. Load Workflow Definition
       │
       ▼
3. Create Execution Context
       │
       ▼
4. Queue Job (if async)
       │
       ▼
5. Execute Nodes (topological order)
       │
       ▼
6. For each node:
   a. Resolve expressions
   b. Get credentials
   c. Execute node logic
   d. Transform output
   e. Pass to next nodes
       │
       ▼
7. Store Execution Result
       │
       ▼
8. Return Response
```

### Data Item Structure

```json
{
  "json": {
    "field1": "value1",
    "field2": 123
  },
  "binary": {
    "file": {
      "data": "base64...",
      "mimeType": "application/pdf",
      "fileName": "document.pdf"
    }
  }
}
```

## Design Principles

### Performance First

- **Compiled language** - Go provides native performance
- **Minimal allocations** - Object pooling, efficient data structures
- **Concurrent execution** - Goroutines for parallel processing
- **Connection pooling** - Efficient database/HTTP connections

### Cloud Native

- **Stateless** - Horizontal scaling
- **12-factor app** - Environment-based configuration
- **Container ready** - Docker/Kubernetes native
- **Observable** - Prometheus metrics, structured logging

### n8n Compatible

- **Workflow format** - Same JSON structure
- **Expression syntax** - Same `{{ }}` expressions
- **Node types** - Compatible node identifiers
- **Seamless migration** - Import n8n workflows directly

## Module Structure

```
m9m/
├── cmd/
│   └── m9m/           # Application entry point
├── internal/
│   ├── api/           # REST API handlers
│   ├── engine/        # Workflow execution engine
│   ├── nodes/         # Node implementations
│   │   ├── base/      # Base interfaces
│   │   ├── transform/ # Data transformation
│   │   ├── http/      # HTTP requests
│   │   ├── database/  # Database operations
│   │   ├── messaging/ # Slack, Discord, etc.
│   │   └── ai/        # OpenAI, Anthropic
│   ├── queue/         # Job queue implementations
│   ├── storage/       # Data persistence
│   ├── credentials/   # Credential management
│   ├── expressions/   # Expression evaluation
│   └── monitoring/    # Metrics and tracing
└── docs/              # Documentation
```

## Scalability

### Horizontal Scaling

```
┌────────────────────────────────────────┐
│            Load Balancer               │
└──────┬─────────────┬─────────────┬─────┘
       │             │             │
       ▼             ▼             ▼
   ┌───────┐     ┌───────┐     ┌───────┐
   │ m9m-1 │     │ m9m-2 │     │ m9m-3 │
   └───┬───┘     └───┬───┘     └───┬───┘
       │             │             │
       └─────────────┼─────────────┘
                     │
                     ▼
              ┌─────────────┐
              │    Redis    │ (Job Queue)
              └─────────────┘
                     │
                     ▼
              ┌─────────────┐
              │  PostgreSQL │ (Storage)
              └─────────────┘
```

### Worker Scaling

Configure worker count based on workload:

```yaml
queue:
  workers: 10        # Number of concurrent workers
  maxRetries: 3      # Retry failed jobs
  retryDelay: 5s     # Delay between retries
```

## Security Architecture

### Authentication Flow

```
Client → JWT/API Key → Auth Middleware → Protected Resource
```

### Credential Security

```
Plaintext → AES-256 Encryption → Storage
Storage → Decryption → Node Execution → Clear from Memory
```

### Network Security

- TLS for all external connections
- Private network for internal services
- Rate limiting on API endpoints

## Next Steps

- [Workflow Engine](engine.md) - Deep dive into execution
- [Job Queue](queue.md) - Queue system details
