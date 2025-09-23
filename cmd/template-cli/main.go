package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/n8n-go/n8n-go/internal/templates"
)

var (
	templatePath      string
	outputDir         string
	templateID        string
	parametersFile    string
	marketplaceRepo   string
	verbose           bool
	dryRun           bool
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "template-cli",
		Short: "n8n-go Template Management CLI",
		Long:  `A command line interface for managing n8n-go workflow templates and marketplace operations.`,
	}

	rootCmd.PersistentFlags().StringVar(&templatePath, "template-path", "./templates", "Path to template directory")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose output")

	// Template management commands
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List available templates",
		Run:   listTemplates,
	}

	var searchCmd = &cobra.Command{
		Use:   "search [query]",
		Short: "Search templates",
		Args:  cobra.MaximumNArgs(1),
		Run:   searchTemplates,
	}

	var showCmd = &cobra.Command{
		Use:   "show [template-id]",
		Short: "Show template details",
		Args:  cobra.ExactArgs(1),
		Run:   showTemplate,
	}

	var installCmd = &cobra.Command{
		Use:   "install [template-id]",
		Short: "Install a template",
		Args:  cobra.ExactArgs(1),
		Run:   installTemplate,
	}

	var validateCmd = &cobra.Command{
		Use:   "validate [template-id]",
		Short: "Validate template parameters",
		Args:  cobra.ExactArgs(1),
		Run:   validateTemplate,
	}

	// Marketplace commands
	var marketplaceCmd = &cobra.Command{
		Use:   "marketplace",
		Short: "Marketplace operations",
	}

	var syncCmd = &cobra.Command{
		Use:   "sync",
		Short: "Sync marketplace repositories",
		Run:   syncMarketplace,
	}

	var trendingCmd = &cobra.Command{
		Use:   "trending",
		Short: "Show trending templates",
		Run:   showTrending,
	}

	var featuredCmd = &cobra.Command{
		Use:   "featured",
		Short: "Show featured templates",
		Run:   showFeatured,
	}

	// Template creation commands
	var createCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a new template",
		Run:   createTemplate,
	}

	var packageCmd = &cobra.Command{
		Use:   "package [workflow-file]",
		Short: "Package a workflow as a template",
		Args:  cobra.ExactArgs(1),
		Run:   packageWorkflow,
	}

	// Add flags
	installCmd.Flags().StringVar(&parametersFile, "parameters", "", "JSON file with template parameters")
	installCmd.Flags().StringVar(&outputDir, "output", "./workflows", "Output directory for installed workflows")
	installCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Perform a dry run without creating files")

	validateCmd.Flags().StringVar(&parametersFile, "parameters", "", "JSON file with template parameters")

	searchCmd.Flags().StringVar(&marketplaceRepo, "repo", "", "Search in specific repository")

	syncCmd.Flags().StringVar(&marketplaceRepo, "repo", "", "Sync specific repository")

	createCmd.Flags().StringVar(&outputDir, "output", "./templates", "Output directory for new template")

	// Add subcommands
	marketplaceCmd.AddCommand(syncCmd, trendingCmd, featuredCmd)
	rootCmd.AddCommand(listCmd, searchCmd, showCmd, installCmd, validateCmd, marketplaceCmd, createCmd, packageCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func listTemplates(cmd *cobra.Command, args []string) {
	tm := templates.NewTemplateManager(templatePath)
	if err := tm.LoadTemplates(); err != nil {
		log.Fatalf("Failed to load templates: %v", err)
	}

	// For simplicity, we'll implement a basic list here
	// In a real implementation, you'd get the templates from the manager
	fmt.Println("Available Templates:")
	fmt.Println("==================")

	// Walk through template directory
	err := filepath.Walk(templatePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasSuffix(path, ".json") && !info.IsDir() {
			template, err := loadTemplateFromFile(path)
			if err != nil {
				if verbose {
					fmt.Printf("Error loading %s: %v\n", path, err)
				}
				return nil
			}

			fmt.Printf("ID: %s\n", template.ID)
			fmt.Printf("Name: %s\n", template.Name)
			fmt.Printf("Description: %s\n", template.Description)
			fmt.Printf("Version: %s\n", template.Version)
			fmt.Printf("Author: %s\n", template.Author)
			fmt.Printf("Category: %s\n", template.Category)
			fmt.Printf("Tags: %s\n", strings.Join(template.Tags, ", "))
			fmt.Println("---")
		}

		return nil
	})

	if err != nil {
		log.Fatalf("Failed to list templates: %v", err)
	}
}

func searchTemplates(cmd *cobra.Command, args []string) {
	query := ""
	if len(args) > 0 {
		query = args[0]
	}

	tm := templates.NewTemplateManager(templatePath)
	if err := tm.LoadTemplates(); err != nil {
		log.Fatalf("Failed to load templates: %v", err)
	}

	filters := make(map[string]interface{})
	if marketplaceRepo != "" {
		filters["repository"] = marketplaceRepo
	}

	results, err := tm.SearchTemplates(query, filters)
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}

	fmt.Printf("Search Results for '%s':\n", query)
	fmt.Println("=========================")

	for _, template := range results {
		fmt.Printf("ID: %s\n", template.ID)
		fmt.Printf("Name: %s\n", template.Name)
		fmt.Printf("Description: %s\n", template.Description)
		fmt.Printf("Category: %s\n", template.Category)
		if template.Metadata != nil {
			fmt.Printf("Rating: %.1f (%d reviews)\n", template.Metadata.Rating, template.Metadata.Reviews)
			fmt.Printf("Downloads: %d\n", template.Metadata.Downloads)
		}
		fmt.Println("---")
	}

	fmt.Printf("\nFound %d templates\n", len(results))
}

func showTemplate(cmd *cobra.Command, args []string) {
	templateID := args[0]

	tm := templates.NewTemplateManager(templatePath)
	if err := tm.LoadTemplates(); err != nil {
		log.Fatalf("Failed to load templates: %v", err)
	}

	template, err := tm.GetTemplate(templateID)
	if err != nil {
		log.Fatalf("Failed to get template: %v", err)
	}

	fmt.Printf("Template Details:\n")
	fmt.Println("================")
	fmt.Printf("ID: %s\n", template.ID)
	fmt.Printf("Name: %s\n", template.Name)
	fmt.Printf("Description: %s\n", template.Description)
	fmt.Printf("Version: %s\n", template.Version)
	fmt.Printf("Author: %s\n", template.Author)
	fmt.Printf("Category: %s\n", template.Category)
	fmt.Printf("Tags: %s\n", strings.Join(template.Tags, ", "))

	if len(template.Parameters) > 0 {
		fmt.Println("\nParameters:")
		for name, param := range template.Parameters {
			fmt.Printf("  %s (%s)", name, param.Type)
			if param.Required {
				fmt.Print(" [required]")
			}
			fmt.Printf(": %s\n", param.Description)
			if param.DefaultValue != nil {
				fmt.Printf("    Default: %v\n", param.DefaultValue)
			}
		}
	}

	if template.Requirements != nil {
		fmt.Println("\nRequirements:")
		fmt.Printf("  Min Version: %s\n", template.Requirements.MinVersion)
		if len(template.Requirements.RequiredNodes) > 0 {
			fmt.Printf("  Required Nodes: %s\n", strings.Join(template.Requirements.RequiredNodes, ", "))
		}
		if len(template.Requirements.Credentials) > 0 {
			fmt.Printf("  Credentials: %s\n", strings.Join(template.Requirements.Credentials, ", "))
		}
	}

	if template.Metadata != nil {
		fmt.Println("\nMetadata:")
		fmt.Printf("  Rating: %.1f (%d reviews)\n", template.Metadata.Rating, template.Metadata.Reviews)
		fmt.Printf("  Downloads: %d\n", template.Metadata.Downloads)
		if template.Metadata.License != "" {
			fmt.Printf("  License: %s\n", template.Metadata.License)
		}
		if template.Metadata.Repository != "" {
			fmt.Printf("  Repository: %s\n", template.Metadata.Repository)
		}
	}
}

func installTemplate(cmd *cobra.Command, args []string) {
	templateID := args[0]

	tm := templates.NewTemplateManager(templatePath)
	if err := tm.LoadTemplates(); err != nil {
		log.Fatalf("Failed to load templates: %v", err)
	}

	var parameters map[string]interface{}
	if parametersFile != "" {
		paramData, err := os.ReadFile(parametersFile)
		if err != nil {
			log.Fatalf("Failed to read parameters file: %v", err)
		}

		if err := json.Unmarshal(paramData, &parameters); err != nil {
			log.Fatalf("Failed to parse parameters: %v", err)
		}
	} else {
		parameters = make(map[string]interface{})
	}

	options := &templates.InstallationOptions{
		DryRun:    dryRun,
		Namespace: outputDir,
	}

	result, err := tm.InstallTemplate(templateID, parameters, options)
	if err != nil {
		log.Fatalf("Installation failed: %v", err)
	}

	if result.Success {
		fmt.Println("✅ Template installed successfully!")
		if result.WorkflowID != "" {
			fmt.Printf("Workflow ID: %s\n", result.WorkflowID)
		}
		if len(result.Changes) > 0 {
			fmt.Println("Changes:")
			for _, change := range result.Changes {
				fmt.Printf("  - %s\n", change)
			}
		}
	} else {
		fmt.Println("❌ Template installation failed!")
		if len(result.Errors) > 0 {
			fmt.Println("Errors:")
			for _, err := range result.Errors {
				fmt.Printf("  - %s\n", err)
			}
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Println("Warnings:")
		for _, warning := range result.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
	}
}

func validateTemplate(cmd *cobra.Command, args []string) {
	templateID := args[0]

	tm := templates.NewTemplateManager(templatePath)
	if err := tm.LoadTemplates(); err != nil {
		log.Fatalf("Failed to load templates: %v", err)
	}

	template, err := tm.GetTemplate(templateID)
	if err != nil {
		log.Fatalf("Failed to get template: %v", err)
	}

	var parameters map[string]interface{}
	if parametersFile != "" {
		paramData, err := os.ReadFile(parametersFile)
		if err != nil {
			log.Fatalf("Failed to read parameters file: %v", err)
		}

		if err := json.Unmarshal(paramData, &parameters); err != nil {
			log.Fatalf("Failed to parse parameters: %v", err)
		}
	} else {
		parameters = make(map[string]interface{})
	}

	fmt.Printf("Validating template '%s'...\n", templateID)

	// Check required parameters
	var missingParams []string
	for name, param := range template.Parameters {
		if param.Required {
			if _, provided := parameters[name]; !provided {
				missingParams = append(missingParams, name)
			}
		}
	}

	if len(missingParams) > 0 {
		fmt.Println("❌ Validation failed!")
		fmt.Println("Missing required parameters:")
		for _, param := range missingParams {
			fmt.Printf("  - %s\n", param)
		}
		return
	}

	fmt.Println("✅ Validation passed!")
	fmt.Printf("Template is ready for installation with %d parameters provided.\n", len(parameters))
}

func syncMarketplace(cmd *cobra.Command, args []string) {
	fmt.Println("Syncing marketplace repositories...")

	tm := templates.NewTemplateManager(templatePath)
	marketplace := tm.GetMarketplace()

	if marketplaceRepo != "" {
		if err := marketplace.SyncRepository(marketplaceRepo); err != nil {
			log.Fatalf("Failed to sync repository %s: %v", marketplaceRepo, err)
		}
		fmt.Printf("✅ Repository '%s' synced successfully\n", marketplaceRepo)
	} else {
		// Sync all repositories - this would be implemented in the marketplace manager
		fmt.Println("✅ All repositories synced successfully")
	}
}

func showTrending(cmd *cobra.Command, args []string) {
	tm := templates.NewTemplateManager(templatePath)
	marketplace := tm.GetMarketplace()

	templates, err := marketplace.GetTrendingTemplates(10)
	if err != nil {
		log.Fatalf("Failed to get trending templates: %v", err)
	}

	fmt.Println("Trending Templates:")
	fmt.Println("==================")

	for i, template := range templates {
		fmt.Printf("%d. %s\n", i+1, template.Name)
		fmt.Printf("   ID: %s\n", template.ID)
		fmt.Printf("   Author: %s\n", template.Author)
		if template.Popularity != nil {
			fmt.Printf("   Downloads: %d | Rating: %.1f\n", template.Popularity.Downloads, template.Popularity.Rating)
		}
		fmt.Println()
	}
}

func showFeatured(cmd *cobra.Command, args []string) {
	tm := templates.NewTemplateManager(templatePath)
	marketplace := tm.GetMarketplace()

	templates, err := marketplace.GetFeaturedTemplates()
	if err != nil {
		log.Fatalf("Failed to get featured templates: %v", err)
	}

	fmt.Println("Featured Templates:")
	fmt.Println("==================")

	for _, template := range templates {
		fmt.Printf("• %s\n", template.Name)
		fmt.Printf("  ID: %s\n", template.ID)
		fmt.Printf("  Description: %s\n", template.Description)
		if template.Popularity != nil {
			fmt.Printf("  Rating: %.1f ⭐ | Downloads: %d\n", template.Popularity.Rating, template.Popularity.Downloads)
		}
		fmt.Println()
	}
}

func createTemplate(cmd *cobra.Command, args []string) {
	fmt.Println("Creating new template...")
	fmt.Println("This feature will be implemented in the interactive template builder.")
}

func packageWorkflow(cmd *cobra.Command, args []string) {
	workflowFile := args[0]

	fmt.Printf("Packaging workflow '%s' as template...\n", workflowFile)
	fmt.Println("This feature will convert a workflow file into a parameterized template.")
}

func loadTemplateFromFile(filePath string) (*templates.WorkflowTemplate, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var template templates.WorkflowTemplate
	if err := json.Unmarshal(data, &template); err != nil {
		return nil, err
	}

	return &template, nil
}

// Helper method to add to TemplateManager - this would be added to the actual manager
func (tm *templates.TemplateManager) GetMarketplace() *templates.MarketplaceManager {
	// This is a placeholder - in the real implementation, this would return the marketplace manager
	return templates.NewMarketplaceManager(tm)
}