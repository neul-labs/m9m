package commands

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/neul-labs/m9m/internal/showcase"
)

var (
	compareSection string
	compareJSON    bool
)

var compareCmd = &cobra.Command{
	Use:   "compare",
	Short: "Show feature parity comparison between m9m and n8n",
	Long: `Print a detailed feature comparison report covering node coverage,
enterprise features, architecture, and expression engine.

Sections:
  nodes         - Node coverage across categories
  features      - Enterprise feature comparison
  architecture  - Architecture and performance comparison
  expressions   - Expression function categories

Examples:
  m9m compare                  Show full comparison
  m9m compare --section nodes  Show node coverage only
  m9m compare --json           Output as JSON`,
	Run: runCompare,
}

func init() {
	compareCmd.Flags().StringVar(&compareSection, "section", "", "Show specific section: nodes, features, architecture, expressions")
	compareCmd.Flags().BoolVar(&compareJSON, "json", false, "Output as JSON")
}

func runCompare(cmd *cobra.Command, args []string) {
	report := showcase.BuildCompareReport()

	if compareJSON {
		report.PrintCompareJSON(os.Stdout)
	} else {
		report.PrintCompareTable(os.Stdout, showcase.CompareOptions{
			Section: compareSection,
		})
	}
}
