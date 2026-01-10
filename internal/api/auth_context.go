package api

import (
	"context"
)

// Context keys for storing authentication information
type contextKey string

const (
	authContextKey contextKey = "auth_context"
)

// ContextWithAuth adds authentication context to a context
func ContextWithAuth(ctx context.Context, auth *AuthContext) context.Context {
	return context.WithValue(ctx, authContextKey, auth)
}

// AuthFromContext retrieves authentication context from a context
func AuthFromContext(ctx context.Context) *AuthContext {
	auth, ok := ctx.Value(authContextKey).(*AuthContext)
	if !ok {
		return nil
	}
	return auth
}

// RequireAuthFromContext retrieves authentication context and panics if not found
// This should only be used in handlers that are protected by authentication middleware
func RequireAuthFromContext(ctx context.Context) *AuthContext {
	auth := AuthFromContext(ctx)
	if auth == nil {
		panic("authentication context not found - ensure authentication middleware is applied")
	}
	return auth
}

// HasPermission checks if the authentication context has a specific permission
func (ac *AuthContext) HasPermission(permission string) bool {
	if ac == nil {
		return false
	}

	for _, perm := range ac.Permissions {
		if perm == permission || perm == "admin" {
			return true
		}
	}
	return false
}

// IsAPIKey returns true if the authentication was done via API key
func (ac *AuthContext) IsAPIKey() bool {
	return ac != nil && ac.AuthMethod == "api_key"
}

// IsJWT returns true if the authentication was done via JWT token
func (ac *AuthContext) IsJWT() bool {
	return ac != nil && ac.AuthMethod == "jwt"
}

// GetUserID returns the user ID if available
func (ac *AuthContext) GetUserID() string {
	if ac == nil {
		return ""
	}
	return ac.UserID
}

// GetIdentifier returns a unique identifier for the authenticated entity
func (ac *AuthContext) GetIdentifier() string {
	if ac == nil {
		return "anonymous"
	}

	if ac.UserID != "" {
		return ac.UserID
	}

	if ac.APIKeyID != "" {
		return "api_key:" + ac.APIKeyID
	}

	return "unknown"
}
