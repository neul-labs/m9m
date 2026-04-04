package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/neul-labs/m9m/internal/storage"
)

// CredentialResponse represents a safe credential response without sensitive data.
type CredentialResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

func (s *APIServer) ListCredentials(w http.ResponseWriter, r *http.Request) {
	credentials, err := s.storage.ListCredentials()
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to list credentials", err)
		return
	}

	safeCredentials := make([]CredentialResponse, 0, len(credentials))
	for _, cred := range credentials {
		safeCredentials = append(safeCredentials, CredentialResponse{
			ID:        cred.ID,
			Name:      cred.Name,
			Type:      cred.Type,
			CreatedAt: cred.CreatedAt,
			UpdatedAt: cred.UpdatedAt,
		})
	}

	s.sendJSON(w, http.StatusOK, safeCredentials)
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
	id := mux.Vars(r)["id"]

	credential, err := s.storage.GetCredential(id)
	if err != nil {
		s.sendError(w, http.StatusNotFound, "Credential not found", err)
		return
	}

	s.sendJSON(w, http.StatusOK, CredentialResponse{
		ID:        credential.ID,
		Name:      credential.Name,
		Type:      credential.Type,
		CreatedAt: credential.CreatedAt,
		UpdatedAt: credential.UpdatedAt,
	})
}

func (s *APIServer) UpdateCredential(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

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
	id := mux.Vars(r)["id"]

	if err := s.storage.DeleteCredential(id); err != nil {
		s.sendError(w, http.StatusNotFound, "Credential not found", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
