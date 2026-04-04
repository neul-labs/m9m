package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/runtime"
	"github.com/spf13/cobra"
)

func createJavaScriptCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "js",
		Short: "JavaScript runtime operations",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "execute [script.js]",
			Short: "Execute JavaScript code in n8n runtime",
			Args:  cobra.ExactArgs(1),
			Run:   executeJavaScript,
		},
		&cobra.Command{
			Use:   "expression [expression]",
			Short: "Evaluate n8n expression",
			Args:  cobra.ExactArgs(1),
			Run:   evaluateExpression,
		},
		&cobra.Command{
			Use:   "npm [package-name]",
			Short: "Test npm package loading",
			Args:  cobra.ExactArgs(1),
			Run:   testNpmPackage,
		},
		&cobra.Command{
			Use:   "benchmark",
			Short: "Run JavaScript performance benchmarks",
			Run:   runJavaScriptBenchmarks,
		},
	)

	return cmd
}

func executeJavaScript(cmd *cobra.Command, args []string) {
	scriptFile := args[0]

	jsRuntime := newJavaScriptRuntime()
	defer jsRuntime.Dispose()

	code := mustReadFile(scriptFile, "script file")
	context := &runtime.ExecutionContext{
		WorkflowID:  "test-workflow",
		ExecutionID: "test-execution",
		NodeID:      "test-node",
		Variables:   make(map[string]interface{}),
	}
	items := []model.DataItem{{JSON: map[string]interface{}{"test": true}}}

	start := time.Now()
	result, err := jsRuntime.Execute(string(code), context, items)
	duration := time.Since(start)
	if err != nil {
		log.Fatalf("JavaScript execution failed: %v", err)
	}

	fmt.Printf("✅ Execution completed in %v\n", duration)
	if outputFormat == "json" {
		printJSON(result)
	} else {
		fmt.Printf("Result: %+v\n", result)
	}
}

func evaluateExpression(cmd *cobra.Command, args []string) {
	expression := args[0]

	jsRuntime := newJavaScriptRuntime()
	defer jsRuntime.Dispose()

	_ = map[string]interface{}{
		"json": map[string]interface{}{
			"name": "Test User",
			"age":  30,
		},
		"workflow": map[string]interface{}{
			"id":   "test-workflow",
			"name": "Test Workflow",
		},
		"parameter": map[string]interface{}{
			"apiKey": "test-key",
		},
	}

	start := time.Now()
	result, err := jsRuntime.ExecuteExpression(expression, nil)
	duration := time.Since(start)
	if err != nil {
		log.Fatalf("Expression evaluation failed: %v", err)
	}

	fmt.Printf("✅ Expression evaluated in %v\n", duration)
	fmt.Printf("Expression: %s\n", expression)
	fmt.Printf("Result: %+v\n", result)
}

func testNpmPackage(cmd *cobra.Command, args []string) {
	packageName := args[0]

	jsRuntime := newJavaScriptRuntime()
	defer jsRuntime.Dispose()

	log.Printf("Testing npm package: %s", packageName)

	start := time.Now()
	pkg, err := jsRuntime.LoadNpmPackage(packageName, "latest")
	duration := time.Since(start)
	if err != nil {
		log.Fatalf("Failed to load package: %v", err)
	}

	fmt.Printf("✅ Package loaded successfully in %v\n", duration)
	fmt.Printf("Package: %s@%s\n", pkg.Name, pkg.Version)
	fmt.Printf("Cache Path: %s\n", pkg.CachePath)
	fmt.Printf("Dependencies: %d\n", len(pkg.Dependencies))

	if verbose {
		fmt.Printf("Dependencies:\n")
		for dep, version := range pkg.Dependencies {
			fmt.Printf("  - %s: %s\n", dep, version)
		}
	}
}

func runJavaScriptBenchmarks(cmd *cobra.Command, args []string) {
	jsRuntime := newJavaScriptRuntime()
	defer jsRuntime.Dispose()

	benchmarks := []struct {
		name string
		code string
	}{
		{name: "Simple Calculation", code: "Math.sqrt(Date.now()) * 42"},
		{
			name: "Array Processing",
			code: `
				const arr = Array.from({length: 1000}, (_, i) => i);
				arr.map(x => x * 2).filter(x => x % 3 === 0).reduce((a, b) => a + b, 0);
			`,
		},
		{
			name: "Object Manipulation",
			code: `
				const obj = {};
				for (let i = 0; i < 1000; i++) {
					obj['key' + i] = 'value' + i;
				}
				Object.keys(obj).length;
			`,
		},
		{
			name: "Lodash Operations",
			code: `
				const data = Array.from({length: 100}, (_, i) => ({id: i, value: Math.random()}));
				_.sortBy(_.filter(data, item => item.value > 0.5), 'value');
			`,
		},
	}

	context := &runtime.ExecutionContext{WorkflowID: "benchmark"}
	items := []model.DataItem{{JSON: map[string]interface{}{}}}

	fmt.Printf("🚀 Running JavaScript Benchmarks\n\n")

	for _, benchmark := range benchmarks {
		fmt.Printf("Testing: %s\n", benchmark.name)
		_, _ = jsRuntime.Execute(benchmark.code, context, items)

		start := time.Now()
		iterations := 100
		for i := 0; i < iterations; i++ {
			if _, err := jsRuntime.Execute(benchmark.code, context, items); err != nil {
				log.Printf("  ❌ Error: %v", err)
				break
			}
		}
		duration := time.Since(start)

		fmt.Printf("  ⏱️  Average: %v (%d iterations)\n", duration/time.Duration(iterations), iterations)
		fmt.Printf("  📊 Total: %v\n\n", duration)
	}
}

func runBasicJavaScriptTests() {
	jsRuntime := newJavaScriptRuntime()
	defer jsRuntime.Dispose()

	context := &runtime.ExecutionContext{WorkflowID: "test"}
	items := []model.DataItem{{JSON: map[string]interface{}{"test": true}}}

	if _, err := jsRuntime.Execute("console.log('Hello World!'); 42;", context, items); err != nil {
		log.Printf("  ❌ Basic execution failed: %v", err)
		return
	}

	if _, err := jsRuntime.LoadNpmPackage("lodash", "latest"); err != nil {
		log.Printf("  ❌ Package loading failed: %v", err)
		return
	}

	fmt.Printf("  ✅ JavaScript runtime working correctly\n")
}

func testLargeDataProcessing(jsRuntime *runtime.JavaScriptRuntime) {
	context := &runtime.ExecutionContext{WorkflowID: "perf-test"}

	items := make([]model.DataItem, 1000)
	for i := 0; i < 1000; i++ {
		items[i] = model.DataItem{
			JSON: map[string]interface{}{
				"id":    i,
				"value": fmt.Sprintf("item_%d", i),
				"data":  make([]int, 100),
			},
		}
	}

	code := `
		const processed = items.map(item => ({
			...item.json,
			processed: true,
			timestamp: new Date().toISOString()
		}));
		processed.length;
	`

	start := time.Now()
	result, err := jsRuntime.Execute(code, context, items)
	duration := time.Since(start)
	if err != nil {
		fmt.Printf("  ❌ Failed: %v\n", err)
		return
	}

	fmt.Printf("  ✅ Processed %v items in %v\n", result, duration)
}

func testConcurrentExecution(jsRuntime *runtime.JavaScriptRuntime) {
	const numGoroutines = 10
	const iterations = 100

	context := &runtime.ExecutionContext{WorkflowID: "concurrent-test"}
	items := []model.DataItem{{JSON: map[string]interface{}{"test": true}}}
	code := `Math.random() * Date.now()`

	start := time.Now()
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				if _, err := jsRuntime.Execute(code, context, items); err != nil {
					results <- err
					return
				}
			}
			results <- nil
		}()
	}

	errors := 0
	for i := 0; i < numGoroutines; i++ {
		if err := <-results; err != nil {
			errors++
		}
	}

	totalOperations := numGoroutines * iterations
	if errors > 0 {
		fmt.Printf("  ❌ %d errors out of %d operations\n", errors, totalOperations)
		return
	}

	fmt.Printf("  ✅ %d concurrent operations completed in %v\n", totalOperations, time.Since(start))
}

var _ = json.Marshal
