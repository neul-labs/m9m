package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/neul-labs/m9m/internal/mcp"
)

// HTTPRequestTool makes HTTP requests
type HTTPRequestTool struct {
	*BaseTool
	client *http.Client
}

// NewHTTPRequestTool creates a new HTTP request tool
func NewHTTPRequestTool() *HTTPRequestTool {
	return &HTTPRequestTool{
		BaseTool: NewBaseTool(
			"http_request",
			"Make an HTTP request to any URL. Supports GET, POST, PUT, PATCH, DELETE methods with custom headers and body.",
			ObjectSchema(map[string]interface{}{
				"url":     StringProp("The URL to request"),
				"method":  StringEnumProp("HTTP method", []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}),
				"headers": ObjectProp("HTTP headers as key-value pairs"),
				"body":    AnyProp("Request body (string or object for JSON)"),
				"timeout": IntPropWithDefault("Request timeout in milliseconds", 30000),
			}, []string{"url"}),
		),
		client: &http.Client{},
	}
}

// Execute makes an HTTP request
func (t *HTTPRequestTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	url := GetString(args, "url")
	method := GetStringOr(args, "method", "GET")
	headers := GetMap(args, "headers")
	timeout := GetIntOr(args, "timeout", 30000)

	// Create request body
	var bodyReader io.Reader
	if body := args["body"]; body != nil {
		switch b := body.(type) {
		case string:
			bodyReader = strings.NewReader(b)
		default:
			jsonBody, err := json.Marshal(b)
			if err != nil {
				return mcp.ErrorContent(fmt.Sprintf("Failed to marshal body: %v", err)), nil
			}
			bodyReader = bytes.NewReader(jsonBody)
		}
	}

	// Create context with timeout
	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
	defer cancel()

	// Create request
	req, err := http.NewRequestWithContext(reqCtx, method, url, bodyReader)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to create request: %v", err)), nil
	}

	// Set headers
	if headers != nil {
		for k, v := range headers {
			if str, ok := v.(string); ok {
				req.Header.Set(k, str)
			}
		}
	}

	// Default content type for JSON bodies
	if req.Header.Get("Content-Type") == "" && bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute request
	resp, err := t.client.Do(req)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Request failed: %v", err)), nil
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to read response: %v", err)), nil
	}

	// Try to parse as JSON
	var parsedBody interface{}
	if err := json.Unmarshal(respBody, &parsedBody); err == nil {
		return mcp.SuccessJSON(map[string]interface{}{
			"statusCode": resp.StatusCode,
			"status":     resp.Status,
			"headers":    headerToMap(resp.Header),
			"body":       parsedBody,
		}), nil
	}

	return mcp.SuccessJSON(map[string]interface{}{
		"statusCode": resp.StatusCode,
		"status":     resp.Status,
		"headers":    headerToMap(resp.Header),
		"body":       string(respBody),
	}), nil
}

func headerToMap(h http.Header) map[string]string {
	result := make(map[string]string)
	for k, v := range h {
		if len(v) > 0 {
			result[k] = v[0]
		}
	}
	return result
}

// SendSlackTool sends messages to Slack
type SendSlackTool struct {
	*BaseTool
	client *http.Client
}

// NewSendSlackTool creates a new Slack message tool
func NewSendSlackTool() *SendSlackTool {
	return &SendSlackTool{
		BaseTool: NewBaseTool(
			"send_slack",
			"Send a message to a Slack channel via webhook URL.",
			ObjectSchema(map[string]interface{}{
				"webhookUrl": StringProp("Slack webhook URL"),
				"text":       StringProp("Message text to send"),
				"channel":    StringProp("Channel to send to (optional, uses webhook default)"),
				"username":   StringProp("Bot username (optional)"),
			}, []string{"webhookUrl", "text"}),
		),
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Execute sends a Slack message
func (t *SendSlackTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	webhookUrl := GetString(args, "webhookUrl")
	text := GetString(args, "text")
	channel := GetString(args, "channel")
	username := GetString(args, "username")

	payload := map[string]interface{}{
		"text": text,
	}
	if channel != "" {
		payload["channel"] = channel
	}
	if username != "" {
		payload["username"] = username
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to marshal payload: %v", err)), nil
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookUrl, bytes.NewReader(jsonBody))
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to create request: %v", err)), nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to send message: %v", err)), nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return mcp.ErrorContent(fmt.Sprintf("Slack returned error: %s - %s", resp.Status, string(body))), nil
	}

	return mcp.SuccessJSON(map[string]interface{}{
		"success": true,
		"message": "Message sent to Slack",
	}), nil
}

// SendDiscordTool sends messages to Discord
type SendDiscordTool struct {
	*BaseTool
	client *http.Client
}

// NewSendDiscordTool creates a new Discord message tool
func NewSendDiscordTool() *SendDiscordTool {
	return &SendDiscordTool{
		BaseTool: NewBaseTool(
			"send_discord",
			"Send a message to a Discord channel via webhook URL.",
			ObjectSchema(map[string]interface{}{
				"webhookUrl": StringProp("Discord webhook URL"),
				"content":    StringProp("Message content to send"),
				"username":   StringProp("Bot username (optional)"),
				"avatarUrl":  StringProp("Avatar URL (optional)"),
			}, []string{"webhookUrl", "content"}),
		),
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Execute sends a Discord message
func (t *SendDiscordTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	webhookUrl := GetString(args, "webhookUrl")
	content := GetString(args, "content")
	username := GetString(args, "username")
	avatarUrl := GetString(args, "avatarUrl")

	payload := map[string]interface{}{
		"content": content,
	}
	if username != "" {
		payload["username"] = username
	}
	if avatarUrl != "" {
		payload["avatar_url"] = avatarUrl
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to marshal payload: %v", err)), nil
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookUrl, bytes.NewReader(jsonBody))
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to create request: %v", err)), nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to send message: %v", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		body, _ := io.ReadAll(resp.Body)
		return mcp.ErrorContent(fmt.Sprintf("Discord returned error: %s - %s", resp.Status, string(body))), nil
	}

	return mcp.SuccessJSON(map[string]interface{}{
		"success": true,
		"message": "Message sent to Discord",
	}), nil
}

// AIOpenAITool gets completions from OpenAI
type AIOpenAITool struct {
	*BaseTool
	client *http.Client
}

// NewAIOpenAITool creates a new OpenAI tool
func NewAIOpenAITool() *AIOpenAITool {
	return &AIOpenAITool{
		BaseTool: NewBaseTool(
			"ai_openai",
			"Get a completion from OpenAI models (GPT-3.5, GPT-4, etc.).",
			ObjectSchema(map[string]interface{}{
				"apiKey":      StringProp("OpenAI API key"),
				"model":       StringPropWithDefault("Model name", "gpt-3.5-turbo"),
				"prompt":      StringProp("Prompt to send to the model"),
				"maxTokens":   IntPropWithDefault("Maximum tokens in response", 1000),
				"temperature": AnyProp("Sampling temperature (0-2, default 0.7)"),
			}, []string{"apiKey", "prompt"}),
		),
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

// Execute gets an OpenAI completion
func (t *AIOpenAITool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	apiKey := GetString(args, "apiKey")
	model := GetStringOr(args, "model", "gpt-3.5-turbo")
	prompt := GetString(args, "prompt")
	maxTokens := GetIntOr(args, "maxTokens", 1000)

	temperature := 0.7
	if temp, ok := args["temperature"].(float64); ok {
		temperature = temp
	}

	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  maxTokens,
		"temperature": temperature,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to marshal payload: %v", err)), nil
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to create request: %v", err)), nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := t.client.Do(req)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Request failed: %v", err)), nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return mcp.ErrorContent(fmt.Sprintf("OpenAI API error: %s - %s", resp.Status, string(body))), nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to parse response: %v", err)), nil
	}

	// Extract the completion text
	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				return mcp.SuccessJSON(map[string]interface{}{
					"response":     message["content"],
					"model":        model,
					"usage":        result["usage"],
					"finishReason": choice["finish_reason"],
				}), nil
			}
		}
	}

	return mcp.SuccessJSON(result), nil
}

// AIAnthropicTool gets completions from Anthropic
type AIAnthropicTool struct {
	*BaseTool
	client *http.Client
}

// NewAIAnthropicTool creates a new Anthropic tool
func NewAIAnthropicTool() *AIAnthropicTool {
	return &AIAnthropicTool{
		BaseTool: NewBaseTool(
			"ai_anthropic",
			"Get a completion from Anthropic Claude models.",
			ObjectSchema(map[string]interface{}{
				"apiKey":      StringProp("Anthropic API key"),
				"model":       StringPropWithDefault("Model name", "claude-3-5-sonnet-20241022"),
				"prompt":      StringProp("Prompt to send to the model"),
				"maxTokens":   IntPropWithDefault("Maximum tokens in response", 1000),
				"temperature": AnyProp("Sampling temperature (0-1, default 0.7)"),
			}, []string{"apiKey", "prompt"}),
		),
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

// Execute gets an Anthropic completion
func (t *AIAnthropicTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	apiKey := GetString(args, "apiKey")
	model := GetStringOr(args, "model", "claude-3-5-sonnet-20241022")
	prompt := GetString(args, "prompt")
	maxTokens := GetIntOr(args, "maxTokens", 1000)

	temperature := 0.7
	if temp, ok := args["temperature"].(float64); ok {
		temperature = temp
	}

	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  maxTokens,
		"temperature": temperature,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to marshal payload: %v", err)), nil
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to create request: %v", err)), nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := t.client.Do(req)
	if err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Request failed: %v", err)), nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return mcp.ErrorContent(fmt.Sprintf("Anthropic API error: %s - %s", resp.Status, string(body))), nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to parse response: %v", err)), nil
	}

	// Extract the completion text
	if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
		if block, ok := content[0].(map[string]interface{}); ok {
			return mcp.SuccessJSON(map[string]interface{}{
				"response":   block["text"],
				"model":      model,
				"usage":      result["usage"],
				"stopReason": result["stop_reason"],
			}), nil
		}
	}

	return mcp.SuccessJSON(result), nil
}

// TransformDataTool transforms data with expressions
type TransformDataTool struct {
	*BaseTool
}

// NewTransformDataTool creates a new transform data tool
func NewTransformDataTool() *TransformDataTool {
	return &TransformDataTool{
		BaseTool: NewBaseTool(
			"transform_data",
			"Transform data by setting or extracting fields. Useful for data manipulation.",
			ObjectSchema(map[string]interface{}{
				"data":        AnyProp("Input data object or array"),
				"assignments": ArrayProp("Array of {name, value} pairs to set", map[string]interface{}{"type": "object"}),
				"extract":     ArrayProp("Array of field names to extract", map[string]interface{}{"type": "string"}),
			}, []string{"data"}),
		),
	}
}

// Execute transforms data
func (t *TransformDataTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	data := args["data"]
	assignments := GetArray(args, "assignments")
	extract := GetArray(args, "extract")

	result := make(map[string]interface{})

	// Copy original data if it's a map
	if dataMap, ok := data.(map[string]interface{}); ok {
		for k, v := range dataMap {
			result[k] = v
		}
	}

	// Apply assignments
	if assignments != nil {
		for _, a := range assignments {
			if assignment, ok := a.(map[string]interface{}); ok {
				name := GetString(assignment, "name")
				value := assignment["value"]
				if name != "" {
					result[name] = value
				}
			}
		}
	}

	// Extract specific fields
	if extract != nil && len(extract) > 0 {
		extracted := make(map[string]interface{})
		for _, field := range extract {
			if fieldName, ok := field.(string); ok {
				if val, exists := result[fieldName]; exists {
					extracted[fieldName] = val
				}
			}
		}
		result = extracted
	}

	return mcp.SuccessJSON(map[string]interface{}{
		"result": result,
	}), nil
}

// RegisterActionTools registers all high-level action tools with a registry
func RegisterActionTools(registry *Registry) {
	registry.Register(NewHTTPRequestTool())
	registry.Register(NewSendSlackTool())
	registry.Register(NewSendDiscordTool())
	registry.Register(NewAIOpenAITool())
	registry.Register(NewAIAnthropicTool())
	registry.Register(NewTransformDataTool())
}
