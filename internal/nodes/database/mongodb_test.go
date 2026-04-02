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

func setupMongoServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *MongoDBNode) {
	t.Helper()
	server := httptest.NewServer(handler)
	node := NewMongoDBNodeWithClient(server.Client())
	return server, node
}

func baseMongoParams(serverURL string) map[string]interface{} {
	return map[string]interface{}{
		"baseUrl":    serverURL,
		"database":   "testdb",
		"collection": "users",
	}
}

func TestMongoDBNode_Find(t *testing.T) {
	server, node := setupMongoServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/action/find")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"documents": []interface{}{
				map[string]interface{}{"_id": "1", "name": "Alice"},
			},
		})
	})
	defer server.Close()

	params := baseMongoParams(server.URL)
	params["operation"] = "find"
	params["filter"] = map[string]interface{}{"name": "Alice"}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.NotNil(t, result[0].JSON["documents"])
}

func TestMongoDBNode_FindOne(t *testing.T) {
	server, node := setupMongoServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/action/findOne")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"document": map[string]interface{}{"_id": "1", "name": "Alice"},
		})
	})
	defer server.Close()

	params := baseMongoParams(server.URL)
	params["operation"] = "findOne"

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.NotNil(t, result[0].JSON["document"])
}

func TestMongoDBNode_Insert(t *testing.T) {
	server, node := setupMongoServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/action/insertOne")
		json.NewEncoder(w).Encode(map[string]interface{}{"insertedId": "abc123"})
	})
	defer server.Close()

	params := baseMongoParams(server.URL)
	params["operation"] = "insert"
	params["document"] = map[string]interface{}{"name": "Bob"}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Equal(t, "abc123", result[0].JSON["insertedId"])
}

func TestMongoDBNode_InsertFromInput(t *testing.T) {
	server, node := setupMongoServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{"insertedId": "xyz"})
	})
	defer server.Close()

	params := baseMongoParams(server.URL)
	params["operation"] = "insert"

	input := []model.DataItem{{JSON: map[string]interface{}{"name": "Charlie"}}}
	result, err := node.Execute(input, params)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMongoDBNode_InsertMany(t *testing.T) {
	server, node := setupMongoServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/action/insertMany")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"insertedIds": []string{"a", "b"},
		})
	})
	defer server.Close()

	params := baseMongoParams(server.URL)
	params["operation"] = "insertMany"

	input := []model.DataItem{
		{JSON: map[string]interface{}{"name": "A"}},
		{JSON: map[string]interface{}{"name": "B"}},
	}

	result, err := node.Execute(input, params)
	require.NoError(t, err)
	assert.NotNil(t, result[0].JSON["insertedIds"])
}

func TestMongoDBNode_Update(t *testing.T) {
	server, node := setupMongoServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/action/updateMany")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"matchedCount":  float64(1),
			"modifiedCount": float64(1),
		})
	})
	defer server.Close()

	params := baseMongoParams(server.URL)
	params["operation"] = "update"
	params["filter"] = map[string]interface{}{"name": "Alice"}
	params["update"] = map[string]interface{}{"$set": map[string]interface{}{"age": 31}}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Equal(t, float64(1), result[0].JSON["modifiedCount"])
}

func TestMongoDBNode_Delete(t *testing.T) {
	server, node := setupMongoServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/action/deleteMany")
		json.NewEncoder(w).Encode(map[string]interface{}{"deletedCount": float64(1)})
	})
	defer server.Close()

	params := baseMongoParams(server.URL)
	params["operation"] = "delete"
	params["filter"] = map[string]interface{}{"name": "Alice"}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Equal(t, float64(1), result[0].JSON["deletedCount"])
}

func TestMongoDBNode_Aggregate(t *testing.T) {
	server, node := setupMongoServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/action/aggregate")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"documents": []interface{}{
				map[string]interface{}{"_id": "group1", "total": float64(10)},
			},
		})
	})
	defer server.Close()

	params := baseMongoParams(server.URL)
	params["operation"] = "aggregate"
	params["pipeline"] = []interface{}{
		map[string]interface{}{"$group": map[string]interface{}{"_id": "$type", "total": map[string]interface{}{"$sum": 1}}},
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.NotNil(t, result[0].JSON["documents"])
}

func TestMongoDBNode_Count(t *testing.T) {
	server, node := setupMongoServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"documents": []interface{}{
				map[string]interface{}{"count": float64(42)},
			},
		})
	})
	defer server.Close()

	params := baseMongoParams(server.URL)
	params["operation"] = "count"

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMongoDBNode_MissingRequired(t *testing.T) {
	node := NewMongoDBNode()

	// Missing database
	_, err := node.Execute(nil, map[string]interface{}{
		"operation":  "find",
		"collection": "test",
	})
	assert.Error(t, err)

	// Missing collection
	_, err = node.Execute(nil, map[string]interface{}{
		"operation": "find",
		"database":  "testdb",
	})
	assert.Error(t, err)

	// Update missing filter
	_, err = node.Execute(nil, map[string]interface{}{
		"operation":  "update",
		"database":   "testdb",
		"collection": "test",
	})
	assert.Error(t, err)

	// Delete missing filter
	_, err = node.Execute(nil, map[string]interface{}{
		"operation":  "delete",
		"database":   "testdb",
		"collection": "test",
	})
	assert.Error(t, err)
}

func TestMongoDBNode_Validate(t *testing.T) {
	node := NewMongoDBNode()

	assert.Error(t, node.ValidateParameters(nil))
	assert.Error(t, node.ValidateParameters(map[string]interface{}{"operation": "invalid"}))
	assert.Error(t, node.ValidateParameters(map[string]interface{}{"operation": "find"}))
	assert.Error(t, node.ValidateParameters(map[string]interface{}{"operation": "find", "database": "db"}))
	assert.NoError(t, node.ValidateParameters(map[string]interface{}{
		"operation":  "find",
		"database":   "db",
		"collection": "col",
	}))
}

func TestMongoDBNode_Description(t *testing.T) {
	node := NewMongoDBNode()
	desc := node.Description()
	assert.Equal(t, "MongoDB", desc.Name)
	assert.Equal(t, "Database", desc.Category)
}
