package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dipankar/n8n-go/internal/model"
	"github.com/dipankar/n8n-go/internal/nodes/base"
)

// OpenAINode interacts with OpenAI API
type OpenAINode struct {
	*base.BaseNode
	httpClient *http.Client
}

// NewOpenAINode creates a new OpenAI node
func NewOpenAINode() *OpenAINode {
	return &OpenAINode{
		BaseNode: base.NewBaseNode(base.NodeDescription{
			Name:        "OpenAI",
			Description: "Use OpenAI's GPT models for text generation and completion",
			Category:    "ai",
		}),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Execute processes input with OpenAI API
func (n *OpenAINode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	// Get parameters
	apiKey := n.GetStringParameter(nodeParams, "apiKey", "")
	modelName := n.GetStringParameter(nodeParams, "model", "gpt-3.5-turbo")
	prompt := n.GetStringParameter(nodeParams, "prompt", "")
	maxTokens := n.GetIntParameter(nodeParams, "maxTokens", 1000)
	temperature := nodeParams["temperature"]
	if temperature == nil {
		temperature = 0.7
	}

	if apiKey == "" {
		return nil, fmt.Errorf("apiKey is required")
	}

	if prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	var results []model.DataItem

	for range inputData {
		// Prepare request payload
		messages := []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		}

		payload := map[string]interface{}{
			"model":       modelName,
			"messages":    messages,
			"max_tokens":  maxTokens,
			"temperature": temperature,
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %v", err)
		}

		// Make API request
		req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %v", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

		resp, err := n.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("openai API returned status %d", resp.StatusCode)
		}

		// Parse response
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %v", err)
		}

		// Extract the generated text
		choices, ok := result["choices"].([]interface{})
		if !ok || len(choices) == 0 {
			return nil, fmt.Errorf("no choices in OpenAI response")
		}

		choice := choices[0].(map[string]interface{})
		message := choice["message"].(map[string]interface{})
		content := message["content"].(string)

		// Create result item
		resultItem := model.DataItem{
			JSON: map[string]interface{}{
				"prompt":      prompt,
				"response":    content,
				"model":       modelName,
				"usage":       result["usage"],
				"finishReason": choice["finish_reason"],
			},
		}

		results = append(results, resultItem)
	}

	return results, nil
}
