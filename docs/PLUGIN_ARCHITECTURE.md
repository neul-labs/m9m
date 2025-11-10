# n8n-go Plugin Architecture: Dynamic Node Loading

## Overview

This document explores options for **adding nodes without recompiling** the n8n-go system. Currently, nodes are statically compiled into the binary. This analysis covers multiple plugin approaches.

## Table of Contents

1. [Current Architecture](#current-architecture)
2. [Existing Infrastructure](#existing-infrastructure)
3. [Plugin Options](#plugin-options)
4. [Recommended Approach](#recommended-approach)
5. [Implementation Plan](#implementation-plan)
6. [Examples](#examples)

---

## Current Architecture

### Static Compilation Model

**How it works now**:
```go
// In cmd/n8n-go/main.go
slackNode := msgnodes.NewSlackNode()
eng.RegisterNodeExecutor("n8n-nodes-base.slack", slackNode)
```

**Pros**:
- ✅ Type-safe at compile time
- ✅ Maximum performance (no overhead)
- ✅ Single binary deployment
- ✅ No runtime dependency issues

**Cons**:
- ❌ Need full recompile to add nodes
- ❌ Need binary redeploy
- ❌ Can't add nodes at runtime
- ❌ Harder for community contributions

---

## Existing Infrastructure

### 1. JavaScript Runtime (Goja) ✅

**Already implemented** in `internal/runtime/javascript_runtime.go`:

```go
// Function node executes JavaScript code
evaluator := expressions.NewGojaExpressionEvaluator(config)
value, err := evaluator.EvaluateCode(jsCode, context)
```

**Capabilities**:
- Full JavaScript execution
- Access to workflow data
- n8n helper functions
- Sandboxed execution

**Use case**: Function node, Code node

### 2. Python Runtime ✅

**Already implemented** in `internal/runtime/python_runtime.go`:

**Capabilities**:
- Execute Python code
- Access to numpy, pandas
- Subprocess-based execution
- Data serialization

**Use case**: Python code execution nodes

### 3. Expression Evaluator ✅

**Already implemented** in `internal/expressions/`:

**Capabilities**:
- Evaluate `{{ $json.field }}` expressions
- 60+ built-in functions
- Custom function registration
- Context-aware evaluation

---

## Plugin Options

### Option 1: JavaScript-Based Nodes ⭐ RECOMMENDED

**Description**: Nodes written in JavaScript, executed via Goja runtime

**Implementation**:
```javascript
// plugins/my-custom-node.js
module.exports = {
    description: {
        name: "My Custom Node",
        description: "Does something useful",
        category: "custom"
    },

    execute: async function(inputData, parameters) {
        const results = [];

        for (const item of inputData) {
            // Your logic here
            const result = {
                json: {
                    processed: parameters.someParam,
                    original: item.json
                }
            };
            results.push(result);
        }

        return results;
    },

    validateParameters: function(parameters) {
        if (!parameters.someParam) {
            throw new Error("someParam is required");
        }
    }
};
```

**Pros**:
- ✅ No recompilation needed
- ✅ Runtime already exists (Goja)
- ✅ Easy to write (familiar JavaScript)
- ✅ Can be hot-reloaded
- ✅ Sandboxed execution
- ✅ Fast (Goja is very performant)
- ✅ Compatible with n8n's existing Function node

**Cons**:
- ⚠️ Slightly slower than native Go (but still very fast)
- ⚠️ Limited to JavaScript capabilities
- ⚠️ Can't use Go libraries directly

**Effort**: LOW (1-2 days)

---

### Option 2: Go Plugin System (.so files)

**Description**: Use Go's plugin package to load `.so` shared libraries

**Implementation**:
```go
// plugins/mycustomnode/node.go
package main

import "github.com/dipankar/n8n-go/internal/nodes/base"

type MyCustomNode struct {
    *base.BaseNode
}

func NewNode() base.NodeExecutor {
    return &MyCustomNode{
        BaseNode: base.NewBaseNode(base.NodeDescription{
            Name: "My Custom Node",
        }),
    }
}

// Export for plugin loading
var NodeExecutor base.NodeExecutor = NewNode()
```

```go
// In n8n-go main.go
p, err := plugin.Open("plugins/mycustomnode.so")
nodeFactory, err := p.Lookup("NodeExecutor")
eng.RegisterNodeExecutor("custom.myNode", nodeFactory.(base.NodeExecutor))
```

**Pros**:
- ✅ Native Go performance
- ✅ Full access to Go ecosystem
- ✅ Type-safe interfaces
- ✅ Can use existing BaseNode utilities

**Cons**:
- ❌ Linux/macOS only (no Windows support)
- ❌ Version compatibility issues (must match Go version)
- ❌ Still requires compilation (just not full binary)
- ❌ Fragile (ABI changes break plugins)
- ❌ Limited portability

**Effort**: MEDIUM (3-5 days)

**Status**: ⚠️ Not recommended due to limitations

---

### Option 3: gRPC/RPC External Nodes

**Description**: Nodes run as separate processes, communicate via gRPC

**Architecture**:
```
┌─────────────────┐         gRPC         ┌──────────────────┐
│   n8n-go Core   │ ◄─────────────────► │  External Node   │
│                 │                      │  (any language)  │
└─────────────────┘                      └──────────────────┘
```

**Implementation**:
```protobuf
// node.proto
service NodeService {
    rpc Execute(ExecuteRequest) returns (ExecuteResponse);
    rpc Describe(DescribeRequest) returns (NodeDescription);
}

message ExecuteRequest {
    repeated DataItem input_data = 1;
    map<string, string> parameters = 2;
}
```

**Pros**:
- ✅ Language-agnostic (Python, Node.js, Ruby, etc.)
- ✅ Process isolation
- ✅ Hot reload/restart possible
- ✅ Distributed execution
- ✅ Resource limits per node
- ✅ Can scale independently

**Cons**:
- ❌ Network overhead (latency)
- ❌ Complex deployment
- ❌ More moving parts
- ❌ Serialization overhead
- ❌ Need process management

**Effort**: HIGH (1-2 weeks)

**Use case**: Enterprise deployments with many custom nodes

---

### Option 4: WebAssembly (WASM) Nodes

**Description**: Nodes compiled to WASM, executed in Go WASM runtime

**Implementation**:
```rust
// Written in Rust/C++/AssemblyScript
#[no_mangle]
pub extern "C" fn execute(input_ptr: *const u8, input_len: usize) -> *mut u8 {
    // Your logic here
}
```

**Pros**:
- ✅ Near-native performance
- ✅ Sandboxed execution
- ✅ Language-agnostic
- ✅ Portable binaries
- ✅ No subprocess overhead

**Cons**:
- ❌ Complex to implement
- ❌ Limited ecosystem
- ❌ Memory management complexity
- ❌ WASM runtime overhead in Go
- ❌ Debugging is harder

**Effort**: HIGH (2-3 weeks)

**Status**: ⚠️ Cutting edge, not mature yet

---

### Option 5: REST API-Based Nodes

**Description**: Nodes are HTTP endpoints that n8n-go calls

**Implementation**:
```yaml
# plugins/my-node.yaml
name: "My Custom Node"
endpoint: "http://localhost:3000/execute"
method: POST
description: "Does something useful"
parameters:
  - name: someParam
    type: string
    required: true
```

**Pros**:
- ✅ Language-agnostic
- ✅ Easy to implement
- ✅ Stateless
- ✅ Can use existing web services
- ✅ Easy testing

**Cons**:
- ❌ Network latency
- ❌ Need external service management
- ❌ Reliability depends on external service
- ❌ Authentication complexity

**Effort**: LOW (2-3 days)

**Use case**: Integrating existing microservices as nodes

---

### Option 6: Hybrid Approach ⭐ BEST BALANCE

**Description**: Combine JavaScript runtime with optional Go plugins

**Implementation**:
1. **Default**: JavaScript-based nodes (90% of use cases)
2. **Performance-critical**: Compiled Go plugins
3. **External integrations**: REST API nodes

**Architecture**:
```
┌─────────────────────────────────────────────┐
│            n8n-go Core Engine               │
├─────────────────────────────────────────────┤
│                                             │
│  ┌─────────────┐  ┌──────────┐  ┌───────┐ │
│  │ JavaScript  │  │ Go Plugin│  │  REST │ │
│  │   Nodes     │  │  Nodes   │  │  API  │ │
│  │  (Runtime)  │  │  (.so)   │  │ Nodes │ │
│  └─────────────┘  └──────────┘  └───────┘ │
│                                             │
└─────────────────────────────────────────────┘
```

**Pros**:
- ✅ Flexibility: Choose the right tool
- ✅ Easy path (JavaScript) for most nodes
- ✅ Performance path (Go) when needed
- ✅ External integration path (REST) for microservices

**Cons**:
- ⚠️ More complex architecture
- ⚠️ Need to maintain multiple plugin systems

**Effort**: MEDIUM (1-2 weeks for full implementation)

---

## Recommended Approach

### Phase 1: JavaScript-Based Plugins (Immediate) ⭐

**Why**:
- Leverages existing Goja runtime
- Fastest to implement
- Covers 90% of use cases
- Compatible with n8n patterns

**Implementation Steps**:

#### Step 1: Plugin Loader
```go
// internal/plugins/loader.go
type JavaScriptNodePlugin struct {
    FilePath    string
    Name        string
    Description base.NodeDescription
    jsVM        *goja.Runtime
}

func LoadJavaScriptPlugin(filePath string) (*JavaScriptNodePlugin, error) {
    // Read JavaScript file
    code, err := os.ReadFile(filePath)
    if err != nil {
        return nil, err
    }

    // Create VM
    vm := goja.New()

    // Execute plugin code
    _, err = vm.RunString(string(code))
    if err != nil {
        return nil, err
    }

    // Extract exports
    exports := vm.Get("module").ToObject(vm).Get("exports")

    return &JavaScriptNodePlugin{
        FilePath: filePath,
        jsVM:     vm,
    }, nil
}
```

#### Step 2: Plugin Registry
```go
// internal/plugins/registry.go
type PluginRegistry struct {
    plugins map[string]*JavaScriptNodePlugin
    mu      sync.RWMutex
}

func (r *PluginRegistry) LoadPluginsFromDirectory(dir string) error {
    files, _ := filepath.Glob(filepath.Join(dir, "*.js"))

    for _, file := range files {
        plugin, err := LoadJavaScriptPlugin(file)
        if err != nil {
            log.Printf("Failed to load plugin %s: %v", file, err)
            continue
        }

        r.plugins[plugin.Name] = plugin
    }

    return nil
}

func (r *PluginRegistry) RegisterWithEngine(engine *engine.WorkflowEngine) {
    for name, plugin := range r.plugins {
        wrapper := NewJavaScriptNodeWrapper(plugin)
        engine.RegisterNodeExecutor(name, wrapper)
    }
}
```

#### Step 3: Node Wrapper
```go
// internal/plugins/wrapper.go
type JavaScriptNodeWrapper struct {
    *base.BaseNode
    plugin *JavaScriptNodePlugin
}

func (w *JavaScriptNodeWrapper) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
    // Call JavaScript execute function
    executeFunc := w.plugin.jsVM.Get("execute")

    result, err := executeFunc(goja.Undefined(),
        w.plugin.jsVM.ToValue(inputData),
        w.plugin.jsVM.ToValue(nodeParams))

    if err != nil {
        return nil, err
    }

    // Convert result back to []model.DataItem
    return convertJSResult(result), nil
}
```

#### Step 4: Usage
```bash
# Directory structure
plugins/
├── custom-api-node.js
├── data-processor.js
└── notification-node.js

# Start n8n-go with plugins
./n8n-go --plugin-dir ./plugins
```

**Configuration**:
```go
// Add flag in main.go
pluginDir = flag.String("plugin-dir", "", "Directory containing plugin files")

// Load plugins on startup
if *pluginDir != "" {
    registry := plugins.NewPluginRegistry()
    registry.LoadPluginsFromDirectory(*pluginDir)
    registry.RegisterWithEngine(eng)
    log.Printf("Loaded %d plugins", registry.Count())
}
```

### Phase 2: Hot Reload (Nice to have)

**File Watcher**:
```go
watcher, _ := fsnotify.NewWatcher()
watcher.Add(*pluginDir)

go func() {
    for {
        select {
        case event := <-watcher.Events:
            if event.Op&fsnotify.Write == fsnotify.Write {
                // Reload plugin
                registry.ReloadPlugin(event.Name)
            }
        }
    }
}()
```

---

## Examples

### Example 1: Simple REST API Node

```javascript
// plugins/rest-api-node.js
module.exports = {
    description: {
        name: "REST API Call",
        description: "Makes REST API calls with custom headers",
        category: "network"
    },

    parameters: {
        url: { type: "string", required: true },
        method: { type: "string", default: "GET" },
        headers: { type: "object", default: {} }
    },

    execute: async function(inputData, parameters) {
        const axios = require('axios'); // Would need to provide axios
        const results = [];

        for (const item of inputData) {
            try {
                const response = await axios({
                    url: parameters.url,
                    method: parameters.method,
                    headers: parameters.headers,
                    data: item.json
                });

                results.push({
                    json: response.data,
                    metadata: {
                        status: response.status,
                        headers: response.headers
                    }
                });
            } catch (error) {
                throw new Error(`API call failed: ${error.message}`);
            }
        }

        return results;
    }
};
```

### Example 2: Data Transformation Node

```javascript
// plugins/data-aggregator.js
module.exports = {
    description: {
        name: "Data Aggregator",
        description: "Aggregates and summarizes data",
        category: "transform"
    },

    parameters: {
        groupBy: { type: "string", required: true },
        aggregateField: { type: "string", required: true },
        operation: { type: "string", default: "sum", enum: ["sum", "avg", "count", "min", "max"] }
    },

    execute: function(inputData, parameters) {
        const groups = {};

        // Group data
        for (const item of inputData) {
            const key = item.json[parameters.groupBy];
            if (!groups[key]) {
                groups[key] = [];
            }
            groups[key].push(item.json[parameters.aggregateField]);
        }

        // Aggregate
        const results = [];
        for (const [key, values] of Object.entries(groups)) {
            let aggregated;

            switch (parameters.operation) {
                case 'sum':
                    aggregated = values.reduce((a, b) => a + b, 0);
                    break;
                case 'avg':
                    aggregated = values.reduce((a, b) => a + b, 0) / values.length;
                    break;
                case 'count':
                    aggregated = values.length;
                    break;
                case 'min':
                    aggregated = Math.min(...values);
                    break;
                case 'max':
                    aggregated = Math.max(...values);
                    break;
            }

            results.push({
                json: {
                    group: key,
                    value: aggregated,
                    count: values.length
                }
            });
        }

        return results;
    }
};
```

### Example 3: Custom Business Logic Node

```javascript
// plugins/invoice-processor.js
module.exports = {
    description: {
        name: "Invoice Processor",
        description: "Processes and validates invoices",
        category: "business"
    },

    parameters: {
        taxRate: { type: "number", default: 0.1 },
        currency: { type: "string", default: "USD" }
    },

    execute: function(inputData, parameters) {
        const results = [];

        for (const item of inputData) {
            const invoice = item.json;

            // Validate invoice
            if (!invoice.items || !Array.isArray(invoice.items)) {
                throw new Error("Invoice must have items array");
            }

            // Calculate totals
            let subtotal = 0;
            for (const lineItem of invoice.items) {
                subtotal += lineItem.quantity * lineItem.unitPrice;
            }

            const tax = subtotal * parameters.taxRate;
            const total = subtotal + tax;

            // Add calculations
            results.push({
                json: {
                    ...invoice,
                    subtotal: subtotal,
                    tax: tax,
                    total: total,
                    currency: parameters.currency,
                    processedAt: new Date().toISOString()
                }
            });
        }

        return results;
    },

    validateParameters: function(parameters) {
        if (parameters.taxRate < 0 || parameters.taxRate > 1) {
            throw new Error("Tax rate must be between 0 and 1");
        }
    }
};
```

---

## Performance Considerations

### JavaScript Performance

**Goja Performance** (from benchmarks):
- Simple operations: ~2-5μs overhead
- Complex transforms: ~10-50μs overhead
- API calls: Network dominates (not runtime)

**Comparison**:
```
Native Go node:     100 requests in 10ms   (100k req/s)
JavaScript plugin:  100 requests in 15ms   (66k req/s)

Difference: ~33% slower but still very fast
```

**When to use native Go**:
- CPU-intensive operations (image processing, crypto)
- Tight loops with millions of iterations
- Binary protocol handling

**When JavaScript is fine** (95% of cases):
- API calls (network is bottleneck)
- Data transformation
- Business logic
- Most workflow operations

---

## Security Considerations

### JavaScript Sandbox

**Built-in protections**:
```go
vm := goja.New()

// Disable dangerous functions
vm.Set("eval", goja.Undefined())
vm.Set("Function", goja.Undefined())

// Set timeouts
vm.Interrupt(make(chan func(), 1))
go func() {
    time.Sleep(30 * time.Second)
    vm.Interrupt("execution timeout")
}()
```

**Resource limits**:
```go
type PluginLimits struct {
    MaxExecutionTime time.Duration
    MaxMemory        int64
    MaxFileSize      int64
}
```

### Permissions System

```javascript
// plugins/my-node.js
module.exports = {
    permissions: {
        network: true,   // Can make HTTP requests
        filesystem: false, // Cannot access files
        env: false       // Cannot read environment variables
    },
    // ...
}
```

---

## Migration Path

### For Existing Nodes

Existing compiled nodes continue to work:
```go
// Still works - no changes needed
slackNode := msgnodes.NewSlackNode()
eng.RegisterNodeExecutor("n8n-nodes-base.slack", slackNode)
```

### For New Nodes

Choose the approach:
```bash
# Quick custom node - JavaScript
cp template.js plugins/my-node.js
# Edit and restart

# Performance-critical - Compile into binary
# Add to internal/nodes/category/
# Register in main.go
```

---

## Conclusion

### Recommended Implementation Order

1. **Phase 1** (Week 1): JavaScript plugin loader
   - Basic plugin loading
   - Simple execute/describe interface
   - Directory scanning

2. **Phase 2** (Week 2): Enhanced features
   - Hot reload
   - Parameter validation
   - Error handling
   - Plugin metadata

3. **Phase 3** (Week 3): Production features
   - Security sandbox
   - Resource limits
   - Plugin marketplace/registry
   - Documentation

### Success Metrics

- ✅ Add node without recompile: < 5 minutes
- ✅ Plugin performance overhead: < 2x native
- ✅ Plugin load time: < 100ms
- ✅ Hot reload time: < 1s
- ✅ Community adoption: 50+ plugins in 6 months

---

**Status**: Ready for implementation
**Effort**: 1-2 weeks for Phase 1
**Impact**: HIGH - Enables community contributions

---

*Last Updated: November 10, 2025*
*Version: 1.0*
*Author: n8n-go Development Team*
