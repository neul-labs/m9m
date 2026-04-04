package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/neul-labs/m9m/internal/storage"
)

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
	id := mux.Vars(r)["id"]

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
	id := mux.Vars(r)["id"]

	if err := s.storage.DeleteTag(id); err != nil {
		s.sendError(w, http.StatusNotFound, "Tag not found", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
