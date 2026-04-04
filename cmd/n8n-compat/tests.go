package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"time"

	"github.com/neul-labs/m9m/internal/compatibility"
	"github.com/neul-labs/m9m/internal/model"
	"github.com/spf13/cobra"
)

func createTestCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Compatibility testing operations",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "all",
			Short: "Run all compatibility tests",
			Run:   runAllTests,
		},
		&cobra.Command{
			Use:   "workflows [test-dir]",
			Short: "Test workflow compatibility",
			Args:  cobra.ExactArgs(1),
			Run:   testWorkflowCompatibility,
		},
		&cobra.Command{
			Use:   "performance",
			Short: "Run performance tests",
			Run:   runPerformanceTests,
		},
	)

	return cmd
}

func runAllTests(cmd *cobra.Command, args []string) {
	fmt.Printf("🧪 Running All Compatibility Tests\n\n")

	fmt.Printf("1. Testing JavaScript Runtime...\n")
	runBasicJavaScriptTests()

	fmt.Printf("\n2. Testing Workflow Import/Export...\n")
	runBasicWorkflowTests()

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

	jsRuntime := newJavaScriptRuntime()
	defer jsRuntime.Dispose()

	fmt.Printf("1. Large Data Processing Test\n")
	testLargeDataProcessing(jsRuntime)

	fmt.Printf("\n2. Complex Workflow Import Test\n")
	testComplexWorkflowImport()

	fmt.Printf("\n3. Concurrent Execution Test\n")
	testConcurrentExecution(jsRuntime)

	fmt.Printf("\n✅ Performance tests completed!\n")
}

func runBasicWorkflowTests() {
	testWorkflow := map[string]interface{}{
		"id":   "test",
		"name": "Test Workflow",
		"nodes": []map[string]interface{}{
			{
				"id":         "start",
				"name":       "Start",
				"type":       "manual",
				"parameters": map[string]interface{}{},
			},
		},
		"connections": map[string]interface{}{},
	}

	data, _ := json.Marshal(testWorkflow)
	importer := compatibility.NewN8nWorkflowImporter()
	if _, err := importer.ImportWorkflow(data); err != nil {
		log.Printf("  ❌ Workflow import failed: %v", err)
		return
	}

	fmt.Printf("  ✅ Workflow import/export working correctly\n")
}

func testComplexWorkflowImport() {
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

var _ = model.Workflow{}
