package vcs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGitHubNode(t *testing.T) {
	node := NewGitHubNode()
	require.NotNil(t, node)
	assert.NotNil(t, node.httpClient)
	assert.NotNil(t, node.BaseNode)
}

func TestGitHubNode_Description(t *testing.T) {
	node := NewGitHubNode()
	desc := node.Description()

	assert.Equal(t, "GitHub", desc.Name)
	assert.Equal(t, "vcs", desc.Category)
	assert.Contains(t, desc.Description, "GitHub API")
}

func TestGitHubNode_ValidateParameters(t *testing.T) {
	node := NewGitHubNode()

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
				"resource":  "repository",
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

func TestGitHubNode_Execute_MissingToken(t *testing.T) {
	node := NewGitHubNode()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"test": "data"}},
	}

	params := map[string]interface{}{
		"resource":  "repository",
		"operation": "get",
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "GitHub token is required")
}

func TestGitHubNode_Execute_UnsupportedResource(t *testing.T) {
	node := NewGitHubNode()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"test": "data"}},
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

func TestGitHubNode_Execute_RepositoryGet(t *testing.T) {
	// Set up mock server
	mockResponse := map[string]interface{}{
		"id":        float64(12345),
		"name":      "test-repo",
		"full_name": "testowner/test-repo",
		"private":   false,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/repos/testowner/test-repo", r.URL.Path)
		assert.Equal(t, "token test-token-123", r.Header.Get("Authorization"))
		assert.Equal(t, "application/vnd.github.v3+json", r.Header.Get("Accept"))
		assert.Equal(t, "m9m", r.Header.Get("User-Agent"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	// Create node with custom HTTP client pointing to mock server
	node := NewGitHubNode()
	node.httpClient = server.Client()

	// Override the makeAPIRequest to use our test server URL
	// Since makeAPIRequest hardcodes https://api.github.com, we need to use
	// a transport that redirects requests.
	node.httpClient.Transport = &rewriteTransport{
		originalURL: "https://api.github.com",
		newURL:      server.URL,
		rt:          http.DefaultTransport,
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":   "repository",
		"operation":  "get",
		"owner":      "testowner",
		"repository": "test-repo",
		"credentials": map[string]interface{}{
			"accessToken": "test-token-123",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.Equal(t, "test-repo", results[0].JSON["name"])
	assert.Equal(t, "testowner/test-repo", results[0].JSON["full_name"])
}

func TestGitHubNode_Execute_RepositoryList(t *testing.T) {
	mockResponse := map[string]interface{}{
		"total_count": float64(1),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/users/testowner/repos", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := NewGitHubNode()
	node.httpClient = server.Client()
	node.httpClient.Transport = &rewriteTransport{
		originalURL: "https://api.github.com",
		newURL:      server.URL,
		rt:          http.DefaultTransport,
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "repository",
		"operation": "list",
		"owner":     "testowner",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
}

func TestGitHubNode_Execute_UnsupportedRepoOperation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer server.Close()

	node := NewGitHubNode()
	node.httpClient = server.Client()
	node.httpClient.Transport = &rewriteTransport{
		originalURL: "https://api.github.com",
		newURL:      server.URL,
		rt:          http.DefaultTransport,
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":   "repository",
		"operation":  "foobar",
		"owner":      "testowner",
		"repository": "test-repo",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported repository operation")
}

func TestGitHubNode_Execute_IssueList(t *testing.T) {
	mockResponse := map[string]interface{}{
		"total": float64(2),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/repos/testowner/test-repo/issues", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := NewGitHubNode()
	node.httpClient = server.Client()
	node.httpClient.Transport = &rewriteTransport{
		originalURL: "https://api.github.com",
		newURL:      server.URL,
		rt:          http.DefaultTransport,
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":   "issue",
		"operation":  "list",
		"owner":      "testowner",
		"repository": "test-repo",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
}

func TestGitHubNode_Execute_IssueGet(t *testing.T) {
	mockResponse := map[string]interface{}{
		"id":     float64(1),
		"number": float64(42),
		"title":  "Test Issue",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/repos/testowner/test-repo/issues/42", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := NewGitHubNode()
	node.httpClient = server.Client()
	node.httpClient.Transport = &rewriteTransport{
		originalURL: "https://api.github.com",
		newURL:      server.URL,
		rt:          http.DefaultTransport,
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":    "issue",
		"operation":   "get",
		"owner":       "testowner",
		"repository":  "test-repo",
		"issueNumber": float64(42),
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Test Issue", results[0].JSON["title"])
}

func TestGitHubNode_Execute_UnsupportedIssueOperation(t *testing.T) {
	node := NewGitHubNode()
	node.httpClient = &http.Client{Transport: &rewriteTransport{
		originalURL: "https://api.github.com",
		newURL:      "http://localhost:1",
		rt:          http.DefaultTransport,
	}}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "issue",
		"operation": "invalid",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported issue operation")
}

func TestGitHubNode_Execute_PullRequestList(t *testing.T) {
	mockResponse := map[string]interface{}{
		"total": float64(3),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/repos/testowner/test-repo/pulls", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := NewGitHubNode()
	node.httpClient = server.Client()
	node.httpClient.Transport = &rewriteTransport{
		originalURL: "https://api.github.com",
		newURL:      server.URL,
		rt:          http.DefaultTransport,
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":   "pullRequest",
		"operation":  "list",
		"owner":      "testowner",
		"repository": "test-repo",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
}

func TestGitHubNode_Execute_PullRequestGet(t *testing.T) {
	mockResponse := map[string]interface{}{
		"id":     float64(1),
		"number": float64(99),
		"title":  "Test PR",
		"state":  "open",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/repos/testowner/test-repo/pulls/99", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := NewGitHubNode()
	node.httpClient = server.Client()
	node.httpClient.Transport = &rewriteTransport{
		originalURL: "https://api.github.com",
		newURL:      server.URL,
		rt:          http.DefaultTransport,
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":   "pullRequest",
		"operation":  "get",
		"owner":      "testowner",
		"repository": "test-repo",
		"pullNumber": float64(99),
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Test PR", results[0].JSON["title"])
	assert.Equal(t, "open", results[0].JSON["state"])
}

func TestGitHubNode_Execute_UnsupportedPROperation(t *testing.T) {
	node := NewGitHubNode()
	node.httpClient = &http.Client{Transport: &rewriteTransport{
		originalURL: "https://api.github.com",
		newURL:      "http://localhost:1",
		rt:          http.DefaultTransport,
	}}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "pullRequest",
		"operation": "invalid",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported pull request operation")
}

func TestGitHubNode_Execute_UserGetAuthenticated(t *testing.T) {
	mockResponse := map[string]interface{}{
		"login": "testuser",
		"id":    float64(1001),
		"email": "test@example.com",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/user", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := NewGitHubNode()
	node.httpClient = server.Client()
	node.httpClient.Transport = &rewriteTransport{
		originalURL: "https://api.github.com",
		newURL:      server.URL,
		rt:          http.DefaultTransport,
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "user",
		"operation": "getAuthenticated",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "testuser", results[0].JSON["login"])
}

func TestGitHubNode_Execute_UserGet(t *testing.T) {
	mockResponse := map[string]interface{}{
		"login": "otheruser",
		"id":    float64(2002),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/users/otheruser", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := NewGitHubNode()
	node.httpClient = server.Client()
	node.httpClient.Transport = &rewriteTransport{
		originalURL: "https://api.github.com",
		newURL:      server.URL,
		rt:          http.DefaultTransport,
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "user",
		"operation": "get",
		"username":  "otheruser",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "otheruser", results[0].JSON["login"])
}

func TestGitHubNode_Execute_UnsupportedUserOperation(t *testing.T) {
	node := NewGitHubNode()
	node.httpClient = &http.Client{Transport: &rewriteTransport{
		originalURL: "https://api.github.com",
		newURL:      "http://localhost:1",
		rt:          http.DefaultTransport,
	}}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":  "user",
		"operation": "invalid",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported user operation")
}

func TestGitHubNode_Execute_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Not Found",
		})
	}))
	defer server.Close()

	node := NewGitHubNode()
	node.httpClient = server.Client()
	node.httpClient.Transport = &rewriteTransport{
		originalURL: "https://api.github.com",
		newURL:      server.URL,
		rt:          http.DefaultTransport,
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":   "repository",
		"operation":  "get",
		"owner":      "testowner",
		"repository": "nonexistent",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "GitHub API error (404)")
}

func TestGitHubNode_Execute_DefaultResourceAndOperation(t *testing.T) {
	mockResponse := map[string]interface{}{
		"id":   float64(1),
		"name": "default-repo",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := NewGitHubNode()
	node.httpClient = server.Client()
	node.httpClient.Transport = &rewriteTransport{
		originalURL: "https://api.github.com",
		newURL:      server.URL,
		rt:          http.DefaultTransport,
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{
			"owner":      "testowner",
			"repository": "default-repo",
		}},
	}

	// No resource or operation specified -- defaults to repository/get
	params := map[string]interface{}{
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
}

func TestGitHubNode_Execute_OwnerRepoFromInputData(t *testing.T) {
	mockResponse := map[string]interface{}{
		"id":   float64(1),
		"name": "from-input",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/inputowner/from-input", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	node := NewGitHubNode()
	node.httpClient = server.Client()
	node.httpClient.Transport = &rewriteTransport{
		originalURL: "https://api.github.com",
		newURL:      server.URL,
		rt:          http.DefaultTransport,
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{
			"owner":      "inputowner",
			"repository": "from-input",
		}},
	}

	params := map[string]interface{}{
		"resource":  "repository",
		"operation": "get",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "from-input", results[0].JSON["name"])
}

func TestGitHubNode_Execute_MultipleInputItems(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"call": float64(callCount),
		})
	}))
	defer server.Close()

	node := NewGitHubNode()
	node.httpClient = server.Client()
	node.httpClient.Transport = &rewriteTransport{
		originalURL: "https://api.github.com",
		newURL:      server.URL,
		rt:          http.DefaultTransport,
	}

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{}},
		{JSON: map[string]interface{}{}},
		{JSON: map[string]interface{}{}},
	}

	params := map[string]interface{}{
		"resource":   "repository",
		"operation":  "get",
		"owner":      "testowner",
		"repository": "test-repo",
		"credentials": map[string]interface{}{
			"accessToken": "test-token",
		},
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	assert.Equal(t, 3, callCount)
}

func TestGitHubNode_getToken(t *testing.T) {
	node := NewGitHubNode()

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
			name: "credentials with accessToken",
			params: map[string]interface{}{
				"credentials": map[string]interface{}{
					"accessToken": "my-token",
				},
			},
			expected: "my-token",
		},
		{
			name: "credentials without accessToken",
			params: map[string]interface{}{
				"credentials": map[string]interface{}{
					"otherKey": "value",
				},
			},
			expected: "",
		},
		{
			name: "credentials is not a map",
			params: map[string]interface{}{
				"credentials": "not-a-map",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := node.getToken(tt.params)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// rewriteTransport is a test helper that rewrites request URLs
// so that requests intended for the real API go to our test server instead.
type rewriteTransport struct {
	originalURL string
	newURL      string
	rt          http.RoundTripper
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite the URL
	reqURL := req.URL.String()
	if len(reqURL) >= len(t.originalURL) && reqURL[:len(t.originalURL)] == t.originalURL {
		newURLStr := t.newURL + reqURL[len(t.originalURL):]
		newReq, err := http.NewRequest(req.Method, newURLStr, req.Body)
		if err != nil {
			return nil, err
		}
		newReq.Header = req.Header
		return t.rt.RoundTrip(newReq)
	}
	return t.rt.RoundTrip(req)
}
