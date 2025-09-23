package templates

import (
	"encoding/json"
	"fmt"

	"github.com/n8n-go/n8n-go/internal/nodes/base"
	"github.com/n8n-go/n8n-go/pkg/model"
)

type TemplateNode struct {
	*base.BaseNode
	templateManager *TemplateManager
}

type TemplateNodeConfig struct {
	Operation  string                 `json:"operation"`
	Parameters map[string]interface{} `json:"parameters"`
}

type TemplateListOperation struct {
	Category string                 `json:"category,omitempty"`
	Tags     []string               `json:"tags,omitempty"`
	Filters  map[string]interface{} `json:"filters,omitempty"`
	Limit    int                    `json:"limit,omitempty"`
}

type TemplateSearchOperation struct {
	Query   string                 `json:"query"`
	Filters map[string]interface{} `json:"filters,omitempty"`
	Limit   int                    `json:"limit,omitempty"`
}

type TemplateInstallOperation struct {
	TemplateID string                 `json:"template_id"`
	Parameters map[string]interface{} `json:"parameters"`
	Options    *InstallationOptions   `json:"options,omitempty"`
}

type TemplateValidateOperation struct {
	TemplateID string                 `json:"template_id"`
	Parameters map[string]interface{} `json:"parameters"`
}

type TemplateGetOperation struct {
	TemplateID string `json:"template_id"`
}

func NewTemplateNode() *TemplateNode {
	return &TemplateNode{
		BaseNode:        base.NewBaseNode("Template", "Manage workflow templates", "1.0.0"),
		templateManager: NewTemplateManager("./templates"),
	}
}

func (n *TemplateNode) Execute(input *model.NodeExecutionInput) (*model.NodeExecutionOutput, error) {
	var config TemplateNodeConfig
	if err := json.Unmarshal(input.Config, &config); err != nil {
		return nil, fmt.Errorf("failed to parse template config: %w", err)
	}

	var results []model.DataItem

	for _, item := range input.Items {
		result, err := n.executeOperation(&config, item)
		if err != nil {
			return &model.NodeExecutionOutput{
				Items: []model.DataItem{},
				Error: err.Error(),
			}, nil
		}

		if result != nil {
			results = append(results, *result)
		}
	}

	return &model.NodeExecutionOutput{
		Items: results,
	}, nil
}

func (n *TemplateNode) executeOperation(config *TemplateNodeConfig, item model.DataItem) (*model.DataItem, error) {
	switch config.Operation {
	case "list":
		return n.listTemplates(config, item)
	case "search":
		return n.searchTemplates(config, item)
	case "get":
		return n.getTemplate(config, item)
	case "install":
		return n.installTemplate(config, item)
	case "validate":
		return n.validateTemplate(config, item)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", config.Operation)
	}
}

func (n *TemplateNode) listTemplates(config *TemplateNodeConfig, item model.DataItem) (*model.DataItem, error) {
	var operation TemplateListOperation
	if err := n.mapParameters(config.Parameters, &operation); err != nil {
		return nil, err
	}

	var templates []*WorkflowTemplate
	var err error

	if operation.Category != "" {
		templates, err = n.templateManager.GetTemplatesByCategory(operation.Category)
	} else {
		templates, err = n.templateManager.SearchTemplates("", operation.Filters)
	}

	if err != nil {
		return nil, err
	}

	if len(operation.Tags) > 0 {
		templates = n.filterTemplatesByTags(templates, operation.Tags)
	}

	if operation.Limit > 0 && len(templates) > operation.Limit {
		templates = templates[:operation.Limit]
	}

	templateList := make([]interface{}, len(templates))
	for i, template := range templates {
		templateList[i] = map[string]interface{}{
			"id":           template.ID,
			"name":         template.Name,
			"description":  template.Description,
			"version":      template.Version,
			"author":       template.Author,
			"category":     template.Category,
			"tags":         template.Tags,
			"created_at":   template.CreatedAt,
			"updated_at":   template.UpdatedAt,
			"parameters":   template.Parameters,
			"requirements": template.Requirements,
			"metadata":     template.Metadata,
		}
	}

	result := model.DataItem{
		JSON: map[string]interface{}{
			"operation": "list",
			"total":     len(templateList),
			"templates": templateList,
		},
	}

	return &result, nil
}

func (n *TemplateNode) searchTemplates(config *TemplateNodeConfig, item model.DataItem) (*model.DataItem, error) {
	var operation TemplateSearchOperation
	if err := n.mapParameters(config.Parameters, &operation); err != nil {
		return nil, err
	}

	templates, err := n.templateManager.SearchTemplates(operation.Query, operation.Filters)
	if err != nil {
		return nil, err
	}

	if operation.Limit > 0 && len(templates) > operation.Limit {
		templates = templates[:operation.Limit]
	}

	templateList := make([]interface{}, len(templates))
	for i, template := range templates {
		templateList[i] = map[string]interface{}{
			"id":          template.ID,
			"name":        template.Name,
			"description": template.Description,
			"version":     template.Version,
			"author":      template.Author,
			"category":    template.Category,
			"tags":        template.Tags,
			"relevance":   1.0, // TODO: Implement relevance scoring
		}
	}

	result := model.DataItem{
		JSON: map[string]interface{}{
			"operation": "search",
			"query":     operation.Query,
			"total":     len(templateList),
			"templates": templateList,
		},
	}

	return &result, nil
}

func (n *TemplateNode) getTemplate(config *TemplateNodeConfig, item model.DataItem) (*model.DataItem, error) {
	var operation TemplateGetOperation
	if err := n.mapParameters(config.Parameters, &operation); err != nil {
		return nil, err
	}

	template, err := n.templateManager.GetTemplate(operation.TemplateID)
	if err != nil {
		return nil, err
	}

	result := model.DataItem{
		JSON: map[string]interface{}{
			"operation":    "get",
			"template_id":  template.ID,
			"template":     template,
		},
	}

	return &result, nil
}

func (n *TemplateNode) installTemplate(config *TemplateNodeConfig, item model.DataItem) (*model.DataItem, error) {
	var operation TemplateInstallOperation
	if err := n.mapParameters(config.Parameters, &operation); err != nil {
		return nil, err
	}

	installResult, err := n.templateManager.InstallTemplate(
		operation.TemplateID,
		operation.Parameters,
		operation.Options,
	)
	if err != nil {
		return nil, err
	}

	result := model.DataItem{
		JSON: map[string]interface{}{
			"operation":     "install",
			"template_id":   operation.TemplateID,
			"success":       installResult.Success,
			"workflow_id":   installResult.WorkflowID,
			"errors":        installResult.Errors,
			"warnings":      installResult.Warnings,
			"changes":       installResult.Changes,
			"metadata":      installResult.Metadata,
		},
	}

	return &result, nil
}

func (n *TemplateNode) validateTemplate(config *TemplateNodeConfig, item model.DataItem) (*model.DataItem, error) {
	var operation TemplateValidateOperation
	if err := n.mapParameters(config.Parameters, &operation); err != nil {
		return nil, err
	}

	template, err := n.templateManager.GetTemplate(operation.TemplateID)
	if err != nil {
		return nil, err
	}

	validationErrors := []string{}
	validationWarnings := []string{}

	if err := n.templateManager.validateInstallationParameters(template, operation.Parameters); err != nil {
		validationErrors = append(validationErrors, err.Error())
	}

	for paramName, param := range template.Parameters {
		if param.Required {
			if _, provided := operation.Parameters[paramName]; !provided {
				validationErrors = append(validationErrors, fmt.Sprintf("Required parameter missing: %s", paramName))
			}
		}
	}

	isValid := len(validationErrors) == 0

	result := model.DataItem{
		JSON: map[string]interface{}{
			"operation":   "validate",
			"template_id": operation.TemplateID,
			"valid":       isValid,
			"errors":      validationErrors,
			"warnings":    validationWarnings,
			"parameters":  template.Parameters,
		},
	}

	return &result, nil
}

func (n *TemplateNode) filterTemplatesByTags(templates []*WorkflowTemplate, tags []string) []*WorkflowTemplate {
	var filtered []*WorkflowTemplate

	for _, template := range templates {
		hasAllTags := true
		for _, requiredTag := range tags {
			found := false
			for _, templateTag := range template.Tags {
				if templateTag == requiredTag {
					found = true
					break
				}
			}
			if !found {
				hasAllTags = false
				break
			}
		}

		if hasAllTags {
			filtered = append(filtered, template)
		}
	}

	return filtered
}

func (n *TemplateNode) mapParameters(source map[string]interface{}, target interface{}) error {
	data, err := json.Marshal(source)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, target)
}

func (n *TemplateNode) GetNodeDefinition() *model.NodeDefinition {
	return &model.NodeDefinition{
		Name:        "Template",
		DisplayName: "Template Manager",
		Description: "Manage and install workflow templates",
		Version:     "1.0.0",
		Category:    "templates",
		Icon:        "template",
		Color:       "#9CA3AF",
		Properties: []model.NodeProperty{
			{
				Name:        "operation",
				DisplayName: "Operation",
				Type:        "select",
				Required:    true,
				Default:     "list",
				Options: []model.NodePropertyOption{
					{Name: "List Templates", Value: "list"},
					{Name: "Search Templates", Value: "search"},
					{Name: "Get Template", Value: "get"},
					{Name: "Install Template", Value: "install"},
					{Name: "Validate Template", Value: "validate"},
				},
				Description: "The operation to perform with templates",
			},
			{
				Name:        "templateId",
				DisplayName: "Template ID",
				Type:        "string",
				Required:    false,
				Description: "The ID of the template to work with",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"get", "install", "validate"},
					},
				},
			},
			{
				Name:        "query",
				DisplayName: "Search Query",
				Type:        "string",
				Required:    false,
				Description: "Search query for finding templates",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"search"},
					},
				},
			},
			{
				Name:        "category",
				DisplayName: "Category",
				Type:        "select",
				Required:    false,
				Options: []model.NodePropertyOption{
					{Name: "Automation", Value: "automation"},
					{Name: "Data Processing", Value: "data-processing"},
					{Name: "Integrations", Value: "integrations"},
					{Name: "Monitoring", Value: "monitoring"},
					{Name: "Notifications", Value: "notifications"},
				},
				Description: "Filter templates by category",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"list"},
					},
				},
			},
			{
				Name:        "parameters",
				DisplayName: "Template Parameters",
				Type:        "json",
				Required:    false,
				Description: "Parameters for template installation or validation",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"install", "validate"},
					},
				},
			},
			{
				Name:        "limit",
				DisplayName: "Limit",
				Type:        "number",
				Required:    false,
				Default:     50,
				Description: "Maximum number of templates to return",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"list", "search"},
					},
				},
			},
		},
		Inputs:  []string{"main"},
		Outputs: []string{"main"},
	}
}