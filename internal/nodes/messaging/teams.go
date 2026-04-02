package messaging

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// TeamsHTTPClient abstracts the HTTP client for testing.
type TeamsHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// TeamsNode implements Microsoft Teams operations.
type TeamsNode struct {
	*base.BaseNode
	client TeamsHTTPClient
}

// NewTeamsNode creates a new Microsoft Teams node.
func NewTeamsNode() *TeamsNode {
	return &TeamsNode{
		BaseNode: base.NewBaseNode(base.NodeDescription{
			Name:        "Microsoft Teams",
			Description: "Send messages and manage Microsoft Teams channels",
			Category:    "Messaging",
		}),
		client: &http.Client{},
	}
}

// NewTeamsNodeWithClient creates a Teams node with a custom HTTP client.
func NewTeamsNodeWithClient(client TeamsHTTPClient) *TeamsNode {
	n := NewTeamsNode()
	n.client = client
	return n
}

// Execute runs the configured Teams operation.
func (n *TeamsNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	baseURL := n.GetStringParameter(nodeParams, "baseUrl", "https://graph.microsoft.com/v1.0")
	baseURL = strings.TrimRight(baseURL, "/")
	operation := n.GetStringParameter(nodeParams, "operation", "sendMessage")
	accessToken := n.GetStringParameter(nodeParams, "accessToken", "")

	if accessToken == "" {
		return nil, n.CreateError("accessToken is required", nil)
	}

	switch operation {
	case "sendMessage":
		return n.sendMessage(baseURL, accessToken, nodeParams)
	case "listChannels":
		return n.listChannels(baseURL, accessToken, nodeParams)
	default:
		return nil, n.CreateError(fmt.Sprintf("unsupported operation: %s", operation), nil)
	}
}

func (n *TeamsNode) sendMessage(baseURL, token string, params map[string]interface{}) ([]model.DataItem, error) {
	teamID := n.GetStringParameter(params, "teamId", "")
	channelID := n.GetStringParameter(params, "channelId", "")
	text := n.GetStringParameter(params, "text", "")

	if teamID == "" || channelID == "" {
		return nil, n.CreateError("teamId and channelId are required for sendMessage", nil)
	}
	if text == "" {
		return nil, n.CreateError("text is required for sendMessage", nil)
	}

	body := map[string]interface{}{
		"body": map[string]interface{}{
			"content": text,
		},
	}

	reqURL := fmt.Sprintf("%s/teams/%s/channels/%s/messages", baseURL, teamID, channelID)
	return n.doJSONRequest("POST", reqURL, token, body)
}

func (n *TeamsNode) listChannels(baseURL, token string, params map[string]interface{}) ([]model.DataItem, error) {
	teamID := n.GetStringParameter(params, "teamId", "")
	if teamID == "" {
		return nil, n.CreateError("teamId is required for listChannels", nil)
	}

	reqURL := fmt.Sprintf("%s/teams/%s/channels", baseURL, teamID)
	return n.doJSONRequest("GET", reqURL, token, nil)
}

func (n *TeamsNode) doJSONRequest(method, reqURL, token string, body interface{}) ([]model.DataItem, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, n.CreateError(fmt.Sprintf("failed to marshal body: %v", err), nil)
		}
		bodyReader = strings.NewReader(string(data))
	}

	req, err := http.NewRequest(method, reqURL, bodyReader)
	if err != nil {
		return nil, n.CreateError(fmt.Sprintf("failed to create request: %v", err), nil)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
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

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return []model.DataItem{{JSON: map[string]interface{}{
			"statusCode": resp.StatusCode,
			"body":       string(respBody),
		}}}, nil
	}

	result["statusCode"] = resp.StatusCode
	return []model.DataItem{{JSON: result}}, nil
}

// ValidateParameters validates Teams node parameters.
func (n *TeamsNode) ValidateParameters(params map[string]interface{}) error {
	if params == nil {
		return n.CreateError("parameters cannot be nil", nil)
	}

	op := n.GetStringParameter(params, "operation", "sendMessage")
	if op != "sendMessage" && op != "listChannels" {
		return n.CreateError(fmt.Sprintf("invalid operation: %s", op), nil)
	}

	if n.GetStringParameter(params, "accessToken", "") == "" {
		return n.CreateError("accessToken is required", nil)
	}

	return nil
}
