package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dipankar/m9m/internal/model"
	"github.com/dipankar/m9m/internal/nodes/base"
)

// AnthropicNode interacts with Anthropic (Claude) API
type AnthropicNode struct {
	*base.BaseNode
	httpClient *http.Client
}

// NewAnthropicNode creates a new Anthropic node
func NewAnthropicNode() *AnthropicNode {
	return &AnthropicNode{
		BaseNode: base.NewBaseNode(base.NodeDescription{
			Name:        "Anthropic",
			Description: "Use Anthropic's Claude models for text generation and analysis",
			Category:    "ai",
		}),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Execute processes input with Anthropic API
func (n *AnthropicNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	// Get parameters
	apiKey := n.GetStringParameter(nodeParams, "apiKey", "")
	modelName := n.GetStringParameter(nodeParams, "model", "claude-3-5-sonnet-20241022")
	prompt := n.GetStringParameter(nodeParams, "prompt", "")
	maxTokens := n.GetIntParameter(nodeParams, "maxTokens", 1024)
	temperature := nodeParams["temperature"]
	if temperature == nil {
		temperature = 1.0
	}

	if apiKey == "" {
		return nil, fmt.Errorf("apiKey is required")
	}

	if prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	var results []model.DataItem

	for range inputData {
		// Prepare request payload for Anthropic API
		messages := []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		}

		payload := map[string]interface{}{
			"model":      modelName,
			"messages":   messages,
			"max_tokens": maxTokens,
			"temperature": temperature,
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %v", err)
		}

		// Make API request
		req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %v", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		resp, err := n.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("anthropic API returned status %d", resp.StatusCode)
		}

		// Parse response
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %v", err)
		}

		// Extract the generated text
		content, ok := result["content"].([]interface{})
		if !ok || len(content) == 0 {
			return nil, fmt.Errorf("no content in Anthropic response")
		}

		firstContent := content[0].(map[string]interface{})
		text := firstContent["text"].(string)

		// Create result item
		resultItem := model.DataItem{
			JSON: map[string]interface{}{
				"prompt":      prompt,
				"response":    text,
				"model":       modelName,
				"usage":       result["usage"],
				"stopReason":  result["stop_reason"],
			},
		}

		results = append(results, resultItem)
	}

	return results, nil
}
