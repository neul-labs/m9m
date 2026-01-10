# Go SDK

The Go SDK provides direct access to the m9m workflow engine with no external dependencies.

## Installation

```go
import "github.com/m9m/m9m/pkg/m9m"
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "github.com/m9m/m9m/pkg/m9m"
)

func main() {
    // Create engine
    engine := m9m.New()

    // Load workflow
    workflow, err := m9m.LoadWorkflow("workflow.json")
    if err != nil {
        log.Fatal(err)
    }

    // Execute
    result, err := engine.Execute(workflow, []m9m.DataItem{
        {JSON: map[string]interface{}{"input": "data"}},
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Result: %+v\n", result.Data)
}
```

## API Reference

### Engine

#### Creating an Engine

```go
// Standard engine
engine := m9m.New()

// Enhanced engine with expression support
engine := m9m.NewEnhanced()

// Reliable engine with retries
engine := m9m.NewReliable(&m9m.ReliableConfig{
    MaxRetries: 3,
    RetryDelay: time.Second,
})
```

#### Executing Workflows

```go
result, err := engine.Execute(workflow, inputData)
if err != nil {
    // Handle error
}

// Access results
for _, item := range result.Data {
    fmt.Println(item.JSON)
}
```

#### Registering Custom Nodes

```go
engine.RegisterNode("custom.myNode", &MyCustomNode{})
```

#### Setting Credentials

```go
credManager := m9m.NewCredentialManager()
engine.SetCredentialManager(credManager)
```

### Workflow

#### Loading Workflows

```go
// From file
workflow, err := m9m.LoadWorkflow("path/to/workflow.json")

// From JSON bytes
workflow, err := m9m.ParseWorkflow(jsonBytes)

// From struct
workflow := &m9m.Workflow{
    Name: "My Workflow",
    Nodes: []m9m.WorkflowNode{
        {
            ID:   "node1",
            Name: "Start",
            Type: "n8n-nodes-base.start",
        },
    },
    Connections: map[string]m9m.NodeConnections{},
}
```

#### Workflow Properties

```go
workflow.Name         // Workflow name
workflow.ID           // Unique identifier
workflow.Active       // Is workflow active
workflow.Nodes        // List of nodes
workflow.Connections  // Node connections
```

### Custom Nodes

Implement the `NodeExecutor` interface:

```go
type MyNode struct {
    m9m.BaseNode
}

func (n *MyNode) Execute(
    input []m9m.DataItem,
    params map[string]interface{},
) ([]m9m.DataItem, error) {
    // Process input data
    result := make([]m9m.DataItem, len(input))
    for i, item := range input {
        result[i] = m9m.DataItem{
            JSON: map[string]interface{}{
                "processed": item.JSON,
            },
        }
    }
    return result, nil
}

func (n *MyNode) Description() m9m.NodeDescription {
    return m9m.NodeDescription{
        Name:        "My Custom Node",
        Description: "Processes data in a custom way",
        Category:    "transform",
        Version:     1,
        Properties: []m9m.NodeProperty{
            {
                Name:        "option",
                Type:        m9m.PropertyTypeString,
                DisplayName: "Option",
                Required:    true,
            },
        },
    }
}

func (n *MyNode) ValidateParameters(params map[string]interface{}) error {
    if _, ok := params["option"]; !ok {
        return errors.New("option is required")
    }
    return nil
}
```

### Data Types

#### DataItem

```go
type DataItem struct {
    JSON       map[string]interface{}
    Binary     map[string]BinaryData
    PairedItem *PairedItemInfo
    Error      *ItemError
}
```

#### ExecutionResult

```go
type ExecutionResult struct {
    Data  []DataItem
    Error string
}

// Check success
if result.Error == "" {
    // Success
}
```

## Testing

```bash
cd pkg/m9m
go test -v ./...
```
