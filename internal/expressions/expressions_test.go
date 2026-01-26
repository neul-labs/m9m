package expressions

import (
	"testing"
	"time"

	"github.com/neul-labs/m9m/internal/model"
)

// TestExpressionCompatibility tests n8n expression compatibility
func TestExpressionCompatibility(t *testing.T) {
	// Note: We create a new evaluator for each subtest to avoid state leakage
	// from the runtime pool between tests

	tests := []struct {
		name       string
		expression string
		context    map[string]interface{}
		expected   interface{}
		shouldFail bool
	}{
		// Basic variable access
		{
			name:       "json_access",
			expression: "{{ $json.name }}",
			context:    map[string]interface{}{"name": "John"},
			expected:   "John",
		},
		{
			name:       "json_nested_access",
			expression: "{{ $json.user.profile.name }}",
			context:    map[string]interface{}{"user": map[string]interface{}{"profile": map[string]interface{}{"name": "Jane"}}},
			expected:   "Jane",
		},

		// String functions
		{
			name:       "uppercase",
			expression: "{{ uppercase($json.name) }}",
			context:    map[string]interface{}{"name": "john"},
			expected:   "JOHN",
		},
		{
			name:       "split",
			expression: "{{ split($json.text, ' ') }}",
			context:    map[string]interface{}{"text": "hello world"},
			expected:   []interface{}{"hello", "world"},
		},
		{
			name:       "join",
			expression: "{{ join(['a', 'b', 'c'], '-') }}",
			context:    map[string]interface{}{},
			expected:   "a-b-c",
		},

		// Math functions
		{
			name:       "add",
			expression: "{{ add(1, 2, 3) }}",
			context:    map[string]interface{}{},
			expected:   6.0,
		},
		{
			name:       "arithmetic",
			expression: "{{ 2 + 3 * 4 }}",
			context:    map[string]interface{}{},
			expected:   14.0,
		},
		{
			name:       "min_max",
			expression: "{{ min(5, 2, 8, 1) }}",
			context:    map[string]interface{}{},
			expected:   1.0,
		},

		// Array functions
		{
			name:       "first",
			expression: "{{ first($json.items) }}",
			context:    map[string]interface{}{"items": []interface{}{1, 2, 3}},
			expected:   1,
		},
		{
			name:       "last",
			expression: "{{ last($json.items) }}",
			context:    map[string]interface{}{"items": []interface{}{1, 2, 3}},
			expected:   3,
		},
		{
			name:       "length",
			expression: "{{ length($json.items) }}",
			context:    map[string]interface{}{"items": []interface{}{1, 2, 3, 4, 5}},
			expected:   5,
		},

		// Conditional expressions
		{
			name:       "ternary",
			expression: "{{ $json.value > 10 ? 'high' : 'low' }}",
			context:    map[string]interface{}{"value": 15},
			expected:   "high",
		},
		{
			name:       "ternary_false",
			expression: "{{ $json.value > 10 ? 'high' : 'low' }}",
			context:    map[string]interface{}{"value": 5},
			expected:   "low",
		},

		// Logic functions
		{
			name:       "if_function",
			expression: "{{ if($json.age >= 18, 'adult', 'minor') }}",
			context:    map[string]interface{}{"age": 25},
			expected:   "adult",
		},
		{
			name:       "isEmpty",
			expression: "{{ isEmpty($json.value) }}",
			context:    map[string]interface{}{"value": ""},
			expected:   true,
		},
		{
			name:       "isNotEmpty",
			expression: "{{ isNotEmpty($json.value) }}",
			context:    map[string]interface{}{"value": "hello"},
			expected:   true,
		},

		// Date functions
		{
			name:       "now",
			expression: "{{ now() }}",
			context:    map[string]interface{}{},
			expected:   "number", // We'll check type rather than exact value
		},

		// Mixed expressions
		{
			name:       "mixed_text_expression",
			expression: "Hello {{ uppercase($json.name) }}!",
			context:    map[string]interface{}{"name": "world"},
			expected:   "Hello WORLD!",
		},

		// Complex nested expressions
		{
			name:       "nested",
			expression: "{{ uppercase(split($json.name, ' ')[0]) }}",
			context:    map[string]interface{}{"name": "john doe"},
			expected:   "JOHN",
		},

		// Error cases
		{
			// Note: Unclosed expressions are treated as literal text, not errors
			name:       "invalid_syntax",
			expression: "{{ $json.name }",
			context:    map[string]interface{}{"name": "test"},
			expected:   "{{ $json.name }", // Returned as-is since not a valid expression
		},
		{
			name:       "undefined_function",
			expression: "{{ undefinedFunction($json.name) }}",
			context:    map[string]interface{}{"name": "test"},
			shouldFail: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a fresh evaluator for each test to avoid runtime pool state leakage
			evaluator := NewGojaExpressionEvaluator(DefaultEvaluatorConfig())

			// Create test context
			context := createTestContext(test.context)

			// Evaluate expression
			result, err := evaluator.EvaluateExpression(test.expression, context)

			if test.shouldFail {
				if err == nil {
					t.Errorf("Expected expression to fail but it succeeded with result: %v", result)
				}
				return
			}

			if err != nil {
				t.Errorf("Expression evaluation failed: %v", err)
				return
			}

			// Special handling for dynamic values like timestamps
			if test.expected == "number" {
				if !isNumeric(result) {
					t.Errorf("Expected number, got %T: %v", result, result)
				}
				return
			}

			// Compare results
			if !deepEqual(result, test.expected) {
				t.Errorf("Expected %v (type %T), got %v (type %T)", test.expected, test.expected, result, result)
			}
		})
	}
}

// TestExpressionParser tests the expression parser
func TestExpressionParser(t *testing.T) {
	parser := NewExpressionParser()

	tests := []struct {
		name           string
		input          string
		expectExpression bool
		expectChunks   int
	}{
		{
			name:           "plain_text",
			input:          "Hello World",
			expectExpression: false,
			expectChunks:   1,
		},
		{
			name:           "simple_expression",
			input:          "={{ $json.name }}",
			expectExpression: true,
			expectChunks:   1,
		},
		{
			name:           "mixed_content",
			input:          "Hello {{ $json.name }}!",
			expectExpression: false,
			expectChunks:   3,
		},
		{
			name:           "multiple_expressions",
			input:          "{{ $json.first }} and {{ $json.second }}",
			expectExpression: false,
			expectChunks:   3,
		},
		{
			name:           "escaped_brackets",
			input:          "Use \\{\\{ and \\}\\} for literal brackets",
			expectExpression: false,
			expectChunks:   1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parsed, err := parser.ParseExpression(test.input)
			if err != nil {
				t.Errorf("Parser failed: %v", err)
				return
			}

			if parsed.IsExpression != test.expectExpression {
				t.Errorf("Expected isExpression=%v, got %v", test.expectExpression, parsed.IsExpression)
			}

			if len(parsed.Chunks) != test.expectChunks {
				t.Errorf("Expected %d chunks, got %d", test.expectChunks, len(parsed.Chunks))
			}
		})
	}
}

// TestBuiltInFunctions tests all built-in functions
func TestBuiltInFunctions(t *testing.T) {
	evaluator := NewGojaExpressionEvaluator(DefaultEvaluatorConfig())

	// Test string functions
	stringTests := []struct {
		expression string
		expected   interface{}
	}{
		{"{{ split('a,b,c', ',') }}", []interface{}{"a", "b", "c"}},
		{"{{ substring('hello', 1, 3) }}", "el"},
		{"{{ replace('hello world', 'world', 'Go') }}", "hello Go"},
		{"{{ trim('  hello  ') }}", "hello"},
		{"{{ toLowerCase('HELLO') }}", "hello"},
		{"{{ toUpperCase('hello') }}", "HELLO"},
		{"{{ base64Encode('hello') }}", "aGVsbG8="},
	}

	for _, test := range stringTests {
		t.Run(test.expression, func(t *testing.T) {
			context := createTestContext(map[string]interface{}{})
			result, err := evaluator.EvaluateExpression(test.expression, context)
			if err != nil {
				t.Errorf("Expression failed: %v", err)
				return
			}
			if !deepEqual(result, test.expected) {
				t.Errorf("Expected %v, got %v", test.expected, result)
			}
		})
	}

	// Test math functions
	mathTests := []struct {
		expression string
		expected   float64
	}{
		{"{{ add(1, 2, 3) }}", 6},
		{"{{ subtract(10, 3) }}", 7},
		{"{{ multiply(2, 3, 4) }}", 24},
		{"{{ divide(15, 3) }}", 5},
		{"{{ round(3.7) }}", 4},
		{"{{ ceil(3.2) }}", 4},
		{"{{ floor(3.8) }}", 3},
		{"{{ abs(-5) }}", 5},
	}

	for _, test := range mathTests {
		t.Run(test.expression, func(t *testing.T) {
			context := createTestContext(map[string]interface{}{})
			result, err := evaluator.EvaluateExpression(test.expression, context)
			if err != nil {
				t.Errorf("Expression failed: %v", err)
				return
			}
			// Compare as float64 to handle both int64 and float64 return types from goja
			resultNum := toFloat64(result)
			if resultNum != test.expected {
				t.Errorf("Expected %v, got %v (type %T)", test.expected, result, result)
			}
		})
	}
}

// toFloat64 converts various numeric types to float64 for comparison
func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int64:
		return float64(n)
	case int32:
		return float64(n)
	case int:
		return float64(n)
	default:
		return 0
	}
}

// TestSecuritySandbox tests that the security sandbox works correctly
func TestSecuritySandbox(t *testing.T) {
	evaluator := NewGojaExpressionEvaluator(DefaultEvaluatorConfig())

	// These expressions should fail due to security restrictions
	dangerousExpressions := []string{
		"{{ eval('malicious code') }}",
		"{{ Function('return process')() }}",
		"{{ setTimeout(function() {}, 1000) }}",
		"{{ fetch('http://evil.com') }}",
		"{{ document.location }}",
		"{{ window.open('http://evil.com') }}",
		"{{ require('fs') }}",
		"{{ process.exit(1) }}",
		"{{ globalThis.eval }}",
		"{{ console.log('test') }}",
	}

	for _, expr := range dangerousExpressions {
		t.Run(expr, func(t *testing.T) {
			context := createTestContext(map[string]interface{}{})
			_, err := evaluator.EvaluateExpression(expr, context)
			if err == nil {
				t.Errorf("Expected dangerous expression to fail, but it succeeded: %s", expr)
			}
		})
	}
}

// TestDataProxy tests the workflow data proxy functionality
func TestDataProxy(t *testing.T) {
	evaluator := NewGojaExpressionEvaluator(DefaultEvaluatorConfig())

	// Create a comprehensive test context
	workflow := &model.Workflow{
		ID:   "test-workflow",
		Name: "Test Workflow",
		Nodes: []model.Node{
			{Name: "node1", Type: "test"},
			{Name: "node2", Type: "test"},
		},
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"name": "John", "age": 30}},
		{JSON: map[string]interface{}{"name": "Jane", "age": 25}},
	}

	runData := &RunExecutionData{
		ResultData: &RunData{
			NodeData: map[string][]NodeExecutionResult{
				"node1": {
					{Data: []model.DataItem{
						{JSON: map[string]interface{}{"result": "test1"}},
					}},
				},
			},
		},
		ExecutionMode: ModeManual,
		StartedAt:     time.Now(),
	}

	context := &ExpressionContext{
		Workflow:            workflow,
		RunExecutionData:    runData,
		RunIndex:           0,
		ItemIndex:          0,
		ActiveNodeName:     "node2",
		ConnectionInputData: inputData,
		Mode:               ModeManual,
	}

	tests := []struct {
		name       string
		expression string
		expected   interface{}
	}{
		{
			name:       "json_access",
			expression: "{{ $json.name }}",
			expected:   "John",
		},
		{
			name:       "workflow_id",
			expression: "{{ $workflow.id }}",
			expected:   "test-workflow",
		},
		{
			name:       "workflow_name",
			expression: "{{ $workflow.name }}",
			expected:   "Test Workflow",
		},
		{
			name:       "input_first",
			expression: "{{ $input.first().name }}",
			expected:   "John",
		},
		{
			name:       "input_last",
			expression: "{{ $input.last().name }}",
			expected:   "Jane",
		},
		{
			name:       "node_access",
			expression: "{{ $node('node1').json.result }}",
			expected:   "test1",
		},
		{
			name:       "execution_mode",
			expression: "{{ $execution.mode }}",
			expected:   "manual",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := evaluator.EvaluateExpression(test.expression, context)
			if err != nil {
				t.Errorf("Expression failed: %v", err)
				return
			}
			if !deepEqual(result, test.expected) {
				t.Errorf("Expected %v, got %v", test.expected, result)
			}
		})
	}
}

// TestPerformance tests expression evaluation performance
func TestPerformance(t *testing.T) {
	evaluator := NewGojaExpressionEvaluator(DefaultEvaluatorConfig())
	context := createTestContext(map[string]interface{}{"name": "test"})

	// Test simple expression performance
	expression := "{{ uppercase($json.name) }}"
	iterations := 1000

	start := time.Now()
	for i := 0; i < iterations; i++ {
		_, err := evaluator.EvaluateExpression(expression, context)
		if err != nil {
			t.Errorf("Expression failed at iteration %d: %v", i, err)
			return
		}
	}
	duration := time.Since(start)

	avgTime := duration / time.Duration(iterations)
	t.Logf("Average expression evaluation time: %v", avgTime)

	// Performance should be under 1ms per expression for simple cases
	if avgTime > time.Millisecond {
		t.Errorf("Expression evaluation too slow: %v per expression", avgTime)
	}
}

// TestCaching tests expression result caching
func TestCaching(t *testing.T) {
	config := DefaultEvaluatorConfig()
	config.EnableCaching = true
	evaluator := NewGojaExpressionEvaluator(config)

	context := createTestContext(map[string]interface{}{"name": "test"})
	expression := "{{ uppercase($json.name) }}"

	// First evaluation (cache miss)
	start := time.Now()
	result1, err := evaluator.EvaluateExpression(expression, context)
	firstDuration := time.Since(start)
	if err != nil {
		t.Errorf("First evaluation failed: %v", err)
		return
	}

	// Second evaluation (cache hit)
	start = time.Now()
	result2, err := evaluator.EvaluateExpression(expression, context)
	secondDuration := time.Since(start)
	if err != nil {
		t.Errorf("Second evaluation failed: %v", err)
		return
	}

	// Results should be identical
	if !deepEqual(result1, result2) {
		t.Errorf("Cached result differs: %v vs %v", result1, result2)
	}

	// Second evaluation should be faster (cached)
	if secondDuration >= firstDuration {
		t.Logf("Warning: Second evaluation not faster (caching may not be working): %v vs %v", firstDuration, secondDuration)
	}

	// Check cache stats
	stats := evaluator.GetStats()
	if cacheStats, ok := stats["cache"].(map[string]interface{}); ok {
		if hits, ok := cacheStats["hits"].(int64); ok && hits == 0 {
			t.Error("Expected cache hits but got 0")
		}
	}
}

// Helper functions

func createTestContext(data map[string]interface{}) *ExpressionContext {
	inputData := []model.DataItem{
		{JSON: data},
	}

	return &ExpressionContext{
		Workflow: &model.Workflow{
			ID:   "test",
			Name: "Test Workflow",
		},
		RunExecutionData: &RunExecutionData{
			ExecutionMode: ModeManual,
			StartedAt:     time.Now(),
		},
		RunIndex:           0,
		ItemIndex:          0,
		ActiveNodeName:     "test-node",
		ConnectionInputData: inputData,
		Mode:               ModeManual,
		AdditionalKeys: &AdditionalKeys{
			ExecutionId:           "test-execution-123",
			RestApiUrl:           "http://localhost:5678/api/v1",
			InstanceBaseUrl:      "http://localhost:5678",
			WebhookBaseUrl:       "http://localhost:5678/webhook",
			WebhookWaitingBaseUrl: "http://localhost:5678/webhook-waiting",
			WebhookTestBaseUrl:   "http://localhost:5678/webhook-test",
		},
	}
}

func deepEqual(a, b interface{}) bool {
	// Simple deep equality check for test purposes
	// In a real implementation, you might use a more robust comparison
	switch va := a.(type) {
	case []interface{}:
		if vb, ok := b.([]interface{}); ok {
			if len(va) != len(vb) {
				return false
			}
			for i := range va {
				if !deepEqual(va[i], vb[i]) {
					return false
				}
			}
			return true
		}
		return false
	case map[string]interface{}:
		if vb, ok := b.(map[string]interface{}); ok {
			if len(va) != len(vb) {
				return false
			}
			for key, valueA := range va {
				if valueB, exists := vb[key]; !exists || !deepEqual(valueA, valueB) {
					return false
				}
			}
			return true
		}
		return false
	default:
		// Handle numeric type comparisons (goja returns int64 for integers)
		if isNumeric(a) && isNumeric(b) {
			return toFloat64(a) == toFloat64(b)
		}
		return a == b
	}
}

// isNumeric checks if a value is a numeric type
func isNumeric(v interface{}) bool {
	switch v.(type) {
	case float64, float32, int64, int32, int:
		return true
	default:
		return false
	}
}