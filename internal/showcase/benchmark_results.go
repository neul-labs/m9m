package showcase

import (
	"encoding/json"
	"fmt"
	"io"
	"runtime"
)

// BenchmarkResult holds the result of a single benchmark test.
type BenchmarkResult struct {
	Name    string  `json:"name"`
	Value   float64 `json:"value"`
	Unit    string  `json:"unit"`
	RefName string  `json:"refName,omitempty"`
	Ref     float64 `json:"ref,omitempty"`
	Speedup float64 `json:"speedup,omitempty"`
	Savings float64 `json:"savings,omitempty"`
}

// BenchmarkCategory groups benchmark results by category.
type BenchmarkCategory struct {
	Name    string            `json:"name"`
	Results []BenchmarkResult `json:"results"`
}

// BenchmarkReport is the full benchmark output.
type BenchmarkReport struct {
	System     SystemInfo          `json:"system"`
	Categories []BenchmarkCategory `json:"categories"`
}

// SystemInfo describes the system running benchmarks.
type SystemInfo struct {
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	CPUs      int    `json:"cpus"`
	GoVersion string `json:"goVersion"`
}

// GetSystemInfo returns current system information.
func GetSystemInfo() SystemInfo {
	return SystemInfo{
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		CPUs:      runtime.NumCPU(),
		GoVersion: runtime.Version(),
	}
}

// PrintTable prints the benchmark report as formatted ASCII tables.
func (r *BenchmarkReport) PrintTable(w io.Writer) {
	SectionHeader(w, "m9m Performance Benchmark")
	fmt.Fprintf(w, "System: %s/%s, %d CPUs, %s\n\n", r.System.OS, r.System.Arch, r.System.CPUs, r.System.GoVersion)

	for _, cat := range r.Categories {
		SubHeader(w, cat.Name)
		tw := NewTable(w)

		switch {
		case cat.Name == "Memory Usage":
			fmt.Fprintln(tw, "Metric\tm9m\tn8n (ref)\tSavings")
			PrintDivider(w, 68)
			for _, res := range cat.Results {
				savings := ""
				if res.Savings > 0 {
					savings = FormatSavings(res.Savings)
				}
				fmt.Fprintf(tw, "%s\t%.1f %s\t%.0f %s\t%s\n",
					res.Name, res.Value, res.Unit, res.Ref, res.Unit, savings)
			}
		case cat.Name == "Startup Time":
			fmt.Fprintln(tw, "Phase\tm9m\tn8n (ref)\tSpeedup")
			PrintDivider(w, 68)
			for _, res := range cat.Results {
				speedup := ""
				if res.Speedup > 0 {
					speedup = FormatSpeedup(res.Speedup)
				}
				fmt.Fprintf(tw, "%s\t%.1f %s\t%.0f %s\t%s\n",
					res.Name, res.Value, res.Unit, res.Ref, res.Unit, speedup)
			}
		case cat.Name == "Concurrency Scaling":
			fmt.Fprintln(tw, "Goroutines\tOps/sec (m9m)\tn8n (ref)\tSpeedup")
			PrintDivider(w, 68)
			for _, res := range cat.Results {
				speedup := ""
				if res.Speedup > 0 {
					speedup = FormatSpeedup(res.Speedup)
				}
				fmt.Fprintf(tw, "%s\t%s ops/sec\t%s ops/sec\t%s\n",
					res.Name, FormatNumber(int64(res.Value)), FormatNumber(int64(res.Ref)), speedup)
			}
		default:
			// Expression / Workflow tables
			fmt.Fprintln(tw, "Test\tOps/sec (m9m)\tn8n (ref)\tSpeedup")
			PrintDivider(w, 68)
			for _, res := range cat.Results {
				speedup := ""
				if res.Speedup > 0 {
					speedup = FormatSpeedup(res.Speedup)
				}
				refStr := fmt.Sprintf("%s %s", FormatNumber(int64(res.Ref)), res.Unit)
				if res.Unit == "wf/sec" {
					fmt.Fprintf(tw, "%s\t%s %s\t%s\t%s\n",
						res.Name, FormatNumber(int64(res.Value)), res.Unit, refStr, speedup)
				} else {
					fmt.Fprintf(tw, "%s\t%s ops/sec\t%s ops/sec\t%s\n",
						res.Name, FormatNumber(int64(res.Value)), FormatNumber(int64(res.Ref)), speedup)
				}
			}
		}
		tw.Flush()
		fmt.Fprintln(w)
	}
}

// PrintJSON prints the benchmark report as JSON.
func (r *BenchmarkReport) PrintJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}
