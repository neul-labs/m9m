package templates

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/n8n-go/n8n-go/internal/expressions"
	"github.com/n8n-go/n8n-go/pkg/model"
)

type TemplateManager struct {
	mu                sync.RWMutex
	templates         map[string]*WorkflowTemplate
	categories        map[string]*TemplateCategory
	marketplace       *MarketplaceManager
	templatePath      string
	evaluator         *expressions.GojaExpressionEvaluator
	validationRules   map[string]*ValidationRule
	installationQueue chan *InstallationRequest
}

type WorkflowTemplate struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Version      string                 `json:"version"`
	Author       string                 `json:"author"`
	Category     string                 `json:"category"`
	Tags         []string               `json:"tags"`
	Parameters   map[string]*Parameter  `json:"parameters"`
	Workflow     *model.Workflow        `json:"workflow"`
	Dependencies []string               `json:"dependencies"`
	Requirements *TemplateRequirements  `json:"requirements"`
	Metadata     *TemplateMetadata      `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

type Parameter struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	Description  string      `json:"description"`
	Required     bool        `json:"required"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	Options      []string    `json:"options,omitempty"`
	Validation   string      `json:"validation,omitempty"`
	Placeholder  string      `json:"placeholder,omitempty"`
	Group        string      `json:"group,omitempty"`
}

type TemplateRequirements struct {
	MinVersion     string            `json:"min_version"`
	RequiredNodes  []string          `json:"required_nodes"`
	OptionalNodes  []string          `json:"optional_nodes"`
	Credentials    []string          `json:"credentials"`
	Environment    map[string]string `json:"environment"`
	Permissions    []string          `json:"permissions"`
}

type TemplateMetadata struct {
	Downloads     int64             `json:"downloads"`
	Rating        float64           `json:"rating"`
	Reviews       int               `json:"reviews"`
	Screenshots   []string          `json:"screenshots"`
	Documentation string            `json:"documentation"`
	Repository    string            `json:"repository"`
	License       string            `json:"license"`
	Keywords      []string          `json:"keywords"`
	UseCases      []string          `json:"use_cases"`
	Industry      []string          `json:"industry"`
}

type TemplateCategory struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`
	Color       string   `json:"color"`
	ParentID    string   `json:"parent_id,omitempty"`
	Children    []string `json:"children,omitempty"`
	Templates   []string `json:"templates"`
}

type ValidationRule struct {
	Name        string `json:"name"`
	Expression  string `json:"expression"`
	Message     string `json:"message"`
	Severity    string `json:"severity"`
}

type InstallationRequest struct {
	TemplateID   string                 `json:"template_id"`
	Parameters   map[string]interface{} `json:"parameters"`
	UserID       string                 `json:"user_id"`
	ProjectID    string                 `json:"project_id"`
	Options      *InstallationOptions   `json:"options"`
	ResponseChan chan *InstallationResult
}

type InstallationOptions struct {
	ReplaceExisting bool              `json:"replace_existing"`
	Namespace       string            `json:"namespace"`
	Tags            []string          `json:"tags"`
	Environment     map[string]string `json:"environment"`
	DryRun          bool              `json:"dry_run"`
}

type InstallationResult struct {
	Success    bool              `json:"success"`
	WorkflowID string            `json:"workflow_id,omitempty"`
	Errors     []string          `json:"errors,omitempty"`
	Warnings   []string          `json:"warnings,omitempty"`
	Changes    []string          `json:"changes,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

func NewTemplateManager(templatePath string) *TemplateManager {
	tm := &TemplateManager{
		templates:         make(map[string]*WorkflowTemplate),
		categories:        make(map[string]*TemplateCategory),
		templatePath:      templatePath,
		evaluator:         expressions.NewGojaExpressionEvaluator(),
		validationRules:   make(map[string]*ValidationRule),
		installationQueue: make(chan *InstallationRequest, 100),
	}

	tm.marketplace = NewMarketplaceManager(tm)
	tm.initializeDefaultCategories()
	tm.initializeDefaultValidationRules()
	tm.startInstallationWorker()

	return tm
}

func (tm *TemplateManager) LoadTemplates() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, err := os.Stat(tm.templatePath); os.IsNotExist(err) {
		return fmt.Errorf("template path does not exist: %s", tm.templatePath)
	}

	return filepath.Walk(tm.templatePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasSuffix(path, ".json") && !info.IsDir() {
			template, err := tm.loadTemplateFromFile(path)
			if err != nil {
				return fmt.Errorf("failed to load template from %s: %w", path, err)
			}

			if template != nil {
				tm.templates[template.ID] = template
				tm.addTemplateToCategory(template)
			}
		}

		return nil
	})
}

func (tm *TemplateManager) loadTemplateFromFile(filePath string) (*WorkflowTemplate, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var template WorkflowTemplate
	if err := json.Unmarshal(data, &template); err != nil {
		return nil, err
	}

	if err := tm.validateTemplate(&template); err != nil {
		return nil, fmt.Errorf("template validation failed: %w", err)
	}

	return &template, nil
}

func (tm *TemplateManager) validateTemplate(template *WorkflowTemplate) error {
	if template.ID == "" {
		return fmt.Errorf("template ID is required")
	}

	if template.Name == "" {
		return fmt.Errorf("template name is required")
	}

	if template.Workflow == nil {
		return fmt.Errorf("template workflow is required")
	}

	for paramName, param := range template.Parameters {
		if err := tm.validateParameter(paramName, param); err != nil {
			return fmt.Errorf("parameter validation failed for %s: %w", paramName, err)
		}
	}

	for _, rule := range tm.validationRules {
		if err := tm.evaluateValidationRule(template, rule); err != nil {
			return fmt.Errorf("validation rule '%s' failed: %w", rule.Name, err)
		}
	}

	return nil
}

func (tm *TemplateManager) validateParameter(name string, param *Parameter) error {
	if param.Name == "" {
		param.Name = name
	}

	if param.Type == "" {
		return fmt.Errorf("parameter type is required")
	}

	validTypes := []string{"string", "number", "boolean", "array", "object", "select", "multiselect", "file", "credential", "expression"}
	isValidType := false
	for _, validType := range validTypes {
		if param.Type == validType {
			isValidType = true
			break
		}
	}

	if !isValidType {
		return fmt.Errorf("invalid parameter type: %s", param.Type)
	}

	if param.Type == "select" || param.Type == "multiselect" {
		if len(param.Options) == 0 {
			return fmt.Errorf("select/multiselect parameters must have options")
		}
	}

	if param.Validation != "" {
		context := expressions.NewExpressionContext()
		context.SetVariable("value", param.DefaultValue)

		result, err := tm.evaluator.Evaluate(param.Validation, context)
		if err != nil {
			return fmt.Errorf("parameter validation expression is invalid: %w", err)
		}

		if !tm.isTruthy(result) {
			return fmt.Errorf("parameter validation expression returned false")
		}
	}

	return nil
}

func (tm *TemplateManager) evaluateValidationRule(template *WorkflowTemplate, rule *ValidationRule) error {
	context := expressions.NewExpressionContext()
	context.SetVariable("template", template)

	result, err := tm.evaluator.Evaluate(rule.Expression, context)
	if err != nil {
		return err
	}

	if !tm.isTruthy(result) {
		if rule.Severity == "error" {
			return fmt.Errorf(rule.Message)
		}
	}

	return nil
}

func (tm *TemplateManager) isTruthy(value interface{}) bool {
	if value == nil {
		return false
	}

	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v != ""
	case int, int32, int64:
		return v != 0
	case float32, float64:
		return v != 0.0
	default:
		return true
	}
}

func (tm *TemplateManager) GetTemplate(id string) (*WorkflowTemplate, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	template, exists := tm.templates[id]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", id)
	}

	return template, nil
}

func (tm *TemplateManager) GetTemplatesByCategory(categoryID string) ([]*WorkflowTemplate, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	category, exists := tm.categories[categoryID]
	if !exists {
		return nil, fmt.Errorf("category not found: %s", categoryID)
	}

	var templates []*WorkflowTemplate
	for _, templateID := range category.Templates {
		if template, exists := tm.templates[templateID]; exists {
			templates = append(templates, template)
		}
	}

	return templates, nil
}

func (tm *TemplateManager) SearchTemplates(query string, filters map[string]interface{}) ([]*WorkflowTemplate, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var results []*WorkflowTemplate
	queryLower := strings.ToLower(query)

	for _, template := range tm.templates {
		if tm.matchesQuery(template, queryLower) && tm.matchesFilters(template, filters) {
			results = append(results, template)
		}
	}

	return results, nil
}

func (tm *TemplateManager) matchesQuery(template *WorkflowTemplate, query string) bool {
	if query == "" {
		return true
	}

	searchableText := strings.ToLower(fmt.Sprintf("%s %s %s %s",
		template.Name,
		template.Description,
		template.Author,
		strings.Join(template.Tags, " ")))

	return strings.Contains(searchableText, query)
}

func (tm *TemplateManager) matchesFilters(template *WorkflowTemplate, filters map[string]interface{}) bool {
	for key, value := range filters {
		switch key {
		case "category":
			if template.Category != value.(string) {
				return false
			}
		case "author":
			if template.Author != value.(string) {
				return false
			}
		case "tags":
			tags := value.([]string)
			hasAllTags := true
			for _, tag := range tags {
				found := false
				for _, templateTag := range template.Tags {
					if templateTag == tag {
						found = true
						break
					}
				}
				if !found {
					hasAllTags = false
					break
				}
			}
			if !hasAllTags {
				return false
			}
		case "min_rating":
			if template.Metadata.Rating < value.(float64) {
				return false
			}
		}
	}

	return true
}

func (tm *TemplateManager) InstallTemplate(templateID string, parameters map[string]interface{}, options *InstallationOptions) (*InstallationResult, error) {
	request := &InstallationRequest{
		TemplateID:   templateID,
		Parameters:   parameters,
		Options:      options,
		ResponseChan: make(chan *InstallationResult, 1),
	}

	select {
	case tm.installationQueue <- request:
		return <-request.ResponseChan, nil
	case <-time.After(30 * time.Second):
		return &InstallationResult{
			Success: false,
			Errors:  []string{"installation request timeout"},
		}, nil
	}
}

func (tm *TemplateManager) startInstallationWorker() {
	go func() {
		for request := range tm.installationQueue {
			result := tm.processInstallation(request)
			request.ResponseChan <- result
		}
	}()
}

func (tm *TemplateManager) processInstallation(request *InstallationRequest) *InstallationResult {
	result := &InstallationResult{
		Success:  false,
		Metadata: make(map[string]string),
	}

	template, err := tm.GetTemplate(request.TemplateID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Template not found: %s", err.Error()))
		return result
	}

	if err := tm.validateInstallationParameters(template, request.Parameters); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Parameter validation failed: %s", err.Error()))
		return result
	}

	workflow, err := tm.instantiateWorkflow(template, request.Parameters)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Workflow instantiation failed: %s", err.Error()))
		return result
	}

	if request.Options != nil && request.Options.DryRun {
		result.Success = true
		result.Changes = append(result.Changes, "Dry run completed successfully")
		result.Metadata["dry_run"] = "true"
		return result
	}

	workflowID := tm.generateWorkflowID(template, request.Parameters)
	workflow.ID = workflowID
	workflow.Name = tm.generateWorkflowName(template, request.Parameters)

	result.Success = true
	result.WorkflowID = workflowID
	result.Changes = append(result.Changes, fmt.Sprintf("Created workflow: %s", workflow.Name))

	return result
}

func (tm *TemplateManager) validateInstallationParameters(template *WorkflowTemplate, parameters map[string]interface{}) error {
	for paramName, param := range template.Parameters {
		value, provided := parameters[paramName]

		if param.Required && !provided {
			return fmt.Errorf("required parameter missing: %s", paramName)
		}

		if provided && param.Validation != "" {
			context := expressions.NewExpressionContext()
			context.SetVariable("value", value)

			result, err := tm.evaluator.Evaluate(param.Validation, context)
			if err != nil {
				return fmt.Errorf("parameter validation failed for %s: %w", paramName, err)
			}

			if !tm.isTruthy(result) {
				return fmt.Errorf("parameter validation failed for %s", paramName)
			}
		}
	}

	return nil
}

func (tm *TemplateManager) instantiateWorkflow(template *WorkflowTemplate, parameters map[string]interface{}) (*model.Workflow, error) {
	workflowData, err := json.Marshal(template.Workflow)
	if err != nil {
		return nil, err
	}

	workflowJSON := string(workflowData)

	context := expressions.NewExpressionContext()
	for key, value := range parameters {
		context.SetVariable(key, value)
	}

	for paramName, param := range template.Parameters {
		if _, provided := parameters[paramName]; !provided && param.DefaultValue != nil {
			context.SetVariable(paramName, param.DefaultValue)
		}
	}

	instantiatedJSON, err := tm.evaluator.EvaluateTemplateString(workflowJSON, context)
	if err != nil {
		return nil, err
	}

	var workflow model.Workflow
	if err := json.Unmarshal([]byte(instantiatedJSON), &workflow); err != nil {
		return nil, err
	}

	return &workflow, nil
}

func (tm *TemplateManager) generateWorkflowID(template *WorkflowTemplate, parameters map[string]interface{}) string {
	return fmt.Sprintf("%s_%d", template.ID, time.Now().Unix())
}

func (tm *TemplateManager) generateWorkflowName(template *WorkflowTemplate, parameters map[string]interface{}) string {
	if nameParam, exists := parameters["workflow_name"]; exists {
		return nameParam.(string)
	}
	return fmt.Sprintf("%s (from template)", template.Name)
}

func (tm *TemplateManager) addTemplateToCategory(template *WorkflowTemplate) {
	if category, exists := tm.categories[template.Category]; exists {
		for _, existingID := range category.Templates {
			if existingID == template.ID {
				return
			}
		}
		category.Templates = append(category.Templates, template.ID)
	}
}

func (tm *TemplateManager) initializeDefaultCategories() {
	defaultCategories := []*TemplateCategory{
		{
			ID:          "automation",
			Name:        "Automation",
			Description: "General automation workflows",
			Icon:        "🤖",
			Color:       "#FF6B6B",
		},
		{
			ID:          "data-processing",
			Name:        "Data Processing",
			Description: "Data transformation and processing workflows",
			Icon:        "📊",
			Color:       "#4ECDC4",
		},
		{
			ID:          "integrations",
			Name:        "Integrations",
			Description: "Third-party service integrations",
			Icon:        "🔗",
			Color:       "#45B7D1",
		},
		{
			ID:          "monitoring",
			Name:        "Monitoring",
			Description: "System and application monitoring",
			Icon:        "📡",
			Color:       "#96CEB4",
		},
		{
			ID:          "notifications",
			Name:        "Notifications",
			Description: "Alert and notification workflows",
			Icon:        "🔔",
			Color:       "#FECA57",
		},
	}

	for _, category := range defaultCategories {
		tm.categories[category.ID] = category
	}
}

func (tm *TemplateManager) initializeDefaultValidationRules() {
	tm.validationRules["require_description"] = &ValidationRule{
		Name:       "require_description",
		Expression: "template.description && template.description.length > 10",
		Message:    "Template description must be at least 10 characters long",
		Severity:   "warning",
	}

	tm.validationRules["require_author"] = &ValidationRule{
		Name:       "require_author",
		Expression: "template.author && template.author.length > 0",
		Message:    "Template must have an author",
		Severity:   "error",
	}

	tm.validationRules["version_format"] = &ValidationRule{
		Name:       "version_format",
		Expression: "template.version && /^\\d+\\.\\d+\\.\\d+$/.test(template.version)",
		Message:    "Template version must follow semantic versioning (x.y.z)",
		Severity:   "warning",
	}
}