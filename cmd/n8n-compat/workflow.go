package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/neul-labs/m9m/internal/compatibility"
	"github.com/neul-labs/m9m/internal/model"
	"github.com/spf13/cobra"
)

func createWorkflowCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workflow",
		Short: "Workflow import/export operations",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "import [n8n-workflow.json]",
			Short: "Import n8n workflow to n8n-go format",
			Args:  cobra.ExactArgs(1),
			Run:   importWorkflow,
		},
		&cobra.Command{
			Use:   "export [n8n-go-workflow.json]",
			Short: "Export n8n-go workflow to n8n format",
			Args:  cobra.ExactArgs(1),
			Run:   exportWorkflow,
		},
		&cobra.Command{
			Use:   "validate [workflow.json]",
			Short: "Validate n8n workflow format",
			Args:  cobra.ExactArgs(1),
			Run:   validateWorkflow,
		},
		&cobra.Command{
			Use:   "convert [input-dir] [output-dir]",
			Short: "Batch convert n8n workflows",
			Args:  cobra.ExactArgs(2),
			Run:   batchConvertWorkflows,
		},
	)

	return cmd
}

func importWorkflow(cmd *cobra.Command, args []string) {
	workflowFile := args[0]

	log.Printf("Importing n8n workflow: %s", workflowFile)

	data := mustReadFile(workflowFile, "workflow file")
	importer := compatibility.NewN8nWorkflowImporter()
	result, err := importer.ImportWorkflow(data)
	if err != nil {
		log.Fatalf("Failed to import workflow: %v", err)
	}

	if outputFormat == "json" {
		printJSON(result)
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

	outputFile := strings.TrimSuffix(workflowFile, filepath.Ext(workflowFile)) + "_converted.json"
	if err := ioutil.WriteFile(outputFile, marshalIndented(result.Workflow), 0644); err != nil {
		log.Printf("Warning: Failed to save converted workflow: %v", err)
	} else {
		log.Printf("💾 Converted workflow saved to: %s", outputFile)
	}
}

func exportWorkflow(cmd *cobra.Command, args []string) {
	workflowFile := args[0]

	log.Printf("Exporting n8n-go workflow: %s", workflowFile)

	data := mustReadFile(workflowFile, "workflow file")

	var workflow model.Workflow
	if err := json.Unmarshal(data, &workflow); err != nil {
		log.Fatalf("Failed to parse n8n-go workflow: %v", err)
	}

	importer := compatibility.NewN8nWorkflowImporter()
	exportedData, err := importer.ExportWorkflow(&workflow)
	if err != nil {
		log.Fatalf("Failed to export workflow: %v", err)
	}

	outputFile := strings.TrimSuffix(workflowFile, filepath.Ext(workflowFile)) + "_n8n.json"
	mustWriteFile(outputFile, exportedData, "exported workflow")
	fmt.Printf("✅ Successfully exported workflow to n8n format: %s\n", outputFile)
}

func validateWorkflow(cmd *cobra.Command, args []string) {
	workflowFile := args[0]

	data := mustReadFile(workflowFile, "workflow file")
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

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

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

		outputFile := filepath.Join(outputDir, filepath.Base(file))
		if err := ioutil.WriteFile(outputFile, marshalIndented(result.Workflow), 0644); err != nil {
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
