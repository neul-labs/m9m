package database

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupESServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *ElasticsearchNode) {
	t.Helper()
	server := httptest.NewServer(handler)
	node := NewElasticsearchNodeWithClient(server.Client())
	return server, node
}

func TestElasticsearchNode_Search(t *testing.T) {
	server, node := setupESServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/_search")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"hits": map[string]interface{}{
				"total": map[string]interface{}{"value": 1},
				"hits": []map[string]interface{}{
					{"_id": "1", "_source": map[string]interface{}{"name": "test"}},
				},
			},
		})
	})
	defer server.Close()

	params := map[string]interface{}{
		"baseUrl":   server.URL,
		"operation": "search",
		"index":     "myindex",
		"query":     map[string]interface{}{"match_all": map[string]interface{}{}},
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.NotNil(t, result[0].JSON["hits"])
}

func TestElasticsearchNode_Index(t *testing.T) {
	server, node := setupESServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/myindex/_doc")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"_id":    "1",
			"result": "created",
		})
	})
	defer server.Close()

	params := map[string]interface{}{
		"baseUrl":   server.URL,
		"operation": "index",
		"index":     "myindex",
		"document":  map[string]interface{}{"name": "test"},
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Equal(t, "created", result[0].JSON["result"])
}

func TestElasticsearchNode_IndexWithID(t *testing.T) {
	server, node := setupESServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Contains(t, r.URL.Path, "/myindex/_doc/abc")
		json.NewEncoder(w).Encode(map[string]interface{}{"result": "created"})
	})
	defer server.Close()

	params := map[string]interface{}{
		"baseUrl":    server.URL,
		"operation":  "index",
		"index":      "myindex",
		"documentId": "abc",
		"document":   map[string]interface{}{"name": "test"},
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Equal(t, "created", result[0].JSON["result"])
}

func TestElasticsearchNode_Get(t *testing.T) {
	server, node := setupESServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"_id":     "1",
			"_source": map[string]interface{}{"name": "test"},
		})
	})
	defer server.Close()

	params := map[string]interface{}{
		"baseUrl":    server.URL,
		"operation":  "get",
		"index":      "myindex",
		"documentId": "1",
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Equal(t, "1", result[0].JSON["_id"])
}

func TestElasticsearchNode_Delete(t *testing.T) {
	server, node := setupESServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		json.NewEncoder(w).Encode(map[string]interface{}{"result": "deleted"})
	})
	defer server.Close()

	params := map[string]interface{}{
		"baseUrl":    server.URL,
		"operation":  "delete",
		"index":      "myindex",
		"documentId": "1",
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Equal(t, "deleted", result[0].JSON["result"])
}

func TestElasticsearchNode_Update(t *testing.T) {
	server, node := setupESServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/_update/")
		json.NewEncoder(w).Encode(map[string]interface{}{"result": "updated"})
	})
	defer server.Close()

	params := map[string]interface{}{
		"baseUrl":    server.URL,
		"operation":  "update",
		"index":      "myindex",
		"documentId": "1",
		"document":   map[string]interface{}{"name": "updated"},
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Equal(t, "updated", result[0].JSON["result"])
}

func TestElasticsearchNode_CreateIndex(t *testing.T) {
	server, node := setupESServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/newindex", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]interface{}{"acknowledged": true})
	})
	defer server.Close()

	params := map[string]interface{}{
		"baseUrl":   server.URL,
		"operation": "createIndex",
		"index":     "newindex",
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Equal(t, true, result[0].JSON["acknowledged"])
}

func TestElasticsearchNode_Bulk(t *testing.T) {
	server, node := setupESServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/x-ndjson", r.Header.Get("Content-Type"))
		json.NewEncoder(w).Encode(map[string]interface{}{"errors": false, "items": []interface{}{}})
	})
	defer server.Close()

	items := []model.DataItem{
		{JSON: map[string]interface{}{"name": "doc1"}},
		{JSON: map[string]interface{}{"name": "doc2"}},
	}

	params := map[string]interface{}{
		"baseUrl":   server.URL,
		"operation": "bulk",
		"index":     "myindex",
	}

	result, err := node.Execute(items, params)
	require.NoError(t, err)
	assert.Equal(t, false, result[0].JSON["errors"])
}

func TestElasticsearchNode_Validate(t *testing.T) {
	node := NewElasticsearchNode()

	assert.Error(t, node.ValidateParameters(nil))
	assert.NoError(t, node.ValidateParameters(map[string]interface{}{"operation": "search"}))
	assert.Error(t, node.ValidateParameters(map[string]interface{}{"operation": "invalid"}))
}

func TestElasticsearchNode_MissingRequiredParams(t *testing.T) {
	node := NewElasticsearchNode()

	// get without index
	_, err := node.Execute(nil, map[string]interface{}{"operation": "get"})
	assert.Error(t, err)

	// get without documentId
	_, err = node.Execute(nil, map[string]interface{}{"operation": "get", "index": "idx"})
	assert.Error(t, err)

	// index without index name
	_, err = node.Execute(nil, map[string]interface{}{"operation": "index"})
	assert.Error(t, err)
}

func TestElasticsearchNode_Description(t *testing.T) {
	node := NewElasticsearchNode()
	desc := node.Description()
	assert.Equal(t, "Elasticsearch", desc.Name)
	assert.Equal(t, "Database", desc.Category)
}
