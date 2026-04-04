package scheduler

import (
	"context"
	"fmt"
	"log"

	"github.com/neul-labs/m9m/internal/engine"
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/storage"
	"github.com/robfig/cron/v3"
)

// NewWorkflowScheduler creates a new workflow scheduler
func NewWorkflowScheduler(eng engine.WorkflowEngine) *WorkflowScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkflowScheduler{
		engine:     eng,
		schedules:  make(map[string]*ScheduleConfig),
		cronJobs:   make(map[string]*cron.Cron),
		executions: make(map[string]*ExecutionHistory),
		ctx:        ctx,
		cancel:     cancel,
		metrics:    &SchedulerMetrics{},
	}
}

// SetStorage sets the workflow storage for loading workflows
func (s *WorkflowScheduler) SetStorage(store storage.WorkflowStorage) {
	s.storage = store
}

// Start initializes and starts the scheduler
func (s *WorkflowScheduler) Start() error {
	log.Println("Starting workflow scheduler...")

	go s.metricsCollector()
	go s.cleanupRoutine()

	log.Println("Workflow scheduler started successfully")
	return nil
}

// Stop gracefully shuts down the scheduler
func (s *WorkflowScheduler) Stop() error {
	log.Println("Stopping workflow scheduler...")

	s.cancel()

	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, cronJob := range s.cronJobs {
		cronJob.Stop()
	}

	log.Println("Workflow scheduler stopped")
	return nil
}

// loadWorkflow loads a workflow by ID from storage
func (s *WorkflowScheduler) loadWorkflow(workflowID string) (*model.Workflow, error) {
	if s.storage == nil {
		return nil, fmt.Errorf("storage not configured - call SetStorage first")
	}
	return s.storage.GetWorkflow(workflowID)
}
