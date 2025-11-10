package messaging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dipankar/n8n-go/internal/model"
	"github.com/dipankar/n8n-go/internal/nodes/base"
)

// SlackNode sends messages to Slack
type SlackNode struct {
	*base.BaseNode
	httpClient *http.Client
}

// NewSlackNode creates a new Slack node
func NewSlackNode() *SlackNode {
	return &SlackNode{
		BaseNode: base.NewBaseNode(base.NodeDescription{
			Name:        "Slack",
			Description: "Send messages and interact with Slack",
			Category:    "communication",
		}),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Execute sends a message to Slack
func (n *SlackNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	// Get parameters
	webhookURL := n.GetStringParameter(nodeParams, "webhookUrl", "")
	text := n.GetStringParameter(nodeParams, "text", "")
	channel := n.GetStringParameter(nodeParams, "channel", "")
	username := n.GetStringParameter(nodeParams, "username", "n8n-go Bot")

	// Also check for authentication token
	token := n.GetStringParameter(nodeParams, "token", "")

	if webhookURL == "" && token == "" {
		return nil, fmt.Errorf("either webhookUrl or token is required")
	}

	if text == "" {
		return nil, fmt.Errorf("text is required")
	}

	var results []model.DataItem

	for range inputData {
		var result model.DataItem

		if webhookURL != "" {
			// Use webhook URL method
			err := n.sendViaWebhook(webhookURL, text, username)
			if err != nil {
				return nil, fmt.Errorf("failed to send message: %v", err)
			}

			result = model.DataItem{
				JSON: map[string]interface{}{
					"success": true,
					"text":    text,
					"method":  "webhook",
				},
			}
		} else {
			// Use API token method
			if channel == "" {
				return nil, fmt.Errorf("channel is required when using token authentication")
			}

			err := n.sendViaAPI(token, channel, text)
			if err != nil {
				return nil, fmt.Errorf("failed to send message: %v", err)
			}

			result = model.DataItem{
				JSON: map[string]interface{}{
					"success": true,
					"text":    text,
					"channel": channel,
					"method":  "api",
				},
			}
		}

		results = append(results, result)
	}

	return results, nil
}

// sendViaWebhook sends a message using Slack webhook
func (n *SlackNode) sendViaWebhook(webhookURL, text, username string) error {
	payload := map[string]interface{}{
		"text":     text,
		"username": username,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	resp, err := n.httpClient.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}

	return nil
}

// sendViaAPI sends a message using Slack API
func (n *SlackNode) sendViaAPI(token, channel, text string) error {
	payload := map[string]interface{}{
		"channel": channel,
		"text":    text,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	if ok, exists := result["ok"].(bool); !exists || !ok {
		errMsg := "unknown error"
		if msg, exists := result["error"].(string); exists {
			errMsg = msg
		}
		return fmt.Errorf("slack API error: %s", errMsg)
	}

	return nil
}
