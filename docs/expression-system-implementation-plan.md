# Goja-based Expression System Implementation Plan

## Executive Summary

This document outlines the implementation of a fully n8n-compatible expression system for n8n-go using the Goja JavaScript engine. The goal is to achieve 100% compatibility with n8n's expression syntax while maintaining Go's performance advantages.

## Architecture Overview

### Core Components

```
┌─────────────────────────────────────────────────────────────┐
│                    Expression System                        │
├─────────────────────────────────────────────────────────────┤
│ ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐ │
│ │  Expression     │ │   Goja Runtime  │ │  Built-in       │ │
│ │  Parser         │ │   & Sandbox     │ │  Functions      │ │
│ │                 │ │                 │ │                 │ │
│ │ • Tokenization  │ │ • JS Execution  │ │ • String        │ │
│ │ • {{ }} Parsing │ │ • Security      │ │ • Math          │ │
│ │ • AST Building  │ │ • Error Handle  │ │ • Date/Time     │ │
│ └─────────────────┘ └─────────────────┘ │ • Array/Object  │ │
│                                         │ • Logic/Utils   │ │
│ ┌─────────────────┐ ┌─────────────────┐ └─────────────────┘ │
│ │ Workflow Data   │ │  Expression     │                     │
│ │ Proxy           │ │  Cache          │                     │
│ │                 │ │                 │                     │
│ │ • $json/$input  │ │ • Parsed AST    │                     │
│ │ • $node/$exec   │ │ • Compiled JS   │                     │
│ │ • Cross-node    │ │ • Performance   │                     │
│ └─────────────────┘ └─────────────────┘                     │
└─────────────────────────────────────────────────────────────┘
```

## Implementation Phases

### Phase 1: Expression Parser (Week 1)

#### 1.1 Expression Tokenizer
```go
type ExpressionChunk struct {
    Type    ChunkType  // Text or Expression
    Content string     // Raw content
    Start   int        // Position in original string
    End     int        // End position
}

type ExpressionParser struct {
    openBracket  *regexp.Regexp  // /(?<escape>\\|)(?<brackets>\{\{)/
    closeBracket *regexp.Regexp  // /(?<escape>\\|)(?<brackets>\}\})/
}

func (p *ExpressionParser) Parse(input string) ([]ExpressionChunk, error)
```

#### 1.2 n8n Compatibility Markers
- Detect `={{ expression }}` vs `{{ expression }}`
- Handle escape sequences `\{\{` and `\}\}`
- Support mixed text and expressions
- Preserve original n8n behavior exactly

#### 1.3 AST Processing
```go
type ExpressionAST struct {
    Type       string      // "expression", "text", "mixed"
    JavaScript string      // Compiled JavaScript code
    Chunks     []ExpressionChunk
    Variables  []string    // Extracted variable references
}

func ParseExpression(input string) (*ExpressionAST, error)
```

### Phase 2: Secure Goja Runtime (Week 1)

#### 2.1 Sandboxed Runtime
```go
type SecureGojaRuntime struct {
    vm            *goja.Runtime
    allowedGlobals map[string]interface{}
    blockedGlobals []string
    timeout       time.Duration
}

// Blocked globals (exactly matching n8n)
var blockedGlobals = []string{
    "eval", "Function", "setTimeout", "setInterval", "clearTimeout",
    "fetch", "XMLHttpRequest", "WebSocket", "Promise", "async",
    "document", "window", "globalThis", "global", "process",
    "require", "module", "exports", "__dirname", "__filename",
    "Reflect", "Proxy", "WebAssembly", "Generator", "AsyncFunction",
}

// Allowed globals (exactly matching n8n)
var allowedGlobals = map[string]interface{}{
    "Array": goja.Undefined(),
    "Object": goja.Undefined(),
    "String": goja.Undefined(),
    "Number": goja.Undefined(),
    "Boolean": goja.Undefined(),
    "Date": goja.Undefined(),
    "Math": goja.Undefined(),
    "JSON": goja.Undefined(),
    "parseInt": goja.Undefined(),
    "parseFloat": goja.Undefined(),
    "isNaN": goja.Undefined(),
    "isFinite": goja.Undefined(),
    "encodeURIComponent": goja.Undefined(),
    "decodeURIComponent": goja.Undefined(),
    // Type arrays
    "Int8Array": goja.Undefined(),
    "Uint8Array": goja.Undefined(),
    "Int16Array": goja.Undefined(),
    "Uint16Array": goja.Undefined(),
    "Int32Array": goja.Undefined(),
    "Uint32Array": goja.Undefined(),
    "Float32Array": goja.Undefined(),
    "Float64Array": goja.Undefined(),
    // Collections
    "Map": goja.Undefined(),
    "Set": goja.Undefined(),
    "WeakMap": goja.Undefined(),
    "WeakSet": goja.Undefined(),
    // Luxon DateTime (implemented in Go)
    "DateTime": goja.Undefined(),
    "Duration": goja.Undefined(),
    "Interval": goja.Undefined(),
}
```

#### 2.2 Prototype Pollution Protection
```go
func (r *SecureGojaRuntime) sanitizeObject(obj *goja.Object) {
    // Block access to dangerous properties
    unsafeProps := []string{"__proto__", "prototype", "constructor", "getPrototypeOf"}
    for _, prop := range unsafeProps {
        obj.Delete(prop)
    }
}
```

### Phase 3: Built-in Function Library (Week 2)

#### 3.1 Function Categories

##### String Functions (40+ functions)
```go
type StringExtensions struct{}

func (s *StringExtensions) RegisterAll(vm *goja.Runtime) {
    // String manipulation
    vm.Set("split", s.split)
    vm.Set("join", s.join)
    vm.Set("substring", s.substring)
    vm.Set("replace", s.replace)
    vm.Set("trim", s.trim)
    vm.Set("toLowerCase", s.toLowerCase)
    vm.Set("toUpperCase", s.toUpperCase)

    // Encoding/Decoding
    vm.Set("base64Encode", s.base64Encode)
    vm.Set("base64Decode", s.base64Decode)
    vm.Set("urlEncode", s.urlEncode)
    vm.Set("urlDecode", s.urlDecode)

    // Hashing
    vm.Set("md5", s.md5)
    vm.Set("sha1", s.sha1)
    vm.Set("sha256", s.sha256)

    // Validation
    vm.Set("isEmail", s.isEmail)
    vm.Set("isUrl", s.isUrl)
    vm.Set("isDomain", s.isDomain)
}

func (s *StringExtensions) split(call goja.FunctionCall, vm *goja.Runtime) goja.Value {
    // Implementation with n8n compatibility
}
```

##### Math Functions (30+ functions)
```go
type MathExtensions struct{}

func (m *MathExtensions) RegisterAll(vm *goja.Runtime) {
    vm.Set("min", m.min)
    vm.Set("max", m.max)
    vm.Set("average", m.average)
    vm.Set("sum", m.sum)
    vm.Set("round", m.round)
    vm.Set("ceil", m.ceil)
    vm.Set("floor", m.floor)
    vm.Set("abs", m.abs)
    vm.Set("random", m.random)
    vm.Set("randomInt", m.randomInt)
}
```

##### Array Functions (25+ functions)
```go
type ArrayExtensions struct{}

func (a *ArrayExtensions) RegisterAll(vm *goja.Runtime) {
    vm.Set("first", a.first)
    vm.Set("last", a.last)
    vm.Set("unique", a.unique)
    vm.Set("compact", a.compact)
    vm.Set("flatten", a.flatten)
    vm.Set("chunk", a.chunk)
    vm.Set("pluck", a.pluck)
    vm.Set("randomItem", a.randomItem)
}
```

##### Date/Time Functions (20+ functions using time package)
```go
type DateExtensions struct{}

func (d *DateExtensions) RegisterAll(vm *goja.Runtime) {
    vm.Set("now", d.now)
    vm.Set("formatDate", d.formatDate)
    vm.Set("toDate", d.toDate)
    vm.Set("addDays", d.addDays)
    vm.Set("subtractDays", d.subtractDays)
    vm.Set("diffDays", d.diffDays)
    vm.Set("getTime", d.getTime)
    vm.Set("dateFormat", d.dateFormat)
}
```

#### 3.2 Function Registration System
```go
type FunctionRegistry struct {
    extensions []ExtensionProvider
}

type ExtensionProvider interface {
    RegisterAll(vm *goja.Runtime)
    GetFunctionNames() []string
}

func (r *FunctionRegistry) RegisterAllExtensions(vm *goja.Runtime) {
    for _, ext := range r.extensions {
        ext.RegisterAll(vm)
    }
}
```

### Phase 4: Workflow Data Proxy (Week 2-3)

#### 4.1 Core Data Proxy
```go
type WorkflowDataProxy struct {
    workflow           *Workflow
    runExecutionData   *RunExecutionData
    runIndex           int
    itemIndex          int
    activeNodeName     string
    connectionInputData []NodeExecutionData
    mode               WorkflowExecuteMode
    additionalKeys     *AdditionalKeys
    executeData        *ExecuteData
}

func (p *WorkflowDataProxy) CreateJSProxy(vm *goja.Runtime) goja.Value {
    proxy := vm.NewObject()

    // Set up context variables
    proxy.Set("$json", p.getJsonProxy(vm))
    proxy.Set("$input", p.getInputProxy(vm))
    proxy.Set("$node", p.getNodeProxy(vm))
    proxy.Set("$parameter", p.getParameterProxy(vm))
    proxy.Set("$workflow", p.getWorkflowProxy(vm))
    proxy.Set("$execution", p.getExecutionProxy(vm))
    proxy.Set("$env", p.getEnvProxy(vm))

    return proxy
}
```

#### 4.2 Variable Context Implementation
```go
func (p *WorkflowDataProxy) getJsonProxy(vm *goja.Runtime) goja.Value {
    if p.itemIndex >= len(p.connectionInputData) {
        return goja.Undefined()
    }

    item := p.connectionInputData[p.itemIndex]
    return vm.ToValue(item.JSON)
}

func (p *WorkflowDataProxy) getInputProxy(vm *goja.Runtime) goja.Value {
    inputProxy := vm.NewObject()

    // $input.all() - all items from all inputs
    inputProxy.Set("all", vm.ToValue(func(call goja.FunctionCall) goja.Value {
        return vm.ToValue(p.connectionInputData)
    }))

    // $input.first() - first item
    inputProxy.Set("first", vm.ToValue(func(call goja.FunctionCall) goja.Value {
        if len(p.connectionInputData) > 0 {
            return vm.ToValue(p.connectionInputData[0].JSON)
        }
        return goja.Undefined()
    }))

    // $input.last() - last item
    inputProxy.Set("last", vm.ToValue(func(call goja.FunctionCall) goja.Value {
        if len(p.connectionInputData) > 0 {
            return vm.ToValue(p.connectionInputData[len(p.connectionInputData)-1].JSON)
        }
        return goja.Undefined()
    }))

    return inputProxy
}

func (p *WorkflowDataProxy) getNodeProxy(vm *goja.Runtime) goja.Value {
    return vm.ToValue(func(call goja.FunctionCall) goja.Value {
        if len(call.Arguments) == 0 {
            return goja.Undefined()
        }

        nodeName := call.Arguments[0].String()
        nodeData := p.getNodeData(nodeName)

        nodeProxy := vm.NewObject()
        nodeProxy.Set("json", vm.ToValue(nodeData))
        nodeProxy.Set("binary", vm.ToValue(p.getNodeBinaryData(nodeName)))

        return nodeProxy
    })
}
```

### Phase 5: Expression Evaluation Engine (Week 3)

#### 5.1 Main Evaluator
```go
type ExpressionEvaluator struct {
    runtime       *SecureGojaRuntime
    functionRegistry *FunctionRegistry
    cache         map[string]*goja.Program  // Compiled program cache
    cacheMutex    sync.RWMutex
}

func (e *ExpressionEvaluator) EvaluateExpression(
    expression string,
    workflow *Workflow,
    runExecutionData *RunExecutionData,
    runIndex int,
    itemIndex int,
    activeNodeName string,
    connectionInputData []NodeExecutionData,
    mode WorkflowExecuteMode,
) (interface{}, error) {

    // 1. Parse expression
    ast, err := ParseExpression(expression)
    if err != nil {
        return nil, fmt.Errorf("expression parse error: %w", err)
    }

    // 2. Create data proxy
    proxy := &WorkflowDataProxy{
        workflow:            workflow,
        runExecutionData:    runExecutionData,
        runIndex:           runIndex,
        itemIndex:          itemIndex,
        activeNodeName:     activeNodeName,
        connectionInputData: connectionInputData,
        mode:               mode,
    }

    // 3. Set up runtime context
    vm := e.runtime.vm
    vm.Set("this", proxy.CreateJSProxy(vm))

    // 4. Execute expression
    result, err := e.executeCompiledExpression(ast.JavaScript)
    if err != nil {
        return nil, &ExpressionError{
            Message:     err.Error(),
            Expression:  expression,
            NodeName:    activeNodeName,
            ItemIndex:   itemIndex,
            RunIndex:    runIndex,
        }
    }

    return result, nil
}
```

#### 5.2 Caching Strategy
```go
func (e *ExpressionEvaluator) executeCompiledExpression(jsCode string) (interface{}, error) {
    e.cacheMutex.RLock()
    program, exists := e.cache[jsCode]
    e.cacheMutex.RUnlock()

    if !exists {
        var err error
        program, err = goja.Compile("expression", jsCode, false)
        if err != nil {
            return nil, fmt.Errorf("compile error: %w", err)
        }

        e.cacheMutex.Lock()
        e.cache[jsCode] = program
        e.cacheMutex.Unlock()
    }

    value, err := e.runtime.vm.RunProgram(program)
    if err != nil {
        return nil, err
    }

    return value.Export(), nil
}
```

### Phase 6: Integration & Testing (Week 4)

#### 6.1 Integration with Node Execution
```go
func (n *BaseNode) ResolveParameterValue(
    parameterValue interface{},
    runExecutionData *RunExecutionData,
    runIndex int,
    itemIndex int,
    activeNodeName string,
    connectionInputData []NodeExecutionData,
    mode WorkflowExecuteMode,
) (interface{}, error) {

    // Check if value is an expression
    if str, ok := parameterValue.(string); ok && strings.HasPrefix(str, "=") {
        expressionText := strings.TrimPrefix(str, "=")

        evaluator := GetExpressionEvaluator()
        return evaluator.EvaluateExpression(
            expressionText,
            n.workflow,
            runExecutionData,
            runIndex,
            itemIndex,
            activeNodeName,
            connectionInputData,
            mode,
        )
    }

    return parameterValue, nil
}
```

#### 6.2 Comprehensive Test Suite
```go
func TestExpressionCompatibility(t *testing.T) {
    tests := []struct{
        name       string
        expression string
        context    map[string]interface{}
        expected   interface{}
    }{
        // Basic variable access
        {"json_access", "{{ $json.name }}", map[string]interface{}{"name": "John"}, "John"},

        // String functions
        {"uppercase", "{{ uppercase($json.name) }}", map[string]interface{}{"name": "john"}, "JOHN"},
        {"split", "{{ split($json.text, ' ') }}", map[string]interface{}{"text": "hello world"}, []string{"hello", "world"}},

        // Math functions
        {"add", "{{ add(1, 2, 3) }}", map[string]interface{}{}, 6},
        {"arithmetic", "{{ 2 + 3 * 4 }}", map[string]interface{}{}, 14},

        // Array functions
        {"first", "{{ first($json.items) }}", map[string]interface{}{"items": []int{1, 2, 3}}, 1},
        {"last", "{{ last($json.items) }}", map[string]interface{}{"items": []int{1, 2, 3}}, 3},

        // Conditional expressions
        {"ternary", "{{ $json.value > 10 ? 'high' : 'low' }}", map[string]interface{}{"value": 15}, "high"},

        // Complex expressions
        {"nested", "{{ uppercase(split($json.name, ' ')[0]) }}", map[string]interface{}{"name": "john doe"}, "JOHN"},
    }

    evaluator := NewExpressionEvaluator()

    for _, test := range tests {
        t.Run(test.name, func(t *testing.T) {
            result, err := evaluator.EvaluateExpression(
                test.expression,
                createMockWorkflow(),
                createMockRunData(test.context),
                0, 0, "TestNode", []NodeExecutionData{},
                ModeManual,
            )

            assert.NoError(t, err)
            assert.Equal(t, test.expected, result)
        })
    }
}
```

## Performance Targets

### Benchmarks to Achieve
- **Expression Parsing**: < 1ms for typical expressions
- **Simple Variable Access**: < 0.1ms
- **Function Calls**: < 0.5ms
- **Complex Expressions**: < 5ms
- **Memory Usage**: < 10MB for expression runtime
- **Compilation Cache**: 90%+ hit rate for repeated expressions

### Optimization Strategies
1. **Program Compilation Caching**: Cache Goja programs by expression text
2. **Context Reuse**: Reuse runtime contexts when possible
3. **Lazy Data Loading**: Load context data on-demand
4. **JIT-like Optimization**: Pre-compile frequently used expressions
5. **Memory Pooling**: Pool Goja runtimes for concurrent execution

## Compatibility Guarantees

### 100% n8n Expression Compatibility
- All n8n expression syntax supported
- All 80+ built-in functions implemented
- Identical error handling and messages
- Same type coercion behavior
- Compatible with all n8n variable contexts

### Migration Path
- Drop-in replacement for existing n8n workflows
- No workflow modifications required
- Identical JSON output format
- Same credential integration patterns

## Security Compliance

### Sandboxing Features
- Identical blocked/allowed globals as n8n
- Prototype pollution protection
- Timeout enforcement for expressions
- Memory limit enforcement
- No file system access from expressions

### Enterprise Security
- Audit logging for expression evaluation
- Performance monitoring and alerting
- Resource usage tracking
- Malicious code detection

This implementation plan provides a complete roadmap for achieving 100% n8n expression compatibility using Goja while maintaining Go's performance advantages.