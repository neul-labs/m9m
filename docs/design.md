# n8n-go Core Abstractions Design

This document details the core abstractions and interfaces that form the foundation of n8n-go.

## Overview

The n8n-go architecture is built around several core abstractions that provide a clean separation of concerns while maintaining compatibility with n8n's workflow model.

## Core Interfaces

### WorkflowEngine Interface

The `WorkflowEngine` is the primary interface for executing workflows:

```go
type WorkflowEngine interface {
    // ExecuteWorkflow executes a complete workflow with optional input data
    ExecuteWorkflow(workflow *Workflow, inputData []DataItem) (*ExecutionResult, error)
    
    // ExecuteNode executes a single node with input data
    ExecuteNode(node *Node, inputData []DataItem) ([]DataItem, error)
    
    // RegisterNodeType registers a new node type with its executor
    RegisterNodeType(nodeType string, executor NodeExecutor)
    
    // GetRegisteredNodeTypes returns a list of all registered node types
    GetRegisteredNodeTypes() []string
}
```

### NodeExecutor Interface

The `NodeExecutor` interface defines how individual nodes are executed:

```go
type NodeExecutor interface {
    // Execute processes input data and returns output data
    Execute(ctx ExecutionContext, inputData []DataItem) ([]DataItem, error)
    
    // Description returns metadata about the node type
    Description() NodeDescription
    
    // ValidateParameters validates node parameters
    ValidateParameters(params map[string]interface{}) error
}
```

### ExecutionContext Interface

The `ExecutionContext` provides the runtime context for node execution:

```go
type ExecutionContext interface {
    // GetWorkflow returns the current workflow being executed
    GetWorkflow() *Workflow
    
    // GetNode returns a node by its name
    GetNode(name string) *Node
    
    // EvaluateExpression evaluates an expression in the current context
    EvaluateExpression(expression string, itemIndex int) (interface{}, error)
    
    // GetCredential retrieves credentials for a node
    GetCredential(nodeName string, credentialType string) (Credential, error)
    
    // GetSetting retrieves a workflow setting
    GetSetting(key string) interface{}
    
    // Logger returns a logger for the current execution
    Logger() Logger
}
```

## Data Models

### Workflow Structure

The `Workflow` struct represents an entire n8n workflow:

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

### Node Structure

The `Node` struct represents an individual node in a workflow:

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

### Data Flow Structures

Data flows between nodes using standardized structures:

```go
type DataItem struct {
    JSON       map[string]interface{} `json:"json"`
    Binary     map[string]BinaryData  `json:"binary,omitempty"`
    PairedItem interface{}            `json:"pairedItem,omitempty"`
    Error      error                  `json:"error,omitempty"`
}

type BinaryData struct {
    Data         string `json:"data"`
    MimeType     string `json:"mimeType"`
    FileSize     string `json:"fileSize,omitempty"`
    FileName     string `json:"fileName,omitempty"`
    Directory    string `json:"directory,omitempty"`
    FileExtension string `json:"fileExtension,omitempty"`
}
```

## Connection Management

### Connection Router

The connection system handles data routing between nodes:

```go
type ConnectionRouter interface {
    // RouteData routes data from a source node to target nodes
    RouteData(sourceNode string, data []DataItem) (map[string][]DataItem, error)
    
    // GetConnections returns connections for a node
    GetConnections(nodeName string) *Connections
    
    // ValidateConnections validates all connections in a workflow
    ValidateConnections() error
}
```

### Connection Structures

Connection data structures mirror n8n's format:

```go
type Connections struct {
    Main [][]Connection `json:"main,omitempty"`
    // Other connection types as needed
}

type Connection struct {
    Node  string `json:"node"`
    Type  string `json:"type"`
    Index int    `json:"index"`
}
```

## Expression System

### Expression Evaluator

The expression system evaluates n8n-style expressions:

```go
type ExpressionEvaluator interface {
    // Evaluate evaluates an expression with given context
    Evaluate(expression string, context map[string]interface{}) (interface{}, error)
    
    // RegisterFunction registers a custom function
    RegisterFunction(name string, function interface{})
    
    // Validate validates an expression syntax
    Validate(expression string) error
}
```

## Credential Management

### Credential Provider

The credential system manages secure access to external services:

```go
type CredentialProvider interface {
    // GetCredential retrieves a credential by name and type
    GetCredential(name string, credentialType string) (Credential, error)
    
    // SetCredential stores a credential
    SetCredential(name string, credential Credential) error
    
    // DeleteCredential removes a credential
    DeleteCredential(name string, credentialType string) error
}
```

### Credential Structure

Credentials are stored securely:

```go
type Credential struct {
    ID       string                 `json:"id,omitempty"`
    Name     string                 `json:"name"`
    Type     string                 `json:"type"`
    Data     map[string]interface{} `json:"data"`
    HomeProject *ProjectSharingData `json:"homeProject,omitempty"`
}
```

## Error Handling

### Error Types

Custom error types provide detailed error information:

```go
type ExecutionError struct {
    NodeName    string
    NodeType    string
    ErrorMessage string
    ErrorData   map[string]interface{}
    Cause       error
}

type NodeError struct {
    NodeName    string
    ErrorMessage string
    ErrorData   map[string]interface{}
    Cause       error
}

type ConnectionError struct {
    SourceNode string
    TargetNode string
    ErrorMessage string
    Cause      error
}
```

## Logging

### Logger Interface

A standardized logging interface:

```go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    
    With(fields ...Field) Logger
}
```

## Module Organization

### Engine Module

The engine module contains the core workflow execution logic:

```
internal/engine/
├── engine.go          // Main workflow engine implementation
├── executor.go        // Node execution coordination
├── workflow.go        // Workflow lifecycle management
└── context.go         // Execution context implementation
```

### Nodes Module

The nodes module contains implementations of all node types:

```
internal/nodes/
├── base/
│   └── node.go        // Base node interface and utilities
├── http/
│   ├── request.go     // HTTP Request node
│   └── response.go    // HTTP Response node
├── transform/
│   ├── set.go         // Set node
│   ├── item_lists.go  // Item Lists node
│   └── function.go    // Function node
└── ...
```

### Model Module

The model module contains data structures and JSON handling:

```
internal/model/
├── workflow.go        // Workflow data structures
├── node.go            // Node data structures
├── connection.go      // Connection data structures
├── data.go            // Data item structures
└── json.go            // JSON serialization utilities
```

### Connections Module

The connections module handles data routing:

```
internal/connections/
├── router.go          // Connection routing implementation
├── validator.go       // Connection validation
└── resolver.go        // Connection resolution
```

### Expressions Module

The expressions module handles expression evaluation:

```
internal/expressions/
├── evaluator.go       // Expression evaluation engine
├── parser.go          // Expression parsing
├── functions.go       // Built-in functions
└── variables.go       // Variable resolution
```

## Implementation Patterns

### Dependency Injection

Components use dependency injection for loose coupling:

```go
type WorkflowEngineImpl struct {
    nodeRegistry    NodeRegistry
    connectionRouter ConnectionRouter
    expressionEvaluator ExpressionEvaluator
    credentialProvider  CredentialProvider
    logger             Logger
}
```

### Factory Pattern

Node executors are created using a factory pattern:

```go
type NodeExecutorFactory interface {
    CreateExecutor(nodeType string) (NodeExecutor, error)
}
```

### Plugin Architecture

The system supports plugin-style extensions:

```go
type Plugin interface {
    Name() string
    Version() string
    Initialize(engine WorkflowEngine) error
    GetNodeExecutors() map[string]NodeExecutor
}
```

## Concurrency Model

The engine supports concurrent workflow execution:

```go
type ConcurrentWorkflowEngine interface {
    WorkflowEngine
    
    // ExecuteWorkflowAsync executes a workflow asynchronously
    ExecuteWorkflowAsync(workflow *Workflow, inputData []DataItem) (chan *ExecutionResult, error)
    
    // ExecuteMultipleWorkflows executes multiple workflows concurrently
    ExecuteMultipleWorkflows(workflows []*WorkflowExecutionRequest) ([]*ExecutionResult, error)
}
```

This design provides a solid foundation for building a high-performance, compatible n8n implementation in Go while maintaining the flexibility to extend and optimize as needed.