# Architecture Overview

m9m is built with a modular, scalable architecture designed for high-performance workflow automation in cloud-native environments.

## Core Components

### 1. Workflow Engine
The heart of m9m, responsible for:
- Workflow parsing and validation
- Node execution orchestration
- Data flow management
- Error handling and recovery
- Performance optimization

**Location**: `/internal/engine/`

### 2. Node Registry
Manages the catalog of available workflow nodes:
- Node registration and discovery
- Metadata management
- Version control
- Plugin architecture support

**Location**: `/internal/nodes/`

### 3. Queue System
Provides horizontal scaling capabilities:
- Job queue management
- Worker pool coordination
- Load distribution
- Retry logic
- Multiple backend support (Memory, Redis, RabbitMQ)

**Location**: `/internal/queue/`

### 4. Monitoring System
Enterprise-grade observability:
- Prometheus metrics collection
- OpenTelemetry distributed tracing
- Health and readiness checks
- Performance monitoring

**Location**: `/internal/monitoring/`

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        Client Layer                             │
├─────────────────┬─────────────────┬─────────────────────────────┤
│     Web UI      │    REST API     │      CLI Interface         │
│   (Planned)     │                 │                            │
└─────────────────┴─────────────────┴─────────────────────────────┘
                           │
┌─────────────────────────────────────────────────────────────────┐
│                     Service Layer                               │
├─────────────────────────────────────────────────────────────────┤
│                   Workflow Engine                               │
│  ┌──────────────┬──────────────┬──────────────┬──────────────┐  │
│  │   Parser     │   Executor   │   Scheduler  │   Monitor    │  │
│  └──────────────┴──────────────┴──────────────┴──────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                           │
┌─────────────────────────────────────────────────────────────────┐
│                    Node Registry                                │
├─────────────┬─────────────┬─────────────┬─────────────┬─────────┤
│ Messaging   │ Databases   │ Cloud Ops   │    AI/LLM   │  Core   │
│   Nodes     │    Nodes    │    Nodes    │    Nodes    │ Nodes   │
├─────────────┼─────────────┼─────────────┼─────────────┼─────────┤
│  - Slack    │ - MongoDB   │ - AWS       │ - OpenAI    │  - HTTP │
│  - Discord  │ - Redis     │ - GCP       │ - Anthropic │  - Set  │
│  - Teams    │ - PostgreSQL│ - Azure     │ - LiteLLM   │  - Code │
│  - Telegram │ - MySQL     │ - Storage   │             │  - File │
└─────────────┴─────────────┴─────────────┴─────────────┴─────────┘
                           │
┌─────────────────────────────────────────────────────────────────┐
│                Infrastructure Layer                             │
├─────────────────┬─────────────────┬─────────────────────────────┤
│  Queue System   │   Monitoring    │       Storage               │
│                 │                 │                             │
│ ┌─────────────┐ │ ┌─────────────┐ │ ┌─────────────────────────┐ │
│ │   Memory    │ │ │ Prometheus  │ │ │     File System         │ │
│ │   Redis     │ │ │ OpenTelemetry│ │ │     Database           │ │
│ │  RabbitMQ   │ │ │   Jaeger    │ │ │     Cloud Storage       │ │
│ └─────────────┘ │ └─────────────┘ │ └─────────────────────────┘ │
└─────────────────┴─────────────────┴─────────────────────────────┘
```

## Data Flow

### 1. Workflow Execution Flow
```
Workflow Definition (JSON) → Parser → Validation → Execution Graph → Node Execution → Results
```

### 2. Node Execution Flow
```
Node Metadata → Parameter Resolution → Credential Injection → Execute → Output Processing
```

### 3. Scaling Flow
```
Workflow Request → Queue Manager → Worker Pool → Node Execution → Result Collection
```

## Design Principles

### 1. Performance First
- Native Go compilation for speed
- Memory-efficient data structures
- Minimal allocations during execution
- Concurrent processing with goroutines

### 2. Cloud Native
- Container-first deployment
- Kubernetes-native scaling
- 12-factor app compliance
- Stateless execution model

### 3. Extensibility
- Plugin-based node architecture
- Interface-driven design
- Hot-pluggable components
- API-first approach

### 4. Reliability
- Comprehensive error handling
- Retry mechanisms
- Circuit breakers
- Graceful degradation

### 5. Observability
- Built-in metrics collection
- Distributed tracing
- Structured logging
- Health monitoring

## Component Interactions

### Workflow Engine ↔ Node Registry
- Node discovery and metadata retrieval
- Execution delegation
- Error reporting

### Queue System ↔ Workflow Engine
- Job scheduling and distribution
- Result collection
- Status updates

### Monitoring ↔ All Components
- Metrics collection
- Trace correlation
- Health status reporting

## Scaling Architecture

### Horizontal Scaling
```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  Worker 1   │    │  Worker 2   │    │  Worker N   │
└──────┬──────┘    └──────┬──────┘    └──────┬──────┘
       │                  │                  │
       └──────────────────┼──────────────────┘
                          │
            ┌─────────────────────────┐
            │     Queue System        │
            │  (Redis/RabbitMQ)       │
            └─────────────────────────┘
```

### Vertical Scaling
- Multi-core processing with goroutines
- Memory pooling for efficiency
- CPU-optimized algorithms
- I/O multiplexing

## Security Architecture

### Authentication & Authorization
- Multiple auth methods (OAuth2, API Keys, Service Accounts)
- Role-based access control (planned)
- Credential encryption at rest

### Execution Isolation
- Sandboxed node execution
- Resource limits
- Network isolation options

### Data Protection
- Encryption in transit (TLS)
- Credential masking in logs
- Secure parameter handling

## Comparison with n8n

| Aspect | n8n | m9m |
|--------|-----|---------|
| Runtime | Node.js | Go |
| Memory Usage | ~512MB | ~150MB |
| Startup Time | 3s | 0.5s |
| Concurrency | Event Loop | Goroutines |
| Scaling | PM2/Cluster | Native Queue |
| Container Size | 1.2GB | 300MB |
| Dependencies | Many | Minimal |

## Future Architecture Plans

### Phase 1: Web UI Integration
- Vue.js frontend
- WebSocket real-time updates
- REST API expansion

### Phase 2: Advanced Features
- Workflow templates
- Custom node marketplace
- Advanced scheduling

### Phase 3: Enterprise Features
- Multi-tenancy
- Advanced security
- Compliance frameworks