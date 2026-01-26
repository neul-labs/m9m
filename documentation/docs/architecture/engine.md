# Workflow Engine

Deep dive into the workflow execution engine.

## Overview

The workflow engine orchestrates the execution of workflows:

```
Workflow JSON → Parse → Build Graph → Execute Nodes → Return Results
```

## Engine Components

### Orchestrator

Coordinates workflow execution:

```go
type WorkflowEngine interface {
    Execute(ctx context.Context, workflow Workflow, input []DataItem) (*ExecutionResult, error)
    ExecuteAsync(ctx context.Context, workflow Workflow, input []DataItem) (string, error)
    RegisterNodeExecutor(nodeType string, executor NodeExecutor)
    GetNodeExecutor(nodeType string) (NodeExecutor, error)
}
```

### Node Registry

Stores and retrieves node implementations:

```go
type NodeRegistry interface {
    Register(nodeType string, executor NodeExecutor)
    Get(nodeType string) (NodeExecutor, bool)
    List() []NodeDescription
}
```

### Expression Evaluator

Processes dynamic expressions:

```go
type ExpressionEvaluator interface {
    Evaluate(expression string, context EvaluationContext) (interface{}, error)
}
```

## Execution Flow

### 1. Workflow Parsing

```json
{
  "name": "My Workflow",
  "nodes": [...],
  "connections": {...}
}
```

Parse and validate:

- Check required fields
- Validate node types exist
- Verify connections are valid

### 2. Graph Construction

Build directed acyclic graph (DAG):

```
      Start
        │
        ▼
   HTTP Request
        │
    ┌───┴───┐
    ▼       ▼
 Filter  Transform
    │       │
    └───┬───┘
        ▼
      Email
```

### 3. Topological Sort

Determine execution order:

```
1. Start (no dependencies)
2. HTTP Request (depends on Start)
3. Filter (depends on HTTP Request)
4. Transform (depends on HTTP Request)
5. Email (depends on Filter AND Transform)
```

### 4. Node Execution

For each node in order:

```go
func executeNode(node Node, inputData []DataItem) ([]DataItem, error) {
    // 1. Get executor
    executor := registry.Get(node.Type)

    // 2. Resolve expressions in parameters
    params := resolveExpressions(node.Parameters, inputData)

    // 3. Get credentials if needed
    creds := getCredentials(node.Credentials)

    // 4. Execute
    output, err := executor.Execute(inputData, params)

    // 5. Handle errors
    if err != nil {
        return handleError(node, err)
    }

    return output, nil
}
```

### 5. Data Propagation

Pass output to connected nodes:

```go
func propagateData(node Node, output []DataItem, connections map[string]Connection) {
    for _, conn := range connections[node.Name] {
        targetNode := getNode(conn.Node)
        targetNode.AddInput(output)
    }
}
```

## Expression Resolution

### Context Building

```go
type EvaluationContext struct {
    JSON       map[string]interface{}  // Current item
    Item       int                     // Item index
    Node       map[string]NodeOutput   // All nodes output
    Input      InputHelper             // Input helpers
    Env        map[string]string       // Environment
    Workflow   WorkflowInfo           // Workflow metadata
    Execution  ExecutionInfo          // Execution metadata
}
```

### Expression Types

| Expression | Description |
|------------|-------------|
| `{{ $json.field }}` | Current item field |
| `{{ $item }}` | Item index |
| `{{ $node["Name"].json.field }}` | Other node's output |
| `{{ $env.VAR }}` | Environment variable |
| `{{ $now }}` | Current timestamp |

### Evaluation Process

```go
func evaluateExpression(expr string, ctx EvaluationContext) (interface{}, error) {
    // 1. Parse expression
    ast := parse(expr)

    // 2. Resolve variables
    for _, variable := range ast.Variables {
        value := ctx.Resolve(variable)
        ast.Substitute(variable, value)
    }

    // 3. Evaluate JavaScript
    result := jsRuntime.Eval(ast.Code)

    return result, nil
}
```

## Error Handling

### Error Types

| Type | Description | Handling |
|------|-------------|----------|
| Validation | Invalid parameters | Stop execution |
| Execution | Node failure | Retry or continue |
| Timeout | Execution too slow | Cancel and fail |
| Resource | Out of memory | Stop execution |

### Retry Logic

```go
func executeWithRetry(node Node, input []DataItem, maxRetries int) ([]DataItem, error) {
    var lastErr error

    for attempt := 0; attempt < maxRetries; attempt++ {
        output, err := executeNode(node, input)
        if err == nil {
            return output, nil
        }

        lastErr = err

        // Check if retryable
        if !isRetryable(err) {
            return nil, err
        }

        // Wait before retry
        time.Sleep(backoff(attempt))
    }

    return nil, lastErr
}
```

### Continue on Error

```json
{
  "nodes": [
    {
      "name": "May Fail",
      "type": "n8n-nodes-base.httpRequest",
      "continueOnFail": true
    }
  ]
}
```

## Parallel Execution

### Independent Nodes

Execute independent nodes concurrently:

```
      Start
        │
        ▼
   ┌────┴────┐
   ▼         ▼
Node A    Node B    (parallel)
   │         │
   └────┬────┘
        ▼
     Node C
```

### Implementation

```go
func executeParallel(nodes []Node, input []DataItem) (map[string][]DataItem, error) {
    results := make(map[string][]DataItem)
    errors := make(chan error, len(nodes))

    var wg sync.WaitGroup
    var mu sync.Mutex

    for _, node := range nodes {
        wg.Add(1)
        go func(n Node) {
            defer wg.Done()

            output, err := executeNode(n, input)
            if err != nil {
                errors <- err
                return
            }

            mu.Lock()
            results[n.Name] = output
            mu.Unlock()
        }(node)
    }

    wg.Wait()
    close(errors)

    // Check for errors
    for err := range errors {
        return nil, err
    }

    return results, nil
}
```

## Resource Management

### Context Cancellation

```go
func executeWithTimeout(ctx context.Context, workflow Workflow) (*Result, error) {
    ctx, cancel := context.WithTimeout(ctx, workflow.Timeout)
    defer cancel()

    resultChan := make(chan *Result)
    errChan := make(chan error)

    go func() {
        result, err := execute(ctx, workflow)
        if err != nil {
            errChan <- err
            return
        }
        resultChan <- result
    }()

    select {
    case result := <-resultChan:
        return result, nil
    case err := <-errChan:
        return nil, err
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}
```

### Memory Management

- Object pooling for data items
- Stream processing for large datasets
- Garbage collection tuning

## Execution Modes

### Synchronous

Wait for completion:

```go
result, err := engine.Execute(ctx, workflow, input)
```

### Asynchronous

Return immediately:

```go
jobID, err := engine.ExecuteAsync(ctx, workflow, input)
// Later...
result, err := engine.GetResult(ctx, jobID)
```

### Webhook Mode

Execute on HTTP request:

```go
func handleWebhook(w http.ResponseWriter, r *http.Request) {
    input := parseRequest(r)
    result, err := engine.Execute(ctx, workflow, input)

    if workflow.ResponseMode == "lastNode" {
        json.NewEncoder(w).Encode(result.Data)
    } else {
        w.WriteHeader(http.StatusAccepted)
    }
}
```

## Metrics

The engine exposes metrics:

| Metric | Type | Description |
|--------|------|-------------|
| `execution_total` | Counter | Total executions |
| `execution_duration` | Histogram | Execution time |
| `node_execution_duration` | Histogram | Per-node time |
| `execution_errors` | Counter | Failed executions |
| `active_executions` | Gauge | Currently running |

## Best Practices

### Workflow Design

1. **Keep workflows simple** - Split complex logic
2. **Handle errors** - Use try/catch patterns
3. **Set timeouts** - Prevent runaway executions
4. **Test thoroughly** - Use sample data

### Performance

1. **Minimize node count** - Consolidate where possible
2. **Use async for long tasks** - Don't block API
3. **Cache when possible** - Reduce external calls
4. **Monitor metrics** - Identify bottlenecks
