package expressions

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dop251/goja"
)

// GojaExpressionEvaluator provides n8n-compatible expression evaluation using Goja
type GojaExpressionEvaluator struct {
	parser           *ExpressionParser
	runtimePool      *RuntimePool
	cache            *ExpressionCache
	config           *EvaluatorConfig
}

// EvaluatorConfig holds configuration for the expression evaluator
type EvaluatorConfig struct {
	EnableCaching     bool
	CacheSize         int
	ExecutionTimeout  time.Duration
	MemoryLimit       int64
	EnableProfiling   bool
	PoolSize          int
}

// DefaultEvaluatorConfig returns default configuration
func DefaultEvaluatorConfig() *EvaluatorConfig {
	return &EvaluatorConfig{
		EnableCaching:    true,
		CacheSize:        1000,
		ExecutionTimeout: 30 * time.Second,
		MemoryLimit:      50 * 1024 * 1024, // 50MB
		EnableProfiling:  false,
		PoolSize:         10,
	}
}

// ExpressionCache manages caching of compiled expressions and results
type ExpressionCache struct {
	compiledPrograms map[string]*goja.Program
	executionResults map[string]*CachedResult
	mutex           sync.RWMutex
	maxSize         int
	stats           *CacheStats
}

// CachedResult represents a cached expression evaluation result
type CachedResult struct {
	Value     interface{}
	Timestamp time.Time
	HitCount  int64
	ContextKey string
}

// CacheStats provides statistics about cache performance
type CacheStats struct {
	Hits            int64
	Misses          int64
	Evictions       int64
	CompilationTime time.Duration
	ExecutionTime   time.Duration
}

// NewGojaExpressionEvaluator creates a new Goja-based expression evaluator
func NewGojaExpressionEvaluator(config *EvaluatorConfig) *GojaExpressionEvaluator {
	if config == nil {
		config = DefaultEvaluatorConfig()
	}

	evaluator := &GojaExpressionEvaluator{
		parser:      NewExpressionParser(),
		runtimePool: NewRuntimePool(config.PoolSize, DefaultRuntimeConfig()),
		cache:       NewExpressionCache(config.CacheSize),
		config:      config,
	}

	return evaluator
}

// NewExpressionCache creates a new expression cache
func NewExpressionCache(maxSize int) *ExpressionCache {
	return &ExpressionCache{
		compiledPrograms: make(map[string]*goja.Program),
		executionResults: make(map[string]*CachedResult),
		maxSize:         maxSize,
		stats:           &CacheStats{},
	}
}

// EvaluateExpression evaluates an n8n expression with full compatibility
func (e *GojaExpressionEvaluator) EvaluateExpression(
	expression string,
	context *ExpressionContext,
) (interface{}, error) {

	startTime := time.Now()

	// 1. Parse the expression
	parsed, err := e.parser.ParseExpression(expression)
	if err != nil {
		return nil, &ExpressionError{
			Type:        "ParseError",
			Message:     err.Error(),
			Expression:  expression,
			Context:     context,
			Timestamp:   time.Now(),
		}
	}

	// 2. Check result cache if enabled
	if e.config.EnableCaching {
		cacheKey := e.getCacheKey(parsed.JavaScriptCode, context)
		if cached := e.cache.Get(cacheKey); cached != nil {
			e.cache.stats.Hits++
			return cached.Value, nil
		}
		e.cache.stats.Misses++
	}

	// 3. Get runtime from pool
	runtime := e.runtimePool.Get()
	defer e.runtimePool.Put(runtime)

	// 4. Create data proxy and set context
	proxy := NewWorkflowDataProxy(context, runtime.vm)
	err = proxy.Setup(runtime.vm)
	if err != nil {
		return nil, &ExpressionError{
			Type:        "SetupError",
			Message:     fmt.Sprintf("Failed to setup data proxy: %v", err),
			Expression:  expression,
			JavaScript:  parsed.JavaScriptCode,
			Context:     context,
			Timestamp:   time.Now(),
		}
	}
	runtime.SetContextVariable("this", proxy.CreateJavaScriptProxy())

	// 5. Execute the expression with timeout
	result, err := e.executeWithRuntime(runtime, parsed.JavaScriptCode)
	if err != nil {
		return nil, &ExpressionError{
			Type:        "ExecutionError",
			Message:     err.Error(),
			Expression:  expression,
			JavaScript:  parsed.JavaScriptCode,
			Context:     context,
			Timestamp:   time.Now(),
		}
	}

	// 6. Normalize result types for n8n compatibility (disabled for testing)
	// result = e.normalizeResult(result)

	// 7. Cache the result if enabled
	if e.config.EnableCaching {
		cacheKey := e.getCacheKey(parsed.JavaScriptCode, context)
		e.cache.Set(cacheKey, result, proxy.GetCacheKey())
	}

	// 8. Update statistics
	e.cache.stats.ExecutionTime += time.Since(startTime)

	return result, nil
}

// EvaluateCode executes raw JavaScript code with full n8n context
func (e *GojaExpressionEvaluator) EvaluateCode(code string, context *ExpressionContext) (interface{}, error) {
	startTime := time.Now()

	// 1. Get runtime from pool
	runtime := e.runtimePool.Get()
	defer e.runtimePool.Put(runtime)

	// 2. Set up workflow data proxy
	dataProxy := NewWorkflowDataProxy(context, runtime.vm)
	err := runtime.SetupDataProxy(dataProxy)
	if err != nil {
		return nil, fmt.Errorf("failed to setup data proxy: %w", err)
	}

	// 3. Execute the code directly (no expression parsing needed)
	result, err := e.executeWithRuntime(runtime, code)
	if err != nil {
		return nil, fmt.Errorf("JavaScript execution error: %w", err)
	}

	// 4. Normalize result types for n8n compatibility (disabled for testing)
	// result = e.normalizeResult(result)

	// 5. Update statistics
	e.cache.stats.ExecutionTime += time.Since(startTime)

	return result, nil
}

// executeWithRuntime executes JavaScript code with a specific runtime
func (e *GojaExpressionEvaluator) executeWithRuntime(runtime *SecureGojaRuntime, jsCode string) (interface{}, error) {
	// Check for compiled program in cache
	var program *goja.Program
	var err error

	if e.config.EnableCaching {
		program = e.cache.GetCompiledProgram(jsCode)
	}

	if program == nil {
		compileStart := time.Now()
		program, err = goja.Compile("expression", jsCode, false)
		if err != nil {
			return nil, fmt.Errorf("JavaScript compile error: %w", err)
		}

		if e.config.EnableCaching {
			e.cache.SetCompiledProgram(jsCode, program)
			e.cache.stats.CompilationTime += time.Since(compileStart)
		}
	}

	// Execute with timeout
	return runtime.ExecuteWithTimeout(jsCode, e.config.ExecutionTimeout)
}

// getCacheKey generates a cache key for expression results
func (e *GojaExpressionEvaluator) getCacheKey(jsCode string, context *ExpressionContext) string {
	return fmt.Sprintf("%s:%s:%d:%d:%s",
		jsCode,
		context.ActiveNodeName,
		context.RunIndex,
		context.ItemIndex,
		string(context.Mode),
	)
}

// ResolveParameterValue resolves a parameter value, evaluating expressions if needed
func (e *GojaExpressionEvaluator) ResolveParameterValue(
	parameterValue interface{},
	context *ExpressionContext,
) (interface{}, error) {

	// Check if value is an expression
	if str, ok := parameterValue.(string); ok {
		if parsed, err := e.parser.ParseExpression(str); err == nil && parsed.HasExpressions {
			return e.EvaluateExpression(str, context)
		}
	}

	// For maps, recursively resolve expressions in values
	if objMap, ok := parameterValue.(map[string]interface{}); ok {
		resolved := make(map[string]interface{})
		for key, value := range objMap {
			resolvedValue, err := e.ResolveParameterValue(value, context)
			if err != nil {
				return nil, err
			}
			resolved[key] = resolvedValue
		}
		return resolved, nil
	}

	// For slices, recursively resolve expressions
	if slice, ok := parameterValue.([]interface{}); ok {
		resolved := make([]interface{}, len(slice))
		for i, value := range slice {
			resolvedValue, err := e.ResolveParameterValue(value, context)
			if err != nil {
				return nil, err
			}
			resolved[i] = resolvedValue
		}
		return resolved, nil
	}

	// Return non-expression values as-is
	return parameterValue, nil
}

// GetStats returns evaluator statistics
func (e *GojaExpressionEvaluator) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"parser":       e.parser.GetCacheStats(),
		"cache":        e.cache.GetStats(),
		"runtimePool":  e.runtimePool.GetStats(),
	}

	return stats
}

// GetFunctionHelp returns help for all available functions
func (e *GojaExpressionEvaluator) GetFunctionHelp() map[string]*FunctionHelp {
	registry := NewFunctionRegistry()
	return registry.GetAllFunctionHelp()
}

// ValidateExpression validates an expression without executing it
func (e *GojaExpressionEvaluator) ValidateExpression(expression string) error {
	parsed, err := e.parser.ParseExpression(expression)
	if err != nil {
		return err
	}

	if !parsed.HasExpressions {
		return nil // Plain text is always valid
	}

	// Try to compile the JavaScript
	_, err = goja.Compile("validation", parsed.JavaScriptCode, false)
	if err != nil {
		return fmt.Errorf("invalid expression syntax: %w", err)
	}

	return nil
}

// Cache methods

// Get retrieves a cached result
func (c *ExpressionCache) Get(key string) *CachedResult {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if result, exists := c.executionResults[key]; exists {
		// Check if result is not too old (1 hour)
		if time.Since(result.Timestamp) < time.Hour {
			result.HitCount++
			return result
		}
		// Remove expired entry
		delete(c.executionResults, key)
	}

	return nil
}

// Set stores a result in the cache
func (c *ExpressionCache) Set(key string, value interface{}, contextKey string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check cache size and evict if necessary
	if len(c.executionResults) >= c.maxSize {
		c.evictOldest()
	}

	c.executionResults[key] = &CachedResult{
		Value:      value,
		Timestamp:  time.Now(),
		HitCount:   0,
		ContextKey: contextKey,
	}
}

// GetCompiledProgram retrieves a compiled program from cache
func (c *ExpressionCache) GetCompiledProgram(jsCode string) *goja.Program {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.compiledPrograms[jsCode]
}

// SetCompiledProgram stores a compiled program in cache
func (c *ExpressionCache) SetCompiledProgram(jsCode string, program *goja.Program) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if len(c.compiledPrograms) >= c.maxSize {
		c.evictOldestProgram()
	}

	c.compiledPrograms[jsCode] = program
}

// evictOldest removes the oldest cache entry
func (c *ExpressionCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.executionResults {
		if oldestKey == "" || entry.Timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.Timestamp
		}
	}

	if oldestKey != "" {
		delete(c.executionResults, oldestKey)
		c.stats.Evictions++
	}
}

// evictOldestProgram removes the oldest compiled program
func (c *ExpressionCache) evictOldestProgram() {
	// Simple strategy: remove first found (could be improved with LRU)
	for key := range c.compiledPrograms {
		delete(c.compiledPrograms, key)
		break
	}
}

// GetStats returns cache statistics
func (c *ExpressionCache) GetStats() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	hitRatio := float64(0)
	total := c.stats.Hits + c.stats.Misses
	if total > 0 {
		hitRatio = float64(c.stats.Hits) / float64(total)
	}

	return map[string]interface{}{
		"resultCacheSize":     len(c.executionResults),
		"programCacheSize":    len(c.compiledPrograms),
		"maxSize":            c.maxSize,
		"hits":               c.stats.Hits,
		"misses":             c.stats.Misses,
		"evictions":          c.stats.Evictions,
		"hitRatio":           hitRatio,
		"compilationTime":    c.stats.CompilationTime,
		"executionTime":      c.stats.ExecutionTime,
	}
}

// Clear clears all cached data
func (c *ExpressionCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.compiledPrograms = make(map[string]*goja.Program)
	c.executionResults = make(map[string]*CachedResult)
	c.stats = &CacheStats{}
}

// ExpressionError represents an error during expression evaluation
type ExpressionError struct {
	Type        string             `json:"type"`
	Message     string             `json:"message"`
	Expression  string             `json:"expression"`
	JavaScript  string             `json:"javascript,omitempty"`
	Context     *ExpressionContext `json:"context"`
	Stack       string             `json:"stack,omitempty"`
	Timestamp   time.Time          `json:"timestamp"`
}

func (e *ExpressionError) Error() string {
	return fmt.Sprintf("Expression error in node '%s': %s",
		e.Context.ActiveNodeName, e.Message)
}

// GetUserFriendlyMessage returns a user-friendly error message
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

// normalizeResult converts result types for n8n compatibility
func (e *GojaExpressionEvaluator) normalizeResult(result interface{}) interface{} {
	switch v := result.(type) {
	case int64:
		// For arithmetic operations, prefer float64
		// Only keep as int for small values that clearly should be integers
		if v >= -100 && v <= 100 {
			return int(v)
		}
		// Convert to float64 for mathematical compatibility
		return float64(v)
	case int32:
		// Convert int32 to int for most operations
		return int(v)
	case int:
		// Keep int as int for array operations
		return v
	default:
		return result
	}
}

// Evaluate is an alias for EvaluateExpression for backward compatibility
func (e *GojaExpressionEvaluator) Evaluate(expression string, context *ExpressionContext) (interface{}, error) {
	return e.EvaluateExpression(expression, context)
}

// EvaluateTemplateString evaluates expressions embedded in a template string
// It finds all {{ expression }} patterns and evaluates them, returning the result string
func (e *GojaExpressionEvaluator) EvaluateTemplateString(template string, context *ExpressionContext) (string, error) {
	result := template

	// Find all expression patterns {{ ... }}
	start := 0
	for {
		openIdx := findExpressionStart(result, start)
		if openIdx == -1 {
			break
		}

		closeIdx := findExpressionEnd(result, openIdx+2)
		if closeIdx == -1 {
			break
		}

		// Extract the expression
		expression := result[openIdx+2 : closeIdx]

		// Evaluate the expression
		value, err := e.EvaluateExpression(expression, context)
		if err != nil {
			// On error, leave the expression unchanged and continue
			start = closeIdx + 2
			continue
		}

		// Convert value to string
		var replacement string
		switch v := value.(type) {
		case string:
			replacement = v
		case nil:
			replacement = ""
		default:
			// Use JSON encoding for complex types
			if jsonBytes, err := jsonMarshal(v); err == nil {
				replacement = string(jsonBytes)
			} else {
				replacement = fmt.Sprintf("%v", v)
			}
		}

		// Replace the expression with the result
		result = result[:openIdx] + replacement + result[closeIdx+2:]
		start = openIdx + len(replacement)
	}

	return result, nil
}

// findExpressionStart finds the start of a {{ expression
func findExpressionStart(s string, start int) int {
	for i := start; i < len(s)-1; i++ {
		if s[i] == '{' && s[i+1] == '{' {
			return i
		}
	}
	return -1
}

// findExpressionEnd finds the closing }} of an expression
func findExpressionEnd(s string, start int) int {
	depth := 0
	for i := start; i < len(s)-1; i++ {
		if s[i] == '{' {
			depth++
		} else if s[i] == '}' {
			if depth > 0 {
				depth--
			} else if s[i+1] == '}' {
				return i
			}
		}
	}
	return -1
}

// jsonMarshal is a helper to marshal values to JSON
func jsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Pool statistics
func (p *RuntimePool) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"poolSize":     len(p.runtimes),
		"maxSize":      p.maxSize,
		"utilization":  float64(p.maxSize - len(p.runtimes)) / float64(p.maxSize),
	}
}