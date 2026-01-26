package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/neul-labs/m9m/internal/credentials"
	"github.com/neul-labs/m9m/internal/engine"
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/storage"
	"github.com/neul-labs/m9m/internal/workspace"

	// Import node packages
	"github.com/neul-labs/m9m/internal/nodes/ai"
	"github.com/neul-labs/m9m/internal/nodes/cli"
	"github.com/neul-labs/m9m/internal/nodes/core"
	"github.com/neul-labs/m9m/internal/nodes/database"
	"github.com/neul-labs/m9m/internal/nodes/email"
	"github.com/neul-labs/m9m/internal/nodes/http"
	"github.com/neul-labs/m9m/internal/nodes/messaging"
	"github.com/neul-labs/m9m/internal/nodes/timer"
	"github.com/neul-labs/m9m/internal/nodes/transform"
	"github.com/neul-labs/m9m/internal/nodes/trigger"
	"github.com/neul-labs/m9m/internal/nodes/vcs"
)

const (
	// DefaultSocketPath is the default Unix socket path
	DefaultSocketPath = "/tmp/m9m.sock"

	// DefaultIdleTimeout is the default idle timeout before daemon exits
	DefaultIdleTimeout = 5 * time.Minute
)

// Request represents a JSON-RPC style request
type Request struct {
	Method    string                 `json:"method"`
	Workspace string                 `json:"workspace"`
	Params    map[string]interface{} `json:"params,omitempty"`
	ID        string                 `json:"id,omitempty"`
}

// Response represents a JSON-RPC style response
type Response struct {
	Success bool                   `json:"success"`
	Data    interface{}            `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
	ID      string                 `json:"id,omitempty"`
}

// workspaceContext holds engine and storage for a workspace
type workspaceContext struct {
	engine    engine.WorkflowEngine
	storage   storage.WorkflowStorage
	credMgr   *credentials.CredentialManager
	lastUsed  time.Time
	workspace *workspace.Workspace
}

// Daemon is the background service that handles all workspaces
type Daemon struct {
	socketPath   string
	idleTimeout  time.Duration
	listener     net.Listener
	workspaces   map[string]*workspaceContext
	workspaceMgr *workspace.Manager
	mu           sync.RWMutex
	lastActivity time.Time
	logger       *log.Logger
	shutdown     chan struct{}
}

// DaemonConfig holds daemon configuration
type DaemonConfig struct {
	SocketPath  string
	IdleTimeout time.Duration
	Logger      *log.Logger
}

// NewDaemon creates a new daemon instance
func NewDaemon(config *DaemonConfig) (*Daemon, error) {
	if config == nil {
		config = &DaemonConfig{}
	}

	socketPath := config.SocketPath
	if socketPath == "" {
		socketPath = DefaultSocketPath
	}

	idleTimeout := config.IdleTimeout
	if idleTimeout == 0 {
		idleTimeout = DefaultIdleTimeout
	}

	logger := config.Logger
	if logger == nil {
		logger = log.New(os.Stderr, "[m9m-daemon] ", log.LstdFlags)
	}

	mgr, err := workspace.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace manager: %w", err)
	}

	return &Daemon{
		socketPath:   socketPath,
		idleTimeout:  idleTimeout,
		workspaces:   make(map[string]*workspaceContext),
		workspaceMgr: mgr,
		lastActivity: time.Now(),
		logger:       logger,
		shutdown:     make(chan struct{}),
	}, nil
}

// Start starts the daemon
func (d *Daemon) Start(ctx context.Context) error {
	// Remove existing socket if it exists
	if err := os.Remove(d.socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	// Create listener
	listener, err := net.Listen("unix", d.socketPath)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}
	d.listener = listener

	// Set socket permissions
	if err := os.Chmod(d.socketPath, 0600); err != nil {
		d.logger.Printf("Warning: failed to set socket permissions: %v", err)
	}

	d.logger.Printf("Daemon started, listening on %s", d.socketPath)

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start idle checker
	go d.idleChecker(ctx)

	// Accept connections
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-d.shutdown:
					return
				default:
					d.logger.Printf("Accept error: %v", err)
					continue
				}
			}
			go d.handleConnection(conn)
		}
	}()

	// Wait for shutdown
	select {
	case <-ctx.Done():
		d.logger.Println("Context cancelled, shutting down...")
	case <-sigCh:
		d.logger.Println("Received signal, shutting down...")
	case <-d.shutdown:
		d.logger.Println("Idle timeout reached, shutting down...")
	}

	return d.Stop()
}

// Stop stops the daemon
func (d *Daemon) Stop() error {
	close(d.shutdown)

	if d.listener != nil {
		d.listener.Close()
	}

	// Close all workspace contexts
	d.mu.Lock()
	for name, ctx := range d.workspaces {
		if ctx.storage != nil {
			ctx.storage.Close()
		}
		d.logger.Printf("Closed workspace: %s", name)
	}
	d.mu.Unlock()

	// Remove socket file
	os.Remove(d.socketPath)

	d.logger.Println("Daemon stopped")
	return nil
}

// idleChecker monitors for idle timeout
func (d *Daemon) idleChecker(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.shutdown:
			return
		case <-ticker.C:
			d.mu.RLock()
			idle := time.Since(d.lastActivity)
			d.mu.RUnlock()

			if idle > d.idleTimeout {
				d.logger.Printf("Idle timeout reached (%v), initiating shutdown", d.idleTimeout)
				close(d.shutdown)
				return
			}
		}
	}
}

// handleConnection handles a single client connection
func (d *Daemon) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Update activity
	d.mu.Lock()
	d.lastActivity = time.Now()
	d.mu.Unlock()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	var req Request
	if err := decoder.Decode(&req); err != nil {
		encoder.Encode(Response{
			Success: false,
			Error:   fmt.Sprintf("failed to decode request: %v", err),
		})
		return
	}

	resp := d.handleRequest(&req)
	resp.ID = req.ID
	encoder.Encode(resp)
}

// handleRequest processes a request
func (d *Daemon) handleRequest(req *Request) Response {
	switch req.Method {
	case "ping":
		return Response{Success: true, Data: "pong"}

	case "status":
		return d.handleStatus(req)

	case "workflow.list":
		return d.handleWorkflowList(req)

	case "workflow.get":
		return d.handleWorkflowGet(req)

	case "workflow.create":
		return d.handleWorkflowCreate(req)

	case "workflow.run":
		return d.handleWorkflowRun(req)

	case "workflow.delete":
		return d.handleWorkflowDelete(req)

	case "execution.list":
		return d.handleExecutionList(req)

	case "execution.get":
		return d.handleExecutionGet(req)

	default:
		return Response{
			Success: false,
			Error:   fmt.Sprintf("unknown method: %s", req.Method),
		}
	}
}

// getWorkspaceContext gets or creates a workspace context
func (d *Daemon) getWorkspaceContext(workspaceName string) (*workspaceContext, error) {
	// Use current workspace if not specified
	if workspaceName == "" {
		current, err := d.workspaceMgr.GetCurrent()
		if err != nil || current == "" {
			return nil, fmt.Errorf("no workspace specified and no current workspace set")
		}
		workspaceName = current
	}

	// Check if already loaded
	d.mu.RLock()
	ctx, exists := d.workspaces[workspaceName]
	d.mu.RUnlock()

	if exists {
		d.mu.Lock()
		ctx.lastUsed = time.Now()
		d.mu.Unlock()
		return ctx, nil
	}

	// Load workspace
	ws, err := d.workspaceMgr.Get(workspaceName)
	if err != nil {
		return nil, fmt.Errorf("workspace not found: %s", workspaceName)
	}

	// Initialize storage
	storagePath, err := d.workspaceMgr.GetStoragePath(workspaceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage path: %w", err)
	}
	store, err := storage.NewSQLiteStorage(storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize engine
	eng := engine.NewWorkflowEngine()
	registerNodes(eng)

	// Initialize credential manager
	credMgr, _ := credentials.NewCredentialManager()
	if credMgr != nil {
		eng.SetCredentialManager(credMgr)
	}

	ctx = &workspaceContext{
		engine:    eng,
		storage:   store,
		credMgr:   credMgr,
		lastUsed:  time.Now(),
		workspace: ws,
	}

	d.mu.Lock()
	d.workspaces[workspaceName] = ctx
	d.mu.Unlock()

	d.logger.Printf("Loaded workspace: %s", workspaceName)
	return ctx, nil
}

// Handler implementations

func (d *Daemon) handleStatus(req *Request) Response {
	d.mu.RLock()
	loadedWorkspaces := make([]string, 0, len(d.workspaces))
	for name := range d.workspaces {
		loadedWorkspaces = append(loadedWorkspaces, name)
	}
	d.mu.RUnlock()

	current, _ := d.workspaceMgr.GetCurrent()

	return Response{
		Success: true,
		Data: map[string]interface{}{
			"running":          true,
			"socketPath":       d.socketPath,
			"currentWorkspace": current,
			"loadedWorkspaces": loadedWorkspaces,
			"idleTimeout":      d.idleTimeout.String(),
		},
	}
}

func (d *Daemon) handleWorkflowList(req *Request) Response {
	ctx, err := d.getWorkspaceContext(req.Workspace)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	limit := 100
	if l, ok := req.Params["limit"].(float64); ok {
		limit = int(l)
	}

	filters := storage.WorkflowFilters{
		Limit: limit,
	}

	if search, ok := req.Params["search"].(string); ok {
		filters.Search = search
	}

	workflows, total, err := ctx.storage.ListWorkflows(filters)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	// Convert to simplified format
	result := make([]map[string]interface{}, len(workflows))
	for i, wf := range workflows {
		result[i] = map[string]interface{}{
			"id":          wf.ID,
			"name":        wf.Name,
			"description": wf.Description,
			"active":      wf.Active,
			"nodeCount":   len(wf.Nodes),
			"createdAt":   wf.CreatedAt,
			"updatedAt":   wf.UpdatedAt,
		}
	}

	return Response{
		Success: true,
		Data: map[string]interface{}{
			"workflows": result,
			"total":     total,
		},
	}
}

func (d *Daemon) handleWorkflowGet(req *Request) Response {
	ctx, err := d.getWorkspaceContext(req.Workspace)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	id, ok := req.Params["id"].(string)
	if !ok {
		// Try by name
		name, ok := req.Params["name"].(string)
		if !ok {
			return Response{Success: false, Error: "id or name required"}
		}
		// Search by name
		workflows, _, err := ctx.storage.ListWorkflows(storage.WorkflowFilters{Search: name, Limit: 1})
		if err != nil || len(workflows) == 0 {
			return Response{Success: false, Error: fmt.Sprintf("workflow not found: %s", name)}
		}
		id = workflows[0].ID
	}

	workflow, err := ctx.storage.GetWorkflow(id)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	return Response{Success: true, Data: workflow}
}

func (d *Daemon) handleWorkflowCreate(req *Request) Response {
	ctx, err := d.getWorkspaceContext(req.Workspace)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	// Extract workflow data
	workflowData, ok := req.Params["workflow"].(map[string]interface{})
	if !ok {
		return Response{Success: false, Error: "workflow data required"}
	}

	// Convert to workflow model
	workflowJSON, _ := json.Marshal(workflowData)
	var workflow model.Workflow
	if err := json.Unmarshal(workflowJSON, &workflow); err != nil {
		return Response{Success: false, Error: fmt.Sprintf("invalid workflow format: %v", err)}
	}

	// Save workflow
	if err := ctx.storage.SaveWorkflow(&workflow); err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	return Response{
		Success: true,
		Data: map[string]interface{}{
			"id":   workflow.ID,
			"name": workflow.Name,
		},
	}
}

func (d *Daemon) handleWorkflowRun(req *Request) Response {
	ctx, err := d.getWorkspaceContext(req.Workspace)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	// Get workflow by ID or name
	var workflow *model.Workflow
	if id, ok := req.Params["id"].(string); ok {
		workflow, err = ctx.storage.GetWorkflow(id)
	} else if name, ok := req.Params["name"].(string); ok {
		workflows, _, err := ctx.storage.ListWorkflows(storage.WorkflowFilters{Search: name, Limit: 1})
		if err == nil && len(workflows) > 0 {
			workflow = workflows[0]
		}
	}

	if workflow == nil {
		return Response{Success: false, Error: "workflow not found"}
	}

	// Prepare input data
	var inputData []model.DataItem
	if input, ok := req.Params["input"].(map[string]interface{}); ok {
		inputData = []model.DataItem{{JSON: input}}
	} else {
		inputData = []model.DataItem{{JSON: make(map[string]interface{})}}
	}

	// Execute workflow
	startTime := time.Now()
	result, err := ctx.engine.ExecuteWorkflow(workflow, inputData)

	// Record execution
	execution := &model.WorkflowExecution{
		WorkflowID: workflow.ID,
		Status:     "success",
		Mode:       "cli",
		StartedAt:  startTime,
	}

	finishedAt := time.Now()
	execution.FinishedAt = &finishedAt

	if err != nil {
		execution.Status = "error"
		execution.Error = err
	} else if result.Error != nil {
		execution.Status = "error"
		execution.Error = result.Error
	} else {
		execution.Data = result.Data
	}

	ctx.storage.SaveExecution(execution)

	if execution.Status == "error" {
		errMsg := "execution failed"
		if execution.Error != nil {
			errMsg = execution.Error.Error()
		}
		return Response{Success: false, Error: errMsg}
	}

	return Response{
		Success: true,
		Data: map[string]interface{}{
			"executionId": execution.ID,
			"status":      execution.Status,
			"data":        result.Data,
			"duration":    time.Since(startTime).String(),
		},
	}
}

func (d *Daemon) handleWorkflowDelete(req *Request) Response {
	ctx, err := d.getWorkspaceContext(req.Workspace)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	id, ok := req.Params["id"].(string)
	if !ok {
		return Response{Success: false, Error: "id required"}
	}

	if err := ctx.storage.DeleteWorkflow(id); err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	return Response{Success: true, Data: map[string]interface{}{"deleted": id}}
}

func (d *Daemon) handleExecutionList(req *Request) Response {
	ctx, err := d.getWorkspaceContext(req.Workspace)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	filters := storage.ExecutionFilters{
		Limit: 50,
	}

	if workflowID, ok := req.Params["workflowId"].(string); ok {
		filters.WorkflowID = workflowID
	}

	executions, total, err := ctx.storage.ListExecutions(filters)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	return Response{
		Success: true,
		Data: map[string]interface{}{
			"executions": executions,
			"total":      total,
		},
	}
}

func (d *Daemon) handleExecutionGet(req *Request) Response {
	ctx, err := d.getWorkspaceContext(req.Workspace)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	id, ok := req.Params["id"].(string)
	if !ok {
		return Response{Success: false, Error: "id required"}
	}

	execution, err := ctx.storage.GetExecution(id)
	if err != nil {
		return Response{Success: false, Error: err.Error()}
	}

	return Response{Success: true, Data: execution}
}

// registerNodes registers all node executors with the engine
func registerNodes(eng engine.WorkflowEngine) {
	// Core nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.start", core.NewStartNode())

	// Transform nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.set", transform.NewSetNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.filter", transform.NewFilterNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.code", transform.NewCodeNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.function", transform.NewFunctionNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.merge", transform.NewMergeNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.json", transform.NewJSONNode())

	// HTTP nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.httpRequest", http.NewHTTPRequestNode())

	// Trigger nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.webhook", trigger.NewWebhookNode())

	// Timer nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.cron", timer.NewCronNode())

	// Messaging nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.slack", messaging.NewSlackNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.discord", messaging.NewDiscordNode())

	// Database nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.postgres", database.NewPostgresNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.mysql", database.NewMySQLNode())

	// Email nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.emailSend", email.NewSendEmailNode())

	// AI nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.openAi", ai.NewOpenAINode())
	eng.RegisterNodeExecutor("n8n-nodes-base.anthropic", ai.NewAnthropicNode())

	// VCS nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.github", vcs.NewGitHubNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.gitlab", vcs.NewGitLabNode())

	// CLI nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.cliExecute", cli.NewExecuteNode())
}

// GetSocketPath returns the socket path for the daemon
func GetSocketPath() string {
	// Check for custom socket path from environment
	if path := os.Getenv("M9M_SOCKET"); path != "" {
		return path
	}
	return DefaultSocketPath
}

// GetPidFile returns the PID file path
func GetPidFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".m9m", "daemon.pid")
}
