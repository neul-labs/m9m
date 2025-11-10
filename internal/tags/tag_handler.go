package tags

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// TagHandler handles tag HTTP requests
type TagHandler struct {
	tagManager *TagManager
}

// NewTagHandler creates a new tag handler
func NewTagHandler(tagManager *TagManager) *TagHandler {
	return &TagHandler{
		tagManager: tagManager,
	}
}

// RegisterRoutes registers tag routes
func (h *TagHandler) RegisterRoutes(router *mux.Router) {
	api := router.PathPrefix("/api/v1").Subrouter()

	// Tag endpoints
	api.HandleFunc("/tags", h.ListTags).Methods("GET", "OPTIONS")
	api.HandleFunc("/tags", h.CreateTag).Methods("POST", "OPTIONS")
	api.HandleFunc("/tags/{id}", h.GetTag).Methods("GET", "OPTIONS")
	api.HandleFunc("/tags/{id}", h.UpdateTag).Methods("PATCH", "PUT", "OPTIONS")
	api.HandleFunc("/tags/{id}", h.DeleteTag).Methods("DELETE", "OPTIONS")

	// Workflow tag endpoints
	api.HandleFunc("/workflows/{id}/tags", h.GetWorkflowTags).Methods("GET", "OPTIONS")
	api.HandleFunc("/workflows/{id}/tags", h.SetWorkflowTags).Methods("POST", "PUT", "OPTIONS")
	api.HandleFunc("/workflows/{id}/tags/{tagId}", h.AddWorkflowTag).Methods("POST", "OPTIONS")
	api.HandleFunc("/workflows/{id}/tags/{tagId}", h.RemoveWorkflowTag).Methods("DELETE", "OPTIONS")
}

// ListTags lists all tags
func (h *TagHandler) ListTags(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	search := r.URL.Query().Get("search")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit == 0 {
		limit = 50 // Default limit
	}

	filters := TagListFilters{
		Search: search,
		Limit:  limit,
		Offset: offset,
	}

	tags, total, err := h.tagManager.ListTags(filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":   tags,
		"total":  total,
		"count":  len(tags),
		"limit":  limit,
		"offset": offset,
	})
}

// CreateTag creates a new tag
func (h *TagHandler) CreateTag(w http.ResponseWriter, r *http.Request) {
	var request TagCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tag, err := h.tagManager.CreateTag(&request)
	if err != nil {
		if err == ErrTagNameRequired || err == ErrTagNameTooLong || err == ErrTagNameExists {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tag)
}

// GetTag retrieves a specific tag
func (h *TagHandler) GetTag(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tagID := vars["id"]

	tag, err := h.tagManager.GetTag(tagID)
	if err != nil {
		if err == ErrTagNotFound {
			http.Error(w, "Tag not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tag)
}

// UpdateTag updates a tag
func (h *TagHandler) UpdateTag(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tagID := vars["id"]

	var request TagUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tag, err := h.tagManager.UpdateTag(tagID, &request)
	if err != nil {
		if err == ErrTagNotFound {
			http.Error(w, "Tag not found", http.StatusNotFound)
		} else if err == ErrTagNameTooLong || err == ErrTagNameExists {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tag)
}

// DeleteTag deletes a tag
func (h *TagHandler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tagID := vars["id"]

	err := h.tagManager.DeleteTag(tagID)
	if err != nil {
		if err == ErrTagNotFound {
			http.Error(w, "Tag not found", http.StatusNotFound)
		} else if err == ErrTagInUse {
			http.Error(w, err.Error(), http.StatusConflict)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetWorkflowTags gets all tags for a workflow
func (h *TagHandler) GetWorkflowTags(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]

	tags, err := h.tagManager.GetWorkflowTags(workflowID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  tags,
		"count": len(tags),
	})
}

// SetWorkflowTags sets all tags for a workflow
func (h *TagHandler) SetWorkflowTags(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]

	var request WorkflowTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.tagManager.SetWorkflowTags(workflowID, request.TagIDs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return updated tags
	tags, err := h.tagManager.GetWorkflowTags(workflowID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    tags,
		"count":   len(tags),
	})
}

// AddWorkflowTag adds a tag to a workflow
func (h *TagHandler) AddWorkflowTag(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]
	tagID := vars["tagId"]

	if err := h.tagManager.AddWorkflowTag(workflowID, tagID); err != nil {
		if err == ErrTagNotFound {
			http.Error(w, "Tag not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Tag added to workflow",
	})
}

// RemoveWorkflowTag removes a tag from a workflow
func (h *TagHandler) RemoveWorkflowTag(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]
	tagID := vars["tagId"]

	if err := h.tagManager.RemoveWorkflowTag(workflowID, tagID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
