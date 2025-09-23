package messaging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/n8n-go/internal/core/base"
	"github.com/n8n-go/internal/core/interfaces"
)

// DiscordNode provides Discord messaging operations
type DiscordNode struct {
	*base.BaseNode
	httpClient *http.Client
}

// NewDiscordNode creates a new Discord node
func NewDiscordNode() *DiscordNode {
	return &DiscordNode{
		BaseNode: base.NewBaseNode("Discord", "Discord Messaging"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetMetadata returns the node metadata
func (n *DiscordNode) GetMetadata() interfaces.NodeMetadata {
	return interfaces.NodeMetadata{
		Name:        "Discord",
		DisplayName: "Discord",
		Description: "Send messages, manage channels, and interact with Discord",
		Group:       []string{"Communication"},
		Version:     1,
		Inputs:      []string{"main"},
		Outputs:     []string{"main"},
		Credentials: []interfaces.CredentialType{
			{
				Name:        "discordApi",
				Required:    true,
				DisplayName: "Discord Bot/Webhook",
			},
		},
		Properties: []interfaces.NodeProperty{
			{
				Name:        "resource",
				DisplayName: "Resource",
				Type:        "options",
				Options: []interfaces.OptionItem{
					{Name: "Message", Value: "message"},
					{Name: "Channel", Value: "channel"},
					{Name: "User", Value: "user"},
					{Name: "Guild", Value: "guild"},
					{Name: "Webhook", Value: "webhook"},
					{Name: "Role", Value: "role"},
					{Name: "Member", Value: "member"},
				},
				Default:     "message",
				Required:    true,
				Description: "The resource to operate on",
			},
			{
				Name:        "operation",
				DisplayName: "Operation",
				Type:        "options",
				Options: []interfaces.OptionItem{
					{Name: "Send", Value: "send"},
					{Name: "Edit", Value: "edit"},
					{Name: "Delete", Value: "delete"},
					{Name: "Get", Value: "get"},
					{Name: "React", Value: "react"},
					{Name: "Pin", Value: "pin"},
					{Name: "Unpin", Value: "unpin"},
				},
				Default:     "send",
				Required:    true,
				Description: "The operation to perform",
			},
			{
				Name:        "channelId",
				DisplayName: "Channel ID",
				Type:        "string",
				Default:     "",
				Description: "Discord channel ID",
				Required:    true,
			},
			{
				Name:        "content",
				DisplayName: "Message Content",
				Type:        "string",
				TypeOptions: map[string]interface{}{
					"rows": 5,
				},
				Default:     "",
				Description: "The message content to send",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"send", "edit"},
					},
				},
			},
			{
				Name:        "username",
				DisplayName: "Username",
				Type:        "string",
				Default:     "",
				Description: "Override the default username (webhook only)",
			},
			{
				Name:        "avatarUrl",
				DisplayName: "Avatar URL",
				Type:        "string",
				Default:     "",
				Description: "Override the default avatar (webhook only)",
			},
			{
				Name:        "embeds",
				DisplayName: "Embeds",
				Type:        "json",
				Default:     "[]",
				Description: "Rich embed objects",
			},
			{
				Name:        "messageId",
				DisplayName: "Message ID",
				Type:        "string",
				Default:     "",
				Description: "The message ID for operations",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"edit", "delete", "get", "react", "pin", "unpin"},
					},
				},
			},
			{
				Name:        "emoji",
				DisplayName: "Emoji",
				Type:        "string",
				Default:     "",
				Description: "Emoji for reaction (Unicode or custom emoji ID)",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"react"},
					},
				},
			},
			{
				Name:        "guildId",
				DisplayName: "Guild ID",
				Type:        "string",
				Default:     "",
				Description: "Discord guild/server ID",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"resource": []string{"guild", "role", "member"},
					},
				},
			},
			{
				Name:        "userId",
				DisplayName: "User ID",
				Type:        "string",
				Default:     "",
				Description: "Discord user ID",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"resource": []string{"user", "member"},
					},
				},
			},
			{
				Name:        "tts",
				DisplayName: "Text-to-Speech",
				Type:        "boolean",
				Default:     false,
				Description: "Send as text-to-speech message",
			},
			{
				Name:        "limit",
				DisplayName: "Limit",
				Type:        "number",
				Default:     50,
				Description: "Number of results to return",
			},
		},
	}
}

// Execute runs the Discord operation
func (n *DiscordNode) Execute(ctx context.Context, params interfaces.ExecutionParams) (interfaces.NodeOutput, error) {
	// Get credentials
	credentials, err := params.GetCredentials("discordApi")
	if err != nil {
		return interfaces.NodeOutput{}, fmt.Errorf("failed to get Discord credentials: %w", err)
	}

	// Check credential type (bot token or webhook URL)
	var botToken, webhookUrl string
	if token, ok := credentials["botToken"].(string); ok && token != "" {
		botToken = token
	}
	if webhook, ok := credentials["webhookUrl"].(string); ok && webhook != "" {
		webhookUrl = webhook
	}

	if botToken == "" && webhookUrl == "" {
		return interfaces.NodeOutput{}, fmt.Errorf("Discord bot token or webhook URL required")
	}

	// Get resource and operation
	resource := params.GetNodeParameter("resource", "message").(string)
	operation := params.GetNodeParameter("operation", "send").(string)

	var result interface{}

	// Handle webhook operations separately
	if webhookUrl != "" && resource == "message" && operation == "send" {
		result, err = n.sendWebhookMessage(webhookUrl, params)
	} else if botToken != "" {
		// Handle bot token operations
		switch resource {
		case "message":
			result, err = n.handleMessageResource(botToken, operation, params)
		case "channel":
			result, err = n.handleChannelResource(botToken, operation, params)
		case "user":
			result, err = n.handleUserResource(botToken, operation, params)
		case "guild":
			result, err = n.handleGuildResource(botToken, operation, params)
		case "role":
			result, err = n.handleRoleResource(botToken, operation, params)
		case "member":
			result, err = n.handleMemberResource(botToken, operation, params)
		default:
			err = fmt.Errorf("unsupported resource: %s", resource)
		}
	} else {
		err = fmt.Errorf("operation requires bot token")
	}

	if err != nil {
		return interfaces.NodeOutput{}, err
	}

	// Format output
	var outputItems []interfaces.ItemData
	switch v := result.(type) {
	case []interface{}:
		for i, item := range v {
			if itemMap, ok := item.(map[string]interface{}); ok {
				outputItems = append(outputItems, interfaces.ItemData{
					JSON:  itemMap,
					Index: i,
				})
			}
		}
	case map[string]interface{}:
		outputItems = append(outputItems, interfaces.ItemData{
			JSON:  v,
			Index: 0,
		})
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

// Webhook operations

func (n *DiscordNode) sendWebhookMessage(webhookUrl string, params interfaces.ExecutionParams) (map[string]interface{}, error) {
	content := params.GetNodeParameter("content", "").(string)
	username := params.GetNodeParameter("username", "").(string)
	avatarUrl := params.GetNodeParameter("avatarUrl", "").(string)
	tts := params.GetNodeParameter("tts", false).(bool)

	body := map[string]interface{}{
		"content": content,
		"tts":     tts,
	}

	if username != "" {
		body["username"] = username
	}
	if avatarUrl != "" {
		body["avatar_url"] = avatarUrl
	}

	// Add embeds if provided
	embedsJSON := params.GetNodeParameter("embeds", "[]").(string)
	if embedsJSON != "[]" {
		var embeds []interface{}
		if err := json.Unmarshal([]byte(embedsJSON), &embeds); err == nil && len(embeds) > 0 {
			body["embeds"] = embeds
		}
	}

	return n.makeDiscordRequest("POST", webhookUrl, "", body)
}

// Message resource handlers

func (n *DiscordNode) handleMessageResource(token, operation string, params interfaces.ExecutionParams) (interface{}, error) {
	channelId := params.GetNodeParameter("channelId", "").(string)
	if channelId == "" {
		return nil, fmt.Errorf("channel ID is required")
	}

	switch operation {
	case "send":
		return n.sendMessage(token, channelId, params)
	case "edit":
		return n.editMessage(token, channelId, params)
	case "delete":
		return n.deleteMessage(token, channelId, params)
	case "get":
		return n.getMessage(token, channelId, params)
	case "react":
		return n.addReaction(token, channelId, params)
	case "pin":
		return n.pinMessage(token, channelId, params)
	case "unpin":
		return n.unpinMessage(token, channelId, params)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

func (n *DiscordNode) sendMessage(token, channelId string, params interfaces.ExecutionParams) (map[string]interface{}, error) {
	content := params.GetNodeParameter("content", "").(string)
	tts := params.GetNodeParameter("tts", false).(bool)

	body := map[string]interface{}{
		"content": content,
		"tts":     tts,
	}

	// Add embeds if provided
	embedsJSON := params.GetNodeParameter("embeds", "[]").(string)
	if embedsJSON != "[]" {
		var embeds []interface{}
		if err := json.Unmarshal([]byte(embedsJSON), &embeds); err == nil && len(embeds) > 0 {
			body["embeds"] = embeds
		}
	}

	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages", channelId)
	return n.makeDiscordRequest("POST", url, token, body)
}

func (n *DiscordNode) editMessage(token, channelId string, params interfaces.ExecutionParams) (map[string]interface{}, error) {
	messageId := params.GetNodeParameter("messageId", "").(string)
	if messageId == "" {
		return nil, fmt.Errorf("message ID is required")
	}

	content := params.GetNodeParameter("content", "").(string)
	body := map[string]interface{}{
		"content": content,
	}

	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages/%s", channelId, messageId)
	return n.makeDiscordRequest("PATCH", url, token, body)
}

func (n *DiscordNode) deleteMessage(token, channelId string, params interfaces.ExecutionParams) (map[string]interface{}, error) {
	messageId := params.GetNodeParameter("messageId", "").(string)
	if messageId == "" {
		return nil, fmt.Errorf("message ID is required")
	}

	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages/%s", channelId, messageId)
	return n.makeDiscordRequest("DELETE", url, token, nil)
}

func (n *DiscordNode) getMessage(token, channelId string, params interfaces.ExecutionParams) (map[string]interface{}, error) {
	messageId := params.GetNodeParameter("messageId", "").(string)
	if messageId == "" {
		// Get multiple messages
		limit := int(params.GetNodeParameter("limit", 50).(float64))
		url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages?limit=%d", channelId, limit)
		return n.makeDiscordRequest("GET", url, token, nil)
	}

	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages/%s", channelId, messageId)
	return n.makeDiscordRequest("GET", url, token, nil)
}

func (n *DiscordNode) addReaction(token, channelId string, params interfaces.ExecutionParams) (map[string]interface{}, error) {
	messageId := params.GetNodeParameter("messageId", "").(string)
	emoji := params.GetNodeParameter("emoji", "").(string)

	if messageId == "" || emoji == "" {
		return nil, fmt.Errorf("message ID and emoji are required")
	}

	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages/%s/reactions/%s/@me", channelId, messageId, emoji)
	return n.makeDiscordRequest("PUT", url, token, nil)
}

func (n *DiscordNode) pinMessage(token, channelId string, params interfaces.ExecutionParams) (map[string]interface{}, error) {
	messageId := params.GetNodeParameter("messageId", "").(string)
	if messageId == "" {
		return nil, fmt.Errorf("message ID is required")
	}

	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/pins/%s", channelId, messageId)
	return n.makeDiscordRequest("PUT", url, token, nil)
}

func (n *DiscordNode) unpinMessage(token, channelId string, params interfaces.ExecutionParams) (map[string]interface{}, error) {
	messageId := params.GetNodeParameter("messageId", "").(string)
	if messageId == "" {
		return nil, fmt.Errorf("message ID is required")
	}

	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/pins/%s", channelId, messageId)
	return n.makeDiscordRequest("DELETE", url, token, nil)
}

// Channel resource handlers

func (n *DiscordNode) handleChannelResource(token, operation string, params interfaces.ExecutionParams) (interface{}, error) {
	channelId := params.GetNodeParameter("channelId", "").(string)

	switch operation {
	case "get":
		if channelId == "" {
			return nil, fmt.Errorf("channel ID is required")
		}
		url := fmt.Sprintf("https://discord.com/api/v10/channels/%s", channelId)
		return n.makeDiscordRequest("GET", url, token, nil)

	case "modify":
		if channelId == "" {
			return nil, fmt.Errorf("channel ID is required")
		}
		body := map[string]interface{}{
			"name":     params.GetNodeParameter("name", "").(string),
			"topic":    params.GetNodeParameter("topic", "").(string),
			"position": params.GetNodeParameter("position", 0).(float64),
		}
		url := fmt.Sprintf("https://discord.com/api/v10/channels/%s", channelId)
		return n.makeDiscordRequest("PATCH", url, token, body)

	case "delete":
		if channelId == "" {
			return nil, fmt.Errorf("channel ID is required")
		}
		url := fmt.Sprintf("https://discord.com/api/v10/channels/%s", channelId)
		return n.makeDiscordRequest("DELETE", url, token, nil)

	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// User resource handlers

func (n *DiscordNode) handleUserResource(token, operation string, params interfaces.ExecutionParams) (interface{}, error) {
	userId := params.GetNodeParameter("userId", "").(string)

	switch operation {
	case "get":
		if userId == "" || userId == "@me" {
			// Get current user
			return n.makeDiscordRequest("GET", "https://discord.com/api/v10/users/@me", token, nil)
		}
		url := fmt.Sprintf("https://discord.com/api/v10/users/%s", userId)
		return n.makeDiscordRequest("GET", url, token, nil)

	case "getDMs":
		return n.makeDiscordRequest("GET", "https://discord.com/api/v10/users/@me/channels", token, nil)

	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// Guild resource handlers

func (n *DiscordNode) handleGuildResource(token, operation string, params interfaces.ExecutionParams) (interface{}, error) {
	guildId := params.GetNodeParameter("guildId", "").(string)

	switch operation {
	case "get":
		if guildId == "" {
			// Get all guilds
			return n.makeDiscordRequest("GET", "https://discord.com/api/v10/users/@me/guilds", token, nil)
		}
		url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s", guildId)
		return n.makeDiscordRequest("GET", url, token, nil)

	case "getChannels":
		if guildId == "" {
			return nil, fmt.Errorf("guild ID is required")
		}
		url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/channels", guildId)
		return n.makeDiscordRequest("GET", url, token, nil)

	case "getMembers":
		if guildId == "" {
			return nil, fmt.Errorf("guild ID is required")
		}
		limit := int(params.GetNodeParameter("limit", 50).(float64))
		url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/members?limit=%d", guildId, limit)
		return n.makeDiscordRequest("GET", url, token, nil)

	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// Role resource handlers

func (n *DiscordNode) handleRoleResource(token, operation string, params interfaces.ExecutionParams) (interface{}, error) {
	guildId := params.GetNodeParameter("guildId", "").(string)
	if guildId == "" {
		return nil, fmt.Errorf("guild ID is required")
	}

	switch operation {
	case "get":
		url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/roles", guildId)
		return n.makeDiscordRequest("GET", url, token, nil)

	case "create":
		body := map[string]interface{}{
			"name":        params.GetNodeParameter("roleName", "").(string),
			"color":       params.GetNodeParameter("color", 0).(float64),
			"permissions": params.GetNodeParameter("permissions", "0").(string),
		}
		url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/roles", guildId)
		return n.makeDiscordRequest("POST", url, token, body)

	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// Member resource handlers

func (n *DiscordNode) handleMemberResource(token, operation string, params interfaces.ExecutionParams) (interface{}, error) {
	guildId := params.GetNodeParameter("guildId", "").(string)
	userId := params.GetNodeParameter("userId", "").(string)

	if guildId == "" || userId == "" {
		return nil, fmt.Errorf("guild ID and user ID are required")
	}

	switch operation {
	case "get":
		url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/members/%s", guildId, userId)
		return n.makeDiscordRequest("GET", url, token, nil)

	case "addRole":
		roleId := params.GetNodeParameter("roleId", "").(string)
		if roleId == "" {
			return nil, fmt.Errorf("role ID is required")
		}
		url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/members/%s/roles/%s", guildId, userId, roleId)
		return n.makeDiscordRequest("PUT", url, token, nil)

	case "removeRole":
		roleId := params.GetNodeParameter("roleId", "").(string)
		if roleId == "" {
			return nil, fmt.Errorf("role ID is required")
		}
		url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/members/%s/roles/%s", guildId, userId, roleId)
		return n.makeDiscordRequest("DELETE", url, token, nil)

	case "kick":
		url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/members/%s", guildId, userId)
		return n.makeDiscordRequest("DELETE", url, token, nil)

	case "ban":
		body := map[string]interface{}{
			"delete_message_days": params.GetNodeParameter("deleteMessageDays", 0).(float64),
		}
		url := fmt.Sprintf("https://discord.com/api/v10/guilds/%s/bans/%s", guildId, userId)
		return n.makeDiscordRequest("PUT", url, token, body)

	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// Helper method for making Discord API requests

func (n *DiscordNode) makeDiscordRequest(method, url, token string, body interface{}) (map[string]interface{}, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bot "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle 204 No Content
	if resp.StatusCode == 204 {
		return map[string]interface{}{"success": true}, nil
	}

	if resp.StatusCode >= 400 {
		var errorResponse map[string]interface{}
		if err := json.Unmarshal(respBody, &errorResponse); err == nil {
			if msg, ok := errorResponse["message"].(string); ok {
				return nil, fmt.Errorf("Discord API error: %s", msg)
			}
		}
		return nil, fmt.Errorf("Discord API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		// Try parsing as array
		var arrayResult []interface{}
		if err := json.Unmarshal(respBody, &arrayResult); err == nil {
			return map[string]interface{}{"data": arrayResult}, nil
		}
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// Clone creates a copy of the node
func (n *DiscordNode) Clone() interfaces.Node {
	return &DiscordNode{
		BaseNode:   n.BaseNode.Clone(),
		httpClient: n.httpClient,
	}
}