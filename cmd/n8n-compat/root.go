package main

import "github.com/spf13/cobra"

var (
	verbose         bool
	outputFormat    string
	nodeModulesPath string
	nodesDirectory  string
)

func newRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "n8n-compat",
		Short: "n8n Compatibility Tool for n8n-go",
		Long: `A comprehensive tool for testing and managing n8n compatibility in n8n-go.
Provides workflow import/export, JavaScript runtime testing, and node compatibility validation.`,
	}

	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "format", "json", "Output format (json, yaml, table)")
	rootCmd.PersistentFlags().StringVar(&nodeModulesPath, "node-modules", "./node_modules", "Path to node_modules directory")
	rootCmd.PersistentFlags().StringVar(&nodesDirectory, "nodes-dir", "./n8n-nodes", "Directory containing n8n node implementations")

	rootCmd.AddCommand(
		createWorkflowCommands(),
		createJavaScriptCommands(),
		createNodeCommands(),
		createTestCommands(),
	)

	return rootCmd
}
