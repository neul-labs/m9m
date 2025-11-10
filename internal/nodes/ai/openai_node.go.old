package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dipankar/n8n-go/internal/nodes/base"
)

// OpenAINode provides direct OpenAI API integration
type OpenAINode struct {
	*base.BaseNode
	httpClient *http.Client
}

// NewOpenAINode creates a new OpenAI node
func NewOpenAINode() *OpenAINode {
	return &OpenAINode{
		BaseNode: base.NewBaseNode(base.NodeDescription{Name: "OpenAI", Description: "OpenAI API", Category: "core"}),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GetMetadata returns the node metadata
func (n *OpenAINode) GetMetadata() base.NodeMetadata {
	return base.NodeMetadata{
		Name:        "OpenAI",
		DisplayName: "OpenAI",
		Description: "Use OpenAI's API for GPT models, embeddings, and more",
		Group:       []string{"AI"},
		Version:     1,
		Inputs:      []string{"main"},
		Outputs:     []string{"main"},
		Credentials: []base.CredentialType{
			{
				Name:        "openAiApi",
				Required:    true,
				DisplayName: "OpenAI API",
			},
		},
		Properties: []base.NodeProperty{
			{
				Name:        "operation",
				DisplayName: "Operation",
				Type:        "options",
				Options: []base.OptionItem{
					{Name: "Chat Completion", Value: "chat"},
					{Name: "Text Completion", Value: "completion"},
					{Name: "Create Embedding", Value: "embedding"},
					{Name: "Generate Image", Value: "image"},
					{Name: "Text to Speech", Value: "tts"},
					{Name: "Speech to Text", Value: "stt"},
					{Name: "Moderation", Value: "moderation"},
				},
				Default:     "chat",
				Required:    true,
				Description: "The operation to perform",
			},
			{
				Name:        "model",
				DisplayName: "Model",
				Type:        "options",
				Options: []base.OptionItem{
					// GPT-4 models
					{Name: "GPT-4 Turbo", Value: "gpt-4-turbo"},
					{Name: "GPT-4", Value: "gpt-4"},
					{Name: "GPT-4 32k", Value: "gpt-4-32k"},
					// GPT-3.5 models
					{Name: "GPT-3.5 Turbo", Value: "gpt-3.5-turbo"},
					{Name: "GPT-3.5 Turbo 16k", Value: "gpt-3.5-turbo-16k"},
					// Embedding models
					{Name: "Text Embedding Ada 002", Value: "text-embedding-ada-002"},
					{Name: "Text Embedding 3 Small", Value: "text-embedding-3-small"},
					{Name: "Text Embedding 3 Large", Value: "text-embedding-3-large"},
					// Image models
					{Name: "DALL-E 3", Value: "dall-e-3"},
					{Name: "DALL-E 2", Value: "dall-e-2"},
				},
				Default:     "gpt-3.5-turbo",
				Required:    true,
				Description: "The model to use",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"chat", "completion", "embedding", "image"},
					},
				},
			},
			{
				Name:        "prompt",
				DisplayName: "Prompt",
				Type:        "string",
				TypeOptions: map[string]interface{}{
					"rows": 5,
				},
				Default:     "",
				Required:    true,
				Description: "The prompt to send to the model",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"chat", "completion", "image"},
					},
				},
			},
			{
				Name:        "systemMessage",
				DisplayName: "System Message",
				Type:        "string",
				TypeOptions: map[string]interface{}{
					"rows": 3,
				},
				Default:     "You are a helpful assistant.",
				Description: "The system message to set context",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"chat"},
					},
				},
			},
			{
				Name:        "temperature",
				DisplayName: "Temperature",
				Type:        "number",
				TypeOptions: map[string]interface{}{
					"numberPrecision": 1,
					"minValue":        0,
					"maxValue":        2,
				},
				Default:     0.7,
				Description: "Controls randomness (0=deterministic, 2=very random)",
			},
			{
				Name:        "maxTokens",
				DisplayName: "Max Tokens",
				Type:        "number",
				Default:     2048,
				Description: "Maximum number of tokens to generate",
			},
			{
				Name:        "responseFormat",
				DisplayName: "Response Format",
				Type:        "options",
				Options: []base.OptionItem{
					{Name: "Text", Value: "text"},
					{Name: "JSON", Value: "json_object"},
				},
				Default:     "text",
				Description: "The format of the response",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"chat"},
						"model":     []string{"gpt-4-turbo", "gpt-3.5-turbo"},
					},
				},
			},
		},
	}
}

// Execute runs the OpenAI operation
func (n *OpenAINode) Execute(ctx context.Context, params base.ExecutionParams) (base.NodeOutput, error) {
	operation := params.GetNodeParameter("operation", "chat").(string)
	model := params.GetNodeParameter("model", "gpt-3.5-turbo").(string)
	
	// Get credentials
	credentials, err := params.GetCredentials("openAiApi")
	if err != nil {
		return base.NodeOutput{}, fmt.Errorf("failed to get OpenAI credentials: %w", err)
	}

	apiKey, ok := credentials["apiKey"].(string)
	if !ok || apiKey == "" {
		return base.NodeOutput{}, fmt.Errorf("OpenAI API key not found in credentials")
	}

	// Get input data
	inputData := params.GetInputData()
	var outputItems []base.ItemData

	for _, item := range inputData {
		var result interface{}
		var err error

		switch operation {
		case "chat":
			result, err = n.executeChatCompletion(apiKey, model, item, params)
		case "completion":
			result, err = n.executeTextCompletion(apiKey, model, item, params)
		case "embedding":
			result, err = n.executeEmbedding(apiKey, model, item, params)
		case "image":
			result, err = n.executeImageGeneration(apiKey, model, item, params)
		case "tts":
			result, err = n.executeTextToSpeech(apiKey, model, item, params)
		case "stt":
			result, err = n.executeSpeechToText(apiKey, model, item, params)
		case "moderation":
			result, err = n.executeModeration(apiKey, item, params)
		default:
			err = fmt.Errorf("unsupported operation: %s", operation)
		}

		if err != nil {
			return base.NodeOutput{}, err
		}

		if resultMap, ok := result.(map[string]interface{}); ok {
			outputItems = append(outputItems, base.ItemData{
				JSON:  resultMap,
				Index: item.Index,
			})
		} else {
			outputItems = append(outputItems, base.ItemData{
				JSON: map[string]interface{}{
					"result": result,
				},
				Index: item.Index,
			})
		}
	}

	return base.NodeOutput{
		Items: outputItems,
	}, nil
}

// executeChatCompletion handles chat completion requests
func (n *OpenAINode) executeChatCompletion(apiKey, model string, item base.ItemData, params base.ExecutionParams) (map[string]interface{}, error) {
	prompt := params.GetNodeParameter("prompt", "").(string)
	systemMessage := params.GetNodeParameter("systemMessage", "You are a helpful assistant.").(string)
	temperature := params.GetNodeParameter("temperature", 0.7).(float64)
	maxTokens := params.GetNodeParameter("maxTokens", 2048).(int)
	responseFormat := params.GetNodeParameter("responseFormat", "text").(string)

	// Build messages
	messages := []map[string]interface{}{
		{
			"role":    "system",
			"content": systemMessage,
		},
		{
			"role":    "user",
			"content": prompt,
		},
	}

	// Build request body
	requestBody := map[string]interface{}{
		"model":       model,
		"messages":    messages,
		"temperature": temperature,
		"max_tokens":  maxTokens,
	}

	if responseFormat == "json_object" {
		requestBody["response_format"] = map[string]string{"type": "json_object"}
	}

	// Make API request
	response, err := n.makeOpenAIRequest("https://api.openai.com/v1/chat/completions", apiKey, requestBody)
	if err != nil {
		return nil, err
	}

	// Extract the response
	choices, ok := response["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, fmt.Errorf("unexpected response format")
	}

	firstChoice := choices[0].(map[string]interface{})
	message := firstChoice["message"].(map[string]interface{})
	content := message["content"].(string)

	// Parse JSON if response format is JSON
	if responseFormat == "json_object" {
		var jsonContent interface{}
		if err := json.Unmarshal([]byte(content), &jsonContent); err == nil {
			return map[string]interface{}{
				"content": jsonContent,
				"usage":   response["usage"],
				"model":   response["model"],
			}, nil
		}
	}

	return map[string]interface{}{
		"content": content,
		"usage":   response["usage"],
		"model":   response["model"],
	}, nil
}

// executeTextCompletion handles text completion requests
func (n *OpenAINode) executeTextCompletion(apiKey, model string, item base.ItemData, params base.ExecutionParams) (map[string]interface{}, error) {
	prompt := params.GetNodeParameter("prompt", "").(string)
	temperature := params.GetNodeParameter("temperature", 0.7).(float64)
	maxTokens := params.GetNodeParameter("maxTokens", 2048).(int)

	requestBody := map[string]interface{}{
		"model":       model,
		"prompt":      prompt,
		"temperature": temperature,
		"max_tokens":  maxTokens,
	}

	response, err := n.makeOpenAIRequest("https://api.openai.com/v1/completions", apiKey, requestBody)
	if err != nil {
		return nil, err
	}

	choices, ok := response["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, fmt.Errorf("unexpected response format")
	}

	firstChoice := choices[0].(map[string]interface{})
	text := firstChoice["text"].(string)

	return map[string]interface{}{
		"text":  text,
		"usage": response["usage"],
		"model": response["model"],
	}, nil
}

// executeEmbedding handles embedding requests
func (n *OpenAINode) executeEmbedding(apiKey, model string, item base.ItemData, params base.ExecutionParams) (map[string]interface{}, error) {
	text := ""
	if textValue, ok := item.JSON["text"].(string); ok {
		text = textValue
	} else if prompt := params.GetNodeParameter("prompt", "").(string); prompt != "" {
		text = prompt
	} else {
		return nil, fmt.Errorf("no text provided for embedding")
	}

	requestBody := map[string]interface{}{
		"model": model,
		"input": text,
	}

	response, err := n.makeOpenAIRequest("https://api.openai.com/v1/embeddings", apiKey, requestBody)
	if err != nil {
		return nil, err
	}

	data, ok := response["data"].([]interface{})
	if !ok || len(data) == 0 {
		return nil, fmt.Errorf("unexpected response format")
	}

	firstEmbedding := data[0].(map[string]interface{})
	embedding := firstEmbedding["embedding"].([]interface{})

	return map[string]interface{}{
		"embedding": embedding,
		"model":     response["model"],
		"usage":     response["usage"],
	}, nil
}

// executeImageGeneration handles image generation requests
func (n *OpenAINode) executeImageGeneration(apiKey, model string, item base.ItemData, params base.ExecutionParams) (map[string]interface{}, error) {
	prompt := params.GetNodeParameter("prompt", "").(string)
	
	requestBody := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"n":      1,
		"size":   "1024x1024",
	}

	response, err := n.makeOpenAIRequest("https://api.openai.com/v1/images/generations", apiKey, requestBody)
	if err != nil {
		return nil, err
	}

	data, ok := response["data"].([]interface{})
	if !ok || len(data) == 0 {
		return nil, fmt.Errorf("unexpected response format")
	}

	firstImage := data[0].(map[string]interface{})

	return map[string]interface{}{
		"url":           firstImage["url"],
		"revised_prompt": firstImage["revised_prompt"],
	}, nil
}

// executeTextToSpeech handles text to speech requests
func (n *OpenAINode) executeTextToSpeech(apiKey, model string, item base.ItemData, params base.ExecutionParams) (map[string]interface{}, error) {
	text := params.GetNodeParameter("prompt", "").(string)

	requestBody := map[string]interface{}{
		"model": "tts-1",
		"input": text,
		"voice": "alloy",
	}

	// This would return audio data - simplified for now
	response, err := n.makeOpenAIRequest("https://api.openai.com/v1/audio/speech", apiKey, requestBody)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// executeSpeechToText handles speech to text requests
func (n *OpenAINode) executeSpeechToText(apiKey, model string, item base.ItemData, params base.ExecutionParams) (map[string]interface{}, error) {
	// This would handle audio file upload - simplified for now
	return map[string]interface{}{
		"text": "Transcribed text would appear here",
	}, nil
}

// executeModeration handles content moderation requests
func (n *OpenAINode) executeModeration(apiKey string, item base.ItemData, params base.ExecutionParams) (map[string]interface{}, error) {
	text := params.GetNodeParameter("prompt", "").(string)

	requestBody := map[string]interface{}{
		"input": text,
	}

	response, err := n.makeOpenAIRequest("https://api.openai.com/v1/moderations", apiKey, requestBody)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// makeOpenAIRequest makes an HTTP request to OpenAI API
func (n *OpenAINode) makeOpenAIRequest(url, apiKey string, body map[string]interface{}) (map[string]interface{}, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errorResponse map[string]interface{}
		if err := json.Unmarshal(respBody, &errorResponse); err == nil {
			if errorMsg, ok := errorResponse["error"].(map[string]interface{}); ok {
				return nil, fmt.Errorf("OpenAI API error: %v", errorMsg["message"])
			}
		}
		return nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// Clone creates a copy of the node
func (n *OpenAINode) Clone() base.Node {
	return &OpenAINode{
		BaseNode:   n.BaseNode.Clone(),
		httpClient: n.httpClient,
	}
}