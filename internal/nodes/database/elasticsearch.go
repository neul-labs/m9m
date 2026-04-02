package database

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// ElasticsearchHTTPClient abstracts the HTTP client for testing.
type ElasticsearchHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// ElasticsearchNode implements Elasticsearch operations over HTTP.
type ElasticsearchNode struct {
	*base.BaseNode
	client ElasticsearchHTTPClient
}

// NewElasticsearchNode creates a new Elasticsearch node.
func NewElasticsearchNode() *ElasticsearchNode {
	return &ElasticsearchNode{
		BaseNode: base.NewBaseNode(base.NodeDescription{
			Name:        "Elasticsearch",
			Description: "Perform operations on Elasticsearch indices",
			Category:    "Database",
		}),
		client: &http.Client{},
	}
}

// NewElasticsearchNodeWithClient creates an Elasticsearch node with a custom HTTP client.
func NewElasticsearchNodeWithClient(client ElasticsearchHTTPClient) *ElasticsearchNode {
	n := NewElasticsearchNode()
	n.client = client
	return n
}

// Execute runs the configured Elasticsearch operation.
func (n *ElasticsearchNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	baseURL := n.GetStringParameter(nodeParams, "baseUrl", "http://localhost:9200")
	baseURL = strings.TrimRight(baseURL, "/")
	operation := n.GetStringParameter(nodeParams, "operation", "search")
	index := n.GetStringParameter(nodeParams, "index", "")

	switch operation {
	case "index":
		return n.doIndex(baseURL, index, nodeParams, inputData)
	case "get":
		return n.doGet(baseURL, index, nodeParams)
	case "search":
		return n.doSearch(baseURL, index, nodeParams)
	case "update":
		return n.doUpdate(baseURL, index, nodeParams)
	case "delete":
		return n.doDelete(baseURL, index, nodeParams)
	case "bulk":
		return n.doBulk(baseURL, index, nodeParams, inputData)
	case "createIndex":
		return n.doCreateIndex(baseURL, index, nodeParams)
	default:
		return nil, n.CreateError(fmt.Sprintf("unsupported operation: %s", operation), nil)
	}
}

func (n *ElasticsearchNode) doIndex(baseURL, index string, params map[string]interface{}, inputData []model.DataItem) ([]model.DataItem, error) {
	if index == "" {
		return nil, n.CreateError("index is required for index operation", nil)
	}

	docID := n.GetStringParameter(params, "documentId", "")
	var body interface{}
	if doc, ok := params["document"]; ok {
		body = doc
	} else if len(inputData) > 0 {
		body = inputData[0].JSON
	} else {
		return nil, n.CreateError("document or input data is required", nil)
	}

	url := fmt.Sprintf("%s/%s/_doc", baseURL, index)
	method := "POST"
	if docID != "" {
		url = fmt.Sprintf("%s/%s/_doc/%s", baseURL, index, docID)
		method = "PUT"
	}

	return n.doRequest(method, url, body, params)
}

func (n *ElasticsearchNode) doGet(baseURL, index string, params map[string]interface{}) ([]model.DataItem, error) {
	if index == "" {
		return nil, n.CreateError("index is required for get operation", nil)
	}
	docID := n.GetStringParameter(params, "documentId", "")
	if docID == "" {
		return nil, n.CreateError("documentId is required for get operation", nil)
	}

	url := fmt.Sprintf("%s/%s/_doc/%s", baseURL, index, docID)
	return n.doRequest("GET", url, nil, params)
}

func (n *ElasticsearchNode) doSearch(baseURL, index string, params map[string]interface{}) ([]model.DataItem, error) {
	url := fmt.Sprintf("%s/%s/_search", baseURL, index)
	if index == "" {
		url = fmt.Sprintf("%s/_search", baseURL)
	}

	query := params["query"]
	return n.doRequest("POST", url, query, params)
}

func (n *ElasticsearchNode) doUpdate(baseURL, index string, params map[string]interface{}) ([]model.DataItem, error) {
	if index == "" {
		return nil, n.CreateError("index is required for update operation", nil)
	}
	docID := n.GetStringParameter(params, "documentId", "")
	if docID == "" {
		return nil, n.CreateError("documentId is required for update operation", nil)
	}

	body := map[string]interface{}{"doc": params["document"]}
	url := fmt.Sprintf("%s/%s/_update/%s", baseURL, index, docID)
	return n.doRequest("POST", url, body, params)
}

func (n *ElasticsearchNode) doDelete(baseURL, index string, params map[string]interface{}) ([]model.DataItem, error) {
	if index == "" {
		return nil, n.CreateError("index is required for delete operation", nil)
	}
	docID := n.GetStringParameter(params, "documentId", "")
	if docID == "" {
		return nil, n.CreateError("documentId is required for delete operation", nil)
	}

	url := fmt.Sprintf("%s/%s/_doc/%s", baseURL, index, docID)
	return n.doRequest("DELETE", url, nil, params)
}

func (n *ElasticsearchNode) doBulk(baseURL, index string, params map[string]interface{}, inputData []model.DataItem) ([]model.DataItem, error) {
	if index == "" {
		return nil, n.CreateError("index is required for bulk operation", nil)
	}

	var buf bytes.Buffer
	for _, item := range inputData {
		action := map[string]interface{}{"index": map[string]interface{}{"_index": index}}
		actionLine, _ := json.Marshal(action)
		buf.Write(actionLine)
		buf.WriteByte('\n')
		docLine, _ := json.Marshal(item.JSON)
		buf.Write(docLine)
		buf.WriteByte('\n')
	}

	url := fmt.Sprintf("%s/_bulk", baseURL)
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, n.CreateError(fmt.Sprintf("failed to create request: %v", err), nil)
	}
	req.Header.Set("Content-Type", "application/x-ndjson")
	n.setAuth(req, params)

	resp, err := n.client.Do(req)
	if err != nil {
		return nil, n.CreateError(fmt.Sprintf("request failed: %v", err), nil)
	}
	defer resp.Body.Close()

	return n.parseResponse(resp)
}

func (n *ElasticsearchNode) doCreateIndex(baseURL, index string, params map[string]interface{}) ([]model.DataItem, error) {
	if index == "" {
		return nil, n.CreateError("index is required for createIndex operation", nil)
	}

	settings := params["settings"]
	url := fmt.Sprintf("%s/%s", baseURL, index)
	return n.doRequest("PUT", url, settings, params)
}

func (n *ElasticsearchNode) doRequest(method, url string, body interface{}, params map[string]interface{}) ([]model.DataItem, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, n.CreateError(fmt.Sprintf("failed to marshal body: %v", err), nil)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, n.CreateError(fmt.Sprintf("failed to create request: %v", err), nil)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	n.setAuth(req, params)

	resp, err := n.client.Do(req)
	if err != nil {
		return nil, n.CreateError(fmt.Sprintf("request failed: %v", err), nil)
	}
	defer resp.Body.Close()

	return n.parseResponse(resp)
}

func (n *ElasticsearchNode) setAuth(req *http.Request, params map[string]interface{}) {
	user := n.GetStringParameter(params, "username", "")
	pass := n.GetStringParameter(params, "password", "")
	if user != "" {
		req.SetBasicAuth(user, pass)
	}
}

func (n *ElasticsearchNode) parseResponse(resp *http.Response) ([]model.DataItem, error) {
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

// ValidateParameters validates Elasticsearch node parameters.
func (n *ElasticsearchNode) ValidateParameters(params map[string]interface{}) error {
	if params == nil {
		return n.CreateError("parameters cannot be nil", nil)
	}

	operation := n.GetStringParameter(params, "operation", "search")
	validOps := map[string]bool{
		"index": true, "get": true, "search": true, "update": true,
		"delete": true, "bulk": true, "createIndex": true,
	}
	if !validOps[operation] {
		return n.CreateError(fmt.Sprintf("invalid operation: %s", operation), nil)
	}

	return nil
}
