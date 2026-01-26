package storage

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neul-labs/m9m/internal/model"
)

// Test NewMemoryStorage

func TestNewMemoryStorage(t *testing.T) {
	store := NewMemoryStorage()
	require.NotNil(t, store)
	assert.NotNil(t, store.workflows)
	assert.NotNil(t, store.executions)
	assert.NotNil(t, store.credentials)
	assert.NotNil(t, store.tags)
	assert.NotNil(t, store.rawData)
}

// Workflow Tests

func TestSaveWorkflow(t *testing.T) {
	store := NewMemoryStorage()

	workflow := &model.Workflow{
		ID:   "workflow-1",
		Name: "Test Workflow",
	}

	err := store.SaveWorkflow(workflow)
	require.NoError(t, err)

	// Verify timestamps were set
	assert.False(t, workflow.CreatedAt.IsZero())
	assert.False(t, workflow.UpdatedAt.IsZero())
}

func TestSaveWorkflow_GeneratesID(t *testing.T) {
	store := NewMemoryStorage()

	workflow := &model.Workflow{
		Name: "Test Workflow",
	}

	err := store.SaveWorkflow(workflow)
	require.NoError(t, err)

	assert.NotEmpty(t, workflow.ID)
	assert.Contains(t, workflow.ID, "workflow_")
}

func TestGetWorkflow(t *testing.T) {
	store := NewMemoryStorage()

	workflow := &model.Workflow{
		ID:   "workflow-1",
		Name: "Test Workflow",
	}
	store.SaveWorkflow(workflow)

	retrieved, err := store.GetWorkflow("workflow-1")
	require.NoError(t, err)
	assert.Equal(t, "workflow-1", retrieved.ID)
	assert.Equal(t, "Test Workflow", retrieved.Name)
}

func TestGetWorkflow_NotFound(t *testing.T) {
	store := NewMemoryStorage()

	_, err := store.GetWorkflow("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestListWorkflows_Empty(t *testing.T) {
	store := NewMemoryStorage()

	workflows, total, err := store.ListWorkflows(WorkflowFilters{})
	require.NoError(t, err)
	assert.Empty(t, workflows)
	assert.Equal(t, 0, total)
}

func TestListWorkflows_Multiple(t *testing.T) {
	store := NewMemoryStorage()

	for i := 0; i < 5; i++ {
		store.SaveWorkflow(&model.Workflow{
			ID:   string(rune('a' + i)),
			Name: "Workflow " + string(rune('A'+i)),
		})
	}

	workflows, total, err := store.ListWorkflows(WorkflowFilters{})
	require.NoError(t, err)
	assert.Len(t, workflows, 5)
	assert.Equal(t, 5, total)
}

func TestListWorkflows_FilterByActive(t *testing.T) {
	store := NewMemoryStorage()

	store.SaveWorkflow(&model.Workflow{ID: "active-1", Name: "Active", Active: true})
	store.SaveWorkflow(&model.Workflow{ID: "inactive-1", Name: "Inactive", Active: false})

	active := true
	workflows, total, err := store.ListWorkflows(WorkflowFilters{Active: &active})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.True(t, workflows[0].Active)
}

func TestListWorkflows_FilterBySearch(t *testing.T) {
	store := NewMemoryStorage()

	store.SaveWorkflow(&model.Workflow{ID: "1", Name: "Email Automation"})
	store.SaveWorkflow(&model.Workflow{ID: "2", Name: "Data Processing"})
	store.SaveWorkflow(&model.Workflow{ID: "3", Name: "Email Notification"})

	workflows, total, err := store.ListWorkflows(WorkflowFilters{Search: "email"})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, workflows, 2)
}

func TestListWorkflows_FilterByTags(t *testing.T) {
	store := NewMemoryStorage()

	store.SaveWorkflow(&model.Workflow{ID: "1", Name: "Workflow 1", Tags: []string{"production"}})
	store.SaveWorkflow(&model.Workflow{ID: "2", Name: "Workflow 2", Tags: []string{"staging"}})
	store.SaveWorkflow(&model.Workflow{ID: "3", Name: "Workflow 3", Tags: []string{"production", "critical"}})

	workflows, total, err := store.ListWorkflows(WorkflowFilters{Tags: []string{"production"}})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, workflows, 2)
}

func TestListWorkflows_Pagination(t *testing.T) {
	store := NewMemoryStorage()

	for i := 0; i < 10; i++ {
		store.SaveWorkflow(&model.Workflow{
			ID:   string(rune('a' + i)),
			Name: "Workflow",
		})
	}

	workflows, total, err := store.ListWorkflows(WorkflowFilters{Limit: 3, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 10, total)
	assert.Len(t, workflows, 3)

	// Second page
	workflows, total, err = store.ListWorkflows(WorkflowFilters{Limit: 3, Offset: 3})
	require.NoError(t, err)
	assert.Equal(t, 10, total)
	assert.Len(t, workflows, 3)
}

func TestListWorkflows_PaginationBeyondEnd(t *testing.T) {
	store := NewMemoryStorage()

	for i := 0; i < 5; i++ {
		store.SaveWorkflow(&model.Workflow{ID: string(rune('a' + i))})
	}

	workflows, total, err := store.ListWorkflows(WorkflowFilters{Limit: 3, Offset: 10})
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Empty(t, workflows)
}

func TestUpdateWorkflow(t *testing.T) {
	store := NewMemoryStorage()

	store.SaveWorkflow(&model.Workflow{ID: "workflow-1", Name: "Original"})

	err := store.UpdateWorkflow("workflow-1", &model.Workflow{Name: "Updated"})
	require.NoError(t, err)

	retrieved, _ := store.GetWorkflow("workflow-1")
	assert.Equal(t, "Updated", retrieved.Name)
}

func TestUpdateWorkflow_NotFound(t *testing.T) {
	store := NewMemoryStorage()

	err := store.UpdateWorkflow("nonexistent", &model.Workflow{Name: "Test"})
	assert.Error(t, err)
}

func TestDeleteWorkflow(t *testing.T) {
	store := NewMemoryStorage()

	store.SaveWorkflow(&model.Workflow{ID: "workflow-1"})

	err := store.DeleteWorkflow("workflow-1")
	require.NoError(t, err)

	_, err = store.GetWorkflow("workflow-1")
	assert.Error(t, err)
}

func TestDeleteWorkflow_NotFound(t *testing.T) {
	store := NewMemoryStorage()

	err := store.DeleteWorkflow("nonexistent")
	assert.Error(t, err)
}

func TestActivateWorkflow(t *testing.T) {
	store := NewMemoryStorage()

	store.SaveWorkflow(&model.Workflow{ID: "workflow-1", Active: false})

	err := store.ActivateWorkflow("workflow-1")
	require.NoError(t, err)

	retrieved, _ := store.GetWorkflow("workflow-1")
	assert.True(t, retrieved.Active)
}

func TestActivateWorkflow_NotFound(t *testing.T) {
	store := NewMemoryStorage()

	err := store.ActivateWorkflow("nonexistent")
	assert.Error(t, err)
}

func TestDeactivateWorkflow(t *testing.T) {
	store := NewMemoryStorage()

	store.SaveWorkflow(&model.Workflow{ID: "workflow-1", Active: true})

	err := store.DeactivateWorkflow("workflow-1")
	require.NoError(t, err)

	retrieved, _ := store.GetWorkflow("workflow-1")
	assert.False(t, retrieved.Active)
}

func TestDeactivateWorkflow_NotFound(t *testing.T) {
	store := NewMemoryStorage()

	err := store.DeactivateWorkflow("nonexistent")
	assert.Error(t, err)
}

// Execution Tests

func TestSaveExecution(t *testing.T) {
	store := NewMemoryStorage()

	execution := &model.WorkflowExecution{
		ID:         "exec-1",
		WorkflowID: "workflow-1",
		Status:     "running",
		StartedAt:  time.Now(),
	}

	err := store.SaveExecution(execution)
	require.NoError(t, err)
}

func TestSaveExecution_GeneratesID(t *testing.T) {
	store := NewMemoryStorage()

	execution := &model.WorkflowExecution{
		WorkflowID: "workflow-1",
		Status:     "running",
	}

	err := store.SaveExecution(execution)
	require.NoError(t, err)

	assert.NotEmpty(t, execution.ID)
	assert.Contains(t, execution.ID, "exec_")
}

func TestGetExecution(t *testing.T) {
	store := NewMemoryStorage()

	execution := &model.WorkflowExecution{
		ID:         "exec-1",
		WorkflowID: "workflow-1",
		Status:     "completed",
	}
	store.SaveExecution(execution)

	retrieved, err := store.GetExecution("exec-1")
	require.NoError(t, err)
	assert.Equal(t, "exec-1", retrieved.ID)
	assert.Equal(t, "completed", retrieved.Status)
}

func TestGetExecution_NotFound(t *testing.T) {
	store := NewMemoryStorage()

	_, err := store.GetExecution("nonexistent")
	assert.Error(t, err)
}

func TestListExecutions_Empty(t *testing.T) {
	store := NewMemoryStorage()

	executions, total, err := store.ListExecutions(ExecutionFilters{})
	require.NoError(t, err)
	assert.Empty(t, executions)
	assert.Equal(t, 0, total)
}

func TestListExecutions_FilterByWorkflowID(t *testing.T) {
	store := NewMemoryStorage()

	store.SaveExecution(&model.WorkflowExecution{ID: "e1", WorkflowID: "w1"})
	store.SaveExecution(&model.WorkflowExecution{ID: "e2", WorkflowID: "w2"})
	store.SaveExecution(&model.WorkflowExecution{ID: "e3", WorkflowID: "w1"})

	executions, total, err := store.ListExecutions(ExecutionFilters{WorkflowID: "w1"})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, executions, 2)
}

func TestListExecutions_FilterByStatus(t *testing.T) {
	store := NewMemoryStorage()

	store.SaveExecution(&model.WorkflowExecution{ID: "e1", Status: "completed"})
	store.SaveExecution(&model.WorkflowExecution{ID: "e2", Status: "failed"})
	store.SaveExecution(&model.WorkflowExecution{ID: "e3", Status: "completed"})

	executions, total, err := store.ListExecutions(ExecutionFilters{Status: "completed"})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, executions, 2)
}

func TestListExecutions_Pagination(t *testing.T) {
	store := NewMemoryStorage()

	for i := 0; i < 10; i++ {
		store.SaveExecution(&model.WorkflowExecution{ID: string(rune('a' + i))})
	}

	executions, total, err := store.ListExecutions(ExecutionFilters{Limit: 3, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 10, total)
	assert.Len(t, executions, 3)
}

func TestDeleteExecution(t *testing.T) {
	store := NewMemoryStorage()

	store.SaveExecution(&model.WorkflowExecution{ID: "exec-1"})

	err := store.DeleteExecution("exec-1")
	require.NoError(t, err)

	_, err = store.GetExecution("exec-1")
	assert.Error(t, err)
}

func TestDeleteExecution_NotFound(t *testing.T) {
	store := NewMemoryStorage()

	err := store.DeleteExecution("nonexistent")
	assert.Error(t, err)
}

// Credential Tests

func TestSaveCredential(t *testing.T) {
	store := NewMemoryStorage()

	credential := &Credential{
		ID:   "cred-1",
		Name: "Test Credential",
		Type: "oauth2",
	}

	err := store.SaveCredential(credential)
	require.NoError(t, err)

	assert.False(t, credential.CreatedAt.IsZero())
	assert.False(t, credential.UpdatedAt.IsZero())
}

func TestSaveCredential_GeneratesID(t *testing.T) {
	store := NewMemoryStorage()

	credential := &Credential{
		Name: "Test Credential",
		Type: "oauth2",
	}

	err := store.SaveCredential(credential)
	require.NoError(t, err)

	assert.NotEmpty(t, credential.ID)
	assert.Contains(t, credential.ID, "cred_")
}

func TestGetCredential(t *testing.T) {
	store := NewMemoryStorage()

	store.SaveCredential(&Credential{ID: "cred-1", Name: "Test"})

	credential, err := store.GetCredential("cred-1")
	require.NoError(t, err)
	assert.Equal(t, "cred-1", credential.ID)
}

func TestGetCredential_NotFound(t *testing.T) {
	store := NewMemoryStorage()

	_, err := store.GetCredential("nonexistent")
	assert.Error(t, err)
}

func TestListCredentials(t *testing.T) {
	store := NewMemoryStorage()

	store.SaveCredential(&Credential{ID: "c1", Name: "Cred 1"})
	store.SaveCredential(&Credential{ID: "c2", Name: "Cred 2"})

	credentials, err := store.ListCredentials()
	require.NoError(t, err)
	assert.Len(t, credentials, 2)
}

func TestUpdateCredential(t *testing.T) {
	store := NewMemoryStorage()

	store.SaveCredential(&Credential{ID: "cred-1", Name: "Original"})

	err := store.UpdateCredential("cred-1", &Credential{Name: "Updated"})
	require.NoError(t, err)

	credential, _ := store.GetCredential("cred-1")
	assert.Equal(t, "Updated", credential.Name)
}

func TestUpdateCredential_NotFound(t *testing.T) {
	store := NewMemoryStorage()

	err := store.UpdateCredential("nonexistent", &Credential{Name: "Test"})
	assert.Error(t, err)
}

func TestDeleteCredential(t *testing.T) {
	store := NewMemoryStorage()

	store.SaveCredential(&Credential{ID: "cred-1"})

	err := store.DeleteCredential("cred-1")
	require.NoError(t, err)

	_, err = store.GetCredential("cred-1")
	assert.Error(t, err)
}

func TestDeleteCredential_NotFound(t *testing.T) {
	store := NewMemoryStorage()

	err := store.DeleteCredential("nonexistent")
	assert.Error(t, err)
}

// Tag Tests

func TestSaveTag(t *testing.T) {
	store := NewMemoryStorage()

	tag := &Tag{
		ID:   "tag-1",
		Name: "Production",
	}

	err := store.SaveTag(tag)
	require.NoError(t, err)

	assert.False(t, tag.CreatedAt.IsZero())
	assert.False(t, tag.UpdatedAt.IsZero())
}

func TestSaveTag_GeneratesID(t *testing.T) {
	store := NewMemoryStorage()

	tag := &Tag{Name: "Production"}

	err := store.SaveTag(tag)
	require.NoError(t, err)

	assert.NotEmpty(t, tag.ID)
	assert.Contains(t, tag.ID, "tag_")
}

func TestGetTag(t *testing.T) {
	store := NewMemoryStorage()

	store.SaveTag(&Tag{ID: "tag-1", Name: "Production"})

	tag, err := store.GetTag("tag-1")
	require.NoError(t, err)
	assert.Equal(t, "tag-1", tag.ID)
	assert.Equal(t, "Production", tag.Name)
}

func TestGetTag_NotFound(t *testing.T) {
	store := NewMemoryStorage()

	_, err := store.GetTag("nonexistent")
	assert.Error(t, err)
}

func TestListTags(t *testing.T) {
	store := NewMemoryStorage()

	store.SaveTag(&Tag{ID: "t1", Name: "Tag 1"})
	store.SaveTag(&Tag{ID: "t2", Name: "Tag 2"})

	tags, err := store.ListTags()
	require.NoError(t, err)
	assert.Len(t, tags, 2)
}

func TestUpdateTag(t *testing.T) {
	store := NewMemoryStorage()

	store.SaveTag(&Tag{ID: "tag-1", Name: "Original"})

	err := store.UpdateTag("tag-1", &Tag{Name: "Updated"})
	require.NoError(t, err)

	tag, _ := store.GetTag("tag-1")
	assert.Equal(t, "Updated", tag.Name)
}

func TestUpdateTag_NotFound(t *testing.T) {
	store := NewMemoryStorage()

	err := store.UpdateTag("nonexistent", &Tag{Name: "Test"})
	assert.Error(t, err)
}

func TestDeleteTag(t *testing.T) {
	store := NewMemoryStorage()

	store.SaveTag(&Tag{ID: "tag-1"})

	err := store.DeleteTag("tag-1")
	require.NoError(t, err)

	_, err = store.GetTag("tag-1")
	assert.Error(t, err)
}

func TestDeleteTag_NotFound(t *testing.T) {
	store := NewMemoryStorage()

	err := store.DeleteTag("nonexistent")
	assert.Error(t, err)
}

// Raw Key-Value Tests

func TestSaveRaw(t *testing.T) {
	store := NewMemoryStorage()

	data := []byte("test data")
	err := store.SaveRaw("key-1", data)
	require.NoError(t, err)
}

func TestGetRaw(t *testing.T) {
	store := NewMemoryStorage()

	data := []byte("test data")
	store.SaveRaw("key-1", data)

	retrieved, err := store.GetRaw("key-1")
	require.NoError(t, err)
	assert.Equal(t, data, retrieved)
}

func TestGetRaw_NotFound(t *testing.T) {
	store := NewMemoryStorage()

	_, err := store.GetRaw("nonexistent")
	assert.Error(t, err)
}

func TestGetRaw_IsCopy(t *testing.T) {
	store := NewMemoryStorage()

	data := []byte("original")
	store.SaveRaw("key-1", data)

	retrieved, _ := store.GetRaw("key-1")
	retrieved[0] = 'X' // Modify the retrieved data

	// Original should be unchanged
	original, _ := store.GetRaw("key-1")
	assert.Equal(t, byte('o'), original[0])
}

func TestListKeys(t *testing.T) {
	store := NewMemoryStorage()

	store.SaveRaw("webhook:1", []byte("data1"))
	store.SaveRaw("webhook:2", []byte("data2"))
	store.SaveRaw("settings:1", []byte("data3"))

	keys, err := store.ListKeys("webhook:")
	require.NoError(t, err)
	assert.Len(t, keys, 2)

	for _, key := range keys {
		assert.Contains(t, key, "webhook:")
	}
}

func TestListKeys_NoMatch(t *testing.T) {
	store := NewMemoryStorage()

	store.SaveRaw("key-1", []byte("data"))

	keys, err := store.ListKeys("nonexistent:")
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestDeleteRaw(t *testing.T) {
	store := NewMemoryStorage()

	store.SaveRaw("key-1", []byte("data"))

	err := store.DeleteRaw("key-1")
	require.NoError(t, err)

	_, err = store.GetRaw("key-1")
	assert.Error(t, err)
}

func TestDeleteRaw_NotFound(t *testing.T) {
	store := NewMemoryStorage()

	err := store.DeleteRaw("nonexistent")
	assert.Error(t, err)
}

// Close Test

func TestClose(t *testing.T) {
	store := NewMemoryStorage()

	err := store.Close()
	require.NoError(t, err)
}

// Concurrency Tests

func TestConcurrentWorkflowOperations(t *testing.T) {
	store := NewMemoryStorage()

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent writes
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			store.SaveWorkflow(&model.Workflow{
				ID:   string(rune('a' + (id % 26))),
				Name: "Workflow",
			})
		}(i)
	}

	wg.Wait()

	// Should not panic and workflows should be accessible
	workflows, _, err := store.ListWorkflows(WorkflowFilters{})
	require.NoError(t, err)
	assert.NotEmpty(t, workflows)
}

func TestConcurrentReadWrite(t *testing.T) {
	store := NewMemoryStorage()

	// Pre-populate
	store.SaveWorkflow(&model.Workflow{ID: "workflow-1", Name: "Test"})

	var wg sync.WaitGroup
	iterations := 50

	// Concurrent reads
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.GetWorkflow("workflow-1")
		}()
	}

	// Concurrent writes
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			store.UpdateWorkflow("workflow-1", &model.Workflow{Name: "Updated"})
		}(i)
	}

	wg.Wait()

	// Should not panic
	workflow, err := store.GetWorkflow("workflow-1")
	require.NoError(t, err)
	assert.NotNil(t, workflow)
}

// Interface Compliance Test

func TestMemoryStorageImplementsInterface(t *testing.T) {
	var _ WorkflowStorage = (*MemoryStorage)(nil)
}
