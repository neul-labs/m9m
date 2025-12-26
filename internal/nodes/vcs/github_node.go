package vcs

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dipankar/m9m/internal/model"
	"github.com/dipankar/m9m/internal/nodes/base"
)

// GitHubNode provides GitHub API integration
type GitHubNode struct {
	*base.BaseNode
	httpClient *http.Client
}

// NewGitHubNode creates a new GitHub node
func NewGitHubNode() *GitHubNode {
	return &GitHubNode{
		BaseNode: base.NewBaseNode(base.NodeDescription{
			Name:        "GitHub",
			Description: "Interact with GitHub API for repositories, issues, pull requests, and more",
			Category:    "vcs",
		}),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Execute runs the GitHub node
func (n *GitHubNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	resource, _ := nodeParams["resource"].(string)
	if resource == "" {
		resource = "repository"
	}

	operation, _ := nodeParams["operation"].(string)
	if operation == "" {
		operation = "get"
	}

	// Get authentication token
	token := n.getToken(nodeParams)
	if token == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}

	var results []model.DataItem

	for _, item := range inputData {
		var result map[string]interface{}
		var err error

		switch resource {
		case "repository":
			result, err = n.handleRepositoryOperation(operation, token, nodeParams, item)
		case "issue":
			result, err = n.handleIssueOperation(operation, token, nodeParams, item)
		case "pullRequest":
			result, err = n.handlePullRequestOperation(operation, token, nodeParams, item)
		case "user":
			result, err = n.handleUserOperation(operation, token, nodeParams, item)
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

func (n *GitHubNode) getToken(nodeParams map[string]interface{}) string {
	if creds, ok := nodeParams["credentials"].(map[string]interface{}); ok {
		if token, ok := creds["accessToken"].(string); ok {
			return token
		}
	}
	return ""
}

func (n *GitHubNode) handleRepositoryOperation(operation, token string, nodeParams map[string]interface{}, item model.DataItem) (map[string]interface{}, error) {
	owner, _ := nodeParams["owner"].(string)
	repo, _ := nodeParams["repository"].(string)

	if owner == "" || repo == "" {
		if o, ok := item.JSON["owner"].(string); ok {
			owner = o
		}
		if r, ok := item.JSON["repository"].(string); ok {
			repo = r
		}
	}

	switch operation {
	case "get":
		return n.makeAPIRequest("GET", fmt.Sprintf("/repos/%s/%s", owner, repo), token, nil)
	case "list":
		return n.makeAPIRequest("GET", fmt.Sprintf("/users/%s/repos", owner), token, nil)
	default:
		return nil, fmt.Errorf("unsupported repository operation: %s", operation)
	}
}

func (n *GitHubNode) handleIssueOperation(operation, token string, nodeParams map[string]interface{}, item model.DataItem) (map[string]interface{}, error) {
	owner, _ := nodeParams["owner"].(string)
	repo, _ := nodeParams["repository"].(string)

	switch operation {
	case "list":
		return n.makeAPIRequest("GET", fmt.Sprintf("/repos/%s/%s/issues", owner, repo), token, nil)
	case "get":
		issueNumber, _ := nodeParams["issueNumber"].(float64)
		return n.makeAPIRequest("GET", fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, int(issueNumber)), token, nil)
	default:
		return nil, fmt.Errorf("unsupported issue operation: %s", operation)
	}
}

func (n *GitHubNode) handlePullRequestOperation(operation, token string, nodeParams map[string]interface{}, item model.DataItem) (map[string]interface{}, error) {
	owner, _ := nodeParams["owner"].(string)
	repo, _ := nodeParams["repository"].(string)

	switch operation {
	case "list":
		return n.makeAPIRequest("GET", fmt.Sprintf("/repos/%s/%s/pulls", owner, repo), token, nil)
	case "get":
		prNumber, _ := nodeParams["pullNumber"].(float64)
		return n.makeAPIRequest("GET", fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, int(prNumber)), token, nil)
	default:
		return nil, fmt.Errorf("unsupported pull request operation: %s", operation)
	}
}

func (n *GitHubNode) handleUserOperation(operation, token string, nodeParams map[string]interface{}, item model.DataItem) (map[string]interface{}, error) {
	switch operation {
	case "getAuthenticated":
		return n.makeAPIRequest("GET", "/user", token, nil)
	case "get":
		username, _ := nodeParams["username"].(string)
		return n.makeAPIRequest("GET", fmt.Sprintf("/users/%s", username), token, nil)
	default:
		return nil, fmt.Errorf("unsupported user operation: %s", operation)
	}
}

func (n *GitHubNode) makeAPIRequest(method, path, token string, body interface{}) (map[string]interface{}, error) {
	url := "https://api.github.com" + path

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "n8n-go")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		return nil, fmt.Errorf("GitHub API error (%d): %v", resp.StatusCode, errorResp)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return map[string]interface{}{"status": "success"}, nil
	}

	return result, nil
}
