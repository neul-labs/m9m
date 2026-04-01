package worker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWorkerConfig_Defaults(t *testing.T) {
	config := WorkerConfig{}
	assert.Empty(t, config.WorkerID)
	assert.Nil(t, config.ControlPlane)
	assert.Equal(t, 0, config.MaxConcurrent)
	assert.Equal(t, time.Duration(0), config.HeartbeatInterval)
	assert.Equal(t, time.Duration(0), config.WorkTimeout)
}

func TestWorkerConfig_CustomValues(t *testing.T) {
	config := WorkerConfig{
		WorkerID:          "worker-1",
		ControlPlane:      []string{"tcp://localhost:5555"},
		MaxConcurrent:     5,
		HeartbeatInterval: 10 * time.Second,
		WorkTimeout:       60 * time.Second,
	}

	assert.Equal(t, "worker-1", config.WorkerID)
	assert.Len(t, config.ControlPlane, 1)
	assert.Equal(t, 5, config.MaxConcurrent)
	assert.Equal(t, 10*time.Second, config.HeartbeatInterval)
	assert.Equal(t, 60*time.Second, config.WorkTimeout)
}

func TestWorkerStats_ZeroValues(t *testing.T) {
	stats := &WorkerStats{}
	assert.Equal(t, int64(0), stats.TotalExecutions)
	assert.Equal(t, int64(0), stats.SuccessfulExecs)
	assert.Equal(t, int64(0), stats.FailedExecs)
	assert.Equal(t, 0, stats.ActiveExecutions)
	assert.Equal(t, int64(0), stats.AverageDurationMs)
	assert.Equal(t, int64(0), stats.UptimeSeconds)
}

func TestExecutionState_Struct(t *testing.T) {
	cancel := make(chan struct{})
	state := &executionState{
		ExecutionID: "exec-1",
		WorkflowID:  "wf-1",
		StartTime:   time.Now(),
		Cancel:      cancel,
	}

	assert.Equal(t, "exec-1", state.ExecutionID)
	assert.Equal(t, "wf-1", state.WorkflowID)
	assert.False(t, state.StartTime.IsZero())
	assert.NotNil(t, state.Cancel)
}
