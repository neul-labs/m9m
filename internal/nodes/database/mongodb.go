package database

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// MongoDBHTTPClient abstracts the HTTP client for testing.
// This implementation uses a MongoDB REST API (Atlas Data API or similar).
type MongoDBHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// MongoDBNode implements MongoDB operations via REST API.
type MongoDBNode struct {
	*base.BaseNode
	client MongoDBHTTPClient
}

// NewMongoDBNode creates a new MongoDB node.
func NewMongoDBNode() *MongoDBNode {
	return &MongoDBNode{
		BaseNode: base.NewBaseNode(base.NodeDescription{
			Name:        "MongoDB",
			Description: "Perform operations on MongoDB collections",
			Category:    "Database",
		}),
		client: &http.Client{},
	}
}

// NewMongoDBNodeWithClient creates a MongoDB node with a custom HTTP client.
func NewMongoDBNodeWithClient(client MongoDBHTTPClient) *MongoDBNode {
	n := NewMongoDBNode()
	n.client = client
	return n
}

// Execute runs the configured MongoDB operation.
func (n *MongoDBNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	baseURL := n.GetStringParameter(nodeParams, "baseUrl", "http://localhost:27017")
	baseURL = strings.TrimRight(baseURL, "/")
	operation := n.GetStringParameter(nodeParams, "operation", "find")
	database := n.GetStringParameter(nodeParams, "database", "")
	collection := n.GetStringParameter(nodeParams, "collection", "")

	if database == "" {
		return nil, n.CreateError("database is required", nil)
	}
	if collection == "" {
		return nil, n.CreateError("collection is required", nil)
	}

	switch operation {
	case "find":
		return n.doFind(baseURL, database, collection, nodeParams)
	case "findOne":
		return n.doFindOne(baseURL, database, collection, nodeParams)
	case "insert":
		return n.doInsert(baseURL, database, collection, nodeParams, inputData)
	case "insertMany":
		return n.doInsertMany(baseURL, database, collection, nodeParams, inputData)
	case "update":
		return n.doUpdate(baseURL, database, collection, nodeParams)
	case "delete":
		return n.doDelete(baseURL, database, collection, nodeParams)
	case "aggregate":
		return n.doAggregate(baseURL, database, collection, nodeParams)
	case "count":
		return n.doCount(baseURL, database, collection, nodeParams)
	default:
		return nil, n.CreateError(fmt.Sprintf("unsupported operation: %s", operation), nil)
	}
}

func (n *MongoDBNode) doFind(baseURL, db, coll string, params map[string]interface{}) ([]model.DataItem, error) {
	body := map[string]interface{}{
		"dataSource": n.GetStringParameter(params, "dataSource", "Cluster0"),
		"database":   db,
		"collection": coll,
	}
	if filter, ok := params["filter"]; ok {
		body["filter"] = filter
	}
	if limit := n.GetIntParameter(params, "limit", 0); limit > 0 {
		body["limit"] = limit
	}
	if sort, ok := params["sort"]; ok {
		body["sort"] = sort
	}

	url := fmt.Sprintf("%s/action/find", baseURL)
	return n.sendRequest(url, body, params)
}

func (n *MongoDBNode) doFindOne(baseURL, db, coll string, params map[string]interface{}) ([]model.DataItem, error) {
	body := map[string]interface{}{
		"dataSource": n.GetStringParameter(params, "dataSource", "Cluster0"),
		"database":   db,
		"collection": coll,
	}
	if filter, ok := params["filter"]; ok {
		body["filter"] = filter
	}

	url := fmt.Sprintf("%s/action/findOne", baseURL)
	return n.sendRequest(url, body, params)
}

func (n *MongoDBNode) doInsert(baseURL, db, coll string, params map[string]interface{}, inputData []model.DataItem) ([]model.DataItem, error) {
	var document interface{}
	if doc, ok := params["document"]; ok {
		document = doc
	} else if len(inputData) > 0 {
		document = inputData[0].JSON
	} else {
		return nil, n.CreateError("document or input data is required", nil)
	}

	body := map[string]interface{}{
		"dataSource": n.GetStringParameter(params, "dataSource", "Cluster0"),
		"database":   db,
		"collection": coll,
		"document":   document,
	}

	url := fmt.Sprintf("%s/action/insertOne", baseURL)
	return n.sendRequest(url, body, params)
}

func (n *MongoDBNode) doInsertMany(baseURL, db, coll string, params map[string]interface{}, inputData []model.DataItem) ([]model.DataItem, error) {
	var documents []interface{}
	if docs, ok := params["documents"].([]interface{}); ok {
		documents = docs
	} else {
		for _, item := range inputData {
			documents = append(documents, item.JSON)
		}
	}

	if len(documents) == 0 {
		return nil, n.CreateError("documents or input data is required", nil)
	}

	body := map[string]interface{}{
		"dataSource": n.GetStringParameter(params, "dataSource", "Cluster0"),
		"database":   db,
		"collection": coll,
		"documents":  documents,
	}

	url := fmt.Sprintf("%s/action/insertMany", baseURL)
	return n.sendRequest(url, body, params)
}

func (n *MongoDBNode) doUpdate(baseURL, db, coll string, params map[string]interface{}) ([]model.DataItem, error) {
	filter, ok := params["filter"]
	if !ok {
		return nil, n.CreateError("filter is required for update", nil)
	}
	update, ok := params["update"]
	if !ok {
		return nil, n.CreateError("update is required for update", nil)
	}

	body := map[string]interface{}{
		"dataSource": n.GetStringParameter(params, "dataSource", "Cluster0"),
		"database":   db,
		"collection": coll,
		"filter":     filter,
		"update":     update,
	}

	url := fmt.Sprintf("%s/action/updateMany", baseURL)
	return n.sendRequest(url, body, params)
}

func (n *MongoDBNode) doDelete(baseURL, db, coll string, params map[string]interface{}) ([]model.DataItem, error) {
	filter, ok := params["filter"]
	if !ok {
		return nil, n.CreateError("filter is required for delete", nil)
	}

	body := map[string]interface{}{
		"dataSource": n.GetStringParameter(params, "dataSource", "Cluster0"),
		"database":   db,
		"collection": coll,
		"filter":     filter,
	}

	url := fmt.Sprintf("%s/action/deleteMany", baseURL)
	return n.sendRequest(url, body, params)
}

func (n *MongoDBNode) doAggregate(baseURL, db, coll string, params map[string]interface{}) ([]model.DataItem, error) {
	pipeline, ok := params["pipeline"]
	if !ok {
		return nil, n.CreateError("pipeline is required for aggregate", nil)
	}

	body := map[string]interface{}{
		"dataSource": n.GetStringParameter(params, "dataSource", "Cluster0"),
		"database":   db,
		"collection": coll,
		"pipeline":   pipeline,
	}

	url := fmt.Sprintf("%s/action/aggregate", baseURL)
	return n.sendRequest(url, body, params)
}

func (n *MongoDBNode) doCount(baseURL, db, coll string, params map[string]interface{}) ([]model.DataItem, error) {
	body := map[string]interface{}{
		"dataSource": n.GetStringParameter(params, "dataSource", "Cluster0"),
		"database":   db,
		"collection": coll,
	}
	if filter, ok := params["filter"]; ok {
		body["filter"] = filter
	}

	// Use find with count projection via aggregate
	url := fmt.Sprintf("%s/action/aggregate", baseURL)
	body["pipeline"] = []interface{}{
		map[string]interface{}{"$count": "count"},
	}
	if filter, ok := params["filter"]; ok {
		body["pipeline"] = []interface{}{
			map[string]interface{}{"$match": filter},
			map[string]interface{}{"$count": "count"},
		}
	}

	return n.sendRequest(url, body, params)
}

func (n *MongoDBNode) sendRequest(url string, body interface{}, params map[string]interface{}) ([]model.DataItem, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, n.CreateError(fmt.Sprintf("failed to marshal request: %v", err), nil)
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(data)))
	if err != nil {
		return nil, n.CreateError(fmt.Sprintf("failed to create request: %v", err), nil)
	}

	req.Header.Set("Content-Type", "application/json")

	apiKey := n.GetStringParameter(params, "apiKey", "")
	if apiKey != "" {
		req.Header.Set("api-key", apiKey)
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

// ValidateParameters validates MongoDB node parameters.
func (n *MongoDBNode) ValidateParameters(params map[string]interface{}) error {
	if params == nil {
		return n.CreateError("parameters cannot be nil", nil)
	}

	operation := n.GetStringParameter(params, "operation", "find")
	validOps := map[string]bool{
		"find": true, "findOne": true, "insert": true, "insertMany": true,
		"update": true, "delete": true, "aggregate": true, "count": true,
	}
	if !validOps[operation] {
		return n.CreateError(fmt.Sprintf("invalid operation: %s", operation), nil)
	}

	if n.GetStringParameter(params, "database", "") == "" {
		return n.CreateError("database is required", nil)
	}
	if n.GetStringParameter(params, "collection", "") == "" {
		return n.CreateError("collection is required", nil)
	}

	return nil
}
