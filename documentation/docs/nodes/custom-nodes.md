# Creating Custom Nodes

Build custom nodes to extend m9m with your own integrations.

## Node Structure

Every node implements the `NodeExecutor` interface:

```go
type NodeExecutor interface {
    Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error)
    Description() NodeDescription
    ValidateParameters(params map[string]interface{}) error
}
```

## Basic Node Template

```go
package custom

import (
    "github.com/yourusername/m9m/internal/nodes/base"
    "github.com/yourusername/m9m/internal/model"
)

type MyCustomNode struct {
    *base.BaseNode
}

func NewMyCustomNode() *MyCustomNode {
    return &MyCustomNode{
        BaseNode: base.NewBaseNode(base.NodeDescription{
            Name:        "My Custom Node",
            Description: "Performs custom processing",
            Category:    "custom",
        }),
    }
}

func (n *MyCustomNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
    results := make([]model.DataItem, 0)

    for _, item := range inputData {
        // Process each item
        processed := model.DataItem{
            JSON: map[string]interface{}{
                "original": item.JSON,
                "processed": true,
            },
        }
        results = append(results, processed)
    }

    return results, nil
}

func (n *MyCustomNode) ValidateParameters(params map[string]interface{}) error {
    // Add validation logic
    return nil
}
```

## Using Parameters

Access node parameters with helper methods:

```go
func (n *MyCustomNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
    // Get string parameter
    message := n.GetStringParameter(nodeParams, "message", "default")

    // Get integer parameter
    limit := n.GetIntParameter(nodeParams, "limit", 100)

    // Get boolean parameter
    enabled := n.GetBoolParameter(nodeParams, "enabled", false)

    // Get any parameter
    config := n.GetParameter(nodeParams, "config", nil)

    // Use parameters in processing
    for _, item := range inputData {
        if enabled {
            // Process with parameters
        }
    }

    return inputData, nil
}
```

## HTTP Integration Node

Example node that calls an external API:

```go
package custom

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/yourusername/m9m/internal/nodes/base"
    "github.com/yourusername/m9m/internal/model"
)

type APIClientNode struct {
    *base.BaseNode
    client *http.Client
}

func NewAPIClientNode() *APIClientNode {
    return &APIClientNode{
        BaseNode: base.NewBaseNode(base.NodeDescription{
            Name:        "API Client",
            Description: "Calls external API",
            Category:    "http",
        }),
        client: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

func (n *APIClientNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
    url := n.GetStringParameter(nodeParams, "url", "")
    if url == "" {
        return nil, fmt.Errorf("url is required")
    }

    results := make([]model.DataItem, 0)

    for _, item := range inputData {
        // Build request
        req, err := http.NewRequest("GET", url, nil)
        if err != nil {
            return nil, fmt.Errorf("failed to create request: %w", err)
        }

        // Add headers
        req.Header.Set("Accept", "application/json")

        // Make request
        resp, err := n.client.Do(req)
        if err != nil {
            return nil, fmt.Errorf("request failed: %w", err)
        }
        defer resp.Body.Close()

        // Parse response
        var data map[string]interface{}
        if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
            return nil, fmt.Errorf("failed to parse response: %w", err)
        }

        results = append(results, model.DataItem{
            JSON: data,
        })
    }

    return results, nil
}

func (n *APIClientNode) ValidateParameters(params map[string]interface{}) error {
    url := n.GetStringParameter(params, "url", "")
    if url == "" {
        return fmt.Errorf("url parameter is required")
    }
    return nil
}
```

## Database Node

Example database integration:

```go
package custom

import (
    "database/sql"
    "fmt"

    "github.com/yourusername/m9m/internal/nodes/base"
    "github.com/yourusername/m9m/internal/model"
    _ "github.com/lib/pq"
)

type DatabaseNode struct {
    *base.BaseNode
}

func NewDatabaseNode() *DatabaseNode {
    return &DatabaseNode{
        BaseNode: base.NewBaseNode(base.NodeDescription{
            Name:        "Database Query",
            Description: "Execute database queries",
            Category:    "database",
        }),
    }
}

func (n *DatabaseNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
    connectionString := n.GetStringParameter(nodeParams, "connectionString", "")
    query := n.GetStringParameter(nodeParams, "query", "")

    db, err := sql.Open("postgres", connectionString)
    if err != nil {
        return nil, fmt.Errorf("failed to connect: %w", err)
    }
    defer db.Close()

    rows, err := db.Query(query)
    if err != nil {
        return nil, fmt.Errorf("query failed: %w", err)
    }
    defer rows.Close()

    columns, _ := rows.Columns()
    results := make([]model.DataItem, 0)

    for rows.Next() {
        values := make([]interface{}, len(columns))
        valuePtrs := make([]interface{}, len(columns))
        for i := range values {
            valuePtrs[i] = &values[i]
        }

        rows.Scan(valuePtrs...)

        row := make(map[string]interface{})
        for i, col := range columns {
            row[col] = values[i]
        }

        results = append(results, model.DataItem{JSON: row})
    }

    return results, nil
}
```

## Node with Credentials

Access credentials securely:

```go
func (n *MyNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
    // Get credentials from context
    creds, ok := nodeParams["credentials"].(map[string]interface{})
    if !ok {
        return nil, fmt.Errorf("credentials not provided")
    }

    apiKey, ok := creds["apiKey"].(string)
    if !ok {
        return nil, fmt.Errorf("apiKey not found in credentials")
    }

    // Use credentials
    client := NewAPIClient(apiKey)
    // ...
}
```

## Registering Nodes

Register your node in `cmd/m9m/main.go`:

```go
func registerNodeTypes(engine engine.WorkflowEngine) {
    // Built-in nodes
    engine.RegisterNodeExecutor("n8n-nodes-base.start", core.NewStartNode())

    // Custom nodes
    customNode := custom.NewMyCustomNode()
    engine.RegisterNodeExecutor("custom.myNode", customNode)

    apiClient := custom.NewAPIClientNode()
    engine.RegisterNodeExecutor("custom.apiClient", apiClient)
}
```

## Node Metadata

Provide detailed metadata for UI:

```go
func (n *MyCustomNode) GetMetadata() base.NodeMetadata {
    return base.NodeMetadata{
        Name:        "myCustomNode",
        DisplayName: "My Custom Node",
        Description: "Performs custom data processing",
        Version:     1,
        Defaults: map[string]interface{}{
            "name": "My Custom Node",
        },
        Inputs:  []string{"main"},
        Outputs: []string{"main"},
        Properties: []base.NodeProperty{
            {
                DisplayName: "Message",
                Name:        "message",
                Type:        "string",
                Default:     "",
                Description: "Message to process",
                Required:    true,
            },
            {
                DisplayName: "Operation",
                Name:        "operation",
                Type:        "options",
                Default:     "process",
                Options: []base.Option{
                    {Name: "Process", Value: "process"},
                    {Name: "Transform", Value: "transform"},
                    {Name: "Validate", Value: "validate"},
                },
            },
            {
                DisplayName: "Limit",
                Name:        "limit",
                Type:        "number",
                Default:     100,
                Description: "Maximum items to process",
            },
        },
    }
}
```

## Error Handling

Implement proper error handling:

```go
func (n *MyNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
    results := make([]model.DataItem, 0)

    for i, item := range inputData {
        result, err := n.processItem(item, nodeParams)
        if err != nil {
            // Option 1: Return error (stops workflow)
            return nil, fmt.Errorf("failed to process item %d: %w", i, err)

            // Option 2: Continue with error data
            results = append(results, model.DataItem{
                JSON: map[string]interface{}{
                    "error": err.Error(),
                    "item":  item.JSON,
                },
            })
            continue
        }

        results = append(results, result)
    }

    return results, nil
}
```

## Testing Nodes

Write comprehensive tests:

```go
package custom

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/yourusername/m9m/internal/model"
)

func TestMyCustomNode_Execute(t *testing.T) {
    tests := []struct {
        name      string
        input     []model.DataItem
        params    map[string]interface{}
        expected  []model.DataItem
        expectErr bool
    }{
        {
            name: "basic processing",
            input: []model.DataItem{
                {JSON: map[string]interface{}{"value": 1}},
            },
            params: map[string]interface{}{
                "message": "test",
            },
            expected: []model.DataItem{
                {JSON: map[string]interface{}{"value": 1, "processed": true}},
            },
            expectErr: false,
        },
        {
            name:      "empty input",
            input:     []model.DataItem{},
            params:    map[string]interface{}{},
            expected:  []model.DataItem{},
            expectErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            node := NewMyCustomNode()

            result, err := node.Execute(tt.input, tt.params)

            if tt.expectErr {
                assert.Error(t, err)
                return
            }

            require.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}

func TestMyCustomNode_ValidateParameters(t *testing.T) {
    node := NewMyCustomNode()

    err := node.ValidateParameters(nil)
    assert.NoError(t, err)

    err = node.ValidateParameters(map[string]interface{}{
        "message": "valid",
    })
    assert.NoError(t, err)
}
```

## Best Practices

1. **Use BaseNode** for common functionality
2. **Validate parameters** before execution
3. **Handle errors gracefully** with context
4. **Write comprehensive tests**
5. **Document node parameters**
6. **Use meaningful error messages**
7. **Clean up resources** (connections, files)
8. **Support cancellation** via context

## Next Steps

- [Node Overview](overview.md) - Understand the node system
- [Testing](../reference/configuration.md) - Run tests
- [Contributing](https://github.com/m9m/m9m/blob/main/CONTRIBUTING.md) - Contribute nodes
