# n8n-go Technical Specification

This document provides detailed technical specifications for the n8n-go implementation.

## Workflow Data Model

### IWorkflowBase Interface Equivalent

```go
type Workflow struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Active      bool                   `json:"active"`
    Nodes       []Node                 `json:"nodes"`
    Connections map[string]Connections `json:"connections"`
    Settings    *WorkflowSettings      `json:"settings,omitempty"`
    StaticData  map[string]interface{} `json:"staticData,omitempty"`
    PinData     map[string][]DataItem  `json:"pinData,omitempty"`
    VersionID   string                 `json:"versionId,omitempty"`
}
```

### INode Interface Equivalent

```go
type Node struct {
    ID          string                 `json:"id,omitempty"`
    Name        string                 `json:"name"`
    Type        string                 `json:"type"`
    TypeVersion int                    `json:"typeVersion"`
    Position    []int                  `json:"position"`
    Parameters  map[string]interface{} `json:"parameters"`
    Credentials map[string]Credential  `json:"credentials,omitempty"`
    WebhookID   string                 `json:"webhookId,omitempty"`
}
```

### Data Structures

```go
type DataItem struct {
    JSON    map[string]interface{} `json:"json"`
    Binary  map[string]BinaryData  `json:"binary,omitempty"`
    PairedItem interface{}           `json:"pairedItem,omitempty"`
}

type BinaryData struct {
    Data         string `json:"data"`
    MimeType     string `json:"mimeType"`
    FileSize     string `json:"fileSize,omitempty"`
    FileName     string `json:"fileName,omitempty"`
}

type Connections struct {
    Main [][]Connection `json:"main,omitempty"`
}

type Connection struct {
    Node  string `json:"node"`
    Type  string `json:"type"`
    Index int    `json:"index"`
}
```

## Core Abstractions

### Node Interface

```go
type NodeExecutor interface {
    // Execute processes the node with given input data
    Execute(ctx ExecutionContext, inputData []DataItem) ([]DataItem, error)
    
    // Description returns metadata about the node
    Description() NodeDescription
    
    // ValidateParameters checks if node parameters are valid
    ValidateParameters(params map[string]interface{}) error
}
```

### Workflow Engine

```go
type WorkflowEngine interface {
    // ExecuteWorkflow runs a complete workflow
    ExecuteWorkflow(workflow *Workflow, inputData []DataItem) (*ExecutionResult, error)
    
    // ExecutePartialWorkflow runs a portion of a workflow
    ExecutePartialWorkflow(workflow *Workflow, startNode string, inputData []DataItem) (*ExecutionResult, error)
    
    // RegisterNodeType registers a new node type
    RegisterNodeType(nodeType string, executor NodeExecutor)
}
```

### Execution Context

```go
type ExecutionContext interface {
    // GetWorkflow returns the current workflow
    GetWorkflow() *Workflow
    
    // GetNode returns a node by name
    GetNode(name string) *Node
    
    // EvaluateExpression evaluates an expression in the current context
    EvaluateExpression(expression string, itemIndex int) (interface{}, error)
    
    // GetCredential retrieves credentials for a node
    GetCredential(nodeName string, credentialType string) (Credential, error)
}
```

## Module Architecture

### Engine Module

The engine module is responsible for:
- Workflow orchestration
- Node execution coordination
- Data flow management
- Error handling

Key components:
- `WorkflowEngine`: Main orchestrator
- `NodeExecutor`: Executes individual nodes
- `ConnectionRouter`: Manages data routing between nodes

### Nodes Module

The nodes module contains implementations of all node types:
- `http`: HTTP Request/Response nodes
- `transform`: Data transformation nodes
- `database`: Database query nodes
- `function`: Custom code execution nodes
- `trigger`: Workflow trigger nodes

Each node type implements the `NodeExecutor` interface.

### Model Module

The model module contains:
- Data structures for workflow representation
- JSON serialization/deserialization
- Data validation functions

### Connections Module

The connections module handles:
- Connection routing logic
- Data flow between nodes
- Multiple connection type support

### Expressions Module

The expressions module provides:
- Expression parsing and evaluation
- Variable resolution
- Built-in function support

## Execution Flow

1. **Workflow Loading**
   - Parse JSON workflow file
   - Validate structure and connections
   - Initialize node executors

2. **Execution Planning**
   - Determine execution order based on connections
   - Identify starting nodes
   - Prepare execution context

3. **Node Execution**
   - Execute nodes in dependency order
   - Route data between connected nodes
   - Handle parallel execution where possible

4. **Result Processing**
   - Collect final output data
   - Handle errors and retries
   - Return execution results

## Compatibility Requirements

### JSON Structure Compatibility
- Support all fields in exported n8n workflow JSON
- Maintain exact field names and structures
- Handle optional fields correctly

### Node Parameter Compatibility
- Accept same parameter structures as n8n nodes
- Validate parameters according to n8n rules
- Produce identical output formats

### Data Flow Compatibility
- Support all connection types (main, AI, etc.)
- Handle paired item tracking correctly
- Maintain data integrity through transformations

### Expression Compatibility
- Support n8n expression syntax
- Provide same built-in functions
- Handle variable resolution identically

## Error Handling

### Error Types
- `ExecutionError`: General execution errors
- `NodeError`: Node-specific errors
- `ConnectionError`: Data routing errors
- `ValidationError`: Workflow validation errors

### Error Propagation
- Preserve error context and stack traces
- Support error recovery where possible
- Provide detailed error information for debugging

## Performance Considerations

### Memory Management
- Reuse data structures where possible
- Minimize allocations during execution
- Efficient handling of binary data

### Concurrency
- Goroutine-based parallel execution
- Connection-safe data structures
- Efficient synchronization primitives

### Caching
- Cache parsed expressions
- Cache node initialization
- Reuse execution contexts where possible