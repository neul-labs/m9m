package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/neul-labs/m9m/internal/engine"
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/storage"
	"github.com/robfig/cron/v3"
)

// WorkflowScheduler manages scheduled workflow executions
type WorkflowScheduler struct {
	engine     engine.WorkflowEngine
	storage    storage.WorkflowStorage
	schedules  map[string]*ScheduleConfig
	cronJobs   map[string]*cron.Cron
	executions map[string]*ExecutionHistory
	mutex      sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	metrics    *SchedulerMetrics
}

// ScheduleConfig defines a scheduled workflow execution
type ScheduleConfig struct {
	ID           string                 `json:"id"`
	WorkflowID   string                 `json:"workflowId"`
	WorkflowName string                 `json:"workflowName"`
	CronExpr     string                 `json:"cronExpression"`
	Timezone     string                 `json:"timezone"`
	Enabled      bool                   `json:"enabled"`
	LastRun      *time.Time             `json:"lastRun,omitempty"`
	NextRun      *time.Time             `json:"nextRun,omitempty"`
	MaxRuns      int                    `json:"maxRuns"`
	RunCount     int                    `json:"runCount"`
	MaxDuration  time.Duration          `json:"maxDuration"`
	InputData    []model.DataItem       `json:"inputData,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
	CreatedBy    string                 `json:"createdBy,omitempty"`
}

// ExecutionHistory tracks execution history for a schedule
type ExecutionHistory struct {
	ScheduleID   string            `json:"scheduleId"`
	Executions   []ExecutionRecord `json:"executions"`
	SuccessCount int               `json:"successCount"`
	FailureCount int               `json:"failureCount"`
	LastSuccess  *time.Time        `json:"lastSuccess,omitempty"`
	LastFailure  *time.Time        `json:"lastFailure,omitempty"`
	AverageTime  time.Duration     `json:"averageTime"`
	mutex        sync.RWMutex
}

// ExecutionRecord represents a single scheduled execution
type ExecutionRecord struct {
	ID         string            `json:"id"`
	ScheduleID string            `json:"scheduleId"`
	WorkflowID string            `json:"workflowId"`
	StartTime  time.Time         `json:"startTime"`
	EndTime    *time.Time        `json:"endTime,omitempty"`
	Duration   time.Duration     `json:"duration"`
	Status     string            `json:"status"`
	Error      string            `json:"error,omitempty"`
	ResultData interface{}       `json:"resultData,omitempty"`
	Metrics    *ExecutionMetrics `json:"metrics,omitempty"`
}

// ExecutionMetrics contains execution performance metrics
type ExecutionMetrics struct {
	NodesExecuted int           `json:"nodesExecuted"`
	DataProcessed int           `json:"dataProcessed"`
	MemoryUsed    int64         `json:"memoryUsed"`
	CPUTime       time.Duration `json:"cpuTime"`
}

// SchedulerMetrics tracks overall scheduler performance
type SchedulerMetrics struct {
	TotalSchedules  int           `json:"totalSchedules"`
	ActiveSchedules int           `json:"activeSchedules"`
	TotalExecutions int64         `json:"totalExecutions"`
	SuccessfulExecs int64         `json:"successfulExecutions"`
	FailedExecs     int64         `json:"failedExecutions"`
	AverageExecTime time.Duration `json:"averageExecutionTime"`
	LastExecution   *time.Time    `json:"lastExecution,omitempty"`
	mutex           sync.RWMutex
}

// SchedulerMetricsSnapshot is a copy of SchedulerMetrics without the mutex (safe to return by value)
type SchedulerMetricsSnapshot struct {
	TotalSchedules  int           `json:"totalSchedules"`
	ActiveSchedules int           `json:"activeSchedules"`
	TotalExecutions int64         `json:"totalExecutions"`
	SuccessfulExecs int64         `json:"successfulExecutions"`
	FailedExecs     int64         `json:"failedExecutions"`
	AverageExecTime time.Duration `json:"averageExecutionTime"`
	LastExecution   *time.Time    `json:"lastExecution,omitempty"`
}
