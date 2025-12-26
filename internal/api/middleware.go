package api

import (
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/dipankar/m9m/internal/auth"
)

// CORSMiddleware handles Cross-Origin Resource Sharing
func CORSMiddleware(allowedOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers
			origin := r.Header.Get("Origin")
			if origin == "" {
				origin = allowedOrigin
			}

			// Allow specific origin or wildcard
			if allowedOrigin == "*" || origin == allowedOrigin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token, X-Requested-With")
			w.Header().Set("Access-Control-Expose-Headers", "Link, Location")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "3600")

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		wrappedWriter := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Call the next handler
		next.ServeHTTP(wrappedWriter, r)

		// Log the request
		duration := time.Since(start)
		log.Printf(
			"%s %s %d %s [%s]",
			r.Method,
			r.RequestURI,
			wrappedWriter.statusCode,
			duration,
			r.RemoteAddr,
		)
	})
}

// RecoveryMiddleware recovers from panics and returns a 500 error
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic
				log.Printf("PANIC: %v\n%s", err, debug.Stack())

				// Return 500 error
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{
					"error": true,
					"message": "Internal server error",
					"code": 500
				}`))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// responseWriter is a wrapper around http.ResponseWriter that captures the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code and calls the underlying WriteHeader
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// RateLimitMiddleware implements basic rate limiting (optional - can be enhanced)
func RateLimitMiddleware(requestsPerMinute int) func(http.Handler) http.Handler {
	type client struct {
		requests  int
		resetTime time.Time
	}

	clients := make(map[string]*client)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			now := time.Now()

			// Get or create client
			c, exists := clients[ip]
			if !exists {
				clients[ip] = &client{
					requests:  0,
					resetTime: now.Add(time.Minute),
				}
				c = clients[ip]
			}

			// Reset if time window has passed
			if now.After(c.resetTime) {
				c.requests = 0
				c.resetTime = now.Add(time.Minute)
			}

			// Check rate limit
			if c.requests >= requestsPerMinute {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{
					"error": true,
					"message": "Rate limit exceeded",
					"code": 429
				}`))
				return
			}

			c.requests++
			next.ServeHTTP(w, r)
		})
	}
}

// AuthMiddleware validates authentication (basic implementation - for backwards compatibility)
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health checks and public endpoints
		if isPublicEndpoint(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// For backwards compatibility, allow all requests when no auth manager is configured
		next.ServeHTTP(w, r)
	})
}

// AuthMiddlewareWithManager creates an authentication middleware with a configured AuthManager
func AuthMiddlewareWithManager(authManager *auth.AuthManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for public endpoints
			if isPublicEndpoint(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			var user *auth.User
			var authContext *AuthContext
			var err error
			var authMethod string

			// Try Bearer token first
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				user, err = authManager.GetCurrentUser(token)
				if err != nil {
					log.Printf("JWT validation failed: %v", err)
				} else {
					authMethod = "jwt"
				}
			}

			// Try API key if no valid JWT
			if user == nil {
				apiKey := r.Header.Get("X-API-Key")
				if apiKey == "" {
					// Also check Authorization header for API key
					if strings.HasPrefix(authHeader, "Api-Key ") {
						apiKey = strings.TrimPrefix(authHeader, "Api-Key ")
					}
				}

				if apiKey != "" {
					user, err = authManager.ValidateAPIKey(apiKey)
					if err != nil {
						log.Printf("API key validation failed: %v", err)
					} else {
						authMethod = "api_key"
					}
				}
			}

			// If no valid authentication found, return 401
			if user == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{
					"error": true,
					"message": "Unauthorized: valid authentication required",
					"code": 401
				}`))
				return
			}

			// Build permissions from role
			userRole := auth.UserRole(user.Role)
			var permissions []string
			if userRole == auth.RoleAdmin {
				permissions = []string{"admin"}
			} else {
				permissions = getPermissionsForRole(userRole)
			}

			// Create auth context
			authContext = &AuthContext{
				UserID:      user.ID,
				Permissions: permissions,
				AuthMethod:  authMethod,
			}

			// Add auth context to request
			ctx := ContextWithAuth(r.Context(), authContext)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// getPermissionsForRole returns the permissions for a given role
func getPermissionsForRole(role auth.UserRole) []string {
	switch role {
	case auth.RoleAdmin:
		return []string{"admin"}
	case auth.RoleMember:
		return []string{
			"workflow:read", "workflow:write", "workflow:execute",
			"execution:read", "credential:read", "credential:write",
		}
	case auth.RoleViewer:
		return []string{"workflow:read", "execution:read"}
	default:
		return []string{}
	}
}

// isPublicEndpoint checks if a path is a public endpoint that doesn't require authentication
func isPublicEndpoint(path string) bool {
	publicPaths := []string{
		"/health",
		"/healthz",
		"/ready",
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/auth/invite/accept",
		"/api/v1/auth/password/reset",
		"/api/v1/auth/password/reset/confirm",
	}

	for _, p := range publicPaths {
		if path == p {
			return true
		}
	}

	// Also allow webhook endpoints (they have their own authentication)
	if strings.HasPrefix(path, "/webhook/") || strings.HasPrefix(path, "/api/v1/webhooks/") {
		return true
	}

	return false
}

// RequireAuthMiddleware creates a middleware that requires authentication
func RequireAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCtx := AuthFromContext(r.Context())
		if authCtx == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{
				"error": true,
				"message": "Authentication required",
				"code": 401
			}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequirePermissionMiddleware creates a middleware that requires a specific permission
func RequirePermissionMiddleware(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx := AuthFromContext(r.Context())
			if authCtx == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{
					"error": true,
					"message": "Authentication required",
					"code": 401
				}`))
				return
			}

			// Check if user has the required permission
			if !authCtx.HasPermission(permission) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{
					"error": true,
					"message": "Permission denied: ` + permission + ` required",
					"code": 403
				}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdminMiddleware creates a middleware that requires admin role
func RequireAdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCtx := AuthFromContext(r.Context())
		if authCtx == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{
				"error": true,
				"message": "Authentication required",
				"code": 401
			}`))
			return
		}

		// Check if user has admin permission
		if !authCtx.HasPermission("admin") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{
				"error": true,
				"message": "Admin access required",
				"code": 403
			}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}
