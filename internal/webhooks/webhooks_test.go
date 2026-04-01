package webhooks

import (
	"testing"
	"time"

	"github.com/neul-labs/m9m/internal/engine"
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newTestStorage creates a MemoryWebhookStorage backed by a real MemoryStorage.
func newTestStorage() *MemoryWebhookStorage {
	return NewMemoryWebhookStorage(storage.NewMemoryStorage())
}

// newTestManager creates a WebhookManager wired with real in-memory stores and
// a real (no-op for empty workflows) engine.
func newTestManager() (*WebhookManager, *MemoryWebhookStorage, *storage.MemoryStorage) {
	ws := storage.NewMemoryStorage()
	whs := NewMemoryWebhookStorage(ws)
	eng := engine.NewWorkflowEngine()
	mgr := NewWebhookManager(whs, ws, eng)
	return mgr, whs, ws
}

// makeWebhook returns a minimal Webhook suitable for most tests.
func makeWebhook(id, workflowID, path, method string, active bool) *Webhook {
	return &Webhook{
		ID:         id,
		WorkflowID: workflowID,
		Path:       path,
		Method:     method,
		Active:     active,
	}
}

// ---------------------------------------------------------------------------
// MemoryWebhookStorage Tests
// ---------------------------------------------------------------------------

func TestMemoryWebhookStorage_SaveAndGetWebhook(t *testing.T) {
	s := newTestStorage()

	wh := makeWebhook("wh-1", "wf-1", "/test", "POST", true)
	err := s.SaveWebhook(wh)
	require.NoError(t, err, "SaveWebhook should succeed")

	// Verify timestamps were set.
	assert.False(t, wh.CreatedAt.IsZero(), "CreatedAt should be set")
	assert.False(t, wh.UpdatedAt.IsZero(), "UpdatedAt should be set")

	got, err := s.GetWebhook("wh-1")
	require.NoError(t, err, "GetWebhook should succeed")
	assert.Equal(t, wh.ID, got.ID)
	assert.Equal(t, wh.WorkflowID, got.WorkflowID)
	assert.Equal(t, wh.Path, got.Path)
	assert.Equal(t, wh.Method, got.Method)
	assert.Equal(t, wh.Active, got.Active)
}

func TestMemoryWebhookStorage_SaveWebhook_EmptyID(t *testing.T) {
	s := newTestStorage()

	wh := makeWebhook("", "wf-1", "/test", "POST", true)
	err := s.SaveWebhook(wh)
	require.Error(t, err, "SaveWebhook with empty ID should fail")
	assert.Contains(t, err.Error(), "ID cannot be empty")
}

func TestMemoryWebhookStorage_SaveWebhook_PreservesCreatedAt(t *testing.T) {
	s := newTestStorage()

	earlier := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	wh := makeWebhook("wh-1", "wf-1", "/test", "POST", true)
	wh.CreatedAt = earlier

	err := s.SaveWebhook(wh)
	require.NoError(t, err)
	assert.Equal(t, earlier, wh.CreatedAt, "CreatedAt should not be overwritten when already set")
}

func TestMemoryWebhookStorage_GetWebhookByPath(t *testing.T) {
	s := newTestStorage()

	wh := makeWebhook("wh-1", "wf-1", "/hook", "GET", true)
	require.NoError(t, s.SaveWebhook(wh))

	got, err := s.GetWebhookByPath("/hook", "GET", false)
	require.NoError(t, err)
	assert.Equal(t, "wh-1", got.ID)

	// Different method should not match.
	_, err = s.GetWebhookByPath("/hook", "POST", false)
	assert.Error(t, err, "different method should not match")

	// Different isTest flag should not match.
	_, err = s.GetWebhookByPath("/hook", "GET", true)
	assert.Error(t, err, "different isTest should not match")
}

func TestMemoryWebhookStorage_GetWebhookByPath_TestFlag(t *testing.T) {
	s := newTestStorage()

	whTest := makeWebhook("wh-test", "wf-1", "/hook", "POST", true)
	whTest.IsTest = true
	require.NoError(t, s.SaveWebhook(whTest))

	// Should find with isTest=true.
	got, err := s.GetWebhookByPath("/hook", "POST", true)
	require.NoError(t, err)
	assert.Equal(t, "wh-test", got.ID)

	// Should not find with isTest=false.
	_, err = s.GetWebhookByPath("/hook", "POST", false)
	assert.Error(t, err)
}

func TestMemoryWebhookStorage_ListWebhooks(t *testing.T) {
	s := newTestStorage()

	require.NoError(t, s.SaveWebhook(makeWebhook("wh-1", "wf-1", "/a", "GET", true)))
	require.NoError(t, s.SaveWebhook(makeWebhook("wh-2", "wf-1", "/b", "POST", true)))
	require.NoError(t, s.SaveWebhook(makeWebhook("wh-3", "wf-2", "/c", "PUT", false)))

	// List all (empty workflowID).
	all, err := s.ListWebhooks("")
	require.NoError(t, err)
	assert.Len(t, all, 3, "should return all webhooks when workflowID is empty")

	// List by workflowID.
	wf1, err := s.ListWebhooks("wf-1")
	require.NoError(t, err)
	assert.Len(t, wf1, 2, "should return only wf-1 webhooks")
	for _, wh := range wf1 {
		assert.Equal(t, "wf-1", wh.WorkflowID)
	}

	// List by non-existent workflowID.
	none, err := s.ListWebhooks("wf-999")
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestMemoryWebhookStorage_DeleteWebhook(t *testing.T) {
	s := newTestStorage()

	wh := makeWebhook("wh-del", "wf-1", "/delete-me", "DELETE", true)
	require.NoError(t, s.SaveWebhook(wh))

	// Verify it exists first.
	_, err := s.GetWebhook("wh-del")
	require.NoError(t, err)

	// Delete.
	err = s.DeleteWebhook("wh-del")
	require.NoError(t, err)

	// Should no longer be found by ID.
	_, err = s.GetWebhook("wh-del")
	assert.Error(t, err, "webhook should be removed from primary map")

	// Should no longer be found by path.
	_, err = s.GetWebhookByPath("/delete-me", "DELETE", false)
	assert.Error(t, err, "webhook should be removed from path index")
}

func TestMemoryWebhookStorage_DeleteWebhook_NotFound(t *testing.T) {
	s := newTestStorage()

	err := s.DeleteWebhook("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMemoryWebhookStorage_GetWebhook_NotFound(t *testing.T) {
	s := newTestStorage()

	_, err := s.GetWebhook("missing-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMemoryWebhookStorage_SaveAndGetExecution(t *testing.T) {
	s := newTestStorage()

	exec := &WebhookExecution{
		ID:          "exec-1",
		WebhookID:   "wh-1",
		ExecutionID: "run-1",
		Status:      "success",
		Request: &WebhookRequest{
			Method: "POST",
			Path:   "/test",
		},
		Duration: 42,
	}

	err := s.SaveWebhookExecution(exec)
	require.NoError(t, err)
	assert.False(t, exec.CreatedAt.IsZero(), "CreatedAt should be set automatically")

	got, err := s.GetWebhookExecution("exec-1")
	require.NoError(t, err)
	assert.Equal(t, "exec-1", got.ID)
	assert.Equal(t, "success", got.Status)
	assert.Equal(t, int64(42), got.Duration)
	assert.Equal(t, "POST", got.Request.Method)
}

func TestMemoryWebhookStorage_SaveExecution_EmptyID(t *testing.T) {
	s := newTestStorage()

	exec := &WebhookExecution{WebhookID: "wh-1", Status: "success"}
	err := s.SaveWebhookExecution(exec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ID cannot be empty")
}

func TestMemoryWebhookStorage_GetExecution_NotFound(t *testing.T) {
	s := newTestStorage()

	_, err := s.GetWebhookExecution("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMemoryWebhookStorage_ListExecutions(t *testing.T) {
	s := newTestStorage()

	for i := 0; i < 5; i++ {
		exec := &WebhookExecution{
			ID:        "exec-" + string(rune('a'+i)),
			WebhookID: "wh-1",
			Status:    "success",
		}
		require.NoError(t, s.SaveWebhookExecution(exec))
	}
	// Add one for a different webhook.
	require.NoError(t, s.SaveWebhookExecution(&WebhookExecution{
		ID:        "exec-other",
		WebhookID: "wh-2",
		Status:    "failed",
	}))

	// List all (empty webhookID).
	all, err := s.ListWebhookExecutions("", 0)
	require.NoError(t, err)
	assert.Len(t, all, 6, "should return all executions when webhookID is empty")

	// List filtered by webhookID.
	wh1, err := s.ListWebhookExecutions("wh-1", 0)
	require.NoError(t, err)
	assert.Len(t, wh1, 5)

	// List with limit.
	limited, err := s.ListWebhookExecutions("wh-1", 2)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(limited), 2, "should respect the limit parameter")
}

func TestMemoryWebhookStorage_ListExecutions_EmptyStore(t *testing.T) {
	s := newTestStorage()

	execs, err := s.ListWebhookExecutions("wh-1", 10)
	require.NoError(t, err)
	assert.Empty(t, execs)
}

// ---------------------------------------------------------------------------
// WebhookManager Tests
// ---------------------------------------------------------------------------

func TestWebhookManager_RegisterWebhook(t *testing.T) {
	mgr, _, _ := newTestManager()

	wh := &Webhook{
		WorkflowID: "wf-1",
		Path:       "my-hook",
		Active:     true,
	}

	err := mgr.RegisterWebhook(wh)
	require.NoError(t, err)

	// ID should have been generated.
	assert.NotEmpty(t, wh.ID, "ID should be auto-generated")

	// Path should have been normalized with a leading slash.
	assert.Equal(t, "/my-hook", wh.Path, "path should be normalized with leading slash")

	// Defaults should be applied.
	assert.Equal(t, "POST", wh.Method, "default method should be POST")
	assert.Equal(t, "onReceived", wh.ResponseMode, "default responseMode should be onReceived")
	assert.Equal(t, "firstEntryJson", wh.ResponseData, "default responseData should be firstEntryJson")

	// Should be retrievable from storage.
	got, err := mgr.GetWebhook(wh.ID)
	require.NoError(t, err)
	assert.Equal(t, wh.ID, got.ID)
}

func TestWebhookManager_RegisterWebhook_ExplicitValues(t *testing.T) {
	mgr, _, _ := newTestManager()

	wh := &Webhook{
		ID:           "explicit-id",
		WorkflowID:   "wf-1",
		Path:         "/explicit",
		Method:       "PUT",
		Active:       true,
		ResponseMode: "lastNode",
		ResponseData: "allEntries",
	}

	err := mgr.RegisterWebhook(wh)
	require.NoError(t, err)
	assert.Equal(t, "explicit-id", wh.ID, "explicit ID should not be overwritten")
	assert.Equal(t, "PUT", wh.Method, "explicit method should not be overwritten")
	assert.Equal(t, "lastNode", wh.ResponseMode, "explicit responseMode should not be overwritten")
	assert.Equal(t, "allEntries", wh.ResponseData, "explicit responseData should not be overwritten")
}

func TestWebhookManager_RegisterWebhook_Validation_MissingWorkflowID(t *testing.T) {
	mgr, _, _ := newTestManager()

	wh := &Webhook{
		Path:   "/test",
		Active: true,
	}

	err := mgr.RegisterWebhook(wh)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workflow ID is required")
}

func TestWebhookManager_RegisterWebhook_Validation_MissingPath(t *testing.T) {
	mgr, _, _ := newTestManager()

	wh := &Webhook{
		WorkflowID: "wf-1",
		Active:     true,
	}

	err := mgr.RegisterWebhook(wh)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "path is required")
}

func TestWebhookManager_RegisterWebhook_InactiveNotCached(t *testing.T) {
	mgr, _, _ := newTestManager()

	wh := &Webhook{
		WorkflowID: "wf-1",
		Path:       "/inactive",
		Method:     "POST",
		Active:     false,
	}

	err := mgr.RegisterWebhook(wh)
	require.NoError(t, err)

	// Should still be in storage.
	_, err = mgr.GetWebhook(wh.ID)
	require.NoError(t, err)

	// But should NOT be findable by path (active hooks cache).
	_, err = mgr.GetWebhookByPath("/inactive", "POST", false)
	assert.Error(t, err, "inactive webhook should not be in active cache")
}

func TestWebhookManager_UnregisterWebhook(t *testing.T) {
	mgr, _, _ := newTestManager()

	wh := &Webhook{
		ID:         "wh-unreg",
		WorkflowID: "wf-1",
		Path:       "/unreg",
		Method:     "POST",
		Active:     true,
	}
	require.NoError(t, mgr.RegisterWebhook(wh))

	// Verify it is in the cache.
	got, err := mgr.GetWebhookByPath("/unreg", "POST", false)
	require.NoError(t, err)
	assert.Equal(t, "wh-unreg", got.ID)

	// Unregister.
	err = mgr.UnregisterWebhook("wh-unreg")
	require.NoError(t, err)

	// Should not be in cache anymore.
	_, err = mgr.GetWebhookByPath("/unreg", "POST", false)
	assert.Error(t, err, "unregistered webhook should be removed from active cache")

	// Should not be in storage anymore.
	_, err = mgr.GetWebhook("wh-unreg")
	assert.Error(t, err, "unregistered webhook should be removed from storage")
}

func TestWebhookManager_UnregisterWebhook_NotFound(t *testing.T) {
	mgr, _, _ := newTestManager()

	err := mgr.UnregisterWebhook("does-not-exist")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestWebhookManager_GetWebhookByPath(t *testing.T) {
	mgr, _, _ := newTestManager()

	wh := &Webhook{
		ID:         "wh-path",
		WorkflowID: "wf-1",
		Path:       "/by-path",
		Method:     "GET",
		Active:     true,
	}
	require.NoError(t, mgr.RegisterWebhook(wh))

	// Should find the webhook.
	got, err := mgr.GetWebhookByPath("/by-path", "GET", false)
	require.NoError(t, err)
	assert.Equal(t, "wh-path", got.ID)

	// Path without leading slash should still work (normalization).
	got, err = mgr.GetWebhookByPath("by-path", "GET", false)
	require.NoError(t, err)
	assert.Equal(t, "wh-path", got.ID)
}

func TestWebhookManager_GetWebhookByPath_NotFound(t *testing.T) {
	mgr, _, _ := newTestManager()

	_, err := mgr.GetWebhookByPath("/nonexistent", "GET", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestWebhookManager_GetWebhookByPath_TestVsProduction(t *testing.T) {
	mgr, _, _ := newTestManager()

	prodWh := &Webhook{
		ID:         "wh-prod",
		WorkflowID: "wf-1",
		Path:       "/dual",
		Method:     "POST",
		Active:     true,
		IsTest:     false,
	}
	testWh := &Webhook{
		ID:         "wh-test",
		WorkflowID: "wf-1",
		Path:       "/dual",
		Method:     "POST",
		Active:     true,
		IsTest:     true,
	}
	require.NoError(t, mgr.RegisterWebhook(prodWh))
	require.NoError(t, mgr.RegisterWebhook(testWh))

	got, err := mgr.GetWebhookByPath("/dual", "POST", false)
	require.NoError(t, err)
	assert.Equal(t, "wh-prod", got.ID, "should return production webhook")

	got, err = mgr.GetWebhookByPath("/dual", "POST", true)
	require.NoError(t, err)
	assert.Equal(t, "wh-test", got.ID, "should return test webhook")
}

func TestWebhookManager_ListWebhooks(t *testing.T) {
	mgr, _, _ := newTestManager()

	require.NoError(t, mgr.RegisterWebhook(&Webhook{
		ID: "wh-a", WorkflowID: "wf-1", Path: "/a", Active: true,
	}))
	require.NoError(t, mgr.RegisterWebhook(&Webhook{
		ID: "wh-b", WorkflowID: "wf-1", Path: "/b", Active: true,
	}))
	require.NoError(t, mgr.RegisterWebhook(&Webhook{
		ID: "wh-c", WorkflowID: "wf-2", Path: "/c", Active: true,
	}))

	wf1, err := mgr.ListWebhooks("wf-1")
	require.NoError(t, err)
	assert.Len(t, wf1, 2)

	wf2, err := mgr.ListWebhooks("wf-2")
	require.NoError(t, err)
	assert.Len(t, wf2, 1)

	all, err := mgr.ListWebhooks("")
	require.NoError(t, err)
	assert.Len(t, all, 3)
}

func TestWebhookManager_LoadActiveWebhooks(t *testing.T) {
	mgr, whs, _ := newTestManager()

	// Manually save webhooks directly to storage (bypassing the manager's cache).
	activeWh := makeWebhook("wh-active", "wf-1", "/active", "POST", true)
	inactiveWh := makeWebhook("wh-inactive", "wf-1", "/inactive", "GET", false)

	require.NoError(t, whs.SaveWebhook(activeWh))
	require.NoError(t, whs.SaveWebhook(inactiveWh))

	// At this point the manager's in-memory cache should be empty.
	_, err := mgr.GetWebhookByPath("/active", "POST", false)
	assert.Error(t, err, "cache should be empty before LoadActiveWebhooks")

	// Load active webhooks.
	err = mgr.LoadActiveWebhooks()
	require.NoError(t, err)

	// Active webhook should be in cache.
	got, err := mgr.GetWebhookByPath("/active", "POST", false)
	require.NoError(t, err)
	assert.Equal(t, "wh-active", got.ID)

	// Inactive webhook should NOT be in cache.
	_, err = mgr.GetWebhookByPath("/inactive", "GET", false)
	assert.Error(t, err, "inactive webhook should not be loaded into cache")
}

func TestWebhookManager_LoadActiveWebhooks_EmptyStorage(t *testing.T) {
	mgr, _, _ := newTestManager()

	err := mgr.LoadActiveWebhooks()
	require.NoError(t, err, "LoadActiveWebhooks on empty storage should not error")
}

// ---------------------------------------------------------------------------
// ExecuteWebhook Tests
// ---------------------------------------------------------------------------

func TestWebhookManager_ExecuteWebhook(t *testing.T) {
	mgr, _, ws := newTestManager()

	// Create a workflow that the manager can retrieve. The engine will run it.
	// An empty-node workflow simply returns the input data.
	wf := &model.Workflow{
		ID:     "wf-exec",
		Name:   "exec-test",
		Active: true,
		Nodes:  []model.Node{},
	}
	require.NoError(t, ws.SaveWorkflow(wf))

	wh := &Webhook{
		ID:           "wh-exec",
		WorkflowID:   "wf-exec",
		Path:         "/exec",
		Method:       "POST",
		Active:       true,
		ResponseData: "firstEntryJson",
	}
	require.NoError(t, mgr.RegisterWebhook(wh))

	req := &WebhookRequest{
		Method: "POST",
		Path:   "/exec",
		Body:   map[string]interface{}{"key": "value"},
		Query:  map[string][]string{"q": {"1"}},
	}

	resp, err := mgr.ExecuteWebhook(wh, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Headers["Content-Type"])
}

func TestWebhookManager_ExecuteWebhook_WorkflowNotFound(t *testing.T) {
	mgr, _, _ := newTestManager()

	wh := &Webhook{
		ID:         "wh-missing-wf",
		WorkflowID: "nonexistent-wf",
		Path:       "/missing",
		Method:     "POST",
		Active:     true,
	}

	req := &WebhookRequest{Method: "POST", Path: "/missing"}
	_, err := mgr.ExecuteWebhook(wh, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workflow not found")
}

// ---------------------------------------------------------------------------
// Helper function tests
// ---------------------------------------------------------------------------

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "/"},
		{"/", "/"},
		{"/foo", "/foo"},
		{"foo", "/foo"},
		{"foo/bar", "/foo/bar"},
		{"/foo/bar", "/foo/bar"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizePath(tt.input))
		})
	}
}

func TestMakeWebhookKey(t *testing.T) {
	assert.Equal(t, "POST:/hook", makeWebhookKey("/hook", "POST", false))
	assert.Equal(t, "POST:/hook", makeWebhookKey("/hook", "post", false))
	assert.Equal(t, "GET:/hook:test", makeWebhookKey("/hook", "GET", true))
}

func TestIsWebhookNode(t *testing.T) {
	assert.True(t, isWebhookNode("n8n-nodes-base.webhook"))
	assert.True(t, isWebhookNode("custom-Webhook-trigger"))
	assert.False(t, isWebhookNode("n8n-nodes-base.httpRequest"))
	assert.False(t, isWebhookNode("set"))
}

func TestGetStringParam(t *testing.T) {
	params := map[string]interface{}{
		"name":   "hello",
		"number": 42,
	}

	assert.Equal(t, "hello", getStringParam(params, "name", "default"))
	assert.Equal(t, "default", getStringParam(params, "missing", "default"))
	assert.Equal(t, "default", getStringParam(params, "number", "default"), "non-string value should return default")
	assert.Equal(t, "default", getStringParam(nil, "name", "default"), "nil params should return default")
}

// ---------------------------------------------------------------------------
// RegisterWorkflowWebhooks / UnregisterWorkflowWebhooks
// ---------------------------------------------------------------------------

func TestWebhookManager_RegisterWorkflowWebhooks(t *testing.T) {
	mgr, _, _ := newTestManager()

	wf := &model.Workflow{
		ID:     "wf-multi",
		Name:   "multi-webhook",
		Active: true,
		Nodes: []model.Node{
			{
				Name: "Webhook1",
				Type: "n8n-nodes-base.webhook",
				Parameters: map[string]interface{}{
					"path":       "hook-a",
					"httpMethod": "GET",
				},
			},
			{
				Name: "Webhook2",
				Type: "n8n-nodes-base.webhook",
				Parameters: map[string]interface{}{
					"path":       "hook-b",
					"httpMethod": "POST",
				},
			},
			{
				Name:       "SetNode",
				Type:       "n8n-nodes-base.set",
				Parameters: map[string]interface{}{},
			},
		},
	}

	err := mgr.RegisterWorkflowWebhooks(wf, false)
	require.NoError(t, err)

	// Should have registered two webhook nodes (not the Set node).
	hooks, err := mgr.ListWebhooks("wf-multi")
	require.NoError(t, err)
	assert.Len(t, hooks, 2)

	// Verify paths were registered.
	_, err = mgr.GetWebhookByPath("/hook-a", "GET", false)
	require.NoError(t, err)

	_, err = mgr.GetWebhookByPath("/hook-b", "POST", false)
	require.NoError(t, err)
}

func TestWebhookManager_RegisterWorkflowWebhooks_TestMode(t *testing.T) {
	mgr, _, _ := newTestManager()

	wf := &model.Workflow{
		ID:     "wf-test-mode",
		Name:   "test-mode",
		Active: true,
		Nodes: []model.Node{
			{
				Name: "Webhook",
				Type: "n8n-nodes-base.webhook",
				Parameters: map[string]interface{}{
					"path":       "test-hook",
					"httpMethod": "POST",
				},
			},
		},
	}

	err := mgr.RegisterWorkflowWebhooks(wf, true)
	require.NoError(t, err)

	hooks, err := mgr.ListWebhooks("wf-test-mode")
	require.NoError(t, err)
	require.Len(t, hooks, 1)
	assert.True(t, hooks[0].IsTest, "webhook should be in test mode")
	// In test mode, Active should be false (workflow.Active && !isTest -> true && false -> false).
	assert.False(t, hooks[0].Active, "test webhook should not be active")
}

func TestWebhookManager_UnregisterWorkflowWebhooks(t *testing.T) {
	mgr, _, _ := newTestManager()

	wf := &model.Workflow{
		ID:     "wf-unreg-all",
		Name:   "unreg-all",
		Active: true,
		Nodes: []model.Node{
			{
				Name: "Webhook1",
				Type: "n8n-nodes-base.webhook",
				Parameters: map[string]interface{}{
					"path":       "unreg-a",
					"httpMethod": "POST",
				},
			},
			{
				Name: "Webhook2",
				Type: "n8n-nodes-base.webhook",
				Parameters: map[string]interface{}{
					"path":       "unreg-b",
					"httpMethod": "GET",
				},
			},
		},
	}

	require.NoError(t, mgr.RegisterWorkflowWebhooks(wf, false))
	hooks, err := mgr.ListWebhooks("wf-unreg-all")
	require.NoError(t, err)
	assert.Len(t, hooks, 2, "should have two webhooks registered")

	err = mgr.UnregisterWorkflowWebhooks("wf-unreg-all")
	require.NoError(t, err)

	hooks, err = mgr.ListWebhooks("wf-unreg-all")
	require.NoError(t, err)
	assert.Empty(t, hooks, "all webhooks for the workflow should be removed")
}

// ---------------------------------------------------------------------------
// prepareResponse
// ---------------------------------------------------------------------------

func TestWebhookManager_prepareResponse_FirstEntryJson(t *testing.T) {
	mgr, _, _ := newTestManager()

	wh := &Webhook{ResponseData: "firstEntryJson"}
	result := &engine.ExecutionResult{
		Data: []model.DataItem{
			{JSON: map[string]interface{}{"first": true}},
			{JSON: map[string]interface{}{"second": true}},
		},
	}

	resp := mgr.prepareResponse(wh, result)
	assert.Equal(t, 200, resp.StatusCode)
	body, ok := resp.Body.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, body["first"])
}

func TestWebhookManager_prepareResponse_FirstEntryJson_Empty(t *testing.T) {
	mgr, _, _ := newTestManager()

	wh := &Webhook{ResponseData: "firstEntryJson"}
	result := &engine.ExecutionResult{Data: []model.DataItem{}}

	resp := mgr.prepareResponse(wh, result)
	body, ok := resp.Body.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "success", body["message"])
}

func TestWebhookManager_prepareResponse_AllEntries(t *testing.T) {
	mgr, _, _ := newTestManager()

	wh := &Webhook{ResponseData: "allEntries"}
	result := &engine.ExecutionResult{
		Data: []model.DataItem{
			{JSON: map[string]interface{}{"a": 1}},
			{JSON: map[string]interface{}{"b": 2}},
		},
	}

	resp := mgr.prepareResponse(wh, result)
	body, ok := resp.Body.([]map[string]interface{})
	require.True(t, ok)
	assert.Len(t, body, 2)
}

func TestWebhookManager_prepareResponse_NoData(t *testing.T) {
	mgr, _, _ := newTestManager()

	wh := &Webhook{ResponseData: "noData"}
	result := &engine.ExecutionResult{
		Data: []model.DataItem{
			{JSON: map[string]interface{}{"ignored": true}},
		},
	}

	resp := mgr.prepareResponse(wh, result)
	body, ok := resp.Body.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "success", body["message"])
}

func TestWebhookManager_prepareResponse_CustomHeaders(t *testing.T) {
	mgr, _, _ := newTestManager()

	wh := &Webhook{
		ResponseData:    "noData",
		ResponseHeaders: map[string]string{"X-Custom": "value"},
	}
	result := &engine.ExecutionResult{Data: []model.DataItem{}}

	resp := mgr.prepareResponse(wh, result)
	assert.Equal(t, "value", resp.Headers["X-Custom"])
	assert.Equal(t, "application/json", resp.Headers["Content-Type"])
}

// ---------------------------------------------------------------------------
// prepareInputData
// ---------------------------------------------------------------------------

func TestWebhookManager_prepareInputData(t *testing.T) {
	mgr, _, _ := newTestManager()

	req := &WebhookRequest{
		Method:  "POST",
		Path:    "/test",
		Headers: map[string][]string{"Content-Type": {"application/json"}},
		Query:   map[string][]string{"page": {"1"}},
		Body:    map[string]interface{}{"data": "hello"},
	}

	items := mgr.prepareInputData(req)
	require.Len(t, items, 1)

	json := items[0].JSON
	assert.Equal(t, "POST", json["method"])
	assert.Equal(t, "/test", json["path"])
	assert.NotNil(t, json["headers"])
	assert.NotNil(t, json["query"])
	assert.NotNil(t, json["body"])
	// "params" is also set (same as query).
	assert.NotNil(t, json["params"])
}
