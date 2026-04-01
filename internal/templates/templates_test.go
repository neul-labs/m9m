package templates

import (
	"fmt"
	"testing"
	"time"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// Helper functions for building test objects
// ============================================================

func newTestTemplateManager() *TemplateManager {
	return NewTemplateManager("/tmp/m9m-test-templates")
}

func newMinimalWorkflowTemplate(id, name, author string) *WorkflowTemplate {
	return &WorkflowTemplate{
		ID:          id,
		Name:        name,
		Description: "A test template with enough length",
		Version:     "1.0.0",
		Author:      author,
		Category:    "automation",
		Tags:        []string{"test"},
		Parameters:  map[string]*Parameter{},
		Workflow:    &model.Workflow{ID: "wf-1", Name: "Test Workflow"},
		Metadata: &TemplateMetadata{
			Rating:   4.0,
			Keywords: []string{"test"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func seedTemplate(tm *TemplateManager, t *WorkflowTemplate) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.templates[t.ID] = t
	tm.addTemplateToCategory(t)
}

// ============================================================
// TemplateManager constructor tests
// ============================================================

func TestNewTemplateManager(t *testing.T) {
	tm := newTestTemplateManager()
	require.NotNil(t, tm)
	assert.NotNil(t, tm.templates)
	assert.NotNil(t, tm.categories)
	assert.NotNil(t, tm.evaluator)
	assert.NotNil(t, tm.validationRules)
	assert.NotNil(t, tm.installationQueue)
}

func TestNewTemplateManager_DefaultCategories(t *testing.T) {
	tm := newTestTemplateManager()
	expectedCategories := []string{"automation", "data-processing", "integrations", "monitoring", "notifications"}
	for _, catID := range expectedCategories {
		cat, exists := tm.categories[catID]
		assert.True(t, exists, "expected category %q to be initialized", catID)
		assert.NotEmpty(t, cat.Name)
	}
}

func TestNewTemplateManager_DefaultValidationRules(t *testing.T) {
	tm := newTestTemplateManager()
	expectedRules := []string{"require_description", "require_author", "version_format"}
	for _, rule := range expectedRules {
		_, exists := tm.validationRules[rule]
		assert.True(t, exists, "expected validation rule %q to be initialized", rule)
	}
}

func TestNewTemplateManager_MarketplaceCreated(t *testing.T) {
	tm := newTestTemplateManager()
	assert.NotNil(t, tm.GetMarketplace())
}

// ============================================================
// GetTemplate tests
// ============================================================

func TestGetTemplate_Success(t *testing.T) {
	tm := newTestTemplateManager()
	tpl := newMinimalWorkflowTemplate("tpl-1", "Template One", "tester")
	seedTemplate(tm, tpl)

	result, err := tm.GetTemplate("tpl-1")
	require.NoError(t, err)
	assert.Equal(t, "tpl-1", result.ID)
	assert.Equal(t, "Template One", result.Name)
}

func TestGetTemplate_NotFound(t *testing.T) {
	tm := newTestTemplateManager()

	result, err := tm.GetTemplate("nonexistent")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template not found")
}

// ============================================================
// GetTemplatesByCategory tests
// ============================================================

func TestGetTemplatesByCategory_Success(t *testing.T) {
	tm := newTestTemplateManager()
	tpl := newMinimalWorkflowTemplate("tpl-1", "Template One", "tester")
	tpl.Category = "automation"
	seedTemplate(tm, tpl)

	results, err := tm.GetTemplatesByCategory("automation")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "tpl-1", results[0].ID)
}

func TestGetTemplatesByCategory_CategoryNotFound(t *testing.T) {
	tm := newTestTemplateManager()

	results, err := tm.GetTemplatesByCategory("nonexistent-cat")
	assert.Nil(t, results)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "category not found")
}

func TestGetTemplatesByCategory_Empty(t *testing.T) {
	tm := newTestTemplateManager()

	results, err := tm.GetTemplatesByCategory("automation")
	require.NoError(t, err)
	assert.Empty(t, results)
}

// ============================================================
// SearchTemplates tests
// ============================================================

func TestSearchTemplates_EmptyQuery(t *testing.T) {
	tm := newTestTemplateManager()
	seedTemplate(tm, newMinimalWorkflowTemplate("tpl-1", "First", "alice"))
	seedTemplate(tm, newMinimalWorkflowTemplate("tpl-2", "Second", "bob"))

	results, err := tm.SearchTemplates("", nil)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestSearchTemplates_ByName(t *testing.T) {
	tm := newTestTemplateManager()
	seedTemplate(tm, newMinimalWorkflowTemplate("tpl-1", "Email Notifier", "alice"))
	seedTemplate(tm, newMinimalWorkflowTemplate("tpl-2", "Data Transformer", "bob"))

	results, err := tm.SearchTemplates("email", nil)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "tpl-1", results[0].ID)
}

func TestSearchTemplates_CaseInsensitive(t *testing.T) {
	tm := newTestTemplateManager()
	seedTemplate(tm, newMinimalWorkflowTemplate("tpl-1", "Email Notifier", "alice"))

	results, err := tm.SearchTemplates("EMAIL", nil)
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestSearchTemplates_ByDescription(t *testing.T) {
	tm := newTestTemplateManager()
	tpl := newMinimalWorkflowTemplate("tpl-1", "Notifier", "alice")
	tpl.Description = "Sends important alerts to Slack channels"
	seedTemplate(tm, tpl)

	results, err := tm.SearchTemplates("slack", nil)
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestSearchTemplates_ByAuthor(t *testing.T) {
	tm := newTestTemplateManager()
	seedTemplate(tm, newMinimalWorkflowTemplate("tpl-1", "Notifier", "alice"))
	seedTemplate(tm, newMinimalWorkflowTemplate("tpl-2", "Transformer", "bob"))

	results, err := tm.SearchTemplates("alice", nil)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "tpl-1", results[0].ID)
}

func TestSearchTemplates_ByTag(t *testing.T) {
	tm := newTestTemplateManager()
	tpl := newMinimalWorkflowTemplate("tpl-1", "Notifier", "alice")
	tpl.Tags = []string{"messaging", "alert"}
	seedTemplate(tm, tpl)

	results, err := tm.SearchTemplates("messaging", nil)
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestSearchTemplates_NoResults(t *testing.T) {
	tm := newTestTemplateManager()
	seedTemplate(tm, newMinimalWorkflowTemplate("tpl-1", "Notifier", "alice"))

	results, err := tm.SearchTemplates("zzzznonexistent", nil)
	require.NoError(t, err)
	assert.Empty(t, results)
}

// ============================================================
// SearchTemplates filter tests
// ============================================================

func TestSearchTemplates_FilterByCategory(t *testing.T) {
	tm := newTestTemplateManager()
	tpl1 := newMinimalWorkflowTemplate("tpl-1", "One", "alice")
	tpl1.Category = "automation"
	tpl2 := newMinimalWorkflowTemplate("tpl-2", "Two", "bob")
	tpl2.Category = "monitoring"
	seedTemplate(tm, tpl1)
	seedTemplate(tm, tpl2)

	results, err := tm.SearchTemplates("", map[string]interface{}{"category": "automation"})
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "tpl-1", results[0].ID)
}

func TestSearchTemplates_FilterByAuthor(t *testing.T) {
	tm := newTestTemplateManager()
	seedTemplate(tm, newMinimalWorkflowTemplate("tpl-1", "One", "alice"))
	seedTemplate(tm, newMinimalWorkflowTemplate("tpl-2", "Two", "bob"))

	results, err := tm.SearchTemplates("", map[string]interface{}{"author": "bob"})
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "tpl-2", results[0].ID)
}

func TestSearchTemplates_FilterByTags(t *testing.T) {
	tm := newTestTemplateManager()
	tpl1 := newMinimalWorkflowTemplate("tpl-1", "One", "alice")
	tpl1.Tags = []string{"messaging", "alert"}
	tpl2 := newMinimalWorkflowTemplate("tpl-2", "Two", "bob")
	tpl2.Tags = []string{"data", "transform"}
	seedTemplate(tm, tpl1)
	seedTemplate(tm, tpl2)

	results, err := tm.SearchTemplates("", map[string]interface{}{"tags": []string{"messaging"}})
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "tpl-1", results[0].ID)
}

func TestSearchTemplates_FilterByMinRating(t *testing.T) {
	tm := newTestTemplateManager()
	tpl1 := newMinimalWorkflowTemplate("tpl-1", "Highly Rated", "alice")
	tpl1.Metadata.Rating = 4.8
	tpl2 := newMinimalWorkflowTemplate("tpl-2", "Low Rated", "bob")
	tpl2.Metadata.Rating = 2.5
	seedTemplate(tm, tpl1)
	seedTemplate(tm, tpl2)

	results, err := tm.SearchTemplates("", map[string]interface{}{"min_rating": 4.0})
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "tpl-1", results[0].ID)
}

// ============================================================
// validateParameter tests
// ============================================================

func TestValidateParameter_ValidTypes(t *testing.T) {
	tm := newTestTemplateManager()
	validTypes := []string{"string", "number", "boolean", "array", "object", "select", "multiselect", "file", "credential", "expression"}

	for _, vt := range validTypes {
		param := &Parameter{Type: vt}
		if vt == "select" || vt == "multiselect" {
			param.Options = []string{"opt1", "opt2"}
		}
		err := tm.validateParameter("test", param)
		assert.NoError(t, err, "type %q should be valid", vt)
	}
}

func TestValidateParameter_InvalidType(t *testing.T) {
	tm := newTestTemplateManager()
	param := &Parameter{Type: "invalidtype"}
	err := tm.validateParameter("test", param)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid parameter type")
}

func TestValidateParameter_MissingType(t *testing.T) {
	tm := newTestTemplateManager()
	param := &Parameter{Type: ""}
	err := tm.validateParameter("test", param)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parameter type is required")
}

func TestValidateParameter_SelectWithoutOptions(t *testing.T) {
	tm := newTestTemplateManager()
	param := &Parameter{Type: "select", Options: []string{}}
	err := tm.validateParameter("test", param)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must have options")
}

func TestValidateParameter_MultiselectWithoutOptions(t *testing.T) {
	tm := newTestTemplateManager()
	param := &Parameter{Type: "multiselect", Options: []string{}}
	err := tm.validateParameter("test", param)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must have options")
}

func TestValidateParameter_SetsNameIfEmpty(t *testing.T) {
	tm := newTestTemplateManager()
	param := &Parameter{Type: "string"}
	err := tm.validateParameter("myParam", param)
	assert.NoError(t, err)
	assert.Equal(t, "myParam", param.Name)
}

// ============================================================
// validateTemplate tests
// ============================================================

func TestValidateTemplate_MissingID(t *testing.T) {
	tm := newTestTemplateManager()
	tpl := &WorkflowTemplate{Name: "Test", Workflow: &model.Workflow{}}
	err := tm.validateTemplate(tpl)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template ID is required")
}

func TestValidateTemplate_MissingName(t *testing.T) {
	tm := newTestTemplateManager()
	tpl := &WorkflowTemplate{ID: "test-id", Workflow: &model.Workflow{}}
	err := tm.validateTemplate(tpl)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template name is required")
}

func TestValidateTemplate_MissingWorkflow(t *testing.T) {
	tm := newTestTemplateManager()
	tpl := &WorkflowTemplate{ID: "test-id", Name: "Test"}
	err := tm.validateTemplate(tpl)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template workflow is required")
}

// ============================================================
// isTruthy tests
// ============================================================

func TestIsTruthy(t *testing.T) {
	tm := newTestTemplateManager()

	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{"nil", nil, false},
		{"true", true, true},
		{"false", false, false},
		{"non-empty string", "hello", true},
		{"empty string", "", false},
		{"positive int", 42, true},
		{"zero int", 0, false},
		{"positive float64", 3.14, true},
		{"struct", struct{}{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tm.isTruthy(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================
// generateWorkflowName tests
// ============================================================

func TestGenerateWorkflowName_WithParameter(t *testing.T) {
	tm := newTestTemplateManager()
	tpl := newMinimalWorkflowTemplate("tpl-1", "My Template", "alice")

	name := tm.generateWorkflowName(tpl, map[string]interface{}{"workflow_name": "Custom Name"})
	assert.Equal(t, "Custom Name", name)
}

func TestGenerateWorkflowName_Default(t *testing.T) {
	tm := newTestTemplateManager()
	tpl := newMinimalWorkflowTemplate("tpl-1", "My Template", "alice")

	name := tm.generateWorkflowName(tpl, map[string]interface{}{})
	assert.Equal(t, "My Template (from template)", name)
}

// ============================================================
// InstallTemplate tests
// ============================================================

func TestInstallTemplate_NotFound(t *testing.T) {
	tm := newTestTemplateManager()

	result, err := tm.InstallTemplate("nonexistent", nil, nil)
	require.NoError(t, err) // err from channel, not template lookup
	assert.False(t, result.Success)
	assert.NotEmpty(t, result.Errors)
}

func TestInstallTemplate_DryRun(t *testing.T) {
	tm := newTestTemplateManager()
	tpl := newMinimalWorkflowTemplate("tpl-1", "My Template", "alice")
	seedTemplate(tm, tpl)

	result, err := tm.InstallTemplate("tpl-1", map[string]interface{}{}, &InstallationOptions{DryRun: true})
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "true", result.Metadata["dry_run"])
}

func TestInstallTemplate_Success(t *testing.T) {
	tm := newTestTemplateManager()
	tpl := newMinimalWorkflowTemplate("tpl-1", "My Template", "alice")
	seedTemplate(tm, tpl)

	result, err := tm.InstallTemplate("tpl-1", map[string]interface{}{}, nil)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.NotEmpty(t, result.WorkflowID)
	assert.NotEmpty(t, result.Changes)
}

func TestInstallTemplate_MissingRequiredParam(t *testing.T) {
	tm := newTestTemplateManager()
	tpl := newMinimalWorkflowTemplate("tpl-1", "My Template", "alice")
	tpl.Parameters = map[string]*Parameter{
		"api_key": {Name: "api_key", Type: "string", Required: true},
	}
	seedTemplate(tm, tpl)

	result, err := tm.InstallTemplate("tpl-1", map[string]interface{}{}, nil)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0], "required parameter missing")
}

// ============================================================
// addTemplateToCategory tests
// ============================================================

func TestAddTemplateToCategory_DuplicatePrevented(t *testing.T) {
	tm := newTestTemplateManager()
	tpl := newMinimalWorkflowTemplate("tpl-1", "One", "alice")
	tpl.Category = "automation"

	seedTemplate(tm, tpl)
	// Attempt to add again
	tm.addTemplateToCategory(tpl)

	cat := tm.categories["automation"]
	count := 0
	for _, id := range cat.Templates {
		if id == "tpl-1" {
			count++
		}
	}
	assert.Equal(t, 1, count, "template should only appear once in category")
}

// ============================================================
// filterTemplatesByTags tests (TemplateNode helper)
// ============================================================

func TestFilterTemplatesByTags(t *testing.T) {
	node := NewTemplateNode()
	templates := []*WorkflowTemplate{
		{ID: "1", Tags: []string{"a", "b"}},
		{ID: "2", Tags: []string{"b", "c"}},
		{ID: "3", Tags: []string{"a", "b", "c"}},
	}

	tests := []struct {
		name     string
		tags     []string
		expected int
	}{
		{"single tag match", []string{"a"}, 2},
		{"multiple tags match", []string{"a", "b"}, 2},
		{"all tags", []string{"a", "b", "c"}, 1},
		{"no match", []string{"z"}, 0},
		{"empty tags", []string{}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := node.filterTemplatesByTags(templates, tt.tags)
			assert.Len(t, result, tt.expected)
		})
	}
}

// ============================================================
// TemplateNode tests
// ============================================================

func TestNewTemplateNode(t *testing.T) {
	node := NewTemplateNode()
	require.NotNil(t, node)
	assert.Equal(t, "Template", node.Description().Name)
	assert.Equal(t, "templates", node.Description().Category)
	assert.NotNil(t, node.templateManager)
}

func TestTemplateNode_Execute_UnsupportedOperation(t *testing.T) {
	node := NewTemplateNode()
	input := []model.DataItem{{JSON: map[string]interface{}{}}}

	_, err := node.Execute(input, map[string]interface{}{"operation": "invalidop"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported operation")
}

func TestTemplateNode_Execute_DefaultOperation(t *testing.T) {
	node := NewTemplateNode()
	input := []model.DataItem{{JSON: map[string]interface{}{}}}

	results, err := node.Execute(input, map[string]interface{}{})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "list", results[0].JSON["operation"])
}

func TestTemplateNode_Execute_ListOperation(t *testing.T) {
	node := NewTemplateNode()
	input := []model.DataItem{{JSON: map[string]interface{}{}}}

	results, err := node.Execute(input, map[string]interface{}{"operation": "list"})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "list", results[0].JSON["operation"])
}

func TestTemplateNode_Execute_SearchOperation(t *testing.T) {
	node := NewTemplateNode()
	input := []model.DataItem{{JSON: map[string]interface{}{}}}

	results, err := node.Execute(input, map[string]interface{}{"operation": "search", "query": "test"})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "search", results[0].JSON["operation"])
	assert.Equal(t, "test", results[0].JSON["query"])
}

func TestTemplateNode_Execute_GetMissingID(t *testing.T) {
	node := NewTemplateNode()
	input := []model.DataItem{{JSON: map[string]interface{}{}}}

	_, err := node.Execute(input, map[string]interface{}{"operation": "get"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "templateId is required")
}

func TestTemplateNode_Execute_InstallMissingID(t *testing.T) {
	node := NewTemplateNode()
	input := []model.DataItem{{JSON: map[string]interface{}{}}}

	_, err := node.Execute(input, map[string]interface{}{"operation": "install"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "templateId is required")
}

func TestTemplateNode_Execute_ValidateMissingID(t *testing.T) {
	node := NewTemplateNode()
	input := []model.DataItem{{JSON: map[string]interface{}{}}}

	_, err := node.Execute(input, map[string]interface{}{"operation": "validate"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "templateId is required")
}

func TestTemplateNode_Execute_EmptyInput(t *testing.T) {
	node := NewTemplateNode()

	results, err := node.Execute([]model.DataItem{}, map[string]interface{}{"operation": "list"})
	require.NoError(t, err)
	assert.Empty(t, results)
}

// ============================================================
// MarketplaceManager tests
// ============================================================

func TestNewMarketplaceManager(t *testing.T) {
	tm := newTestTemplateManager()
	mm := NewMarketplaceManager(tm)
	require.NotNil(t, mm)
	assert.NotNil(t, mm.repositories)
	assert.NotNil(t, mm.cache)
	assert.NotNil(t, mm.ratingSystem)
	assert.NotNil(t, mm.analytics)
	assert.NotNil(t, mm.securityScanner)
}

func TestMarketplaceManager_DefaultRepositories(t *testing.T) {
	tm := newTestTemplateManager()
	mm := NewMarketplaceManager(tm)

	mm.mu.RLock()
	defer mm.mu.RUnlock()
	official, exists := mm.repositories["official"]
	assert.True(t, exists)
	assert.Equal(t, "Official m9m Templates", official.Name)
	assert.True(t, official.Enabled)
	assert.True(t, official.Metadata.Official)
}

func TestMarketplaceManager_AddRepository_Success(t *testing.T) {
	tm := newTestTemplateManager()
	mm := NewMarketplaceManager(tm)

	repo := &Repository{
		ID:      "test-repo",
		Name:    "Test Repository",
		URL:     "https://example.com/templates",
		Type:    "http",
		Enabled: false,
		Metadata: &RepositoryMetadata{
			Description: "Test",
		},
	}

	err := mm.AddRepository(repo)
	assert.NoError(t, err)

	mm.mu.RLock()
	_, exists := mm.repositories["test-repo"]
	mm.mu.RUnlock()
	assert.True(t, exists)
}

func TestMarketplaceManager_AddRepository_Duplicate(t *testing.T) {
	tm := newTestTemplateManager()
	mm := NewMarketplaceManager(tm)

	repo := &Repository{
		ID:       "official",
		Name:     "Duplicate",
		Metadata: &RepositoryMetadata{},
	}

	err := mm.AddRepository(repo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repository already exists")
}

func TestMarketplaceManager_SyncRepository_NotFound(t *testing.T) {
	tm := newTestTemplateManager()
	mm := NewMarketplaceManager(tm)

	err := mm.SyncRepository("nonexistent-repo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repository not found")
}

// ============================================================
// MarketplaceCache tests
// ============================================================

func TestMarketplaceCache_AddAndGetTemplate(t *testing.T) {
	cache := NewMarketplaceCache(1 * time.Hour)

	tpl := &MarketplaceTemplate{
		WorkflowTemplate: newMinimalWorkflowTemplate("mt-1", "Cached Template", "tester"),
	}

	cache.AddTemplate(tpl)

	cache.mu.RLock()
	result, exists := cache.templates["mt-1"]
	cache.mu.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, "mt-1", result.ID)
}

func TestMarketplaceCache_CacheQuery(t *testing.T) {
	cache := NewMarketplaceCache(1 * time.Hour)

	query := &CachedQuery{
		Query:     "test",
		Results:   []*MarketplaceTemplate{},
		Timestamp: time.Now(),
	}

	cache.CacheQuery("key1", query)
	result := cache.GetCachedQuery("key1")
	assert.NotNil(t, result)
	assert.Equal(t, "test", result.Query)
}

func TestMarketplaceCache_CacheQueryExpired(t *testing.T) {
	cache := NewMarketplaceCache(1 * time.Millisecond)

	query := &CachedQuery{
		Query:     "test",
		Results:   []*MarketplaceTemplate{},
		Timestamp: time.Now().Add(-1 * time.Second),
	}

	cache.CacheQuery("key1", query)
	result := cache.GetCachedQuery("key1")
	assert.Nil(t, result)
}

func TestMarketplaceCache_CacheQueryMiss(t *testing.T) {
	cache := NewMarketplaceCache(1 * time.Hour)
	result := cache.GetCachedQuery("nonexistent")
	assert.Nil(t, result)
}

// ============================================================
// RatingSystem tests
// ============================================================

func TestRatingSystem_AddRating_Success(t *testing.T) {
	rs := NewRatingSystem()

	err := rs.AddRating(&Rating{
		UserID:     "user1",
		TemplateID: "tpl-1",
		Score:      5,
		CreatedAt:  time.Now(),
	})
	assert.NoError(t, err)
}

func TestRatingSystem_AddRating_InvalidScore(t *testing.T) {
	rs := NewRatingSystem()

	tests := []struct {
		name  string
		score int
	}{
		{"too low", 0},
		{"too high", 6},
		{"negative", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rs.AddRating(&Rating{
				UserID:     "user1",
				TemplateID: "tpl-1",
				Score:      tt.score,
				CreatedAt:  time.Now(),
			})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "rating score must be between 1 and 5")
		})
	}
}

func TestRatingSystem_GetAverageRating(t *testing.T) {
	rs := NewRatingSystem()

	rs.AddRating(&Rating{UserID: "u1", TemplateID: "tpl-1", Score: 4, CreatedAt: time.Now()})
	rs.AddRating(&Rating{UserID: "u2", TemplateID: "tpl-1", Score: 5, CreatedAt: time.Now()})
	rs.AddRating(&Rating{UserID: "u3", TemplateID: "tpl-1", Score: 3, CreatedAt: time.Now()})

	avg, count, err := rs.GetAverageRating("tpl-1")
	require.NoError(t, err)
	assert.Equal(t, 3, count)
	assert.Equal(t, 4.0, avg)
}

func TestRatingSystem_GetAverageRating_NoRatings(t *testing.T) {
	rs := NewRatingSystem()

	avg, count, err := rs.GetAverageRating("tpl-nonexistent")
	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Equal(t, 0.0, avg)
}

// ============================================================
// MarketplaceAnalytics tests
// ============================================================

func TestMarketplaceAnalytics_RecordSearch(t *testing.T) {
	analytics := NewMarketplaceAnalytics()
	analytics.RecordSearch("test query")
	analytics.RecordSearch("test query")
	analytics.RecordSearch("another query")

	analytics.mu.RLock()
	defer analytics.mu.RUnlock()
	assert.Equal(t, 2, analytics.searchQueries["test query"])
	assert.Equal(t, 1, analytics.searchQueries["another query"])
}

func TestMarketplaceAnalytics_RecordDownload(t *testing.T) {
	analytics := NewMarketplaceAnalytics()
	analytics.RecordDownload("tpl-1")
	analytics.RecordDownload("tpl-1")

	analytics.mu.RLock()
	defer analytics.mu.RUnlock()
	assert.Equal(t, int64(2), analytics.downloads["tpl-1"])
}

// ============================================================
// SecurityScanner tests
// ============================================================

func TestSecurityScanner_ScanTemplate_Safe(t *testing.T) {
	scanner := NewSecurityScanner()

	tpl := &MarketplaceTemplate{
		WorkflowTemplate: &WorkflowTemplate{
			ID:          "safe-tpl",
			Name:        "Safe Template",
			Description: "A safe template",
			Workflow: &model.Workflow{
				ID:   "wf-1",
				Name: "Safe Workflow",
			},
			Metadata: &TemplateMetadata{Keywords: []string{}},
		},
	}

	report := scanner.ScanTemplate(tpl)
	assert.NotNil(t, report)
	assert.NotEmpty(t, report.ScanDate.String())
}

func TestSecurityScanner_DefaultRules(t *testing.T) {
	scanner := NewSecurityScanner()

	assert.Contains(t, scanner.rules, "hardcoded_credentials")
	assert.Contains(t, scanner.rules, "external_scripts")
}

// ============================================================
// UpdateScheduler tests
// ============================================================

func TestUpdateScheduler_New(t *testing.T) {
	scheduler := NewUpdateScheduler(24 * time.Hour)
	assert.NotNil(t, scheduler)
	assert.NotNil(t, scheduler.stopChan)
	assert.Equal(t, 24*time.Hour, scheduler.interval)
}

// ============================================================
// MarketplaceManager.SearchMarketplace tests
// ============================================================

func TestSearchMarketplace_EmptyResults(t *testing.T) {
	tm := newTestTemplateManager()
	mm := NewMarketplaceManager(tm)

	results, err := mm.SearchMarketplace("nonexistent", nil)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearchMarketplace_MatchesQuery(t *testing.T) {
	tm := newTestTemplateManager()
	mm := NewMarketplaceManager(tm)

	tpl := &MarketplaceTemplate{
		WorkflowTemplate: newMinimalWorkflowTemplate("mt-1", "Email Automation", "alice"),
		RepositoryID:     "official",
		Popularity:       &PopularityMetrics{},
		SecurityReport:   &SecurityReport{SecurityLevel: "safe"},
	}

	mm.mu.Lock()
	mm.repositories["official"].Templates = append(mm.repositories["official"].Templates, tpl)
	mm.mu.Unlock()

	results, err := mm.SearchMarketplace("email", nil)
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestSearchMarketplace_EmptyQueryReturnsAll(t *testing.T) {
	tm := newTestTemplateManager()
	mm := NewMarketplaceManager(tm)

	tpl1 := &MarketplaceTemplate{
		WorkflowTemplate: newMinimalWorkflowTemplate("mt-1", "One", "alice"),
		RepositoryID:     "official",
		Popularity:       &PopularityMetrics{},
		SecurityReport:   &SecurityReport{SecurityLevel: "safe"},
	}
	tpl2 := &MarketplaceTemplate{
		WorkflowTemplate: newMinimalWorkflowTemplate("mt-2", "Two", "bob"),
		RepositoryID:     "official",
		Popularity:       &PopularityMetrics{},
		SecurityReport:   &SecurityReport{SecurityLevel: "safe"},
	}

	mm.mu.Lock()
	mm.repositories["official"].Templates = []*MarketplaceTemplate{tpl1, tpl2}
	mm.mu.Unlock()

	results, err := mm.SearchMarketplace("", nil)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

// ============================================================
// MarketplaceManager.GetTrendingTemplates tests
// ============================================================

func TestGetTrendingTemplates_ReturnsLimited(t *testing.T) {
	tm := newTestTemplateManager()
	mm := NewMarketplaceManager(tm)

	var templates []*MarketplaceTemplate
	for i := 0; i < 5; i++ {
		tpl := &MarketplaceTemplate{
			WorkflowTemplate: newMinimalWorkflowTemplate(
				fmt.Sprintf("mt-%d", i),
				fmt.Sprintf("Template %d", i),
				"tester",
			),
			RepositoryID: "official",
			Popularity:   &PopularityMetrics{TrendingScore: float64(i * 10)},
		}
		templates = append(templates, tpl)
	}

	mm.mu.Lock()
	mm.repositories["official"].Templates = templates
	mm.mu.Unlock()

	results, err := mm.GetTrendingTemplates(3)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	// Should be sorted by trending score descending
	assert.True(t, results[0].Popularity.TrendingScore >= results[1].Popularity.TrendingScore)
}

func TestGetTrendingTemplates_LimitGreaterThanTotal(t *testing.T) {
	tm := newTestTemplateManager()
	mm := NewMarketplaceManager(tm)

	tpl := &MarketplaceTemplate{
		WorkflowTemplate: newMinimalWorkflowTemplate("mt-1", "One", "alice"),
		RepositoryID:     "official",
		Popularity:       &PopularityMetrics{TrendingScore: 10},
	}

	mm.mu.Lock()
	mm.repositories["official"].Templates = []*MarketplaceTemplate{tpl}
	mm.mu.Unlock()

	results, err := mm.GetTrendingTemplates(100)
	require.NoError(t, err)
	assert.Len(t, results, 1)
}

// ============================================================
// MarketplaceManager rating integration tests
// ============================================================

func TestMarketplaceManager_RateTemplate(t *testing.T) {
	tm := newTestTemplateManager()
	mm := NewMarketplaceManager(tm)

	err := mm.RateTemplate("tpl-1", "user1", 4, "Great template")
	assert.NoError(t, err)

	avg, count, err := mm.GetTemplateRating("tpl-1")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Equal(t, 4.0, avg)
}

func TestMarketplaceManager_RateTemplate_InvalidScore(t *testing.T) {
	tm := newTestTemplateManager()
	mm := NewMarketplaceManager(tm)

	err := mm.RateTemplate("tpl-1", "user1", 0, "Bad rating")
	assert.Error(t, err)
}

// ============================================================
// Helper: fmt import for test file
// ============================================================

// fmt is used by TestGetTrendingTemplates_ReturnsLimited
var _ = fmt.Sprintf
