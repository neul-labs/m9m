package webhooks

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// Handler handles incoming webhook HTTP requests
type Handler struct {
	manager *WebhookManager
}

// NewHandler creates a new webhook handler
func NewHandler(manager *WebhookManager) *Handler {
	return &Handler{
		manager: manager,
	}
}

// RegisterRoutes registers webhook routes on the router
func (h *Handler) RegisterRoutes(router *mux.Router) {
	// Test webhooks (for workflow testing)
	router.HandleFunc("/api/v1/webhooks/test/{path:.*}", h.HandleTestWebhook).Methods("GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS")

	// Production webhooks
	router.HandleFunc("/webhook/{path:.*}", h.HandleProductionWebhook).Methods("GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS")
	router.HandleFunc("/webhook-test/{path:.*}", h.HandleTestWebhook).Methods("GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS")

	// Webhook management API
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/webhooks", h.ListWebhooks).Methods("GET", "OPTIONS")
	api.HandleFunc("/webhooks", h.CreateWebhook).Methods("POST", "OPTIONS")
	api.HandleFunc("/webhooks/{id}", h.GetWebhook).Methods("GET", "OPTIONS")
	api.HandleFunc("/webhooks/{id}", h.DeleteWebhook).Methods("DELETE", "OPTIONS")
}

// HandleProductionWebhook handles production webhook requests
func (h *Handler) HandleProductionWebhook(w http.ResponseWriter, r *http.Request) {
	h.handleWebhookRequest(w, r, false)
}

// HandleTestWebhook handles test webhook requests
func (h *Handler) HandleTestWebhook(w http.ResponseWriter, r *http.Request) {
	h.handleWebhookRequest(w, r, true)
}

// handleWebhookRequest processes a webhook request (test or production)
func (h *Handler) handleWebhookRequest(w http.ResponseWriter, r *http.Request, isTest bool) {
	// Extract path from URL
	vars := mux.Vars(r)
	path := "/" + vars["path"]

	log.Printf("📥 Webhook request: %s %s (test=%v)", r.Method, path, isTest)

	// Find webhook
	webhook, err := h.manager.GetWebhookByPath(path, r.Method, isTest)
	if err != nil {
		log.Printf("⚠️  Webhook not found: %s %s (test=%v)", r.Method, path, isTest)
		http.Error(w, fmt.Sprintf("Webhook not found: %s", path), http.StatusNotFound)
		return
	}

	// Authenticate request
	if err := h.authenticateRequest(r, webhook); err != nil {
		log.Printf("⚠️  Authentication failed for webhook %s: %v", webhook.ID, err)
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	// Parse request
	webhookRequest, err := h.parseRequest(r)
	if err != nil {
		log.Printf("⚠️  Failed to parse webhook request: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Execute webhook
	response, err := h.manager.ExecuteWebhook(webhook, webhookRequest)
	if err != nil {
		log.Printf("⚠️  Webhook execution failed: %v", err)
		http.Error(w, fmt.Sprintf("Webhook execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Send response
	h.sendResponse(w, response)

	log.Printf("✅ Webhook executed successfully: %s %s", r.Method, path)
}

// ListWebhooks lists all webhooks
func (h *Handler) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	workflowID := r.URL.Query().Get("workflowId")

	webhooks, err := h.manager.ListWebhooks(workflowID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  webhooks,
		"count": len(webhooks),
	})
}

// CreateWebhook creates a new webhook
func (h *Handler) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	var webhook Webhook
	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.manager.RegisterWebhook(&webhook); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(webhook)
}

// GetWebhook retrieves a webhook by ID
func (h *Handler) GetWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	webhookID := vars["id"]

	webhook, err := h.manager.GetWebhook(webhookID)
	if err != nil {
		http.Error(w, "Webhook not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(webhook)
}

// DeleteWebhook deletes a webhook
func (h *Handler) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	webhookID := vars["id"]

	if err := h.manager.UnregisterWebhook(webhookID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper methods

func (h *Handler) parseRequest(r *http.Request) (*WebhookRequest, error) {
	request := &WebhookRequest{
		Method:  r.Method,
		Path:    r.URL.Path,
		Headers: r.Header,
		Query:   r.URL.Query(),
	}

	// Parse body based on content type
	if r.Method != "GET" && r.Method != "DELETE" {
		contentType := r.Header.Get("Content-Type")

		if strings.Contains(contentType, "application/json") {
			var body interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err != io.EOF {
				return nil, fmt.Errorf("failed to parse JSON body: %w", err)
			}
			request.Body = body
		} else if strings.Contains(contentType, "application/x-www-form-urlencoded") {
			if err := r.ParseForm(); err != nil {
				return nil, fmt.Errorf("failed to parse form data: %w", err)
			}
			formData := make(map[string]interface{})
			for key, values := range r.PostForm {
				if len(values) == 1 {
					formData[key] = values[0]
				} else {
					formData[key] = values
				}
			}
			request.Body = formData
		} else {
			// Read raw body
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read body: %w", err)
			}
			request.Body = string(bodyBytes)
		}
	}

	return request, nil
}

func (h *Handler) authenticateRequest(r *http.Request, webhook *Webhook) error {
	switch webhook.AuthType {
	case "none", "":
		return nil

	case "basic":
		// Basic authentication
		username, password, ok := r.BasicAuth()
		if !ok {
			return fmt.Errorf("basic auth required")
		}

		expectedUsername := getAuthData(webhook.AuthData, "username", "")
		expectedPassword := getAuthData(webhook.AuthData, "password", "")

		if username != expectedUsername || password != expectedPassword {
			return fmt.Errorf("invalid credentials")
		}

	case "apiKey":
		// API key authentication
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			apiKey = r.URL.Query().Get("apiKey")
		}

		expectedKey := getAuthData(webhook.AuthData, "apiKey", "")
		if apiKey != expectedKey {
			return fmt.Errorf("invalid API key")
		}

	case "header":
		// Custom header authentication
		headerName := getAuthData(webhook.AuthData, "headerName", "Authorization")
		headerValue := getAuthData(webhook.AuthData, "headerValue", "")

		if r.Header.Get(headerName) != headerValue {
			return fmt.Errorf("invalid header authentication")
		}

	default:
		return fmt.Errorf("unsupported auth type: %s", webhook.AuthType)
	}

	return nil
}

func (h *Handler) sendResponse(w http.ResponseWriter, response *WebhookResponse) {
	// Set headers
	for key, value := range response.Headers {
		w.Header().Set(key, value)
	}

	// Set status code
	w.WriteHeader(response.StatusCode)

	// Write body
	if response.Body != nil {
		switch body := response.Body.(type) {
		case string:
			w.Write([]byte(body))
		case []byte:
			w.Write(body)
		default:
			json.NewEncoder(w).Encode(body)
		}
	}
}

func getAuthData(authData map[string]interface{}, key, defaultValue string) string {
	if authData == nil {
		return defaultValue
	}
	if val, ok := authData[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}
