package commands

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/neul-labs/m9m/internal/showcase"
)

var (
	demoCategory string
	demoList     bool
	demoVerbose  bool
	demoJSON     bool
)

var demoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Run curated demo workflows showcasing m9m capabilities",
	Long: `Run end-to-end demo workflows demonstrating key capabilities.
All demos run locally without external services.

Categories:
  transform    - Data transformation and routing demos
  business     - Business logic demos (orders, segmentation)
  expressions  - Expression engine showcase
  enterprise   - Batch processing and enterprise features
  all          - Run all demos (default)

Examples:
  m9m demo                       Run all demos
  m9m demo --list                List available demos
  m9m demo --category business   Run business demos only
  m9m demo --verbose             Show full output data
  m9m demo --json                Output results as JSON`,
	Run: runDemo,
}

func init() {
	demoCmd.Flags().StringVar(&demoCategory, "category", "", "Run demos in category: transform, business, expressions, enterprise, all")
	demoCmd.Flags().BoolVar(&demoList, "list", false, "List available demos without running them")
	demoCmd.Flags().BoolVar(&demoVerbose, "verbose", false, "Show full output data for each demo")
	demoCmd.Flags().BoolVar(&demoJSON, "json", false, "Output results as JSON")
}

func runDemo(cmd *cobra.Command, args []string) {
	opts := showcase.DemoOptions{
		Category: demoCategory,
		Verbose:  demoVerbose,
		ListOnly: demoList,
	}

	results := showcase.RunDemos(opts, RegisterAllNodes)

	if demoJSON {
		showcase.PrintDemoJSON(os.Stdout, results)
	} else {
		showcase.PrintDemoTable(os.Stdout, results, demoVerbose)
	}
}
