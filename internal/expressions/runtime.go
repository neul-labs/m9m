package expressions

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dop251/goja"
)

// SecureGojaRuntime provides a sandboxed JavaScript execution environment
// that exactly matches n8n's security model
type SecureGojaRuntime struct {
	vm               *goja.Runtime
	allowedGlobals   map[string]interface{}
	blockedGlobals   map[string]bool
	functionRegistry *FunctionRegistry
	timeout          time.Duration
	memoryLimit      int64
	executionCount   int64
	isInitialized    bool
	mutex            sync.RWMutex
}

// RuntimeConfig holds configuration for the Goja runtime
type RuntimeConfig struct {
	Timeout         time.Duration
	MemoryLimit     int64
	EnableProfiling bool
	MaxExecutions   int64
}

// DefaultRuntimeConfig returns a default runtime configuration
func DefaultRuntimeConfig() *RuntimeConfig {
	return &RuntimeConfig{
		Timeout:         30 * time.Second,
		MemoryLimit:     50 * 1024 * 1024, // 50MB
		EnableProfiling: false,
		MaxExecutions:   10000,
	}
}

// NewSecureGojaRuntime creates a new secure Goja runtime with n8n-compatible sandboxing
func NewSecureGojaRuntime(config *RuntimeConfig) *SecureGojaRuntime {
	if config == nil {
		config = DefaultRuntimeConfig()
	}

	runtime := &SecureGojaRuntime{
		vm:               goja.New(),
		allowedGlobals:   make(map[string]interface{}),
		blockedGlobals:   make(map[string]bool),
		functionRegistry: NewFunctionRegistry(),
		timeout:          config.Timeout,
		memoryLimit:      config.MemoryLimit,
		executionCount:   0,
		isInitialized:    false,
	}

	runtime.initializeSecurity()
	runtime.registerAllowedGlobals()
	runtime.blockDangerousGlobals()
	runtime.registerBuiltInFunctions()

	runtime.isInitialized = true
	return runtime
}

// initializeSecurity sets up the basic security features
func (r *SecureGojaRuntime) initializeSecurity() {
	// Set up console object (limited)
	console := r.vm.NewObject()
	console.Set("log", func(call goja.FunctionCall) goja.Value {
		// No-op for security - expressions shouldn't log
		return goja.Undefined()
	})
	r.vm.Set("console", console)

	// Disable dangerous constructor access
	r.vm.Set("constructor", goja.Undefined())
}

// registerAllowedGlobals registers globals that are allowed in n8n expressions
func (r *SecureGojaRuntime) registerAllowedGlobals() {
	// JavaScript built-in objects (exactly matching n8n)
	allowedGlobals := map[string]interface{}{
		// Core JavaScript objects
		"Array":               r.vm.Get("Array"),
		"Object":              r.vm.Get("Object"),
		"String":              r.vm.Get("String"),
		"Number":              r.vm.Get("Number"),
		"Boolean":             r.vm.Get("Boolean"),
		"Date":                r.vm.Get("Date"),
		"Math":                r.vm.Get("Math"),
		"JSON":                r.vm.Get("JSON"),
		"RegExp":              r.vm.Get("RegExp"),
		"Error":               r.vm.Get("Error"),
		"TypeError":           r.vm.Get("TypeError"),
		"ReferenceError":      r.vm.Get("ReferenceError"),
		"SyntaxError":         r.vm.Get("SyntaxError"),

		// Global functions
		"parseInt":            r.vm.Get("parseInt"),
		"parseFloat":          r.vm.Get("parseFloat"),
		"isNaN":               r.vm.Get("isNaN"),
		"isFinite":            r.vm.Get("isFinite"),
		"encodeURIComponent":  r.vm.Get("encodeURIComponent"),
		"decodeURIComponent":  r.vm.Get("decodeURIComponent"),
		"encodeURI":           r.vm.Get("encodeURI"),
		"decodeURI":           r.vm.Get("decodeURI"),

		// Typed arrays
		"Int8Array":           r.vm.Get("Int8Array"),
		"Uint8Array":          r.vm.Get("Uint8Array"),
		"Uint8ClampedArray":   r.vm.Get("Uint8ClampedArray"),
		"Int16Array":          r.vm.Get("Int16Array"),
		"Uint16Array":         r.vm.Get("Uint16Array"),
		"Int32Array":          r.vm.Get("Int32Array"),
		"Uint32Array":         r.vm.Get("Uint32Array"),
		"Float32Array":        r.vm.Get("Float32Array"),
		"Float64Array":        r.vm.Get("Float64Array"),
		"ArrayBuffer":         r.vm.Get("ArrayBuffer"),
		"DataView":            r.vm.Get("DataView"),

		// Collections
		"Map":                 r.vm.Get("Map"),
		"Set":                 r.vm.Get("Set"),
		"WeakMap":             r.vm.Get("WeakMap"),
		"WeakSet":             r.vm.Get("WeakSet"),

		// Internationalization
		"Intl":                r.vm.Get("Intl"),
	}

	// Store allowed globals for reference
	for name, value := range allowedGlobals {
		r.allowedGlobals[name] = value
	}

	// Register Luxon DateTime equivalent (Go implementation)
	r.registerDateTimeObjects()
}

// blockDangerousGlobals blocks globals that n8n considers dangerous
func (r *SecureGojaRuntime) blockDangerousGlobals() {
	// Exactly matching n8n's blocked globals
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
		"__dirname", "__filename",

		// Console (we provide our own limited version)
		"console",

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

		// Dangerous constructors
		"constructor", "__proto__", "prototype",
	}

	for _, name := range blockedGlobals {
		r.blockedGlobals[name] = true
		r.vm.Set(name, goja.Undefined())
	}
}

// registerDateTimeObjects registers Luxon-compatible DateTime objects
func (r *SecureGojaRuntime) registerDateTimeObjects() {
	// DateTime object (Go-based implementation of Luxon DateTime)
	dateTime := r.vm.NewObject()
	dateTime.Set("now", r.vm.ToValue(func(call goja.FunctionCall) goja.Value {
		return r.vm.ToValue(time.Now().Unix() * 1000) // JavaScript timestamp
	}))

	dateTime.Set("fromISO", r.vm.ToValue(func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			return goja.Undefined()
		}

		isoString := call.Arguments[0].String()
		t, err := time.Parse(time.RFC3339, isoString)
		if err != nil {
			return goja.Undefined()
		}

		dt := r.vm.NewObject()
		dt.Set("toMillis", r.vm.ToValue(func(call goja.FunctionCall) goja.Value {
			return r.vm.ToValue(t.Unix() * 1000)
		}))
		dt.Set("toISO", r.vm.ToValue(func(call goja.FunctionCall) goja.Value {
			return r.vm.ToValue(t.Format(time.RFC3339))
		}))
		dt.Set("toFormat", r.vm.ToValue(func(call goja.FunctionCall) goja.Value {
			if len(call.Arguments) == 0 {
				return r.vm.ToValue(t.Format(time.RFC3339))
			}
			// Simple format mapping (could be extended)
			format := call.Arguments[0].String()
			switch format {
			case "yyyy-MM-dd":
				return r.vm.ToValue(t.Format("2006-01-02"))
			case "yyyy-MM-dd HH:mm:ss":
				return r.vm.ToValue(t.Format("2006-01-02 15:04:05"))
			default:
				return r.vm.ToValue(t.Format(time.RFC3339))
			}
		}))

		return dt
	}))

	r.vm.Set("DateTime", dateTime)

	// Duration object
	duration := r.vm.NewObject()
	duration.Set("fromObject", r.vm.ToValue(func(call goja.FunctionCall) goja.Value {
		// Simple duration implementation
		return r.vm.NewObject()
	}))
	r.vm.Set("Duration", duration)

	// Interval object
	interval := r.vm.NewObject()
	r.vm.Set("Interval", interval)
}

// registerBuiltInFunctions registers all n8n built-in functions
func (r *SecureGojaRuntime) registerBuiltInFunctions() {
	r.functionRegistry.RegisterAllExtensions(r.vm)
}

// ExecuteWithTimeout executes JavaScript code with a timeout
func (r *SecureGojaRuntime) ExecuteWithTimeout(jsCode string, timeout time.Duration) (interface{}, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.executionCount++

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Channel to receive the result
	resultChan := make(chan interface{}, 1)
	errorChan := make(chan error, 1)

	// Execute in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errorChan <- fmt.Errorf("JavaScript execution panic: %v", r)
			}
		}()

		// Compile and run the program
		program, err := goja.Compile("expression", jsCode, false)
		if err != nil {
			errorChan <- fmt.Errorf("JavaScript compile error: %w", err)
			return
		}

		value, err := r.vm.RunProgram(program)
		if err != nil {
			errorChan <- fmt.Errorf("JavaScript execution error: %w", err)
			return
		}

		resultChan <- value.Export()
	}()

	// Wait for result or timeout
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("JavaScript execution timeout after %v", timeout)
	}
}

// Execute executes JavaScript code with the default timeout
func (r *SecureGojaRuntime) Execute(jsCode string) (interface{}, error) {
	return r.ExecuteWithTimeout(jsCode, r.timeout)
}

// SetContextVariable sets a variable in the JavaScript context
// SECURITY: Sanitizes objects to prevent prototype pollution attacks
func (r *SecureGojaRuntime) SetContextVariable(name string, value interface{}) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Convert value to goja value
	gojaValue := r.vm.ToValue(value)

	// SECURITY: If the value is an object, sanitize it to prevent prototype pollution
	if obj, ok := gojaValue.(*goja.Object); ok {
		r.sanitizeObject(obj)
	}

	r.vm.Set(name, gojaValue)
}

// GetContextVariable gets a variable from the JavaScript context
func (r *SecureGojaRuntime) GetContextVariable(name string) goja.Value {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.vm.Get(name)
}

// Reset resets the runtime to a clean state (for reuse)
func (r *SecureGojaRuntime) Reset() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Create new VM instance
	r.vm = goja.New()

	// Reinitialize security
	r.initializeSecurity()
	r.registerAllowedGlobals()
	r.blockDangerousGlobals()
	r.registerBuiltInFunctions()
}

// GetStats returns runtime statistics
func (r *SecureGojaRuntime) GetStats() map[string]interface{} {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return map[string]interface{}{
		"executionCount": r.executionCount,
		"isInitialized":  r.isInitialized,
		"timeout":        r.timeout,
		"memoryLimit":    r.memoryLimit,
	}
}

// sanitizeObject sanitizes a JavaScript object to prevent prototype pollution
func (r *SecureGojaRuntime) sanitizeObject(obj *goja.Object) {
	if obj == nil {
		return
	}

	// Block access to dangerous properties
	unsafeProps := []string{"__proto__", "prototype", "constructor", "getPrototypeOf"}
	for _, prop := range unsafeProps {
		obj.Delete(prop)
	}
}

// RuntimePool manages a pool of SecureGojaRuntime instances for better performance
type RuntimePool struct {
	runtimes chan *SecureGojaRuntime
	factory  func() *SecureGojaRuntime
	maxSize  int
}

// NewRuntimePool creates a new pool of Goja runtimes
func NewRuntimePool(maxSize int, config *RuntimeConfig) *RuntimePool {
	pool := &RuntimePool{
		runtimes: make(chan *SecureGojaRuntime, maxSize),
		maxSize:  maxSize,
		factory: func() *SecureGojaRuntime {
			return NewSecureGojaRuntime(config)
		},
	}

	// Pre-populate the pool
	for i := 0; i < maxSize/2; i++ {
		pool.runtimes <- pool.factory()
	}

	return pool
}

// Get retrieves a runtime from the pool
func (p *RuntimePool) Get() *SecureGojaRuntime {
	select {
	case runtime := <-p.runtimes:
		return runtime
	default:
		return p.factory()
	}
}

// Put returns a runtime to the pool
func (p *RuntimePool) Put(runtime *SecureGojaRuntime) {
	// Reset runtime state
	runtime.Reset()

	select {
	case p.runtimes <- runtime:
		// Successfully returned to pool
	default:
		// Pool is full, discard runtime
	}
}

// SetupDataProxy sets up the workflow data proxy in the runtime
func (r *SecureGojaRuntime) SetupDataProxy(dataProxy *WorkflowDataProxy) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Set up all the n8n context variables
	return dataProxy.Setup(r.vm)
}

// Close closes the runtime pool
func (p *RuntimePool) Close() {
	close(p.runtimes)
	// Drain remaining runtimes
	for range p.runtimes {
		// Just drain the channel
	}
}