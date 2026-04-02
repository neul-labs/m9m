package productivity

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// NotionHTTPClient abstracts the HTTP client for testing.
type NotionHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// NotionNode implements Notion API operations.
type NotionNode struct {
	*base.BaseNode
	client NotionHTTPClient
}

// NewNotionNode creates a new Notion node.
func NewNotionNode() *NotionNode {
	return &NotionNode{
		BaseNode: base.NewBaseNode(base.NodeDescription{
			Name:        "Notion",
			Description: "Create, read, and update Notion pages and databases",
			Category:    "Productivity",
		}),
		client: &http.Client{},
	}
}

// NewNotionNodeWithClient creates a Notion node with a custom HTTP client.
func NewNotionNodeWithClient(client NotionHTTPClient) *NotionNode {
	n := NewNotionNode()
	n.client = client
	return n
}

// Execute runs the configured Notion operation.
func (n *NotionNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	baseURL := n.GetStringParameter(nodeParams, "baseUrl", "https://api.notion.com/v1")
	baseURL = strings.TrimRight(baseURL, "/")
	operation := n.GetStringParameter(nodeParams, "operation", "getPage")
	apiKey := n.GetStringParameter(nodeParams, "apiKey", "")

	if apiKey == "" {
		return nil, n.CreateError("apiKey is required", nil)
	}

	switch operation {
	case "createPage":
		return n.createPage(baseURL, apiKey, nodeParams)
	case "getPage":
		return n.getPage(baseURL, apiKey, nodeParams)
	case "updatePage":
		return n.updatePage(baseURL, apiKey, nodeParams)
	case "queryDatabase":
		return n.queryDatabase(baseURL, apiKey, nodeParams)
	case "search":
		return n.search(baseURL, apiKey, nodeParams)
	default:
		return nil, n.CreateError(fmt.Sprintf("unsupported operation: %s", operation), nil)
	}
}

func (n *NotionNode) createPage(baseURL, apiKey string, params map[string]interface{}) ([]model.DataItem, error) {
	parentID := n.GetStringParameter(params, "parentId", "")
	if parentID == "" {
		return nil, n.CreateError("parentId is required for createPage", nil)
	}

	body := map[string]interface{}{
		"parent": map[string]interface{}{
			"database_id": parentID,
		},
	}
	if props, ok := params["properties"]; ok {
		body["properties"] = props
	}

	return n.doRequest("POST", baseURL+"/pages", apiKey, body)
}

func (n *NotionNode) getPage(baseURL, apiKey string, params map[string]interface{}) ([]model.DataItem, error) {
	pageID := n.GetStringParameter(params, "pageId", "")
	if pageID == "" {
		return nil, n.CreateError("pageId is required for getPage", nil)
	}

	return n.doRequest("GET", fmt.Sprintf("%s/pages/%s", baseURL, pageID), apiKey, nil)
}

func (n *NotionNode) updatePage(baseURL, apiKey string, params map[string]interface{}) ([]model.DataItem, error) {
	pageID := n.GetStringParameter(params, "pageId", "")
	if pageID == "" {
		return nil, n.CreateError("pageId is required for updatePage", nil)
	}

	body := map[string]interface{}{}
	if props, ok := params["properties"]; ok {
		body["properties"] = props
	}

	return n.doRequest("PATCH", fmt.Sprintf("%s/pages/%s", baseURL, pageID), apiKey, body)
}

func (n *NotionNode) queryDatabase(baseURL, apiKey string, params map[string]interface{}) ([]model.DataItem, error) {
	dbID := n.GetStringParameter(params, "databaseId", "")
	if dbID == "" {
		return nil, n.CreateError("databaseId is required for queryDatabase", nil)
	}

	body := map[string]interface{}{}
	if filter, ok := params["filter"]; ok {
		body["filter"] = filter
	}
	if sort, ok := params["sort"]; ok {
		body["sorts"] = sort
	}

	return n.doRequest("POST", fmt.Sprintf("%s/databases/%s/query", baseURL, dbID), apiKey, body)
}

func (n *NotionNode) search(baseURL, apiKey string, params map[string]interface{}) ([]model.DataItem, error) {
	body := map[string]interface{}{}
	if query := n.GetStringParameter(params, "query", ""); query != "" {
		body["query"] = query
	}
	if filter, ok := params["filter"]; ok {
		body["filter"] = filter
	}

	return n.doRequest("POST", baseURL+"/search", apiKey, body)
}

func (n *NotionNode) doRequest(method, url, apiKey string, body interface{}) ([]model.DataItem, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, n.CreateError(fmt.Sprintf("failed to marshal body: %v", err), nil)
		}
		bodyReader = strings.NewReader(string(data))
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, n.CreateError(fmt.Sprintf("failed to create request: %v", err), nil)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Notion-Version", "2022-06-28")
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

// ValidateParameters validates Notion node parameters.
func (n *NotionNode) ValidateParameters(params map[string]interface{}) error {
	if params == nil {
		return n.CreateError("parameters cannot be nil", nil)
	}

	op := n.GetStringParameter(params, "operation", "getPage")
	validOps := map[string]bool{
		"createPage": true, "getPage": true, "updatePage": true,
		"queryDatabase": true, "search": true,
	}
	if !validOps[op] {
		return n.CreateError(fmt.Sprintf("invalid operation: %s", op), nil)
	}

	if n.GetStringParameter(params, "apiKey", "") == "" {
		return n.CreateError("apiKey is required", nil)
	}

	return nil
}
