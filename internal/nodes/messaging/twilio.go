package messaging

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// TwilioHTTPClient abstracts the HTTP client for testing.
type TwilioHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// TwilioNode implements Twilio SMS operations.
type TwilioNode struct {
	*base.BaseNode
	client TwilioHTTPClient
}

// NewTwilioNode creates a new Twilio node.
func NewTwilioNode() *TwilioNode {
	return &TwilioNode{
		BaseNode: base.NewBaseNode(base.NodeDescription{
			Name:        "Twilio",
			Description: "Send SMS messages via Twilio",
			Category:    "Messaging",
		}),
		client: &http.Client{},
	}
}

// NewTwilioNodeWithClient creates a Twilio node with a custom HTTP client.
func NewTwilioNodeWithClient(client TwilioHTTPClient) *TwilioNode {
	n := NewTwilioNode()
	n.client = client
	return n
}

// Execute runs the configured Twilio operation.
func (n *TwilioNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	operation := n.GetStringParameter(nodeParams, "operation", "sendSms")
	accountSid := n.GetStringParameter(nodeParams, "accountSid", "")
	authToken := n.GetStringParameter(nodeParams, "authToken", "")

	if accountSid == "" || authToken == "" {
		return nil, n.CreateError("accountSid and authToken are required", nil)
	}

	baseURL := n.GetStringParameter(nodeParams, "baseUrl",
		fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s", accountSid))
	baseURL = strings.TrimRight(baseURL, "/")

	switch operation {
	case "sendSms":
		return n.sendSms(baseURL, accountSid, authToken, nodeParams)
	case "getMessages":
		return n.getMessages(baseURL, accountSid, authToken, nodeParams)
	default:
		return nil, n.CreateError(fmt.Sprintf("unsupported operation: %s", operation), nil)
	}
}

func (n *TwilioNode) sendSms(baseURL, accountSid, authToken string, params map[string]interface{}) ([]model.DataItem, error) {
	from := n.GetStringParameter(params, "from", "")
	to := n.GetStringParameter(params, "to", "")
	body := n.GetStringParameter(params, "body", "")

	if from == "" || to == "" || body == "" {
		return nil, n.CreateError("from, to, and body are required for sendSms", nil)
	}

	formData := url.Values{}
	formData.Set("From", from)
	formData.Set("To", to)
	formData.Set("Body", body)

	reqURL := fmt.Sprintf("%s/Messages.json", baseURL)
	req, err := http.NewRequest("POST", reqURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, n.CreateError(fmt.Sprintf("failed to create request: %v", err), nil)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(accountSid, authToken)

	return n.doRequest(req)
}

func (n *TwilioNode) getMessages(baseURL, accountSid, authToken string, params map[string]interface{}) ([]model.DataItem, error) {
	reqURL := fmt.Sprintf("%s/Messages.json", baseURL)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, n.CreateError(fmt.Sprintf("failed to create request: %v", err), nil)
	}

	req.SetBasicAuth(accountSid, authToken)
	return n.doRequest(req)
}

func (n *TwilioNode) doRequest(req *http.Request) ([]model.DataItem, error) {
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

// ValidateParameters validates Twilio node parameters.
func (n *TwilioNode) ValidateParameters(params map[string]interface{}) error {
	if params == nil {
		return n.CreateError("parameters cannot be nil", nil)
	}

	op := n.GetStringParameter(params, "operation", "sendSms")
	if op != "sendSms" && op != "getMessages" {
		return n.CreateError(fmt.Sprintf("invalid operation: %s", op), nil)
	}

	if n.GetStringParameter(params, "accountSid", "") == "" {
		return n.CreateError("accountSid is required", nil)
	}
	if n.GetStringParameter(params, "authToken", "") == "" {
		return n.CreateError("authToken is required", nil)
	}

	return nil
}
