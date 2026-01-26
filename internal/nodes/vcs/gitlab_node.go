package vcs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// GitLabNode provides GitLab API integration
type GitLabNode struct {
	*base.BaseNode
	httpClient *http.Client
}

// NewGitLabNode creates a new GitLab node
func NewGitLabNode() *GitLabNode {
	return &GitLabNode{
		BaseNode: base.NewBaseNode(base.NodeDescription{
			Name:        "GitLab",
			Description: "Interact with GitLab API for repository management, CI/CD, and more",
			Category:    "vcs",
		}),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Execute runs the GitLab node
func (n *GitLabNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	resource, _ := nodeParams["resource"].(string)
	if resource == "" {
		resource = "issue"
	}

	operation, _ := nodeParams["operation"].(string)
	if operation == "" {
		operation = "get"
	}

	// Get authentication token
	token := n.getToken(nodeParams)
	if token == "" {
		return nil, fmt.Errorf("GitLab token is required")
	}

	var results []model.DataItem

	for _, item := range inputData {
		var result map[string]interface{}
		var err error

		switch resource {
		case "issue":
			result, err = n.handleIssueOperation(operation, token, nodeParams, item)
		case "mergeRequest":
			result, err = n.handleMergeRequestOperation(operation, token, nodeParams, item)
		case "project":
			result, err = n.handleProjectOperation(operation, token, nodeParams, item)
		case "pipeline":
			result, err = n.handlePipelineOperation(operation, token, nodeParams, item)
		case "user":
			result, err = n.handleUserOperation(operation, token, nodeParams, item)
		case "branch":
			result, err = n.handleBranchOperation(operation, token, nodeParams, item)
		default:
			return nil, fmt.Errorf("unsupported resource: %s", resource)
		}

		if err != nil {
			return nil, err
		}

		results = append(results, model.DataItem{JSON: result})
	}

	return results, nil
}

func (n *GitLabNode) getToken(nodeParams map[string]interface{}) string {
	if creds, ok := nodeParams["credentials"].(map[string]interface{}); ok {
		if token, ok := creds["accessToken"].(string); ok {
			return token
		}
		if token, ok := creds["apiToken"].(string); ok {
			return token
		}
	}
	return ""
}

func (n *GitLabNode) getBaseURL(nodeParams map[string]interface{}) string {
	if baseURL, ok := nodeParams["baseUrl"].(string); ok && baseURL != "" {
		return baseURL
	}
	return "https://gitlab.com/api/v4"
}

// Issue operations
func (n *GitLabNode) handleIssueOperation(operation, token string, nodeParams map[string]interface{}, item model.DataItem) (map[string]interface{}, error) {
	projectId, _ := nodeParams["projectId"].(string)
	if projectId == "" {
		if id, ok := item.JSON["projectId"].(string); ok {
			projectId = id
		}
	}

	switch operation {
	case "create":
		return n.createIssue(token, projectId, nodeParams)
	case "get":
		issueIid := n.getIntParam(nodeParams, "issueIid")
		return n.getIssue(token, projectId, issueIid, nodeParams)
	case "getAll":
		return n.getAllIssues(token, projectId, nodeParams)
	case "update":
		issueIid := n.getIntParam(nodeParams, "issueIid")
		return n.updateIssue(token, projectId, issueIid, nodeParams)
	case "delete":
		issueIid := n.getIntParam(nodeParams, "issueIid")
		return n.deleteIssue(token, projectId, issueIid, nodeParams)
	default:
		return nil, fmt.Errorf("unsupported issue operation: %s", operation)
	}
}

func (n *GitLabNode) createIssue(token, projectId string, nodeParams map[string]interface{}) (map[string]interface{}, error) {
	if projectId == "" {
		return nil, fmt.Errorf("projectId is required")
	}

	title, _ := nodeParams["title"].(string)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}

	body := map[string]interface{}{
		"title": title,
	}

	if description, ok := nodeParams["description"].(string); ok && description != "" {
		body["description"] = description
	}
	if labels, ok := nodeParams["labels"].(string); ok && labels != "" {
		body["labels"] = labels
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("%s/projects/%s/issues", n.getBaseURL(nodeParams), url.QueryEscape(projectId))
	return n.makeAPIRequest("POST", apiURL, token, jsonBody)
}

func (n *GitLabNode) getIssue(token, projectId string, issueIid int, nodeParams map[string]interface{}) (map[string]interface{}, error) {
	if projectId == "" || issueIid == 0 {
		return nil, fmt.Errorf("projectId and issueIid are required")
	}

	apiURL := fmt.Sprintf("%s/projects/%s/issues/%d", n.getBaseURL(nodeParams), url.QueryEscape(projectId), issueIid)
	return n.makeAPIRequest("GET", apiURL, token, nil)
}

func (n *GitLabNode) getAllIssues(token, projectId string, nodeParams map[string]interface{}) (map[string]interface{}, error) {
	if projectId == "" {
		return nil, fmt.Errorf("projectId is required")
	}

	queryParams := url.Values{}
	if state, ok := nodeParams["state"].(string); ok && state != "" {
		queryParams.Set("state", state)
	}
	if labels, ok := nodeParams["labels"].(string); ok && labels != "" {
		queryParams.Set("labels", labels)
	}

	apiURL := fmt.Sprintf("%s/projects/%s/issues?%s", n.getBaseURL(nodeParams), url.QueryEscape(projectId), queryParams.Encode())
	return n.makeAPIRequest("GET", apiURL, token, nil)
}

func (n *GitLabNode) updateIssue(token, projectId string, issueIid int, nodeParams map[string]interface{}) (map[string]interface{}, error) {
	if projectId == "" || issueIid == 0 {
		return nil, fmt.Errorf("projectId and issueIid are required")
	}

	body := map[string]interface{}{}
	if title, ok := nodeParams["title"].(string); ok && title != "" {
		body["title"] = title
	}
	if description, ok := nodeParams["description"].(string); ok && description != "" {
		body["description"] = description
	}
	if stateEvent, ok := nodeParams["state_event"].(string); ok && stateEvent != "" {
		body["state_event"] = stateEvent
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("%s/projects/%s/issues/%d", n.getBaseURL(nodeParams), url.QueryEscape(projectId), issueIid)
	return n.makeAPIRequest("PUT", apiURL, token, jsonBody)
}

func (n *GitLabNode) deleteIssue(token, projectId string, issueIid int, nodeParams map[string]interface{}) (map[string]interface{}, error) {
	if projectId == "" || issueIid == 0 {
		return nil, fmt.Errorf("projectId and issueIid are required")
	}

	apiURL := fmt.Sprintf("%s/projects/%s/issues/%d", n.getBaseURL(nodeParams), url.QueryEscape(projectId), issueIid)
	req, err := http.NewRequest("DELETE", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("PRIVATE-TOKEN", token)

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to delete issue: %s", body)
	}

	return map[string]interface{}{"success": true}, nil
}

// Merge Request operations
func (n *GitLabNode) handleMergeRequestOperation(operation, token string, nodeParams map[string]interface{}, item model.DataItem) (map[string]interface{}, error) {
	projectId, _ := nodeParams["projectId"].(string)
	if projectId == "" {
		if id, ok := item.JSON["projectId"].(string); ok {
			projectId = id
		}
	}

	switch operation {
	case "create":
		return n.createMergeRequest(token, projectId, nodeParams)
	case "get":
		mrIid := n.getIntParam(nodeParams, "mergeRequestIid")
		return n.getMergeRequest(token, projectId, mrIid, nodeParams)
	case "getAll":
		return n.getAllMergeRequests(token, projectId, nodeParams)
	case "merge":
		mrIid := n.getIntParam(nodeParams, "mergeRequestIid")
		return n.mergeMergeRequest(token, projectId, mrIid, nodeParams)
	default:
		return nil, fmt.Errorf("unsupported merge request operation: %s", operation)
	}
}

func (n *GitLabNode) createMergeRequest(token, projectId string, nodeParams map[string]interface{}) (map[string]interface{}, error) {
	if projectId == "" {
		return nil, fmt.Errorf("projectId is required")
	}

	sourceBranch, _ := nodeParams["source_branch"].(string)
	targetBranch, _ := nodeParams["target_branch"].(string)
	title, _ := nodeParams["title"].(string)

	if sourceBranch == "" || targetBranch == "" || title == "" {
		return nil, fmt.Errorf("source_branch, target_branch, and title are required")
	}

	body := map[string]interface{}{
		"source_branch": sourceBranch,
		"target_branch": targetBranch,
		"title":         title,
	}

	if description, ok := nodeParams["description"].(string); ok && description != "" {
		body["description"] = description
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("%s/projects/%s/merge_requests", n.getBaseURL(nodeParams), url.QueryEscape(projectId))
	return n.makeAPIRequest("POST", apiURL, token, jsonBody)
}

func (n *GitLabNode) getMergeRequest(token, projectId string, mrIid int, nodeParams map[string]interface{}) (map[string]interface{}, error) {
	if projectId == "" || mrIid == 0 {
		return nil, fmt.Errorf("projectId and mergeRequestIid are required")
	}

	apiURL := fmt.Sprintf("%s/projects/%s/merge_requests/%d", n.getBaseURL(nodeParams), url.QueryEscape(projectId), mrIid)
	return n.makeAPIRequest("GET", apiURL, token, nil)
}

func (n *GitLabNode) getAllMergeRequests(token, projectId string, nodeParams map[string]interface{}) (map[string]interface{}, error) {
	if projectId == "" {
		return nil, fmt.Errorf("projectId is required")
	}

	queryParams := url.Values{}
	if state, ok := nodeParams["state"].(string); ok && state != "" {
		queryParams.Set("state", state)
	}

	apiURL := fmt.Sprintf("%s/projects/%s/merge_requests?%s", n.getBaseURL(nodeParams), url.QueryEscape(projectId), queryParams.Encode())
	return n.makeAPIRequest("GET", apiURL, token, nil)
}

func (n *GitLabNode) mergeMergeRequest(token, projectId string, mrIid int, nodeParams map[string]interface{}) (map[string]interface{}, error) {
	if projectId == "" || mrIid == 0 {
		return nil, fmt.Errorf("projectId and mergeRequestIid are required")
	}

	body := map[string]interface{}{}
	if mergeCommitMessage, ok := nodeParams["merge_commit_message"].(string); ok && mergeCommitMessage != "" {
		body["merge_commit_message"] = mergeCommitMessage
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("%s/projects/%s/merge_requests/%d/merge", n.getBaseURL(nodeParams), url.QueryEscape(projectId), mrIid)
	return n.makeAPIRequest("PUT", apiURL, token, jsonBody)
}

// Pipeline operations
func (n *GitLabNode) handlePipelineOperation(operation, token string, nodeParams map[string]interface{}, item model.DataItem) (map[string]interface{}, error) {
	projectId, _ := nodeParams["projectId"].(string)
	if projectId == "" {
		if id, ok := item.JSON["projectId"].(string); ok {
			projectId = id
		}
	}

	switch operation {
	case "trigger":
		return n.triggerPipeline(token, projectId, nodeParams)
	case "get":
		pipelineId := n.getIntParam(nodeParams, "pipelineId")
		return n.getPipeline(token, projectId, pipelineId, nodeParams)
	case "getAll":
		return n.getAllPipelines(token, projectId, nodeParams)
	case "cancel":
		pipelineId := n.getIntParam(nodeParams, "pipelineId")
		return n.cancelPipeline(token, projectId, pipelineId, nodeParams)
	case "retry":
		pipelineId := n.getIntParam(nodeParams, "pipelineId")
		return n.retryPipeline(token, projectId, pipelineId, nodeParams)
	default:
		return nil, fmt.Errorf("unsupported pipeline operation: %s", operation)
	}
}

func (n *GitLabNode) triggerPipeline(token, projectId string, nodeParams map[string]interface{}) (map[string]interface{}, error) {
	if projectId == "" {
		return nil, fmt.Errorf("projectId is required")
	}

	ref, _ := nodeParams["ref"].(string)
	if ref == "" {
		return nil, fmt.Errorf("ref is required")
	}

	body := map[string]interface{}{
		"ref": ref,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("%s/projects/%s/pipeline", n.getBaseURL(nodeParams), url.QueryEscape(projectId))
	return n.makeAPIRequest("POST", apiURL, token, jsonBody)
}

func (n *GitLabNode) getPipeline(token, projectId string, pipelineId int, nodeParams map[string]interface{}) (map[string]interface{}, error) {
	if projectId == "" || pipelineId == 0 {
		return nil, fmt.Errorf("projectId and pipelineId are required")
	}

	apiURL := fmt.Sprintf("%s/projects/%s/pipelines/%d", n.getBaseURL(nodeParams), url.QueryEscape(projectId), pipelineId)
	return n.makeAPIRequest("GET", apiURL, token, nil)
}

func (n *GitLabNode) getAllPipelines(token, projectId string, nodeParams map[string]interface{}) (map[string]interface{}, error) {
	if projectId == "" {
		return nil, fmt.Errorf("projectId is required")
	}

	queryParams := url.Values{}
	if status, ok := nodeParams["status"].(string); ok && status != "" {
		queryParams.Set("status", status)
	}
	if ref, ok := nodeParams["ref"].(string); ok && ref != "" {
		queryParams.Set("ref", ref)
	}

	apiURL := fmt.Sprintf("%s/projects/%s/pipelines?%s", n.getBaseURL(nodeParams), url.QueryEscape(projectId), queryParams.Encode())
	return n.makeAPIRequest("GET", apiURL, token, nil)
}

func (n *GitLabNode) cancelPipeline(token, projectId string, pipelineId int, nodeParams map[string]interface{}) (map[string]interface{}, error) {
	if projectId == "" || pipelineId == 0 {
		return nil, fmt.Errorf("projectId and pipelineId are required")
	}

	apiURL := fmt.Sprintf("%s/projects/%s/pipelines/%d/cancel", n.getBaseURL(nodeParams), url.QueryEscape(projectId), pipelineId)
	return n.makeAPIRequest("POST", apiURL, token, nil)
}

func (n *GitLabNode) retryPipeline(token, projectId string, pipelineId int, nodeParams map[string]interface{}) (map[string]interface{}, error) {
	if projectId == "" || pipelineId == 0 {
		return nil, fmt.Errorf("projectId and pipelineId are required")
	}

	apiURL := fmt.Sprintf("%s/projects/%s/pipelines/%d/retry", n.getBaseURL(nodeParams), url.QueryEscape(projectId), pipelineId)
	return n.makeAPIRequest("POST", apiURL, token, nil)
}

// Project operations
func (n *GitLabNode) handleProjectOperation(operation, token string, nodeParams map[string]interface{}, item model.DataItem) (map[string]interface{}, error) {
	switch operation {
	case "get":
		projectId, _ := nodeParams["projectId"].(string)
		if projectId == "" {
			if id, ok := item.JSON["projectId"].(string); ok {
				projectId = id
			}
		}
		if projectId == "" {
			return nil, fmt.Errorf("projectId is required")
		}
		apiURL := fmt.Sprintf("%s/projects/%s", n.getBaseURL(nodeParams), url.QueryEscape(projectId))
		return n.makeAPIRequest("GET", apiURL, token, nil)
	case "list":
		apiURL := fmt.Sprintf("%s/projects", n.getBaseURL(nodeParams))
		return n.makeAPIRequest("GET", apiURL, token, nil)
	default:
		return nil, fmt.Errorf("unsupported project operation: %s", operation)
	}
}

// User operations
func (n *GitLabNode) handleUserOperation(operation, token string, nodeParams map[string]interface{}, item model.DataItem) (map[string]interface{}, error) {
	switch operation {
	case "getAuthenticated":
		apiURL := fmt.Sprintf("%s/user", n.getBaseURL(nodeParams))
		return n.makeAPIRequest("GET", apiURL, token, nil)
	case "get":
		userId := n.getIntParam(nodeParams, "userId")
		if userId == 0 {
			return nil, fmt.Errorf("userId is required")
		}
		apiURL := fmt.Sprintf("%s/users/%d", n.getBaseURL(nodeParams), userId)
		return n.makeAPIRequest("GET", apiURL, token, nil)
	default:
		return nil, fmt.Errorf("unsupported user operation: %s", operation)
	}
}

// Branch operations
func (n *GitLabNode) handleBranchOperation(operation, token string, nodeParams map[string]interface{}, item model.DataItem) (map[string]interface{}, error) {
	projectId, _ := nodeParams["projectId"].(string)
	if projectId == "" {
		if id, ok := item.JSON["projectId"].(string); ok {
			projectId = id
		}
	}

	switch operation {
	case "list":
		if projectId == "" {
			return nil, fmt.Errorf("projectId is required")
		}
		apiURL := fmt.Sprintf("%s/projects/%s/repository/branches", n.getBaseURL(nodeParams), url.QueryEscape(projectId))
		return n.makeAPIRequest("GET", apiURL, token, nil)
	case "get":
		branchName, _ := nodeParams["branch"].(string)
		if projectId == "" || branchName == "" {
			return nil, fmt.Errorf("projectId and branch are required")
		}
		apiURL := fmt.Sprintf("%s/projects/%s/repository/branches/%s", n.getBaseURL(nodeParams), url.QueryEscape(projectId), url.QueryEscape(branchName))
		return n.makeAPIRequest("GET", apiURL, token, nil)
	default:
		return nil, fmt.Errorf("unsupported branch operation: %s", operation)
	}
}

func (n *GitLabNode) makeAPIRequest(method, apiURL, token string, body []byte) (map[string]interface{}, error) {
	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, apiURL, bytes.NewReader(body))
	} else {
		req, err = http.NewRequest(method, apiURL, nil)
	}
	if err != nil {
		return nil, err
	}

	req.Header.Set("PRIVATE-TOKEN", token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitLab API error (%d): %s", resp.StatusCode, respBody)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// Response might be an array, wrap it
		resp.Body.Close()
		return map[string]interface{}{"status": "success"}, nil
	}

	return result, nil
}

func (n *GitLabNode) getIntParam(nodeParams map[string]interface{}, key string) int {
	if v, ok := nodeParams[key].(float64); ok {
		return int(v)
	}
	if v, ok := nodeParams[key].(int); ok {
		return v
	}
	if v, ok := nodeParams[key].(string); ok {
		i, _ := strconv.Atoi(v)
		return i
	}
	return 0
}
