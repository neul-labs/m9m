// Package mcp provides Model Context Protocol (MCP) server implementation for m9m.
// It enables Claude Code to orchestrate workflow automation through a standardized protocol.
package mcp

import (
	"encoding/json"
)

// Protocol version
const ProtocolVersion = "2024-11-05"

// JSONRPCVersion is the JSON-RPC version used by MCP
const JSONRPCVersion = "2.0"

// Request represents a JSON-RPC request from the MCP client
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC response to the MCP client
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

// Notification represents a JSON-RPC notification (no ID, no response expected)
type Notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Error represents a JSON-RPC error
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard JSON-RPC error codes
const (
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603
)

// MCP-specific error codes
const (
	ErrCodeWorkflowNotFound  = -32001
	ErrCodeExecutionFailed   = -32002
	ErrCodeCredentialDenied  = -32003
	ErrCodeValidationFailed  = -32004
	ErrCodePluginError       = -32005
	ErrCodeResourceNotFound  = -32006
	ErrCodeScheduleError     = -32007
	ErrCodeNodeNotFound      = -32008
	ErrCodeTimeout           = -32009
)

// NewError creates a new Error
func NewError(code int, message string, data interface{}) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// ServerCapabilities describes what the server can do
type ServerCapabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
	Logging   *LoggingCapability   `json:"logging,omitempty"`
}

// ToolsCapability indicates tools support
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability indicates resources support
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptsCapability indicates prompts support
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// LoggingCapability indicates logging support
type LoggingCapability struct{}

// ServerInfo provides server identification
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ClientInfo provides client identification
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeParams are sent by the client when initializing
type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      ClientInfo         `json:"clientInfo"`
}

// ClientCapabilities describes what the client can do
type ClientCapabilities struct {
	Roots    *RootsCapability    `json:"roots,omitempty"`
	Sampling *SamplingCapability `json:"sampling,omitempty"`
}

// RootsCapability indicates roots support
type RootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// SamplingCapability indicates sampling support
type SamplingCapability struct{}

// InitializeResult is returned after successful initialization
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

// Tool represents an MCP tool definition
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ListToolsResult is returned for tools/list
type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

// CallToolParams are sent when calling a tool
type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// CallToolResult is returned after a tool call
type CallToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock is a piece of content in a tool result
type ContentBlock struct {
	Type     string    `json:"type"` // "text", "image", "resource"
	Text     string    `json:"text,omitempty"`
	MimeType string    `json:"mimeType,omitempty"`
	Data     string    `json:"data,omitempty"` // base64 for images
	Resource *Resource `json:"resource,omitempty"`
}

// TextContent creates a text content block
func TextContent(text string) ContentBlock {
	return ContentBlock{
		Type: "text",
		Text: text,
	}
}

// JSONContent creates a text content block with JSON
func JSONContent(v interface{}) ContentBlock {
	data, _ := json.MarshalIndent(v, "", "  ")
	return ContentBlock{
		Type: "text",
		Text: string(data),
	}
}

// ErrorContent creates an error result
func ErrorContent(message string) *CallToolResult {
	return &CallToolResult{
		Content: []ContentBlock{TextContent(message)},
		IsError: true,
	}
}

// SuccessContent creates a success result with text
func SuccessContent(text string) *CallToolResult {
	return &CallToolResult{
		Content: []ContentBlock{TextContent(text)},
	}
}

// SuccessJSON creates a success result with JSON
func SuccessJSON(v interface{}) *CallToolResult {
	return &CallToolResult{
		Content: []ContentBlock{JSONContent(v)},
	}
}

// Resource represents an MCP resource
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ListResourcesResult is returned for resources/list
type ListResourcesResult struct {
	Resources []Resource `json:"resources"`
}

// ReadResourceParams are sent when reading a resource
type ReadResourceParams struct {
	URI string `json:"uri"`
}

// ReadResourceResult is returned after reading a resource
type ReadResourceResult struct {
	Contents []ResourceContent `json:"contents"`
}

// ResourceContent is the content of a resource
type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"` // base64
}

// ResourceTemplate represents a resource template with URI patterns
type ResourceTemplate struct {
	URITemplate string `json:"uriTemplate"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ListResourceTemplatesResult is returned for resources/templates/list
type ListResourceTemplatesResult struct {
	ResourceTemplates []ResourceTemplate `json:"resourceTemplates"`
}

// LogLevel represents logging severity
type LogLevel string

const (
	LogLevelDebug     LogLevel = "debug"
	LogLevelInfo      LogLevel = "info"
	LogLevelNotice    LogLevel = "notice"
	LogLevelWarning   LogLevel = "warning"
	LogLevelError     LogLevel = "error"
	LogLevelCritical  LogLevel = "critical"
	LogLevelAlert     LogLevel = "alert"
	LogLevelEmergency LogLevel = "emergency"
)

// LoggingMessageNotification is sent by the server for logging
type LoggingMessageNotification struct {
	Level  LogLevel    `json:"level"`
	Logger string      `json:"logger,omitempty"`
	Data   interface{} `json:"data"`
}

// ProgressNotification is sent to report progress
type ProgressNotification struct {
	ProgressToken interface{} `json:"progressToken"`
	Progress      float64     `json:"progress"`
	Total         float64     `json:"total,omitempty"`
}

// Prompt represents an MCP prompt template
type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument is an argument for a prompt
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// ListPromptsResult is returned for prompts/list
type ListPromptsResult struct {
	Prompts []Prompt `json:"prompts"`
}

// GetPromptParams are sent when getting a prompt
type GetPromptParams struct {
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

// GetPromptResult is returned after getting a prompt
type GetPromptResult struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// PromptMessage is a message in a prompt
type PromptMessage struct {
	Role    string        `json:"role"` // "user" or "assistant"
	Content ContentBlock  `json:"content"`
}
