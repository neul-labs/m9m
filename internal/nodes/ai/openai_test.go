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

// roundTripperFunc is an adapter to allow the use of ordinary functions as http.RoundTripper.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// ---------------------------------------------------------------
// Construction and metadata tests
// ---------------------------------------------------------------

func TestNewOpenAINode(t *testing.T) {
	node := NewOpenAINode()
	require.NotNil(t, node, "NewOpenAINode should return a non-nil node")
	require.NotNil(t, node.httpClient, "httpClient should be initialised")
}

func TestOpenAINode_ImplementsNodeExecutor(t *testing.T) {
	// Compile-time check that OpenAINode satisfies the NodeExecutor interface.
	var _ base.NodeExecutor = (*OpenAINode)(nil)
}

func TestOpenAINode_Description(t *testing.T) {
	node := NewOpenAINode()
	desc := node.Description()

	assert.Equal(t, "OpenAI", desc.Name)
	assert.Equal(t, "ai", desc.Category)
	assert.NotEmpty(t, desc.Description, "Description should not be empty")
}

// ---------------------------------------------------------------
// ValidateParameters tests (inherits from BaseNode)
// ---------------------------------------------------------------

func TestOpenAINode_ValidateParameters(t *testing.T) {
	node := NewOpenAINode()

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
				"apiKey": "sk-test",
				"prompt": "Hello",
				"model":  "gpt-4",
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

func TestOpenAINode_Execute_MissingAPIKey(t *testing.T) {
	node := NewOpenAINode()

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

func TestOpenAINode_Execute_MissingPrompt(t *testing.T) {
	node := NewOpenAINode()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"input": "test"}},
	}

	params := map[string]interface{}{
		"apiKey": "sk-test-key",
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prompt is required")
}

func TestOpenAINode_Execute_EmptyInputData(t *testing.T) {
	node := NewOpenAINode()

	params := map[string]interface{}{
		"apiKey": "sk-test-key",
		"prompt": "Hello",
	}

	result, err := node.Execute([]model.DataItem{}, params)
	require.NoError(t, err)
	assert.Empty(t, result, "Empty input should produce empty output")
}

// ---------------------------------------------------------------
// Execute – request construction tests using httptest
// ---------------------------------------------------------------

func TestOpenAINode_Execute_RequestConstruction(t *testing.T) {
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
			"choices": []interface{}{
				map[string]interface{}{
					"message": map[string]interface{}{
						"content": "Hello!",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"total_tokens":      15,
			},
		})
	}))
	defer server.Close()

	node := NewOpenAINode()
	// Override the httpClient to use the test server's transport so requests
	// go to our mock. We also need to redirect the URL. Since the node hard-codes
	// the URL, we use a custom transport that redirects.
	node.httpClient = server.Client()
	originalTransport := node.httpClient.Transport
	node.httpClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		// Redirect the request to our test server
		req.URL.Scheme = "http"
		req.URL.Host = server.Listener.Addr().String()
		if originalTransport != nil {
			return originalTransport.RoundTrip(req)
		}
		return http.DefaultTransport.RoundTrip(req)
	})

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"user": "alice"}},
	}

	params := map[string]interface{}{
		"apiKey":      "sk-test-key-123",
		"prompt":      "Hello, world!",
		"model":       "gpt-4",
		"maxTokens":   2048,
		"temperature": 0.5,
	}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)

	// Verify request method and headers
	assert.Equal(t, "POST", capturedReq.Method)
	assert.Equal(t, "application/json", capturedReq.Header.Get("Content-Type"))
	assert.Equal(t, "Bearer sk-test-key-123", capturedReq.Header.Get("Authorization"))

	// Verify request body
	require.NotNil(t, capturedBody)
	assert.Equal(t, "gpt-4", capturedBody["model"])
	assert.InDelta(t, 0.5, capturedBody["temperature"], 0.001)

	// max_tokens comes through as float64 from JSON unmarshal
	assert.InDelta(t, 2048, capturedBody["max_tokens"], 0.001)

	messages, ok := capturedBody["messages"].([]interface{})
	require.True(t, ok)
	require.Len(t, messages, 1)

	msg := messages[0].(map[string]interface{})
	assert.Equal(t, "user", msg["role"])
	assert.Equal(t, "Hello, world!", msg["content"])

	// Verify result
	require.Len(t, result, 1)
	assert.Equal(t, "Hello, world!", result[0].JSON["prompt"])
	assert.Equal(t, "Hello!", result[0].JSON["response"])
	assert.Equal(t, "gpt-4", result[0].JSON["model"])
	assert.Equal(t, "stop", result[0].JSON["finishReason"])
	assert.NotNil(t, result[0].JSON["usage"])
}

func TestOpenAINode_Execute_DefaultParameters(t *testing.T) {
	var capturedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"message":       map[string]interface{}{"content": "Hi"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{},
		})
	}))
	defer server.Close()

	node := NewOpenAINode()
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

	// Only required parameters; model and maxTokens should get defaults
	params := map[string]interface{}{
		"apiKey": "sk-key",
		"prompt": "Test",
	}

	_, err := node.Execute(inputData, params)
	require.NoError(t, err)

	// Verify defaults were applied
	assert.Equal(t, "gpt-3.5-turbo", capturedBody["model"])
	assert.InDelta(t, 1000, capturedBody["max_tokens"], 0.001)
	assert.InDelta(t, 0.7, capturedBody["temperature"], 0.001)
}

// ---------------------------------------------------------------
// Execute – multiple input items
// ---------------------------------------------------------------

func TestOpenAINode_Execute_MultipleInputItems(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"message":       map[string]interface{}{"content": "Reply"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{},
		})
	}))
	defer server.Close()

	node := NewOpenAINode()
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
		"apiKey": "sk-key",
		"prompt": "Summarize",
	}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	assert.Len(t, result, 3, "Should produce one result per input item")
	assert.Equal(t, 3, callCount, "Should make one API call per input item")

	for _, item := range result {
		assert.Equal(t, "Reply", item.JSON["response"])
	}
}

// ---------------------------------------------------------------
// Execute – error responses
// ---------------------------------------------------------------

func TestOpenAINode_Execute_APIErrorStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"unauthorized", http.StatusUnauthorized},
		{"rate limited", http.StatusTooManyRequests},
		{"server error", http.StatusInternalServerError},
		{"bad request", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]interface{}{
						"message": "error occurred",
					},
				})
			}))
			defer server.Close()

			node := NewOpenAINode()
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
				"apiKey": "sk-key",
				"prompt": "Hello",
			}

			_, err := node.Execute(inputData, params)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "openai API returned status")
		})
	}
}

func TestOpenAINode_Execute_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []interface{}{},
		})
	}))
	defer server.Close()

	node := NewOpenAINode()
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
		"apiKey": "sk-key",
		"prompt": "Hello",
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no choices in OpenAI response")
}

func TestOpenAINode_Execute_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{not valid json`))
	}))
	defer server.Close()

	node := NewOpenAINode()
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
		"apiKey": "sk-key",
		"prompt": "Hello",
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestOpenAINode_Execute_MissingChoicesField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "chatcmpl-123",
		})
	}))
	defer server.Close()

	node := NewOpenAINode()
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
		"apiKey": "sk-key",
		"prompt": "Hello",
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no choices in OpenAI response")
}

// ---------------------------------------------------------------
// Execute – successful response with usage data
// ---------------------------------------------------------------

func TestOpenAINode_Execute_SuccessfulResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     "chatcmpl-abc123",
			"object": "chat.completion",
			"choices": []interface{}{
				map[string]interface{}{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "The capital of France is Paris.",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     15,
				"completion_tokens": 8,
				"total_tokens":      23,
			},
		})
	}))
	defer server.Close()

	node := NewOpenAINode()
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
		"apiKey":      "sk-test-key-xyz",
		"prompt":      "What is the capital of France?",
		"model":       "gpt-4",
		"maxTokens":   100,
		"temperature": 0.0,
	}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, result, 1)

	item := result[0]
	assert.Equal(t, "What is the capital of France?", item.JSON["prompt"])
	assert.Equal(t, "The capital of France is Paris.", item.JSON["response"])
	assert.Equal(t, "gpt-4", item.JSON["model"])
	assert.Equal(t, "stop", item.JSON["finishReason"])

	usage, ok := item.JSON["usage"].(map[string]interface{})
	require.True(t, ok)
	assert.InDelta(t, 23, usage["total_tokens"], 0.001)
}
