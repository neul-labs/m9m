# n8n-go Project Plan

This document outlines the plan for building a modular n8n implementation in Go that can run exported n8n workflows without changes.

## Project Goals

1. **Compatibility**: Run exported n8n workflows without modification
2. **Performance**: Significant performance improvements over Node.js implementation
3. **Modularity**: Clean, well-defined interfaces for extensibility
4. **Maintainability**: Easy to understand and extend codebase
5. **Scalability**: Efficient handling of concurrent workflows

## Key Abstractions

### 1. Workflow Engine
- Core workflow execution engine
- Node execution orchestration
- Data flow management
- Error handling and recovery

### 2. Node Interface
- Standard interface for all node types
- Input/output data handling
- Parameter validation
- Execution context management

### 3. Connection System
- Data routing between nodes
- Multiple connection types support
- Dynamic connection management

### 4. Data Model
- Standard data structures for workflow representation
- JSON serialization/deserialization
- Type safety for workflow data

### 5. Execution Context
- Workflow state management
- Variable resolution
- Expression evaluation
- Credential handling

## Module Structure

```
n8n-go/
├── cmd/
│   └── n8n-go/
│       └── main.go
├── internal/
│   ├── engine/
│   │   ├── engine.go
│   │   ├── executor.go
│   │   └── workflow.go
│   ├── nodes/
│   │   ├── base/
│   │   │   └── node.go
│   │   ├── http/
│   │   ├── database/
│   │   ├── transform/
│   │   └── ... (other node types)
│   ├── model/
│   │   ├── workflow.go
│   │   ├── node.go
│   │   ├── connection.go
│   │   └── data.go
│   ├── connections/
│   │   └── router.go
│   ├── expressions/
│   │   └── evaluator.go
│   └── utils/
│       └── helpers.go
├── pkg/
│   └── ... (public packages if needed)
├── docs/
│   └── project-plan.md
├── go.mod
└── README.md
```

## Implementation Plan

### Phase 1: Core Infrastructure (Weeks 1-2)
1. Define data models for workflow representation
2. Implement JSON import/export functionality
3. Create basic workflow engine structure
4. Implement connection routing system

### Phase 2: Node System (Weeks 3-4)
1. Define node interface and base implementation
2. Implement core node types:
   - HTTP Request
   - Set
   - Function
   - Item Lists
3. Create node registry system

### Phase 3: Execution Engine (Weeks 5-6)
1. Implement workflow execution logic
2. Add support for parallel execution
3. Implement error handling and recovery
4. Add expression evaluation support

### Phase 4: Additional Node Types (Weeks 7-10)
1. Database nodes
2. File operation nodes
3. Email nodes
4. Timer/trigger nodes
5. Cloud service integrations

### Phase 5: Advanced Features (Weeks 11-12)
1. Credential management
2. Webhook support
3. CLI interface
4. Performance optimization
5. Testing and documentation

## Compatibility Requirements

### Workflow JSON Structure
The Go implementation must support the exact JSON structure exported by n8n:
- Workflow metadata (ID, name, active status)
- Node definitions with parameters
- Connection definitions
- Settings and static data
- Pin data for testing

### Node Compatibility
Each node implementation must:
- Accept the same parameter structure
- Produce the same output format
- Handle errors in the same way
- Support the same configuration options

### Data Flow Compatibility
- Support for multiple connection types
- Proper data routing between nodes
- Handling of empty/missing data
- Support for paired item tracking

## Performance Targets

- 5x improvement in CPU usage
- 50% reduction in memory usage
- 10x faster startup time
- 100x improvement in concurrent workflow handling