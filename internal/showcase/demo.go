package showcase

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/neul-labs/m9m/internal/engine"
)

// DemoOptions controls demo execution.
type DemoOptions struct {
	Category string // filter by category, or "" for all
	Verbose  bool
	ListOnly bool
}

// DemoResult holds the result of running one demo.
type DemoResult struct {
	Name     string                   `json:"name"`
	Category string                   `json:"category"`
	Nodes    string                   `json:"nodes"`
	Success  bool                     `json:"success"`
	Duration string                   `json:"duration"`
	Items    int                      `json:"items"`
	Output   []map[string]interface{} `json:"output,omitempty"`
	Error    string                   `json:"error,omitempty"`
}

// RunDemos executes demo workflows and returns results.
func RunDemos(opts DemoOptions, registerNodes func(engine.WorkflowEngine)) []DemoResult {
	demos := AllDemos()
	var results []DemoResult

	for _, demo := range demos {
		if opts.Category != "" && opts.Category != "all" && demo.Category != opts.Category {
			continue
		}

		if opts.ListOnly {
			results = append(results, DemoResult{
				Name:     demo.Name,
				Category: demo.Category,
				Nodes:    demo.Nodes,
			})
			continue
		}

		eng := engine.NewWorkflowEngine()
		registerNodes(eng)

		start := time.Now()
		execResult, err := eng.ExecuteWorkflow(demo.Workflow, demo.Input)
		elapsed := time.Since(start)

		dr := DemoResult{
			Name:     demo.Name,
			Category: demo.Category,
			Nodes:    demo.Nodes,
			Duration: elapsed.String(),
		}

		if err != nil {
			dr.Success = false
			dr.Error = err.Error()
		} else if execResult.Error != nil {
			dr.Success = false
			dr.Error = execResult.Error.Error()
		} else {
			dr.Success = true
			dr.Items = len(execResult.Data)
			if opts.Verbose {
				for _, item := range execResult.Data {
					dr.Output = append(dr.Output, item.JSON)
				}
			}
		}

		results = append(results, dr)
	}

	return results
}

// PrintDemoTable prints demo results in table format.
func PrintDemoTable(w io.Writer, results []DemoResult, verbose bool) {
	SectionHeader(w, "m9m Capability Demo")

	for i, r := range results {
		if r.Duration == "" {
			// List mode
			fmt.Fprintf(w, "  %d. %-35s [%s]\n", i+1, r.Name, r.Category)
			fmt.Fprintf(w, "     Pipeline: %s\n", r.Nodes)
			continue
		}

		status := "PASS"
		if !r.Success {
			status = "FAIL"
		}

		fmt.Fprintf(w, "Demo %d: %s\n", i+1, r.Name)
		PrintDivider(w, 50)
		fmt.Fprintf(w, "  Category:  %s\n", r.Category)
		fmt.Fprintf(w, "  Pipeline:  %s\n", r.Nodes)
		fmt.Fprintf(w, "  Status:    %s\n", status)
		fmt.Fprintf(w, "  Duration:  %s\n", r.Duration)

		if r.Success {
			fmt.Fprintf(w, "  Output:    %d items\n", r.Items)
		} else {
			fmt.Fprintf(w, "  Error:     %s\n", r.Error)
		}

		if verbose && len(r.Output) > 0 {
			fmt.Fprintln(w, "  Data:")
			for j, item := range r.Output {
				data, _ := json.MarshalIndent(item, "    ", "  ")
				fmt.Fprintf(w, "    [%d] %s\n", j, string(data))
			}
		}

		fmt.Fprintln(w)
	}

	// Summary
	passed := 0
	total := 0
	for _, r := range results {
		if r.Duration != "" {
			total++
			if r.Success {
				passed++
			}
		}
	}
	if total > 0 {
		fmt.Fprintf(w, "Results: %d/%d demos passed\n", passed, total)
	}
}

// PrintDemoJSON prints demo results as JSON.
func PrintDemoJSON(w io.Writer, results []DemoResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}
