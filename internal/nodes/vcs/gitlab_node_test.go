package vcs

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGitLabNode(t *testing.T) {
	node := NewGitLabNode()
	require.NotNil(t, node)
	assert.NotNil(t, node.httpClient)
	assert.NotNil(t, node.BaseNode)
}

func TestGitLabNode_Description(t *testing.T) {
	node := NewGitLabNode()
	desc := node.Description()

	assert.Equal(t, "GitLab", desc.Name)
	assert.Equal(t, "vcs", desc.Category)
	assert.Contains(t, desc.Description, "GitLab API")
}

func TestGitLabNode_ValidateParameters(t *testing.T) {
	node := NewGitLabNode()

	tests := []struct {
		name      string
		params    map[string]interface{}
		expectErr bool
	}{
		{
			name:      "nil params",
			params:    nil,
			expectErr: false,
		},
		{
			name:      "empty params",
			params:    map[string]interface{}{},
			expectErr: false,
		},
		{
			name: "valid params",
			params: map[string]interface{}{
				"resource":  "issue",
				"operation": "get",
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

func TestGitLabNode_Execute_MissingToken(t *testing.T) {
	node := NewGitLabNode()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"test": "data"}},
	}

	params := map[string]interface{}{
		"resource":  "issue",
		"operation": "get",
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "GitLab token is required")
}

func TestGitLabNode_Execute_UnsupportedResource(t *testing.T) {
	node := NewGitLabNode()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "unsupported",
		"operation": "get",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported resource")
}

func TestGitLabNode_getToken(t *testing.T) {
	node := NewGitLabNode()

	tests := []struct {
		name     string
		params   map[string]interface{}
		expected string
	}{
		{
			name:     "no credentials",
			params:   map[string]interface{}{},
			expected: "",
		},
		{
			name: "accessToken",
			params: map[string]interface{}{
				"credentials": map[string]interface{}{
					"accessToken": "access-token-123",
				},
			},
			expected: "access-token-123",
		},
		{
			name: "apiToken",
			params: map[string]interface{}{
				"credentials": map[string]interface{}{
					"apiToken": "api-token-456",
				},
			},
			expected: "api-token-456",
		},
		{
			name: "accessToken takes precedence over apiToken",
			params: map[string]interface{}{
				"credentials": map[string]interface{}{
					"accessToken": "access-first",
					"apiToken":    "api-second",
				},
			},
			expected: "access-first",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := node.getToken(tt.params)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGitLabNode_getBaseURL(t *testing.T) {
	node := NewGitLabNode()

	tests := []struct {
		name     string
		params   map[string]interface{}
		expected string
	}{
		{
			name:     "default URL",
			params:   map[string]interface{}{},
			expected: "https://gitlab.com/api/v4",
		},
		{
			name: "custom base URL",
			params: map[string]interface{}{
				"baseUrl": "https://gitlab.example.com/api/v4",
			},
			expected: "https://gitlab.example.com/api/v4",
		},
		{
			name: "empty base URL uses default",
			params: map[string]interface{}{
				"baseUrl": "",
			},
			expected: "https://gitlab.com/api/v4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := node.getBaseURL(tt.params)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGitLabNode_getIntParam(t *testing.T) {
	node := NewGitLabNode()

	tests := []struct {
		name     string
		params   map[string]interface{}
		key      string
		expected int
	}{
		{
			name:     "missing key",
			params:   map[string]interface{}{},
			key:      "missing",
			expected: 0,
		},
		{
			name:     "float64 value",
			params:   map[string]interface{}{"id": float64(42)},
			key:      "id",
			expected: 42,
		},
		{
			name:     "int value",
			params:   map[string]interface{}{"id": 99},
			key:      "id",
			expected: 99,
		},
		{
			name:     "string value",
			params:   map[string]interface{}{"id": "123"},
			key:      "id",
			expected: 123,
		},
		{
			name:     "invalid string value",
			params:   map[string]interface{}{"id": "notanumber"},
			key:      "id",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := node.getIntParam(tt.params, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper to create a GitLab test node that uses the given test server as base URL.
func newTestGitLabNode(server *httptest.Server) *GitLabNode {
	node := NewGitLabNode()
	node.httpClient = server.Client()
	return node
}

func TestGitLabNode_Execute_IssueCreate(t *testing.T) {
	mockResponse := map[string]interface{}{
		"iid":   float64(1),
		"title": "New Issue",
		"state": "opened",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/projects/123/issues")
		assert.Equal(t, "test-token", r.Header.Get("PRIVATE-TOKEN"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Verify body
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)
		assert.Equal(t, "New Issue", reqBody["title"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "issue",
		"operation": "create",
		"projectId": "123",
		"title":     "New Issue",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "New Issue", results[0].JSON["title"])
}

func TestGitLabNode_Execute_IssueCreate_MissingProjectId(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "issue",
		"operation": "create",
		"title":     "New Issue",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "projectId is required")
}

func TestGitLabNode_Execute_IssueCreate_MissingTitle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "issue",
		"operation": "create",
		"projectId": "123",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "title is required")
}

func TestGitLabNode_Execute_IssueGet(t *testing.T) {
	mockResponse := map[string]interface{}{
		"iid":   float64(5),
		"title": "Test Issue",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/projects/123/issues/5")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "issue",
		"operation": "get",
		"projectId": "123",
		"issueIid":  float64(5),
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Test Issue", results[0].JSON["title"])
}

func TestGitLabNode_Execute_IssueGet_MissingParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	// Missing both projectId and issueIid
	params := map[string]interface{}{
		"resource":  "issue",
		"operation": "get",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "projectId and issueIid are required")
}

func TestGitLabNode_Execute_IssueGetAll(t *testing.T) {
	mockResponse := map[string]interface{}{
		"total": float64(5),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/projects/123/issues")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "issue",
		"operation": "getAll",
		"projectId": "123",
		"state":     "opened",
		"labels":    "bug",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
}

func TestGitLabNode_Execute_IssueUpdate(t *testing.T) {
	mockResponse := map[string]interface{}{
		"iid":   float64(5),
		"title": "Updated Title",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Contains(t, r.URL.Path, "/projects/123/issues/5")

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)
		assert.Equal(t, "Updated Title", reqBody["title"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "issue",
		"operation": "update",
		"projectId": "123",
		"issueIid":  float64(5),
		"title":     "Updated Title",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
}

func TestGitLabNode_Execute_IssueDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "/projects/123/issues/5")
		assert.Equal(t, "test-token", r.Header.Get("PRIVATE-TOKEN"))

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "issue",
		"operation": "delete",
		"projectId": "123",
		"issueIid":  float64(5),
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, true, results[0].JSON["success"])
}

func TestGitLabNode_Execute_UnsupportedIssueOperation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "issue",
		"operation": "invalid",
		"projectId": "123",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported issue operation")
}

func TestGitLabNode_Execute_MergeRequestCreate(t *testing.T) {
	mockResponse := map[string]interface{}{
		"iid":   float64(1),
		"title": "New MR",
		"state": "opened",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/projects/123/merge_requests")

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)
		assert.Equal(t, "feature", reqBody["source_branch"])
		assert.Equal(t, "main", reqBody["target_branch"])
		assert.Equal(t, "New MR", reqBody["title"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":      "mergeRequest",
		"operation":     "create",
		"projectId":     "123",
		"source_branch": "feature",
		"target_branch": "main",
		"title":         "New MR",
		"baseUrl":       server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "New MR", results[0].JSON["title"])
}

func TestGitLabNode_Execute_MergeRequestCreate_MissingFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	// Missing source_branch, target_branch, and title
	params := map[string]interface{}{
		"resource":  "mergeRequest",
		"operation": "create",
		"projectId": "123",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source_branch, target_branch, and title are required")
}

func TestGitLabNode_Execute_MergeRequestGet(t *testing.T) {
	mockResponse := map[string]interface{}{
		"iid":   float64(10),
		"title": "Test MR",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/projects/123/merge_requests/10")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":        "mergeRequest",
		"operation":       "get",
		"projectId":       "123",
		"mergeRequestIid": float64(10),
		"baseUrl":         server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Test MR", results[0].JSON["title"])
}

func TestGitLabNode_Execute_MergeRequestGetAll(t *testing.T) {
	mockResponse := map[string]interface{}{
		"total": float64(2),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/projects/123/merge_requests")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "mergeRequest",
		"operation": "getAll",
		"projectId": "123",
		"state":     "opened",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
}

func TestGitLabNode_Execute_MergeRequestMerge(t *testing.T) {
	mockResponse := map[string]interface{}{
		"iid":   float64(10),
		"state": "merged",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Contains(t, r.URL.Path, "/projects/123/merge_requests/10/merge")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":        "mergeRequest",
		"operation":       "merge",
		"projectId":       "123",
		"mergeRequestIid": float64(10),
		"baseUrl":         server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "merged", results[0].JSON["state"])
}

func TestGitLabNode_Execute_UnsupportedMergeRequestOperation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "mergeRequest",
		"operation": "invalid",
		"projectId": "123",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported merge request operation")
}

func TestGitLabNode_Execute_PipelineTrigger(t *testing.T) {
	mockResponse := map[string]interface{}{
		"id":     float64(1),
		"status": "pending",
		"ref":    "main",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/projects/123/pipeline")

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)
		assert.Equal(t, "main", reqBody["ref"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "pipeline",
		"operation": "trigger",
		"projectId": "123",
		"ref":       "main",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "pending", results[0].JSON["status"])
}

func TestGitLabNode_Execute_PipelineTrigger_MissingRef(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "pipeline",
		"operation": "trigger",
		"projectId": "123",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ref is required")
}

func TestGitLabNode_Execute_PipelineGet(t *testing.T) {
	mockResponse := map[string]interface{}{
		"id":     float64(42),
		"status": "success",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/projects/123/pipelines/42")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":   "pipeline",
		"operation":  "get",
		"projectId":  "123",
		"pipelineId": float64(42),
		"baseUrl":    server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "success", results[0].JSON["status"])
}

func TestGitLabNode_Execute_PipelineGetAll(t *testing.T) {
	mockResponse := map[string]interface{}{
		"total": float64(3),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/projects/123/pipelines")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "pipeline",
		"operation": "getAll",
		"projectId": "123",
		"status":    "success",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
}

func TestGitLabNode_Execute_PipelineCancel(t *testing.T) {
	mockResponse := map[string]interface{}{
		"id":     float64(42),
		"status": "canceled",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/projects/123/pipelines/42/cancel")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":   "pipeline",
		"operation":  "cancel",
		"projectId":  "123",
		"pipelineId": float64(42),
		"baseUrl":    server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
}

func TestGitLabNode_Execute_PipelineRetry(t *testing.T) {
	mockResponse := map[string]interface{}{
		"id":     float64(42),
		"status": "pending",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/projects/123/pipelines/42/retry")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":   "pipeline",
		"operation":  "retry",
		"projectId":  "123",
		"pipelineId": float64(42),
		"baseUrl":    server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
}

func TestGitLabNode_Execute_UnsupportedPipelineOperation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "pipeline",
		"operation": "invalid",
		"projectId": "123",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported pipeline operation")
}

func TestGitLabNode_Execute_ProjectGet(t *testing.T) {
	mockResponse := map[string]interface{}{
		"id":   float64(123),
		"name": "test-project",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/projects/123")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "project",
		"operation": "get",
		"projectId": "123",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "test-project", results[0].JSON["name"])
}

func TestGitLabNode_Execute_ProjectGet_MissingId(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "project",
		"operation": "get",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "projectId is required")
}

func TestGitLabNode_Execute_ProjectList(t *testing.T) {
	mockResponse := map[string]interface{}{
		"total": float64(5),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		// Should be /projects with no project ID
		assert.Equal(t, "/projects", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "project",
		"operation": "list",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
}

func TestGitLabNode_Execute_UnsupportedProjectOperation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "project",
		"operation": "invalid",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported project operation")
}

func TestGitLabNode_Execute_UserGetAuthenticated(t *testing.T) {
	mockResponse := map[string]interface{}{
		"id":       float64(1),
		"username": "testuser",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/user", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "user",
		"operation": "getAuthenticated",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "testuser", results[0].JSON["username"])
}

func TestGitLabNode_Execute_UserGet(t *testing.T) {
	mockResponse := map[string]interface{}{
		"id":       float64(42),
		"username": "someuser",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/users/42", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "user",
		"operation": "get",
		"userId":    float64(42),
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "someuser", results[0].JSON["username"])
}

func TestGitLabNode_Execute_UserGet_MissingUserId(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "user",
		"operation": "get",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "userId is required")
}

func TestGitLabNode_Execute_UnsupportedUserOperation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "user",
		"operation": "invalid",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported user operation")
}

func TestGitLabNode_Execute_BranchList(t *testing.T) {
	mockResponse := map[string]interface{}{
		"total": float64(3),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/projects/123/repository/branches")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "branch",
		"operation": "list",
		"projectId": "123",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
}

func TestGitLabNode_Execute_BranchGet(t *testing.T) {
	mockResponse := map[string]interface{}{
		"name":    "main",
		"merged":  false,
		"default": true,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/projects/123/repository/branches/main")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "branch",
		"operation": "get",
		"projectId": "123",
		"branch":    "main",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "main", results[0].JSON["name"])
}

func TestGitLabNode_Execute_BranchGet_MissingParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "branch",
		"operation": "get",
		"projectId": "123",
		// Missing branch name
		"baseUrl": server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "projectId and branch are required")
}

func TestGitLabNode_Execute_UnsupportedBranchOperation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "branch",
		"operation": "invalid",
		"projectId": "123",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported branch operation")
}

func TestGitLabNode_Execute_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message": "401 Unauthorized"}`))
	}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "project",
		"operation": "list",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "bad-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "GitLab API error (401)")
}

func TestGitLabNode_Execute_DefaultResourceAndOperation(t *testing.T) {
	mockResponse := map[string]interface{}{
		"iid": float64(1),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := newTestGitLabNode(server)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{
			"projectId": "123",
		}},
	}

	// No resource or operation -- defaults to issue/get
	// issueIid defaults to 0, so we expect the "projectId and issueIid are required" error
	params := map[string]interface{}{
		"projectId": "123",
		"issueIid":  float64(1),
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
}

func TestGitLabNode_Execute_ProjectIdFromInputData(t *testing.T) {
	mockResponse := map[string]interface{}{
		"iid": float64(1),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := newTestGitLabNode(server)

	// projectId comes from the input data item, not params
	inputData := []model.DataItem{
		{JSON: map[string]interface{}{
			"projectId": "456",
		}},
	}

	params := map[string]interface{}{
		"resource":  "issue",
		"operation": "getAll",
		"baseUrl":   server.URL,
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
}
