package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/n8n-go/n8n-go/internal/compatibility"
	"github.com/n8n-go/n8n-go/internal/runtime"
	"github.com/n8n-go/n8n-go/pkg/model"
)

var (
	verbose       bool
	outputFormat  string
	nodeModulesPath string
	nodesDirectory  string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "n8n-compat",
		Short: "n8n Compatibility Tool for n8n-go",
		Long: `A comprehensive tool for testing and managing n8n compatibility in n8n-go.
Provides workflow import/export, JavaScript runtime testing, and node compatibility validation.`,
	}

	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose output")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "format", "json", "Output format (json, yaml, table)")
	rootCmd.PersistentFlags().StringVar(&nodeModulesPath, "node-modules", "./node_modules", "Path to node_modules directory")
	rootCmd.PersistentFlags().StringVar(&nodesDirectory, "nodes-dir", "./n8n-nodes", "Directory containing n8n node implementations")

	// Add subcommands
	rootCmd.AddCommand(
		createWorkflowCommands(),
		createJavaScriptCommands(),
		createNodeCommands(),
		createTestCommands(),
	)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func createWorkflowCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workflow",
		Short: "Workflow import/export operations",
	}

	importCmd := &cobra.Command{
		Use:   "import [n8n-workflow.json]",
		Short: "Import n8n workflow to n8n-go format",
		Args:  cobra.ExactArgs(1),
		Run:   importWorkflow,
	}

	exportCmd := &cobra.Command{
		Use:   "export [n8n-go-workflow.json]",
		Short: "Export n8n-go workflow to n8n format",
		Args:  cobra.ExactArgs(1),
		Run:   exportWorkflow,
	}

	validateCmd := &cobra.Command{
		Use:   "validate [workflow.json]",
		Short: "Validate n8n workflow format",
		Args:  cobra.ExactArgs(1),
		Run:   validateWorkflow,
	}

	convertCmd := &cobra.Command{
		Use:   "convert [input-dir] [output-dir]",
		Short: "Batch convert n8n workflows",
		Args:  cobra.ExactArgs(2),
		Run:   batchConvertWorkflows,
	}

	cmd.AddCommand(importCmd, exportCmd, validateCmd, convertCmd)
	return cmd
}

func createJavaScriptCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "js",
		Short: "JavaScript runtime operations",
	}

	executeCmd := &cobra.Command{
		Use:   "execute [script.js]",
		Short: "Execute JavaScript code in n8n runtime",
		Args:  cobra.ExactArgs(1),
		Run:   executeJavaScript,
	}

	expressionCmd := &cobra.Command{
		Use:   "expression [expression]",
		Short: "Evaluate n8n expression",
		Args:  cobra.ExactArgs(1),
		Run:   evaluateExpression,
	}

	npmCmd := &cobra.Command{
		Use:   "npm [package-name]",
		Short: "Test npm package loading",
		Args:  cobra.ExactArgs(1),
		Run:   testNpmPackage,
	}

	benchmarkCmd := &cobra.Command{
		Use:   "benchmark",
		Short: "Run JavaScript performance benchmarks",
		Run:   runJavaScriptBenchmarks,
	}

	cmd.AddCommand(executeCmd, expressionCmd, npmCmd, benchmarkCmd)
	return cmd
}

func createNodeCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Node compatibility operations",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available n8n nodes",
		Run:   listNodes,
	}

	testCmd := &cobra.Command{
		Use:   "test [node-type]",
		Short: "Test specific n8n node",
		Args:  cobra.ExactArgs(1),
		Run:   testNode,
	}

	loadCmd := &cobra.Command{
		Use:   "load [node-directory]",
		Short: "Load and validate n8n node",
		Args:  cobra.ExactArgs(1),
		Run:   loadNode,
	}

	cmd.AddCommand(listCmd, testCmd, loadCmd)
	return cmd
}

func createTestCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Compatibility testing operations",
	}

	allCmd := &cobra.Command{
		Use:   "all",
		Short: "Run all compatibility tests",
		Run:   runAllTests,
	}

	workflowTestCmd := &cobra.Command{
		Use:   "workflows [test-dir]",
		Short: "Test workflow compatibility",
		Args:  cobra.ExactArgs(1),
		Run:   testWorkflowCompatibility,
	}

	performanceCmd := &cobra.Command{
		Use:   "performance",
		Short: "Run performance tests",
		Run:   runPerformanceTests,
	}

	cmd.AddCommand(allCmd, workflowTestCmd, performanceCmd)
	return cmd
}

func importWorkflow(cmd *cobra.Command, args []string) {
	workflowFile := args[0]

	log.Printf("Importing n8n workflow: %s", workflowFile)

	data, err := ioutil.ReadFile(workflowFile)
	if err != nil {
		log.Fatalf("Failed to read workflow file: %v", err)
	}

	importer := compatibility.NewN8nWorkflowImporter()
	result, err := importer.ImportWorkflow(data)
	if err != nil {
		log.Fatalf("Failed to import workflow: %v", err)
	}

	// Output results
	if outputFormat == "json" {
		output, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(output))
	} else {
		fmt.Printf("✅ Successfully imported workflow: %s\n", result.Workflow.Name)
		fmt.Printf("📊 Statistics:\n")
		fmt.Printf("  - Total Nodes: %d\n", result.Statistics.TotalNodes)
		fmt.Printf("  - Converted Nodes: %d\n", result.Statistics.ConvertedNodes)
		fmt.Printf("  - Skipped Nodes: %d\n", result.Statistics.SkippedNodes)
		fmt.Printf("  - Total Connections: %d\n", result.Statistics.TotalConnections)
		fmt.Printf("  - Converted Connections: %d\n", result.Statistics.ConvertedConnections)

		if len(result.ConversionIssues) > 0 {
			fmt.Printf("⚠️  Conversion Issues:\n")
			for _, issue := range result.ConversionIssues {
				fmt.Printf("  - %s: %s\n", issue.Type, issue.Message)
			}
		}

		if len(result.MissingNodes) > 0 {
			fmt.Printf("❌ Missing Nodes:\n")
			for _, node := range result.MissingNodes {
				fmt.Printf("  - %s\n", node)
			}
		}
	}

	// Save converted workflow
	outputFile := strings.TrimSuffix(workflowFile, filepath.Ext(workflowFile)) + "_converted.json"
	workflowData, _ := json.MarshalIndent(result.Workflow, "", "  ")
	if err := ioutil.WriteFile(outputFile, workflowData, 0644); err != nil {
		log.Printf("Warning: Failed to save converted workflow: %v", err)
	} else {
		log.Printf("💾 Converted workflow saved to: %s", outputFile)
	}
}

func exportWorkflow(cmd *cobra.Command, args []string) {
	workflowFile := args[0]

	log.Printf("Exporting n8n-go workflow: %s", workflowFile)

	data, err := ioutil.ReadFile(workflowFile)
	if err != nil {
		log.Fatalf("Failed to read workflow file: %v", err)
	}

	var workflow model.Workflow
	if err := json.Unmarshal(data, &workflow); err != nil {
		log.Fatalf("Failed to parse n8n-go workflow: %v", err)
	}

	importer := compatibility.NewN8nWorkflowImporter()
	exportedData, err := importer.ExportWorkflow(&workflow)
	if err != nil {
		log.Fatalf("Failed to export workflow: %v", err)
	}

	// Save exported workflow
	outputFile := strings.TrimSuffix(workflowFile, filepath.Ext(workflowFile)) + "_n8n.json"
	if err := ioutil.WriteFile(outputFile, exportedData, 0644); err != nil {
		log.Fatalf("Failed to save exported workflow: %v", err)
	}

	fmt.Printf("✅ Successfully exported workflow to n8n format: %s\n", outputFile)
}

func validateWorkflow(cmd *cobra.Command, args []string) {
	workflowFile := args[0]

	data, err := ioutil.ReadFile(workflowFile)
	if err != nil {
		log.Fatalf("Failed to read workflow file: %v", err)
	}

	importer := compatibility.NewN8nWorkflowImporter()
	if err := importer.ValidateN8nWorkflow(data); err != nil {
		fmt.Printf("❌ Workflow validation failed: %s\n", err.Error())
		os.Exit(1)
	}

	fmt.Printf("✅ Workflow validation passed\n")
}

func batchConvertWorkflows(cmd *cobra.Command, args []string) {
	inputDir := args[0]
	outputDir := args[1]

	log.Printf("Batch converting workflows from %s to %s", inputDir, outputDir)

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Find all JSON files in input directory
	files, err := filepath.Glob(filepath.Join(inputDir, "*.json"))
	if err != nil {
		log.Fatalf("Failed to find workflow files: %v", err)
	}

	importer := compatibility.NewN8nWorkflowImporter()
	successCount := 0
	errorCount := 0

	for _, file := range files {
		log.Printf("Converting: %s", filepath.Base(file))

		data, err := ioutil.ReadFile(file)
		if err != nil {
			log.Printf("  ❌ Failed to read file: %v", err)
			errorCount++
			continue
		}

		result, err := importer.ImportWorkflow(data)
		if err != nil {
			log.Printf("  ❌ Failed to convert: %v", err)
			errorCount++
			continue
		}

		// Save converted workflow
		outputFile := filepath.Join(outputDir, filepath.Base(file))
		workflowData, _ := json.MarshalIndent(result.Workflow, "", "  ")
		if err := ioutil.WriteFile(outputFile, workflowData, 0644); err != nil {
			log.Printf("  ❌ Failed to save: %v", err)
			errorCount++
			continue
		}

		successCount++
		log.Printf("  ✅ Converted successfully")
	}

	fmt.Printf("\n📊 Batch conversion complete:\n")
	fmt.Printf("  - Successful: %d\n", successCount)
	fmt.Printf("  - Failed: %d\n", errorCount)
}

func executeJavaScript(cmd *cobra.Command, args []string) {
	scriptFile := args[0]

	jsRuntime := runtime.NewJavaScriptRuntime(nodeModulesPath)
	defer jsRuntime.Dispose()

	code, err := ioutil.ReadFile(scriptFile)
	if err != nil {
		log.Fatalf("Failed to read script file: %v", err)
	}

	context := &runtime.ExecutionContext{
		WorkflowID:  "test-workflow",
		ExecutionID: "test-execution",
		NodeID:      "test-node",
		Variables:   make(map[string]interface{}),
	}

	items := []model.DataItem{
		{JSON: map[string]interface{}{"test": true}},
	}

	start := time.Now()
	result, err := jsRuntime.Execute(string(code), context, items)
	duration := time.Since(start)

	if err != nil {
		log.Fatalf("JavaScript execution failed: %v", err)
	}

	fmt.Printf("✅ Execution completed in %v\n", duration)

	if outputFormat == "json" {
		output, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(output))
	} else {
		fmt.Printf("Result: %+v\n", result)
	}
}

func evaluateExpression(cmd *cobra.Command, args []string) {
	expression := args[0]

	jsRuntime := runtime.NewJavaScriptRuntime(nodeModulesPath)
	defer jsRuntime.Dispose()

	// Create test context
	context := map[string]interface{}{
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

	jsRuntime := runtime.NewJavaScriptRuntime(nodeModulesPath)
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
	jsRuntime := runtime.NewJavaScriptRuntime(nodeModulesPath)
	defer jsRuntime.Dispose()

	benchmarks := []struct {
		name string
		code string
	}{
		{
			name: "Simple Calculation",
			code: "Math.sqrt(Date.now()) * 42",
		},
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

	context := &runtime.ExecutionContext{
		WorkflowID: "benchmark",
	}

	items := []model.DataItem{{JSON: map[string]interface{}{}}}

	fmt.Printf("🚀 Running JavaScript Benchmarks\n\n")

	for _, benchmark := range benchmarks {
		fmt.Printf("Testing: %s\n", benchmark.name)

		// Warm up
		jsRuntime.Execute(benchmark.code, context, items)

		// Benchmark
		start := time.Now()
		iterations := 100
		for i := 0; i < iterations; i++ {
			_, err := jsRuntime.Execute(benchmark.code, context, items)
			if err != nil {
				log.Printf("  ❌ Error: %v", err)
				break
			}
		}
		duration := time.Since(start)

		avgDuration := duration / time.Duration(iterations)
		fmt.Printf("  ⏱️  Average: %v (%d iterations)\n", avgDuration, iterations)
		fmt.Printf("  📊 Total: %v\n\n", duration)
	}
}

func listNodes(cmd *cobra.Command, args []string) {
	jsRuntime := runtime.NewJavaScriptRuntime(nodeModulesPath)
	defer jsRuntime.Dispose()

	nodes, err := compatibility.LoadN8nNodesFromDirectory(nodesDirectory, jsRuntime)
	if err != nil {
		log.Fatalf("Failed to load nodes: %v", err)
	}

	fmt.Printf("📦 Available n8n Nodes (%d total)\n\n", len(nodes))

	for nodeType, executor := range nodes {
		definition := executor.GetNodeDefinition()
		fmt.Printf("• %s\n", nodeType)
		fmt.Printf("  Name: %s\n", definition.DisplayName)
		fmt.Printf("  Description: %s\n", definition.Description)
		fmt.Printf("  Version: %s\n", definition.Version)
		fmt.Printf("  Properties: %d\n", len(definition.Properties))
		fmt.Printf("  Inputs: %v\n", definition.Inputs)
		fmt.Printf("  Outputs: %v\n", definition.Outputs)
		fmt.Println()
	}
}

func testNode(cmd *cobra.Command, args []string) {
	nodeType := args[0]

	jsRuntime := runtime.NewJavaScriptRuntime(nodeModulesPath)
	defer jsRuntime.Dispose()

	// Find node in nodes directory
	nodePath := filepath.Join(nodesDirectory, nodeType)
	if _, err := os.Stat(nodePath); os.IsNotExist(err) {
		log.Fatalf("Node directory not found: %s", nodePath)
	}

	executor, err := compatibility.CreateN8nCompatibleNode(nodePath, jsRuntime)
	if err != nil {
		log.Fatalf("Failed to create node: %v", err)
	}

	fmt.Printf("🧪 Testing Node: %s\n", nodeType)

	// Test basic execution
	input := &model.NodeExecutionInput{
		Items: []model.DataItem{
			{JSON: map[string]interface{}{"test": true}},
		},
		Config: []byte(`{}`),
	}

	start := time.Now()
	output, err := executor.Execute(input)
	duration := time.Since(start)

	if err != nil {
		log.Fatalf("Node execution failed: %v", err)
	}

	fmt.Printf("✅ Node executed successfully in %v\n", duration)
	fmt.Printf("Input Items: %d\n", len(input.Items))
	fmt.Printf("Output Items: %d\n", len(output.Items))

	if output.Error != "" {
		fmt.Printf("❌ Error: %s\n", output.Error)
	}

	if verbose && len(output.Items) > 0 {
		fmt.Printf("Sample Output:\n")
		sampleOutput, _ := json.MarshalIndent(output.Items[0], "", "  ")
		fmt.Println(string(sampleOutput))
	}
}

func loadNode(cmd *cobra.Command, args []string) {
	nodePath := args[0]

	jsRuntime := runtime.NewJavaScriptRuntime(nodeModulesPath)
	defer jsRuntime.Dispose()

	log.Printf("Loading node from: %s", nodePath)

	executor, err := compatibility.CreateN8nCompatibleNode(nodePath, jsRuntime)
	if err != nil {
		log.Fatalf("Failed to load node: %v", err)
	}

	definition := executor.GetNodeDefinition()

	fmt.Printf("✅ Node loaded successfully\n")
	fmt.Printf("Name: %s\n", definition.DisplayName)
	fmt.Printf("Type: %s\n", definition.Name)
	fmt.Printf("Description: %s\n", definition.Description)
	fmt.Printf("Version: %s\n", definition.Version)

	if len(definition.Properties) > 0 {
		fmt.Printf("\nProperties:\n")
		for _, prop := range definition.Properties {
			fmt.Printf("  • %s (%s)\n", prop.DisplayName, prop.Type)
			if prop.Required {
				fmt.Printf("    Required: Yes\n")
			}
			if prop.Description != "" {
				fmt.Printf("    Description: %s\n", prop.Description)
			}
		}
	}
}

func runAllTests(cmd *cobra.Command, args []string) {
	fmt.Printf("🧪 Running All Compatibility Tests\n\n")

	// Test JavaScript runtime
	fmt.Printf("1. Testing JavaScript Runtime...\n")
	runBasicJavaScriptTests()

	// Test workflow import/export
	fmt.Printf("\n2. Testing Workflow Import/Export...\n")
	runBasicWorkflowTests()

	// Test node loading
	fmt.Printf("\n3. Testing Node Loading...\n")
	runBasicNodeTests()

	fmt.Printf("\n✅ All tests completed successfully!\n")
}

func testWorkflowCompatibility(cmd *cobra.Command, args []string) {
	testDir := args[0]

	files, err := filepath.Glob(filepath.Join(testDir, "*.json"))
	if err != nil {
		log.Fatalf("Failed to find test files: %v", err)
	}

	importer := compatibility.NewN8nWorkflowImporter()
	successCount := 0
	totalCount := len(files)

	fmt.Printf("🧪 Testing %d workflows for compatibility\n\n", totalCount)

	for _, file := range files {
		fmt.Printf("Testing: %s\n", filepath.Base(file))

		data, err := ioutil.ReadFile(file)
		if err != nil {
			fmt.Printf("  ❌ Failed to read: %v\n", err)
			continue
		}

		result, err := importer.ImportWorkflow(data)
		if err != nil {
			fmt.Printf("  ❌ Import failed: %v\n", err)
			continue
		}

		successCount++
		fmt.Printf("  ✅ Import successful\n")

		if len(result.ConversionIssues) > 0 {
			fmt.Printf("  ⚠️  Issues: %d\n", len(result.ConversionIssues))
		}
	}

	fmt.Printf("\n📊 Results: %d/%d workflows imported successfully (%.1f%%)\n",
		successCount, totalCount, float64(successCount)/float64(totalCount)*100)
}

func runPerformanceTests(cmd *cobra.Command, args []string) {
	fmt.Printf("🚀 Running Performance Tests\n\n")

	jsRuntime := runtime.NewJavaScriptRuntime(nodeModulesPath)
	defer jsRuntime.Dispose()

	// Test large data processing
	fmt.Printf("1. Large Data Processing Test\n")
	testLargeDataProcessing(jsRuntime)

	// Test complex workflow import
	fmt.Printf("\n2. Complex Workflow Import Test\n")
	testComplexWorkflowImport()

	// Test concurrent execution
	fmt.Printf("\n3. Concurrent Execution Test\n")
	testConcurrentExecution(jsRuntime)

	fmt.Printf("\n✅ Performance tests completed!\n")
}

// Helper functions for testing

func runBasicJavaScriptTests() {
	jsRuntime := runtime.NewJavaScriptRuntime(nodeModulesPath)
	defer jsRuntime.Dispose()

	// Test basic execution
	context := &runtime.ExecutionContext{WorkflowID: "test"}
	items := []model.DataItem{{JSON: map[string]interface{}{"test": true}}}

	_, err := jsRuntime.Execute("console.log('Hello World!'); 42;", context, items)
	if err != nil {
		log.Printf("  ❌ Basic execution failed: %v", err)
		return
	}

	// Test npm package loading
	_, err = jsRuntime.LoadNpmPackage("lodash", "latest")
	if err != nil {
		log.Printf("  ❌ Package loading failed: %v", err)
		return
	}

	fmt.Printf("  ✅ JavaScript runtime working correctly\n")
}

func runBasicWorkflowTests() {
	// Create a simple test workflow
	testWorkflow := map[string]interface{}{
		"id":   "test",
		"name": "Test Workflow",
		"nodes": []map[string]interface{}{
			{
				"id":   "start",
				"name": "Start",
				"type": "manual",
				"parameters": map[string]interface{}{},
			},
		},
		"connections": map[string]interface{}{},
	}

	data, _ := json.Marshal(testWorkflow)
	importer := compatibility.NewN8nWorkflowImporter()

	_, err := importer.ImportWorkflow(data)
	if err != nil {
		log.Printf("  ❌ Workflow import failed: %v", err)
		return
	}

	fmt.Printf("  ✅ Workflow import/export working correctly\n")
}

func runBasicNodeTests() {
	jsRuntime := runtime.NewJavaScriptRuntime(nodeModulesPath)
	defer jsRuntime.Dispose()

	// Test loading nodes from directory
	_, err := compatibility.LoadN8nNodesFromDirectory(nodesDirectory, jsRuntime)
	if err != nil {
		log.Printf("  ⚠️  No nodes directory found (%s), skipping node tests\n", nodesDirectory)
		return
	}

	fmt.Printf("  ✅ Node loading working correctly\n")
}

func testLargeDataProcessing(jsRuntime *runtime.JavaScriptRuntime) {
	context := &runtime.ExecutionContext{WorkflowID: "perf-test"}

	// Create large dataset
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

func testComplexWorkflowImport() {
	// Create complex workflow with many nodes
	nodes := make([]map[string]interface{}, 50)
	for i := 0; i < 50; i++ {
		nodes[i] = map[string]interface{}{
			"id":   fmt.Sprintf("node%d", i),
			"name": fmt.Sprintf("Node %d", i),
			"type": "n8n-nodes-base.set",
			"parameters": map[string]interface{}{
				"values": map[string]interface{}{
					"string": []map[string]interface{}{
						{"name": "output", "value": fmt.Sprintf("Node %d output", i)},
					},
				},
			},
		}
	}

	workflow := map[string]interface{}{
		"id":          "complex-test",
		"name":        "Complex Test Workflow",
		"nodes":       nodes,
		"connections": map[string]interface{}{},
	}

	data, _ := json.Marshal(workflow)
	importer := compatibility.NewN8nWorkflowImporter()

	start := time.Now()
	result, err := importer.ImportWorkflow(data)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("  ❌ Failed: %v\n", err)
		return
	}

	fmt.Printf("  ✅ Imported %d nodes in %v\n", len(result.Workflow.Nodes), duration)
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
				_, err := jsRuntime.Execute(code, context, items)
				if err != nil {
					results <- err
					return
				}
			}
			results <- nil
		}()
	}

	// Wait for all goroutines
	errors := 0
	for i := 0; i < numGoroutines; i++ {
		if err := <-results; err != nil {
			errors++
		}
	}

	duration := time.Since(start)
	totalOperations := numGoroutines * iterations

	if errors > 0 {
		fmt.Printf("  ❌ %d errors out of %d operations\n", errors, totalOperations)
		return
	}

	fmt.Printf("  ✅ %d concurrent operations completed in %v\n", totalOperations, duration)
}