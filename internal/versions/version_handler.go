package versions

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/dipankar/m9m/internal/api"
	"github.com/gorilla/mux"
)

// VersionHandler handles version HTTP requests
type VersionHandler struct {
	versionManager *VersionManager
}

// NewVersionHandler creates a new version handler
func NewVersionHandler(versionManager *VersionManager) *VersionHandler {
	return &VersionHandler{
		versionManager: versionManager,
	}
}

// RegisterRoutes registers version routes
func (h *VersionHandler) RegisterRoutes(router *mux.Router) {
	api := router.PathPrefix("/api/v1").Subrouter()

	// Version endpoints
	api.HandleFunc("/workflows/{id}/versions", h.ListVersions).Methods("GET", "OPTIONS")
	api.HandleFunc("/workflows/{id}/versions", h.CreateVersion).Methods("POST", "OPTIONS")
	api.HandleFunc("/workflows/{id}/versions/{versionId}", h.GetVersion).Methods("GET", "OPTIONS")
	api.HandleFunc("/workflows/{id}/versions/{versionId}", h.DeleteVersion).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/workflows/{id}/versions/{versionId}/restore", h.RestoreVersion).Methods("POST", "OPTIONS")
	api.HandleFunc("/workflows/{id}/versions/compare", h.CompareVersions).Methods("GET", "OPTIONS")
}

// ListVersions lists all versions for a workflow
func (h *VersionHandler) ListVersions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]

	// Parse query parameters
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit == 0 {
		limit = 50 // Default limit
	}

	filters := VersionListFilters{
		WorkflowID: workflowID,
		Limit:      limit,
		Offset:     offset,
	}

	versions, total, err := h.versionManager.ListVersions(filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":   versions,
		"total":  total,
		"count":  len(versions),
		"limit":  limit,
		"offset": offset,
	})
}

// CreateVersion creates a new version for a workflow
func (h *VersionHandler) CreateVersion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]

	var request VersionCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get user from auth context
	author := "system"
	if authCtx := api.AuthFromContext(r.Context()); authCtx != nil {
		author = authCtx.UserID
	}

	version, err := h.versionManager.CreateVersion(workflowID, author, &request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(version)
}

// GetVersion retrieves a specific version
func (h *VersionHandler) GetVersion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	versionID := vars["versionId"]

	version, err := h.versionManager.GetVersion(versionID)
	if err != nil {
		http.Error(w, "Version not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(version)
}

// DeleteVersion deletes a version
func (h *VersionHandler) DeleteVersion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	versionID := vars["versionId"]

	if err := h.versionManager.DeleteVersion(versionID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RestoreVersion restores a workflow to a specific version
func (h *VersionHandler) RestoreVersion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	versionID := vars["versionId"]

	var request VersionRestoreRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		// Use default values if no body provided
		request.CreateBackup = true
	}

	// Get user from auth context
	author := "system"
	if authCtx := api.AuthFromContext(r.Context()); authCtx != nil {
		author = authCtx.UserID
	}

	version, err := h.versionManager.RestoreVersion(versionID, author, &request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":        "Workflow restored successfully",
		"restoredFrom":   version,
		"backupCreated":  request.CreateBackup,
	})
}

// CompareVersions compares two versions
func (h *VersionHandler) CompareVersions(w http.ResponseWriter, r *http.Request) {
	fromVersionID := r.URL.Query().Get("from")
	toVersionID := r.URL.Query().Get("to")

	if fromVersionID == "" || toVersionID == "" {
		http.Error(w, "Both 'from' and 'to' query parameters are required", http.StatusBadRequest)
		return
	}

	comparison, err := h.versionManager.CompareVersions(fromVersionID, toVersionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comparison)
}
