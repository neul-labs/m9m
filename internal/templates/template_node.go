package templates

import (
	"encoding/json"
	"fmt"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
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
		BaseNode: base.NewBaseNode(base.NodeDescription{
			Name:        "Template",
			Description: "Manage and install workflow templates",
			Category:    "templates",
		}),
		templateManager: NewTemplateManager("./templates"),
	}
}

// Execute implements the node interface
func (n *TemplateNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	operation, _ := nodeParams["operation"].(string)
	if operation == "" {
		operation = "list"
	}

	var results []model.DataItem

	for _, item := range inputData {
		result, err := n.executeOperation(operation, nodeParams, item)
		if err != nil {
			return nil, err
		}

		if result != nil {
			results = append(results, *result)
		}
	}

	return results, nil
}

func (n *TemplateNode) executeOperation(operation string, nodeParams map[string]interface{}, item model.DataItem) (*model.DataItem, error) {
	switch operation {
	case "list":
		return n.listTemplates(nodeParams, item)
	case "search":
		return n.searchTemplates(nodeParams, item)
	case "get":
		return n.getTemplate(nodeParams, item)
	case "install":
		return n.installTemplate(nodeParams, item)
	case "validate":
		return n.validateTemplate(nodeParams, item)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

func (n *TemplateNode) listTemplates(nodeParams map[string]interface{}, item model.DataItem) (*model.DataItem, error) {
	category, _ := nodeParams["category"].(string)
	limit := 50
	if l, ok := nodeParams["limit"].(float64); ok {
		limit = int(l)
	}

	var templates []*WorkflowTemplate
	var err error

	if category != "" {
		templates, err = n.templateManager.GetTemplatesByCategory(category)
	} else {
		templates, err = n.templateManager.SearchTemplates("", nil)
	}

	if err != nil {
		return nil, err
	}

	// Apply tag filtering if provided
	if tags, ok := nodeParams["tags"].([]interface{}); ok && len(tags) > 0 {
		tagStrings := make([]string, len(tags))
		for i, t := range tags {
			tagStrings[i], _ = t.(string)
		}
		templates = n.filterTemplatesByTags(templates, tagStrings)
	}

	if limit > 0 && len(templates) > limit {
		templates = templates[:limit]
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

func (n *TemplateNode) searchTemplates(nodeParams map[string]interface{}, item model.DataItem) (*model.DataItem, error) {
	query, _ := nodeParams["query"].(string)
	limit := 50
	if l, ok := nodeParams["limit"].(float64); ok {
		limit = int(l)
	}

	var filters map[string]interface{}
	if f, ok := nodeParams["filters"].(map[string]interface{}); ok {
		filters = f
	}

	templates, err := n.templateManager.SearchTemplates(query, filters)
	if err != nil {
		return nil, err
	}

	if limit > 0 && len(templates) > limit {
		templates = templates[:limit]
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
			"relevance":   1.0,
		}
	}

	result := model.DataItem{
		JSON: map[string]interface{}{
			"operation": "search",
			"query":     query,
			"total":     len(templateList),
			"templates": templateList,
		},
	}

	return &result, nil
}

func (n *TemplateNode) getTemplate(nodeParams map[string]interface{}, item model.DataItem) (*model.DataItem, error) {
	templateID, _ := nodeParams["templateId"].(string)
	if templateID == "" {
		if id, ok := item.JSON["template_id"].(string); ok {
			templateID = id
		}
	}
	if templateID == "" {
		return nil, fmt.Errorf("templateId is required")
	}

	template, err := n.templateManager.GetTemplate(templateID)
	if err != nil {
		return nil, err
	}

	result := model.DataItem{
		JSON: map[string]interface{}{
			"operation":   "get",
			"template_id": template.ID,
			"template":    template,
		},
	}

	return &result, nil
}

func (n *TemplateNode) installTemplate(nodeParams map[string]interface{}, item model.DataItem) (*model.DataItem, error) {
	templateID, _ := nodeParams["templateId"].(string)
	if templateID == "" {
		if id, ok := item.JSON["template_id"].(string); ok {
			templateID = id
		}
	}
	if templateID == "" {
		return nil, fmt.Errorf("templateId is required")
	}

	var parameters map[string]interface{}
	if p, ok := nodeParams["parameters"].(map[string]interface{}); ok {
		parameters = p
	}

	var options *InstallationOptions
	if o, ok := nodeParams["options"].(map[string]interface{}); ok {
		optData, _ := json.Marshal(o)
		options = &InstallationOptions{}
		json.Unmarshal(optData, options)
	}

	installResult, err := n.templateManager.InstallTemplate(templateID, parameters, options)
	if err != nil {
		return nil, err
	}

	result := model.DataItem{
		JSON: map[string]interface{}{
			"operation":   "install",
			"template_id": templateID,
			"success":     installResult.Success,
			"workflow_id": installResult.WorkflowID,
			"errors":      installResult.Errors,
			"warnings":    installResult.Warnings,
			"changes":     installResult.Changes,
			"metadata":    installResult.Metadata,
		},
	}

	return &result, nil
}

func (n *TemplateNode) validateTemplate(nodeParams map[string]interface{}, item model.DataItem) (*model.DataItem, error) {
	templateID, _ := nodeParams["templateId"].(string)
	if templateID == "" {
		if id, ok := item.JSON["template_id"].(string); ok {
			templateID = id
		}
	}
	if templateID == "" {
		return nil, fmt.Errorf("templateId is required")
	}

	var parameters map[string]interface{}
	if p, ok := nodeParams["parameters"].(map[string]interface{}); ok {
		parameters = p
	}

	template, err := n.templateManager.GetTemplate(templateID)
	if err != nil {
		return nil, err
	}

	validationErrors := []string{}
	validationWarnings := []string{}

	if err := n.templateManager.validateInstallationParameters(template, parameters); err != nil {
		validationErrors = append(validationErrors, err.Error())
	}

	for paramName, param := range template.Parameters {
		if param.Required {
			if _, provided := parameters[paramName]; !provided {
				validationErrors = append(validationErrors, fmt.Sprintf("Required parameter missing: %s", paramName))
			}
		}
	}

	isValid := len(validationErrors) == 0

	result := model.DataItem{
		JSON: map[string]interface{}{
			"operation":   "validate",
			"template_id": templateID,
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
