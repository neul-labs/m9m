package ai

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------
// Construction and metadata tests
// ---------------------------------------------------------------

func TestNewAnthropicNode(t *testing.T) {
	node := NewAnthropicNode()
	require.NotNil(t, node, "NewAnthropicNode should return a non-nil node")
	require.NotNil(t, node.httpClient, "httpClient should be initialised")
}

func TestAnthropicNode_ImplementsNodeExecutor(t *testing.T) {
	// Compile-time check that AnthropicNode satisfies the NodeExecutor interface.
	var _ base.NodeExecutor = (*AnthropicNode)(nil)
}

func TestAnthropicNode_Description(t *testing.T) {
	node := NewAnthropicNode()
	desc := node.Description()

	assert.Equal(t, "Anthropic", desc.Name)
	assert.Equal(t, "ai", desc.Category)
	assert.Contains(t, desc.Description, "Claude", "Description should mention Claude")
}

// ---------------------------------------------------------------
// ValidateParameters tests (inherits from BaseNode)
// ---------------------------------------------------------------

func TestAnthropicNode_ValidateParameters(t *testing.T) {
	node := NewAnthropicNode()

	tests := []struct {
		name      string
		params    map[string]interface{}
		expectErr bool
	}{
		{
			name:      "nil params returns no error (base implementation)",
			params:    nil,
			expectErr: false,
		},
		{
			name:      "empty params returns no error",
			params:    map[string]interface{}{},
			expectErr: false,
		},
		{
			name: "valid params returns no error",
			params: map[string]interface{}{
				"apiKey": "sk-ant-test",
				"prompt": "Hello",
				"model":  "claude-3-5-sonnet-20241022",
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := node.ValidateParameters(tt.params)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------
// Execute – parameter validation (no HTTP call needed)
// ---------------------------------------------------------------

func TestAnthropicNode_Execute_MissingAPIKey(t *testing.T) {
	node := NewAnthropicNode()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"input": "test"}},
	}

	params := map[string]interface{}{
		"prompt": "Say hello",
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apiKey is required")
}

func TestAnthropicNode_Execute_MissingPrompt(t *testing.T) {
	node := NewAnthropicNode()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"input": "test"}},
	}

	params := map[string]interface{}{
		"apiKey": "sk-ant-test-key",
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prompt is required")
}

func TestAnthropicNode_Execute_EmptyInputData(t *testing.T) {
	node := NewAnthropicNode()

	params := map[string]interface{}{
		"apiKey": "sk-ant-test-key",
		"prompt": "Hello",
	}

	result, err := node.Execute([]model.DataItem{}, params)
	require.NoError(t, err)
	assert.Empty(t, result, "Empty input should produce empty output")
}

func TestAnthropicNode_Execute_BothAPIKeyAndPromptMissing(t *testing.T) {
	node := NewAnthropicNode()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	// apiKey is checked first in the implementation
	_, err := node.Execute(inputData, map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apiKey is required")
}

// ---------------------------------------------------------------
// Execute – request construction tests using httptest
// ---------------------------------------------------------------

func TestAnthropicNode_Execute_RequestConstruction(t *testing.T) {
	var capturedReq *http.Request
	var capturedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusInternalServerError)
			return
		}
		json.Unmarshal(body, &capturedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "msg_123",
			"type": "message",
			"role": "assistant",
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Hello there!",
				},
			},
			"stop_reason": "end_turn",
			"usage": map[string]interface{}{
				"input_tokens":  10,
				"output_tokens": 5,
			},
		})
	}))
	defer server.Close()

	node := NewAnthropicNode()
	node.httpClient = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = "http"
			req.URL.Host = server.Listener.Addr().String()
			return http.DefaultTransport.RoundTrip(req)
		}),
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"user": "bob"}},
	}

	params := map[string]interface{}{
		"apiKey":      "sk-ant-test-key-456",
		"prompt":      "Hello, Claude!",
		"model":       "claude-3-5-sonnet-20241022",
		"maxTokens":   2048,
		"temperature": 0.3,
	}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)

	// Verify request method and headers
	assert.Equal(t, "POST", capturedReq.Method)
	assert.Equal(t, "application/json", capturedReq.Header.Get("Content-Type"))
	assert.Equal(t, "sk-ant-test-key-456", capturedReq.Header.Get("x-api-key"))
	assert.Equal(t, "2023-06-01", capturedReq.Header.Get("anthropic-version"))

	// Verify request body
	require.NotNil(t, capturedBody)
	assert.Equal(t, "claude-3-5-sonnet-20241022", capturedBody["model"])
	assert.InDelta(t, 0.3, capturedBody["temperature"], 0.001)
	assert.InDelta(t, 2048, capturedBody["max_tokens"], 0.001)

	messages, ok := capturedBody["messages"].([]interface{})
	require.True(t, ok)
	require.Len(t, messages, 1)

	msg := messages[0].(map[string]interface{})
	assert.Equal(t, "user", msg["role"])
	assert.Equal(t, "Hello, Claude!", msg["content"])

	// Verify result
	require.Len(t, result, 1)
	assert.Equal(t, "Hello, Claude!", result[0].JSON["prompt"])
	assert.Equal(t, "Hello there!", result[0].JSON["response"])
	assert.Equal(t, "claude-3-5-sonnet-20241022", result[0].JSON["model"])
	assert.Equal(t, "end_turn", result[0].JSON["stopReason"])
	assert.NotNil(t, result[0].JSON["usage"])
}

func TestAnthropicNode_Execute_DefaultParameters(t *testing.T) {
	var capturedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Hi",
				},
			},
			"stop_reason": "end_turn",
			"usage":       map[string]interface{}{},
		})
	}))
	defer server.Close()

	node := NewAnthropicNode()
	node.httpClient = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = "http"
			req.URL.Host = server.Listener.Addr().String()
			return http.DefaultTransport.RoundTrip(req)
		}),
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"apiKey": "sk-ant-key",
		"prompt": "Test",
	}

	_, err := node.Execute(inputData, params)
	require.NoError(t, err)

	// Verify defaults were applied
	assert.Equal(t, "claude-3-5-sonnet-20241022", capturedBody["model"])
	assert.InDelta(t, 1024, capturedBody["max_tokens"], 0.001)
	assert.InDelta(t, 1.0, capturedBody["temperature"], 0.001)
}

// ---------------------------------------------------------------
// Execute – multiple input items
// ---------------------------------------------------------------

func TestAnthropicNode_Execute_MultipleInputItems(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Response",
				},
			},
			"stop_reason": "end_turn",
			"usage":       map[string]interface{}{},
		})
	}))
	defer server.Close()

	node := NewAnthropicNode()
	node.httpClient = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = "http"
			req.URL.Host = server.Listener.Addr().String()
			return http.DefaultTransport.RoundTrip(req)
		}),
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"index": 1}},
		{JSON: map[string]interface{}{"index": 2}},
		{JSON: map[string]interface{}{"index": 3}},
	}

	params := map[string]interface{}{
		"apiKey": "sk-ant-key",
		"prompt": "Summarize",
	}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	assert.Len(t, result, 3, "Should produce one result per input item")
	assert.Equal(t, 3, callCount, "Should make one API call per input item")

	for _, item := range result {
		assert.Equal(t, "Response", item.JSON["response"])
	}
}

// ---------------------------------------------------------------
// Execute – error responses
// ---------------------------------------------------------------

func TestAnthropicNode_Execute_APIErrorStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"unauthorized", http.StatusUnauthorized},
		{"rate limited", http.StatusTooManyRequests},
		{"server error", http.StatusInternalServerError},
		{"bad request", http.StatusBadRequest},
		{"forbidden", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"type": "error",
					"error": map[string]interface{}{
						"type":    "invalid_request_error",
						"message": "error occurred",
					},
				})
			}))
			defer server.Close()

			node := NewAnthropicNode()
			node.httpClient = &http.Client{
				Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
					req.URL.Scheme = "http"
					req.URL.Host = server.Listener.Addr().String()
					return http.DefaultTransport.RoundTrip(req)
				}),
			}

			inputData := []model.DataItem{
				{JSON: map[string]interface{}{}},
			}

			params := map[string]interface{}{
				"apiKey": "sk-ant-key",
				"prompt": "Hello",
			}

			_, err := node.Execute(inputData, params)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "anthropic API returned status")
		})
	}
}

func TestAnthropicNode_Execute_EmptyContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []interface{}{},
		})
	}))
	defer server.Close()

	node := NewAnthropicNode()
	node.httpClient = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = "http"
			req.URL.Host = server.Listener.Addr().String()
			return http.DefaultTransport.RoundTrip(req)
		}),
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"apiKey": "sk-ant-key",
		"prompt": "Hello",
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no content in Anthropic response")
}

func TestAnthropicNode_Execute_MissingContentField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "msg_123",
			"type": "message",
		})
	}))
	defer server.Close()

	node := NewAnthropicNode()
	node.httpClient = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = "http"
			req.URL.Host = server.Listener.Addr().String()
			return http.DefaultTransport.RoundTrip(req)
		}),
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"apiKey": "sk-ant-key",
		"prompt": "Hello",
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no content in Anthropic response")
}

func TestAnthropicNode_Execute_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{not valid json`))
	}))
	defer server.Close()

	node := NewAnthropicNode()
	node.httpClient = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = "http"
			req.URL.Host = server.Listener.Addr().String()
			return http.DefaultTransport.RoundTrip(req)
		}),
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"apiKey": "sk-ant-key",
		"prompt": "Hello",
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
}

// ---------------------------------------------------------------
// Execute – successful response with full usage data
// ---------------------------------------------------------------

func TestAnthropicNode_Execute_SuccessfulResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "msg_01XFDUDYJgAACzvnptvVoYEL",
			"type": "message",
			"role": "assistant",
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Paris is the capital of France.",
				},
			},
			"model":       "claude-3-5-sonnet-20241022",
			"stop_reason": "end_turn",
			"usage": map[string]interface{}{
				"input_tokens":  12,
				"output_tokens": 8,
			},
		})
	}))
	defer server.Close()

	node := NewAnthropicNode()
	node.httpClient = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = "http"
			req.URL.Host = server.Listener.Addr().String()
			return http.DefaultTransport.RoundTrip(req)
		}),
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"question": "What is the capital of France?"}},
	}

	params := map[string]interface{}{
		"apiKey":      "sk-ant-test-key-xyz",
		"prompt":      "What is the capital of France?",
		"model":       "claude-3-5-sonnet-20241022",
		"maxTokens":   100,
		"temperature": 0.0,
	}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, result, 1)

	item := result[0]
	assert.Equal(t, "What is the capital of France?", item.JSON["prompt"])
	assert.Equal(t, "Paris is the capital of France.", item.JSON["response"])
	assert.Equal(t, "claude-3-5-sonnet-20241022", item.JSON["model"])
	assert.Equal(t, "end_turn", item.JSON["stopReason"])

	usage, ok := item.JSON["usage"].(map[string]interface{})
	require.True(t, ok)
	assert.InDelta(t, 12, usage["input_tokens"], 0.001)
	assert.InDelta(t, 8, usage["output_tokens"], 0.001)
}

// ---------------------------------------------------------------
// Execute – custom temperature override
// ---------------------------------------------------------------

func TestAnthropicNode_Execute_CustomTemperature(t *testing.T) {
	var capturedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Creative response",
				},
			},
			"stop_reason": "end_turn",
			"usage":       map[string]interface{}{},
		})
	}))
	defer server.Close()

	node := NewAnthropicNode()
	node.httpClient = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = "http"
			req.URL.Host = server.Listener.Addr().String()
			return http.DefaultTransport.RoundTrip(req)
		}),
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"apiKey":      "sk-ant-key",
		"prompt":      "Be creative",
		"temperature": 0.95,
	}

	_, err := node.Execute(inputData, params)
	require.NoError(t, err)
	assert.InDelta(t, 0.95, capturedBody["temperature"], 0.001)
}
