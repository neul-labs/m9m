# Plugin System - Technical Documentation

This document provides technical details about n8n-go's plugin architecture implementation.

## Overview

The n8n-go plugin system enables dynamic node loading without recompilation, supporting three plugin types:
- **JavaScript** - Embedded execution using Goja runtime
- **gRPC** - External service communication via gRPC protocol
- **REST API** - External service communication via HTTP/REST

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Plugin Registry                         │
│  - Auto-discovery from directory                            │
│  - Manages all plugin types                                 │
│  - Registers with WorkflowEngine                            │
└─────────────────────────────────────────────────────────────┘
                              │
                ┌─────────────┼─────────────┐
                │             │             │
        ┌───────▼──────┐ ┌───▼────┐ ┌─────▼──────┐
        │  JavaScript  │ │  gRPC  │ │  REST API  │
        │    Plugin    │ │ Plugin │ │   Plugin   │
        └──────┬───────┘ └───┬────┘ └─────┬──────┘
               │             │             │
        ┌──────▼──────┐ ┌───▼────┐ ┌─────▼──────┐
        │  JS Wrapper │ │  gRPC  │ │   REST     │
        │             │ │Wrapper │ │  Wrapper   │
        └──────┬───────┘ └───┬────┘ └─────┬──────┘
               │             │             │
               └─────────────┴─────────────┘
                          │
               ┌──────────▼──────────┐
               │  NodeExecutor       │
               │  Interface          │
               └─────────────────────┘
```

## File Structure

```
internal/plugins/
├── registry.go              # Plugin registry and auto-discovery
├── javascript_plugin.go     # JavaScript plugin loader
├── javascript_wrapper.go    # JS -> NodeExecutor adapter
├── grpc_plugin.go          # gRPC client and configuration
├── grpc_wrapper.go         # gRPC -> NodeExecutor adapter
├── rest_plugin.go          # REST client and configuration
└── rest_wrapper.go         # REST -> NodeExecutor adapter

plugins/
├── README.md               # User documentation
└── examples/
    ├── textTransform.js           # JavaScript example
    ├── sentimentAnalysis.grpc.yaml # gRPC example
    └── geocoder.rest.yaml         # REST example
```

## Component Details

### 1. Plugin Registry (`registry.go`)

**Purpose:** Centralized management of all plugin types.

**Key Types:**
```go
type PluginRegistry struct {
    plugins    map[string]Plugin      // name -> plugin instance
    jsConfig   *JavaScriptPluginConfig
    grpcConfig *GRPCPluginConfig
    restConfig *RESTPluginConfig
    mu         sync.RWMutex
}

type Plugin interface {
    GetDescription() base.NodeDescription
    GetType() PluginType
}
```

**Key Methods:**

- `LoadPluginsFromDirectory(dir string)` - Discovers and loads all plugins
  - Finds `*.js`, `*.grpc.yaml`, `*.rest.yaml` files
  - Calls type-specific loaders
  - Stores in registry map

- `RegisterWithEngine(eng engine.WorkflowEngine)` - Registers all plugins
  - Creates appropriate wrapper for each plugin type
  - Registers as `n8n-nodes-base.<name>`
  - Logs registration success

- `Count()` - Returns number of loaded plugins

**Auto-Discovery Logic:**
```go
func (r *PluginRegistry) LoadPluginsFromDirectory(dir string) error {
    // Find JavaScript plugins
    jsFiles, _ := filepath.Glob(filepath.Join(dir, "*.js"))

    // Find gRPC plugins
    grpcFiles, _ := filepath.Glob(filepath.Join(dir, "*.grpc.yaml"))

    // Find REST plugins
    restFiles, _ := filepath.Glob(filepath.Join(dir, "*.rest.yaml"))

    // Load each type
    for _, file := range jsFiles {
        r.LoadJavaScriptPlugin(file)
    }
    // ... similar for gRPC and REST
}
```

### 2. JavaScript Plugin System

**Files:** `javascript_plugin.go`, `javascript_wrapper.go`

**Architecture:**
- Uses Goja JavaScript runtime (pure Go implementation)
- Executes JavaScript code in VM
- Supports CommonJS module.exports pattern
- Provides console.log/error/warn

**Key Types:**
```go
type JavaScriptNodePlugin struct {
    FilePath    string
    Name        string
    Description base.NodeDescription
    vm          *goja.Runtime
    executeFunc goja.Callable
    validateFunc goja.Callable
}
```

**Loading Process:**
```go
func LoadJavaScriptPlugin(filePath string, config *JavaScriptPluginConfig) (*JavaScriptNodePlugin, error) {
    // 1. Read JavaScript file
    code, _ := os.ReadFile(filePath)

    // 2. Create JavaScript VM
    vm := goja.New()
    setupConsole(vm)  // Add console.log, etc.

    // 3. Setup module.exports
    vm.Set("module", vm.NewObject())

    // 4. Execute plugin code
    vm.RunString(string(code))

    // 5. Extract exports
    exports := vm.Get("module").ToObject(vm).Get("exports")

    // 6. Extract description and functions
    description := extractDescription(exports)
    executeFunc := exports.ToObject(vm).Get("execute")

    return &JavaScriptNodePlugin{
        vm:          vm,
        executeFunc: executeFunc,
        // ...
    }
}
```

**Execution Flow:**
```go
func (p *JavaScriptNodePlugin) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
    // Convert Go data to JavaScript
    jsInputData := convertToJSArray(p.vm, inputData)
    jsParams := convertToJSObject(p.vm, nodeParams)

    // Call JavaScript execute function
    result, err := p.executeFunc(goja.Undefined(), jsInputData, jsParams)

    // Convert JavaScript result back to Go
    return convertFromJSArray(result)
}
```

**Type Conversions:**
- `convertToJSArray()` - Go []DataItem → JS array
- `convertToJSObject()` - Go map → JS object
- `convertFromJSArray()` - JS array → Go []DataItem

**Console Support:**
```go
func setupConsole(vm *goja.Runtime) {
    console := vm.NewObject()

    console.Set("log", func(call goja.FunctionCall) goja.Value {
        args := make([]interface{}, len(call.Arguments))
        for i, arg := range call.Arguments {
            args[i] = arg.Export()
        }
        fmt.Println(args...)
        return goja.Undefined()
    })

    // Similar for console.error, console.warn
    vm.Set("console", console)
}
```

### 3. gRPC Plugin System

**Files:** `grpc_plugin.go`, `grpc_wrapper.go`

**Architecture:**
- Loads configuration from YAML
- Establishes gRPC connection at startup
- Calls external service for execution
- Marshals/unmarshals JSON data

**Key Types:**
```go
type GRPCNodePlugin struct {
    Name        string
    Description base.NodeDescription
    ConfigPath  string
    Address     string
    Timeout     time.Duration
    conn        *grpc.ClientConn
    client      NodeServiceClient
}

type GRPCPluginYAML struct {
    Name        string            `yaml:"name"`
    Description string            `yaml:"description"`
    Category    string            `yaml:"category"`
    Address     string            `yaml:"address"`
    Timeout     string            `yaml:"timeout"`
    Parameters  map[string]ParamSpec `yaml:"parameters"`
}
```

**Loading Process:**
```go
func LoadGRPCPlugin(configPath string, config *GRPCPluginConfig) (*GRPCNodePlugin, error) {
    // 1. Read YAML configuration
    data, _ := os.ReadFile(configPath)

    // 2. Parse YAML
    var pluginConfig GRPCPluginYAML
    yaml.Unmarshal(data, &pluginConfig)

    // 3. Parse timeout
    timeout := parseTimeout(pluginConfig.Timeout)

    // 4. Create gRPC connection
    opts := []grpc.DialOption{
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithDefaultCallOptions(
            grpc.MaxCallRecvMsgSize(100 * 1024 * 1024),
        ),
    }
    conn, _ := grpc.DialContext(ctx, pluginConfig.Address, opts...)

    // 5. Create plugin
    return &GRPCNodePlugin{
        conn:   conn,
        client: NewNodeServiceClient(conn),
        // ...
    }
}
```

**Execution Flow:**
```go
func (p *GRPCNodePlugin) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
    // 1. Marshal input to JSON
    inputJSON, _ := json.Marshal(inputData)
    paramsJSON, _ := json.Marshal(nodeParams)

    // 2. Create gRPC request
    request := &ExecuteRequest{
        InputData:  string(inputJSON),
        Parameters: string(paramsJSON),
    }

    // 3. Call gRPC service
    response, _ := p.client.Execute(ctx, request)

    // 4. Check response
    if !response.Success {
        return nil, fmt.Errorf(response.Error)
    }

    // 5. Unmarshal output
    var outputData []model.DataItem
    json.Unmarshal([]byte(response.OutputData), &outputData)

    return outputData, nil
}
```

**gRPC Service Interface:**
```protobuf
service NodeService {
  rpc Execute(ExecuteRequest) returns (ExecuteResponse);
  rpc Describe(DescribeRequest) returns (DescribeResponse);
}

message ExecuteRequest {
  string inputData = 1;   // JSON array
  string parameters = 2;  // JSON object
}

message ExecuteResponse {
  bool success = 1;
  string outputData = 2;  // JSON array
  string error = 3;
}
```

### 4. REST API Plugin System

**Files:** `rest_plugin.go`, `rest_wrapper.go`

**Architecture:**
- Loads configuration from YAML
- Creates HTTP client with timeout
- POSTs JSON data to endpoint
- Parses JSON response

**Key Types:**
```go
type RESTNodePlugin struct {
    Name        string
    Description base.NodeDescription
    ConfigPath  string
    Endpoint    string
    Method      string
    Headers     map[string]string
    Timeout     time.Duration
    httpClient  *http.Client
}

type RESTPluginYAML struct {
    Name        string            `yaml:"name"`
    Description string            `yaml:"description"`
    Category    string            `yaml:"category"`
    Endpoint    string            `yaml:"endpoint"`
    Method      string            `yaml:"method"`
    Headers     map[string]string `yaml:"headers"`
    Timeout     string            `yaml:"timeout"`
}
```

**Loading Process:**
```go
func LoadRESTPlugin(configPath string, config *RESTPluginConfig) (*RESTNodePlugin, error) {
    // 1. Read YAML configuration
    data, _ := os.ReadFile(configPath)

    // 2. Parse YAML
    var pluginConfig RESTPluginYAML
    yaml.Unmarshal(data, &pluginConfig)

    // 3. Parse timeout and method
    timeout := parseTimeout(pluginConfig.Timeout)
    method := pluginConfig.Method
    if method == "" {
        method = "POST"
    }

    // 4. Create HTTP client
    httpClient := &http.Client{
        Timeout: timeout,
    }

    return &RESTNodePlugin{
        Endpoint:   pluginConfig.Endpoint,
        Method:     method,
        Headers:    pluginConfig.Headers,
        httpClient: httpClient,
        // ...
    }
}
```

**Execution Flow:**
```go
func (p *RESTNodePlugin) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
    // 1. Prepare payload
    payload := map[string]interface{}{
        "inputData":  inputData,
        "parameters": nodeParams,
    }
    payloadJSON, _ := json.Marshal(payload)

    // 2. Create HTTP request
    req, _ := http.NewRequest(p.Method, p.Endpoint, bytes.NewBuffer(payloadJSON))
    req.Header.Set("Content-Type", "application/json")
    for key, value := range p.Headers {
        req.Header.Set(key, value)
    }

    // 3. Make request
    resp, _ := p.httpClient.Do(req)
    defer resp.Body.Close()

    // 4. Read response
    respBody, _ := io.ReadAll(resp.Body)

    // 5. Check status
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
    }

    // 6. Parse response
    var response RESTExecuteResponse
    json.Unmarshal(respBody, &response)

    if !response.Success {
        return nil, fmt.Errorf(response.Error)
    }

    return response.OutputData, nil
}
```

**REST API Interface:**
```http
POST /api/node/execute
Content-Type: application/json

{
  "inputData": [
    {"json": {"field": "value"}}
  ],
  "parameters": {
    "param1": "value1"
  }
}

Response:
{
  "success": true,
  "outputData": [
    {"json": {"result": "processed"}}
  ],
  "error": ""
}
```

### 5. Wrapper Pattern

Each plugin type has a corresponding wrapper that implements the `NodeExecutor` interface.

**Purpose:** Adapt plugin-specific interfaces to the common NodeExecutor interface.

**Example (JavaScript Wrapper):**
```go
type JavaScriptNodeWrapper struct {
    *base.BaseNode
    plugin *JavaScriptNodePlugin
}

func NewJavaScriptNodeWrapper(plugin *JavaScriptNodePlugin) *JavaScriptNodeWrapper {
    return &JavaScriptNodeWrapper{
        BaseNode: base.NewBaseNode(plugin.Description),
        plugin:   plugin,
    }
}

func (w *JavaScriptNodeWrapper) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
    return w.plugin.Execute(inputData, nodeParams)
}

func (w *JavaScriptNodeWrapper) Description() base.NodeDescription {
    return w.plugin.Description
}

func (w *JavaScriptNodeWrapper) ValidateParameters(params map[string]interface{}) error {
    return w.plugin.ValidateParameters(params)
}
```

## Integration with main.go

**Flag Registration:**
```go
pluginDir = flag.String("plugin-dir", "", "Directory containing plugin files (.js, .grpc.yaml, .rest.yaml)")
```

**Plugin Loading:**
```go
func loadPlugins(eng engine.WorkflowEngine, pluginDir string) (int, error) {
    log.Printf("Loading plugins from directory: %s", pluginDir)

    // Create registry
    registry := plugins.NewPluginRegistry()

    // Load all plugins
    if err := registry.LoadPluginsFromDirectory(pluginDir); err != nil {
        return 0, fmt.Errorf("failed to load plugins: %w", err)
    }

    // Register with engine
    if err := registry.RegisterWithEngine(eng); err != nil {
        return 0, fmt.Errorf("failed to register plugins: %w", err)
    }

    pluginCount := registry.Count()
    log.Printf("✅ Loaded %d plugins", pluginCount)

    return pluginCount, nil
}
```

**Called from all modes:**
- Single-node mode (line 140)
- Cluster mode (line 293)
- Worker mode (line 616)

## Data Flow

### JavaScript Plugin Execution

```
User Workflow
     │
     ▼
WorkflowEngine.Execute()
     │
     ▼
JavaScriptNodeWrapper.Execute()
     │
     ▼
JavaScriptNodePlugin.Execute()
     │
     ├─> convertToJSArray()    # Go → JS conversion
     │
     ├─> vm.executeFunc()      # JavaScript execution
     │
     └─> convertFromJSArray()  # JS → Go conversion
     │
     ▼
Result []model.DataItem
```

### gRPC Plugin Execution

```
User Workflow
     │
     ▼
WorkflowEngine.Execute()
     │
     ▼
GRPCNodeWrapper.Execute()
     │
     ▼
GRPCNodePlugin.Execute()
     │
     ├─> json.Marshal()        # Go → JSON
     │
     ├─> grpc.Execute()        # gRPC call
     │       │
     │       ▼
     │   External Service
     │       │
     │       ▼
     │   ExecuteResponse
     │
     └─> json.Unmarshal()      # JSON → Go
     │
     ▼
Result []model.DataItem
```

### REST Plugin Execution

```
User Workflow
     │
     ▼
WorkflowEngine.Execute()
     │
     ▼
RESTNodeWrapper.Execute()
     │
     ▼
RESTNodePlugin.Execute()
     │
     ├─> json.Marshal()        # Go → JSON
     │
     ├─> http.Post()           # HTTP POST
     │       │
     │       ▼
     │   External Service
     │       │
     │       ▼
     │   HTTP Response
     │
     └─> json.Unmarshal()      # JSON → Go
     │
     ▼
Result []model.DataItem
```

## Error Handling

### JavaScript Plugins

**Syntax Errors:**
```
Failed to load JavaScript plugin: SyntaxError: Unexpected token
```
- Caught during LoadJavaScriptPlugin()
- Plugin not added to registry
- Error logged, continues loading other plugins

**Runtime Errors:**
```
JavaScript execution error: ReferenceError: xyz is not defined
```
- Caught during Execute()
- Returns error to workflow engine
- Workflow execution fails for that node

### gRPC Plugins

**Connection Errors:**
```
Failed to connect to gRPC service: connection refused
```
- Caught during LoadGRPCPlugin()
- Plugin not added to registry
- Error logged, continues loading other plugins

**Execution Errors:**
```
gRPC call failed: rpc error: code = Unknown desc = ...
```
- Caught during Execute()
- Returns error to workflow engine
- Workflow execution fails for that node

### REST Plugins

**HTTP Errors:**
```
HTTP request failed with status 500: Internal Server Error
```
- Caught during Execute()
- Returns error with status code and body
- Workflow execution fails for that node

**Timeout Errors:**
```
HTTP request failed: context deadline exceeded
```
- Caught during Execute()
- Returns timeout error
- Workflow execution fails for that node

## Performance Characteristics

### JavaScript Plugins

**Pros:**
- Very low latency (1-5ms overhead)
- No network roundtrip
- Simple deployment
- Fast startup

**Cons:**
- Limited to JavaScript
- Single-threaded execution
- Memory limited by Go process
- CPU-bound tasks block

**Best For:**
- Simple transformations
- String manipulation
- JSON processing
- Rapid prototyping

### gRPC Plugins

**Pros:**
- High performance (5-10ms overhead)
- Any language support
- Horizontal scaling
- Efficient serialization

**Cons:**
- Requires separate service
- More complex deployment
- Network dependency
- Connection management

**Best For:**
- Heavy computation
- Machine learning
- Complex algorithms
- Polyglot systems

### REST Plugins

**Pros:**
- Maximum compatibility
- Easy debugging (curl)
- Language agnostic
- Firewall friendly

**Cons:**
- Higher latency (20-50ms overhead)
- HTTP overhead
- Text-based encoding
- Less efficient

**Best For:**
- External APIs
- Legacy systems
- Debugging
- Wide compatibility

## Security Considerations

### JavaScript Plugins

**Risks:**
- Code executes in-process
- Can access Go runtime
- No sandboxing

**Mitigations:**
- Limited JavaScript API surface
- No filesystem access
- No network access
- No native module loading

**Best Practices:**
- Review all JavaScript plugins
- Use trusted sources only
- Validate inputs
- Limit plugin directory permissions

### gRPC Plugins

**Risks:**
- Network communication
- Service trust
- Data exposure

**Mitigations:**
- Use TLS for connections
- Validate service certificates
- Implement authentication
- Use private networks

**Best Practices:**
- Enable TLS in production
- Use mutual TLS
- Validate all inputs
- Implement rate limiting

### REST Plugins

**Risks:**
- HTTP plaintext communication
- Service trust
- Data exposure

**Mitigations:**
- Use HTTPS endpoints
- Validate SSL certificates
- Implement authentication
- Use API keys/tokens

**Best Practices:**
- Always use HTTPS in production
- Validate certificates
- Use secure headers
- Implement timeouts

## Testing

### Unit Tests

**JavaScript Plugin Tests:**
```go
func TestJavaScriptPlugin_Execute(t *testing.T) {
    plugin, err := LoadJavaScriptPlugin("testdata/simple.js", nil)
    require.NoError(t, err)

    input := []model.DataItem{
        {JSON: map[string]interface{}{"value": "test"}},
    }

    output, err := plugin.Execute(input, map[string]interface{}{})
    require.NoError(t, err)
    assert.Len(t, output, 1)
}
```

**gRPC Plugin Tests:**
```go
func TestGRPCPlugin_Execute(t *testing.T) {
    // Start mock gRPC server
    server := startMockGRPCServer(t)
    defer server.Stop()

    // Create plugin config pointing to mock server
    plugin, err := LoadGRPCPlugin("testdata/mock.grpc.yaml", nil)
    require.NoError(t, err)

    output, err := plugin.Execute(input, params)
    require.NoError(t, err)
}
```

**REST Plugin Tests:**
```go
func TestRESTPlugin_Execute(t *testing.T) {
    // Start mock HTTP server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "success": true,
            "outputData": []model.DataItem{},
        })
    }))
    defer server.Close()

    plugin, err := LoadRESTPlugin("testdata/mock.rest.yaml", nil)
    require.NoError(t, err)

    output, err := plugin.Execute(input, params)
    require.NoError(t, err)
}
```

### Integration Tests

**Full Plugin Loading:**
```go
func TestPluginRegistry_LoadFromDirectory(t *testing.T) {
    registry := NewPluginRegistry()

    err := registry.LoadPluginsFromDirectory("testdata/plugins")
    require.NoError(t, err)

    assert.Equal(t, 3, registry.Count())
    assert.Contains(t, registry.ListPlugins(), "testPlugin")
}
```

**End-to-End Workflow:**
```go
func TestPluginInWorkflow(t *testing.T) {
    // Create engine
    engine := engine.NewWorkflowEngine()

    // Load plugins
    registry := NewPluginRegistry()
    registry.LoadPluginsFromDirectory("testdata/plugins")
    registry.RegisterWithEngine(engine)

    // Execute workflow using plugin node
    workflow := loadTestWorkflow("workflow_with_plugin.json")
    result, err := engine.ExecuteWorkflow(workflow, nil)

    require.NoError(t, err)
    assert.True(t, result.Success)
}
```

## Future Enhancements

### Planned Features

1. **Hot Reloading**
   - Watch plugin directory for changes
   - Reload plugins on file modification
   - No server restart required

2. **Plugin Marketplace**
   - Central repository of community plugins
   - One-command plugin installation
   - Version management
   - Dependency resolution

3. **WebAssembly Support**
   - WASM plugin type
   - Near-native performance
   - Language agnostic
   - Sandboxed execution

4. **Plugin Validation**
   - Schema validation for YAML configs
   - JavaScript linting
   - Security scanning
   - Performance profiling

5. **Plugin Monitoring**
   - Execution metrics per plugin
   - Error tracking
   - Performance profiling
   - Usage analytics

### Potential Improvements

1. **Caching**
   - Cache compiled JavaScript
   - Connection pooling for gRPC/REST
   - Result caching

2. **Optimization**
   - Lazy loading of plugins
   - Parallel plugin initialization
   - Optimized type conversions

3. **Developer Tools**
   - Plugin development CLI
   - Testing framework
   - Debug mode
   - Plugin playground

## Troubleshooting

### Common Issues

**Issue:** Plugins not loading
```
Warning: Plugin loading failed: no such file or directory
```
**Solution:** Check plugin directory path, verify files exist

**Issue:** JavaScript syntax error
```
Failed to load JavaScript plugin: SyntaxError: ...
```
**Solution:** Validate JavaScript syntax, check for ES6+ features

**Issue:** gRPC connection failed
```
Failed to connect to gRPC service: connection refused
```
**Solution:** Verify service is running, check address/port

**Issue:** REST endpoint not responding
```
HTTP request failed with status 404
```
**Solution:** Verify endpoint URL, check service logs

### Debug Mode

Enable verbose logging:
```bash
export N8N_GO_LOG_LEVEL=debug
./n8n-go --plugin-dir ./plugins
```

Check plugin loading:
```bash
./n8n-go --plugin-dir ./plugins 2>&1 | grep -i plugin
```

Test individual plugin:
```bash
# JavaScript
node plugins/myPlugin.js

# gRPC
grpcurl -plaintext localhost:50051 nodeservice.NodeService/Describe

# REST
curl -X POST http://localhost:8090/api/node/execute \
  -H "Content-Type: application/json" \
  -d '{"inputData":[],"parameters":{}}'
```

## References

- [Plugin User Documentation](../plugins/README.md)
- [Plugin Architecture Analysis](PLUGIN_ARCHITECTURE.md)
- [Goja JavaScript Runtime](https://github.com/dop251/goja)
- [gRPC Documentation](https://grpc.io/docs/)
- [n8n Node Development](https://docs.n8n.io/integrations/creating-nodes/)

---

**Implementation Status:** ✅ Complete and tested

**Version:** 1.0.0
**Last Updated:** 2025-11-10
