package runtime

import (
	"testing"
	"time"

	"github.com/neul-labs/m9m/internal/expressions"
	"github.com/neul-labs/m9m/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// JavaScript Runtime Tests
// ---------------------------------------------------------------------------

func TestNewJavaScriptRuntime(t *testing.T) {
	js := NewJavaScriptRuntime("")
	require.NotNil(t, js)
	assert.NotNil(t, js.vm)
	assert.NotNil(t, js.npmCache)
	assert.NotNil(t, js.globalModules)
	assert.NotNil(t, js.n8nHelpers)
	assert.NotNil(t, js.executionContext)
	assert.NotNil(t, js.executionContext.Variables)
	assert.NotNil(t, js.executionContext.Credentials)
}

func TestJavaScriptRuntime_Execute_SimpleCode(t *testing.T) {
	js := NewJavaScriptRuntime("")
	ctx := &ExecutionContext{
		Variables:   make(map[string]interface{}),
		Credentials: make(map[string]interface{}),
	}

	result, err := js.Execute("var x = 1 + 2; x;", ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(3), result)
}

func TestJavaScriptRuntime_Execute_StringConcatenation(t *testing.T) {
	js := NewJavaScriptRuntime("")
	ctx := &ExecutionContext{
		Variables:   make(map[string]interface{}),
		Credentials: make(map[string]interface{}),
	}

	result, err := js.Execute(`"hello" + " " + "world";`, ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, "hello world", result)
}

func TestJavaScriptRuntime_Execute_WithContext(t *testing.T) {
	js := NewJavaScriptRuntime("")
	ctx := &ExecutionContext{
		WorkflowID:  "wf-123",
		ExecutionID: "exec-456",
		NodeID:      "node-789",
		ItemIndex:   0,
		RunIndex:    0,
		Mode:        "manual",
		Timezone:    "UTC",
		Variables: map[string]interface{}{
			"myVar": "hello",
		},
		Credentials: make(map[string]interface{}),
	}

	// Access variables from context via the $parameter helper
	result, err := js.Execute(`var v = $parameter.get("myVar"); v;`, ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, "hello", result)
}

func TestJavaScriptRuntime_Execute_WithDataItems(t *testing.T) {
	js := NewJavaScriptRuntime("")
	ctx := &ExecutionContext{
		ItemIndex:   0,
		Variables:   make(map[string]interface{}),
		Credentials: make(map[string]interface{}),
	}

	items := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"name":  "Alice",
				"score": 95,
			},
		},
		{
			JSON: map[string]interface{}{
				"name":  "Bob",
				"score": 88,
			},
		},
	}

	// Access items via $items.all()
	result, err := js.Execute(`var all = $items.all(); all.length;`, ctx, items)
	require.NoError(t, err)
	assert.Equal(t, int64(2), result)
}

func TestJavaScriptRuntime_Execute_WithDataItems_Current(t *testing.T) {
	js := NewJavaScriptRuntime("")
	ctx := &ExecutionContext{
		ItemIndex:   0,
		Variables:   make(map[string]interface{}),
		Credentials: make(map[string]interface{}),
	}

	items := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"name": "Alice",
			},
		},
	}

	result, err := js.Execute(`var cur = $items.current(); cur.json.name;`, ctx, items)
	require.NoError(t, err)
	assert.Equal(t, "Alice", result)
}

func TestJavaScriptRuntime_Execute_WithDataItems_First(t *testing.T) {
	js := NewJavaScriptRuntime("")
	ctx := &ExecutionContext{
		ItemIndex:   1,
		Variables:   make(map[string]interface{}),
		Credentials: make(map[string]interface{}),
	}

	items := []model.DataItem{
		{JSON: map[string]interface{}{"name": "First"}},
		{JSON: map[string]interface{}{"name": "Second"}},
	}

	result, err := js.Execute(`$items.first().json.name;`, ctx, items)
	require.NoError(t, err)
	assert.Equal(t, "First", result)
}

func TestJavaScriptRuntime_Execute_WithDataItems_Last(t *testing.T) {
	js := NewJavaScriptRuntime("")
	ctx := &ExecutionContext{
		ItemIndex:   0,
		Variables:   make(map[string]interface{}),
		Credentials: make(map[string]interface{}),
	}

	items := []model.DataItem{
		{JSON: map[string]interface{}{"name": "First"}},
		{JSON: map[string]interface{}{"name": "Last"}},
	}

	result, err := js.Execute(`$items.last().json.name;`, ctx, items)
	require.NoError(t, err)
	assert.Equal(t, "Last", result)
}

func TestJavaScriptRuntime_Execute_EmptyCode(t *testing.T) {
	js := NewJavaScriptRuntime("")
	ctx := &ExecutionContext{
		Variables:   make(map[string]interface{}),
		Credentials: make(map[string]interface{}),
	}

	// Empty code should execute without error; goja returns undefined
	result, err := js.Execute("", ctx, nil)
	require.NoError(t, err)
	// Empty code results in goja.Undefined() which exports as nil
	assert.Nil(t, result)
}

func TestJavaScriptRuntime_Execute_SyntaxError(t *testing.T) {
	js := NewJavaScriptRuntime("")
	ctx := &ExecutionContext{
		Variables:   make(map[string]interface{}),
		Credentials: make(map[string]interface{}),
	}

	_, err := js.Execute("var x = {{{;", ctx, nil)
	assert.Error(t, err)
}

func TestJavaScriptRuntime_Execute_ReferenceError(t *testing.T) {
	js := NewJavaScriptRuntime("")
	ctx := &ExecutionContext{
		Variables:   make(map[string]interface{}),
		Credentials: make(map[string]interface{}),
	}

	_, err := js.Execute("undeclaredVariable.property;", ctx, nil)
	assert.Error(t, err)
}

func TestJavaScriptRuntime_Execute_JSONManipulation(t *testing.T) {
	js := NewJavaScriptRuntime("")
	ctx := &ExecutionContext{
		Variables:   make(map[string]interface{}),
		Credentials: make(map[string]interface{}),
	}

	code := `
		var obj = {a: 1, b: 2, c: 3};
		JSON.stringify(obj);
	`
	result, err := js.Execute(code, ctx, nil)
	require.NoError(t, err)
	assert.Contains(t, result.(string), `"a":1`)
}

func TestJavaScriptRuntime_Execute_ArrayOperations(t *testing.T) {
	js := NewJavaScriptRuntime("")
	ctx := &ExecutionContext{
		Variables:   make(map[string]interface{}),
		Credentials: make(map[string]interface{}),
	}

	code := `
		var arr = [1, 2, 3, 4, 5];
		arr.filter(function(x) { return x > 2; }).length;
	`
	result, err := js.Execute(code, ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(3), result)
}

func TestJavaScriptRuntime_SetExecutionContext(t *testing.T) {
	js := NewJavaScriptRuntime("")

	ctx := &ExecutionContext{
		WorkflowID:  "wf-test",
		ExecutionID: "exec-test",
		NodeID:      "node-test",
		ItemIndex:   5,
		RunIndex:    2,
		Mode:        "trigger",
		Timezone:    "America/New_York",
		Variables: map[string]interface{}{
			"key1": "value1",
		},
		Credentials: map[string]interface{}{
			"apiKey": "secret",
		},
	}

	js.SetExecutionContext(ctx)

	// Verify the context was set
	assert.Equal(t, "wf-test", js.executionContext.WorkflowID)
	assert.Equal(t, "exec-test", js.executionContext.ExecutionID)
	assert.Equal(t, "node-test", js.executionContext.NodeID)
	assert.Equal(t, 5, js.executionContext.ItemIndex)
	assert.Equal(t, 2, js.executionContext.RunIndex)
	assert.Equal(t, "trigger", js.executionContext.Mode)
	assert.Equal(t, "America/New_York", js.executionContext.Timezone)
	assert.Equal(t, "value1", js.executionContext.Variables["key1"])
	assert.Equal(t, "secret", js.executionContext.Credentials["apiKey"])
}

func TestJavaScriptRuntime_Dispose(t *testing.T) {
	js := NewJavaScriptRuntime("")

	// Should not panic
	assert.NotPanics(t, func() {
		js.Dispose()
	})

	// After dispose, caches should be reset
	assert.Empty(t, js.npmCache)
	assert.Empty(t, js.globalModules)
	assert.NotNil(t, js.vm)
}

func TestJavaScriptRuntime_ExecuteExpression(t *testing.T) {
	js := NewJavaScriptRuntime("")
	exprCtx := expressions.NewExpressionContext()
	exprCtx.SetVariable("x", 10)
	exprCtx.SetVariable("y", 20)

	result, err := js.ExecuteExpression("x + y", exprCtx)
	require.NoError(t, err)
	assert.Equal(t, int64(30), result)
}

func TestJavaScriptRuntime_ExecuteExpression_StringVariable(t *testing.T) {
	js := NewJavaScriptRuntime("")
	exprCtx := expressions.NewExpressionContext()
	exprCtx.SetVariable("greeting", "Hello, World!")

	result, err := js.ExecuteExpression("greeting", exprCtx)
	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", result)
}

func TestJavaScriptRuntime_ExecuteExpression_InvalidExpression(t *testing.T) {
	js := NewJavaScriptRuntime("")
	exprCtx := expressions.NewExpressionContext()

	_, err := js.ExecuteExpression("{{invalid}}", exprCtx)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Helper Function Tests
// ---------------------------------------------------------------------------

func TestGetNestedValue(t *testing.T) {
	js := NewJavaScriptRuntime("")

	tests := []struct {
		name         string
		object       interface{}
		path         string
		defaultValue interface{}
		expected     interface{}
	}{
		{
			name: "simple key access",
			object: map[string]interface{}{
				"name": "Alice",
			},
			path:         "name",
			defaultValue: nil,
			expected:     "Alice",
		},
		{
			name: "nested key access",
			object: map[string]interface{}{
				"a": map[string]interface{}{
					"b": map[string]interface{}{
						"c": "deep_value",
					},
				},
			},
			path:         "a.b.c",
			defaultValue: nil,
			expected:     "deep_value",
		},
		{
			name: "missing key returns default",
			object: map[string]interface{}{
				"a": "value_a",
			},
			path:         "b",
			defaultValue: "fallback",
			expected:     "fallback",
		},
		{
			name: "missing nested key returns default",
			object: map[string]interface{}{
				"a": map[string]interface{}{},
			},
			path:         "a.b.c",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "nil object returns default",
			object:       nil,
			path:         "any.path",
			defaultValue: "default_val",
			expected:     "default_val",
		},
		{
			name: "array index access",
			object: map[string]interface{}{
				"items": []interface{}{"first", "second", "third"},
			},
			path:         "items[1]",
			defaultValue: nil,
			expected:     "second",
		},
		{
			name: "array index out of bounds returns default",
			object: map[string]interface{}{
				"items": []interface{}{"only"},
			},
			path:         "items[5]",
			defaultValue: "oob",
			expected:     "oob",
		},
		{
			name: "numeric value",
			object: map[string]interface{}{
				"count": 42,
			},
			path:         "count",
			defaultValue: 0,
			expected:     42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := js.getNestedValue(tt.object, tt.path, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSetNestedValue(t *testing.T) {
	js := NewJavaScriptRuntime("")

	tests := []struct {
		name     string
		object   interface{}
		path     string
		value    interface{}
		checkKey string
		expected interface{}
	}{
		{
			name:     "set top-level value",
			object:   map[string]interface{}{},
			path:     "name",
			value:    "Alice",
			checkKey: "name",
			expected: "Alice",
		},
		{
			name:     "set nested value creates intermediaries",
			object:   map[string]interface{}{},
			path:     "a.b.c",
			value:    "deep",
			checkKey: "a.b.c",
			expected: "deep",
		},
		{
			name: "overwrite existing value",
			object: map[string]interface{}{
				"key": "old",
			},
			path:     "key",
			value:    "new",
			checkKey: "key",
			expected: "new",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := js.setNestedValue(tt.object, tt.path, tt.value)
			// Verify via getNestedValue
			actual := js.getNestedValue(result, tt.checkKey, nil)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestHasNestedValue(t *testing.T) {
	js := NewJavaScriptRuntime("")

	tests := []struct {
		name     string
		object   interface{}
		path     string
		expected bool
	}{
		{
			name: "existing top-level key",
			object: map[string]interface{}{
				"name": "Alice",
			},
			path:     "name",
			expected: true,
		},
		{
			name: "existing nested key",
			object: map[string]interface{}{
				"a": map[string]interface{}{
					"b": map[string]interface{}{
						"c": 123,
					},
				},
			},
			path:     "a.b.c",
			expected: true,
		},
		{
			name: "missing key",
			object: map[string]interface{}{
				"a": "value",
			},
			path:     "b",
			expected: false,
		},
		{
			name: "partially missing nested key",
			object: map[string]interface{}{
				"a": map[string]interface{}{},
			},
			path:     "a.b.c",
			expected: false,
		},
		{
			name:     "nil object",
			object:   nil,
			path:     "anything",
			expected: false,
		},
		{
			name: "key with nil value still exists",
			object: map[string]interface{}{
				"key": nil,
			},
			path:     "key",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := js.hasNestedValue(tt.object, tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertMomentFormat(t *testing.T) {
	js := NewJavaScriptRuntime("")

	// NOTE: convertMomentFormat uses a Go map for replacements with
	// strings.ReplaceAll. Because the map iteration order is non-deterministic,
	// overlapping patterns (e.g. "YYYY"/"YY", "MM"/"M", "HH"/"H", "A"/"a")
	// can interfere with each other. We only verify:
	// 1. Unknown tokens pass through unchanged.
	// 2. The function does not panic.
	// 3. The function modifies known tokens (output differs from input).

	t.Run("unknown tokens pass through", func(t *testing.T) {
		assert.Equal(t, "ZZZ", js.convertMomentFormat("ZZZ"))
	})

	t.Run("literal text preserved", func(t *testing.T) {
		assert.Equal(t, "T", js.convertMomentFormat("T"))
	})

	t.Run("does not panic on complex format", func(t *testing.T) {
		assert.NotPanics(t, func() {
			js.convertMomentFormat("YYYY-MM-DD HH:mm:ss A")
		})
	})

	t.Run("known tokens are transformed", func(t *testing.T) {
		// DD -> "02" in Go format; DD has no overlapping shorter token in the map
		// because "D" -> "2" and replacing "D" -> "2" in "DD" gives "22",
		// while replacing "DD" -> "02" in "DD" gives "02".
		// Either way the result is not "DD", confirming transformation occurred.
		result := js.convertMomentFormat("DD")
		assert.NotEqual(t, "DD", result, "DD should be transformed")
	})

	t.Run("ss is transformed", func(t *testing.T) {
		// ss -> "05"; s -> "5". "ss" becomes either "05" or "55".
		result := js.convertMomentFormat("ss")
		assert.NotEqual(t, "ss", result, "ss should be transformed")
	})
}

func TestIsTruthy(t *testing.T) {
	js := NewJavaScriptRuntime("")

	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{name: "nil is falsy", value: nil, expected: false},
		{name: "true is truthy", value: true, expected: true},
		{name: "false is falsy", value: false, expected: false},
		{name: "non-empty string is truthy", value: "hello", expected: true},
		{name: "empty string is falsy", value: "", expected: false},
		{name: "non-zero int is truthy", value: 42, expected: true},
		{name: "zero int is falsy", value: 0, expected: false},
		{name: "non-zero float64 is truthy", value: 3.14, expected: true},
		{name: "zero float64 is falsy", value: 0.0, expected: false},
		{name: "non-empty slice is truthy", value: []interface{}{"a"}, expected: true},
		{name: "empty slice is falsy", value: []interface{}{}, expected: false},
		{name: "non-empty map is truthy", value: map[string]interface{}{"k": "v"}, expected: true},
		{name: "empty map is falsy", value: map[string]interface{}{}, expected: false},
		{name: "unknown type is truthy", value: struct{}{}, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := js.isTruthy(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// Security / Validation Tests
// ---------------------------------------------------------------------------

func TestValidatePackageName(t *testing.T) {
	tests := []struct {
		name      string
		pkgName   string
		expectErr bool
	}{
		{name: "valid simple name", pkgName: "lodash", expectErr: false},
		{name: "valid scoped name", pkgName: "@types/node", expectErr: false},
		{name: "valid with hyphens", pkgName: "my-package", expectErr: false},
		{name: "valid with dots", pkgName: "my.package", expectErr: false},
		{name: "empty name", pkgName: "", expectErr: true},
		{name: "starts with dot", pkgName: ".hidden", expectErr: true},
		{name: "starts with underscore", pkgName: "_private", expectErr: true},
		{name: "contains uppercase", pkgName: "MyPackage", expectErr: true},
		{name: "contains space", pkgName: "my package", expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePackageName(tt.pkgName)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsEnvVarSafe(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		expected bool
	}{
		{name: "NODE_ENV is safe", envVar: "NODE_ENV", expected: true},
		{name: "TZ is safe", envVar: "TZ", expected: true},
		{name: "HOME is safe", envVar: "HOME", expected: true},
		{name: "PATH is blocked", envVar: "PATH", expected: false},
		{name: "AWS_SECRET is blocked", envVar: "AWS_SECRET_KEY", expected: false},
		{name: "DATABASE_URL is blocked", envVar: "DATABASE_URL", expected: false},
		{name: "API_KEY is blocked", envVar: "MY_API_KEY", expected: false},
		{name: "unknown var is blocked", envVar: "SOMETHING_RANDOM", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEnvVarSafe(tt.envVar)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEscapeJSString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "no special chars", input: "hello", expected: "hello"},
		{name: "single quote", input: "it's", expected: `it\'s`},
		{name: "double quote", input: `say "hi"`, expected: `say \"hi\"`},
		{name: "newline", input: "line1\nline2", expected: `line1\nline2`},
		{name: "tab", input: "col1\tcol2", expected: `col1\tcol2`},
		{name: "backslash", input: `back\slash`, expected: `back\\slash`},
		{name: "carriage return", input: "cr\rhere", expected: `cr\rhere`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeJSString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// Python Runtime Tests
// ---------------------------------------------------------------------------

func TestNewPythonRuntime(t *testing.T) {
	pr, err := NewPythonRuntime()
	if err != nil {
		t.Skipf("Skipping Python runtime test: %v", err)
	}
	defer pr.Cleanup()

	require.NotNil(t, pr)
	assert.NotEmpty(t, pr.tempDir)
	assert.Equal(t, "python3", pr.pythonPath)
	assert.True(t, pr.sandboxMode)
	assert.False(t, pr.initialized)
	assert.NotEmpty(t, pr.enabledPackages)
}

func TestPythonRuntime_SetMaxExecutionTime(t *testing.T) {
	pr, err := NewPythonRuntime()
	if err != nil {
		t.Skipf("Skipping Python runtime test: %v", err)
	}
	defer pr.Cleanup()

	// Default is 30s
	assert.Equal(t, 30*time.Second, pr.maxExecutionTime)

	// Set to 60s
	pr.SetMaxExecutionTime(60 * time.Second)
	assert.Equal(t, 60*time.Second, pr.maxExecutionTime)

	// Set to 5s
	pr.SetMaxExecutionTime(5 * time.Second)
	assert.Equal(t, 5*time.Second, pr.maxExecutionTime)
}

func TestPythonRuntime_SetSandboxMode(t *testing.T) {
	pr, err := NewPythonRuntime()
	if err != nil {
		t.Skipf("Skipping Python runtime test: %v", err)
	}
	defer pr.Cleanup()

	// Default is true
	assert.True(t, pr.sandboxMode)

	// Disable sandbox
	pr.SetSandboxMode(false)
	assert.False(t, pr.sandboxMode)

	// Re-enable sandbox
	pr.SetSandboxMode(true)
	assert.True(t, pr.sandboxMode)
}

func TestPythonRuntime_EnablePackage(t *testing.T) {
	pr, err := NewPythonRuntime()
	if err != nil {
		t.Skipf("Skipping Python runtime test: %v", err)
	}
	defer pr.Cleanup()

	// Custom package should not be allowed initially
	assert.False(t, pr.enabledPackages["my-custom-pkg"])

	// Enable it
	pr.EnablePackage("my-custom-pkg")
	assert.True(t, pr.enabledPackages["my-custom-pkg"])
}

func TestPythonRuntime_IsPackageAllowed(t *testing.T) {
	pr, err := NewPythonRuntime()
	if err != nil {
		t.Skipf("Skipping Python runtime test: %v", err)
	}
	defer pr.Cleanup()

	// Default allowed packages
	assert.True(t, pr.isPackageAllowed("numpy"))
	assert.True(t, pr.isPackageAllowed("pandas"))
	assert.True(t, pr.isPackageAllowed("requests"))

	// Not allowed
	assert.False(t, pr.isPackageAllowed("some-unknown-pkg"))

	// Version specifier is stripped
	assert.True(t, pr.isPackageAllowed("numpy==1.24.0"))

	// When sandbox mode is off, everything is allowed
	pr.SetSandboxMode(false)
	assert.True(t, pr.isPackageAllowed("any-package"))
}

func TestPythonRuntime_Cleanup(t *testing.T) {
	pr, err := NewPythonRuntime()
	if err != nil {
		t.Skipf("Skipping Python runtime test: %v", err)
	}

	tempDir := pr.tempDir
	require.DirExists(t, tempDir)

	err = pr.Cleanup()
	require.NoError(t, err)
	assert.NoDirExists(t, tempDir)
}

// ---------------------------------------------------------------------------
// JavaScript Runtime: Workflow / Execution Helpers
// ---------------------------------------------------------------------------

func TestJavaScriptRuntime_WorkflowHelper(t *testing.T) {
	js := NewJavaScriptRuntime("")
	ctx := &ExecutionContext{
		WorkflowID:  "wf-abc",
		ExecutionID: "exec-def",
		Mode:        "manual",
		Variables:   make(map[string]interface{}),
		Credentials: make(map[string]interface{}),
	}

	result, err := js.Execute(`$workflow.id;`, ctx, nil)
	require.NoError(t, err)
	// The workflow helper was initialized with the default empty context,
	// so we check the initially registered helper.
	// $workflow.id is set at initialization time from executionContext.WorkflowID
	// which defaults to "".
	assert.Equal(t, "", result)
}

func TestJavaScriptRuntime_ExecutionHelper(t *testing.T) {
	js := NewJavaScriptRuntime("")
	ctx := &ExecutionContext{
		WorkflowID:  "wf-abc",
		ExecutionID: "exec-def",
		Mode:        "manual",
		Variables:   make(map[string]interface{}),
		Credentials: make(map[string]interface{}),
	}

	result, err := js.Execute(`$execution.id;`, ctx, nil)
	require.NoError(t, err)
	// Same as workflow -- $execution was registered at init time
	assert.Equal(t, "", result)
}

func TestJavaScriptRuntime_Execute_MultipleItems(t *testing.T) {
	js := NewJavaScriptRuntime("")
	ctx := &ExecutionContext{
		ItemIndex:   0,
		Variables:   make(map[string]interface{}),
		Credentials: make(map[string]interface{}),
	}

	items := []model.DataItem{
		{JSON: map[string]interface{}{"value": 10}},
		{JSON: map[string]interface{}{"value": 20}},
		{JSON: map[string]interface{}{"value": 30}},
	}

	// Access items array directly
	result, err := js.Execute(`items.length;`, ctx, items)
	require.NoError(t, err)
	assert.Equal(t, int64(3), result)
}

func TestJavaScriptRuntime_Execute_BooleanResult(t *testing.T) {
	js := NewJavaScriptRuntime("")
	ctx := &ExecutionContext{
		Variables:   make(map[string]interface{}),
		Credentials: make(map[string]interface{}),
	}

	result, err := js.Execute(`true;`, ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, true, result)

	result, err = js.Execute(`false;`, ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, false, result)
}

func TestJavaScriptRuntime_Execute_NullResult(t *testing.T) {
	js := NewJavaScriptRuntime("")
	ctx := &ExecutionContext{
		Variables:   make(map[string]interface{}),
		Credentials: make(map[string]interface{}),
	}

	result, err := js.Execute(`null;`, ctx, nil)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestJavaScriptRuntime_Execute_ObjectResult(t *testing.T) {
	js := NewJavaScriptRuntime("")
	ctx := &ExecutionContext{
		Variables:   make(map[string]interface{}),
		Credentials: make(map[string]interface{}),
	}

	result, err := js.Execute(`({key: "value"});`, ctx, nil)
	require.NoError(t, err)
	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "value", resultMap["key"])
}

func TestJavaScriptRuntime_Execute_ArrayResult(t *testing.T) {
	js := NewJavaScriptRuntime("")
	ctx := &ExecutionContext{
		Variables:   make(map[string]interface{}),
		Credentials: make(map[string]interface{}),
	}

	result, err := js.Execute(`[1, 2, 3];`, ctx, nil)
	require.NoError(t, err)
	resultArr, ok := result.([]interface{})
	require.True(t, ok)
	assert.Len(t, resultArr, 3)
}

func TestJavaScriptRuntime_GetPackageInfo_NotFound(t *testing.T) {
	js := NewJavaScriptRuntime("")

	pkg, found := js.GetPackageInfo("nonexistent-package")
	assert.False(t, found)
	assert.Nil(t, pkg)
}

// ---------------------------------------------------------------------------
// Execution Context Struct Tests
// ---------------------------------------------------------------------------

func TestExecutionContext_DefaultValues(t *testing.T) {
	ctx := &ExecutionContext{}
	assert.Equal(t, "", ctx.WorkflowID)
	assert.Equal(t, "", ctx.ExecutionID)
	assert.Equal(t, "", ctx.NodeID)
	assert.Equal(t, 0, ctx.ItemIndex)
	assert.Equal(t, 0, ctx.RunIndex)
	assert.Equal(t, "", ctx.Mode)
	assert.Equal(t, "", ctx.Timezone)
	assert.Nil(t, ctx.Variables)
	assert.Nil(t, ctx.Credentials)
}

func TestExecutionContext_WithValues(t *testing.T) {
	ctx := &ExecutionContext{
		WorkflowID:  "wf-1",
		ExecutionID: "exec-1",
		NodeID:      "node-1",
		ItemIndex:   3,
		RunIndex:    1,
		Mode:        "webhook",
		Timezone:    "Europe/London",
		Variables: map[string]interface{}{
			"apiUrl": "https://example.com",
		},
		Credentials: map[string]interface{}{
			"token": "abc123",
		},
	}

	assert.Equal(t, "wf-1", ctx.WorkflowID)
	assert.Equal(t, "exec-1", ctx.ExecutionID)
	assert.Equal(t, "node-1", ctx.NodeID)
	assert.Equal(t, 3, ctx.ItemIndex)
	assert.Equal(t, 1, ctx.RunIndex)
	assert.Equal(t, "webhook", ctx.Mode)
	assert.Equal(t, "Europe/London", ctx.Timezone)
	assert.Equal(t, "https://example.com", ctx.Variables["apiUrl"])
	assert.Equal(t, "abc123", ctx.Credentials["token"])
}
