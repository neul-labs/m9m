package productivity

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Description tests ---

func TestGoogleSheetsNode_Description(t *testing.T) {
	node := NewGoogleSheetsNode()
	desc := node.Description()

	assert.Equal(t, "Google Sheets", desc.Name)
	assert.Equal(t, "Read, write, and manipulate Google Sheets data", desc.Description)
	assert.Equal(t, "productivity", desc.Category)
}

func TestGoogleSheetsNode_ImplementsNodeExecutor(t *testing.T) {
	var _ base.NodeExecutor = (*GoogleSheetsNode)(nil)
}

// --- ValidateParameters tests (inherited BaseNode) ---

func TestGoogleSheetsNode_ValidateParameters(t *testing.T) {
	node := NewGoogleSheetsNode()

	tests := []struct {
		name      string
		params    map[string]interface{}
		expectErr bool
	}{
		{
			name:      "nil params accepted by base validation",
			params:    nil,
			expectErr: false,
		},
		{
			name:      "empty params accepted by base validation",
			params:    map[string]interface{}{},
			expectErr: false,
		},
		{
			name: "valid params accepted",
			params: map[string]interface{}{
				"resource":      "sheet",
				"operation":     "read",
				"spreadsheetId": "abc123",
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

// --- Execute: credential handling ---

func TestGoogleSheetsNode_Execute_NoCredentials(t *testing.T) {
	node := NewGoogleSheetsNode()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"key": "value"}},
	}
	params := map[string]interface{}{
		"resource":      "sheet",
		"operation":     "read",
		"spreadsheetId": "abc123",
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no valid credentials provided")
}

func TestGoogleSheetsNode_Execute_EmptyCredentialsMap(t *testing.T) {
	node := NewGoogleSheetsNode()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}
	params := map[string]interface{}{
		"resource":    "sheet",
		"operation":   "read",
		"credentials": map[string]interface{}{},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no valid credentials provided")
}

func TestGoogleSheetsNode_Execute_OauthTokenUsed(t *testing.T) {
	node := NewGoogleSheetsNode()

	// The node will try to call the Google API, which will fail; but we verify it
	// gets past the credential check.
	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"spreadsheetId": "test-id"}},
	}
	params := map[string]interface{}{
		"resource":  "sheet",
		"operation": "read",
		"credentials": map[string]interface{}{
			"oauthToken": "test-oauth-token",
		},
		"spreadsheetId": "test-id",
	}

	// This will fail because it actually tries to call Google Sheets API,
	// but it should NOT fail with "no valid credentials provided".
	_, err := node.Execute(inputData, params)
	if err != nil {
		assert.NotContains(t, err.Error(), "no valid credentials provided")
	}
}

func TestGoogleSheetsNode_Execute_AccessTokenUsed(t *testing.T) {
	node := NewGoogleSheetsNode()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}
	params := map[string]interface{}{
		"resource":  "sheet",
		"operation": "read",
		"credentials": map[string]interface{}{
			"accessToken": "test-access-token",
		},
		"spreadsheetId": "test-id",
	}

	_, err := node.Execute(inputData, params)
	if err != nil {
		assert.NotContains(t, err.Error(), "no valid credentials provided")
	}
}

// --- Execute: unsupported resource ---

func TestGoogleSheetsNode_Execute_UnsupportedResource(t *testing.T) {
	node := NewGoogleSheetsNode()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}
	params := map[string]interface{}{
		"resource":  "calendar",
		"operation": "read",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported resource: calendar")
}

// --- Execute: unsupported sheet operation ---

func TestGoogleSheetsNode_Execute_UnsupportedSheetOperation(t *testing.T) {
	node := NewGoogleSheetsNode()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}
	params := map[string]interface{}{
		"resource":      "sheet",
		"operation":     "delete",
		"spreadsheetId": "test-id",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported sheet operation: delete")
}

// --- Execute: unsupported spreadsheet operation ---

func TestGoogleSheetsNode_Execute_UnsupportedSpreadsheetOperation(t *testing.T) {
	node := NewGoogleSheetsNode()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}
	params := map[string]interface{}{
		"resource":  "spreadsheet",
		"operation": "delete",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported spreadsheet operation: delete")
}

// --- Execute: missing spreadsheetId ---

func TestGoogleSheetsNode_Execute_MissingSpreadsheetId(t *testing.T) {
	node := NewGoogleSheetsNode()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}
	params := map[string]interface{}{
		"resource":  "sheet",
		"operation": "read",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
		// No spreadsheetId provided anywhere
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spreadsheetId is required")
}

func TestGoogleSheetsNode_Execute_SpreadsheetIdFromInputData(t *testing.T) {
	node := NewGoogleSheetsNode()

	// spreadsheetId comes from the input data item's JSON, not from params
	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"spreadsheetId": "from-input"}},
	}
	params := map[string]interface{}{
		"resource":  "sheet",
		"operation": "read",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
		// No spreadsheetId in params
	}

	// Will fail on the HTTP call but should get past spreadsheetId validation
	_, err := node.Execute(inputData, params)
	if err != nil {
		assert.NotContains(t, err.Error(), "spreadsheetId is required")
	}
}

// --- Execute: spreadsheet get missing id ---

func TestGoogleSheetsNode_Execute_SpreadsheetGetMissingId(t *testing.T) {
	node := NewGoogleSheetsNode()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}
	params := map[string]interface{}{
		"resource":  "spreadsheet",
		"operation": "get",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "spreadsheetId is required")
}

// --- Execute: default values for resource and operation ---

func TestGoogleSheetsNode_Execute_DefaultResourceAndOperation(t *testing.T) {
	node := NewGoogleSheetsNode()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}
	// Not providing resource or operation - they should default to "sheet" and "read"
	params := map[string]interface{}{
		"spreadsheetId": "test-id",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	// Will fail on the actual API call, but should default to sheet/read and
	// not fail with "unsupported resource" or "unsupported operation"
	_, err := node.Execute(inputData, params)
	if err != nil {
		assert.NotContains(t, err.Error(), "unsupported resource")
		assert.NotContains(t, err.Error(), "unsupported operation")
	}
}

// --- Execute: empty input data ---

func TestGoogleSheetsNode_Execute_EmptyInputData(t *testing.T) {
	node := NewGoogleSheetsNode()

	params := map[string]interface{}{
		"resource":      "sheet",
		"operation":     "read",
		"spreadsheetId": "test-id",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	result, err := node.Execute([]model.DataItem{}, params)
	require.NoError(t, err)
	assert.Empty(t, result)
}

// --- Execute with httptest mock server: read operation ---

func TestGoogleSheetsNode_Execute_ReadWithMockServer(t *testing.T) {
	responseBody := map[string]interface{}{
		"range":          "Sheet1!A1:Z1000",
		"majorDimension": "ROWS",
		"values": []interface{}{
			[]interface{}{"Name", "Age"},
			[]interface{}{"Alice", "30"},
			[]interface{}{"Bob", "25"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer test-token")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(responseBody)
	}))
	defer server.Close()

	node := NewGoogleSheetsNode()
	// Override the httpClient to point to our test server.
	// We use a custom transport to redirect requests to the test server.
	node.httpClient = server.Client()
	// We need to actually intercept the URL. Since the node builds URLs with
	// the Google API domain, we use a RoundTripper to redirect.
	node.httpClient.Transport = &rewriteTransport{
		base:    http.DefaultTransport,
		baseURL: server.URL,
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}
	params := map[string]interface{}{
		"resource":      "sheet",
		"operation":     "read",
		"spreadsheetId": "test-spreadsheet-id",
		"range":         "Sheet1!A1:Z1000",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "Sheet1!A1:Z1000", result[0].JSON["range"])
	assert.Equal(t, "ROWS", result[0].JSON["majorDimension"])
}

// --- Execute with httptest mock server: append operation ---

func TestGoogleSheetsNode_Execute_AppendWithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.String(), ":append")
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer test-token")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Decode the request body to verify it has values
		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		require.NoError(t, err)
		assert.NotNil(t, body["values"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"spreadsheetId": "test-id",
			"updates": map[string]interface{}{
				"updatedRows":    1,
				"updatedColumns": 2,
			},
		})
	}))
	defer server.Close()

	node := NewGoogleSheetsNode()
	node.httpClient = server.Client()
	node.httpClient.Transport = &rewriteTransport{
		base:    http.DefaultTransport,
		baseURL: server.URL,
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"name": "Charlie", "age": 35}},
	}
	params := map[string]interface{}{
		"resource":      "sheet",
		"operation":     "append",
		"spreadsheetId": "test-id",
		"range":         "Sheet1!A:B",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "test-id", result[0].JSON["spreadsheetId"])
}

// --- Execute with httptest mock server: update operation ---

func TestGoogleSheetsNode_Execute_UpdateWithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer test-token")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"updatedRange": "Sheet1!A1:B1",
			"updatedRows":  1,
		})
	}))
	defer server.Close()

	node := NewGoogleSheetsNode()
	node.httpClient = server.Client()
	node.httpClient.Transport = &rewriteTransport{
		base:    http.DefaultTransport,
		baseURL: server.URL,
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"name": "Updated"}},
	}
	params := map[string]interface{}{
		"resource":      "sheet",
		"operation":     "update",
		"spreadsheetId": "test-id",
		"range":         "Sheet1!A1:B1",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "Sheet1!A1:B1", result[0].JSON["updatedRange"])
}

// --- Execute with httptest mock server: clear operation ---

func TestGoogleSheetsNode_Execute_ClearWithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.String(), ":clear")
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer test-token")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"clearedRange":  "Sheet1!A:Z",
			"spreadsheetId": "test-id",
		})
	}))
	defer server.Close()

	node := NewGoogleSheetsNode()
	node.httpClient = server.Client()
	node.httpClient.Transport = &rewriteTransport{
		base:    http.DefaultTransport,
		baseURL: server.URL,
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}
	params := map[string]interface{}{
		"resource":      "sheet",
		"operation":     "clear",
		"spreadsheetId": "test-id",
		"range":         "Sheet1!A:Z",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "Sheet1!A:Z", result[0].JSON["clearedRange"])
}

// --- Execute with httptest mock server: create spreadsheet ---

func TestGoogleSheetsNode_Execute_CreateSpreadsheetWithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer test-token")

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		require.NoError(t, err)
		props := body["properties"].(map[string]interface{})
		assert.Equal(t, "My Spreadsheet", props["title"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"spreadsheetId": "new-spreadsheet-id",
			"properties": map[string]interface{}{
				"title": "My Spreadsheet",
			},
		})
	}))
	defer server.Close()

	node := NewGoogleSheetsNode()
	node.httpClient = server.Client()
	node.httpClient.Transport = &rewriteTransport{
		base:    http.DefaultTransport,
		baseURL: server.URL,
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}
	params := map[string]interface{}{
		"resource":  "spreadsheet",
		"operation": "create",
		"title":     "My Spreadsheet",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "new-spreadsheet-id", result[0].JSON["spreadsheetId"])
}

// --- Execute with httptest mock server: create spreadsheet default title ---

func TestGoogleSheetsNode_Execute_CreateSpreadsheetDefaultTitle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		props := body["properties"].(map[string]interface{})
		assert.Equal(t, "New Spreadsheet", props["title"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"spreadsheetId": "default-title-id",
		})
	}))
	defer server.Close()

	node := NewGoogleSheetsNode()
	node.httpClient = server.Client()
	node.httpClient.Transport = &rewriteTransport{
		base:    http.DefaultTransport,
		baseURL: server.URL,
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}
	params := map[string]interface{}{
		"resource":  "spreadsheet",
		"operation": "create",
		// No title provided - should default to "New Spreadsheet"
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, result, 1)
}

// --- Execute with httptest mock server: get spreadsheet ---

func TestGoogleSheetsNode_Execute_GetSpreadsheetWithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"spreadsheetId": "get-test-id",
			"properties": map[string]interface{}{
				"title": "Test Sheet",
			},
		})
	}))
	defer server.Close()

	node := NewGoogleSheetsNode()
	node.httpClient = server.Client()
	node.httpClient.Transport = &rewriteTransport{
		base:    http.DefaultTransport,
		baseURL: server.URL,
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}
	params := map[string]interface{}{
		"resource":      "spreadsheet",
		"operation":     "get",
		"spreadsheetId": "get-test-id",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "get-test-id", result[0].JSON["spreadsheetId"])
}

// --- Execute: API error response ---

func TestGoogleSheetsNode_Execute_APIErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    403,
				"message": "The caller does not have permission",
				"status":  "PERMISSION_DENIED",
			},
		})
	}))
	defer server.Close()

	node := NewGoogleSheetsNode()
	node.httpClient = server.Client()
	node.httpClient.Transport = &rewriteTransport{
		base:    http.DefaultTransport,
		baseURL: server.URL,
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}
	params := map[string]interface{}{
		"resource":      "sheet",
		"operation":     "read",
		"spreadsheetId": "test-id",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Google Sheets API error")
}

// --- Execute: multiple input items ---

func TestGoogleSheetsNode_Execute_MultipleInputItems(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"range":  "Sheet1!A:Z",
			"values": []interface{}{},
		})
	}))
	defer server.Close()

	node := NewGoogleSheetsNode()
	node.httpClient = server.Client()
	node.httpClient.Transport = &rewriteTransport{
		base:    http.DefaultTransport,
		baseURL: server.URL,
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"item": 1}},
		{JSON: map[string]interface{}{"item": 2}},
		{JSON: map[string]interface{}{"item": 3}},
	}
	params := map[string]interface{}{
		"resource":      "sheet",
		"operation":     "read",
		"spreadsheetId": "test-id",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, 3, callCount, "should call API once per input item")
}

// --- Execute: default range ---

func TestGoogleSheetsNode_Execute_DefaultRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The default range "A:Z" should be URL-encoded in the request
		assert.Contains(t, r.URL.String(), "A%3AZ")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"values": []interface{}{}})
	}))
	defer server.Close()

	node := NewGoogleSheetsNode()
	node.httpClient = server.Client()
	node.httpClient.Transport = &rewriteTransport{
		base:    http.DefaultTransport,
		baseURL: server.URL,
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}
	params := map[string]interface{}{
		"resource":      "sheet",
		"operation":     "read",
		"spreadsheetId": "test-id",
		// No range provided - should default to "A:Z"
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.NoError(t, err)
}

// --- Execute: append with explicit values ---

func TestGoogleSheetsNode_Execute_AppendWithExplicitValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		assert.NotNil(t, body["values"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"updates": map[string]interface{}{}})
	}))
	defer server.Close()

	node := NewGoogleSheetsNode()
	node.httpClient = server.Client()
	node.httpClient.Transport = &rewriteTransport{
		base:    http.DefaultTransport,
		baseURL: server.URL,
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}
	params := map[string]interface{}{
		"resource":      "sheet",
		"operation":     "append",
		"spreadsheetId": "test-id",
		"range":         "Sheet1!A:B",
		"values":        [][]interface{}{{"Alice", "30"}, {"Bob", "25"}},
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, result, 1)
}

// --- rewriteTransport helper ---

// rewriteTransport is an http.RoundTripper that rewrites all request URLs
// to point at the test server.
type rewriteTransport struct {
	base    http.RoundTripper
	baseURL string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite the URL to point to the mock server while preserving the path and query.
	req.URL.Scheme = "http"
	req.URL.Host = t.baseURL[len("http://"):]
	return t.base.RoundTrip(req)
}
