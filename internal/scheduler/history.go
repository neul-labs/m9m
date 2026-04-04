package scheduler

import (
	"fmt"
	"time"
)

// GetExecutionHistory returns execution history for a schedule
func (s *WorkflowScheduler) GetExecutionHistory(scheduleID string, limit int) (*ExecutionHistory, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	history, exists := s.executions[scheduleID]
	if !exists {
		return nil, fmt.Errorf("execution history not found for schedule: %s", scheduleID)
	}

	history.mutex.RLock()
	defer history.mutex.RUnlock()

	historyCopy := &ExecutionHistory{
		ScheduleID:   history.ScheduleID,
		SuccessCount: history.SuccessCount,
		FailureCount: history.FailureCount,
		LastSuccess:  history.LastSuccess,
		LastFailure:  history.LastFailure,
		AverageTime:  history.AverageTime,
	}

	if limit > 0 && limit < len(history.Executions) {
		historyCopy.Executions = make([]ExecutionRecord, limit)
		copy(historyCopy.Executions, history.Executions[:limit])
	} else {
		historyCopy.Executions = make([]ExecutionRecord, len(history.Executions))
		copy(historyCopy.Executions, history.Executions)
	}

	return historyCopy, nil
}

// addExecutionRecord adds an execution record to history
func (s *WorkflowScheduler) addExecutionRecord(scheduleID string, execution ExecutionRecord) {
	s.mutex.RLock()
	history, exists := s.executions[scheduleID]
	s.mutex.RUnlock()

	if !exists {
		return
	}

	history.mutex.Lock()
	defer history.mutex.Unlock()

	history.Executions = append([]ExecutionRecord{execution}, history.Executions...)
	if len(history.Executions) > 100 {
		history.Executions = history.Executions[:100]
	}
}

// completeExecution marks an execution as complete
func (s *WorkflowScheduler) completeExecution(scheduleID, executionID, status, errorMsg string, result interface{}) {
	s.mutex.RLock()
	history, exists := s.executions[scheduleID]
	s.mutex.RUnlock()

	if !exists {
		return
	}

	history.mutex.Lock()
	defer history.mutex.Unlock()

	for i := range history.Executions {
		if history.Executions[i].ID == executionID {
			now := time.Now()
			history.Executions[i].EndTime = &now
			history.Executions[i].Duration = now.Sub(history.Executions[i].StartTime)
			history.Executions[i].Status = status
			history.Executions[i].Error = errorMsg
			history.Executions[i].ResultData = result

			if status == "success" {
				history.SuccessCount++
				history.LastSuccess = &now
			} else {
				history.FailureCount++
				history.LastFailure = &now
			}

			s.recalculateAverageTime(history)
			break
		}
	}
}

// recalculateAverageTime recalculates the average execution time
func (s *WorkflowScheduler) recalculateAverageTime(history *ExecutionHistory) {
	var totalDuration time.Duration
	var completedCount int

	for _, exec := range history.Executions {
		if exec.EndTime != nil {
			totalDuration += exec.Duration
			completedCount++
		}
	}

	if completedCount > 0 {
		history.AverageTime = totalDuration / time.Duration(completedCount)
	}
}
