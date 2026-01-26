package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/neul-labs/m9m/internal/engine"
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/core"
	"github.com/neul-labs/m9m/internal/nodes/transform"
	"github.com/neul-labs/m9m/internal/nodes/http"
	"github.com/neul-labs/m9m/internal/nodes/trigger"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: integration-test <workflow-file-or-directory>")
		os.Exit(1)
	}

	path := os.Args[1]

	// Check if path is directory or file
	info, err := os.Stat(path)
	if err != nil {
		log.Fatalf("Error accessing path %s: %v", path, err)
	}

	var workflowFiles []string
	if info.IsDir() {
		// Find all JSON files in directory
		files, err := filepath.Glob(filepath.Join(path, "*.json"))
		if err != nil {
			log.Fatalf("Error finding workflow files: %v", err)
		}
		workflowFiles = files
	} else {
		workflowFiles = []string{path}
	}

	if len(workflowFiles) == 0 {
		log.Fatalf("No workflow files found in %s", path)
	}

	fmt.Printf("🚀 Starting integration tests for %d workflow(s)\n\n", len(workflowFiles))

	// Create and setup workflow engine
	workflowEngine := engine.NewWorkflowEngine()
	registerNodes(workflowEngine)

	totalTests := 0
	passedTests := 0

	for _, workflowFile := range workflowFiles {
		fmt.Printf("📋 Testing workflow: %s\n", filepath.Base(workflowFile))

		success := runWorkflowTest(workflowFile, workflowEngine)
		totalTests++
		if success {
			passedTests++
			fmt.Printf("✅ PASSED\n\n")
		} else {
			fmt.Printf("❌ FAILED\n\n")
		}
	}

	fmt.Printf("📊 Integration Test Summary:\n")
	fmt.Printf("   Total Tests: %d\n", totalTests)
	fmt.Printf("   Passed: %d\n", passedTests)
	fmt.Printf("   Failed: %d\n", totalTests-passedTests)
	fmt.Printf("   Success Rate: %.1f%%\n", float64(passedTests)/float64(totalTests)*100)

	if passedTests == totalTests {
		fmt.Printf("\n🎉 All integration tests passed!\n")
		os.Exit(0)
	} else {
		fmt.Printf("\n⚠️  Some integration tests failed.\n")
		os.Exit(1)
	}
}

func registerNodes(engine engine.WorkflowEngine) {
	// Register core nodes
	engine.RegisterNodeExecutor("n8n-nodes-base.start", core.NewStartNode())

	// Register transform nodes
	engine.RegisterNodeExecutor("n8n-nodes-base.set", transform.NewSetNode())
	engine.RegisterNodeExecutor("n8n-nodes-base.filter", transform.NewFilterNode())
	engine.RegisterNodeExecutor("n8n-nodes-base.function", transform.NewFunctionNode())
	engine.RegisterNodeExecutor("n8n-nodes-base.code", transform.NewCodeNode())
	engine.RegisterNodeExecutor("n8n-nodes-base.json", transform.NewJSONNode())
	engine.RegisterNodeExecutor("n8n-nodes-base.merge", transform.NewMergeNode())
	engine.RegisterNodeExecutor("n8n-nodes-base.switch", transform.NewSwitchNode())

	// Register HTTP nodes
	engine.RegisterNodeExecutor("n8n-nodes-base.httpRequest", http.NewHTTPRequestNode())

	// Register trigger nodes
	engine.RegisterNodeExecutor("n8n-nodes-base.webhook", trigger.NewWebhookNode())

	fmt.Printf("   Registered %d node types\n", 10)
}

func runWorkflowTest(workflowFile string, workflowEngine engine.WorkflowEngine) bool {
	// Read workflow file
	data, err := ioutil.ReadFile(workflowFile)
	if err != nil {
		fmt.Printf("   Error reading workflow file: %v\n", err)
		return false
	}

	// Parse workflow
	var workflow model.Workflow
	err = json.Unmarshal(data, &workflow)
	if err != nil {
		fmt.Printf("   Error parsing workflow JSON: %v\n", err)
		return false
	}

	// Create test input data for various node types
	testData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"firstName":  "John",
				"lastName":   "Doe",
				"age":        25,
				"email":      "john.doe@example.com",
				"items":      []interface{}{"apple", "banana", "cherry"},
				"numbers":    []interface{}{1, 2, 3, 4, 5},
				"price":      99.99,
				"quantity":   3,
				"jsonString": `{"name": "John", "age": 25, "city": "New York"}`,
				"emptyField": "",
			},
		},
		{
			JSON: map[string]interface{}{
				"firstName":  "Jane",
				"lastName":   "Smith",
				"age":        17,
				"email":      "jane.smith@example.com",
				"items":      []interface{}{"orange", "grape", "apple"},
				"numbers":    []interface{}{6, 7, 8, 9, 10},
				"price":      149.50,
				"quantity":   2,
				"jsonString": `{"name": "Jane", "age": 17, "city": "Boston"}`,
				"emptyField": nil,
			},
		},
	}

	// Use the provided workflow engine

	// Execute workflow
	fmt.Printf("   Executing workflow...\n")
	startTime := time.Now()

	result, err := workflowEngine.ExecuteWorkflow(&workflow, testData)

	executionTime := time.Since(startTime)
	fmt.Printf("   Execution time: %v\n", executionTime)

	if err != nil {
		fmt.Printf("   Workflow execution failed: %v\n", err)
		return false
	}

	if result == nil {
		fmt.Printf("   No result returned from workflow\n")
		return false
	}

	// Validate results
	fmt.Printf("   Validating results...\n")

	// Check that we got some output
	if len(result.Data) == 0 {
		fmt.Printf("   Warning: No output data from workflow\n")
	} else {
		fmt.Printf("   Output items: %d\n", len(result.Data))

		// Sample some output to verify expressions worked
		for i, item := range result.Data {
			if i >= 2 { // Only show first 2 items
				break
			}
			fmt.Printf("   Item %d keys: %v\n", i+1, getJSONKeys(item.JSON))
		}
	}

	// Performance validation
	if executionTime > 5*time.Second {
		fmt.Printf("   Warning: Execution took longer than expected (%v)\n", executionTime)
	}

	fmt.Printf("   Workflow executed successfully\n")
	return true
}

func getJSONKeys(data map[string]interface{}) []string {
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	return keys
}