package email

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// SendGridHTTPClient abstracts the HTTP client for testing.
type SendGridHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// SendGridNode implements SendGrid email operations.
type SendGridNode struct {
	*base.BaseNode
	client SendGridHTTPClient
}

// NewSendGridNode creates a new SendGrid node.
func NewSendGridNode() *SendGridNode {
	return &SendGridNode{
		BaseNode: base.NewBaseNode(base.NodeDescription{
			Name:        "SendGrid",
			Description: "Send emails via SendGrid",
			Category:    "Email",
		}),
		client: &http.Client{},
	}
}

// NewSendGridNodeWithClient creates a SendGrid node with a custom HTTP client.
func NewSendGridNodeWithClient(client SendGridHTTPClient) *SendGridNode {
	n := NewSendGridNode()
	n.client = client
	return n
}

// Execute runs the configured SendGrid operation.
func (n *SendGridNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	baseURL := n.GetStringParameter(nodeParams, "baseUrl", "https://api.sendgrid.com/v3")
	baseURL = strings.TrimRight(baseURL, "/")
	operation := n.GetStringParameter(nodeParams, "operation", "send")
	apiKey := n.GetStringParameter(nodeParams, "apiKey", "")

	if apiKey == "" {
		return nil, n.CreateError("apiKey is required", nil)
	}

	switch operation {
	case "send":
		return n.send(baseURL, apiKey, nodeParams)
	case "sendTemplate":
		return n.sendTemplate(baseURL, apiKey, nodeParams)
	default:
		return nil, n.CreateError(fmt.Sprintf("unsupported operation: %s", operation), nil)
	}
}

func (n *SendGridNode) send(baseURL, apiKey string, params map[string]interface{}) ([]model.DataItem, error) {
	from := n.GetStringParameter(params, "from", "")
	to := n.GetStringParameter(params, "to", "")
	subject := n.GetStringParameter(params, "subject", "")
	content := n.GetStringParameter(params, "content", "")

	if from == "" || to == "" {
		return nil, n.CreateError("from and to are required", nil)
	}

	body := map[string]interface{}{
		"personalizations": []interface{}{
			map[string]interface{}{
				"to": []interface{}{
					map[string]interface{}{"email": to},
				},
			},
		},
		"from":    map[string]interface{}{"email": from},
		"subject": subject,
		"content": []interface{}{
			map[string]interface{}{
				"type":  "text/plain",
				"value": content,
			},
		},
	}

	return n.doRequest(baseURL+"/mail/send", apiKey, body)
}

func (n *SendGridNode) sendTemplate(baseURL, apiKey string, params map[string]interface{}) ([]model.DataItem, error) {
	from := n.GetStringParameter(params, "from", "")
	to := n.GetStringParameter(params, "to", "")
	templateID := n.GetStringParameter(params, "templateId", "")

	if from == "" || to == "" {
		return nil, n.CreateError("from and to are required", nil)
	}
	if templateID == "" {
		return nil, n.CreateError("templateId is required for sendTemplate", nil)
	}

	body := map[string]interface{}{
		"personalizations": []interface{}{
			map[string]interface{}{
				"to": []interface{}{
					map[string]interface{}{"email": to},
				},
			},
		},
		"from":        map[string]interface{}{"email": from},
		"template_id": templateID,
	}

	// Add dynamic template data if provided
	if dynamicData, ok := params["dynamicData"]; ok {
		personalizations := body["personalizations"].([]interface{})
		p := personalizations[0].(map[string]interface{})
		p["dynamic_template_data"] = dynamicData
	}

	return n.doRequest(baseURL+"/mail/send", apiKey, body)
}

func (n *SendGridNode) doRequest(url, apiKey string, body interface{}) ([]model.DataItem, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, n.CreateError(fmt.Sprintf("failed to marshal body: %v", err), nil)
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(data)))
	if err != nil {
		return nil, n.CreateError(fmt.Sprintf("failed to create request: %v", err), nil)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := n.client.Do(req)
	if err != nil {
		return nil, n.CreateError(fmt.Sprintf("request failed: %v", err), nil)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, n.CreateError(fmt.Sprintf("failed to read response: %v", err), nil)
	}

	result := map[string]interface{}{
		"statusCode": resp.StatusCode,
		"success":    resp.StatusCode >= 200 && resp.StatusCode < 300,
	}

	if len(respBody) > 0 {
		var parsed interface{}
		if err := json.Unmarshal(respBody, &parsed); err == nil {
			result["data"] = parsed
		} else {
			result["body"] = string(respBody)
		}
	}

	return []model.DataItem{{JSON: result}}, nil
}

// ValidateParameters validates SendGrid node parameters.
func (n *SendGridNode) ValidateParameters(params map[string]interface{}) error {
	if params == nil {
		return n.CreateError("parameters cannot be nil", nil)
	}

	op := n.GetStringParameter(params, "operation", "send")
	if op != "send" && op != "sendTemplate" {
		return n.CreateError(fmt.Sprintf("invalid operation: %s", op), nil)
	}

	if n.GetStringParameter(params, "apiKey", "") == "" {
		return n.CreateError("apiKey is required", nil)
	}

	return nil
}
