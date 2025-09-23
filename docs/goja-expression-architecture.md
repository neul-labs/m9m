# Goja-based Expression System Architecture

## Overview

This document details the technical architecture for implementing a fully n8n-compatible expression system in n8n-go using the Goja JavaScript engine. The system provides 100% compatibility with n8n expressions while leveraging Go's performance advantages.

## Core Architecture

### System Layers

```
┌─────────────────────────────────────────────────────────────┐
│                     Application Layer                       │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────┐  │
│  │  Node Execution │  │   Workflow      │  │   CLI/API   │  │
│  │     Engine      │  │   Processor     │  │  Interface  │  │
│  └─────────────────┘  └─────────────────┘  └─────────────┘  │
└─────────────────────────────────────────────────────────────┘
                               │
┌─────────────────────────────────────────────────────────────┐
│                   Expression Layer                          │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────┐  │
│  │   Expression    │  │   Parameter     │  │  Context    │  │
│  │   Evaluator     │  │   Resolver      │  │  Manager    │  │
│  └─────────────────┘  └─────────────────┘  └─────────────┘  │
└─────────────────────────────────────────────────────────────┘
                               │
┌─────────────────────────────────────────────────────────────┐
│                    Goja Runtime Layer                       │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────┐  │
│  │  Secure Goja    │  │   Built-in      │  │ Data Proxy  │  │
│  │   Runtime       │  │   Functions     │  │   System    │  │
│  └─────────────────┘  └─────────────────┘  └─────────────┘  │
└─────────────────────────────────────────────────────────────┘
                               │
┌─────────────────────────────────────────────────────────────┐
│                     Parser Layer                            │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────┐  │
│  │   Expression    │  │     AST         │  │   Cache     │  │
│  │    Parser       │  │   Builder       │  │   Manager   │  │
│  └─────────────────┘  └─────────────────┘  └─────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Component Details

### 1. Expression Parser

#### Purpose
Converts n8n expression strings into executable JavaScript code while maintaining 100% syntax compatibility.

#### Key Components

```go
type ExpressionParser struct {
    openBracketRegex  *regexp.Regexp
    closeBracketRegex *regexp.Regexp
    cache            map[string]*ParsedExpression
    cacheMutex       sync.RWMutex
}

type ParsedExpression struct {
    OriginalText    string
    JavaScriptCode  string
    Chunks          []ExpressionChunk
    Variables       []string
    HasExpressions  bool
    CacheTime       time.Time
}

type ExpressionChunk struct {
    Type    ChunkType  // Text, Expression, or Mixed
    Content string     // Raw content
    Start   int        // Position in original
    End     int        // End position
}
```

#### Parsing Algorithm

```go
func (p *ExpressionParser) ParseExpression(input string) (*ParsedExpression, error) {
    // 1. Check cache first
    if cached := p.getFromCache(input); cached != nil {
        return cached, nil
    }

    // 2. Detect expression type
    isExpression := strings.HasPrefix(input, "=")
    if isExpression {
        input = strings.TrimPrefix(input, "=")
    }

    // 3. Split into chunks
    chunks, err := p.splitIntoChunks(input)
    if err != nil {
        return nil, err
    }

    // 4. Build JavaScript code
    jsCode := p.buildJavaScriptCode(chunks, isExpression)

    // 5. Extract variable references
    variables := p.extractVariables(chunks)

    parsed := &ParsedExpression{
        OriginalText:   input,
        JavaScriptCode: jsCode,
        Chunks:         chunks,
        Variables:      variables,
        HasExpressions: len(chunks) > 1 || chunks[0].Type == ExpressionChunkType,
        CacheTime:      time.Now(),
    }

    // 6. Cache result
    p.cacheResult(input, parsed)

    return parsed, nil
}
```

### 2. Secure Goja Runtime

#### Purpose
Provides a sandboxed JavaScript execution environment that matches n8n's security model exactly.

#### Security Implementation

```go
type SecureGojaRuntime struct {
    vm               *goja.Runtime
    allowedGlobals   map[string]interface{}
    blockedGlobals   map[string]bool
    functionRegistry *FunctionRegistry
    timeout          time.Duration
    memoryLimit      int64
    executionCount   int64
    isInitialized    bool
    mutex           sync.RWMutex
}

func NewSecureGojaRuntime() *SecureGojaRuntime {
    runtime := &SecureGojaRuntime{
        vm:             goja.New(),
        allowedGlobals: make(map[string]interface{}),
        blockedGlobals: make(map[string]bool),
        timeout:        30 * time.Second,
        memoryLimit:    50 * 1024 * 1024, // 50MB
    }

    runtime.initializeSecurity()
    runtime.registerAllowedGlobals()
    runtime.blockDangerousGlobals()

    return runtime
}
```

#### Global Object Management

```go
// Exactly matching n8n's allowed globals
func (r *SecureGojaRuntime) registerAllowedGlobals() {
    allowedGlobals := map[string]interface{}{
        // JavaScript built-ins
        "Array":               goja.Undefined(),
        "Object":              goja.Undefined(),
        "String":              goja.Undefined(),
        "Number":              goja.Undefined(),
        "Boolean":             goja.Undefined(),
        "Date":                goja.Undefined(),
        "Math":                goja.Undefined(),
        "JSON":                goja.Undefined(),
        "RegExp":              goja.Undefined(),
        "Error":               goja.Undefined(),

        // Global functions
        "parseInt":            goja.Undefined(),
        "parseFloat":          goja.Undefined(),
        "isNaN":               goja.Undefined(),
        "isFinite":            goja.Undefined(),
        "encodeURIComponent":  goja.Undefined(),
        "decodeURIComponent":  goja.Undefined(),
        "encodeURI":           goja.Undefined(),
        "decodeURI":           goja.Undefined(),

        // Typed arrays
        "Int8Array":           goja.Undefined(),
        "Uint8Array":          goja.Undefined(),
        "Uint8ClampedArray":   goja.Undefined(),
        "Int16Array":          goja.Undefined(),
        "Uint16Array":         goja.Undefined(),
        "Int32Array":          goja.Undefined(),
        "Uint32Array":         goja.Undefined(),
        "Float32Array":        goja.Undefined(),
        "Float64Array":        goja.Undefined(),
        "ArrayBuffer":         goja.Undefined(),
        "DataView":            goja.Undefined(),

        // Collections
        "Map":                 goja.Undefined(),
        "Set":                 goja.Undefined(),
        "WeakMap":             goja.Undefined(),
        "WeakSet":             goja.Undefined(),

        // Internationalization
        "Intl":                goja.Undefined(),

        // Luxon DateTime (Go implementation)
        "DateTime":            r.createDateTimeObject(),
        "Duration":            r.createDurationObject(),
        "Interval":            r.createIntervalObject(),
    }

    for name, value := range allowedGlobals {
        r.vm.Set(name, value)
        r.allowedGlobals[name] = value
    }
}

// Exactly matching n8n's blocked globals
func (r *SecureGojaRuntime) blockDangerousGlobals() {
    blockedGlobals := []string{
        // Code execution
        "eval", "Function", "GeneratorFunction", "AsyncFunction",

        // Timers
        "setTimeout", "setInterval", "clearTimeout", "clearInterval",
        "setImmediate", "clearImmediate",

        // Network
        "fetch", "XMLHttpRequest", "WebSocket", "EventSource",

        // Async
        "Promise", "async", "await",

        // DOM/Browser
        "document", "window", "globalThis", "self", "top", "parent",
        "frames", "opener", "closed", "length", "location", "navigator",
        "history", "screen", "alert", "confirm", "prompt",

        // Node.js
        "global", "process", "Buffer", "require", "module", "exports",
        "__dirname", "__filename", "console",

        // Reflection
        "Reflect", "Proxy",

        // WebAssembly
        "WebAssembly",

        // Worker APIs
        "Worker", "SharedWorker", "ServiceWorker",

        // Storage
        "localStorage", "sessionStorage", "indexedDB",

        // Performance
        "performance", "PerformanceObserver",
    }

    for _, name := range blockedGlobals {
        r.blockedGlobals[name] = true
        r.vm.Set(name, goja.Undefined())
    }
}
```

### 3. Built-in Function Library

#### Function Organization

```go
type FunctionRegistry struct {
    extensions map[string]ExtensionProvider
    vm         *goja.Runtime
}

type ExtensionProvider interface {
    GetCategory() string
    GetFunctionNames() []string
    RegisterFunctions(vm *goja.Runtime) error
    GetFunctionHelp(name string) *FunctionHelp
}

type FunctionHelp struct {
    Name        string
    Description string
    Parameters  []ParameterInfo
    ReturnType  string
    Examples    []string
}
```

#### String Extensions

```go
type StringExtensions struct{}

func (s *StringExtensions) RegisterFunctions(vm *goja.Runtime) error {
    functions := map[string]func(call goja.FunctionCall, vm *goja.Runtime) goja.Value{
        "split":           s.split,
        "join":            s.join,
        "substring":       s.substring,
        "substr":          s.substr,
        "replace":         s.replace,
        "replaceAll":      s.replaceAll,
        "trim":            s.trim,
        "trimStart":       s.trimStart,
        "trimEnd":         s.trimEnd,
        "toLowerCase":     s.toLowerCase,
        "toUpperCase":     s.toUpperCase,
        "charAt":          s.charAt,
        "charCodeAt":      s.charCodeAt,
        "indexOf":         s.indexOf,
        "lastIndexOf":     s.lastIndexOf,
        "includes":        s.includes,
        "startsWith":      s.startsWith,
        "endsWith":        s.endsWith,
        "padStart":        s.padStart,
        "padEnd":          s.padEnd,
        "repeat":          s.repeat,
        "slice":           s.slice,

        // n8n specific extensions
        "base64Encode":    s.base64Encode,
        "base64Decode":    s.base64Decode,
        "urlEncode":       s.urlEncode,
        "urlDecode":       s.urlDecode,
        "htmlEncode":      s.htmlEncode,
        "htmlDecode":      s.htmlDecode,
        "md5":             s.md5,
        "sha1":            s.sha1,
        "sha256":          s.sha256,
        "sha512":          s.sha512,
        "isEmail":         s.isEmail,
        "isUrl":           s.isUrl,
        "isDomain":        s.isDomain,
        "stripTags":       s.stripTags,
        "extractDomain":   s.extractDomain,
        "extractUrl":      s.extractUrl,
        "toTitleCase":     s.toTitleCase,
        "toCamelCase":     s.toCamelCase,
        "toSnakeCase":     s.toSnakeCase,
        "toKebabCase":     s.toKebabCase,
    }

    for name, fn := range functions {
        vm.Set(name, fn)
    }

    return nil
}

func (s *StringExtensions) split(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
    if len(call.Arguments) < 2 {
        panic(vm.NewTypeError("split requires 2 arguments"))
    }

    str := call.Arguments[0].String()
    separator := call.Arguments[1].String()

    var limit int = -1
    if len(call.Arguments) > 2 {
        limit = int(call.Arguments[2].ToInteger())
    }

    var parts []string
    if separator == "" {
        // Split each character
        parts = strings.Split(str, "")
    } else {
        parts = strings.Split(str, separator)
    }

    if limit > 0 && len(parts) > limit {
        parts = parts[:limit]
    }

    return vm.ToValue(parts)
}
```

### 4. Workflow Data Proxy

#### Proxy Architecture

```go
type WorkflowDataProxy struct {
    // Context
    workflow            *Workflow
    runExecutionData    *RunExecutionData
    runIndex            int
    itemIndex           int
    activeNodeName      string
    connectionInputData []NodeExecutionData
    siblingParameters   map[string]interface{}
    mode                WorkflowExecuteMode

    // Additional context
    additionalKeys      *AdditionalKeys
    executeData         *ExecuteData

    // Caching
    dataCache           map[string]interface{}
    cacheMutex          sync.RWMutex

    // JavaScript VM reference
    vm                  *goja.Runtime
}

func (p *WorkflowDataProxy) CreateJavaScriptProxy() goja.Value {
    proxy := p.vm.NewObject()

    // Core n8n variables
    proxy.Set("$json", p.createJsonProxy())
    proxy.Set("$input", p.createInputProxy())
    proxy.Set("$node", p.createNodeProxy())
    proxy.Set("$parameter", p.createParameterProxy())
    proxy.Set("$workflow", p.createWorkflowProxy())
    proxy.Set("$execution", p.createExecutionProxy())
    proxy.Set("$env", p.createEnvProxy())
    proxy.Set("$binary", p.createBinaryProxy())
    proxy.Set("$vars", p.createVarsProxy())

    // Legacy support
    proxy.Set("$evaluateExpression", p.createEvaluateExpressionProxy())

    return proxy
}
```

#### Variable Context Implementation

```go
func (p *WorkflowDataProxy) createInputProxy() goja.Value {
    inputProxy := p.vm.NewObject()

    // $input.all() - all input items
    inputProxy.Set("all", p.vm.ToValue(func(call goja.FunctionCall) goja.Value {
        var connectionIndex int = 0
        if len(call.Arguments) > 0 {
            connectionIndex = int(call.Arguments[0].ToInteger())
        }

        inputData := p.getInputConnectionData(connectionIndex)
        return p.vm.ToValue(inputData)
    }))

    // $input.first() - first item
    inputProxy.Set("first", p.vm.ToValue(func(call goja.FunctionCall) goja.Value {
        var connectionIndex int = 0
        if len(call.Arguments) > 0 {
            connectionIndex = int(call.Arguments[0].ToInteger())
        }

        inputData := p.getInputConnectionData(connectionIndex)
        if len(inputData) > 0 {
            return p.vm.ToValue(inputData[0].JSON)
        }
        return goja.Undefined()
    }))

    // $input.last() - last item
    inputProxy.Set("last", p.vm.ToValue(func(call goja.FunctionCall) goja.Value {
        var connectionIndex int = 0
        if len(call.Arguments) > 0 {
            connectionIndex = int(call.Arguments[0].ToInteger())
        }

        inputData := p.getInputConnectionData(connectionIndex)
        if len(inputData) > 0 {
            return p.vm.ToValue(inputData[len(inputData)-1].JSON)
        }
        return goja.Undefined()
    }))

    // $input.item - current item
    inputProxy.Set("item", p.vm.ToValue(func(call goja.FunctionCall) goja.Value {
        var itemIndex int = p.itemIndex
        var connectionIndex int = 0

        if len(call.Arguments) > 0 {
            itemIndex = int(call.Arguments[0].ToInteger())
        }
        if len(call.Arguments) > 1 {
            connectionIndex = int(call.Arguments[1].ToInteger())
        }

        inputData := p.getInputConnectionData(connectionIndex)
        if itemIndex >= 0 && itemIndex < len(inputData) {
            return p.vm.ToValue(inputData[itemIndex].JSON)
        }
        return goja.Undefined()
    }))

    return inputProxy
}

func (p *WorkflowDataProxy) createNodeProxy() goja.Value {
    // $node("NodeName") or $("NodeName") function
    return p.vm.ToValue(func(call goja.FunctionCall) goja.Value {
        if len(call.Arguments) == 0 {
            panic(p.vm.NewTypeError("Node name is required"))
        }

        nodeName := call.Arguments[0].String()
        nodeData := p.getNodeExecutionData(nodeName)

        nodeProxy := p.vm.NewObject()

        // Basic access
        nodeProxy.Set("json", p.vm.ToValue(nodeData.JSON))
        nodeProxy.Set("binary", p.vm.ToValue(nodeData.Binary))

        // Array-like access methods
        nodeProxy.Set("first", p.vm.ToValue(func(call goja.FunctionCall) goja.Value {
            allData := p.getAllNodeExecutionData(nodeName)
            if len(allData) > 0 {
                return p.vm.ToValue(allData[0].JSON)
            }
            return goja.Undefined()
        }))

        nodeProxy.Set("last", p.vm.ToValue(func(call goja.FunctionCall) goja.Value {
            allData := p.getAllNodeExecutionData(nodeName)
            if len(allData) > 0 {
                return p.vm.ToValue(allData[len(allData)-1].JSON)
            }
            return goja.Undefined()
        }))

        nodeProxy.Set("all", p.vm.ToValue(func(call goja.FunctionCall) goja.Value {
            allData := p.getAllNodeExecutionData(nodeName)
            jsonData := make([]interface{}, len(allData))
            for i, item := range allData {
                jsonData[i] = item.JSON
            }
            return p.vm.ToValue(jsonData)
        }))

        nodeProxy.Set("item", p.vm.ToValue(func(call goja.FunctionCall) goja.Value {
            var itemIndex int = 0
            if len(call.Arguments) > 0 {
                itemIndex = int(call.Arguments[0].ToInteger())
            }

            allData := p.getAllNodeExecutionData(nodeName)
            if itemIndex >= 0 && itemIndex < len(allData) {
                return p.vm.ToValue(allData[itemIndex].JSON)
            }
            return goja.Undefined()
        }))

        // pairedItem support for data lineage
        nodeProxy.Set("pairedItem", p.vm.ToValue(func(call goja.FunctionCall) goja.Value {
            var itemIndex int = p.itemIndex
            if len(call.Arguments) > 0 {
                itemIndex = int(call.Arguments[0].ToInteger())
            }

            return p.getPairedItemData(nodeName, itemIndex)
        }))

        return nodeProxy
    })
}
```

### 5. Expression Evaluator

#### Main Evaluation Engine

```go
type ExpressionEvaluator struct {
    runtime          *SecureGojaRuntime
    parser           *ExpressionParser
    functionRegistry *FunctionRegistry
    cache            *ExpressionCache
    config           *EvaluatorConfig
}

type EvaluatorConfig struct {
    EnableCaching    bool
    CacheSize        int
    ExecutionTimeout time.Duration
    MemoryLimit      int64
    EnableProfiling  bool
}

func (e *ExpressionEvaluator) EvaluateExpression(
    expression string,
    context *ExpressionContext,
) (interface{}, error) {
    // 1. Parse expression
    parsed, err := e.parser.ParseExpression(expression)
    if err != nil {
        return nil, &ExpressionError{
            Type:        "ParseError",
            Message:     err.Error(),
            Expression:  expression,
            Context:     context,
        }
    }

    // 2. Check cache
    if e.config.EnableCaching {
        if cached := e.cache.Get(parsed.JavaScriptCode, context.GetCacheKey()); cached != nil {
            return cached.Value, nil
        }
    }

    // 3. Create data proxy
    proxy := &WorkflowDataProxy{
        workflow:            context.Workflow,
        runExecutionData:    context.RunExecutionData,
        runIndex:           context.RunIndex,
        itemIndex:          context.ItemIndex,
        activeNodeName:     context.ActiveNodeName,
        connectionInputData: context.ConnectionInputData,
        mode:               context.Mode,
        vm:                 e.runtime.vm,
    }

    // 4. Set execution context
    e.runtime.vm.Set("this", proxy.CreateJavaScriptProxy())

    // 5. Execute with timeout
    resultChan := make(chan interface{}, 1)
    errorChan := make(chan error, 1)

    go func() {
        defer func() {
            if r := recover(); r != nil {
                errorChan <- fmt.Errorf("expression panic: %v", r)
            }
        }()

        result, err := e.executeJavaScript(parsed.JavaScriptCode)
        if err != nil {
            errorChan <- err
            return
        }
        resultChan <- result
    }()

    select {
    case result := <-resultChan:
        // Cache successful result
        if e.config.EnableCaching {
            e.cache.Set(parsed.JavaScriptCode, context.GetCacheKey(), result)
        }
        return result, nil

    case err := <-errorChan:
        return nil, &ExpressionError{
            Type:        "ExecutionError",
            Message:     err.Error(),
            Expression:  expression,
            JavaScript:  parsed.JavaScriptCode,
            Context:     context,
        }

    case <-time.After(e.config.ExecutionTimeout):
        return nil, &ExpressionError{
            Type:        "TimeoutError",
            Message:     "Expression execution timed out",
            Expression:  expression,
            Context:     context,
        }
    }
}
```

## Performance Optimizations

### 1. Compilation Caching

```go
type ExpressionCache struct {
    compiledPrograms map[string]*goja.Program
    executionResults map[string]*CachedResult
    mutex           sync.RWMutex
    maxSize         int
    stats           *CacheStats
}

type CachedResult struct {
    Value     interface{}
    Timestamp time.Time
    HitCount  int64
}

func (c *ExpressionCache) GetCompiledProgram(jsCode string) *goja.Program {
    c.mutex.RLock()
    defer c.mutex.RUnlock()
    return c.compiledPrograms[jsCode]
}

func (c *ExpressionCache) SetCompiledProgram(jsCode string, program *goja.Program) {
    c.mutex.Lock()
    defer c.mutex.Unlock()

    if len(c.compiledPrograms) >= c.maxSize {
        c.evictOldest()
    }

    c.compiledPrograms[jsCode] = program
}
```

### 2. Context Reuse

```go
type RuntimePool struct {
    runtimes chan *SecureGojaRuntime
    factory  func() *SecureGojaRuntime
    maxSize  int
}

func (p *RuntimePool) Get() *SecureGojaRuntime {
    select {
    case runtime := <-p.runtimes:
        return runtime
    default:
        return p.factory()
    }
}

func (p *RuntimePool) Put(runtime *SecureGojaRuntime) {
    // Reset runtime state
    runtime.Reset()

    select {
    case p.runtimes <- runtime:
    default:
        // Pool full, discard runtime
    }
}
```

### 3. Lazy Data Loading

```go
func (p *WorkflowDataProxy) getNodeExecutionData(nodeName string) *NodeExecutionData {
    p.cacheMutex.RLock()
    if cached, exists := p.dataCache[nodeName]; exists {
        p.cacheMutex.RUnlock()
        return cached.(*NodeExecutionData)
    }
    p.cacheMutex.RUnlock()

    // Load data on demand
    data := p.loadNodeExecutionData(nodeName)

    p.cacheMutex.Lock()
    p.dataCache[nodeName] = data
    p.cacheMutex.Unlock()

    return data
}
```

## Error Handling

### Error Types and Context

```go
type ExpressionError struct {
    Type        string                 `json:"type"`
    Message     string                 `json:"message"`
    Expression  string                 `json:"expression"`
    JavaScript  string                 `json:"javascript,omitempty"`
    Context     *ExpressionContext     `json:"context"`
    Stack       string                 `json:"stack,omitempty"`
    Timestamp   time.Time              `json:"timestamp"`
}

func (e *ExpressionError) Error() string {
    return fmt.Sprintf("Expression error in node '%s': %s",
        e.Context.ActiveNodeName, e.Message)
}

func (e *ExpressionError) GetUserFriendlyMessage() string {
    switch e.Type {
    case "ParseError":
        return fmt.Sprintf("Invalid expression syntax: %s", e.Message)
    case "ReferenceError":
        return fmt.Sprintf("Variable or function not found: %s", e.Message)
    case "TypeError":
        return fmt.Sprintf("Type error: %s", e.Message)
    case "TimeoutError":
        return "Expression took too long to execute"
    default:
        return e.Message
    }
}
```

This architecture provides a robust, secure, and highly compatible expression system that matches n8n's behavior while leveraging Go's performance advantages.