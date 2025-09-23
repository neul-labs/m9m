package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/n8n-go/internal/core/base"
	"github.com/n8n-go/internal/core/interfaces"
)

// RedisNode provides Redis database operations
type RedisNode struct {
	*base.BaseNode
	client *redis.Client
}

// NewRedisNode creates a new Redis node
func NewRedisNode() *RedisNode {
	return &RedisNode{
		BaseNode: base.NewBaseNode("Redis", "Redis Operations"),
	}
}

// GetMetadata returns the node metadata
func (n *RedisNode) GetMetadata() interfaces.NodeMetadata {
	return interfaces.NodeMetadata{
		Name:        "Redis",
		DisplayName: "Redis",
		Description: "Get, set, and manipulate data in Redis",
		Group:       []string{"Database"},
		Version:     1,
		Inputs:      []string{"main"},
		Outputs:     []string{"main"},
		Credentials: []interfaces.CredentialType{
			{
				Name:        "redis",
				Required:    true,
				DisplayName: "Redis",
			},
		},
		Properties: []interfaces.NodeProperty{
			{
				Name:        "operation",
				DisplayName: "Operation",
				Type:        "options",
				Options: []interfaces.OptionItem{
					// Key operations
					{Name: "Get", Value: "get"},
					{Name: "Set", Value: "set"},
					{Name: "Delete", Value: "delete"},
					{Name: "Exists", Value: "exists"},
					{Name: "Keys", Value: "keys"},
					{Name: "Expire", Value: "expire"},
					{Name: "TTL", Value: "ttl"},
					// String operations
					{Name: "Increment", Value: "incr"},
					{Name: "Decrement", Value: "decr"},
					{Name: "Append", Value: "append"},
					// Hash operations
					{Name: "Hash Set", Value: "hset"},
					{Name: "Hash Get", Value: "hget"},
					{Name: "Hash Get All", Value: "hgetall"},
					{Name: "Hash Delete", Value: "hdel"},
					{Name: "Hash Keys", Value: "hkeys"},
					{Name: "Hash Values", Value: "hvals"},
					// List operations
					{Name: "List Push", Value: "lpush"},
					{Name: "List Pop", Value: "lpop"},
					{Name: "List Range", Value: "lrange"},
					{Name: "List Length", Value: "llen"},
					// Set operations
					{Name: "Set Add", Value: "sadd"},
					{Name: "Set Remove", Value: "srem"},
					{Name: "Set Members", Value: "smembers"},
					{Name: "Set Is Member", Value: "sismember"},
					// Sorted Set operations
					{Name: "ZSet Add", Value: "zadd"},
					{Name: "ZSet Range", Value: "zrange"},
					{Name: "ZSet Score", Value: "zscore"},
					{Name: "ZSet Remove", Value: "zrem"},
					// Pub/Sub operations
					{Name: "Publish", Value: "publish"},
					// Info operations
					{Name: "Info", Value: "info"},
					{Name: "Ping", Value: "ping"},
				},
				Default:     "get",
				Required:    true,
				Description: "The operation to perform",
			},
			{
				Name:        "key",
				DisplayName: "Key",
				Type:        "string",
				Default:     "",
				Description: "The Redis key",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{
							"get", "set", "delete", "exists", "expire", "ttl", "incr", "decr", "append",
							"hset", "hget", "hgetall", "hdel", "hkeys", "hvals",
							"lpush", "lpop", "lrange", "llen",
							"sadd", "srem", "smembers", "sismember",
							"zadd", "zrange", "zscore", "zrem",
						},
					},
				},
			},
			{
				Name:        "value",
				DisplayName: "Value",
				Type:        "string",
				Default:     "",
				Description: "The value to set",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"set", "append", "lpush", "sadd", "sismember"},
					},
				},
			},
			{
				Name:        "field",
				DisplayName: "Field",
				Type:        "string",
				Default:     "",
				Description: "Hash field name",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"hset", "hget", "hdel"},
					},
				},
			},
			{
				Name:        "ttl",
				DisplayName: "TTL (seconds)",
				Type:        "number",
				Default:     0,
				Description: "Time to live in seconds (0 = no expiration)",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"set", "expire"},
					},
				},
			},
			{
				Name:        "pattern",
				DisplayName: "Pattern",
				Type:        "string",
				Default:     "*",
				Description: "Pattern for key matching",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"keys"},
					},
				},
			},
			{
				Name:        "start",
				DisplayName: "Start Index",
				Type:        "number",
				Default:     0,
				Description: "Start index for range operations",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"lrange", "zrange"},
					},
				},
			},
			{
				Name:        "stop",
				DisplayName: "Stop Index",
				Type:        "number",
				Default:     -1,
				Description: "Stop index for range operations (-1 = end)",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"lrange", "zrange"},
					},
				},
			},
			{
				Name:        "score",
				DisplayName: "Score",
				Type:        "number",
				Default:     0,
				Description: "Score for sorted set operations",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"zadd"},
					},
				},
			},
			{
				Name:        "channel",
				DisplayName: "Channel",
				Type:        "string",
				Default:     "",
				Description: "Pub/Sub channel name",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"publish"},
					},
				},
			},
			{
				Name:        "message",
				DisplayName: "Message",
				Type:        "string",
				Default:     "",
				Description: "Message to publish",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"publish"},
					},
				},
			},
			{
				Name:        "keyType",
				DisplayName: "Auto-detect Value Type",
				Type:        "boolean",
				Default:     true,
				Description: "Automatically detect if value is JSON",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"set"},
					},
				},
			},
		},
	}
}

// Execute runs the Redis operation
func (n *RedisNode) Execute(ctx context.Context, params interfaces.ExecutionParams) (interfaces.NodeOutput, error) {
	// Get credentials
	credentials, err := params.GetCredentials("redis")
	if err != nil {
		return interfaces.NodeOutput{}, fmt.Errorf("failed to get Redis credentials: %w", err)
	}

	// Connect to Redis
	if err := n.connect(credentials); err != nil {
		return interfaces.NodeOutput{}, err
	}
	defer n.disconnect()

	// Get operation
	operation := params.GetNodeParameter("operation", "get").(string)

	// Execute operation
	var result interface{}
	switch operation {
	// Key operations
	case "get":
		result, err = n.executeGet(params)
	case "set":
		result, err = n.executeSet(params)
	case "delete":
		result, err = n.executeDelete(params)
	case "exists":
		result, err = n.executeExists(params)
	case "keys":
		result, err = n.executeKeys(params)
	case "expire":
		result, err = n.executeExpire(params)
	case "ttl":
		result, err = n.executeTTL(params)
	// String operations
	case "incr":
		result, err = n.executeIncr(params)
	case "decr":
		result, err = n.executeDecr(params)
	case "append":
		result, err = n.executeAppend(params)
	// Hash operations
	case "hset":
		result, err = n.executeHSet(params)
	case "hget":
		result, err = n.executeHGet(params)
	case "hgetall":
		result, err = n.executeHGetAll(params)
	case "hdel":
		result, err = n.executeHDel(params)
	case "hkeys":
		result, err = n.executeHKeys(params)
	case "hvals":
		result, err = n.executeHVals(params)
	// List operations
	case "lpush":
		result, err = n.executeLPush(params)
	case "lpop":
		result, err = n.executeLPop(params)
	case "lrange":
		result, err = n.executeLRange(params)
	case "llen":
		result, err = n.executeLLen(params)
	// Set operations
	case "sadd":
		result, err = n.executeSAdd(params)
	case "srem":
		result, err = n.executeSRem(params)
	case "smembers":
		result, err = n.executeSMembers(params)
	case "sismember":
		result, err = n.executeSIsMember(params)
	// Sorted set operations
	case "zadd":
		result, err = n.executeZAdd(params)
	case "zrange":
		result, err = n.executeZRange(params)
	case "zscore":
		result, err = n.executeZScore(params)
	case "zrem":
		result, err = n.executeZRem(params)
	// Pub/Sub operations
	case "publish":
		result, err = n.executePublish(params)
	// Info operations
	case "info":
		result, err = n.executeInfo()
	case "ping":
		result, err = n.executePing()
	default:
		err = fmt.Errorf("unsupported operation: %s", operation)
	}

	if err != nil {
		return interfaces.NodeOutput{}, err
	}

	// Format output
	var outputItems []interfaces.ItemData
	switch v := result.(type) {
	case []string:
		for i, item := range v {
			outputItems = append(outputItems, interfaces.ItemData{
				JSON: map[string]interface{}{
					"value": item,
				},
				Index: i,
			})
		}
	case map[string]string:
		outputItems = append(outputItems, interfaces.ItemData{
			JSON: map[string]interface{}(n.convertStringMap(v)),
			Index: 0,
		})
	case string:
		// Try to parse as JSON
		var jsonData interface{}
		if err := json.Unmarshal([]byte(v), &jsonData); err == nil {
			outputItems = append(outputItems, interfaces.ItemData{
				JSON: map[string]interface{}{
					"data": jsonData,
				},
				Index: 0,
			})
		} else {
			outputItems = append(outputItems, interfaces.ItemData{
				JSON: map[string]interface{}{
					"value": v,
				},
				Index: 0,
			})
		}
	default:
		outputItems = append(outputItems, interfaces.ItemData{
			JSON: map[string]interface{}{
				"result": result,
			},
			Index: 0,
		})
	}

	return interfaces.NodeOutput{
		Items: outputItems,
	}, nil
}

// Connection methods

func (n *RedisNode) connect(credentials map[string]interface{}) error {
	host := "localhost"
	if h, ok := credentials["host"].(string); ok {
		host = h
	}

	port := 6379
	if p, ok := credentials["port"].(float64); ok {
		port = int(p)
	}

	password := ""
	if p, ok := credentials["password"].(string); ok {
		password = p
	}

	db := 0
	if d, ok := credentials["database"].(float64); ok {
		db = int(d)
	}

	n.client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: password,
		DB:       db,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	if err := n.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return nil
}

func (n *RedisNode) disconnect() {
	if n.client != nil {
		n.client.Close()
	}
}

// Operation implementations

func (n *RedisNode) executeGet(params interfaces.ExecutionParams) (string, error) {
	key := params.GetNodeParameter("key", "").(string)
	if key == "" {
		return "", fmt.Errorf("key is required")
	}

	ctx := context.Background()
	val, err := n.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

func (n *RedisNode) executeSet(params interfaces.ExecutionParams) (string, error) {
	key := params.GetNodeParameter("key", "").(string)
	value := params.GetNodeParameter("value", "").(string)
	ttl := int(params.GetNodeParameter("ttl", 0).(float64))
	autoDetect := params.GetNodeParameter("keyType", true).(bool)

	if key == "" {
		return "", fmt.Errorf("key is required")
	}

	// Auto-detect and stringify JSON if needed
	if autoDetect {
		if data, ok := params.GetNodeParameter("value", nil).(map[string]interface{}); ok {
			jsonBytes, _ := json.Marshal(data)
			value = string(jsonBytes)
		}
	}

	ctx := context.Background()
	var expiration time.Duration
	if ttl > 0 {
		expiration = time.Duration(ttl) * time.Second
	}

	err := n.client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		return "", err
	}
	return "OK", nil
}

func (n *RedisNode) executeDelete(params interfaces.ExecutionParams) (int64, error) {
	key := params.GetNodeParameter("key", "").(string)
	if key == "" {
		return 0, fmt.Errorf("key is required")
	}

	ctx := context.Background()
	return n.client.Del(ctx, key).Result()
}

func (n *RedisNode) executeExists(params interfaces.ExecutionParams) (bool, error) {
	key := params.GetNodeParameter("key", "").(string)
	if key == "" {
		return false, fmt.Errorf("key is required")
	}

	ctx := context.Background()
	result, err := n.client.Exists(ctx, key).Result()
	return result > 0, err
}

func (n *RedisNode) executeKeys(params interfaces.ExecutionParams) ([]string, error) {
	pattern := params.GetNodeParameter("pattern", "*").(string)

	ctx := context.Background()
	return n.client.Keys(ctx, pattern).Result()
}

func (n *RedisNode) executeExpire(params interfaces.ExecutionParams) (bool, error) {
	key := params.GetNodeParameter("key", "").(string)
	ttl := int(params.GetNodeParameter("ttl", 0).(float64))

	if key == "" {
		return false, fmt.Errorf("key is required")
	}

	ctx := context.Background()
	expiration := time.Duration(ttl) * time.Second
	return n.client.Expire(ctx, key, expiration).Result()
}

func (n *RedisNode) executeTTL(params interfaces.ExecutionParams) (int64, error) {
	key := params.GetNodeParameter("key", "").(string)
	if key == "" {
		return -1, fmt.Errorf("key is required")
	}

	ctx := context.Background()
	duration, err := n.client.TTL(ctx, key).Result()
	if err != nil {
		return -1, err
	}
	return int64(duration.Seconds()), nil
}

func (n *RedisNode) executeIncr(params interfaces.ExecutionParams) (int64, error) {
	key := params.GetNodeParameter("key", "").(string)
	if key == "" {
		return 0, fmt.Errorf("key is required")
	}

	ctx := context.Background()
	return n.client.Incr(ctx, key).Result()
}

func (n *RedisNode) executeDecr(params interfaces.ExecutionParams) (int64, error) {
	key := params.GetNodeParameter("key", "").(string)
	if key == "" {
		return 0, fmt.Errorf("key is required")
	}

	ctx := context.Background()
	return n.client.Decr(ctx, key).Result()
}

func (n *RedisNode) executeAppend(params interfaces.ExecutionParams) (int64, error) {
	key := params.GetNodeParameter("key", "").(string)
	value := params.GetNodeParameter("value", "").(string)

	if key == "" {
		return 0, fmt.Errorf("key is required")
	}

	ctx := context.Background()
	return n.client.Append(ctx, key, value).Result()
}

// Hash operations

func (n *RedisNode) executeHSet(params interfaces.ExecutionParams) (int64, error) {
	key := params.GetNodeParameter("key", "").(string)
	field := params.GetNodeParameter("field", "").(string)
	value := params.GetNodeParameter("value", "").(string)

	if key == "" || field == "" {
		return 0, fmt.Errorf("key and field are required")
	}

	ctx := context.Background()
	return n.client.HSet(ctx, key, field, value).Result()
}

func (n *RedisNode) executeHGet(params interfaces.ExecutionParams) (string, error) {
	key := params.GetNodeParameter("key", "").(string)
	field := params.GetNodeParameter("field", "").(string)

	if key == "" || field == "" {
		return "", fmt.Errorf("key and field are required")
	}

	ctx := context.Background()
	return n.client.HGet(ctx, key, field).Result()
}

func (n *RedisNode) executeHGetAll(params interfaces.ExecutionParams) (map[string]string, error) {
	key := params.GetNodeParameter("key", "").(string)
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}

	ctx := context.Background()
	return n.client.HGetAll(ctx, key).Result()
}

func (n *RedisNode) executeHDel(params interfaces.ExecutionParams) (int64, error) {
	key := params.GetNodeParameter("key", "").(string)
	field := params.GetNodeParameter("field", "").(string)

	if key == "" || field == "" {
		return 0, fmt.Errorf("key and field are required")
	}

	ctx := context.Background()
	return n.client.HDel(ctx, key, field).Result()
}

func (n *RedisNode) executeHKeys(params interfaces.ExecutionParams) ([]string, error) {
	key := params.GetNodeParameter("key", "").(string)
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}

	ctx := context.Background()
	return n.client.HKeys(ctx, key).Result()
}

func (n *RedisNode) executeHVals(params interfaces.ExecutionParams) ([]string, error) {
	key := params.GetNodeParameter("key", "").(string)
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}

	ctx := context.Background()
	return n.client.HVals(ctx, key).Result()
}

// List operations

func (n *RedisNode) executeLPush(params interfaces.ExecutionParams) (int64, error) {
	key := params.GetNodeParameter("key", "").(string)
	value := params.GetNodeParameter("value", "").(string)

	if key == "" {
		return 0, fmt.Errorf("key is required")
	}

	ctx := context.Background()
	return n.client.LPush(ctx, key, value).Result()
}

func (n *RedisNode) executeLPop(params interfaces.ExecutionParams) (string, error) {
	key := params.GetNodeParameter("key", "").(string)
	if key == "" {
		return "", fmt.Errorf("key is required")
	}

	ctx := context.Background()
	return n.client.LPop(ctx, key).Result()
}

func (n *RedisNode) executeLRange(params interfaces.ExecutionParams) ([]string, error) {
	key := params.GetNodeParameter("key", "").(string)
	start := int64(params.GetNodeParameter("start", 0).(float64))
	stop := int64(params.GetNodeParameter("stop", -1).(float64))

	if key == "" {
		return nil, fmt.Errorf("key is required")
	}

	ctx := context.Background()
	return n.client.LRange(ctx, key, start, stop).Result()
}

func (n *RedisNode) executeLLen(params interfaces.ExecutionParams) (int64, error) {
	key := params.GetNodeParameter("key", "").(string)
	if key == "" {
		return 0, fmt.Errorf("key is required")
	}

	ctx := context.Background()
	return n.client.LLen(ctx, key).Result()
}

// Set operations

func (n *RedisNode) executeSAdd(params interfaces.ExecutionParams) (int64, error) {
	key := params.GetNodeParameter("key", "").(string)
	value := params.GetNodeParameter("value", "").(string)

	if key == "" {
		return 0, fmt.Errorf("key is required")
	}

	ctx := context.Background()
	return n.client.SAdd(ctx, key, value).Result()
}

func (n *RedisNode) executeSRem(params interfaces.ExecutionParams) (int64, error) {
	key := params.GetNodeParameter("key", "").(string)
	value := params.GetNodeParameter("value", "").(string)

	if key == "" {
		return 0, fmt.Errorf("key is required")
	}

	ctx := context.Background()
	return n.client.SRem(ctx, key, value).Result()
}

func (n *RedisNode) executeSMembers(params interfaces.ExecutionParams) ([]string, error) {
	key := params.GetNodeParameter("key", "").(string)
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}

	ctx := context.Background()
	return n.client.SMembers(ctx, key).Result()
}

func (n *RedisNode) executeSIsMember(params interfaces.ExecutionParams) (bool, error) {
	key := params.GetNodeParameter("key", "").(string)
	value := params.GetNodeParameter("value", "").(string)

	if key == "" {
		return false, fmt.Errorf("key is required")
	}

	ctx := context.Background()
	return n.client.SIsMember(ctx, key, value).Result()
}

// Sorted set operations

func (n *RedisNode) executeZAdd(params interfaces.ExecutionParams) (int64, error) {
	key := params.GetNodeParameter("key", "").(string)
	value := params.GetNodeParameter("value", "").(string)
	score := params.GetNodeParameter("score", 0).(float64)

	if key == "" {
		return 0, fmt.Errorf("key is required")
	}

	ctx := context.Background()
	return n.client.ZAdd(ctx, key, &redis.Z{
		Score:  score,
		Member: value,
	}).Result()
}

func (n *RedisNode) executeZRange(params interfaces.ExecutionParams) ([]string, error) {
	key := params.GetNodeParameter("key", "").(string)
	start := int64(params.GetNodeParameter("start", 0).(float64))
	stop := int64(params.GetNodeParameter("stop", -1).(float64))

	if key == "" {
		return nil, fmt.Errorf("key is required")
	}

	ctx := context.Background()
	return n.client.ZRange(ctx, key, start, stop).Result()
}

func (n *RedisNode) executeZScore(params interfaces.ExecutionParams) (float64, error) {
	key := params.GetNodeParameter("key", "").(string)
	value := params.GetNodeParameter("value", "").(string)

	if key == "" {
		return 0, fmt.Errorf("key is required")
	}

	ctx := context.Background()
	return n.client.ZScore(ctx, key, value).Result()
}

func (n *RedisNode) executeZRem(params interfaces.ExecutionParams) (int64, error) {
	key := params.GetNodeParameter("key", "").(string)
	value := params.GetNodeParameter("value", "").(string)

	if key == "" {
		return 0, fmt.Errorf("key is required")
	}

	ctx := context.Background()
	return n.client.ZRem(ctx, key, value).Result()
}

// Pub/Sub operations

func (n *RedisNode) executePublish(params interfaces.ExecutionParams) (int64, error) {
	channel := params.GetNodeParameter("channel", "").(string)
	message := params.GetNodeParameter("message", "").(string)

	if channel == "" {
		return 0, fmt.Errorf("channel is required")
	}

	ctx := context.Background()
	return n.client.Publish(ctx, channel, message).Result()
}

// Info operations

func (n *RedisNode) executeInfo() (string, error) {
	ctx := context.Background()
	return n.client.Info(ctx).Result()
}

func (n *RedisNode) executePing() (string, error) {
	ctx := context.Background()
	return n.client.Ping(ctx).Result()
}

// Helper methods

func (n *RedisNode) convertStringMap(m map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = v
	}
	return result
}

// Clone creates a copy of the node
func (n *RedisNode) Clone() interfaces.Node {
	return &RedisNode{
		BaseNode: n.BaseNode.Clone(),
	}
}