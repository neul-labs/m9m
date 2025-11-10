package variables

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// VariableHandler handles variable HTTP requests
type VariableHandler struct {
	variableManager *VariableManager
}

// NewVariableHandler creates a new variable handler
func NewVariableHandler(variableManager *VariableManager) *VariableHandler {
	return &VariableHandler{
		variableManager: variableManager,
	}
}

// RegisterRoutes registers variable routes
func (h *VariableHandler) RegisterRoutes(router *mux.Router) {
	api := router.PathPrefix("/api/v1").Subrouter()

	// Variable endpoints
	api.HandleFunc("/variables", h.ListVariables).Methods("GET", "OPTIONS")
	api.HandleFunc("/variables", h.CreateVariable).Methods("POST", "OPTIONS")
	api.HandleFunc("/variables/{id}", h.GetVariable).Methods("GET", "OPTIONS")
	api.HandleFunc("/variables/{id}", h.UpdateVariable).Methods("PUT", "PATCH", "OPTIONS")
	api.HandleFunc("/variables/{id}", h.DeleteVariable).Methods("DELETE", "OPTIONS")

	// Environment endpoints
	api.HandleFunc("/environments", h.ListEnvironments).Methods("GET", "OPTIONS")
	api.HandleFunc("/environments", h.CreateEnvironment).Methods("POST", "OPTIONS")
	api.HandleFunc("/environments/{id}", h.GetEnvironment).Methods("GET", "OPTIONS")
	api.HandleFunc("/environments/{id}", h.UpdateEnvironment).Methods("PUT", "PATCH", "OPTIONS")
	api.HandleFunc("/environments/{id}", h.DeleteEnvironment).Methods("DELETE", "OPTIONS")

	// Workflow variables endpoints
	api.HandleFunc("/workflows/{id}/variables", h.GetWorkflowVariables).Methods("GET", "OPTIONS")
	api.HandleFunc("/workflows/{id}/variables", h.SaveWorkflowVariables).Methods("POST", "PUT", "OPTIONS")
}

// ListVariables lists all variables
func (h *VariableHandler) ListVariables(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	typeStr := r.URL.Query().Get("type")
	search := r.URL.Query().Get("search")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit == 0 {
		limit = 50 // Default limit
	}

	filters := VariableListFilters{
		Type:   VariableType(typeStr),
		Search: search,
		Limit:  limit,
		Offset: offset,
	}

	variables, total, err := h.variableManager.ListVariables(filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":   variables,
		"total":  total,
		"count":  len(variables),
		"limit":  limit,
		"offset": offset,
	})
}

// CreateVariable creates a new variable
func (h *VariableHandler) CreateVariable(w http.ResponseWriter, r *http.Request) {
	var request VariableCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Default to global if not specified
	if request.Type == "" {
		request.Type = GlobalVariable
	}

	variable, err := h.variableManager.CreateVariable(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(variable)
}

// GetVariable retrieves a specific variable
func (h *VariableHandler) GetVariable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	variableID := vars["id"]

	// Check if we should decrypt
	decrypt := r.URL.Query().Get("decrypt") == "true"

	variable, err := h.variableManager.GetVariable(variableID, decrypt)
	if err != nil {
		http.Error(w, "Variable not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(variable)
}

// UpdateVariable updates a variable
func (h *VariableHandler) UpdateVariable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	variableID := vars["id"]

	var request VariableUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	variable, err := h.variableManager.UpdateVariable(variableID, &request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(variable)
}

// DeleteVariable deletes a variable
func (h *VariableHandler) DeleteVariable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	variableID := vars["id"]

	if err := h.variableManager.DeleteVariable(variableID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListEnvironments lists all environments
func (h *VariableHandler) ListEnvironments(w http.ResponseWriter, r *http.Request) {
	environments, err := h.variableManager.ListEnvironments()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  environments,
		"total": len(environments),
		"count": len(environments),
	})
}

// CreateEnvironment creates a new environment
func (h *VariableHandler) CreateEnvironment(w http.ResponseWriter, r *http.Request) {
	var request EnvironmentCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	environment, err := h.variableManager.CreateEnvironment(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(environment)
}

// GetEnvironment retrieves a specific environment
func (h *VariableHandler) GetEnvironment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	environmentID := vars["id"]

	environment, err := h.variableManager.GetEnvironment(environmentID)
	if err != nil {
		http.Error(w, "Environment not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(environment)
}

// UpdateEnvironment updates an environment
func (h *VariableHandler) UpdateEnvironment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	environmentID := vars["id"]

	var request EnvironmentUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	environment, err := h.variableManager.UpdateEnvironment(environmentID, &request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(environment)
}

// DeleteEnvironment deletes an environment
func (h *VariableHandler) DeleteEnvironment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	environmentID := vars["id"]

	if err := h.variableManager.DeleteEnvironment(environmentID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetWorkflowVariables retrieves variables for a workflow
func (h *VariableHandler) GetWorkflowVariables(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]

	variables, err := h.variableManager.GetWorkflowVariables(workflowID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workflowId": workflowID,
		"variables":  variables,
	})
}

// SaveWorkflowVariables saves variables for a workflow
func (h *VariableHandler) SaveWorkflowVariables(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]

	var request map[string]string
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.variableManager.SaveWorkflowVariables(workflowID, request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"workflowId": workflowID,
		"variables":  request,
	})
}
