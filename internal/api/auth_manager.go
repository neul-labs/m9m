package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// AuthManager handles authentication for the API
type AuthManager struct {
	jwtSecret    []byte
	apiKeys      map[string]*APIKey
	sessions     map[string]*Session
	keysMutex    sync.RWMutex
	sessionMutex sync.RWMutex
	tokenTTL     time.Duration
}

// APIKey represents an API key with metadata
type APIKey struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Key         string    `json:"key"`
	HashedKey   string    `json:"-"` // Never expose the actual key
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	LastUsed    time.Time `json:"last_used"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Active      bool      `json:"active"`
}

// Session represents a user session
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
}

// JWTClaims represents JWT token claims
type JWTClaims struct {
	UserID      string   `json:"user_id"`
	SessionID   string   `json:"session_id"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// AuthContext contains authentication information
type AuthContext struct {
	UserID      string
	SessionID   string
	APIKeyID    string
	Permissions []string
	AuthMethod  string // "jwt", "api_key"
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(jwtSecret string) *AuthManager {
	if jwtSecret == "" {
		// Generate a random secret if none provided
		secretBytes := make([]byte, 32)
		rand.Read(secretBytes)
		jwtSecret = hex.EncodeToString(secretBytes)
	}

	return &AuthManager{
		jwtSecret: []byte(jwtSecret),
		apiKeys:   make(map[string]*APIKey),
		sessions:  make(map[string]*Session),
		tokenTTL:  24 * time.Hour, // Default 24 hour token TTL
	}
}

// AuthenticationMiddleware validates incoming requests
func (am *AuthManager) AuthenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authContext, err := am.authenticateRequest(r)
		if err != nil {
			am.writeErrorResponse(w, http.StatusUnauthorized, "Authentication failed", err.Error())
			return
		}

		// Add auth context to request context
		ctx := r.Context()
		ctx = ContextWithAuth(ctx, authContext)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// authenticateRequest validates the request authentication
func (am *AuthManager) authenticateRequest(r *http.Request) (*AuthContext, error) {
	// Try JWT token first
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			return am.validateJWTToken(token)
		}
	}

	// Try API key authentication
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		return am.validateAPIKey(apiKey, r.RemoteAddr, r.UserAgent())
	}

	// Try API key from query parameter (less secure, but sometimes needed)
	if apiKey := r.URL.Query().Get("api_key"); apiKey != "" {
		return am.validateAPIKey(apiKey, r.RemoteAddr, r.UserAgent())
	}

	return nil, fmt.Errorf("no valid authentication provided")
}

// validateJWTToken validates a JWT token and returns auth context
func (am *AuthManager) validateJWTToken(tokenString string) (*AuthContext, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return am.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Validate session if present
	if claims.SessionID != "" {
		am.sessionMutex.RLock()
		session, exists := am.sessions[claims.SessionID]
		am.sessionMutex.RUnlock()

		if !exists {
			return nil, fmt.Errorf("session not found")
		}

		if time.Now().After(session.ExpiresAt) {
			// Clean up expired session
			am.sessionMutex.Lock()
			delete(am.sessions, claims.SessionID)
			am.sessionMutex.Unlock()
			return nil, fmt.Errorf("session expired")
		}
	}

	return &AuthContext{
		UserID:      claims.UserID,
		SessionID:   claims.SessionID,
		Permissions: claims.Permissions,
		AuthMethod:  "jwt",
	}, nil
}

// validateAPIKey validates an API key and returns auth context
func (am *AuthManager) validateAPIKey(key, ipAddress, userAgent string) (*AuthContext, error) {
	am.keysMutex.Lock()
	defer am.keysMutex.Unlock()

	for id, apiKey := range am.apiKeys {
		if !apiKey.Active {
			continue
		}

		// Check expiration
		if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
			continue
		}

		// Use constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(key), []byte(apiKey.Key)) == 1 {
			// Update last used timestamp
			apiKey.LastUsed = time.Now()

			return &AuthContext{
				APIKeyID:    id,
				Permissions: apiKey.Permissions,
				AuthMethod:  "api_key",
			}, nil
		}
	}

	return nil, fmt.Errorf("invalid API key")
}

// CreateAPIKey creates a new API key
func (am *AuthManager) CreateAPIKey(name string, permissions []string, expiresAt *time.Time) (*APIKey, error) {
	// Generate secure random key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	key := hex.EncodeToString(keyBytes)
	id := generateID()

	apiKey := &APIKey{
		ID:          id,
		Name:        name,
		Key:         key,
		Permissions: permissions,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
		Active:      true,
	}

	am.keysMutex.Lock()
	am.apiKeys[id] = apiKey
	am.keysMutex.Unlock()

	return apiKey, nil
}

// RevokeAPIKey revokes an API key
func (am *AuthManager) RevokeAPIKey(keyID string) error {
	am.keysMutex.Lock()
	defer am.keysMutex.Unlock()

	apiKey, exists := am.apiKeys[keyID]
	if !exists {
		return fmt.Errorf("API key not found")
	}

	apiKey.Active = false
	return nil
}

// ListAPIKeys returns all API keys (without the actual key values)
func (am *AuthManager) ListAPIKeys() []*APIKey {
	am.keysMutex.RLock()
	defer am.keysMutex.RUnlock()

	keys := make([]*APIKey, 0, len(am.apiKeys))
	for _, key := range am.apiKeys {
		// Create a copy without the actual key
		keyCopy := *key
		keyCopy.Key = "" // Never expose the actual key
		keys = append(keys, &keyCopy)
	}

	return keys
}

// CreateJWTToken creates a new JWT token
func (am *AuthManager) CreateJWTToken(userID string, permissions []string, sessionID string) (string, error) {
	claims := &JWTClaims{
		UserID:      userID,
		SessionID:   sessionID,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(am.tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "n8n-go",
			Subject:   userID,
			ID:        generateID(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(am.jwtSecret)
}

// CreateSession creates a new user session
func (am *AuthManager) CreateSession(userID, ipAddress, userAgent string) *Session {
	session := &Session{
		ID:        generateID(),
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(am.tokenTTL),
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}

	am.sessionMutex.Lock()
	am.sessions[session.ID] = session
	am.sessionMutex.Unlock()

	return session
}

// InvalidateSession removes a session
func (am *AuthManager) InvalidateSession(sessionID string) {
	am.sessionMutex.Lock()
	delete(am.sessions, sessionID)
	am.sessionMutex.Unlock()
}

// CleanupExpiredSessions removes expired sessions
func (am *AuthManager) CleanupExpiredSessions() {
	am.sessionMutex.Lock()
	defer am.sessionMutex.Unlock()

	now := time.Now()
	for id, session := range am.sessions {
		if now.After(session.ExpiresAt) {
			delete(am.sessions, id)
		}
	}
}

// RequirePermission middleware to check specific permissions
func (am *AuthManager) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authContext := AuthFromContext(r.Context())
			if authContext == nil {
				am.writeErrorResponse(w, http.StatusUnauthorized, "Authentication required", "No authentication context")
				return
			}

			// Check if user has required permission
			hasPermission := false
			for _, perm := range authContext.Permissions {
				if perm == permission || perm == "admin" { // admin has all permissions
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				am.writeErrorResponse(w, http.StatusForbidden, "Insufficient permissions", fmt.Sprintf("Required permission: %s", permission))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// writeErrorResponse writes a standard error response
func (am *AuthManager) writeErrorResponse(w http.ResponseWriter, statusCode int, message, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"detail":  detail,
			"code":    statusCode,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	WriteJSONResponse(w, errorResponse)
}

// generateID generates a random ID
func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// SetTokenTTL sets the JWT token time-to-live
func (am *AuthManager) SetTokenTTL(ttl time.Duration) {
	am.tokenTTL = ttl
}

// GetSessionCount returns the number of active sessions
func (am *AuthManager) GetSessionCount() int {
	am.sessionMutex.RLock()
	defer am.sessionMutex.RUnlock()
	return len(am.sessions)
}

// GetAPIKeyCount returns the number of active API keys
func (am *AuthManager) GetAPIKeyCount() int {
	am.keysMutex.RLock()
	defer am.keysMutex.RUnlock()

	count := 0
	for _, key := range am.apiKeys {
		if key.Active && (key.ExpiresAt == nil || time.Now().Before(*key.ExpiresAt)) {
			count++
		}
	}
	return count
}