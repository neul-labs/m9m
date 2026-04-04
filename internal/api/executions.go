package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/neul-labs/m9m/internal/engine"
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/queue"
	"github.com/neul-labs/m9m/internal/storage"
)

func (s *APIServer) ExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	workflow, err := s.storage.GetWorkflow(id)
	if err != nil {
		s.sendError(w, http.StatusNotFound, "Workflow not found", err)
		return
	}

	inputData, err := parseExecutionInputData(r)
	if err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid execution input payload", err)
		return
	}

	startTime := time.Now()
	execution := &model.WorkflowExecution{
		ID:         fmt.Sprintf("exec_%d", startTime.UnixNano()),
		WorkflowID: id,
		StartedAt:  startTime,
		Mode:       "manual",
		Status:     "running",
	}

	if err := s.storage.SaveExecution(execution); err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to save execution", err)
		return
	}

	execCtx, cancel := context.WithCancel(r.Context())
	s.trackExecutionCancel(execution.ID, cancel)
	defer func() {
		cancel()
		s.untrackExecutionCancel(execution.ID)
	}()

	result, execErr := engine.ExecuteWorkflowWithContext(execCtx, s.engine, workflow, inputData)
	executionErr := engine.ResolveExecutionError(result, execErr)

	finishedAt := time.Now()
	execution.FinishedAt = &finishedAt

	if executionErr != nil {
		if errors.Is(executionErr, context.Canceled) {
			execution.Status = "cancelled"
		} else {
			execution.Status = "failed"
		}
		execution.Error = executionErr
	} else {
		execution.Status = "completed"
		execution.Data = result.Data
	}

	if err := s.storage.SaveExecution(execution); err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to update execution", err)
		return
	}

	s.sendJSON(w, http.StatusOK, execution)
}

// ExecuteWorkflowByDefinition executes an inline workflow definition payload.
func (s *APIServer) ExecuteWorkflowByDefinition(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Workflow  *model.Workflow  `json:"workflow"`
		InputData []model.DataItem `json:"inputData"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}
	if request.Workflow == nil {
		s.sendError(w, http.StatusBadRequest, "workflow is required", nil)
		return
	}
	if err := validateWorkflow(request.Workflow); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid workflow", err)
		return
	}

	inputData := request.InputData
	if len(inputData) == 0 {
		inputData = defaultExecutionInputData()
	}

	result, execErr := engine.ExecuteWorkflowWithContext(r.Context(), s.engine, request.Workflow, inputData)
	executionErr := engine.ResolveExecutionError(result, execErr)
	if executionErr != nil {
		s.sendJSON(w, http.StatusOK, map[string]interface{}{
			"data":  []model.DataItem{},
			"error": executionErr.Error(),
		})
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"data": result.Data,
	})
}

func (s *APIServer) ExecuteWorkflowAsync(w http.ResponseWriter, r *http.Request) {
	if s.jobQueue == nil {
		s.sendError(w, http.StatusServiceUnavailable, "Job queue not available", nil)
		return
	}

	id := mux.Vars(r)["id"]

	workflow, err := s.storage.GetWorkflow(id)
	if err != nil {
		s.sendError(w, http.StatusNotFound, "Workflow not found", err)
		return
	}

	inputData, err := parseExecutionInputData(r)
	if err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid execution input payload", err)
		return
	}

	jobID := fmt.Sprintf("job_%d", time.Now().UnixNano())
	job := &queue.Job{
		ID:         jobID,
		WorkflowID: id,
		Workflow:   workflow,
		InputData:  inputData,
		Priority:   0,
		MaxRetries: 3,
		CreatedAt:  time.Now(),
	}

	if err := s.jobQueue.Enqueue(job); err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to enqueue job", err)
		return
	}

	s.sendJSON(w, http.StatusAccepted, map[string]interface{}{
		"jobId":      jobID,
		"workflowId": id,
		"status":     "pending",
		"message":    "Workflow execution queued",
	})
}

func (s *APIServer) ListJobs(w http.ResponseWriter, r *http.Request) {
	if s.jobQueue == nil {
		s.sendError(w, http.StatusServiceUnavailable, "Job queue not available", nil)
		return
	}

	var status *queue.JobStatus
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		value := queue.JobStatus(statusStr)
		status = &value
	}

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	jobs, err := s.jobQueue.ListJobs(status, limit)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to list jobs", err)
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"jobs":    jobs,
		"count":   len(jobs),
		"pending": s.jobQueue.GetPendingCount(),
		"running": s.jobQueue.GetRunningCount(),
	})
}

func (s *APIServer) GetJob(w http.ResponseWriter, r *http.Request) {
	if s.jobQueue == nil {
		s.sendError(w, http.StatusServiceUnavailable, "Job queue not available", nil)
		return
	}

	id := mux.Vars(r)["id"]
	job, err := s.jobQueue.GetJob(id)
	if err != nil {
		s.sendError(w, http.StatusNotFound, "Job not found", err)
		return
	}

	s.sendJSON(w, http.StatusOK, job)
}

func (s *APIServer) CreateExecution(w http.ResponseWriter, r *http.Request) {
	var execution model.WorkflowExecution
	if err := json.NewDecoder(r.Body).Decode(&execution); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	if execution.WorkflowID == "" {
		s.sendError(w, http.StatusBadRequest, "workflowId is required", nil)
		return
	}
	if execution.Status == "" {
		execution.Status = "running"
	}
	if execution.Mode == "" {
		execution.Mode = "manual"
	}
	if execution.StartedAt.IsZero() {
		execution.StartedAt = time.Now()
	}

	if err := s.storage.SaveExecution(&execution); err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to save execution", err)
		return
	}

	s.sendJSON(w, http.StatusCreated, execution)
}

func (s *APIServer) ListExecutions(w http.ResponseWriter, r *http.Request) {
	offset := parseIntParam(r.URL.Query().Get("offset"), 0, 0)
	limit := parseIntParam(r.URL.Query().Get("limit"), 20, s.config.MaxPaginationLimit)

	filters := storage.ExecutionFilters{
		WorkflowID: r.URL.Query().Get("workflowId"),
		Status:     r.URL.Query().Get("status"),
		Offset:     offset,
		Limit:      limit,
	}

	executions, total, err := s.storage.ListExecutions(filters)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to list executions", err)
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"data":       executions,
		"executions": executions,
		"total":      total,
		"offset":     offset,
		"limit":      limit,
	})
}

func (s *APIServer) GetExecution(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	execution, err := s.storage.GetExecution(id)
	if err != nil {
		s.sendError(w, http.StatusNotFound, "Execution not found", err)
		return
	}

	s.sendJSON(w, http.StatusOK, execution)
}

func (s *APIServer) DeleteExecution(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	if err := s.storage.DeleteExecution(id); err != nil {
		s.sendError(w, http.StatusNotFound, "Execution not found", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *APIServer) RetryExecution(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	execution, err := s.storage.GetExecution(id)
	if err != nil {
		s.sendError(w, http.StatusNotFound, "Execution not found", err)
		return
	}

	workflow, err := s.storage.GetWorkflow(execution.WorkflowID)
	if err != nil {
		s.sendError(w, http.StatusNotFound, "Workflow not found", err)
		return
	}

	startTime := time.Now()
	newExecution := &model.WorkflowExecution{
		ID:         fmt.Sprintf("exec_%d", startTime.UnixNano()),
		WorkflowID: execution.WorkflowID,
		StartedAt:  startTime,
		Mode:       "retry",
		Status:     "running",
	}

	if err := s.storage.SaveExecution(newExecution); err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to save execution", err)
		return
	}

	execCtx, cancel := context.WithCancel(r.Context())
	s.trackExecutionCancel(newExecution.ID, cancel)
	defer func() {
		cancel()
		s.untrackExecutionCancel(newExecution.ID)
	}()

	result, execErr := engine.ExecuteWorkflowWithContext(execCtx, s.engine, workflow, execution.Data)
	executionErr := engine.ResolveExecutionError(result, execErr)

	finishedAt := time.Now()
	newExecution.FinishedAt = &finishedAt

	if executionErr != nil {
		if errors.Is(executionErr, context.Canceled) {
			newExecution.Status = "cancelled"
		} else {
			newExecution.Status = "failed"
		}
		newExecution.Error = executionErr
	} else {
		newExecution.Status = "completed"
		newExecution.Data = result.Data
	}

	if err := s.storage.SaveExecution(newExecution); err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to update execution", err)
		return
	}

	s.sendJSON(w, http.StatusOK, newExecution)
}

func (s *APIServer) CancelExecution(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	execution, err := s.storage.GetExecution(id)
	if err != nil {
		s.sendError(w, http.StatusNotFound, "Execution not found", err)
		return
	}

	if execution.Status != "running" {
		s.sendError(w, http.StatusBadRequest, "Execution is not running", nil)
		return
	}

	cancel, exists := s.getExecutionCancel(id)
	if !exists {
		s.sendJSON(w, http.StatusConflict, map[string]interface{}{
			"error":       true,
			"message":     "Execution is running but cancellation is not supported by this runtime",
			"executionId": id,
			"status":      "running",
		})
		return
	}

	cancel()
	s.sendJSON(w, http.StatusAccepted, map[string]interface{}{
		"message":     "Cancellation requested",
		"executionId": id,
		"status":      "cancel_requested",
	})
}
