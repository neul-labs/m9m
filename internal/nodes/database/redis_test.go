package database

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRedisServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *RedisNode) {
	t.Helper()
	server := httptest.NewServer(handler)
	node := NewRedisNodeWithClient(server.Client())
	return server, node
}

func TestRedisNode_Get(t *testing.T) {
	server, node := setupRedisServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode("hello")
	})
	defer server.Close()

	params := map[string]interface{}{
		"baseUrl":   server.URL,
		"operation": "get",
		"key":       "mykey",
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "hello", result[0].JSON["result"])
}

func TestRedisNode_Set(t *testing.T) {
	server, node := setupRedisServer(t, func(w http.ResponseWriter, r *http.Request) {
		var cmd []interface{}
		json.NewDecoder(r.Body).Decode(&cmd)
		assert.Equal(t, "SET", cmd[0])
		assert.Equal(t, "mykey", cmd[1])
		assert.Equal(t, "myvalue", cmd[2])
		json.NewEncoder(w).Encode("OK")
	})
	defer server.Close()

	params := map[string]interface{}{
		"baseUrl":   server.URL,
		"operation": "set",
		"key":       "mykey",
		"value":     "myvalue",
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Equal(t, "OK", result[0].JSON["result"])
}

func TestRedisNode_Delete(t *testing.T) {
	server, node := setupRedisServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(float64(1))
	})
	defer server.Close()

	params := map[string]interface{}{
		"baseUrl":   server.URL,
		"operation": "delete",
		"key":       "mykey",
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Equal(t, float64(1), result[0].JSON["result"])
}

func TestRedisNode_Keys(t *testing.T) {
	server, node := setupRedisServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]string{"key1", "key2"})
	})
	defer server.Close()

	params := map[string]interface{}{
		"baseUrl":   server.URL,
		"operation": "keys",
		"pattern":   "*",
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.NotNil(t, result[0].JSON["result"])
}

func TestRedisNode_HashOps(t *testing.T) {
	server, node := setupRedisServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode("OK")
	})
	defer server.Close()

	t.Run("hset", func(t *testing.T) {
		params := map[string]interface{}{
			"baseUrl":   server.URL,
			"operation": "hset",
			"key":       "myhash",
			"field":     "myfield",
			"value":     "myvalue",
		}
		result, err := node.Execute(nil, params)
		require.NoError(t, err)
		assert.Equal(t, "OK", result[0].JSON["result"])
	})

	t.Run("hget", func(t *testing.T) {
		params := map[string]interface{}{
			"baseUrl":   server.URL,
			"operation": "hget",
			"key":       "myhash",
			"field":     "myfield",
		}
		result, err := node.Execute(nil, params)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestRedisNode_ListOps(t *testing.T) {
	server, node := setupRedisServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(float64(1))
	})
	defer server.Close()

	t.Run("lpush", func(t *testing.T) {
		params := map[string]interface{}{
			"baseUrl":   server.URL,
			"operation": "lpush",
			"key":       "mylist",
			"value":     "item1",
		}
		_, err := node.Execute(nil, params)
		require.NoError(t, err)
	})

	t.Run("rpush", func(t *testing.T) {
		params := map[string]interface{}{
			"baseUrl":   server.URL,
			"operation": "rpush",
			"key":       "mylist",
			"value":     "item2",
		}
		_, err := node.Execute(nil, params)
		require.NoError(t, err)
	})
}

func TestRedisNode_LRange(t *testing.T) {
	server, node := setupRedisServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]string{"a", "b", "c"})
	})
	defer server.Close()

	params := map[string]interface{}{
		"baseUrl":   server.URL,
		"operation": "lrange",
		"key":       "mylist",
		"start":     0,
		"stop":      -1,
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.NotNil(t, result[0].JSON["result"])
}

func TestRedisNode_Incr(t *testing.T) {
	server, node := setupRedisServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(float64(5))
	})
	defer server.Close()

	params := map[string]interface{}{
		"baseUrl":   server.URL,
		"operation": "incr",
		"key":       "counter",
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Equal(t, float64(5), result[0].JSON["result"])
}

func TestRedisNode_Expire(t *testing.T) {
	server, node := setupRedisServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(float64(1))
	})
	defer server.Close()

	params := map[string]interface{}{
		"baseUrl":   server.URL,
		"operation": "expire",
		"key":       "mykey",
		"seconds":   300,
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Equal(t, float64(1), result[0].JSON["result"])
}

func TestRedisNode_MissingKey(t *testing.T) {
	node := NewRedisNode()

	_, err := node.Execute(nil, map[string]interface{}{"operation": "get"})
	assert.Error(t, err)

	_, err = node.Execute(nil, map[string]interface{}{"operation": "set"})
	assert.Error(t, err)

	_, err = node.Execute(nil, map[string]interface{}{"operation": "set", "key": "k"})
	assert.Error(t, err) // missing value
}

func TestRedisNode_Validate(t *testing.T) {
	node := NewRedisNode()

	assert.Error(t, node.ValidateParameters(nil))
	assert.NoError(t, node.ValidateParameters(map[string]interface{}{"operation": "get"}))
	assert.NoError(t, node.ValidateParameters(map[string]interface{}{"operation": "set"}))
	assert.Error(t, node.ValidateParameters(map[string]interface{}{"operation": "invalid"}))
}

func TestRedisNode_Description(t *testing.T) {
	node := NewRedisNode()
	desc := node.Description()
	assert.Equal(t, "Redis", desc.Name)
	assert.Equal(t, "Database", desc.Category)
}
