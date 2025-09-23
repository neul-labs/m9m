package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/n8n-go/n8n-go/internal/engine"
	"github.com/n8n-go/n8n-go/internal/model"
	"github.com/n8n-go/n8n-go/internal/scheduler"
)

// WorkflowAPI provides RESTful API for workflow management
type WorkflowAPI struct {
	engine    *engine.WorkflowEngine
	scheduler *scheduler.WorkflowScheduler
	auth      *AuthManager
}

// WorkflowRequest represents a workflow creation/update request
type WorkflowRequest struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description,omitempty"`
	Nodes       []model.Node             `json:"nodes"`
	Connections map[string]model.NodeConnections `json:"connections"`
	Settings    map[string]interface{}   `json:"settings,omitempty"`
	Active      bool                     `json:"active"`
	Tags        []string                 `json:"tags,omitempty"`
}

// WorkflowResponse represents a workflow in API responses
type WorkflowResponse struct {
	ID          string                   `json:"id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description,omitempty"`
	Nodes       []model.Node             `json:"nodes"`
	Connections map[string]model.NodeConnections `json:"connections"`
	Settings    map[string]interface{}   `json:"settings,omitempty"`
	Active      bool                     `json:"active"`
	Tags        []string                 `json:"tags,omitempty"`
	CreatedAt   time.Time                `json:"createdAt"`
	UpdatedAt   time.Time                `json:"updatedAt"`
	CreatedBy   string                   `json:"createdBy,omitempty"`
}

// ExecutionRequest represents a workflow execution request
type ExecutionRequest struct {
	WorkflowID string             `json:"workflowId"`
	InputData  []model.DataItem   `json:"inputData,omitempty"`
	Mode       string             `json:"mode,omitempty"` // manual, trigger, test
}

// ExecutionResponse represents a workflow execution response
type ExecutionResponse struct {
	ID           string             `json:"id"`
	WorkflowID   string             `json:"workflowId"`
	Status       string             `json:"status"`
	StartTime    time.Time          `json:"startTime"`
	EndTime      *time.Time         `json:"endTime,omitempty"`
	Duration     *time.Duration     `json:"duration,omitempty"`
	Mode         string             `json:"mode"`
	InputData    []model.DataItem   `json:"inputData,omitempty"`
	OutputData   []model.DataItem   `json:"outputData,omitempty"`
	Error        string             `json:"error,omitempty"`
	NodesExecuted int               `json:"nodesExecuted"`
}

// ListResponse represents a paginated list response
type ListResponse struct {
	Items  interface{} `json:"items"`
	Total  int         `json:"total"`
	Offset int         `json:"offset"`
	Limit  int         `json:"limit"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// NewWorkflowAPI creates a new workflow API instance
func NewWorkflowAPI(engine *engine.WorkflowEngine, scheduler *scheduler.WorkflowScheduler) *WorkflowAPI {
	return &WorkflowAPI{
		engine:    engine,
		scheduler: scheduler,
		auth:      NewAuthManager(),
	}
}

// RegisterRoutes registers all workflow API routes
func (api *WorkflowAPI) RegisterRoutes(router *mux.Router) {
	// Apply authentication middleware
	authRouter := router.PathPrefix("/api/v1").Subrouter()
	authRouter.Use(api.auth.AuthenticationMiddleware)

	// Workflow management
	authRouter.HandleFunc("/workflows", api.ListWorkflows).Methods("GET")
	authRouter.HandleFunc("/workflows", api.CreateWorkflow).Methods("POST")
	authRouter.HandleFunc("/workflows/{id}", api.GetWorkflow).Methods("GET")
	authRouter.HandleFunc("/workflows/{id}", api.UpdateWorkflow).Methods("PUT")
	authRouter.HandleFunc("/workflows/{id}", api.DeleteWorkflow).Methods("DELETE")
	authRouter.HandleFunc("/workflows/{id}/activate", api.ActivateWorkflow).Methods("POST")
	authRouter.HandleFunc("/workflows/{id}/deactivate", api.DeactivateWorkflow).Methods("POST")

	// Workflow execution
	authRouter.HandleFunc("/workflows/{id}/execute", api.ExecuteWorkflow).Methods("POST")
	authRouter.HandleFunc("/workflows/{id}/test", api.TestWorkflow).Methods("POST")
	authRouter.HandleFunc("/executions", api.ListExecutions).Methods("GET")
	authRouter.HandleFunc("/executions/{id}", api.GetExecution).Methods("GET")
	authRouter.HandleFunc("/executions/{id}", api.DeleteExecution).Methods("DELETE")
	authRouter.HandleFunc("/executions/{id}/retry", api.RetryExecution).Methods("POST")
	authRouter.HandleFunc("/executions/{id}/cancel", api.CancelExecution).Methods("POST")

	// Scheduling
	authRouter.HandleFunc("/schedules", api.ListSchedules).Methods("GET")
	authRouter.HandleFunc("/schedules", api.CreateSchedule).Methods("POST")
	authRouter.HandleFunc("/schedules/{id}", api.GetSchedule).Methods("GET")
	authRouter.HandleFunc("/schedules/{id}", api.UpdateSchedule).Methods("PUT")
	authRouter.HandleFunc("/schedules/{id}", api.DeleteSchedule).Methods("DELETE")
	authRouter.HandleFunc("/schedules/{id}/history", api.GetScheduleHistory).Methods("GET")

	// System information
	authRouter.HandleFunc("/health", api.HealthCheck).Methods("GET")
	authRouter.HandleFunc("/metrics", api.GetMetrics).Methods("GET")
	authRouter.HandleFunc("/version", api.GetVersion).Methods("GET")
}

// ListWorkflows handles GET /api/v1/workflows
func (api *WorkflowAPI) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	active := r.URL.Query().Get("active")
	search := r.URL.Query().Get("search")
	tags := r.URL.Query().Get("tags")

	// TODO: Implement actual workflow storage and retrieval
	// This is a placeholder implementation
	workflows := []WorkflowResponse{
		{
			ID:        "workflow_1",
			Name:      "Sample Workflow",
			Active:    true,
			CreatedAt: time.Now().AddDate(0, 0, -1),
			UpdatedAt: time.Now(),
		},
	}

	// Apply filters (placeholder implementation)
	var filteredWorkflows []WorkflowResponse
	for _, workflow := range workflows {
		// Filter by active status
		if active != "" {
			isActive, _ := strconv.ParseBool(active)
			if workflow.Active != isActive {
				continue
			}
		}

		// Filter by search term
		if search != "" && !strings.Contains(strings.ToLower(workflow.Name), strings.ToLower(search)) {
			continue
		}

		// Filter by tags
		if tags != "" {
			tagList := strings.Split(tags, ",")
			hasTag := false
			for _, tag := range tagList {
				for _, workflowTag := range workflow.Tags {
					if strings.EqualFold(tag, workflowTag) {
						hasTag = true
						break
					}
				}
				if hasTag {
					break
				}
			}
			if !hasTag {
				continue
			}
		}

		filteredWorkflows = append(filteredWorkflows, workflow)
	}

	// Apply pagination
	total := len(filteredWorkflows)
	end := offset + limit
	if end > total {
		end = total
	}
	if offset >= total {
		filteredWorkflows = []WorkflowResponse{}
	} else {
		filteredWorkflows = filteredWorkflows[offset:end]
	}

	response := ListResponse{
		Items:  filteredWorkflows,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}

	api.sendJSON(w, http.StatusOK, response)
}

// CreateWorkflow handles POST /api/v1/workflows
func (api *WorkflowAPI) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var req WorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	// Validate request
	if err := api.validateWorkflowRequest(&req); err != nil {
		api.sendError(w, http.StatusBadRequest, "Validation failed", err)
		return
	}

	// Create workflow
	workflow := &model.Workflow{
		Name:        req.Name,
		Nodes:       req.Nodes,
		Connections: req.Connections,
		Settings:    req.Settings,
		Active:      req.Active,
	}

	// TODO: Save workflow to storage and generate ID
	workflowID := fmt.Sprintf("workflow_%d", time.Now().UnixNano())

	response := WorkflowResponse{
		ID:          workflowID,
		Name:        req.Name,
		Description: req.Description,
		Nodes:       req.Nodes,
		Connections: req.Connections,
		Settings:    req.Settings,
		Active:      req.Active,
		Tags:        req.Tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   api.getUserFromContext(r),
	}

	api.sendJSON(w, http.StatusCreated, response)
}

// GetWorkflow handles GET /api/v1/workflows/{id}
func (api *WorkflowAPI) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]

	// TODO: Load workflow from storage
	// This is a placeholder implementation
	if workflowID == "workflow_1" {
		response := WorkflowResponse{
			ID:        workflowID,
			Name:      "Sample Workflow",
			Active:    true,
			CreatedAt: time.Now().AddDate(0, 0, -1),
			UpdatedAt: time.Now(),
		}
		api.sendJSON(w, http.StatusOK, response)
		return
	}

	api.sendError(w, http.StatusNotFound, "Workflow not found", nil)
}

// UpdateWorkflow handles PUT /api/v1/workflows/{id}
func (api *WorkflowAPI) UpdateWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]

	var req WorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	// Validate request
	if err := api.validateWorkflowRequest(&req); err != nil {
		api.sendError(w, http.StatusBadRequest, "Validation failed", err)
		return
	}

	// TODO: Update workflow in storage
	response := WorkflowResponse{
		ID:          workflowID,
		Name:        req.Name,
		Description: req.Description,
		Nodes:       req.Nodes,
		Connections: req.Connections,
		Settings:    req.Settings,
		Active:      req.Active,
		Tags:        req.Tags,
		UpdatedAt:   time.Now(),
		CreatedBy:   api.getUserFromContext(r),
	}

	api.sendJSON(w, http.StatusOK, response)
}

// DeleteWorkflow handles DELETE /api/v1/workflows/{id}
func (api *WorkflowAPI) DeleteWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]

	// TODO: Delete workflow from storage
	_ = workflowID

	w.WriteHeader(http.StatusNoContent)
}

// ExecuteWorkflow handles POST /api/v1/workflows/{id}/execute
func (api *WorkflowAPI) ExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]

	var req ExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	// TODO: Load workflow and execute
	// This is a placeholder implementation
	executionID := fmt.Sprintf("exec_%d", time.Now().UnixNano())
	startTime := time.Now()

	response := ExecutionResponse{
		ID:           executionID,
		WorkflowID:   workflowID,
		Status:       "completed",
		StartTime:    startTime,
		EndTime:      &[]time.Time{startTime.Add(time.Second)}[0],
		Duration:     &[]time.Duration{time.Second}[0],
		Mode:         "manual",
		InputData:    req.InputData,
		OutputData:   []model.DataItem{{JSON: map[string]interface{}{"result": "success"}}},
		NodesExecuted: 3,
	}

	api.sendJSON(w, http.StatusOK, response)
}

// TestWorkflow handles POST /api/v1/workflows/{id}/test
func (api *WorkflowAPI) TestWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]

	var req ExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	// TODO: Test workflow execution
	// This is a placeholder implementation
	executionID := fmt.Sprintf("test_%d", time.Now().UnixNano())
	startTime := time.Now()

	response := ExecutionResponse{
		ID:           executionID,
		WorkflowID:   workflowID,
		Status:       "completed",
		StartTime:    startTime,
		EndTime:      &[]time.Time{startTime.Add(500 * time.Millisecond)}[0],
		Duration:     &[]time.Duration{500 * time.Millisecond}[0],
		Mode:         "test",
		InputData:    req.InputData,
		OutputData:   []model.DataItem{{JSON: map[string]interface{}{"test": "passed"}}},
		NodesExecuted: 3,
	}

	api.sendJSON(w, http.StatusOK, response)
}

// ActivateWorkflow handles POST /api/v1/workflows/{id}/activate
func (api *WorkflowAPI) ActivateWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]

	// TODO: Activate workflow
	_ = workflowID

	api.sendJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Workflow activated successfully",
		"active":  true,
	})
}

// DeactivateWorkflow handles POST /api/v1/workflows/{id}/deactivate
func (api *WorkflowAPI) DeactivateWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]

	// TODO: Deactivate workflow
	_ = workflowID

	api.sendJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Workflow deactivated successfully",
		"active":  false,
	})
}

// ListExecutions handles GET /api/v1/executions
func (api *WorkflowAPI) ListExecutions(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 20
	}

	workflowID := r.URL.Query().Get("workflowId")
	status := r.URL.Query().Get("status")

	// TODO: Load executions from storage
	// This is a placeholder implementation
	executions := []ExecutionResponse{
		{
			ID:         "exec_1",
			WorkflowID: "workflow_1",
			Status:     "completed",
			StartTime:  time.Now().Add(-time.Hour),
			Mode:       "manual",
		},
	}

	response := ListResponse{
		Items:  executions,
		Total:  len(executions),
		Offset: offset,
		Limit:  limit,
	}

	api.sendJSON(w, http.StatusOK, response)
}

// GetExecution handles GET /api/v1/executions/{id}
func (api *WorkflowAPI) GetExecution(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	executionID := vars["id"]

	// TODO: Load execution from storage
	if executionID == "exec_1" {
		response := ExecutionResponse{
			ID:         executionID,
			WorkflowID: "workflow_1",
			Status:     "completed",
			StartTime:  time.Now().Add(-time.Hour),
			Mode:       "manual",
		}
		api.sendJSON(w, http.StatusOK, response)
		return
	}

	api.sendError(w, http.StatusNotFound, "Execution not found", nil)
}

// ListSchedules handles GET /api/v1/schedules
func (api *WorkflowAPI) ListSchedules(w http.ResponseWriter, r *http.Request) {
	schedules := api.scheduler.ListSchedules()

	response := ListResponse{
		Items:  schedules,
		Total:  len(schedules),
		Offset: 0,
		Limit:  len(schedules),
	}

	api.sendJSON(w, http.StatusOK, response)
}

// CreateSchedule handles POST /api/v1/schedules
func (api *WorkflowAPI) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	var config scheduler.ScheduleConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		api.sendError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	if err := api.scheduler.AddSchedule(&config); err != nil {
		api.sendError(w, http.StatusBadRequest, "Failed to create schedule", err)
		return
	}

	api.sendJSON(w, http.StatusCreated, config)
}

// GetSchedule handles GET /api/v1/schedules/{id}
func (api *WorkflowAPI) GetSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scheduleID := vars["id"]

	schedule, err := api.scheduler.GetSchedule(scheduleID)
	if err != nil {
		api.sendError(w, http.StatusNotFound, "Schedule not found", err)
		return
	}

	api.sendJSON(w, http.StatusOK, schedule)
}

// UpdateSchedule handles PUT /api/v1/schedules/{id}
func (api *WorkflowAPI) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scheduleID := vars["id"]

	var updates scheduler.ScheduleConfig
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		api.sendError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	if err := api.scheduler.UpdateSchedule(scheduleID, &updates); err != nil {
		api.sendError(w, http.StatusBadRequest, "Failed to update schedule", err)
		return
	}

	schedule, _ := api.scheduler.GetSchedule(scheduleID)
	api.sendJSON(w, http.StatusOK, schedule)
}

// DeleteSchedule handles DELETE /api/v1/schedules/{id}
func (api *WorkflowAPI) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scheduleID := vars["id"]

	if err := api.scheduler.RemoveSchedule(scheduleID); err != nil {
		api.sendError(w, http.StatusNotFound, "Schedule not found", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetScheduleHistory handles GET /api/v1/schedules/{id}/history
func (api *WorkflowAPI) GetScheduleHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scheduleID := vars["id"]

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 50
	}

	history, err := api.scheduler.GetExecutionHistory(scheduleID, limit)
	if err != nil {
		api.sendError(w, http.StatusNotFound, "Schedule history not found", err)
		return
	}

	api.sendJSON(w, http.StatusOK, history)
}

// HealthCheck handles GET /api/v1/health
func (api *WorkflowAPI) HealthCheck(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "0.2.0",
		"uptime":    time.Since(time.Now().Add(-time.Hour)).String(),
		"services": map[string]string{
			"workflow_engine": "healthy",
			"scheduler":       "healthy",
			"api":            "healthy",
		},
	}

	api.sendJSON(w, http.StatusOK, health)
}

// GetMetrics handles GET /api/v1/metrics
func (api *WorkflowAPI) GetMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := map[string]interface{}{
		"scheduler": api.scheduler.GetMetrics(),
		"system": map[string]interface{}{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"uptime":    time.Since(time.Now().Add(-time.Hour)).String(),
		},
	}

	api.sendJSON(w, http.StatusOK, metrics)
}

// GetVersion handles GET /api/v1/version
func (api *WorkflowAPI) GetVersion(w http.ResponseWriter, r *http.Request) {
	version := map[string]interface{}{
		"version":    "0.2.0",
		"buildDate":  "2024-01-22",
		"gitCommit":  "abc123",
		"platform":   "n8n-go",
		"apiVersion": "v1",
	}

	api.sendJSON(w, http.StatusOK, version)
}

// Helper methods

func (api *WorkflowAPI) validateWorkflowRequest(req *WorkflowRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(req.Nodes) == 0 {
		return fmt.Errorf("at least one node is required")
	}
	return nil
}

func (api *WorkflowAPI) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// WriteJSONResponse is a helper function used by auth manager
func WriteJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (api *WorkflowAPI) sendError(w http.ResponseWriter, status int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errorResp := ErrorResponse{
		Error:   message,
		Code:    status,
		Message: message,
	}

	if err != nil {
		errorResp.Details = err.Error()
	}

	json.NewEncoder(w).Encode(errorResp)
}

func (api *WorkflowAPI) getUserFromContext(r *http.Request) string {
	// TODO: Extract user from authentication context
	return "api_user"
}

// Helper functions for missing methods in the execution handlers
func (api *WorkflowAPI) DeleteExecution(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	executionID := vars["id"]

	// TODO: Delete execution from storage
	_ = executionID

	w.WriteHeader(http.StatusNoContent)
}

func (api *WorkflowAPI) RetryExecution(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	executionID := vars["id"]

	// TODO: Retry execution
	_ = executionID

	api.sendJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Execution retry initiated",
		"status":  "pending",
	})
}

func (api *WorkflowAPI) CancelExecution(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	executionID := vars["id"]

	// TODO: Cancel execution
	_ = executionID

	api.sendJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Execution cancelled",
		"status":  "cancelled",
	})
}