package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dipankar/m9m/internal/connections"
	"github.com/dipankar/m9m/internal/credentials"
	"github.com/dipankar/m9m/internal/engine"
	"github.com/dipankar/m9m/internal/model"
	"github.com/dipankar/m9m/internal/nodes/base"
	"github.com/dipankar/m9m/internal/storage"
)

// MockWorkflowEngine implements engine.WorkflowEngine for testing
type MockWorkflowEngine struct {
	executeResult *engine.ExecutionResult
	executeError  error
	executeCalls  int
}

func (m *MockWorkflowEngine) ExecuteWorkflow(workflow *model.Workflow, inputData []model.DataItem) (*engine.ExecutionResult, error) {
	m.executeCalls++
	if m.executeError != nil {
		return nil, m.executeError
	}
	if m.executeResult != nil {
		return m.executeResult, nil
	}
	return &engine.ExecutionResult{
		Data: []model.DataItem{{JSON: map[string]interface{}{"success": true}}},
	}, nil
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

// Test NewWorkflowScheduler

func TestNewWorkflowScheduler(t *testing.T) {
	eng := &MockWorkflowEngine{}
	scheduler := NewWorkflowScheduler(eng)

	require.NotNil(t, scheduler)
	assert.NotNil(t, scheduler.schedules)
	assert.NotNil(t, scheduler.cronJobs)
	assert.NotNil(t, scheduler.executions)
	assert.NotNil(t, scheduler.metrics)
}

// Test SetStorage

func TestSetStorage(t *testing.T) {
	eng := &MockWorkflowEngine{}
	scheduler := NewWorkflowScheduler(eng)
	store := storage.NewMemoryStorage()

	scheduler.SetStorage(store)

	assert.NotNil(t, scheduler.storage)
}

// Test Schedule CRUD

func TestAddSchedule(t *testing.T) {
	eng := &MockWorkflowEngine{}
	scheduler := NewWorkflowScheduler(eng)

	config := &ScheduleConfig{
		ID:           "schedule-1",
		WorkflowID:   "workflow-1",
		WorkflowName: "Test Workflow",
		CronExpr:     "*/5 * * * *",
		Enabled:      true,
		CreatedAt:    time.Now(),
	}

	err := scheduler.AddSchedule(config)
	require.NoError(t, err)

	schedules := scheduler.ListSchedules()
	assert.Len(t, schedules, 1)
	assert.Equal(t, "schedule-1", schedules[0].ID)
}

func TestAddSchedule_DuplicateID(t *testing.T) {
	eng := &MockWorkflowEngine{}
	scheduler := NewWorkflowScheduler(eng)

	config := &ScheduleConfig{
		ID:         "schedule-1",
		WorkflowID: "workflow-1",
		CronExpr:   "*/5 * * * *",
		Enabled:    true,
	}

	err := scheduler.AddSchedule(config)
	require.NoError(t, err)

	// Adding same ID replaces existing (updates)
	err = scheduler.AddSchedule(config)
	require.NoError(t, err)

	// Should still have only one schedule
	schedules := scheduler.ListSchedules()
	assert.Len(t, schedules, 1)
}

func TestAddSchedule_InvalidCron(t *testing.T) {
	eng := &MockWorkflowEngine{}
	scheduler := NewWorkflowScheduler(eng)

	config := &ScheduleConfig{
		ID:         "schedule-1",
		WorkflowID: "workflow-1",
		CronExpr:   "invalid cron",
		Enabled:    true,
	}

	err := scheduler.AddSchedule(config)
	assert.Error(t, err)
}

func TestListSchedules_Empty(t *testing.T) {
	eng := &MockWorkflowEngine{}
	scheduler := NewWorkflowScheduler(eng)

	schedules := scheduler.ListSchedules()
	assert.Empty(t, schedules)
}

func TestListSchedules_Multiple(t *testing.T) {
	eng := &MockWorkflowEngine{}
	scheduler := NewWorkflowScheduler(eng)

	for i := 0; i < 3; i++ {
		config := &ScheduleConfig{
			ID:         string(rune('a' + i)),
			WorkflowID: "workflow-1",
			CronExpr:   "*/5 * * * *",
			Enabled:    true,
		}
		scheduler.AddSchedule(config)
	}

	schedules := scheduler.ListSchedules()
	assert.Len(t, schedules, 3)
}

func TestGetSchedule(t *testing.T) {
	eng := &MockWorkflowEngine{}
	scheduler := NewWorkflowScheduler(eng)

	config := &ScheduleConfig{
		ID:           "schedule-1",
		WorkflowID:   "workflow-1",
		WorkflowName: "Test Workflow",
		CronExpr:     "*/5 * * * *",
		Enabled:      true,
	}

	scheduler.AddSchedule(config)

	retrieved, err := scheduler.GetSchedule("schedule-1")
	require.NoError(t, err)
	assert.Equal(t, "schedule-1", retrieved.ID)
	assert.Equal(t, "Test Workflow", retrieved.WorkflowName)
}

func TestGetSchedule_NotFound(t *testing.T) {
	eng := &MockWorkflowEngine{}
	scheduler := NewWorkflowScheduler(eng)

	_, err := scheduler.GetSchedule("nonexistent")
	assert.Error(t, err)
}

func TestUpdateSchedule(t *testing.T) {
	eng := &MockWorkflowEngine{}
	scheduler := NewWorkflowScheduler(eng)

	config := &ScheduleConfig{
		ID:         "schedule-1",
		WorkflowID: "workflow-1",
		CronExpr:   "*/5 * * * *",
		Enabled:    true,
	}

	scheduler.AddSchedule(config)

	updatedConfig := &ScheduleConfig{
		ID:         "schedule-1",
		WorkflowID: "workflow-1",
		CronExpr:   "*/10 * * * *",
		Enabled:    false,
	}

	err := scheduler.UpdateSchedule("schedule-1", updatedConfig)
	require.NoError(t, err)

	retrieved, _ := scheduler.GetSchedule("schedule-1")
	assert.Equal(t, "*/10 * * * *", retrieved.CronExpr)
	assert.False(t, retrieved.Enabled)
}

func TestUpdateSchedule_NotFound(t *testing.T) {
	eng := &MockWorkflowEngine{}
	scheduler := NewWorkflowScheduler(eng)

	config := &ScheduleConfig{
		ID:         "schedule-1",
		WorkflowID: "workflow-1",
		CronExpr:   "*/5 * * * *",
	}

	err := scheduler.UpdateSchedule("nonexistent", config)
	assert.Error(t, err)
}

func TestRemoveSchedule(t *testing.T) {
	eng := &MockWorkflowEngine{}
	scheduler := NewWorkflowScheduler(eng)

	config := &ScheduleConfig{
		ID:         "schedule-1",
		WorkflowID: "workflow-1",
		CronExpr:   "*/5 * * * *",
		Enabled:    true,
	}

	scheduler.AddSchedule(config)

	err := scheduler.RemoveSchedule("schedule-1")
	require.NoError(t, err)

	schedules := scheduler.ListSchedules()
	assert.Empty(t, schedules)
}

func TestRemoveSchedule_NotFound(t *testing.T) {
	eng := &MockWorkflowEngine{}
	scheduler := NewWorkflowScheduler(eng)

	err := scheduler.RemoveSchedule("nonexistent")
	assert.Error(t, err)
}

// Test Execution History

func TestGetExecutionHistory_Empty(t *testing.T) {
	eng := &MockWorkflowEngine{}
	scheduler := NewWorkflowScheduler(eng)

	config := &ScheduleConfig{
		ID:         "schedule-1",
		WorkflowID: "workflow-1",
		CronExpr:   "*/5 * * * *",
		Enabled:    true,
	}
	scheduler.AddSchedule(config)

	history, err := scheduler.GetExecutionHistory("schedule-1", 10)
	require.NoError(t, err)
	assert.Empty(t, history.Executions)
}

func TestGetExecutionHistory_NotFound(t *testing.T) {
	eng := &MockWorkflowEngine{}
	scheduler := NewWorkflowScheduler(eng)

	_, err := scheduler.GetExecutionHistory("nonexistent", 10)
	assert.Error(t, err)
}

// Test Metrics

func TestGetMetrics(t *testing.T) {
	eng := &MockWorkflowEngine{}
	scheduler := NewWorkflowScheduler(eng)

	metrics := scheduler.GetMetrics()
	require.NotNil(t, metrics)
	assert.Equal(t, 0, metrics.TotalSchedules)
	assert.Equal(t, 0, metrics.ActiveSchedules)
}

func TestGetMetrics_WithSchedules(t *testing.T) {
	eng := &MockWorkflowEngine{}
	scheduler := NewWorkflowScheduler(eng)

	for i := 0; i < 3; i++ {
		config := &ScheduleConfig{
			ID:         string(rune('a' + i)),
			WorkflowID: "workflow-1",
			CronExpr:   "*/5 * * * *",
			Enabled:    i < 2,
		}
		scheduler.AddSchedule(config)
	}

	metrics := scheduler.GetMetrics()
	assert.Equal(t, 3, metrics.TotalSchedules)
	assert.Equal(t, 2, metrics.ActiveSchedules)
}

// Test Start/Stop

func TestStartStop(t *testing.T) {
	eng := &MockWorkflowEngine{}
	scheduler := NewWorkflowScheduler(eng)

	err := scheduler.Start()
	require.NoError(t, err)

	err = scheduler.Stop()
	require.NoError(t, err)
}

// Test Workflow Loading

func TestLoadWorkflow_NoStorage(t *testing.T) {
	eng := &MockWorkflowEngine{}
	scheduler := NewWorkflowScheduler(eng)

	workflow, err := scheduler.loadWorkflow("workflow-1")
	assert.Error(t, err)
	assert.Nil(t, workflow)
}

func TestLoadWorkflow_WithStorage(t *testing.T) {
	eng := &MockWorkflowEngine{}
	scheduler := NewWorkflowScheduler(eng)
	store := storage.NewMemoryStorage()
	scheduler.SetStorage(store)

	workflow := &model.Workflow{
		ID:   "workflow-1",
		Name: "Test Workflow",
	}
	store.SaveWorkflow(workflow)

	loaded, err := scheduler.loadWorkflow("workflow-1")
	require.NoError(t, err)
	assert.Equal(t, "workflow-1", loaded.ID)
}

func TestLoadWorkflow_NotFound(t *testing.T) {
	eng := &MockWorkflowEngine{}
	scheduler := NewWorkflowScheduler(eng)
	store := storage.NewMemoryStorage()
	scheduler.SetStorage(store)

	_, err := scheduler.loadWorkflow("nonexistent")
	assert.Error(t, err)
}

// Test ScheduleConfig Fields

func TestScheduleConfig_MaxRuns(t *testing.T) {
	eng := &MockWorkflowEngine{}
	scheduler := NewWorkflowScheduler(eng)

	config := &ScheduleConfig{
		ID:         "schedule-1",
		WorkflowID: "workflow-1",
		CronExpr:   "*/5 * * * *",
		Enabled:    true,
		MaxRuns:    10,
	}

	scheduler.AddSchedule(config)

	retrieved, _ := scheduler.GetSchedule("schedule-1")
	assert.Equal(t, 10, retrieved.MaxRuns)
}

func TestScheduleConfig_MaxDuration(t *testing.T) {
	eng := &MockWorkflowEngine{}
	scheduler := NewWorkflowScheduler(eng)

	config := &ScheduleConfig{
		ID:          "schedule-1",
		WorkflowID:  "workflow-1",
		CronExpr:    "*/5 * * * *",
		Enabled:     true,
		MaxDuration: 5 * time.Minute,
	}

	scheduler.AddSchedule(config)

	retrieved, _ := scheduler.GetSchedule("schedule-1")
	assert.Equal(t, 5*time.Minute, retrieved.MaxDuration)
}

func TestScheduleConfig_Timezone(t *testing.T) {
	eng := &MockWorkflowEngine{}
	scheduler := NewWorkflowScheduler(eng)

	config := &ScheduleConfig{
		ID:         "schedule-1",
		WorkflowID: "workflow-1",
		CronExpr:   "*/5 * * * *",
		Timezone:   "America/New_York",
		Enabled:    true,
	}

	scheduler.AddSchedule(config)

	retrieved, _ := scheduler.GetSchedule("schedule-1")
	assert.Equal(t, "America/New_York", retrieved.Timezone)
}

func TestScheduleConfig_InputData(t *testing.T) {
	eng := &MockWorkflowEngine{}
	scheduler := NewWorkflowScheduler(eng)

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"key": "value"}},
	}

	config := &ScheduleConfig{
		ID:         "schedule-1",
		WorkflowID: "workflow-1",
		CronExpr:   "*/5 * * * *",
		Enabled:    true,
		InputData:  inputData,
	}

	scheduler.AddSchedule(config)

	retrieved, _ := scheduler.GetSchedule("schedule-1")
	assert.Len(t, retrieved.InputData, 1)
}

// Test ExecutionRecord

func TestExecutionRecord_Fields(t *testing.T) {
	record := ExecutionRecord{
		ID:         "exec-1",
		ScheduleID: "schedule-1",
		WorkflowID: "workflow-1",
		StartTime:  time.Now(),
		Status:     "running",
	}

	assert.Equal(t, "exec-1", record.ID)
	assert.Equal(t, "schedule-1", record.ScheduleID)
	assert.Equal(t, "running", record.Status)
}

// Test ExecutionMetrics

func TestExecutionMetrics_Fields(t *testing.T) {
	metrics := ExecutionMetrics{
		NodesExecuted: 5,
		DataProcessed: 100,
		MemoryUsed:    1024,
		CPUTime:       time.Second,
	}

	assert.Equal(t, 5, metrics.NodesExecuted)
	assert.Equal(t, 100, metrics.DataProcessed)
	assert.Equal(t, int64(1024), metrics.MemoryUsed)
}

// Test SchedulerMetrics

func TestSchedulerMetrics_Fields(t *testing.T) {
	metrics := SchedulerMetrics{
		TotalSchedules:  10,
		ActiveSchedules: 5,
		TotalExecutions: 100,
		SuccessfulExecs: 95,
		FailedExecs:     5,
	}

	assert.Equal(t, 10, metrics.TotalSchedules)
	assert.Equal(t, 5, metrics.ActiveSchedules)
	assert.Equal(t, int64(100), metrics.TotalExecutions)
}
