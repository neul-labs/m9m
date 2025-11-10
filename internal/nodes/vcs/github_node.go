package vcs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dipankar/n8n-go/internal/nodes/base"
)

// GitHubNode provides GitHub API operations
type GitHubNode struct {
	*base.BaseNode
	httpClient *http.Client
}

// NewGitHubNode creates a new GitHub node
func NewGitHubNode() *GitHubNode {
	return &GitHubNode{
		BaseNode: base.NewBaseNode(base.NodeDescription{Name: "GitHub", Description: "GitHub API", Category: "core"}),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetMetadata returns the node metadata
func (n *GitHubNode) GetMetadata() base.NodeMetadata {
	return base.NodeMetadata{
		Name:        "GitHub",
		DisplayName: "GitHub",
		Description: "Interact with GitHub repositories, issues, pull requests, and more",
		Group:       []string{"Version Control"},
		Version:     1,
		Inputs:      []string{"main"},
		Outputs:     []string{"main"},
		Credentials: []base.CredentialType{
			{
				Name:        "githubApi",
				Required:    true,
				DisplayName: "GitHub API",
			},
		},
		Properties: []base.NodeProperty{
			{
				Name:        "resource",
				DisplayName: "Resource",
				Type:        "options",
				Options: []base.OptionItem{
					{Name: "Repository", Value: "repository"},
					{Name: "Issue", Value: "issue"},
					{Name: "Pull Request", Value: "pullRequest"},
					{Name: "Release", Value: "release"},
					{Name: "User", Value: "user"},
					{Name: "Organization", Value: "organization"},
					{Name: "File", Value: "file"},
					{Name: "Commit", Value: "commit"},
					{Name: "Branch", Value: "branch"},
					{Name: "Tag", Value: "tag"},
				},
				Default:     "repository",
				Required:    true,
				Description: "The GitHub resource to operate on",
			},
			{
				Name:        "operation",
				DisplayName: "Operation",
				Type:        "options",
				Default:     "get",
				Required:    true,
				Description: "The operation to perform",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"resource": []string{"repository"},
					},
				},
				Options: []base.OptionItem{
					{Name: "Get", Value: "get"},
					{Name: "Get Many", Value: "getMany"},
					{Name: "Create", Value: "create"},
					{Name: "Update", Value: "update"},
					{Name: "Delete", Value: "delete"},
				},
			},
			{
				Name:        "owner",
				DisplayName: "Repository Owner",
				Type:        "string",
				Default:     "",
				Description: "The owner of the repository (username or organization)",
				Required:    true,
			},
			{
				Name:        "repository",
				DisplayName: "Repository Name",
				Type:        "string",
				Default:     "",
				Description: "The name of the repository",
				Required:    true,
			},
			{
				Name:        "issueNumber",
				DisplayName: "Issue Number",
				Type:        "number",
				Default:     0,
				Description: "The issue number",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"resource": []string{"issue"},
					},
				},
			},
			{
				Name:        "pullRequestNumber",
				DisplayName: "Pull Request Number",
				Type:        "number",
				Default:     0,
				Description: "The pull request number",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"resource": []string{"pullRequest"},
					},
				},
			},
			{
				Name:        "title",
				DisplayName: "Title",
				Type:        "string",
				Default:     "",
				Description: "The title",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"resource":  []string{"issue", "pullRequest"},
						"operation": []string{"create", "update"},
					},
				},
			},
			{
				Name:        "body",
				DisplayName: "Body",
				Type:        "string",
				TypeOptions: map[string]interface{}{
					"rows": 5,
				},
				Default:     "",
				Description: "The body/description",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"resource":  []string{"issue", "pullRequest"},
						"operation": []string{"create", "update"},
					},
				},
			},
			{
				Name:        "state",
				DisplayName: "State",
				Type:        "options",
				Options: []base.OptionItem{
					{Name: "Open", Value: "open"},
					{Name: "Closed", Value: "closed"},
					{Name: "All", Value: "all"},
				},
				Default:     "open",
				Description: "The state filter",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"resource":  []string{"issue", "pullRequest"},
						"operation": []string{"getMany"},
					},
				},
			},
			{
				Name:        "labels",
				DisplayName: "Labels",
				Type:        "string",
				Default:     "",
				Description: "Comma-separated list of labels",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"resource":  []string{"issue"},
						"operation": []string{"create", "update"},
					},
				},
			},
			{
				Name:        "assignees",
				DisplayName: "Assignees",
				Type:        "string",
				Default:     "",
				Description: "Comma-separated list of assignee usernames",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"resource":  []string{"issue"},
						"operation": []string{"create", "update"},
					},
				},
			},
			{
				Name:        "path",
				DisplayName: "File Path",
				Type:        "string",
				Default:     "",
				Description: "Path to the file in the repository",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"resource": []string{"file"},
					},
				},
			},
			{
				Name:        "content",
				DisplayName: "File Content",
				Type:        "string",
				TypeOptions: map[string]interface{}{
					"rows": 10,
				},
				Default:     "",
				Description: "The content of the file",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"resource":  []string{"file"},
						"operation": []string{"create", "update"},
					},
				},
			},
			{
				Name:        "commitMessage",
				DisplayName: "Commit Message",
				Type:        "string",
				Default:     "",
				Description: "The commit message",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"resource":  []string{"file"},
						"operation": []string{"create", "update", "delete"},
					},
				},
			},
			{
				Name:        "branch",
				DisplayName: "Branch",
				Type:        "string",
				Default:     "main",
				Description: "The branch name",
			},
			{
				Name:        "baseBranch",
				DisplayName: "Base Branch",
				Type:        "string",
				Default:     "main",
				Description: "The base branch for pull requests",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"resource":  []string{"pullRequest"},
						"operation": []string{"create"},
					},
				},
			},
			{
				Name:        "headBranch",
				DisplayName: "Head Branch",
				Type:        "string",
				Default:     "",
				Description: "The head branch for pull requests",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"resource":  []string{"pullRequest"},
						"operation": []string{"create"},
					},
				},
			},
			{
				Name:        "limit",
				DisplayName: "Limit",
				Type:        "number",
				Default:     30,
				Description: "Maximum number of results to return",
			},
		},
	}
}

// Execute runs the GitHub operation
func (n *GitHubNode) Execute(ctx context.Context, params base.ExecutionParams) (base.NodeOutput, error) {
	// Get credentials
	credentials, err := params.GetCredentials("githubApi")
	if err != nil {
		return base.NodeOutput{}, fmt.Errorf("failed to get GitHub credentials: %w", err)
	}

	token, ok := credentials["accessToken"].(string)
	if !ok || token == "" {
		return base.NodeOutput{}, fmt.Errorf("GitHub access token not found")
	}

	// Get resource
	resource := params.GetNodeParameter("resource", "repository").(string)

	var result interface{}
	switch resource {
	case "repository":
		result, err = n.handleRepositoryResource(token, params)
	case "issue":
		result, err = n.handleIssueResource(token, params)
	case "pullRequest":
		result, err = n.handlePullRequestResource(token, params)
	case "release":
		result, err = n.handleReleaseResource(token, params)
	case "user":
		result, err = n.handleUserResource(token, params)
	case "organization":
		result, err = n.handleOrganizationResource(token, params)
	case "file":
		result, err = n.handleFileResource(token, params)
	case "commit":
		result, err = n.handleCommitResource(token, params)
	case "branch":
		result, err = n.handleBranchResource(token, params)
	case "tag":
		result, err = n.handleTagResource(token, params)
	default:
		err = fmt.Errorf("unsupported resource: %s", resource)
	}

	if err != nil {
		return base.NodeOutput{}, err
	}

	// Format output
	var outputItems []base.ItemData
	switch v := result.(type) {
	case []interface{}:
		for i, item := range v {
			if itemMap, ok := item.(map[string]interface{}); ok {
				outputItems = append(outputItems, base.ItemData{
					JSON:  itemMap,
					Index: i,
				})
			}
		}
	case map[string]interface{}:
		outputItems = append(outputItems, base.ItemData{
			JSON:  v,
			Index: 0,
		})
	default:
		outputItems = append(outputItems, base.ItemData{
			JSON: map[string]interface{}{
				"result": result,
			},
			Index: 0,
		})
	}

	return base.NodeOutput{
		Items: outputItems,
	}, nil
}

// Repository resource handlers

func (n *GitHubNode) handleRepositoryResource(token string, params base.ExecutionParams) (interface{}, error) {
	operation := params.GetNodeParameter("operation", "get").(string)
	owner := params.GetNodeParameter("owner", "").(string)
	repo := params.GetNodeParameter("repository", "").(string)

	if owner == "" || repo == "" {
		return nil, fmt.Errorf("owner and repository are required")
	}

	switch operation {
	case "get":
		return n.makeGitHubRequest("GET", fmt.Sprintf("/repos/%s/%s", owner, repo), token, nil)
	case "getMany":
		return n.makeGitHubRequest("GET", fmt.Sprintf("/users/%s/repos", owner), token, nil)
	case "create":
		body := map[string]interface{}{
			"name":        repo,
			"description": params.GetNodeParameter("description", "").(string),
			"private":     params.GetNodeParameter("private", false).(bool),
		}
		return n.makeGitHubRequest("POST", "/user/repos", token, body)
	case "update":
		body := map[string]interface{}{
			"description": params.GetNodeParameter("description", "").(string),
			"private":     params.GetNodeParameter("private", false).(bool),
		}
		return n.makeGitHubRequest("PATCH", fmt.Sprintf("/repos/%s/%s", owner, repo), token, body)
	case "delete":
		return n.makeGitHubRequest("DELETE", fmt.Sprintf("/repos/%s/%s", owner, repo), token, nil)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// Issue resource handlers

func (n *GitHubNode) handleIssueResource(token string, params base.ExecutionParams) (interface{}, error) {
	operation := params.GetNodeParameter("operation", "get").(string)
	owner := params.GetNodeParameter("owner", "").(string)
	repo := params.GetNodeParameter("repository", "").(string)

	if owner == "" || repo == "" {
		return nil, fmt.Errorf("owner and repository are required")
	}

	switch operation {
	case "get":
		issueNumber := int(params.GetNodeParameter("issueNumber", 0).(float64))
		if issueNumber == 0 {
			return nil, fmt.Errorf("issue number is required")
		}
		return n.makeGitHubRequest("GET", fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, issueNumber), token, nil)
	
	case "getMany":
		state := params.GetNodeParameter("state", "open").(string)
		limit := int(params.GetNodeParameter("limit", 30).(float64))
		return n.makeGitHubRequest("GET", fmt.Sprintf("/repos/%s/%s/issues?state=%s&per_page=%d", owner, repo, state, limit), token, nil)
	
	case "create":
		body := map[string]interface{}{
			"title": params.GetNodeParameter("title", "").(string),
			"body":  params.GetNodeParameter("body", "").(string),
		}
		
		// Add labels
		labels := params.GetNodeParameter("labels", "").(string)
		if labels != "" {
			body["labels"] = strings.Split(labels, ",")
		}
		
		// Add assignees
		assignees := params.GetNodeParameter("assignees", "").(string)
		if assignees != "" {
			body["assignees"] = strings.Split(assignees, ",")
		}
		
		return n.makeGitHubRequest("POST", fmt.Sprintf("/repos/%s/%s/issues", owner, repo), token, body)
	
	case "update":
		issueNumber := int(params.GetNodeParameter("issueNumber", 0).(float64))
		if issueNumber == 0 {
			return nil, fmt.Errorf("issue number is required")
		}
		
		body := map[string]interface{}{}
		if title := params.GetNodeParameter("title", "").(string); title != "" {
			body["title"] = title
		}
		if bodyText := params.GetNodeParameter("body", "").(string); bodyText != "" {
			body["body"] = bodyText
		}
		if state := params.GetNodeParameter("state", "").(string); state != "" {
			body["state"] = state
		}
		
		return n.makeGitHubRequest("PATCH", fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, issueNumber), token, body)
	
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// Pull Request resource handlers

func (n *GitHubNode) handlePullRequestResource(token string, params base.ExecutionParams) (interface{}, error) {
	operation := params.GetNodeParameter("operation", "get").(string)
	owner := params.GetNodeParameter("owner", "").(string)
	repo := params.GetNodeParameter("repository", "").(string)

	if owner == "" || repo == "" {
		return nil, fmt.Errorf("owner and repository are required")
	}

	switch operation {
	case "get":
		prNumber := int(params.GetNodeParameter("pullRequestNumber", 0).(float64))
		if prNumber == 0 {
			return nil, fmt.Errorf("pull request number is required")
		}
		return n.makeGitHubRequest("GET", fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, prNumber), token, nil)
	
	case "getMany":
		state := params.GetNodeParameter("state", "open").(string)
		limit := int(params.GetNodeParameter("limit", 30).(float64))
		return n.makeGitHubRequest("GET", fmt.Sprintf("/repos/%s/%s/pulls?state=%s&per_page=%d", owner, repo, state, limit), token, nil)
	
	case "create":
		body := map[string]interface{}{
			"title": params.GetNodeParameter("title", "").(string),
			"body":  params.GetNodeParameter("body", "").(string),
			"head":  params.GetNodeParameter("headBranch", "").(string),
			"base":  params.GetNodeParameter("baseBranch", "main").(string),
		}
		return n.makeGitHubRequest("POST", fmt.Sprintf("/repos/%s/%s/pulls", owner, repo), token, body)
	
	case "update":
		prNumber := int(params.GetNodeParameter("pullRequestNumber", 0).(float64))
		if prNumber == 0 {
			return nil, fmt.Errorf("pull request number is required")
		}
		
		body := map[string]interface{}{}
		if title := params.GetNodeParameter("title", "").(string); title != "" {
			body["title"] = title
		}
		if bodyText := params.GetNodeParameter("body", "").(string); bodyText != "" {
			body["body"] = bodyText
		}
		if state := params.GetNodeParameter("state", "").(string); state != "" {
			body["state"] = state
		}
		
		return n.makeGitHubRequest("PATCH", fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, prNumber), token, body)
	
	case "merge":
		prNumber := int(params.GetNodeParameter("pullRequestNumber", 0).(float64))
		if prNumber == 0 {
			return nil, fmt.Errorf("pull request number is required")
		}
		
		body := map[string]interface{}{
			"commit_title":   params.GetNodeParameter("commitTitle", "").(string),
			"commit_message": params.GetNodeParameter("commitMessage", "").(string),
			"merge_method":   params.GetNodeParameter("mergeMethod", "merge").(string),
		}
		return n.makeGitHubRequest("PUT", fmt.Sprintf("/repos/%s/%s/pulls/%d/merge", owner, repo, prNumber), token, body)
	
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// Release resource handlers

func (n *GitHubNode) handleReleaseResource(token string, params base.ExecutionParams) (interface{}, error) {
	operation := params.GetNodeParameter("operation", "get").(string)
	owner := params.GetNodeParameter("owner", "").(string)
	repo := params.GetNodeParameter("repository", "").(string)

	if owner == "" || repo == "" {
		return nil, fmt.Errorf("owner and repository are required")
	}

	switch operation {
	case "get":
		releaseID := params.GetNodeParameter("releaseId", "").(string)
		if releaseID == "" {
			return n.makeGitHubRequest("GET", fmt.Sprintf("/repos/%s/%s/releases/latest", owner, repo), token, nil)
		}
		return n.makeGitHubRequest("GET", fmt.Sprintf("/repos/%s/%s/releases/%s", owner, repo, releaseID), token, nil)
	
	case "getMany":
		limit := int(params.GetNodeParameter("limit", 30).(float64))
		return n.makeGitHubRequest("GET", fmt.Sprintf("/repos/%s/%s/releases?per_page=%d", owner, repo, limit), token, nil)
	
	case "create":
		body := map[string]interface{}{
			"tag_name":         params.GetNodeParameter("tagName", "").(string),
			"name":             params.GetNodeParameter("name", "").(string),
			"body":             params.GetNodeParameter("body", "").(string),
			"draft":            params.GetNodeParameter("draft", false).(bool),
			"prerelease":       params.GetNodeParameter("prerelease", false).(bool),
			"generate_release_notes": params.GetNodeParameter("generateReleaseNotes", false).(bool),
		}
		return n.makeGitHubRequest("POST", fmt.Sprintf("/repos/%s/%s/releases", owner, repo), token, body)
	
	case "update":
		releaseID := params.GetNodeParameter("releaseId", "").(string)
		if releaseID == "" {
			return nil, fmt.Errorf("release ID is required")
		}
		
		body := map[string]interface{}{}
		if name := params.GetNodeParameter("name", "").(string); name != "" {
			body["name"] = name
		}
		if bodyText := params.GetNodeParameter("body", "").(string); bodyText != "" {
			body["body"] = bodyText
		}
		
		return n.makeGitHubRequest("PATCH", fmt.Sprintf("/repos/%s/%s/releases/%s", owner, repo, releaseID), token, body)
	
	case "delete":
		releaseID := params.GetNodeParameter("releaseId", "").(string)
		if releaseID == "" {
			return nil, fmt.Errorf("release ID is required")
		}
		return n.makeGitHubRequest("DELETE", fmt.Sprintf("/repos/%s/%s/releases/%s", owner, repo, releaseID), token, nil)
	
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// User resource handlers

func (n *GitHubNode) handleUserResource(token string, params base.ExecutionParams) (interface{}, error) {
	operation := params.GetNodeParameter("operation", "get").(string)

	switch operation {
	case "get":
		username := params.GetNodeParameter("username", "").(string)
		if username == "" {
			return n.makeGitHubRequest("GET", "/user", token, nil)
		}
		return n.makeGitHubRequest("GET", fmt.Sprintf("/users/%s", username), token, nil)
	
	case "getRepos":
		username := params.GetNodeParameter("username", "").(string)
		if username == "" {
			return n.makeGitHubRequest("GET", "/user/repos", token, nil)
		}
		return n.makeGitHubRequest("GET", fmt.Sprintf("/users/%s/repos", username), token, nil)
	
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// Organization resource handlers

func (n *GitHubNode) handleOrganizationResource(token string, params base.ExecutionParams) (interface{}, error) {
	operation := params.GetNodeParameter("operation", "get").(string)
	org := params.GetNodeParameter("organization", "").(string)

	if org == "" {
		return nil, fmt.Errorf("organization is required")
	}

	switch operation {
	case "get":
		return n.makeGitHubRequest("GET", fmt.Sprintf("/orgs/%s", org), token, nil)
	
	case "getRepos":
		limit := int(params.GetNodeParameter("limit", 30).(float64))
		return n.makeGitHubRequest("GET", fmt.Sprintf("/orgs/%s/repos?per_page=%d", org, limit), token, nil)
	
	case "getMembers":
		limit := int(params.GetNodeParameter("limit", 30).(float64))
		return n.makeGitHubRequest("GET", fmt.Sprintf("/orgs/%s/members?per_page=%d", org, limit), token, nil)
	
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// File resource handlers

func (n *GitHubNode) handleFileResource(token string, params base.ExecutionParams) (interface{}, error) {
	operation := params.GetNodeParameter("operation", "get").(string)
	owner := params.GetNodeParameter("owner", "").(string)
	repo := params.GetNodeParameter("repository", "").(string)
	path := params.GetNodeParameter("path", "").(string)

	if owner == "" || repo == "" || path == "" {
		return nil, fmt.Errorf("owner, repository, and path are required")
	}

	switch operation {
	case "get":
		return n.makeGitHubRequest("GET", fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, path), token, nil)
	
	case "create", "update":
		content := params.GetNodeParameter("content", "").(string)
		commitMessage := params.GetNodeParameter("commitMessage", "Update file").(string)
		branch := params.GetNodeParameter("branch", "main").(string)
		
		// Base64 encode content
		encodedContent := base64.StdEncoding.EncodeToString([]byte(content))
		
		body := map[string]interface{}{
			"message": commitMessage,
			"content": encodedContent,
			"branch":  branch,
		}
		
		// For updates, need SHA
		if operation == "update" {
			// Get current file to get SHA
			currentFile, err := n.makeGitHubRequest("GET", fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, path), token, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to get current file: %w", err)
			}
			if sha, ok := currentFile["sha"].(string); ok {
				body["sha"] = sha
			}
		}
		
		return n.makeGitHubRequest("PUT", fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, path), token, body)
	
	case "delete":
		commitMessage := params.GetNodeParameter("commitMessage", "Delete file").(string)
		branch := params.GetNodeParameter("branch", "main").(string)
		
		// Get current file to get SHA
		currentFile, err := n.makeGitHubRequest("GET", fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, path), token, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get current file: %w", err)
		}
		
		body := map[string]interface{}{
			"message": commitMessage,
			"branch":  branch,
		}
		
		if sha, ok := currentFile["sha"].(string); ok {
			body["sha"] = sha
		}
		
		return n.makeGitHubRequest("DELETE", fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, path), token, body)
	
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// Commit resource handlers

func (n *GitHubNode) handleCommitResource(token string, params base.ExecutionParams) (interface{}, error) {
	operation := params.GetNodeParameter("operation", "get").(string)
	owner := params.GetNodeParameter("owner", "").(string)
	repo := params.GetNodeParameter("repository", "").(string)

	if owner == "" || repo == "" {
		return nil, fmt.Errorf("owner and repository are required")
	}

	switch operation {
	case "get":
		sha := params.GetNodeParameter("sha", "").(string)
		if sha == "" {
			return nil, fmt.Errorf("commit SHA is required")
		}
		return n.makeGitHubRequest("GET", fmt.Sprintf("/repos/%s/%s/commits/%s", owner, repo, sha), token, nil)
	
	case "getMany":
		branch := params.GetNodeParameter("branch", "main").(string)
		limit := int(params.GetNodeParameter("limit", 30).(float64))
		return n.makeGitHubRequest("GET", fmt.Sprintf("/repos/%s/%s/commits?sha=%s&per_page=%d", owner, repo, branch, limit), token, nil)
	
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// Branch resource handlers

func (n *GitHubNode) handleBranchResource(token string, params base.ExecutionParams) (interface{}, error) {
	operation := params.GetNodeParameter("operation", "get").(string)
	owner := params.GetNodeParameter("owner", "").(string)
	repo := params.GetNodeParameter("repository", "").(string)

	if owner == "" || repo == "" {
		return nil, fmt.Errorf("owner and repository are required")
	}

	switch operation {
	case "get":
		branch := params.GetNodeParameter("branch", "main").(string)
		return n.makeGitHubRequest("GET", fmt.Sprintf("/repos/%s/%s/branches/%s", owner, repo, branch), token, nil)
	
	case "getMany":
		limit := int(params.GetNodeParameter("limit", 30).(float64))
		return n.makeGitHubRequest("GET", fmt.Sprintf("/repos/%s/%s/branches?per_page=%d", owner, repo, limit), token, nil)
	
	case "create":
		newBranch := params.GetNodeParameter("newBranch", "").(string)
		baseBranch := params.GetNodeParameter("baseBranch", "main").(string)
		
		if newBranch == "" {
			return nil, fmt.Errorf("new branch name is required")
		}
		
		// Get base branch SHA
		baseRef, err := n.makeGitHubRequest("GET", fmt.Sprintf("/repos/%s/%s/git/ref/heads/%s", owner, repo, baseBranch), token, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get base branch: %w", err)
		}
		
		var sha string
		if obj, ok := baseRef["object"].(map[string]interface{}); ok {
			sha = obj["sha"].(string)
		}
		
		body := map[string]interface{}{
			"ref": fmt.Sprintf("refs/heads/%s", newBranch),
			"sha": sha,
		}
		
		return n.makeGitHubRequest("POST", fmt.Sprintf("/repos/%s/%s/git/refs", owner, repo), token, body)
	
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// Tag resource handlers

func (n *GitHubNode) handleTagResource(token string, params base.ExecutionParams) (interface{}, error) {
	operation := params.GetNodeParameter("operation", "get").(string)
	owner := params.GetNodeParameter("owner", "").(string)
	repo := params.GetNodeParameter("repository", "").(string)

	if owner == "" || repo == "" {
		return nil, fmt.Errorf("owner and repository are required")
	}

	switch operation {
	case "get":
		tag := params.GetNodeParameter("tag", "").(string)
		if tag == "" {
			return nil, fmt.Errorf("tag is required")
		}
		return n.makeGitHubRequest("GET", fmt.Sprintf("/repos/%s/%s/git/tags/%s", owner, repo, tag), token, nil)
	
	case "getMany":
		limit := int(params.GetNodeParameter("limit", 30).(float64))
		return n.makeGitHubRequest("GET", fmt.Sprintf("/repos/%s/%s/tags?per_page=%d", owner, repo, limit), token, nil)
	
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// Helper method for making GitHub API requests

func (n *GitHubNode) makeGitHubRequest(method, endpoint, token string, body interface{}) (map[string]interface{}, error) {
	url := "https://api.github.com" + endpoint

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Authorization", "token "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errorResponse map[string]interface{}
		if err := json.Unmarshal(respBody, &errorResponse); err == nil {
			if msg, ok := errorResponse["message"].(string); ok {
				return nil, fmt.Errorf("GitHub API error: %s", msg)
			}
		}
		return nil, fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Handle array responses
	var result interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// If result is an array, wrap it
	if arr, ok := result.([]interface{}); ok {
		return map[string]interface{}{"data": arr}, nil
	}

	// If result is already a map, return it
	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}

	return map[string]interface{}{"data": result}, nil
}

// Clone creates a copy of the node
func (n *GitHubNode) Clone() base.Node {
	return &GitHubNode{
		BaseNode:   n.BaseNode.Clone(),
		httpClient: n.httpClient,
	}
}