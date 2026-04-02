package productivity

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

// StripeHTTPClient abstracts the HTTP client for testing.
type StripeHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// StripeNode implements Stripe payment operations.
type StripeNode struct {
	*base.BaseNode
	client StripeHTTPClient
}

// NewStripeNode creates a new Stripe node.
func NewStripeNode() *StripeNode {
	return &StripeNode{
		BaseNode: base.NewBaseNode(base.NodeDescription{
			Name:        "Stripe",
			Description: "Create customers and payment intents via Stripe",
			Category:    "Productivity",
		}),
		client: &http.Client{},
	}
}

// NewStripeNodeWithClient creates a Stripe node with a custom HTTP client.
func NewStripeNodeWithClient(client StripeHTTPClient) *StripeNode {
	n := NewStripeNode()
	n.client = client
	return n
}

// Execute runs the configured Stripe operation.
func (n *StripeNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	baseURL := n.GetStringParameter(nodeParams, "baseUrl", "https://api.stripe.com/v1")
	baseURL = strings.TrimRight(baseURL, "/")
	operation := n.GetStringParameter(nodeParams, "operation", "createCustomer")
	apiKey := n.GetStringParameter(nodeParams, "apiKey", "")

	if apiKey == "" {
		return nil, n.CreateError("apiKey is required", nil)
	}

	switch operation {
	case "createCustomer":
		return n.createCustomer(baseURL, apiKey, nodeParams)
	case "createPaymentIntent":
		return n.createPaymentIntent(baseURL, apiKey, nodeParams)
	case "listPayments":
		return n.listPayments(baseURL, apiKey, nodeParams)
	default:
		return nil, n.CreateError(fmt.Sprintf("unsupported operation: %s", operation), nil)
	}
}

func (n *StripeNode) createCustomer(baseURL, apiKey string, params map[string]interface{}) ([]model.DataItem, error) {
	form := url.Values{}
	if email := n.GetStringParameter(params, "email", ""); email != "" {
		form.Set("email", email)
	}
	if name := n.GetStringParameter(params, "name", ""); name != "" {
		form.Set("name", name)
	}
	if desc := n.GetStringParameter(params, "description", ""); desc != "" {
		form.Set("description", desc)
	}

	return n.doFormRequest("POST", baseURL+"/customers", apiKey, form)
}

func (n *StripeNode) createPaymentIntent(baseURL, apiKey string, params map[string]interface{}) ([]model.DataItem, error) {
	amount := n.GetIntParameter(params, "amount", 0)
	currency := n.GetStringParameter(params, "currency", "usd")

	if amount <= 0 {
		return nil, n.CreateError("amount must be greater than 0", nil)
	}

	form := url.Values{}
	form.Set("amount", fmt.Sprintf("%d", amount))
	form.Set("currency", currency)

	if customerId := n.GetStringParameter(params, "customerId", ""); customerId != "" {
		form.Set("customer", customerId)
	}
	if desc := n.GetStringParameter(params, "description", ""); desc != "" {
		form.Set("description", desc)
	}

	return n.doFormRequest("POST", baseURL+"/payment_intents", apiKey, form)
}

func (n *StripeNode) listPayments(baseURL, apiKey string, params map[string]interface{}) ([]model.DataItem, error) {
	reqURL := baseURL + "/payment_intents"
	if limit := n.GetIntParameter(params, "limit", 0); limit > 0 {
		reqURL = fmt.Sprintf("%s?limit=%d", reqURL, limit)
	}

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, n.CreateError(fmt.Sprintf("failed to create request: %v", err), nil)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	return n.executeRequest(req)
}

func (n *StripeNode) doFormRequest(method, reqURL, apiKey string, form url.Values) ([]model.DataItem, error) {
	req, err := http.NewRequest(method, reqURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, n.CreateError(fmt.Sprintf("failed to create request: %v", err), nil)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	return n.executeRequest(req)
}

func (n *StripeNode) executeRequest(req *http.Request) ([]model.DataItem, error) {
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

// ValidateParameters validates Stripe node parameters.
func (n *StripeNode) ValidateParameters(params map[string]interface{}) error {
	if params == nil {
		return n.CreateError("parameters cannot be nil", nil)
	}

	op := n.GetStringParameter(params, "operation", "createCustomer")
	validOps := map[string]bool{
		"createCustomer": true, "createPaymentIntent": true, "listPayments": true,
	}
	if !validOps[op] {
		return n.CreateError(fmt.Sprintf("invalid operation: %s", op), nil)
	}

	if n.GetStringParameter(params, "apiKey", "") == "" {
		return n.CreateError("apiKey is required", nil)
	}

	return nil
}
