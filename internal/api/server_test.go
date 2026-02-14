package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neul-labs/m9m/internal/connections"
	"github.com/neul-labs/m9m/internal/credentials"
	"github.com/neul-labs/m9m/internal/engine"
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
	"github.com/neul-labs/m9m/internal/storage"
)

// MockWorkflowEngine implements engine.WorkflowEngine for testing
type MockWorkflowEngine struct {
	executeResult          *engine.ExecutionResult
	executeError           error
	executeWithContextFunc func(ctx context.Context, workflow *model.Workflow, inputData []model.DataItem) (*engine.ExecutionResult, error)
	lastWorkflow           *model.Workflow
	lastInputData          []model.DataItem
}

func (m *MockWorkflowEngine) ExecuteWorkflow(workflow *model.Workflow, inputData []model.DataItem) (*engine.ExecutionResult, error) {
	if m.executeWithContextFunc != nil {
		return m.executeWithContextFunc(context.Background(), workflow, inputData)
	}
	m.lastWorkflow = workflow
	m.lastInputData = inputData
	if m.executeError != nil {
		return nil, m.executeError
	}
	if m.executeResult != nil {
		return m.executeResult, nil
	}
	return &engine.ExecutionResult{
		Data: []model.DataItem{{JSON: map[string]interface{}{"ok": true}}},
	}, nil
}

func (m *MockWorkflowEngine) ExecuteWorkflowWithContext(ctx context.Context, workflow *model.Workflow, inputData []model.DataItem) (*engine.ExecutionResult, error) {
	m.lastWorkflow = workflow
	m.lastInputData = inputData
	if m.executeWithContextFunc != nil {
		return m.executeWithContextFunc(ctx, workflow, inputData)
	}
	return m.ExecuteWorkflow(workflow, inputData)
}

func (m *MockWorkflowEngine) ExecuteWorkflowParallel(workflows []*model.Workflow, inputData [][]model.DataItem) ([]*engine.ExecutionResult, error) {
	return nil, nil
}

func (m *MockWorkflowEngine) RegisterNodeExecutor(nodeType string, executor base.NodeExecutor) {}

func (m *MockWorkflowEngine) GetNodeExecutor(nodeType string) (base.NodeExecutor, error) {
	return nil, nil
}

func (m *MockWorkflowEngine) SetCredentialManager(credentialManager *credentials.CredentialManager) {}

func (m *MockWorkflowEngine) SetConnectionRouter(connectionRouter connections.ConnectionRouter) {}

// Helper to create a test server
func setupTestServer(t *testing.T) (*APIServer, *mux.Router, *storage.MemoryStorage) {
	store := storage.NewMemoryStorage()
	server := NewAPIServer(nil, nil, store)

	router := mux.NewRouter()
	server.RegisterRoutes(router)

	return server, router, store
}

// Health Check Tests

func TestHealthCheck(t *testing.T) {
	_, router, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response["status"])
	assert.Equal(t, "m9m", response["service"])
	assert.NotEmpty(t, response["version"])
	assert.NotEmpty(t, response["time"])
}

func TestHealthzCheck(t *testing.T) {
	_, router, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReadyCheck_NotReady(t *testing.T) {
	_, router, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Server is not ready when engine is nil
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "not ready", response["status"])
}

func TestReadyCheck_Ready(t *testing.T) {
	store := storage.NewMemoryStorage()
	engine := &MockWorkflowEngine{}
	server := NewAPIServer(engine, nil, store)

	router := mux.NewRouter()
	server.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "ready", response["status"])
}

// Workflow CRUD Tests

func TestListWorkflows_Empty(t *testing.T) {
	_, router, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/workflows", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(0), response["total"])
	assert.NotNil(t, response["data"])
}

func TestCreateWorkflow(t *testing.T) {
	_, router, _ := setupTestServer(t)

	workflow := model.Workflow{
		ID:     "test-workflow-1",
		Name:   "Test Workflow",
		Active: false,
		Nodes: []model.Node{
			{
				ID:   "node1",
				Name: "Start",
				Type: "n8n-nodes-base.start",
			},
		},
	}

	body, _ := json.Marshal(workflow)
	req := httptest.NewRequest("POST", "/api/v1/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response model.Workflow
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Test Workflow", response.Name)
}

func TestCreateWorkflow_InvalidJSON(t *testing.T) {
	_, router, _ := setupTestServer(t)

	req := httptest.NewRequest("POST", "/api/v1/workflows", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["error"].(bool))
}

func TestGetWorkflow(t *testing.T) {
	_, router, store := setupTestServer(t)

	// First create a workflow
	workflow := &model.Workflow{
		ID:     "test-workflow-1",
		Name:   "Test Workflow",
		Active: false,
	}
	store.SaveWorkflow(workflow)

	req := httptest.NewRequest("GET", "/api/v1/workflows/test-workflow-1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.Workflow
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "test-workflow-1", response.ID)
	assert.Equal(t, "Test Workflow", response.Name)
}

func TestGetWorkflow_NotFound(t *testing.T) {
	_, router, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/workflows/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateWorkflow(t *testing.T) {
	_, router, store := setupTestServer(t)

	// First create a workflow
	workflow := &model.Workflow{
		ID:     "test-workflow-1",
		Name:   "Original Name",
		Active: false,
	}
	store.SaveWorkflow(workflow)

	// Update it
	updatedWorkflow := model.Workflow{
		ID:     "test-workflow-1",
		Name:   "Updated Name",
		Active: true,
	}

	body, _ := json.Marshal(updatedWorkflow)
	req := httptest.NewRequest("PUT", "/api/v1/workflows/test-workflow-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.Workflow
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Updated Name", response.Name)
}

func TestDeleteWorkflow(t *testing.T) {
	_, router, store := setupTestServer(t)

	// First create a workflow
	workflow := &model.Workflow{
		ID:     "test-workflow-1",
		Name:   "Test Workflow",
		Active: false,
	}
	store.SaveWorkflow(workflow)

	req := httptest.NewRequest("DELETE", "/api/v1/workflows/test-workflow-1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify workflow is deleted
	_, err := store.GetWorkflow("test-workflow-1")
	assert.Error(t, err)
}

func TestDeleteWorkflow_NotFound(t *testing.T) {
	_, router, _ := setupTestServer(t)

	req := httptest.NewRequest("DELETE", "/api/v1/workflows/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestActivateWorkflow(t *testing.T) {
	_, router, store := setupTestServer(t)

	// First create an inactive workflow
	workflow := &model.Workflow{
		ID:     "test-workflow-1",
		Name:   "Test Workflow",
		Active: false,
	}
	store.SaveWorkflow(workflow)

	req := httptest.NewRequest("POST", "/api/v1/workflows/test-workflow-1/activate", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify workflow is activated
	updated, _ := store.GetWorkflow("test-workflow-1")
	assert.True(t, updated.Active)
}

func TestDeactivateWorkflow(t *testing.T) {
	_, router, store := setupTestServer(t)

	// First create an active workflow
	workflow := &model.Workflow{
		ID:     "test-workflow-1",
		Name:   "Test Workflow",
		Active: true,
	}
	store.SaveWorkflow(workflow)

	req := httptest.NewRequest("POST", "/api/v1/workflows/test-workflow-1/deactivate", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify workflow is deactivated
	updated, _ := store.GetWorkflow("test-workflow-1")
	assert.False(t, updated.Active)
}

// Execution Tests

func TestListExecutions_Empty(t *testing.T) {
	_, router, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/executions", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(0), response["total"])
}

func TestGetExecution(t *testing.T) {
	_, router, store := setupTestServer(t)

	// Create an execution
	execution := &model.WorkflowExecution{
		ID:         "exec-1",
		WorkflowID: "workflow-1",
		Status:     "completed",
		StartedAt:  time.Now(),
	}
	store.SaveExecution(execution)

	req := httptest.NewRequest("GET", "/api/v1/executions/exec-1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response model.WorkflowExecution
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "exec-1", response.ID)
	assert.Equal(t, "completed", response.Status)
}

func TestGetExecution_NotFound(t *testing.T) {
	_, router, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/executions/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteExecution(t *testing.T) {
	_, router, store := setupTestServer(t)

	// Create an execution
	execution := &model.WorkflowExecution{
		ID:         "exec-1",
		WorkflowID: "workflow-1",
		Status:     "completed",
		StartedAt:  time.Now(),
	}
	store.SaveExecution(execution)

	req := httptest.NewRequest("DELETE", "/api/v1/executions/exec-1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify execution is deleted
	_, err := store.GetExecution("exec-1")
	assert.Error(t, err)
}

func TestCancelExecution_NotRunning(t *testing.T) {
	_, router, store := setupTestServer(t)

	// Create a completed execution
	execution := &model.WorkflowExecution{
		ID:         "exec-1",
		WorkflowID: "workflow-1",
		Status:     "completed",
		StartedAt:  time.Now(),
	}
	store.SaveExecution(execution)

	req := httptest.NewRequest("POST", "/api/v1/executions/exec-1/cancel", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCancelExecution_Running(t *testing.T) {
	_, router, store := setupTestServer(t)

	// Create a running execution
	execution := &model.WorkflowExecution{
		ID:         "exec-1",
		WorkflowID: "workflow-1",
		Status:     "running",
		StartedAt:  time.Now(),
	}
	store.SaveExecution(execution)

	req := httptest.NewRequest("POST", "/api/v1/executions/exec-1/cancel", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	// Verify execution state is unchanged
	updated, _ := store.GetExecution("exec-1")
	assert.Equal(t, "running", updated.Status)
	assert.Nil(t, updated.FinishedAt)
}

func TestCancelExecution_TrackedRunning(t *testing.T) {
	store := storage.NewMemoryStorage()
	mockEngine := &MockWorkflowEngine{
		executeWithContextFunc: func(ctx context.Context, workflow *model.Workflow, inputData []model.DataItem) (*engine.ExecutionResult, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}
	server := NewAPIServer(mockEngine, nil, store)

	router := mux.NewRouter()
	server.RegisterRoutes(router)

	store.SaveWorkflow(&model.Workflow{
		ID:   "wf-1",
		Name: "Cancelable Workflow",
		Nodes: []model.Node{
			{ID: "start", Name: "Start", Type: "n8n-nodes-base.start"},
		},
	})

	execResponseChan := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		req := httptest.NewRequest("POST", "/api/v1/workflows/wf-1/execute", bytes.NewReader([]byte(`[]`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		execResponseChan <- w
	}()

	var executionID string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		executions, _, err := store.ListExecutions(storage.ExecutionFilters{Status: "running"})
		require.NoError(t, err)
		if len(executions) > 0 {
			executionID = executions[0].ID
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	require.NotEmpty(t, executionID)

	cancelReq := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/executions/%s/cancel", executionID), nil)
	cancelResp := httptest.NewRecorder()
	router.ServeHTTP(cancelResp, cancelReq)
	assert.Equal(t, http.StatusAccepted, cancelResp.Code)

	select {
	case execResp := <-execResponseChan:
		assert.Equal(t, http.StatusOK, execResp.Code)
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for execution response")
	}

	var final *model.WorkflowExecution
	for i := 0; i < 100; i++ {
		current, err := store.GetExecution(executionID)
		require.NoError(t, err)
		if current.Status == "cancelled" {
			final = current
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	require.NotNil(t, final)
	assert.Equal(t, "cancelled", final.Status)
	assert.NotNil(t, final.FinishedAt)
}

func TestExecuteWorkflow_InputEnvelope(t *testing.T) {
	store := storage.NewMemoryStorage()
	mockEngine := &MockWorkflowEngine{
		executeResult: &engine.ExecutionResult{
			Data: []model.DataItem{{JSON: map[string]interface{}{"status": "ok"}}},
		},
	}
	server := NewAPIServer(mockEngine, nil, store)

	router := mux.NewRouter()
	server.RegisterRoutes(router)

	store.SaveWorkflow(&model.Workflow{
		ID:   "wf-1",
		Name: "Test",
		Nodes: []model.Node{
			{ID: "start", Name: "Start", Type: "n8n-nodes-base.start"},
		},
	})

	body := []byte(`{"inputData":[{"json":{"hello":"world"}}]}`)
	req := httptest.NewRequest("POST", "/api/v1/workflows/wf-1/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.Len(t, mockEngine.lastInputData, 1)
	assert.Equal(t, "world", mockEngine.lastInputData[0].JSON["hello"])
}

func TestExecuteWorkflow_InvalidInputPayload(t *testing.T) {
	store := storage.NewMemoryStorage()
	mockEngine := &MockWorkflowEngine{}
	server := NewAPIServer(mockEngine, nil, store)

	router := mux.NewRouter()
	server.RegisterRoutes(router)

	store.SaveWorkflow(&model.Workflow{
		ID:   "wf-1",
		Name: "Test",
		Nodes: []model.Node{
			{ID: "start", Name: "Start", Type: "n8n-nodes-base.start"},
		},
	})

	req := httptest.NewRequest("POST", "/api/v1/workflows/wf-1/execute", bytes.NewReader([]byte(`{"inputData"`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExecuteWorkflowByDefinition(t *testing.T) {
	store := storage.NewMemoryStorage()
	mockEngine := &MockWorkflowEngine{
		executeResult: &engine.ExecutionResult{
			Data: []model.DataItem{{JSON: map[string]interface{}{"inline": true}}},
		},
	}
	server := NewAPIServer(mockEngine, nil, store)

	router := mux.NewRouter()
	server.RegisterRoutes(router)

	reqBody := map[string]interface{}{
		"workflow": map[string]interface{}{
			"id":   "wf-inline",
			"name": "Inline Workflow",
			"nodes": []map[string]interface{}{
				{
					"id":   "start",
					"name": "Start",
					"type": "n8n-nodes-base.start",
				},
			},
		},
		"inputData": []map[string]interface{}{
			{
				"json": map[string]interface{}{"value": 42},
			},
		},
	}
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/workflows/run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, mockEngine.lastWorkflow)
	assert.Equal(t, "wf-inline", mockEngine.lastWorkflow.ID)
	require.Len(t, mockEngine.lastInputData, 1)
	assert.Equal(t, float64(42), mockEngine.lastInputData[0].JSON["value"])
}

func TestExecuteWorkflow_ResultErrorMarksFailed(t *testing.T) {
	store := storage.NewMemoryStorage()
	mockEngine := &MockWorkflowEngine{
		executeResult: &engine.ExecutionResult{
			Error: fmt.Errorf("node execution failed"),
		},
	}
	server := NewAPIServer(mockEngine, nil, store)

	router := mux.NewRouter()
	server.RegisterRoutes(router)

	store.SaveWorkflow(&model.Workflow{
		ID:   "wf-1",
		Name: "Test",
		Nodes: []model.Node{
			{ID: "start", Name: "Start", Type: "n8n-nodes-base.start"},
		},
	})

	req := httptest.NewRequest("POST", "/api/v1/workflows/wf-1/execute", bytes.NewReader([]byte(`[]`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var execution map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &execution)
	require.NoError(t, err)
	assert.Equal(t, "failed", execution["status"])
}

// Credential Tests

func TestListCredentials_Empty(t *testing.T) {
	_, router, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/credentials", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCreateCredential(t *testing.T) {
	_, router, _ := setupTestServer(t)

	credential := storage.Credential{
		ID:   "cred-1",
		Name: "Test Credential",
		Type: "oauth2",
		Data: map[string]interface{}{
			"accessToken": "secret-token",
		},
	}

	body, _ := json.Marshal(credential)
	req := httptest.NewRequest("POST", "/api/v1/credentials", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response storage.Credential
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Test Credential", response.Name)
}

func TestGetCredential(t *testing.T) {
	_, router, store := setupTestServer(t)

	// Create a credential
	credential := &storage.Credential{
		ID:   "cred-1",
		Name: "Test Credential",
		Type: "oauth2",
	}
	store.SaveCredential(credential)

	req := httptest.NewRequest("GET", "/api/v1/credentials/cred-1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response storage.Credential
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "cred-1", response.ID)
}

func TestGetCredential_NotFound(t *testing.T) {
	_, router, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/credentials/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateCredential(t *testing.T) {
	_, router, store := setupTestServer(t)

	// Create a credential
	credential := &storage.Credential{
		ID:   "cred-1",
		Name: "Original Name",
		Type: "oauth2",
	}
	store.SaveCredential(credential)

	// Update it
	updatedCredential := storage.Credential{
		ID:   "cred-1",
		Name: "Updated Name",
		Type: "oauth2",
	}

	body, _ := json.Marshal(updatedCredential)
	req := httptest.NewRequest("PUT", "/api/v1/credentials/cred-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response storage.Credential
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Updated Name", response.Name)
}

func TestDeleteCredential(t *testing.T) {
	_, router, store := setupTestServer(t)

	// Create a credential
	credential := &storage.Credential{
		ID:   "cred-1",
		Name: "Test Credential",
		Type: "oauth2",
	}
	store.SaveCredential(credential)

	req := httptest.NewRequest("DELETE", "/api/v1/credentials/cred-1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify credential is deleted
	_, err := store.GetCredential("cred-1")
	assert.Error(t, err)
}

// Settings Tests

func TestGetSettings(t *testing.T) {
	_, router, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/settings", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "UTC", response["timezone"])
	assert.NotEmpty(t, response["executionMode"])
}

func TestUpdateSettings(t *testing.T) {
	_, router, _ := setupTestServer(t)

	settings := map[string]interface{}{
		"timezone":      "America/New_York",
		"executionMode": "queue",
	}

	body, _ := json.Marshal(settings)
	req := httptest.NewRequest("PATCH", "/api/v1/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "America/New_York", response["timezone"])
}

func TestGetLicense(t *testing.T) {
	_, router, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/settings/license", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, false, response["licensed"])
	assert.Equal(t, "community", response["licenseType"])
}

func TestGetLDAP(t *testing.T) {
	_, router, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/settings/ldap", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, false, response["enabled"])
}

// Node Types Tests

func TestListNodeTypes(t *testing.T) {
	_, router, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/node-types", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response)
}

func TestGetNodeType(t *testing.T) {
	_, router, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/node-types/n8n-nodes-base.httpRequest", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "n8n-nodes-base.httpRequest", response["name"])
}

// Version Tests

func TestGetVersion(t *testing.T) {
	_, router, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/version", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Version response should have some fields
	assert.NotNil(t, response)
}

// CORS and OPTIONS Tests

func TestCORSPreflight(t *testing.T) {
	_, router, _ := setupTestServer(t)

	req := httptest.NewRequest("OPTIONS", "/api/v1/workflows", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// OPTIONS requests should be handled
	assert.Contains(t, []int{http.StatusOK, http.StatusMethodNotAllowed}, w.Code)
}

// Pagination Tests

func TestListWorkflows_Pagination(t *testing.T) {
	_, router, store := setupTestServer(t)

	// Create multiple workflows
	for i := 0; i < 5; i++ {
		workflow := &model.Workflow{
			ID:     string(rune('a' + i)),
			Name:   "Workflow " + string(rune('A'+i)),
			Active: false,
		}
		store.SaveWorkflow(workflow)
	}

	req := httptest.NewRequest("GET", "/api/v1/workflows?limit=2&offset=0", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(5), response["total"])
	assert.Equal(t, float64(2), response["limit"])
	assert.Equal(t, float64(0), response["offset"])
}

func TestListWorkflows_FilterByActive(t *testing.T) {
	_, router, store := setupTestServer(t)

	// Create active and inactive workflows
	store.SaveWorkflow(&model.Workflow{ID: "active-1", Name: "Active", Active: true})
	store.SaveWorkflow(&model.Workflow{ID: "inactive-1", Name: "Inactive", Active: false})

	req := httptest.NewRequest("GET", "/api/v1/workflows?active=true", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].([]interface{})
	for _, item := range data {
		workflow := item.(map[string]interface{})
		assert.True(t, workflow["active"].(bool))
	}
}

// Error Handling Tests

func TestUpdateWorkflow_InvalidJSON(t *testing.T) {
	_, router, store := setupTestServer(t)

	// Create a workflow first
	store.SaveWorkflow(&model.Workflow{ID: "test-1", Name: "Test"})

	req := httptest.NewRequest("PUT", "/api/v1/workflows/test-1", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateCredential_InvalidJSON(t *testing.T) {
	_, router, _ := setupTestServer(t)

	req := httptest.NewRequest("POST", "/api/v1/credentials", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSettings_InvalidJSON(t *testing.T) {
	_, router, _ := setupTestServer(t)

	req := httptest.NewRequest("PATCH", "/api/v1/settings", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
