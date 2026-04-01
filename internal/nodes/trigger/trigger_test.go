package trigger

import (
	"testing"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Constructor tests
// ---------------------------------------------------------------------------

// TestNewWebhookNode verifies that the constructor creates a valid node with
// a non-nil BaseNode and no pre-configured HTTP server.
func TestNewWebhookNode(t *testing.T) {
	node := NewWebhookNode()
	require.NotNil(t, node, "NewWebhookNode should return a non-nil node")
	assert.NotNil(t, node.BaseNode, "BaseNode should be embedded")
	assert.Nil(t, node.server, "server should initially be nil")
}

// ---------------------------------------------------------------------------
// Description tests
// ---------------------------------------------------------------------------

// TestWebhookNode_Description verifies that the node description returns the
// correct name, description text, and category.
func TestWebhookNode_Description(t *testing.T) {
	node := NewWebhookNode()
	desc := node.Description()

	assert.Equal(t, "Webhook", desc.Name)
	assert.Equal(t, "Receives HTTP webhook requests", desc.Description)
	assert.Equal(t, "Trigger", desc.Category)
}

// ---------------------------------------------------------------------------
// Interface compliance
// ---------------------------------------------------------------------------

// TestWebhookNode_ImplementsNodeExecutor is a compile-time check that
// WebhookNode satisfies the base.NodeExecutor interface.
func TestWebhookNode_ImplementsNodeExecutor(t *testing.T) {
	var _ base.NodeExecutor = (*WebhookNode)(nil)
}

// ---------------------------------------------------------------------------
// ValidateParameters tests
// ---------------------------------------------------------------------------

func TestWebhookNode_ValidateParameters(t *testing.T) {
	node := NewWebhookNode()

	tests := []struct {
		name      string
		params    map[string]interface{}
		expectErr bool
		errMsg    string
	}{
		{
			name:      "missing path returns error",
			params:    map[string]interface{}{},
			expectErr: true,
			errMsg:    "webhook path is required",
		},
		{
			name:      "nil params returns error",
			params:    nil,
			expectErr: true,
			errMsg:    "webhook path is required",
		},
		{
			name: "valid path only",
			params: map[string]interface{}{
				"path": "/hooks/my-webhook",
			},
			expectErr: false,
		},
		{
			name: "valid path with POST method",
			params: map[string]interface{}{
				"path":       "/hooks/test",
				"httpMethod": "POST",
			},
			expectErr: false,
		},
		{
			name: "valid path with GET method",
			params: map[string]interface{}{
				"path":       "/hooks/test",
				"httpMethod": "GET",
			},
			expectErr: false,
		},
		{
			name: "valid path with PUT method",
			params: map[string]interface{}{
				"path":       "/hooks/test",
				"httpMethod": "PUT",
			},
			expectErr: false,
		},
		{
			name: "valid path with DELETE method",
			params: map[string]interface{}{
				"path":       "/hooks/test",
				"httpMethod": "DELETE",
			},
			expectErr: false,
		},
		{
			name: "valid path with PATCH method",
			params: map[string]interface{}{
				"path":       "/hooks/test",
				"httpMethod": "PATCH",
			},
			expectErr: false,
		},
		{
			name: "valid path with HEAD method",
			params: map[string]interface{}{
				"path":       "/hooks/test",
				"httpMethod": "HEAD",
			},
			expectErr: false,
		},
		{
			name: "valid path with OPTIONS method",
			params: map[string]interface{}{
				"path":       "/hooks/test",
				"httpMethod": "OPTIONS",
			},
			expectErr: false,
		},
		{
			name: "invalid HTTP method returns error",
			params: map[string]interface{}{
				"path":       "/hooks/test",
				"httpMethod": "INVALID",
			},
			expectErr: true,
			errMsg:    "invalid HTTP method: INVALID",
		},
		{
			name: "lowercase method is accepted (validated as uppercase)",
			params: map[string]interface{}{
				"path":       "/hooks/test",
				"httpMethod": "post",
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := node.ValidateParameters(tt.params)
			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Execute tests
// ---------------------------------------------------------------------------

// TestWebhookNode_Execute_EmptyInput verifies that Execute with no input data
// returns an empty slice and no error.
func TestWebhookNode_Execute_EmptyInput(t *testing.T) {
	node := NewWebhookNode()

	params := map[string]interface{}{
		"path":       "/hooks/test",
		"httpMethod": "POST",
	}

	result, err := node.Execute([]model.DataItem{}, params)
	require.NoError(t, err)
	assert.Empty(t, result)
}

// TestWebhookNode_Execute_NilInput verifies that Execute with nil input
// returns an empty slice and no error.
func TestWebhookNode_Execute_NilInput(t *testing.T) {
	node := NewWebhookNode()

	params := map[string]interface{}{
		"path":       "/hooks/test",
		"httpMethod": "POST",
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Empty(t, result)
}

// TestWebhookNode_Execute_BasicRequest verifies that the webhook node
// processes input data and structures it into the webhook response format.
func TestWebhookNode_Execute_BasicRequest(t *testing.T) {
	node := NewWebhookNode()

	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"headers": map[string]interface{}{
					"content-type": "application/json",
				},
				"params": map[string]interface{}{},
				"query": map[string]interface{}{
					"foo": "bar",
				},
				"body": map[string]interface{}{
					"message": "hello",
				},
			},
		},
	}

	params := map[string]interface{}{
		"path":       "/hooks/test",
		"httpMethod": "POST",
	}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, result, 1)

	item := result[0].JSON
	assert.Equal(t, "/hooks/test", item["path"])
	assert.Equal(t, "POST", item["method"])
	assert.Equal(t, map[string]interface{}{"content-type": "application/json"}, item["headers"])
	assert.Equal(t, map[string]interface{}{"foo": "bar"}, item["query"])
	assert.Equal(t, map[string]interface{}{"message": "hello"}, item["body"])
}

// TestWebhookNode_Execute_DefaultMethod verifies that the method defaults to
// POST when not specified in parameters.
func TestWebhookNode_Execute_DefaultMethod(t *testing.T) {
	node := NewWebhookNode()

	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"body": map[string]interface{}{"data": "test"},
			},
		},
	}

	params := map[string]interface{}{
		"path": "/hooks/default-method",
	}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "POST", result[0].JSON["method"], "method should default to POST")
}

// TestWebhookNode_Execute_MultipleItems verifies that the webhook node
// processes multiple input items correctly.
func TestWebhookNode_Execute_MultipleItems(t *testing.T) {
	node := NewWebhookNode()

	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"headers": map[string]interface{}{"x-req": "1"},
				"body":    map[string]interface{}{"id": float64(1)},
			},
		},
		{
			JSON: map[string]interface{}{
				"headers": map[string]interface{}{"x-req": "2"},
				"body":    map[string]interface{}{"id": float64(2)},
			},
		},
	}

	params := map[string]interface{}{
		"path":       "/hooks/multi",
		"httpMethod": "PUT",
	}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, result, 2)

	for i, item := range result {
		assert.Equal(t, "/hooks/multi", item.JSON["path"])
		assert.Equal(t, "PUT", item.JSON["method"])
		assert.NotNil(t, item.JSON["body"], "item %d should have a body", i)
	}
}

// ---------------------------------------------------------------------------
// Authentication tests
// ---------------------------------------------------------------------------

// TestWebhookNode_Execute_NoAuth verifies that the webhook node passes
// through data without errors when authentication is set to "none".
func TestWebhookNode_Execute_NoAuth(t *testing.T) {
	node := NewWebhookNode()

	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"headers": map[string]interface{}{},
				"body":    map[string]interface{}{"data": "test"},
			},
		},
	}

	params := map[string]interface{}{
		"path":           "/hooks/no-auth",
		"httpMethod":     "POST",
		"authentication": "none",
	}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, result, 1)
}

// TestWebhookNode_Execute_BasicAuthValid verifies that the webhook node
// passes data through when a valid Basic Auth header is present.
func TestWebhookNode_Execute_BasicAuthValid(t *testing.T) {
	node := NewWebhookNode()

	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"headers": map[string]interface{}{
					"authorization": "Basic dXNlcjpwYXNz", // user:pass in base64
				},
				"body": map[string]interface{}{"data": "secure"},
			},
		},
	}

	params := map[string]interface{}{
		"path":           "/hooks/basic-auth",
		"httpMethod":     "POST",
		"authentication": "basicAuth",
	}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, result, 1)
}

// TestWebhookNode_Execute_BasicAuthMissingHeader verifies that the webhook
// node returns an error when Basic Auth is configured but no authorization
// header is present.
func TestWebhookNode_Execute_BasicAuthMissingHeader(t *testing.T) {
	node := NewWebhookNode()

	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"headers": map[string]interface{}{},
				"body":    map[string]interface{}{"data": "test"},
			},
		},
	}

	params := map[string]interface{}{
		"path":           "/hooks/basic-auth",
		"httpMethod":     "POST",
		"authentication": "basicAuth",
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook authentication failed")
}

// TestWebhookNode_Execute_BasicAuthNoHeaders verifies that the webhook node
// returns an error when Basic Auth is configured but no headers exist at all.
func TestWebhookNode_Execute_BasicAuthNoHeaders(t *testing.T) {
	node := NewWebhookNode()

	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"body": map[string]interface{}{"data": "test"},
			},
		},
	}

	params := map[string]interface{}{
		"path":           "/hooks/basic-auth",
		"httpMethod":     "POST",
		"authentication": "basicAuth",
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook authentication failed")
}

// TestWebhookNode_Execute_BasicAuthInvalidFormat verifies that the webhook
// node returns an error when the authorization header does not start with
// "Basic ".
func TestWebhookNode_Execute_BasicAuthInvalidFormat(t *testing.T) {
	node := NewWebhookNode()

	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"headers": map[string]interface{}{
					"authorization": "Bearer sometoken",
				},
				"body": map[string]interface{}{"data": "test"},
			},
		},
	}

	params := map[string]interface{}{
		"path":           "/hooks/basic-auth",
		"httpMethod":     "POST",
		"authentication": "basicAuth",
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook authentication failed")
}

// TestWebhookNode_Execute_HeaderAuthValid verifies that the webhook node
// accepts a request when header-based auth is configured and the correct
// header is present.
func TestWebhookNode_Execute_HeaderAuthValid(t *testing.T) {
	node := NewWebhookNode()

	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"headers": map[string]interface{}{
					"x-api-key": "my-secret-key",
				},
				"body": map[string]interface{}{"data": "secure"},
			},
		},
	}

	params := map[string]interface{}{
		"path":           "/hooks/header-auth",
		"httpMethod":     "POST",
		"authentication": "headerAuth",
		"headerName":     "X-API-Key",
		"headerValue":    "my-secret-key",
	}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, result, 1)
}

// TestWebhookNode_Execute_HeaderAuthInvalidValue verifies that the webhook
// node rejects requests when the header value does not match the expected
// value.
func TestWebhookNode_Execute_HeaderAuthInvalidValue(t *testing.T) {
	node := NewWebhookNode()

	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"headers": map[string]interface{}{
					"x-api-key": "wrong-key",
				},
				"body": map[string]interface{}{"data": "test"},
			},
		},
	}

	params := map[string]interface{}{
		"path":           "/hooks/header-auth",
		"httpMethod":     "POST",
		"authentication": "headerAuth",
		"headerName":     "X-API-Key",
		"headerValue":    "correct-key",
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

// TestWebhookNode_Execute_HeaderAuthMissingHeader verifies that the webhook
// node returns an error when header auth is configured but the required
// header is missing from the request.
func TestWebhookNode_Execute_HeaderAuthMissingHeader(t *testing.T) {
	node := NewWebhookNode()

	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"headers": map[string]interface{}{
					"other-header": "some-value",
				},
				"body": map[string]interface{}{"data": "test"},
			},
		},
	}

	params := map[string]interface{}{
		"path":           "/hooks/header-auth",
		"httpMethod":     "POST",
		"authentication": "headerAuth",
		"headerName":     "X-API-Key",
		"headerValue":    "secret",
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook authentication failed")
}

// TestWebhookNode_Execute_HeaderAuthMissingHeaderName verifies that the
// webhook node returns an error when header auth is configured but no
// header name is specified.
func TestWebhookNode_Execute_HeaderAuthMissingHeaderName(t *testing.T) {
	node := NewWebhookNode()

	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"headers": map[string]interface{}{
					"x-api-key": "secret",
				},
				"body": map[string]interface{}{"data": "test"},
			},
		},
	}

	params := map[string]interface{}{
		"path":           "/hooks/header-auth",
		"httpMethod":     "POST",
		"authentication": "headerAuth",
		// headerName intentionally omitted
		"headerValue": "secret",
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook authentication failed")
}

// ---------------------------------------------------------------------------
// GetWebhookInfo tests
// ---------------------------------------------------------------------------

// TestWebhookNode_GetWebhookInfo verifies that GetWebhookInfo returns the
// correct webhook configuration.
func TestWebhookNode_GetWebhookInfo(t *testing.T) {
	node := NewWebhookNode()

	tests := []struct {
		name           string
		params         map[string]interface{}
		expectedPath   string
		expectedMethod string
	}{
		{
			name: "returns configured path and method",
			params: map[string]interface{}{
				"path":       "/hooks/info-test",
				"httpMethod": "GET",
			},
			expectedPath:   "/hooks/info-test",
			expectedMethod: "GET",
		},
		{
			name: "defaults method to POST when not specified",
			params: map[string]interface{}{
				"path": "/hooks/default",
			},
			expectedPath:   "/hooks/default",
			expectedMethod: "POST",
		},
		{
			name: "uppercases the method",
			params: map[string]interface{}{
				"path":       "/hooks/upper",
				"httpMethod": "put",
			},
			expectedPath:   "/hooks/upper",
			expectedMethod: "PUT",
		},
		{
			name:           "handles empty params",
			params:         map[string]interface{}{},
			expectedPath:   "",
			expectedMethod: "POST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := node.GetWebhookInfo(tt.params)
			assert.Equal(t, tt.expectedPath, info["path"])
			assert.Equal(t, tt.expectedMethod, info["method"])
			assert.Equal(t, "webhook", info["type"])
		})
	}
}
