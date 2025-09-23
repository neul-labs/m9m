package database

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/yourusername/n8n-go/internal/interfaces"
	"github.com/yourusername/n8n-go/internal/nodes/base"
)

// ElasticsearchNode provides Elasticsearch database operations
type ElasticsearchNode struct {
	*base.BaseNode
	httpClient *http.Client
}

// NewElasticsearchNode creates a new Elasticsearch node
func NewElasticsearchNode() interfaces.Node {
	return &ElasticsearchNode{
		BaseNode: base.NewBaseNode("Elasticsearch"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetMetadata returns node metadata
func (n *ElasticsearchNode) GetMetadata() interfaces.NodeMetadata {
	return interfaces.NodeMetadata{
		Name:        "Elasticsearch",
		Version:     "1.0.0",
		Description: "Search, store, and analyze data with Elasticsearch",
		Icon:        "elasticsearch",
		Category:    "Database",
		Credentials: []interfaces.CredentialType{
			{
				Name: "elasticsearchApi",
				Type: "api",
			},
		},
		Properties: []interfaces.NodeProperty{
			{
				Name:        "operation",
				Type:        "options",
				DisplayName: "Operation",
				Description: "The operation to perform",
				Options: []interfaces.PropertyOption{
					{Name: "Index Document", Value: "index"},
					{Name: "Get Document", Value: "get"},
					{Name: "Search", Value: "search"},
					{Name: "Update Document", Value: "update"},
					{Name: "Delete Document", Value: "delete"},
					{Name: "Bulk Operations", Value: "bulk"},
					{Name: "Create Index", Value: "createIndex"},
					{Name: "Delete Index", Value: "deleteIndex"},
					{Name: "Get Mapping", Value: "getMapping"},
					{Name: "Put Mapping", Value: "putMapping"},
					{Name: "Aggregate", Value: "aggregate"},
					{Name: "Count", Value: "count"},
				},
				Default:  "search",
				Required: true,
			},
		},
	}
}

// Execute runs the node
func (n *ElasticsearchNode) Execute(ctx context.Context, params interfaces.ExecutionParams) (interfaces.NodeOutput, error) {
	operation := params.GetString("operation")
	if operation == "" {
		operation = "search"
	}

	// Get credentials
	baseURL, username, password, err := n.getCredentials(params)
	if err != nil {
		return nil, err
	}

	var result interface{}

	switch operation {
	case "index":
		result, err = n.indexDocument(ctx, baseURL, username, password, params)
	case "get":
		result, err = n.getDocument(ctx, baseURL, username, password, params)
	case "search":
		result, err = n.searchDocuments(ctx, baseURL, username, password, params)
	case "update":
		result, err = n.updateDocument(ctx, baseURL, username, password, params)
	case "delete":
		result, err = n.deleteDocument(ctx, baseURL, username, password, params)
	case "bulk":
		result, err = n.bulkOperations(ctx, baseURL, username, password, params)
	case "createIndex":
		result, err = n.createIndex(ctx, baseURL, username, password, params)
	case "deleteIndex":
		result, err = n.deleteIndex(ctx, baseURL, username, password, params)
	case "getMapping":
		result, err = n.getMapping(ctx, baseURL, username, password, params)
	case "putMapping":
		result, err = n.putMapping(ctx, baseURL, username, password, params)
	case "aggregate":
		result, err = n.aggregate(ctx, baseURL, username, password, params)
	case "count":
		result, err = n.countDocuments(ctx, baseURL, username, password, params)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}

	if err != nil {
		return nil, err
	}

	return &base.NodeOutput{
		Data: result,
	}, nil
}

func (n *ElasticsearchNode) getCredentials(params interfaces.ExecutionParams) (string, string, string, error) {
	baseURL := params.GetString("credentials.baseUrl")
	if baseURL == "" {
		baseURL = "http://localhost:9200"
	}

	username := params.GetString("credentials.username")
	password := params.GetString("credentials.password")
	apiKey := params.GetString("credentials.apiKey")

	if apiKey != "" {
		// Use API key authentication
		return baseURL, "", apiKey, nil
	}

	return baseURL, username, password, nil
}

func (n *ElasticsearchNode) makeRequest(ctx context.Context, method, url, username, password string, body interface{}) (map[string]interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	// Set authentication
	if password != "" && username == "" {
		// API key authentication
		req.Header.Set("Authorization", "ApiKey "+password)
	} else if username != "" && password != "" {
		// Basic authentication
		req.SetBasicAuth(username, password)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("elasticsearch error (status %d): %s", resp.StatusCode, respBody)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Document operations

func (n *ElasticsearchNode) indexDocument(ctx context.Context, baseURL, username, password string, params interfaces.ExecutionParams) (interface{}, error) {
	index := params.GetString("index")
	if index == "" {
		return nil, fmt.Errorf("index is required")
	}

	document := params.Get("document")
	if document == nil {
		return nil, fmt.Errorf("document is required")
	}

	documentId := params.GetString("documentId")
	pipeline := params.GetString("pipeline")
	refresh := params.GetString("refresh")

	// Build URL
	var url string
	if documentId != "" {
		url = fmt.Sprintf("%s/%s/_doc/%s", baseURL, index, documentId)
	} else {
		url = fmt.Sprintf("%s/%s/_doc", baseURL, index)
	}

	// Add query parameters
	queryParams := make([]string, 0)
	if pipeline != "" {
		queryParams = append(queryParams, "pipeline="+pipeline)
	}
	if refresh != "" {
		queryParams = append(queryParams, "refresh="+refresh)
	}
	if len(queryParams) > 0 {
		url += "?" + strings.Join(queryParams, "&")
	}

	method := "POST"
	if documentId != "" {
		method = "PUT"
	}

	return n.makeRequest(ctx, method, url, username, password, document)
}

func (n *ElasticsearchNode) getDocument(ctx context.Context, baseURL, username, password string, params interfaces.ExecutionParams) (interface{}, error) {
	index := params.GetString("index")
	documentId := params.GetString("documentId")

	if index == "" || documentId == "" {
		return nil, fmt.Errorf("index and documentId are required")
	}

	url := fmt.Sprintf("%s/%s/_doc/%s", baseURL, index, documentId)

	// Add source filtering if specified
	if source := params.GetString("_source"); source != "" {
		url += "?_source=" + source
	}

	return n.makeRequest(ctx, "GET", url, username, password, nil)
}

func (n *ElasticsearchNode) searchDocuments(ctx context.Context, baseURL, username, password string, params interfaces.ExecutionParams) (interface{}, error) {
	index := params.GetString("index")
	if index == "" {
		index = "_all"
	}

	// Build search body
	searchBody := map[string]interface{}{}

	// Query
	if query := params.Get("query"); query != nil {
		searchBody["query"] = query
	} else if queryString := params.GetString("q"); queryString != "" {
		searchBody["query"] = map[string]interface{}{
			"query_string": map[string]interface{}{
				"query": queryString,
			},
		}
	}

	// Size and from for pagination
	if size := params.GetInt("size"); size > 0 {
		searchBody["size"] = size
	}
	if from := params.GetInt("from"); from > 0 {
		searchBody["from"] = from
	}

	// Sort
	if sort := params.Get("sort"); sort != nil {
		searchBody["sort"] = sort
	}

	// Source filtering
	if source := params.Get("_source"); source != nil {
		searchBody["_source"] = source
	}

	// Aggregations
	if aggs := params.Get("aggs"); aggs != nil {
		searchBody["aggs"] = aggs
	}

	// Highlight
	if highlight := params.Get("highlight"); highlight != nil {
		searchBody["highlight"] = highlight
	}

	url := fmt.Sprintf("%s/%s/_search", baseURL, index)

	// Add query parameters
	queryParams := url.Values{}
	if scroll := params.GetString("scroll"); scroll != "" {
		queryParams.Set("scroll", scroll)
	}
	if trackTotalHits := params.GetBool("track_total_hits"); trackTotalHits {
		queryParams.Set("track_total_hits", "true")
	}

	if len(queryParams) > 0 {
		url += "?" + queryParams.Encode()
	}

	return n.makeRequest(ctx, "POST", url, username, password, searchBody)
}

func (n *ElasticsearchNode) updateDocument(ctx context.Context, baseURL, username, password string, params interfaces.ExecutionParams) (interface{}, error) {
	index := params.GetString("index")
	documentId := params.GetString("documentId")

	if index == "" || documentId == "" {
		return nil, fmt.Errorf("index and documentId are required")
	}

	// Build update body
	updateBody := map[string]interface{}{}

	if doc := params.Get("doc"); doc != nil {
		updateBody["doc"] = doc
	}

	if script := params.Get("script"); script != nil {
		updateBody["script"] = script
	}

	if upsert := params.Get("upsert"); upsert != nil {
		updateBody["upsert"] = upsert
	}

	if docAsUpsert := params.GetBool("doc_as_upsert"); docAsUpsert {
		updateBody["doc_as_upsert"] = true
	}

	url := fmt.Sprintf("%s/%s/_update/%s", baseURL, index, documentId)

	// Add refresh parameter if specified
	if refresh := params.GetString("refresh"); refresh != "" {
		url += "?refresh=" + refresh
	}

	return n.makeRequest(ctx, "POST", url, username, password, updateBody)
}

func (n *ElasticsearchNode) deleteDocument(ctx context.Context, baseURL, username, password string, params interfaces.ExecutionParams) (interface{}, error) {
	index := params.GetString("index")
	documentId := params.GetString("documentId")

	if index == "" || documentId == "" {
		return nil, fmt.Errorf("index and documentId are required")
	}

	url := fmt.Sprintf("%s/%s/_doc/%s", baseURL, index, documentId)

	// Add refresh parameter if specified
	if refresh := params.GetString("refresh"); refresh != "" {
		url += "?refresh=" + refresh
	}

	return n.makeRequest(ctx, "DELETE", url, username, password, nil)
}

func (n *ElasticsearchNode) bulkOperations(ctx context.Context, baseURL, username, password string, params interfaces.ExecutionParams) (interface{}, error) {
	operations := params.GetSlice("operations")
	if len(operations) == 0 {
		return nil, fmt.Errorf("operations are required")
	}

	// Build bulk request body (NDJSON format)
	var bulkBody strings.Builder
	for _, op := range operations {
		opMap, ok := op.(map[string]interface{})
		if !ok {
			continue
		}

		// Action line
		if action := opMap["action"]; action != nil {
			actionJSON, err := json.Marshal(action)
			if err != nil {
				return nil, err
			}
			bulkBody.WriteString(string(actionJSON) + "\n")
		}

		// Document line (for index and update operations)
		if doc := opMap["document"]; doc != nil {
			docJSON, err := json.Marshal(doc)
			if err != nil {
				return nil, err
			}
			bulkBody.WriteString(string(docJSON) + "\n")
		}
	}

	url := fmt.Sprintf("%s/_bulk", baseURL)

	// Add index if specified
	if index := params.GetString("index"); index != "" {
		url = fmt.Sprintf("%s/%s/_bulk", baseURL, index)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(bulkBody.String()))
	if err != nil {
		return nil, err
	}

	// Set authentication
	if password != "" && username == "" {
		req.Header.Set("Authorization", "ApiKey "+password)
	} else if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	req.Header.Set("Content-Type", "application/x-ndjson")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("elasticsearch bulk error (status %d): %s", resp.StatusCode, respBody)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Index operations

func (n *ElasticsearchNode) createIndex(ctx context.Context, baseURL, username, password string, params interfaces.ExecutionParams) (interface{}, error) {
	index := params.GetString("index")
	if index == "" {
		return nil, fmt.Errorf("index is required")
	}

	body := map[string]interface{}{}

	// Settings
	if settings := params.Get("settings"); settings != nil {
		body["settings"] = settings
	}

	// Mappings
	if mappings := params.Get("mappings"); mappings != nil {
		body["mappings"] = mappings
	}

	// Aliases
	if aliases := params.Get("aliases"); aliases != nil {
		body["aliases"] = aliases
	}

	url := fmt.Sprintf("%s/%s", baseURL, index)

	var bodyToSend interface{}
	if len(body) > 0 {
		bodyToSend = body
	}

	return n.makeRequest(ctx, "PUT", url, username, password, bodyToSend)
}

func (n *ElasticsearchNode) deleteIndex(ctx context.Context, baseURL, username, password string, params interfaces.ExecutionParams) (interface{}, error) {
	index := params.GetString("index")
	if index == "" {
		return nil, fmt.Errorf("index is required")
	}

	url := fmt.Sprintf("%s/%s", baseURL, index)

	return n.makeRequest(ctx, "DELETE", url, username, password, nil)
}

func (n *ElasticsearchNode) getMapping(ctx context.Context, baseURL, username, password string, params interfaces.ExecutionParams) (interface{}, error) {
	index := params.GetString("index")
	if index == "" {
		index = "_all"
	}

	url := fmt.Sprintf("%s/%s/_mapping", baseURL, index)

	return n.makeRequest(ctx, "GET", url, username, password, nil)
}

func (n *ElasticsearchNode) putMapping(ctx context.Context, baseURL, username, password string, params interfaces.ExecutionParams) (interface{}, error) {
	index := params.GetString("index")
	if index == "" {
		return nil, fmt.Errorf("index is required")
	}

	properties := params.Get("properties")
	if properties == nil {
		return nil, fmt.Errorf("properties are required")
	}

	body := map[string]interface{}{
		"properties": properties,
	}

	url := fmt.Sprintf("%s/%s/_mapping", baseURL, index)

	return n.makeRequest(ctx, "PUT", url, username, password, body)
}

// Analytics operations

func (n *ElasticsearchNode) aggregate(ctx context.Context, baseURL, username, password string, params interfaces.ExecutionParams) (interface{}, error) {
	index := params.GetString("index")
	if index == "" {
		index = "_all"
	}

	aggs := params.Get("aggs")
	if aggs == nil {
		return nil, fmt.Errorf("aggregations are required")
	}

	body := map[string]interface{}{
		"size": 0, // We only want aggregation results
		"aggs": aggs,
	}

	// Add query if specified
	if query := params.Get("query"); query != nil {
		body["query"] = query
	}

	url := fmt.Sprintf("%s/%s/_search", baseURL, index)

	return n.makeRequest(ctx, "POST", url, username, password, body)
}

func (n *ElasticsearchNode) countDocuments(ctx context.Context, baseURL, username, password string, params interfaces.ExecutionParams) (interface{}, error) {
	index := params.GetString("index")
	if index == "" {
		index = "_all"
	}

	body := map[string]interface{}{}

	// Add query if specified
	if query := params.Get("query"); query != nil {
		body["query"] = query
	}

	url := fmt.Sprintf("%s/%s/_count", baseURL, index)

	var bodyToSend interface{}
	if len(body) > 0 {
		bodyToSend = body
	}

	return n.makeRequest(ctx, "POST", url, username, password, bodyToSend)
}