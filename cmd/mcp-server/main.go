// Package main provides the standalone MCP server for m9m workflow automation.
// This server can be used with Claude Code to orchestrate workflows conversationally.
//
// Usage modes:
//
//	Local mode (default):  ./mcp-server --db ./data/m9m.db
//	Cloud mode:            ./mcp-server --api-url https://m9m.example.com
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/neul-labs/m9m/internal/credentials"
	"github.com/neul-labs/m9m/internal/engine"
	"github.com/neul-labs/m9m/internal/mcp"
	"github.com/neul-labs/m9m/internal/mcp/tools"
	"github.com/neul-labs/m9m/internal/mcp/transport"
	"github.com/neul-labs/m9m/internal/plugins"
	"github.com/neul-labs/m9m/internal/scheduler"
	"github.com/neul-labs/m9m/internal/storage"

	// Import node packages to register them
	"github.com/neul-labs/m9m/internal/nodes/ai"
	"github.com/neul-labs/m9m/internal/nodes/cli"
	"github.com/neul-labs/m9m/internal/nodes/core"
	"github.com/neul-labs/m9m/internal/nodes/database"
	"github.com/neul-labs/m9m/internal/nodes/email"
	"github.com/neul-labs/m9m/internal/nodes/http"
	"github.com/neul-labs/m9m/internal/nodes/messaging"
	"github.com/neul-labs/m9m/internal/nodes/timer"
	"github.com/neul-labs/m9m/internal/nodes/transform"
	"github.com/neul-labs/m9m/internal/nodes/trigger"
	"github.com/neul-labs/m9m/internal/nodes/vcs"
)

// Version information
var (
	version = "1.0.0"
	name    = "m9m-mcp-server"
)

func main() {
	// Parse flags
	dbPath := flag.String("db", "", "Path to SQLite database (local mode)")
	postgresURL := flag.String("postgres", "", "PostgreSQL connection URL (local mode)")
	apiURL := flag.String("api-url", "", "m9m API URL for cloud mode (e.g., https://m9m.example.com)")
	dataDir := flag.String("data", "./data", "Data directory for storage and plugins")
	pluginDir := flag.String("plugins", "", "Plugin directory (defaults to {data}/plugins)")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("%s version %s\n", name, version)
		os.Exit(0)
	}

	// Create logger
	logger := log.New(os.Stderr, "[m9m-mcp] ", log.LstdFlags)
	logger.Printf("Starting %s v%s", name, version)

	// Determine mode and initialize accordingly
	var store storage.WorkflowStorage
	var eng engine.WorkflowEngine
	var err error

	if *apiURL != "" {
		// Cloud mode - connect to remote m9m API
		logger.Printf("Running in CLOUD mode, connecting to: %s", *apiURL)
		store, eng, err = initCloudMode(*apiURL)
		if err != nil {
			logger.Fatalf("Failed to initialize cloud mode: %v", err)
		}
	} else {
		// Local mode - use local storage and engine
		logger.Println("Running in LOCAL mode")
		store, eng, err = initLocalMode(*dbPath, *postgresURL, *dataDir)
		if err != nil {
			logger.Fatalf("Failed to initialize local mode: %v", err)
		}
	}
	defer store.Close()

	// Initialize credential manager
	credMgr, err := credentials.NewCredentialManager()
	if err != nil {
		logger.Fatalf("Failed to initialize credential manager: %v", err)
	}

	// Initialize scheduler
	sched := scheduler.NewWorkflowScheduler(eng)

	// Determine plugin directory
	pluginPath := *pluginDir
	if pluginPath == "" {
		pluginPath = filepath.Join(*dataDir, "plugins")
	}

	// Initialize plugin registry
	pluginRegistry := plugins.NewPluginRegistry()
	if err := os.MkdirAll(pluginPath, 0755); err == nil {
		if err := pluginRegistry.LoadPluginsFromDirectory(pluginPath); err != nil {
			logger.Printf("Warning: failed to load plugins: %v", err)
		}
		pluginRegistry.RegisterWithEngine(eng)
	}

	// Create MCP server
	server := mcp.NewServer(
		mcp.WithName(name),
		mcp.WithVersion(version),
		mcp.WithLogger(logger),
	)

	// Initialize server with components
	if err := server.Initialize(eng, store, sched, credMgr); err != nil {
		logger.Fatalf("Failed to initialize MCP server: %v", err)
	}

	// Create tool registry and register all tools
	toolRegistry := tools.NewRegistry()

	// Register node discovery tools
	tools.RegisterNodeTools(toolRegistry)

	// Register action tools (http_request, send_slack, ai_openai, etc.)
	tools.RegisterActionTools(toolRegistry)

	// Register workflow management tools
	tools.RegisterWorkflowTools(toolRegistry, store)

	// Register execution tools
	execManager := tools.RegisterExecutionTools(toolRegistry, eng, store)

	// Register debugging tools
	tools.RegisterDebugTools(toolRegistry, store, execManager)

	// Register plugin tools
	tools.RegisterPluginTools(toolRegistry, pluginRegistry, pluginPath, eng)

	// Register all tools with MCP server
	toolRegistry.RegisterAll(server)

	// Log registered tools
	toolDefs := toolRegistry.ListDefinitions()
	logger.Printf("Registered %d MCP tools", len(toolDefs))

	// Create transport
	stdio := transport.NewStdioTransport(os.Stdin, os.Stdout)

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		logger.Println("Shutting down...")
		cancel()
	}()

	// Start server
	logger.Println("MCP server ready, listening on stdio")
	if err := server.Serve(ctx, stdio); err != nil && err != context.Canceled {
		logger.Fatalf("Server error: %v", err)
	}

	logger.Println("Server stopped")
}

// initLocalMode initializes storage and engine for local operation
func initLocalMode(dbPath, postgresURL, dataDir string) (storage.WorkflowStorage, engine.WorkflowEngine, error) {
	var store storage.WorkflowStorage
	var err error

	// Priority: explicit postgres > explicit sqlite > default sqlite
	if postgresURL != "" {
		store, err = storage.NewPostgresStorage(postgresURL)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
		}
	} else {
		// Use SQLite (default)
		sqlitePath := dbPath
		if sqlitePath == "" {
			// Default to data directory
			if err := os.MkdirAll(dataDir, 0755); err != nil {
				return nil, nil, fmt.Errorf("failed to create data directory: %w", err)
			}
			sqlitePath = filepath.Join(dataDir, "m9m.db")
		}
		store, err = storage.NewSQLiteStorage(sqlitePath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open SQLite database: %w", err)
		}
	}

	// Create and configure engine
	eng := engine.NewEnhancedWorkflowEngine()
	registerNodes(eng)

	return store, eng, nil
}

// initCloudMode initializes remote storage and engine for cloud operation
func initCloudMode(apiURL string) (storage.WorkflowStorage, engine.WorkflowEngine, error) {
	// Create remote client that implements both storage and engine interfaces
	client, err := NewRemoteClient(apiURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create remote client: %w", err)
	}

	return client, client, nil
}

func registerNodes(eng engine.WorkflowEngine) {
	// Core nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.start", core.NewStartNode())

	// Transform nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.set", transform.NewSetNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.filter", transform.NewFilterNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.code", transform.NewCodeNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.function", transform.NewFunctionNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.merge", transform.NewMergeNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.json", transform.NewJSONNode())

	// HTTP nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.httpRequest", http.NewHTTPRequestNode())

	// Trigger nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.webhook", trigger.NewWebhookNode())

	// Timer nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.cron", timer.NewCronNode())

	// Messaging nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.slack", messaging.NewSlackNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.discord", messaging.NewDiscordNode())

	// Database nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.postgres", database.NewPostgresNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.mysql", database.NewMySQLNode())

	// Email nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.emailSend", email.NewSendEmailNode())

	// AI nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.openAi", ai.NewOpenAINode())
	eng.RegisterNodeExecutor("n8n-nodes-base.anthropic", ai.NewAnthropicNode())

	// VCS nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.github", vcs.NewGitHubNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.gitlab", vcs.NewGitLabNode())

	// CLI nodes
	eng.RegisterNodeExecutor("n8n-nodes-base.cliExecute", cli.NewExecuteNode())
}
