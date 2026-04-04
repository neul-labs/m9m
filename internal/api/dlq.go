package api

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func (s *APIServer) ListDLQ(w http.ResponseWriter, r *http.Request) {
	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"data":  []interface{}{},
		"total": 0,
	})
}

func (s *APIServer) GetDLQItem(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	s.sendError(w, http.StatusNotFound, fmt.Sprintf("DLQ item '%s' not found", id), nil)
}

func (s *APIServer) RetryDLQItem(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"id":      id,
		"status":  "retrying",
		"message": "Item queued for retry",
	})
}

func (s *APIServer) DiscardDLQItem(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"id":      id,
		"status":  "discarded",
		"message": "Item discarded",
	})
}

func (s *APIServer) GetDLQStats(w http.ResponseWriter, r *http.Request) {
	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"totalItems": 0,
		"pending":    0,
		"retrying":   0,
		"resolved":   0,
		"discarded":  0,
	})
}
