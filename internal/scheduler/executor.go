package scheduler

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/neul-labs/m9m/internal/engine"
	"github.com/neul-labs/m9m/internal/model"
)

// executeScheduledWorkflow executes a scheduled workflow
func (s *WorkflowScheduler) executeScheduledWorkflow(scheduleID string) {
	s.mutex.RLock()
	config, exists := s.schedules[scheduleID]
	if !exists {
		s.mutex.RUnlock()
		log.Printf("Schedule not found during execution: %s", scheduleID)
		return
	}

	if config.MaxRuns > 0 && config.RunCount >= config.MaxRuns {
		s.mutex.RUnlock()
		log.Printf("Schedule %s has reached maximum runs (%d)", scheduleID, config.MaxRuns)
		return
	}
	s.mutex.RUnlock()

	execution := ExecutionRecord{
		ID:         generateExecutionID(),
		ScheduleID: scheduleID,
		WorkflowID: config.WorkflowID,
		StartTime:  time.Now(),
		Status:     "running",
	}

	s.addExecutionRecord(scheduleID, execution)

	log.Printf("Starting scheduled execution: %s for workflow: %s", execution.ID, config.WorkflowID)

	ctx := s.ctx
	if config.MaxDuration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(s.ctx, config.MaxDuration)
		defer cancel()
	}

	workflow, loadErr := s.loadWorkflow(config.WorkflowID)
	if loadErr != nil {
		s.completeExecution(scheduleID, execution.ID, "failed", loadErr.Error(), nil)
		log.Printf("Failed to load workflow %s: %v", config.WorkflowID, loadErr)
		return
	}

	inputData := config.InputData
	if inputData == nil {
		inputData = []model.DataItem{{JSON: make(map[string]interface{})}}
	}

	startTime := time.Now()
	result, execErr := engine.ExecuteWorkflowWithContext(ctx, s.engine, workflow, inputData)
	err := engine.ResolveExecutionError(result, execErr)
	duration := time.Since(startTime)

	if err != nil {
		status := "failed"
		if errors.Is(err, context.DeadlineExceeded) {
			status = "timeout"
		}
		s.completeExecution(scheduleID, execution.ID, status, err.Error(), nil)
		log.Printf("Scheduled execution %s failed: %v", execution.ID, err)
	} else {
		s.completeExecution(scheduleID, execution.ID, "success", "", result)
		log.Printf("Scheduled execution %s completed successfully in %v", execution.ID, duration)
	}

	s.mutex.Lock()
	if config, exists := s.schedules[scheduleID]; exists {
		config.RunCount++
		now := time.Now()
		config.LastRun = &now

		if cronJob, exists := s.cronJobs[scheduleID]; exists {
			entries := cronJob.Entries()
			if len(entries) > 0 {
				config.NextRun = &[]time.Time{entries[0].Next}[0]
			}
		}
	}
	s.mutex.Unlock()

	s.metrics.mutex.Lock()
	s.metrics.TotalExecutions++
	if err == nil {
		s.metrics.SuccessfulExecs++
	} else {
		s.metrics.FailedExecs++
	}
	now := time.Now()
	s.metrics.LastExecution = &now
	s.metrics.mutex.Unlock()
}
