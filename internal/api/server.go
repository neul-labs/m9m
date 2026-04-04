package api

import (
	"context"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/neul-labs/m9m/internal/engine"
	"github.com/neul-labs/m9m/internal/queue"
	"github.com/neul-labs/m9m/internal/scheduler"
	"github.com/neul-labs/m9m/internal/storage"
)

// APIServerConfig configures the API server
type APIServerConfig struct {
	// AllowedOrigins for WebSocket CORS (comma-separated in env)
	AllowedOrigins []string
	// DevMode enables permissive security settings
	DevMode bool
	// MaxPaginationLimit caps the number of items per page
	MaxPaginationLimit int
}

// DefaultAPIServerConfig returns default configuration from environment
func DefaultAPIServerConfig() *APIServerConfig {
	config := &APIServerConfig{
		DevMode:            os.Getenv("M9M_DEV_MODE") == "true",
		MaxPaginationLimit: 100,
	}

	// Parse allowed origins from environment
	originsEnv := os.Getenv("M9M_ALLOWED_ORIGINS")
	if originsEnv != "" {
		config.AllowedOrigins = strings.Split(originsEnv, ",")
		for i := range config.AllowedOrigins {
			config.AllowedOrigins[i] = strings.TrimSpace(config.AllowedOrigins[i])
		}
	}

	return config
}

// APIServer provides REST API with workflow compatibility
type APIServer struct {
	engine    engine.WorkflowEngine
	scheduler *scheduler.WorkflowScheduler
	storage   storage.WorkflowStorage
	jobQueue  queue.JobQueue
	upgrader  websocket.Upgrader
	wsClients map[string]*websocket.Conn
	config    *APIServerConfig

	executionMu      sync.RWMutex
	executionCancels map[string]context.CancelFunc
}

// NewAPIServer creates a new API server instance
func NewAPIServer(eng engine.WorkflowEngine, scheduler *scheduler.WorkflowScheduler, storage storage.WorkflowStorage) *APIServer {
	return NewAPIServerWithConfig(eng, scheduler, storage, DefaultAPIServerConfig())
}

// NewAPIServerWithConfig creates a new API server with custom configuration
func NewAPIServerWithConfig(eng engine.WorkflowEngine, scheduler *scheduler.WorkflowScheduler, storage storage.WorkflowStorage, config *APIServerConfig) *APIServer {
	if config == nil {
		config = DefaultAPIServerConfig()
	}

	server := &APIServer{
		engine:    eng,
		scheduler: scheduler,
		storage:   storage,
		wsClients: make(map[string]*websocket.Conn),
		config:    config,

		executionCancels: make(map[string]context.CancelFunc),
	}

	// Configure WebSocket upgrader with proper CORS
	server.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// In dev mode, allow all origins
			if config.DevMode {
				return true
			}

			// Check if origin is in allowed list
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true // Same-origin requests have no Origin header
			}

			for _, allowed := range config.AllowedOrigins {
				if origin == allowed {
					return true
				}
			}
			return false
		},
	}

	return server
}

// SetJobQueue sets the job queue for async execution
func (s *APIServer) SetJobQueue(jq queue.JobQueue) {
	s.jobQueue = jq
}
