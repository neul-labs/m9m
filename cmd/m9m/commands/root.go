package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	workspaceFlag string
	outputFlag    string
	verboseFlag   bool

	// Version info
	version   string
	commit    string
	buildDate string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "m9m",
	Short: "m9m - High-performance workflow automation",
	Long: `m9m is a high-performance, cloud-native workflow automation platform.

Running m9m with no arguments starts the server with the embedded web UI.

For agents and scripts, use 'm9m exec' for direct workflow execution:
  m9m exec workflow.json                    Execute a workflow file directly
  m9m exec workflow.json --input '{"x":1}'  Execute with input data

Other commands:
  m9m serve --port 3000       Start server on a custom port
  m9m node list               List available node types
  m9m health                  Check system health`,
	Run: runServe,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

// SetVersionInfo sets version information from build flags
func SetVersionInfo(v, c, b string) {
	version = v
	commit = c
	buildDate = b
}

func init() {
	// Global flags available to all commands
	rootCmd.PersistentFlags().StringVarP(&workspaceFlag, "workspace", "w", "", "Workspace to use (default: current)")
	rootCmd.PersistentFlags().StringVarP(&outputFlag, "output", "o", "table", "Output format: table, json, yaml")
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose output")

	// Server flags on root command (m9m defaults to running the server)
	rootCmd.Flags().IntVar(&servePort, "port", 8080, "Server port")
	rootCmd.Flags().StringVar(&serveHost, "host", "0.0.0.0", "Server host")
	rootCmd.Flags().IntVar(&serveMetricsPort, "metrics-port", 0, "Metrics port (0 = disabled)")
	rootCmd.Flags().BoolVar(&serveDevMode, "dev", false, "Enable development mode (permissive CORS)")
	rootCmd.Flags().StringVar(&serveDB, "db", "", "SQLite database path")
	rootCmd.Flags().StringVar(&servePostgres, "postgres", "", "PostgreSQL connection URL")
	rootCmd.Flags().StringVar(&serveQueueType, "queue", "sqlite", "Queue type: memory, sqlite")
	rootCmd.Flags().StringVar(&serveQueueDB, "queue-db", "", "Queue SQLite database path (for sqlite queue)")
	rootCmd.Flags().IntVar(&serveWorkers, "workers", 4, "Number of worker threads for job processing")

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(nodeCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(workspaceCmd)
	rootCmd.AddCommand(statusCmd)

	// Workflow commands
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(execCmd) // Agent-friendly direct execution

	// Execution commands
	rootCmd.AddCommand(executionCmd)

	// Service commands
	rootCmd.AddCommand(serviceCmd)

	// Server command
	rootCmd.AddCommand(serveCmd)

	// Showcase commands
	rootCmd.AddCommand(benchmarkCmd)
	rootCmd.AddCommand(demoCmd)
	rootCmd.AddCommand(compareCmd)

	// Operator commands
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(upgradeCmd)
}

// GetWorkspace returns the workspace to use
func GetWorkspace() string {
	if workspaceFlag != "" {
		return workspaceFlag
	}
	// Check environment variable
	if ws := os.Getenv("M9M_WORKSPACE"); ws != "" {
		return ws
	}
	// Return empty to use default
	return ""
}

// GetOutputFormat returns the output format
func GetOutputFormat() string {
	return outputFlag
}

// IsVerbose returns whether verbose output is enabled
func IsVerbose() bool {
	return verboseFlag
}

// PrintVerbose prints message only if verbose mode is enabled
func PrintVerbose(format string, args ...interface{}) {
	if verboseFlag {
		fmt.Printf(format, args...)
	}
}
