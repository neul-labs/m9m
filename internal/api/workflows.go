package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/storage"
)

func (s *APIServer) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	active := r.URL.Query().Get("active")
	search := r.URL.Query().Get("search")
	offsetStr := r.URL.Query().Get("offset")
	limitStr := r.URL.Query().Get("limit")

	offset := parseIntParam(offsetStr, 0, 0)
	limit := parseIntParam(limitStr, 20, s.config.MaxPaginationLimit)

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
		"data":      workflows,
		"workflows": workflows,
		"total":     total,
		"offset":    offset,
		"limit":     limit,
	})
}

func (s *APIServer) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var workflow model.Workflow
	if err := json.NewDecoder(r.Body).Decode(&workflow); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	if err := validateWorkflow(&workflow); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid workflow", err)
		return
	}

	if err := s.storage.SaveWorkflow(&workflow); err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to save workflow", err)
		return
	}

	s.sendJSON(w, http.StatusCreated, workflow)
}

func validateWorkflow(workflow *model.Workflow) error {
	if workflow.Name == "" {
		return fmt.Errorf("workflow name is required")
	}
	if len(workflow.Name) > 255 {
		return fmt.Errorf("workflow name too long (max 255 characters)")
	}

	for i, node := range workflow.Nodes {
		if node.Type == "" {
			return fmt.Errorf("node %d: type is required", i)
		}
		if node.Name == "" {
			return fmt.Errorf("node %d: name is required", i)
		}
		if len(node.Name) > 255 {
			return fmt.Errorf("node %d: name too long (max 255 characters)", i)
		}
	}

	return nil
}

func (s *APIServer) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	workflow, err := s.storage.GetWorkflow(id)
	if err != nil {
		s.sendError(w, http.StatusNotFound, "Workflow not found", err)
		return
	}

	s.sendJSON(w, http.StatusOK, workflow)
}

func (s *APIServer) UpdateWorkflow(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

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
	id := mux.Vars(r)["id"]

	if err := s.storage.DeleteWorkflow(id); err != nil {
		s.sendError(w, http.StatusNotFound, "Workflow not found", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *APIServer) ActivateWorkflow(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

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
	id := mux.Vars(r)["id"]

	if err := s.storage.DeactivateWorkflow(id); err != nil {
		s.sendError(w, http.StatusNotFound, "Workflow not found", err)
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Workflow deactivated",
		"active":  false,
	})
}

func (s *APIServer) DuplicateWorkflow(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	original, err := s.storage.GetWorkflow(id)
	if err != nil {
		s.sendError(w, http.StatusNotFound, "Workflow not found", err)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}

	duplicate := &model.Workflow{
		Name:        original.Name + " (Copy)",
		Active:      false,
		Nodes:       make([]model.Node, len(original.Nodes)),
		Connections: make(map[string]model.Connections),
		Settings:    original.Settings,
		Tags:        original.Tags,
	}

	if req.Name != "" {
		duplicate.Name = req.Name
	}

	for i, node := range original.Nodes {
		duplicate.Nodes[i] = model.Node{
			ID:          fmt.Sprintf("node-%d", time.Now().UnixNano()+int64(i)),
			Name:        node.Name,
			Type:        node.Type,
			TypeVersion: node.TypeVersion,
			Position:    node.Position,
			Parameters:  node.Parameters,
			Credentials: node.Credentials,
			Disabled:    node.Disabled,
		}
	}

	for key, conn := range original.Connections {
		duplicate.Connections[key] = conn
	}

	if err := s.storage.SaveWorkflow(duplicate); err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to save duplicate workflow", err)
		return
	}

	s.sendJSON(w, http.StatusCreated, duplicate)
}
