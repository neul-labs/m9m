package productivity

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotionNode_GetPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/pages/page123")
		assert.Equal(t, "Bearer notion-key", r.Header.Get("Authorization"))
		assert.Equal(t, "2022-06-28", r.Header.Get("Notion-Version"))

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     "page123",
			"object": "page",
		})
	}))
	defer server.Close()

	node := NewNotionNodeWithClient(server.Client())
	params := map[string]interface{}{
		"baseUrl":   server.URL,
		"operation": "getPage",
		"apiKey":    "notion-key",
		"pageId":    "page123",
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Equal(t, "page123", result[0].JSON["id"])
}

func TestNotionNode_CreatePage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/pages")

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		parent := body["parent"].(map[string]interface{})
		assert.Equal(t, "db123", parent["database_id"])

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     "newpage",
			"object": "page",
		})
	}))
	defer server.Close()

	node := NewNotionNodeWithClient(server.Client())
	params := map[string]interface{}{
		"baseUrl":   server.URL,
		"operation": "createPage",
		"apiKey":    "key",
		"parentId":  "db123",
		"properties": map[string]interface{}{
			"Name": map[string]interface{}{
				"title": []interface{}{
					map[string]interface{}{"text": map[string]interface{}{"content": "Test"}},
				},
			},
		},
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Equal(t, "newpage", result[0].JSON["id"])
}

func TestNotionNode_UpdatePage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "page1", "object": "page"})
	}))
	defer server.Close()

	node := NewNotionNodeWithClient(server.Client())
	params := map[string]interface{}{
		"baseUrl":   server.URL,
		"operation": "updatePage",
		"apiKey":    "key",
		"pageId":    "page1",
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Equal(t, "page1", result[0].JSON["id"])
}

func TestNotionNode_QueryDatabase(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/databases/db1/query")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []interface{}{
				map[string]interface{}{"id": "page1"},
			},
		})
	}))
	defer server.Close()

	node := NewNotionNodeWithClient(server.Client())
	params := map[string]interface{}{
		"baseUrl":    server.URL,
		"operation":  "queryDatabase",
		"apiKey":     "key",
		"databaseId": "db1",
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.NotNil(t, result[0].JSON["results"])
}

func TestNotionNode_Search(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/search")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []interface{}{},
		})
	}))
	defer server.Close()

	node := NewNotionNodeWithClient(server.Client())
	params := map[string]interface{}{
		"baseUrl":   server.URL,
		"operation": "search",
		"apiKey":    "key",
		"query":     "test",
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNotionNode_MissingApiKey(t *testing.T) {
	node := NewNotionNode()
	_, err := node.Execute(nil, map[string]interface{}{"operation": "getPage"})
	assert.Error(t, err)
}

func TestNotionNode_MissingPageId(t *testing.T) {
	node := NewNotionNode()
	_, err := node.Execute(nil, map[string]interface{}{
		"operation": "getPage",
		"apiKey":    "key",
	})
	assert.Error(t, err)
}

func TestNotionNode_Validate(t *testing.T) {
	node := NewNotionNode()
	assert.Error(t, node.ValidateParameters(nil))
	assert.Error(t, node.ValidateParameters(map[string]interface{}{}))
	assert.NoError(t, node.ValidateParameters(map[string]interface{}{
		"apiKey":    "key",
		"operation": "getPage",
	}))
	assert.Error(t, node.ValidateParameters(map[string]interface{}{
		"apiKey":    "key",
		"operation": "invalid",
	}))
}

func TestNotionNode_Description(t *testing.T) {
	node := NewNotionNode()
	desc := node.Description()
	assert.Equal(t, "Notion", desc.Name)
	assert.Equal(t, "Productivity", desc.Category)
}
