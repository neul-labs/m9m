package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/neul-labs/m9m/internal/credentials"
	"github.com/neul-labs/m9m/internal/engine"
	"github.com/neul-labs/m9m/internal/scheduler"
	"github.com/neul-labs/m9m/internal/storage"
)

// Server represents an MCP server for m9m
type Server struct {
	mu sync.RWMutex

	// Core m9m components
	engine      engine.WorkflowEngine
	storage     storage.WorkflowStorage
	scheduler   *scheduler.WorkflowScheduler
	credentials *credentials.CredentialManager

	// MCP components
	tools     map[string]ToolHandler
	resources map[string]ResourceHandler

	// Server info
	name    string
	version string

	// State
	initialized bool
	logger      *log.Logger
}

// ToolHandler is a function that handles a tool call
type ToolHandler func(ctx context.Context, args map[string]interface{}) (*CallToolResult, error)

// ResourceHandler is a function that handles a resource read
type ResourceHandler func(ctx context.Context, uri string) (*ReadResourceResult, error)

// ServerOption configures the server
type ServerOption func(*Server)

// WithLogger sets a custom logger
func WithLogger(logger *log.Logger) ServerOption {
	return func(s *Server) {
		s.logger = logger
	}
}

// WithName sets the server name
func WithName(name string) ServerOption {
	return func(s *Server) {
		s.name = name
	}
}

// WithVersion sets the server version
func WithVersion(version string) ServerOption {
	return func(s *Server) {
		s.version = version
	}
}

// NewServer creates a new MCP server
func NewServer(opts ...ServerOption) *Server {
	s := &Server{
		tools:     make(map[string]ToolHandler),
		resources: make(map[string]ResourceHandler),
		name:      "m9m-mcp-server",
		version:   "1.0.0",
		logger:    log.Default(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Initialize sets up the server with m9m components
func (s *Server) Initialize(
	eng engine.WorkflowEngine,
	store storage.WorkflowStorage,
	sched *scheduler.WorkflowScheduler,
	creds *credentials.CredentialManager,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.engine = eng
	s.storage = store
	s.scheduler = sched
	s.credentials = creds

	return nil
}

// RegisterTool registers a tool handler
func (s *Server) RegisterTool(name string, handler ToolHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[name] = handler
}

// RegisterResource registers a resource handler
func (s *Server) RegisterResource(pattern string, handler ResourceHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resources[pattern] = handler
}

// Engine returns the workflow engine
func (s *Server) Engine() engine.WorkflowEngine {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.engine
}

// Storage returns the workflow storage
func (s *Server) Storage() storage.WorkflowStorage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.storage
}

// Scheduler returns the workflow scheduler
func (s *Server) Scheduler() *scheduler.WorkflowScheduler {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.scheduler
}

// Credentials returns the credential manager
func (s *Server) Credentials() *credentials.CredentialManager {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.credentials
}

// HandleRequest processes a single MCP request and returns a response
func (s *Server) HandleRequest(ctx context.Context, req *Request) *Response {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(ctx, req)
	case "initialized":
		return s.handleInitialized(ctx, req)
	case "ping":
		return s.handlePing(ctx, req)
	case "tools/list":
		return s.handleListTools(ctx, req)
	case "tools/call":
		return s.handleCallTool(ctx, req)
	case "resources/list":
		return s.handleListResources(ctx, req)
	case "resources/read":
		return s.handleReadResource(ctx, req)
	case "resources/templates/list":
		return s.handleListResourceTemplates(ctx, req)
	default:
		return &Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Error:   ErrMethodNotFound,
		}
	}
}

func (s *Server) handleInitialize(ctx context.Context, req *Request) *Response {
	var params InitializeParams
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &Response{
				JSONRPC: JSONRPCVersion,
				ID:      req.ID,
				Error:   InvalidParamsError(err.Error()),
			}
		}
	}

	s.mu.Lock()
	s.initialized = true
	s.mu.Unlock()

	result := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{
				ListChanged: false,
			},
			Resources: &ResourcesCapability{
				Subscribe:   false,
				ListChanged: false,
			},
			Logging: &LoggingCapability{},
		},
		ServerInfo: ServerInfo{
			Name:    s.name,
			Version: s.version,
		},
	}

	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      req.ID,
		Result:  result,
	}
}

func (s *Server) handleInitialized(ctx context.Context, req *Request) *Response {
	// Notification - no response needed, but we return empty for consistency
	return nil
}

func (s *Server) handlePing(ctx context.Context, req *Request) *Response {
	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      req.ID,
		Result:  map[string]interface{}{},
	}
}

func (s *Server) handleListTools(ctx context.Context, req *Request) *Response {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]Tool, 0, len(s.tools))
	for name := range s.tools {
		tool := s.getToolDefinition(name)
		if tool != nil {
			tools = append(tools, *tool)
		}
	}

	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      req.ID,
		Result:  ListToolsResult{Tools: tools},
	}
}

func (s *Server) handleCallTool(ctx context.Context, req *Request) *Response {
	var params CallToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Error:   InvalidParamsError(err.Error()),
		}
	}

	s.mu.RLock()
	handler, ok := s.tools[params.Name]
	s.mu.RUnlock()

	if !ok {
		return &Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Error:   NewError(ErrCodeMethodNotFound, fmt.Sprintf("Tool not found: %s", params.Name), nil),
		}
	}

	result, err := handler(ctx, params.Arguments)
	if err != nil {
		return &Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Result:  ErrorContent(err.Error()),
		}
	}

	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      req.ID,
		Result:  result,
	}
}

func (s *Server) handleListResources(ctx context.Context, req *Request) *Response {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resources := make([]Resource, 0)
	// Resources will be populated by resource handlers

	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      req.ID,
		Result:  ListResourcesResult{Resources: resources},
	}
}

func (s *Server) handleReadResource(ctx context.Context, req *Request) *Response {
	var params ReadResourceParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Error:   InvalidParamsError(err.Error()),
		}
	}

	// Find matching resource handler
	s.mu.RLock()
	var handler ResourceHandler
	for pattern, h := range s.resources {
		if matchResourcePattern(pattern, params.URI) {
			handler = h
			break
		}
	}
	s.mu.RUnlock()

	if handler == nil {
		return &Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Error:   ResourceNotFoundError(params.URI),
		}
	}

	result, err := handler(ctx, params.URI)
	if err != nil {
		return &Response{
			JSONRPC: JSONRPCVersion,
			ID:      req.ID,
			Error:   NewError(ErrCodeInternal, err.Error(), nil),
		}
	}

	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      req.ID,
		Result:  result,
	}
}

func (s *Server) handleListResourceTemplates(ctx context.Context, req *Request) *Response {
	templates := []ResourceTemplate{
		{
			URITemplate: "m9m://workflows",
			Name:        "Workflows",
			Description: "List of all workflows",
			MimeType:    "application/json",
		},
		{
			URITemplate: "m9m://workflows/{id}",
			Name:        "Workflow",
			Description: "Single workflow by ID",
			MimeType:    "application/json",
		},
		{
			URITemplate: "m9m://executions",
			Name:        "Executions",
			Description: "Recent workflow executions",
			MimeType:    "application/json",
		},
		{
			URITemplate: "m9m://executions/{id}",
			Name:        "Execution",
			Description: "Single execution by ID",
			MimeType:    "application/json",
		},
		{
			URITemplate: "m9m://executions/{id}/logs",
			Name:        "Execution Logs",
			Description: "Detailed logs for an execution",
			MimeType:    "application/json",
		},
		{
			URITemplate: "m9m://node-types",
			Name:        "Node Types",
			Description: "All available node types",
			MimeType:    "application/json",
		},
		{
			URITemplate: "m9m://node-types/{type}",
			Name:        "Node Type",
			Description: "Detailed node type schema",
			MimeType:    "application/json",
		},
		{
			URITemplate: "m9m://schedules",
			Name:        "Schedules",
			Description: "All workflow schedules",
			MimeType:    "application/json",
		},
		{
			URITemplate: "m9m://plugins",
			Name:        "Plugins",
			Description: "All custom plugins",
			MimeType:    "application/json",
		},
		{
			URITemplate: "m9m://plugins/{name}",
			Name:        "Plugin",
			Description: "Plugin source code",
			MimeType:    "application/json",
		},
	}

	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      req.ID,
		Result:  ListResourceTemplatesResult{ResourceTemplates: templates},
	}
}

// getToolDefinition returns the tool definition for a given tool name
// This will be overridden by the tools package
func (s *Server) getToolDefinition(name string) *Tool {
	// Default implementation - tools will register their own definitions
	return &Tool{
		Name:        name,
		Description: "m9m tool",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	}
}

// matchResourcePattern checks if a URI matches a resource pattern
func matchResourcePattern(pattern, uri string) bool {
	// Simple prefix match for now
	// TODO: Implement proper URI template matching
	return len(uri) >= len(pattern) && uri[:len(pattern)] == pattern
}

// Transport is the interface for MCP communication
type Transport interface {
	// ReadRequest reads the next request from the transport
	ReadRequest() (*Request, error)

	// WriteResponse writes a response to the transport
	WriteResponse(response *Response) error

	// WriteNotification writes a notification to the transport
	WriteNotification(notification *Notification) error

	// Close closes the transport
	Close() error
}

// Serve starts the server with the given transport
func (s *Server) Serve(ctx context.Context, transport Transport) error {
	defer transport.Close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := transport.ReadRequest()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			s.logger.Printf("Error reading request: %v", err)
			continue
		}

		resp := s.HandleRequest(ctx, req)
		if resp == nil {
			// Notification - no response needed
			continue
		}

		if err := transport.WriteResponse(resp); err != nil {
			s.logger.Printf("Error writing response: %v", err)
			continue
		}
	}
}

// Log sends a log message notification
func (s *Server) Log(transport Transport, level LogLevel, logger string, data interface{}) error {
	notification := &Notification{
		JSONRPC: JSONRPCVersion,
		Method:  "notifications/message",
		Params:  mustMarshal(LoggingMessageNotification{Level: level, Logger: logger, Data: data}),
	}
	return transport.WriteNotification(notification)
}

func mustMarshal(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
