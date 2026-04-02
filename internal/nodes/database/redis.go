package database

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// RedisHTTPClient abstracts the HTTP client for testing.
// This implementation uses the Redis REST-like interface (e.g. via a sidecar or
// a REST-to-Redis gateway). For native driver support, swap this for go-redis.
type RedisHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// RedisNode implements Redis operations.
// It communicates via an HTTP REST endpoint (baseUrl) that proxies to Redis.
// The default baseUrl can be a Redis REST API like Upstash or a local proxy.
type RedisNode struct {
	*base.BaseNode
	client RedisHTTPClient
}

// NewRedisNode creates a new Redis node.
func NewRedisNode() *RedisNode {
	return &RedisNode{
		BaseNode: base.NewBaseNode(base.NodeDescription{
			Name:        "Redis",
			Description: "Execute Redis commands",
			Category:    "Database",
		}),
		client: &http.Client{},
	}
}

// NewRedisNodeWithClient creates a Redis node with a custom HTTP client.
func NewRedisNodeWithClient(client RedisHTTPClient) *RedisNode {
	n := NewRedisNode()
	n.client = client
	return n
}

// Execute runs the configured Redis operation.
func (n *RedisNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	baseURL := n.GetStringParameter(nodeParams, "baseUrl", "http://localhost:6380")
	baseURL = strings.TrimRight(baseURL, "/")
	operation := n.GetStringParameter(nodeParams, "operation", "get")

	switch operation {
	case "get":
		return n.doCommand(baseURL, nodeParams, "GET", "key")
	case "set":
		return n.doSetCommand(baseURL, nodeParams)
	case "delete":
		return n.doCommand(baseURL, nodeParams, "DEL", "key")
	case "keys":
		return n.doCommand(baseURL, nodeParams, "KEYS", "pattern")
	case "hget":
		return n.doHashCommand(baseURL, nodeParams, "HGET")
	case "hset":
		return n.doHashSetCommand(baseURL, nodeParams)
	case "lpush":
		return n.doListCommand(baseURL, nodeParams, "LPUSH")
	case "rpush":
		return n.doListCommand(baseURL, nodeParams, "RPUSH")
	case "lrange":
		return n.doLRange(baseURL, nodeParams)
	case "incr":
		return n.doCommand(baseURL, nodeParams, "INCR", "key")
	case "expire":
		return n.doExpire(baseURL, nodeParams)
	default:
		return nil, n.CreateError(fmt.Sprintf("unsupported operation: %s", operation), nil)
	}
}

func (n *RedisNode) doCommand(baseURL string, params map[string]interface{}, cmd, keyParam string) ([]model.DataItem, error) {
	key := n.GetStringParameter(params, keyParam, "")
	if key == "" {
		return nil, n.CreateError(fmt.Sprintf("%s is required for %s", keyParam, cmd), nil)
	}
	return n.sendCommand(baseURL, params, []interface{}{cmd, key})
}

func (n *RedisNode) doSetCommand(baseURL string, params map[string]interface{}) ([]model.DataItem, error) {
	key := n.GetStringParameter(params, "key", "")
	if key == "" {
		return nil, n.CreateError("key is required for SET", nil)
	}
	value, ok := params["value"]
	if !ok {
		return nil, n.CreateError("value is required for SET", nil)
	}
	return n.sendCommand(baseURL, params, []interface{}{"SET", key, value})
}

func (n *RedisNode) doHashCommand(baseURL string, params map[string]interface{}, cmd string) ([]model.DataItem, error) {
	key := n.GetStringParameter(params, "key", "")
	field := n.GetStringParameter(params, "field", "")
	if key == "" || field == "" {
		return nil, n.CreateError("key and field are required for "+cmd, nil)
	}
	return n.sendCommand(baseURL, params, []interface{}{cmd, key, field})
}

func (n *RedisNode) doHashSetCommand(baseURL string, params map[string]interface{}) ([]model.DataItem, error) {
	key := n.GetStringParameter(params, "key", "")
	field := n.GetStringParameter(params, "field", "")
	if key == "" || field == "" {
		return nil, n.CreateError("key and field are required for HSET", nil)
	}
	value, ok := params["value"]
	if !ok {
		return nil, n.CreateError("value is required for HSET", nil)
	}
	return n.sendCommand(baseURL, params, []interface{}{"HSET", key, field, value})
}

func (n *RedisNode) doListCommand(baseURL string, params map[string]interface{}, cmd string) ([]model.DataItem, error) {
	key := n.GetStringParameter(params, "key", "")
	if key == "" {
		return nil, n.CreateError("key is required for "+cmd, nil)
	}
	value, ok := params["value"]
	if !ok {
		return nil, n.CreateError("value is required for "+cmd, nil)
	}
	return n.sendCommand(baseURL, params, []interface{}{cmd, key, value})
}

func (n *RedisNode) doLRange(baseURL string, params map[string]interface{}) ([]model.DataItem, error) {
	key := n.GetStringParameter(params, "key", "")
	if key == "" {
		return nil, n.CreateError("key is required for LRANGE", nil)
	}
	start := n.GetIntParameter(params, "start", 0)
	stop := n.GetIntParameter(params, "stop", -1)
	return n.sendCommand(baseURL, params, []interface{}{"LRANGE", key, start, stop})
}

func (n *RedisNode) doExpire(baseURL string, params map[string]interface{}) ([]model.DataItem, error) {
	key := n.GetStringParameter(params, "key", "")
	if key == "" {
		return nil, n.CreateError("key is required for EXPIRE", nil)
	}
	seconds := n.GetIntParameter(params, "seconds", 60)
	return n.sendCommand(baseURL, params, []interface{}{"EXPIRE", key, seconds})
}

func (n *RedisNode) sendCommand(baseURL string, params map[string]interface{}, args []interface{}) ([]model.DataItem, error) {
	body, err := json.Marshal(args)
	if err != nil {
		return nil, n.CreateError(fmt.Sprintf("failed to marshal command: %v", err), nil)
	}

	req, err := http.NewRequest("POST", baseURL, strings.NewReader(string(body)))
	if err != nil {
		return nil, n.CreateError(fmt.Sprintf("failed to create request: %v", err), nil)
	}
	req.Header.Set("Content-Type", "application/json")

	password := n.GetStringParameter(params, "password", "")
	if password != "" {
		req.Header.Set("Authorization", "Bearer "+password)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return nil, n.CreateError(fmt.Sprintf("request failed: %v", err), nil)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, n.CreateError(fmt.Sprintf("failed to read response: %v", err), nil)
	}

	var result interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		result = string(respBody)
	}

	return []model.DataItem{{JSON: map[string]interface{}{
		"result":     result,
		"statusCode": resp.StatusCode,
	}}}, nil
}

// ValidateParameters validates Redis node parameters.
func (n *RedisNode) ValidateParameters(params map[string]interface{}) error {
	if params == nil {
		return n.CreateError("parameters cannot be nil", nil)
	}

	operation := n.GetStringParameter(params, "operation", "get")
	validOps := map[string]bool{
		"get": true, "set": true, "delete": true, "keys": true,
		"hget": true, "hset": true, "lpush": true, "rpush": true,
		"lrange": true, "incr": true, "expire": true,
	}
	if !validOps[operation] {
		return n.CreateError(fmt.Sprintf("invalid operation: %s", operation), nil)
	}

	return nil
}
