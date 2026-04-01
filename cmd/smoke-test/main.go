package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type testResult struct {
	Name     string
	Pass     bool
	Duration time.Duration
	Error    string
}

func main() {
	fmt.Println("=== m9m Smoke Test Suite ===")
	fmt.Println()

	var results []testResult
	startTime := time.Now()

	// Phase 1: Verify binary builds
	results = append(results, runPhase("Build binary", func() error {
		cmd := exec.Command("make", "build")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}))

	// Phase 2: Run unit tests
	results = append(results, runPhase("Unit tests", func() error {
		cmd := exec.Command("go", "test", "./...")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}))

	// Phase 3: Execute smoke test workflows
	smokeDir := "test-workflows/smoke"
	files, err := filepath.Glob(filepath.Join(smokeDir, "*.json"))
	if err != nil || len(files) == 0 {
		results = append(results, testResult{
			Name:  "Find smoke test workflows",
			Pass:  false,
			Error: fmt.Sprintf("no smoke test workflows found in %s", smokeDir),
		})
	} else {
		for _, f := range files {
			name := filepath.Base(f)
			results = append(results, runPhase(fmt.Sprintf("Smoke: %s", name), func() error {
				return execWorkflow(f)
			}))
		}
	}

	// Phase 4: Execute example workflows (skip network-dependent ones)
	exampleDir := "examples"
	skipExamples := map[string]bool{
		"error-handling.json":     true, // Requires external HTTP
		"webhook-processing.json": true, // Requires incoming HTTP request
	}

	exampleFiles, _ := filepath.Glob(filepath.Join(exampleDir, "**", "*.json"))
	for _, f := range exampleFiles {
		name := filepath.Base(f)
		if skipExamples[name] {
			continue
		}
		results = append(results, runPhase(fmt.Sprintf("Example: %s", name), func() error {
			return execWorkflow(f)
		}))
	}

	// Phase 5: Verify node registration
	results = append(results, runPhase("Node registration check", func() error {
		// Execute a minimal workflow that uses the start node
		return execWorkflow("test-workflows/smoke/set-node.json")
	}))

	// Print summary
	totalDuration := time.Since(startTime)
	fmt.Println()
	fmt.Println("=== Smoke Test Results ===")
	fmt.Println()

	passed := 0
	failed := 0
	for _, r := range results {
		status := "PASS"
		if !r.Pass {
			status = "FAIL"
			failed++
		} else {
			passed++
		}
		fmt.Printf("  [%s] %-50s %s\n", status, r.Name, r.Duration.Round(time.Millisecond))
		if r.Error != "" {
			fmt.Printf("         Error: %s\n", r.Error)
		}
	}

	fmt.Println()
	fmt.Printf("Total: %d passed, %d failed (%.1fs)\n", passed, failed, totalDuration.Seconds())

	if failed > 0 {
		os.Exit(1)
	}
}

func runPhase(name string, fn func() error) testResult {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	r := testResult{
		Name:     name,
		Duration: duration,
		Pass:     err == nil,
	}
	if err != nil {
		r.Error = err.Error()
	}

	status := "PASS"
	if !r.Pass {
		status = "FAIL"
	}
	fmt.Printf("  [%s] %s (%s)\n", status, name, duration.Round(time.Millisecond))
	return r
}

type execResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func execWorkflow(path string) error {
	cmd := exec.Command("./m9m", "exec", path)
	out, err := cmd.Output()
	if err != nil {
		// Try to parse JSON error from stdout
		if len(out) > 0 {
			var result execResult
			if jsonErr := json.Unmarshal(out, &result); jsonErr == nil {
				return fmt.Errorf("%s", result.Error)
			}
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if stderr != "" {
				return fmt.Errorf("%s", stderr)
			}
		}
		return err
	}

	var result execResult
	if err := json.Unmarshal(out, &result); err != nil {
		return fmt.Errorf("invalid JSON output: %s", string(out[:min(len(out), 200)]))
	}

	if !result.Success {
		return fmt.Errorf("%s", result.Error)
	}

	return nil
}
