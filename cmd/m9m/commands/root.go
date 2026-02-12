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

It provides a CLI interface for creating, managing, and executing workflows.
Use 'm9m init' to initialize a workspace and get started.

Examples:
  m9m init                    Initialize workspace in current directory
  m9m node list               List available node types
  m9m create --from wf.json   Create a workflow from JSON
  m9m run my-workflow         Execute a workflow
  m9m serve --port 8080       Start full server mode`,
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

	// Execution commands
	rootCmd.AddCommand(executionCmd)

	// Service commands
	rootCmd.AddCommand(serviceCmd)

	// Server command
	rootCmd.AddCommand(serveCmd)
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
