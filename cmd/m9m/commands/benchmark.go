package commands

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/neul-labs/m9m/internal/showcase"
)

var (
	benchJSON     bool
	benchQuick    bool
	benchCategory string
)

var benchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Run live performance benchmarks with comparison to n8n",
	Long: `Run real benchmarks measuring m9m performance and compare against
documented n8n reference values.

Categories:
  expressions  - Expression evaluation (ops/sec)
  workflows    - Workflow execution throughput
  concurrency  - Parallel scaling at multiple goroutine levels
  memory       - Memory usage for expressions and workflows
  startup      - Engine init, node registration, first execution

Examples:
  m9m benchmark                     Run all benchmarks
  m9m benchmark --quick             Run with reduced iterations
  m9m benchmark --category expressions  Run expression benchmarks only
  m9m benchmark --json              Output results as JSON`,
	Run: runBenchmark,
}

func init() {
	benchmarkCmd.Flags().BoolVar(&benchJSON, "json", false, "Output results as JSON")
	benchmarkCmd.Flags().BoolVar(&benchQuick, "quick", false, "Run with reduced iterations (faster)")
	benchmarkCmd.Flags().StringVar(&benchCategory, "category", "", "Run specific category: expressions, workflows, concurrency, memory, startup")
}

func runBenchmark(cmd *cobra.Command, args []string) {
	opts := showcase.BenchmarkOptions{
		Quick:    benchQuick,
		Category: benchCategory,
	}

	report := showcase.RunBenchmark(opts, RegisterAllNodes)

	if benchJSON {
		report.PrintJSON(os.Stdout)
	} else {
		report.PrintTable(os.Stdout)
	}
}
