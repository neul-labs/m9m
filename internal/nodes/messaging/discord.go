package messaging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// DiscordNode sends messages to Discord
type DiscordNode struct {
	*base.BaseNode
	httpClient *http.Client
}

// NewDiscordNode creates a new Discord node
func NewDiscordNode() *DiscordNode {
	return &DiscordNode{
		BaseNode: base.NewBaseNode(base.NodeDescription{
			Name:        "Discord",
			Description: "Send messages and interact with Discord",
			Category:    "communication",
		}),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Execute sends a message to Discord
func (n *DiscordNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	// Get parameters
	webhookURL := n.GetStringParameter(nodeParams, "webhookUrl", "")
	content := n.GetStringParameter(nodeParams, "content", "")
	username := n.GetStringParameter(nodeParams, "username", "m9m Bot")
	avatarURL := n.GetStringParameter(nodeParams, "avatarUrl", "")

	if webhookURL == "" {
		return nil, fmt.Errorf("webhookUrl is required")
	}

	if content == "" {
		return nil, fmt.Errorf("content is required")
	}

	var results []model.DataItem

	for range inputData {
		// Prepare Discord webhook payload
		payload := map[string]interface{}{
			"content":  content,
			"username": username,
		}

		if avatarURL != "" {
			payload["avatar_url"] = avatarURL
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %v", err)
		}

		// Send to Discord
		resp, err := n.httpClient.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("discord returned status %d", resp.StatusCode)
		}

		result := model.DataItem{
			JSON: map[string]interface{}{
				"success":  true,
				"content":  content,
				"username": username,
				"status":   resp.StatusCode,
			},
		}

		results = append(results, result)
	}

	return results, nil
}
