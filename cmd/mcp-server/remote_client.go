package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/neul-labs/m9m/internal/connections"
	"github.com/neul-labs/m9m/internal/credentials"
	"github.com/neul-labs/m9m/internal/engine"
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
	"github.com/neul-labs/m9m/internal/storage"
)

// RemoteClient implements both WorkflowStorage and WorkflowEngine interfaces
// by proxying calls to a remote m9m API server.
type RemoteClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewRemoteClient creates a new remote client for cloud mode
func NewRemoteClient(apiURL string) (*RemoteClient, error) {
	// Validate URL
	parsed, err := url.Parse(apiURL)
	if err != nil {
		return nil, fmt.Errorf("invalid API URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("API URL must use http or https scheme")
	}

	return &RemoteClient{
		baseURL: apiURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// ============================================================================
// WorkflowStorage Implementation
// ============================================================================

func (c *RemoteClient) SaveWorkflow(workflow *model.Workflow) error {
	if workflow.ID == "" {
		// Create new workflow
		return c.post("/api/v1/workflows", workflow, workflow)
	}
	// Update existing workflow
	return c.put(fmt.Sprintf("/api/v1/workflows/%s", workflow.ID), workflow, workflow)
}

func (c *RemoteClient) GetWorkflow(id string) (*model.Workflow, error) {
	var workflow model.Workflow
	if err := c.get(fmt.Sprintf("/api/v1/workflows/%s", id), &workflow); err != nil {
		return nil, err
	}
	return &workflow, nil
}

func (c *RemoteClient) ListWorkflows(filters storage.WorkflowFilters) ([]*model.Workflow, int, error) {
	params := url.Values{}
	if filters.Active != nil {
		params.Set("active", fmt.Sprintf("%v", *filters.Active))
	}
	if filters.Search != "" {
		params.Set("search", filters.Search)
	}
	if filters.Limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", filters.Limit))
	}
	if filters.Offset > 0 {
		params.Set("offset", fmt.Sprintf("%d", filters.Offset))
	}

	var response struct {
		Workflows []*model.Workflow `json:"workflows"`
		Total     int               `json:"total"`
	}

	path := "/api/v1/workflows"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	if err := c.get(path, &response); err != nil {
		return nil, 0, err
	}

	return response.Workflows, response.Total, nil
}

func (c *RemoteClient) UpdateWorkflow(id string, workflow *model.Workflow) error {
	workflow.ID = id
	return c.put(fmt.Sprintf("/api/v1/workflows/%s", id), workflow, workflow)
}

func (c *RemoteClient) DeleteWorkflow(id string) error {
	return c.delete(fmt.Sprintf("/api/v1/workflows/%s", id))
}

func (c *RemoteClient) ActivateWorkflow(id string) error {
	return c.post(fmt.Sprintf("/api/v1/workflows/%s/activate", id), nil, nil)
}

func (c *RemoteClient) DeactivateWorkflow(id string) error {
	return c.post(fmt.Sprintf("/api/v1/workflows/%s/deactivate", id), nil, nil)
}

// Execution operations

func (c *RemoteClient) SaveExecution(execution *model.WorkflowExecution) error {
	return c.post("/api/v1/executions", execution, execution)
}

func (c *RemoteClient) GetExecution(id string) (*model.WorkflowExecution, error) {
	var execution model.WorkflowExecution
	if err := c.get(fmt.Sprintf("/api/v1/executions/%s", id), &execution); err != nil {
		return nil, err
	}
	return &execution, nil
}

func (c *RemoteClient) ListExecutions(filters storage.ExecutionFilters) ([]*model.WorkflowExecution, int, error) {
	params := url.Values{}
	if filters.WorkflowID != "" {
		params.Set("workflowId", filters.WorkflowID)
	}
	if filters.Status != "" {
		params.Set("status", filters.Status)
	}
	if filters.Limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", filters.Limit))
	}
	if filters.Offset > 0 {
		params.Set("offset", fmt.Sprintf("%d", filters.Offset))
	}

	var response struct {
		Executions []*model.WorkflowExecution `json:"executions"`
		Total      int                        `json:"total"`
	}

	path := "/api/v1/executions"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	if err := c.get(path, &response); err != nil {
		return nil, 0, err
	}

	return response.Executions, response.Total, nil
}

func (c *RemoteClient) DeleteExecution(id string) error {
	return c.delete(fmt.Sprintf("/api/v1/executions/%s", id))
}

// Credential operations

func (c *RemoteClient) SaveCredential(credential *storage.Credential) error {
	return c.post("/api/v1/credentials", credential, credential)
}

func (c *RemoteClient) GetCredential(id string) (*storage.Credential, error) {
	var credential storage.Credential
	if err := c.get(fmt.Sprintf("/api/v1/credentials/%s", id), &credential); err != nil {
		return nil, err
	}
	return &credential, nil
}

func (c *RemoteClient) ListCredentials() ([]*storage.Credential, error) {
	var response struct {
		Credentials []*storage.Credential `json:"credentials"`
	}
	if err := c.get("/api/v1/credentials", &response); err != nil {
		return nil, err
	}
	return response.Credentials, nil
}

func (c *RemoteClient) UpdateCredential(id string, credential *storage.Credential) error {
	credential.ID = id
	return c.put(fmt.Sprintf("/api/v1/credentials/%s", id), credential, credential)
}

func (c *RemoteClient) DeleteCredential(id string) error {
	return c.delete(fmt.Sprintf("/api/v1/credentials/%s", id))
}

// Tag operations

func (c *RemoteClient) SaveTag(tag *storage.Tag) error {
	return c.post("/api/v1/tags", tag, tag)
}

func (c *RemoteClient) GetTag(id string) (*storage.Tag, error) {
	var tag storage.Tag
	if err := c.get(fmt.Sprintf("/api/v1/tags/%s", id), &tag); err != nil {
		return nil, err
	}
	return &tag, nil
}

func (c *RemoteClient) ListTags() ([]*storage.Tag, error) {
	var response struct {
		Tags []*storage.Tag `json:"tags"`
	}
	if err := c.get("/api/v1/tags", &response); err != nil {
		return nil, err
	}
	return response.Tags, nil
}

func (c *RemoteClient) UpdateTag(id string, tag *storage.Tag) error {
	tag.ID = id
	return c.put(fmt.Sprintf("/api/v1/tags/%s", id), tag, tag)
}

func (c *RemoteClient) DeleteTag(id string) error {
	return c.delete(fmt.Sprintf("/api/v1/tags/%s", id))
}

// Raw key-value operations

func (c *RemoteClient) SaveRaw(key string, value []byte) error {
	return c.post("/api/v1/raw/"+key, value, nil)
}

func (c *RemoteClient) GetRaw(key string) ([]byte, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/v1/raw/" + key)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func (c *RemoteClient) ListKeys(prefix string) ([]string, error) {
	var keys []string
	if err := c.get("/api/v1/raw?prefix="+url.QueryEscape(prefix), &keys); err != nil {
		return nil, err
	}
	return keys, nil
}

func (c *RemoteClient) DeleteRaw(key string) error {
	return c.delete("/api/v1/raw/" + key)
}

func (c *RemoteClient) Close() error {
	// HTTP client doesn't need explicit closing
	return nil
}

// ============================================================================
// WorkflowEngine Implementation
// ============================================================================

func (c *RemoteClient) ExecuteWorkflow(workflow *model.Workflow, inputData []model.DataItem) (*engine.ExecutionResult, error) {
	request := struct {
		Workflow  *model.Workflow  `json:"workflow"`
		InputData []model.DataItem `json:"inputData"`
	}{
		Workflow:  workflow,
		InputData: inputData,
	}

	var response struct {
		Data  []model.DataItem `json:"data"`
		Error string           `json:"error,omitempty"`
	}

	if err := c.post("/api/v1/execute", request, &response); err != nil {
		return nil, err
	}

	result := &engine.ExecutionResult{
		Data: response.Data,
	}
	if response.Error != "" {
		result.Error = fmt.Errorf("%s", response.Error)
	}

	return result, nil
}

func (c *RemoteClient) ExecuteWorkflowParallel(workflows []*model.Workflow, inputData [][]model.DataItem) ([]*engine.ExecutionResult, error) {
	// Execute workflows sequentially via remote API
	results := make([]*engine.ExecutionResult, len(workflows))
	for i, workflow := range workflows {
		var input []model.DataItem
		if i < len(inputData) {
			input = inputData[i]
		}
		result, err := c.ExecuteWorkflow(workflow, input)
		if err != nil {
			results[i] = &engine.ExecutionResult{Error: err}
		} else {
			results[i] = result
		}
	}
	return results, nil
}

func (c *RemoteClient) RegisterNodeExecutor(nodeType string, executor base.NodeExecutor) {
	// Remote client cannot register local node executors
	// Nodes are registered on the remote server
}

func (c *RemoteClient) GetNodeExecutor(nodeType string) (base.NodeExecutor, error) {
	// Node executors are on the remote server
	// Return error to indicate this should be handled remotely
	return nil, fmt.Errorf("node executor %s not available locally (remote mode)", nodeType)
}

func (c *RemoteClient) SetCredentialManager(credentialManager *credentials.CredentialManager) {
	// Credentials are managed on the remote server
}

func (c *RemoteClient) SetConnectionRouter(connectionRouter connections.ConnectionRouter) {
	// Connections are managed on the remote server
}

// ============================================================================
// HTTP Helper Methods
// ============================================================================

func (c *RemoteClient) get(path string, result interface{}) error {
	resp, err := c.httpClient.Get(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, result)
}

func (c *RemoteClient) post(path string, body interface{}, result interface{}) error {
	return c.postWithContext(context.Background(), path, body, result)
}

func (c *RemoteClient) postWithContext(ctx context.Context, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, result)
}

func (c *RemoteClient) put(path string, body interface{}, result interface{}) error {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, c.baseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, result)
}

func (c *RemoteClient) delete(path string) error {
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("not found")
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *RemoteClient) handleResponse(resp *http.Response, result interface{}) error {
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("not found")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	if result != nil && len(body) > 0 {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}
