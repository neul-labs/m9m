package commands

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/cobra"

	"github.com/neul-labs/m9m/internal/api"
	"github.com/neul-labs/m9m/internal/credentials"
	"github.com/neul-labs/m9m/internal/engine"
	"github.com/neul-labs/m9m/internal/queue"
	"github.com/neul-labs/m9m/internal/scheduler"
	"github.com/neul-labs/m9m/internal/storage"
	"github.com/neul-labs/m9m/internal/web"
	"github.com/neul-labs/m9m/internal/workspace"
)

var (
	servePort        int
	serveHost        string
	serveMetricsPort int
	serveDevMode     bool
	serveDB          string
	servePostgres    string
	serveQueueType   string
	serveQueueDB     string
	serveWorkers     int
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the m9m server",
	Long: `Start the m9m server with REST API and optional web UI.

This runs m9m as a full server, similar to n8n, with:
- REST API for workflow management
- Web UI (if built)
- Webhook handling
- Scheduled workflow execution

Examples:
  m9m serve                      Start on default port 8080
  m9m serve --port 3000          Start on custom port
  m9m serve --dev                Start in development mode
  m9m serve --db ./data/m9m.db   Use specific database`,
	Run: runServe,
}

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 8080, "Server port")
	serveCmd.Flags().StringVar(&serveHost, "host", "0.0.0.0", "Server host")
	serveCmd.Flags().IntVar(&serveMetricsPort, "metrics-port", 0, "Metrics port (0 = disabled)")
	serveCmd.Flags().BoolVar(&serveDevMode, "dev", false, "Enable development mode (permissive CORS)")
	serveCmd.Flags().StringVar(&serveDB, "db", "", "SQLite database path")
	serveCmd.Flags().StringVar(&servePostgres, "postgres", "", "PostgreSQL connection URL")
	serveCmd.Flags().StringVar(&serveQueueType, "queue", "sqlite", "Queue type: memory, sqlite")
	serveCmd.Flags().StringVar(&serveQueueDB, "queue-db", "", "Queue SQLite database path (for sqlite queue)")
	serveCmd.Flags().IntVar(&serveWorkers, "workers", 4, "Number of worker threads for job processing")
}

func runServe(cmd *cobra.Command, args []string) {
	logger := log.New(os.Stderr, "[m9m] ", log.LstdFlags)

	logger.Printf("Starting m9m server v%s", version)

	// Initialize storage
	var store storage.WorkflowStorage
	var err error

	if servePostgres != "" {
		logger.Printf("Using PostgreSQL storage")
		store, err = storage.NewPostgresStorage(servePostgres)
	} else {
		// Use SQLite
		dbPath := serveDB
		if dbPath == "" {
			// Check workspace first
			if workspaceFlag != "" {
				mgr, _ := workspace.NewManager()
				if mgr != nil {
					dbPath, _ = mgr.GetStoragePath(workspaceFlag)
				}
			}
			// Default to data directory
			if dbPath == "" {
				homeDir, _ := os.UserHomeDir()
				dataDir := filepath.Join(homeDir, ".m9m", "data")
				os.MkdirAll(dataDir, 0755)
				dbPath = filepath.Join(dataDir, "m9m.db")
			}
		}
		logger.Printf("Using SQLite storage: %s", dbPath)
		store, err = storage.NewSQLiteStorage(dbPath)
	}

	if err != nil {
		logger.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Initialize engine
	eng := engine.NewWorkflowEngine()
	RegisterAllNodes(eng)

	// Initialize credential manager
	credMgr, err := credentials.NewCredentialManager()
	if err != nil {
		logger.Printf("Warning: Failed to initialize credential manager: %v", err)
	} else {
		eng.SetCredentialManager(credMgr)
	}

	// Initialize job queue
	var jobQueue queue.JobQueue
	switch serveQueueType {
	case "memory":
		logger.Println("Using in-memory job queue (jobs will be lost on restart)")
		jobQueue = queue.NewMemoryJobQueue(1000)
	case "sqlite":
		queueDBPath := serveQueueDB
		if queueDBPath == "" {
			// Default to data directory
			homeDir, _ := os.UserHomeDir()
			dataDir := filepath.Join(homeDir, ".m9m", "data")
			os.MkdirAll(dataDir, 0755)
			queueDBPath = filepath.Join(dataDir, "queue.db")
		}
		logger.Printf("Using SQLite job queue: %s", queueDBPath)
		var queueErr error
		jobQueue, queueErr = queue.NewSQLiteJobQueue(queueDBPath, 1000)
		if queueErr != nil {
			logger.Fatalf("Failed to initialize job queue: %v", queueErr)
		}
	default:
		logger.Fatalf("Unknown queue type: %s (supported: memory, sqlite)", serveQueueType)
	}
	defer jobQueue.Close()

	// Initialize worker pool
	workerPool := queue.NewWorkerPool(jobQueue, eng, serveWorkers)
	workerPool.Start()
	defer workerPool.Stop()
	logger.Printf("Started %d workers for job processing", serveWorkers)

	// Initialize scheduler
	sched := scheduler.NewWorkflowScheduler(eng)
	sched.Start()
	defer sched.Stop()

	// Create API server
	apiConfig := &api.APIServerConfig{
		DevMode:            serveDevMode,
		MaxPaginationLimit: 100,
	}

	if serveDevMode {
		apiConfig.AllowedOrigins = []string{"*"}
		logger.Println("Development mode enabled (permissive CORS)")
	}

	apiServer := api.NewAPIServerWithConfig(eng, sched, store, apiConfig)
	apiServer.SetJobQueue(jobQueue)

	// Setup router
	router := mux.NewRouter()

	// Register API routes
	apiServer.RegisterRoutes(router)

	// Serve embedded web UI
	// In dev mode, you can pass a path to serve from filesystem for hot-reload
	var webDevPath string
	if serveDevMode {
		// Check if there's a local web/dist folder for development
		if _, err := os.Stat("web/dist"); err == nil {
			webDevPath = "web/dist"
		}
	}
	webHandler := web.NewHandler(webDevPath)
	webHandler.RegisterRoutes(router)

	logger.Printf("Web UI enabled (embedded: %v)", webDevPath == "")

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", serveHost, servePort)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start metrics server if enabled
	if serveMetricsPort > 0 {
		go startMetricsServer(serveMetricsPort, logger)
	}

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		logger.Println("Shutting down server...")

		shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 30*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Printf("Error during shutdown: %v", err)
		}

		cancel()
	}()

	// Start server
	nodeTypes := eng.GetRegisteredNodeTypes()
	logger.Printf("Server listening on http://%s", addr)
	logger.Printf("Registered nodes: %d", len(nodeTypes))
	logger.Printf("Queue: %s | Workers: %d", serveQueueType, serveWorkers)
	logger.Printf("Web UI: http://%s", addr)
	logger.Printf("API: http://%s/api/v1", addr)
	logger.Printf("Health: http://%s/health", addr)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Server error: %v", err)
	}

	logger.Println("Server stopped")
}

func startMetricsServer(port int, logger *log.Logger) {
	router := mux.NewRouter()
	router.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		// Prometheus-compatible metrics
		fmt.Fprintln(w, "# HELP m9m_up Whether the m9m server is up")
		fmt.Fprintln(w, "# TYPE m9m_up gauge")
		fmt.Fprintln(w, "m9m_up 1")
	})

	addr := fmt.Sprintf(":%d", port)
	logger.Printf("Metrics server listening on %s", addr)

	if err := http.ListenAndServe(addr, router); err != nil {
		logger.Printf("Metrics server error: %v", err)
	}
}

