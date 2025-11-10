package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// AuthHandler handles authentication HTTP requests
type AuthHandler struct {
	authManager *AuthManager
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(authManager *AuthManager) *AuthHandler {
	return &AuthHandler{
		authManager: authManager,
	}
}

// RegisterRoutes registers authentication routes
func (h *AuthHandler) RegisterRoutes(router *mux.Router) {
	api := router.PathPrefix("/api/v1").Subrouter()

	// Public endpoints (no auth required)
	api.HandleFunc("/login", h.Login).Methods("POST", "OPTIONS")
	api.HandleFunc("/register", h.Register).Methods("POST", "OPTIONS")
	api.HandleFunc("/invites/accept", h.AcceptInvite).Methods("POST", "OPTIONS")

	// Authenticated endpoints (require JWT)
	api.HandleFunc("/logout", h.AuthMiddleware(h.Logout)).Methods("POST", "OPTIONS")
	api.HandleFunc("/me", h.AuthMiddleware(h.GetCurrentUser)).Methods("GET", "OPTIONS")
	api.HandleFunc("/me", h.AuthMiddleware(h.UpdateCurrentUser)).Methods("PATCH", "OPTIONS")

	// User management (admin only)
	api.HandleFunc("/users", h.AuthMiddleware(h.ListUsers)).Methods("GET", "OPTIONS")
	api.HandleFunc("/users", h.AuthMiddleware(h.CreateUser)).Methods("POST", "OPTIONS")
	api.HandleFunc("/users/{id}", h.AuthMiddleware(h.GetUser)).Methods("GET", "OPTIONS")
	api.HandleFunc("/users/{id}", h.AuthMiddleware(h.UpdateUser)).Methods("PATCH", "OPTIONS")
	api.HandleFunc("/users/{id}", h.AuthMiddleware(h.DeleteUser)).Methods("DELETE", "OPTIONS")

	// Invitations (admin/member)
	api.HandleFunc("/invites", h.AuthMiddleware(h.InviteUser)).Methods("POST", "OPTIONS")

	// API keys
	api.HandleFunc("/api-keys", h.AuthMiddleware(h.ListAPIKeys)).Methods("GET", "OPTIONS")
	api.HandleFunc("/api-keys", h.AuthMiddleware(h.CreateAPIKey)).Methods("POST", "OPTIONS")
	api.HandleFunc("/api-keys/{id}", h.AuthMiddleware(h.DeleteAPIKey)).Methods("DELETE", "OPTIONS")
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var credentials UserCredentials
	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	token, user, err := h.authManager.Login(&credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var registration UserRegistration
	if err := json.NewDecoder(r.Body).Decode(&registration); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.authManager.Register(&registration)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// Logout handles user logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "No token provided", http.StatusUnauthorized)
		return
	}

	if err := h.authManager.Logout(token); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetCurrentUser returns the current authenticated user
func (h *AuthHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// UpdateCurrentUser updates the current user's information
func (h *AuthHandler) UpdateCurrentUser(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	var update UserUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.authManager.UpdateUser(user.ID, &update, user.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get updated user
	updatedUser, err := h.authManager.userStorage.GetUser(user.ID)
	if err != nil {
		http.Error(w, "Failed to fetch updated user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedUser.Sanitize())
}

// ListUsers lists all users (admin only)
func (h *AuthHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	if user.Role != string(RoleAdmin) {
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
		return
	}

	users, err := h.authManager.userStorage.ListUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Sanitize all users
	sanitized := make([]*User, len(users))
	for i, u := range users {
		sanitized[i] = u.Sanitize()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  sanitized,
		"count": len(sanitized),
	})
}

// CreateUser creates a new user (admin only)
func (h *AuthHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	if user.Role != string(RoleAdmin) {
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
		return
	}

	var registration UserRegistration
	if err := json.NewDecoder(r.Body).Decode(&registration); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	newUser, err := h.authManager.Register(&registration)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newUser)
}

// GetUser gets a user by ID (admin only)
func (h *AuthHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	currentUser := r.Context().Value("user").(*User)

	if currentUser.Role != string(RoleAdmin) {
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	userID := vars["id"]

	user, err := h.authManager.userStorage.GetUser(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user.Sanitize())
}

// UpdateUser updates a user (admin only)
func (h *AuthHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	currentUser := r.Context().Value("user").(*User)

	vars := mux.Vars(r)
	userID := vars["id"]

	var update UserUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.authManager.UpdateUser(userID, &update, currentUser.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get updated user
	updatedUser, err := h.authManager.userStorage.GetUser(userID)
	if err != nil {
		http.Error(w, "Failed to fetch updated user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedUser.Sanitize())
}

// DeleteUser deletes a user (admin only)
func (h *AuthHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	currentUser := r.Context().Value("user").(*User)

	vars := mux.Vars(r)
	userID := vars["id"]

	if err := h.authManager.DeleteUser(userID, currentUser.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// InviteUser creates an invitation (admin/member)
func (h *AuthHandler) InviteUser(w http.ResponseWriter, r *http.Request) {
	currentUser := r.Context().Value("user").(*User)

	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	invite, err := h.authManager.InviteUser(req.Email, req.Role, currentUser.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(invite)
}

// AcceptInvite accepts an invitation
func (h *AuthHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.authManager.AcceptInvite(req.Token, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// CreateAPIKey creates a new API key
func (h *AuthHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	var req struct {
		Name      string     `json:"name"`
		ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	apiKey, plainKey, err := h.authManager.CreateAPIKey(user.ID, req.Name, req.ExpiresAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"apiKey": apiKey,
		"key":    plainKey, // Only shown once!
	})
}

// ListAPIKeys lists user's API keys
func (h *AuthHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	keys, err := h.authManager.userStorage.ListUserAPIKeys(user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  keys,
		"count": len(keys),
	})
}

// DeleteAPIKey deletes an API key
func (h *AuthHandler) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	keyID := vars["id"]

	if err := h.authManager.userStorage.DeleteAPIKey(keyID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AuthMiddleware is a middleware that validates JWT tokens
func (h *AuthHandler) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			http.Error(w, "No authorization token provided", http.StatusUnauthorized)
			return
		}

		user, err := h.authManager.GetCurrentUser(token)
		if err != nil {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Add user to context
		ctx := r.Context()
		ctx = context.WithValue(ctx, "user", user)
		r = r.WithContext(ctx)

		next(w, r)
	}
}

// extractToken extracts JWT token from Authorization header or query parameter
func extractToken(r *http.Request) string {
	// Try Authorization header first
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// Format: "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}

	// Try query parameter as fallback
	return r.URL.Query().Get("token")
}
