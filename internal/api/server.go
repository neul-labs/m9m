package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/dipankar/n8n-go/internal/engine"
	"github.com/dipankar/n8n-go/internal/model"
	"github.com/dipankar/n8n-go/internal/scheduler"
	"github.com/dipankar/n8n-go/internal/storage"
)

// APIServer provides n8n-compatible REST API
type APIServer struct {
	engine    engine.WorkflowEngine
	scheduler *scheduler.WorkflowScheduler
	storage   storage.WorkflowStorage
	upgrader  websocket.Upgrader
	wsClients map[string]*websocket.Conn
}

// NewAPIServer creates a new API server instance
func NewAPIServer(eng engine.WorkflowEngine, scheduler *scheduler.WorkflowScheduler, storage storage.WorkflowStorage) *APIServer {
	return &APIServer{
		engine:    eng,
		scheduler: scheduler,
		storage:   storage,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
		wsClients: make(map[string]*websocket.Conn),
	}
}

// RegisterRoutes registers all API routes
func (s *APIServer) RegisterRoutes(router *mux.Router) {
	// Health and info endpoints
	router.HandleFunc("/health", s.HealthCheck).Methods("GET")
	router.HandleFunc("/healthz", s.HealthCheck).Methods("GET")
	router.HandleFunc("/ready", s.ReadyCheck).Methods("GET")

	// API v1 routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Workflow management
	api.HandleFunc("/workflows", s.ListWorkflows).Methods("GET", "OPTIONS")
	api.HandleFunc("/workflows", s.CreateWorkflow).Methods("POST", "OPTIONS")
	api.HandleFunc("/workflows/{id}", s.GetWorkflow).Methods("GET", "OPTIONS")
	api.HandleFunc("/workflows/{id}", s.UpdateWorkflow).Methods("PUT", "PATCH", "OPTIONS")
	api.HandleFunc("/workflows/{id}", s.DeleteWorkflow).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/workflows/{id}/activate", s.ActivateWorkflow).Methods("POST", "OPTIONS")
	api.HandleFunc("/workflows/{id}/deactivate", s.DeactivateWorkflow).Methods("POST", "OPTIONS")

	// Workflow execution
	api.HandleFunc("/workflows/{id}/execute", s.ExecuteWorkflow).Methods("POST", "OPTIONS")
	api.HandleFunc("/workflows/run", s.ExecuteWorkflow).Methods("POST", "OPTIONS") // n8n alias

	// Executions
	api.HandleFunc("/executions", s.ListExecutions).Methods("GET", "OPTIONS")
	api.HandleFunc("/executions/{id}", s.GetExecution).Methods("GET", "OPTIONS")
	api.HandleFunc("/executions/{id}", s.DeleteExecution).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/executions/{id}/retry", s.RetryExecution).Methods("POST", "OPTIONS")
	api.HandleFunc("/executions/{id}/cancel", s.CancelExecution).Methods("POST", "OPTIONS")

	// Credentials
	api.HandleFunc("/credentials", s.ListCredentials).Methods("GET", "OPTIONS")
	api.HandleFunc("/credentials", s.CreateCredential).Methods("POST", "OPTIONS")
	api.HandleFunc("/credentials/{id}", s.GetCredential).Methods("GET", "OPTIONS")
	api.HandleFunc("/credentials/{id}", s.UpdateCredential).Methods("PUT", "OPTIONS")
	api.HandleFunc("/credentials/{id}", s.DeleteCredential).Methods("DELETE", "OPTIONS")

	// Node types
	api.HandleFunc("/node-types", s.ListNodeTypes).Methods("GET", "OPTIONS")
	api.HandleFunc("/node-types/{name}", s.GetNodeType).Methods("GET", "OPTIONS")

	// Settings
	api.HandleFunc("/settings", s.GetSettings).Methods("GET", "OPTIONS")
	api.HandleFunc("/settings", s.UpdateSettings).Methods("PATCH", "OPTIONS")

	// Tags
	api.HandleFunc("/tags", s.ListTags).Methods("GET", "OPTIONS")
	api.HandleFunc("/tags", s.CreateTag).Methods("POST", "OPTIONS")
	api.HandleFunc("/tags/{id}", s.UpdateTag).Methods("PUT", "OPTIONS")
	api.HandleFunc("/tags/{id}", s.DeleteTag).Methods("DELETE", "OPTIONS")

	// System
	api.HandleFunc("/version", s.GetVersion).Methods("GET", "OPTIONS")
	api.HandleFunc("/metrics", s.GetMetrics).Methods("GET", "OPTIONS")

	// WebSocket for real-time updates
	api.HandleFunc("/push", s.HandleWebSocket).Methods("GET")
}

// HealthCheck handles health check requests
func (s *APIServer) HealthCheck(w http.ResponseWriter, r *http.Request) {
	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"service": "n8n-go",
		"version": "0.2.0",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

// ReadyCheck handles readiness check requests
func (s *APIServer) ReadyCheck(w http.ResponseWriter, r *http.Request) {
	// Check if engine is ready
	ready := s.engine != nil && s.storage != nil

	if ready {
		s.sendJSON(w, http.StatusOK, map[string]interface{}{
			"status": "ready",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	} else {
		s.sendJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
			"status": "not ready",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// GetVersion returns version information
func (s *APIServer) GetVersion(w http.ResponseWriter, r *http.Request) {
	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"n8nVersion":    "1.0.0-compatible",
		"serverVersion": "0.2.0",
		"implementation": "n8n-go",
		"compatibility": map[string]interface{}{
			"workflows":   true,
			"nodes":       true,
			"expressions": true,
			"credentials": true,
		},
	})
}

// GetSettings returns system settings (n8n compatible)
func (s *APIServer) GetSettings(w http.ResponseWriter, r *http.Request) {
	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"timezone":           "UTC",
		"executionMode":      "regular",
		"saveDataSuccessExecution": "all",
		"saveDataErrorExecution":   "all",
		"saveExecutionProgress": true,
		"saveManualExecutions":  true,
		"communityNodesEnabled": false,
		"versionNotifications":  map[string]bool{
			"enabled": false,
		},
		"instanceId": "n8n-go-instance",
		"telemetry": map[string]bool{
			"enabled": false,
		},
	})
}

// UpdateSettings handles settings updates (stub)
func (s *APIServer) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var settings map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	// TODO: Persist settings
	s.sendJSON(w, http.StatusOK, settings)
}

// ListNodeTypes returns available node types
func (s *APIServer) ListNodeTypes(w http.ResponseWriter, r *http.Request) {
	// Return registered node types
	nodeTypes := []map[string]interface{}{
		{
			"name":        "n8n-nodes-base.httpRequest",
			"displayName": "HTTP Request",
			"description": "Makes HTTP requests",
			"version":     1,
			"defaults": map[string]interface{}{
				"name": "HTTP Request",
			},
			"inputs":  []string{"main"},
			"outputs": []string{"main"},
			"properties": []map[string]interface{}{
				{
					"displayName": "Method",
					"name":        "method",
					"type":        "options",
					"options": []map[string]string{
						{"name": "GET", "value": "GET"},
						{"name": "POST", "value": "POST"},
						{"name": "PUT", "value": "PUT"},
						{"name": "DELETE", "value": "DELETE"},
					},
					"default": "GET",
				},
				{
					"displayName": "URL",
					"name":        "url",
					"type":        "string",
					"default":     "",
					"required":    true,
				},
			},
		},
		{
			"name":        "n8n-nodes-base.set",
			"displayName": "Set",
			"description": "Sets values in items",
			"version":     1,
			"defaults": map[string]interface{}{
				"name": "Set",
			},
			"inputs":  []string{"main"},
			"outputs": []string{"main"},
		},
		{
			"name":        "n8n-nodes-base.function",
			"displayName": "Function",
			"description": "Execute custom JavaScript code",
			"version":     1,
			"defaults": map[string]interface{}{
				"name": "Function",
			},
			"inputs":  []string{"main"},
			"outputs": []string{"main"},
		},
		{
			"name":        "n8n-nodes-base.code",
			"displayName": "Code",
			"description": "Execute custom code",
			"version":     1,
			"defaults": map[string]interface{}{
				"name": "Code",
			},
			"inputs":  []string{"main"},
			"outputs": []string{"main"},
		},
	}

	s.sendJSON(w, http.StatusOK, nodeTypes)
}

// GetNodeType returns a specific node type
func (s *APIServer) GetNodeType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	// For now, return a basic node type structure
	nodeType := map[string]interface{}{
		"name":        name,
		"displayName": name,
		"description": fmt.Sprintf("Node type: %s", name),
		"version":     1,
		"defaults": map[string]interface{}{
			"name": name,
		},
		"inputs":  []string{"main"},
		"outputs": []string{"main"},
	}

	s.sendJSON(w, http.StatusOK, nodeType)
}

// HandleWebSocket handles WebSocket connections for real-time updates
func (s *APIServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade to WebSocket", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	// Generate client ID
	clientID := fmt.Sprintf("client_%d", time.Now().UnixNano())
	s.wsClients[clientID] = conn
	defer delete(s.wsClients, clientID)

	// Send initial connection message
	conn.WriteJSON(map[string]interface{}{
		"type": "connected",
		"data": map[string]interface{}{
			"clientId": clientID,
			"time":     time.Now().UTC().Format(time.RFC3339),
		},
	})

	// Keep connection alive
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// Echo message back (for now)
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err == nil {
			conn.WriteJSON(map[string]interface{}{
				"type": "response",
				"data": msg,
			})
		}
	}
}

// BroadcastExecutionUpdate sends execution updates to all connected clients
func (s *APIServer) BroadcastExecutionUpdate(execution *model.WorkflowExecution) {
	message := map[string]interface{}{
		"type": "executionUpdate",
		"data": execution,
	}

	for _, conn := range s.wsClients {
		conn.WriteJSON(message)
	}
}

// Helper methods

func (s *APIServer) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *APIServer) sendError(w http.ResponseWriter, status int, message string, err error) {
	response := map[string]interface{}{
		"error":   true,
		"message": message,
		"code":    status,
	}

	if err != nil {
		response["details"] = err.Error()
	}

	s.sendJSON(w, status, response)
}

// Workflow handlers

func (s *APIServer) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for filtering
	active := r.URL.Query().Get("active")
	search := r.URL.Query().Get("search")
	offsetStr := r.URL.Query().Get("offset")
	limitStr := r.URL.Query().Get("limit")

	offset := 0
	limit := 20
	if offsetStr != "" {
		fmt.Sscanf(offsetStr, "%d", &offset)
	}
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	filters := storage.WorkflowFilters{
		Search: search,
		Offset: offset,
		Limit:  limit,
	}

	if active != "" {
		activeBool := active == "true"
		filters.Active = &activeBool
	}

	workflows, total, err := s.storage.ListWorkflows(filters)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to list workflows", err)
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"data":   workflows,
		"total":  total,
		"offset": offset,
		"limit":  limit,
	})
}

func (s *APIServer) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var workflow model.Workflow
	if err := json.NewDecoder(r.Body).Decode(&workflow); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	if err := s.storage.SaveWorkflow(&workflow); err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to save workflow", err)
		return
	}

	s.sendJSON(w, http.StatusCreated, workflow)
}

func (s *APIServer) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	workflow, err := s.storage.GetWorkflow(id)
	if err != nil {
		s.sendError(w, http.StatusNotFound, "Workflow not found", err)
		return
	}

	s.sendJSON(w, http.StatusOK, workflow)
}

func (s *APIServer) UpdateWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var workflow model.Workflow
	if err := json.NewDecoder(r.Body).Decode(&workflow); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	if err := s.storage.UpdateWorkflow(id, &workflow); err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to update workflow", err)
		return
	}

	s.sendJSON(w, http.StatusOK, workflow)
}

func (s *APIServer) DeleteWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := s.storage.DeleteWorkflow(id); err != nil {
		s.sendError(w, http.StatusNotFound, "Workflow not found", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *APIServer) ActivateWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := s.storage.ActivateWorkflow(id); err != nil {
		s.sendError(w, http.StatusNotFound, "Workflow not found", err)
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Workflow activated",
		"active":  true,
	})
}

func (s *APIServer) DeactivateWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := s.storage.DeactivateWorkflow(id); err != nil {
		s.sendError(w, http.StatusNotFound, "Workflow not found", err)
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Workflow deactivated",
		"active":  false,
	})
}

func (s *APIServer) ExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Load workflow
	workflow, err := s.storage.GetWorkflow(id)
	if err != nil {
		s.sendError(w, http.StatusNotFound, "Workflow not found", err)
		return
	}

	// Parse input data if provided
	var inputData []model.DataItem
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&inputData)
	}
	if len(inputData) == 0 {
		inputData = []model.DataItem{{JSON: make(map[string]interface{})}}
	}

	// Execute workflow
	startTime := time.Now()
	result, err := s.engine.ExecuteWorkflow(workflow, inputData)

	execution := &model.WorkflowExecution{
		WorkflowID: id,
		StartedAt:  startTime,
		Mode:       "manual",
		Status:     "completed",
	}

	finishedAt := time.Now()
	execution.FinishedAt = &finishedAt

	if err != nil {
		execution.Status = "failed"
		execution.Error = err
	} else {
		execution.Data = result.Data
	}

	// Save execution
	s.storage.SaveExecution(execution)

	s.sendJSON(w, http.StatusOK, execution)
}

// Execution handlers

func (s *APIServer) ListExecutions(w http.ResponseWriter, r *http.Request) {
	workflowID := r.URL.Query().Get("workflowId")
	status := r.URL.Query().Get("status")
	offsetStr := r.URL.Query().Get("offset")
	limitStr := r.URL.Query().Get("limit")

	offset := 0
	limit := 20
	if offsetStr != "" {
		fmt.Sscanf(offsetStr, "%d", &offset)
	}
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	filters := storage.ExecutionFilters{
		WorkflowID: workflowID,
		Status:     status,
		Offset:     offset,
		Limit:      limit,
	}

	executions, total, err := s.storage.ListExecutions(filters)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to list executions", err)
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"data":   executions,
		"total":  total,
		"offset": offset,
		"limit":  limit,
	})
}

func (s *APIServer) GetExecution(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	execution, err := s.storage.GetExecution(id)
	if err != nil {
		s.sendError(w, http.StatusNotFound, "Execution not found", err)
		return
	}

	s.sendJSON(w, http.StatusOK, execution)
}

func (s *APIServer) DeleteExecution(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := s.storage.DeleteExecution(id); err != nil {
		s.sendError(w, http.StatusNotFound, "Execution not found", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *APIServer) RetryExecution(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Get original execution
	execution, err := s.storage.GetExecution(id)
	if err != nil {
		s.sendError(w, http.StatusNotFound, "Execution not found", err)
		return
	}

	// Load workflow and re-execute
	workflow, err := s.storage.GetWorkflow(execution.WorkflowID)
	if err != nil {
		s.sendError(w, http.StatusNotFound, "Workflow not found", err)
		return
	}

	// Re-execute
	startTime := time.Now()
	result, err := s.engine.ExecuteWorkflow(workflow, execution.Data)

	newExecution := &model.WorkflowExecution{
		WorkflowID: execution.WorkflowID,
		StartedAt:  startTime,
		Mode:       "retry",
		Status:     "completed",
	}

	finishedAt := time.Now()
	newExecution.FinishedAt = &finishedAt

	if err != nil {
		newExecution.Status = "failed"
		newExecution.Error = err
	} else {
		newExecution.Data = result.Data
	}

	s.storage.SaveExecution(newExecution)

	s.sendJSON(w, http.StatusOK, newExecution)
}

func (s *APIServer) CancelExecution(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// TODO: Implement execution cancellation
	_ = id

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Execution cancelled",
		"status":  "cancelled",
	})
}

func (s *APIServer) GetMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := s.scheduler.GetMetrics()

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"scheduler": metrics,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// Credential handlers

func (s *APIServer) ListCredentials(w http.ResponseWriter, r *http.Request) {
	credentials, err := s.storage.ListCredentials()
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to list credentials", err)
		return
	}

	s.sendJSON(w, http.StatusOK, credentials)
}

func (s *APIServer) CreateCredential(w http.ResponseWriter, r *http.Request) {
	var credential storage.Credential
	if err := json.NewDecoder(r.Body).Decode(&credential); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	if err := s.storage.SaveCredential(&credential); err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to save credential", err)
		return
	}

	s.sendJSON(w, http.StatusCreated, credential)
}

func (s *APIServer) GetCredential(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	credential, err := s.storage.GetCredential(id)
	if err != nil {
		s.sendError(w, http.StatusNotFound, "Credential not found", err)
		return
	}

	s.sendJSON(w, http.StatusOK, credential)
}

func (s *APIServer) UpdateCredential(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var credential storage.Credential
	if err := json.NewDecoder(r.Body).Decode(&credential); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	if err := s.storage.UpdateCredential(id, &credential); err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to update credential", err)
		return
	}

	s.sendJSON(w, http.StatusOK, credential)
}

func (s *APIServer) DeleteCredential(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := s.storage.DeleteCredential(id); err != nil {
		s.sendError(w, http.StatusNotFound, "Credential not found", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Tag handlers

func (s *APIServer) ListTags(w http.ResponseWriter, r *http.Request) {
	tags, err := s.storage.ListTags()
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to list tags", err)
		return
	}

	s.sendJSON(w, http.StatusOK, tags)
}

func (s *APIServer) CreateTag(w http.ResponseWriter, r *http.Request) {
	var tag storage.Tag
	if err := json.NewDecoder(r.Body).Decode(&tag); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	if err := s.storage.SaveTag(&tag); err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to save tag", err)
		return
	}

	s.sendJSON(w, http.StatusCreated, tag)
}

func (s *APIServer) UpdateTag(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var tag storage.Tag
	if err := json.NewDecoder(r.Body).Decode(&tag); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	if err := s.storage.UpdateTag(id, &tag); err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to update tag", err)
		return
	}

	s.sendJSON(w, http.StatusOK, tag)
}

func (s *APIServer) DeleteTag(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := s.storage.DeleteTag(id); err != nil {
		s.sendError(w, http.StatusNotFound, "Tag not found", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
