package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/neul-labs/m9m/internal/mcp"
)

// NodeTypeInfo describes a node type
type NodeTypeInfo struct {
	Name        string                   `json:"name"`
	DisplayName string                   `json:"displayName"`
	Description string                   `json:"description"`
	Category    string                   `json:"category"`
	Version     int                      `json:"version"`
	Properties  []map[string]interface{} `json:"properties,omitempty"`
}

// NodeCategory describes a category of nodes
type NodeCategory struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Count       int    `json:"count"`
}

// nodeTypeCatalog is the comprehensive catalog of all m9m node types
var nodeTypeCatalog = []NodeTypeInfo{
	// Core nodes
	{Name: "n8n-nodes-base.start", DisplayName: "Start", Description: "Workflow entry point", Category: "core", Version: 1},

	// Transform nodes
	{Name: "n8n-nodes-base.set", DisplayName: "Set", Description: "Set values on items using assignments", Category: "transform", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "assignments", "type": "array", "description": "Array of name/value pairs to set"},
		}},
	{Name: "n8n-nodes-base.filter", DisplayName: "Filter", Description: "Filter items based on conditions (equals, contains, regex, etc.)", Category: "transform", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "field", "type": "string", "description": "Field to filter on"},
			{"name": "operator", "type": "string", "description": "Filter operator: equals, notEquals, contains, startsWith, endsWith, regex, greaterThan, lessThan"},
			{"name": "value", "type": "any", "description": "Value to compare against"},
			{"name": "combiner", "type": "string", "description": "How to combine conditions: and, or"},
		}},
	{Name: "n8n-nodes-base.code", DisplayName: "Code", Description: "Execute JavaScript, Python, or Go code", Category: "transform", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "language", "type": "string", "description": "Language: javascript, python, go"},
			{"name": "code", "type": "string", "description": "Code to execute"},
			{"name": "mode", "type": "string", "description": "Mode: runOnceForAllItems, runOnceForEachItem"},
		}},
	{Name: "n8n-nodes-base.function", DisplayName: "Function", Description: "Execute custom JavaScript function", Category: "transform", Version: 1},
	{Name: "n8n-nodes-base.merge", DisplayName: "Merge", Description: "Merge data from multiple inputs", Category: "transform", Version: 1},
	{Name: "n8n-nodes-base.json", DisplayName: "JSON", Description: "Parse and manipulate JSON data", Category: "transform", Version: 1},
	{Name: "n8n-nodes-base.switch", DisplayName: "Switch", Description: "Route items to different outputs based on conditions", Category: "transform", Version: 1},
	{Name: "n8n-nodes-base.splitInBatches", DisplayName: "Split In Batches", Description: "Split items into batches for processing", Category: "transform", Version: 1},
	{Name: "n8n-nodes-base.itemLists", DisplayName: "Item Lists", Description: "Manipulate item lists (split, concatenate, limit)", Category: "transform", Version: 1},

	// HTTP nodes
	{Name: "n8n-nodes-base.httpRequest", DisplayName: "HTTP Request", Description: "Make HTTP requests (GET, POST, PUT, DELETE, etc.)", Category: "http", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "url", "type": "string", "required": true, "description": "URL to request"},
			{"name": "method", "type": "string", "description": "HTTP method: GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS"},
			{"name": "headers", "type": "object", "description": "HTTP headers"},
			{"name": "body", "type": "any", "description": "Request body"},
		}},

	// Trigger nodes
	{Name: "n8n-nodes-base.webhook", DisplayName: "Webhook", Description: "Receive HTTP webhooks with authentication support", Category: "trigger", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "path", "type": "string", "description": "Webhook path"},
			{"name": "httpMethod", "type": "string", "description": "HTTP method to accept"},
			{"name": "authentication", "type": "string", "description": "Auth type: none, basicAuth, headerAuth"},
		}},
	{Name: "n8n-nodes-base.cron", DisplayName: "Cron", Description: "Schedule workflows using cron expressions", Category: "trigger", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "cronExpression", "type": "string", "description": "Cron expression (e.g., '0 */5 * * *')"},
		}},

	// Messaging nodes
	{Name: "n8n-nodes-base.slack", DisplayName: "Slack", Description: "Send messages to Slack channels via webhook or API", Category: "messaging", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "webhookUrl", "type": "string", "description": "Slack webhook URL"},
			{"name": "text", "type": "string", "required": true, "description": "Message text"},
			{"name": "channel", "type": "string", "description": "Channel to send to"},
		}},
	{Name: "n8n-nodes-base.discord", DisplayName: "Discord", Description: "Send messages to Discord channels", Category: "messaging", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "webhookUrl", "type": "string", "required": true, "description": "Discord webhook URL"},
			{"name": "content", "type": "string", "required": true, "description": "Message content"},
			{"name": "username", "type": "string", "description": "Bot username"},
		}},

	// Database nodes
	{Name: "n8n-nodes-base.postgres", DisplayName: "PostgreSQL", Description: "Execute PostgreSQL queries (SELECT, INSERT, UPDATE, DELETE)", Category: "database", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "operation", "type": "string", "description": "Operation: executeQuery, insert, update, delete"},
			{"name": "connectionUrl", "type": "string", "description": "PostgreSQL connection URL"},
			{"name": "query", "type": "string", "description": "SQL query to execute"},
		}},
	{Name: "n8n-nodes-base.mysql", DisplayName: "MySQL", Description: "Execute MySQL queries", Category: "database", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "host", "type": "string", "description": "MySQL host"},
			{"name": "database", "type": "string", "description": "Database name"},
			{"name": "query", "type": "string", "description": "SQL query to execute"},
		}},
	{Name: "n8n-nodes-base.sqlite", DisplayName: "SQLite", Description: "Execute SQLite queries", Category: "database", Version: 1},

	// Email nodes
	{Name: "n8n-nodes-base.emailSend", DisplayName: "Send Email", Description: "Send emails via SMTP", Category: "email", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "smtpHost", "type": "string", "description": "SMTP server host"},
			{"name": "fromEmail", "type": "string", "description": "From email address"},
			{"name": "toEmail", "type": "string", "required": true, "description": "To email address"},
			{"name": "subject", "type": "string", "description": "Email subject"},
			{"name": "body", "type": "string", "description": "Email body"},
		}},

	// Cloud nodes - AWS
	{Name: "n8n-nodes-base.awsS3", DisplayName: "AWS S3", Description: "AWS S3 operations (upload, download, list, delete, presigned URLs)", Category: "cloud", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "operation", "type": "string", "description": "Operation: upload, download, delete, list, copy, createBucket, generatePresignedUrl"},
			{"name": "bucket", "type": "string", "description": "S3 bucket name"},
			{"name": "key", "type": "string", "description": "Object key/path"},
		}},
	{Name: "n8n-nodes-base.awsLambda", DisplayName: "AWS Lambda", Description: "Invoke AWS Lambda functions", Category: "cloud", Version: 1},

	// Cloud nodes - Azure
	{Name: "n8n-nodes-base.azureBlobStorage", DisplayName: "Azure Blob Storage", Description: "Azure Blob Storage operations", Category: "cloud", Version: 1},

	// Cloud nodes - GCP
	{Name: "n8n-nodes-base.gcpCloudStorage", DisplayName: "GCP Cloud Storage", Description: "Google Cloud Storage operations", Category: "cloud", Version: 1},

	// AI nodes
	{Name: "n8n-nodes-base.openAi", DisplayName: "OpenAI", Description: "Get completions from OpenAI models (GPT-3.5, GPT-4)", Category: "ai", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "model", "type": "string", "description": "Model name (gpt-3.5-turbo, gpt-4)"},
			{"name": "prompt", "type": "string", "required": true, "description": "Prompt to send"},
			{"name": "maxTokens", "type": "integer", "description": "Maximum tokens in response"},
			{"name": "temperature", "type": "number", "description": "Sampling temperature (0-2)"},
		}},
	{Name: "n8n-nodes-base.anthropic", DisplayName: "Anthropic (Claude)", Description: "Get completions from Anthropic Claude models", Category: "ai", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "model", "type": "string", "description": "Model name (claude-3-5-sonnet-20241022)"},
			{"name": "prompt", "type": "string", "required": true, "description": "Prompt to send"},
			{"name": "maxTokens", "type": "integer", "description": "Maximum tokens in response"},
		}},

	// File nodes
	{Name: "n8n-nodes-base.readBinaryFile", DisplayName: "Read Binary File", Description: "Read files with encoding and hash support", Category: "file", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "filePath", "type": "string", "required": true, "description": "Path to file"},
			{"name": "encoding", "type": "string", "description": "Encoding: binary, base64, hex, utf8"},
			{"name": "includeHashes", "type": "boolean", "description": "Include MD5, SHA256 hashes"},
		}},
	{Name: "n8n-nodes-base.writeBinaryFile", DisplayName: "Write Binary File", Description: "Write files with encoding support", Category: "file", Version: 1},

	// VCS nodes
	{Name: "n8n-nodes-base.github", DisplayName: "GitHub", Description: "GitHub API operations (repos, issues, PRs, users)", Category: "vcs", Version: 1,
		Properties: []map[string]interface{}{
			{"name": "resource", "type": "string", "description": "Resource: repository, issue, pullRequest, user"},
			{"name": "operation", "type": "string", "description": "Operation: get, list"},
			{"name": "owner", "type": "string", "description": "Repository owner"},
			{"name": "repository", "type": "string", "description": "Repository name"},
		}},
	{Name: "n8n-nodes-base.gitlab", DisplayName: "GitLab", Description: "GitLab API operations", Category: "vcs", Version: 1},

	// Productivity nodes
	{Name: "n8n-nodes-base.googleSheets", DisplayName: "Google Sheets", Description: "Read and write Google Sheets", Category: "productivity", Version: 1},

	// Python code node
	{Name: "n8n-nodes-base.pythonCode", DisplayName: "Python Code", Description: "Execute Python code", Category: "code", Version: 1},
}

// nodeCategoryCatalog is the list of node categories
var nodeCategoryCatalog = []NodeCategory{
	{Name: "core", Description: "Core workflow nodes (start, end)", Count: 1},
	{Name: "transform", Description: "Data transformation nodes (set, filter, code, merge)", Count: 9},
	{Name: "http", Description: "HTTP request nodes", Count: 1},
	{Name: "trigger", Description: "Workflow trigger nodes (webhook, cron)", Count: 2},
	{Name: "messaging", Description: "Messaging platform nodes (Slack, Discord)", Count: 2},
	{Name: "database", Description: "Database nodes (PostgreSQL, MySQL, SQLite)", Count: 3},
	{Name: "email", Description: "Email sending nodes", Count: 1},
	{Name: "cloud", Description: "Cloud platform nodes (AWS, Azure, GCP)", Count: 4},
	{Name: "ai", Description: "AI/LLM nodes (OpenAI, Anthropic)", Count: 2},
	{Name: "file", Description: "File read/write nodes", Count: 2},
	{Name: "vcs", Description: "Version control nodes (GitHub, GitLab)", Count: 2},
	{Name: "productivity", Description: "Productivity app nodes (Google Sheets)", Count: 1},
	{Name: "code", Description: "Code execution nodes", Count: 1},
}

// NodeTypesListTool lists all available node types
type NodeTypesListTool struct {
	*BaseTool
}

// NewNodeTypesListTool creates a new node types list tool
func NewNodeTypesListTool() *NodeTypesListTool {
	return &NodeTypesListTool{
		BaseTool: NewBaseTool(
			"node_types_list",
			"List all available m9m node types. Optionally filter by category.",
			ObjectSchema(map[string]interface{}{
				"category": StringProp("Filter by category (transform, database, ai, cloud, etc.)"),
			}, nil),
		),
	}
}

// Execute lists all node types
func (t *NodeTypesListTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	category := GetString(args, "category")

	var filtered []NodeTypeInfo
	for _, node := range nodeTypeCatalog {
		if category == "" || node.Category == category {
			filtered = append(filtered, node)
		}
	}

	return mcp.SuccessJSON(map[string]interface{}{
		"nodeTypes": filtered,
		"count":     len(filtered),
	}), nil
}

// NodeTypeGetTool gets detailed info about a specific node type
type NodeTypeGetTool struct {
	*BaseTool
}

// NewNodeTypeGetTool creates a new node type get tool
func NewNodeTypeGetTool() *NodeTypeGetTool {
	return &NodeTypeGetTool{
		BaseTool: NewBaseTool(
			"node_type_get",
			"Get detailed information about a specific node type including its parameters and schema.",
			ObjectSchema(map[string]interface{}{
				"nodeType": StringProp("Node type name (e.g., 'n8n-nodes-base.httpRequest')"),
			}, []string{"nodeType"}),
		),
	}
}

// Execute gets a specific node type
func (t *NodeTypeGetTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	nodeType := GetString(args, "nodeType")

	for _, node := range nodeTypeCatalog {
		if node.Name == nodeType {
			return mcp.SuccessJSON(node), nil
		}
	}

	return mcp.ErrorContent(fmt.Sprintf("Node type not found: %s", nodeType)), nil
}

// NodeCategoriesListTool lists all node categories
type NodeCategoriesListTool struct {
	*BaseTool
}

// NewNodeCategoriesListTool creates a new categories list tool
func NewNodeCategoriesListTool() *NodeCategoriesListTool {
	return &NodeCategoriesListTool{
		BaseTool: NewBaseTool(
			"node_categories_list",
			"List all node categories with descriptions and node counts.",
			ObjectSchema(map[string]interface{}{}, nil),
		),
	}
}

// Execute lists all categories
func (t *NodeCategoriesListTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	return mcp.SuccessJSON(map[string]interface{}{
		"categories": nodeCategoryCatalog,
	}), nil
}

// NodeSearchTool searches for nodes by name or description
type NodeSearchTool struct {
	*BaseTool
}

// NewNodeSearchTool creates a new node search tool
func NewNodeSearchTool() *NodeSearchTool {
	return &NodeSearchTool{
		BaseTool: NewBaseTool(
			"node_search",
			"Search for nodes by name or description. Case-insensitive.",
			ObjectSchema(map[string]interface{}{
				"query": StringProp("Search query (matches against name and description)"),
			}, []string{"query"}),
		),
	}
}

// Execute searches for nodes
func (t *NodeSearchTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	query := strings.ToLower(GetString(args, "query"))

	var matches []NodeTypeInfo
	for _, node := range nodeTypeCatalog {
		if strings.Contains(strings.ToLower(node.Name), query) ||
			strings.Contains(strings.ToLower(node.DisplayName), query) ||
			strings.Contains(strings.ToLower(node.Description), query) {
			matches = append(matches, node)
		}
	}

	return mcp.SuccessJSON(map[string]interface{}{
		"results": matches,
		"count":   len(matches),
		"query":   query,
	}), nil
}

// RegisterNodeTools registers all node discovery tools with a registry
func RegisterNodeTools(registry *Registry) {
	registry.Register(NewNodeTypesListTool())
	registry.Register(NewNodeTypeGetTool())
	registry.Register(NewNodeCategoriesListTool())
	registry.Register(NewNodeSearchTool())
}
