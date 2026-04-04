package scheduler

import "time"

// GetMetrics returns scheduler metrics as a snapshot (safe to use without mutex concerns)
func (s *WorkflowScheduler) GetMetrics() SchedulerMetricsSnapshot {
	s.metrics.mutex.RLock()
	defer s.metrics.mutex.RUnlock()

	var lastExec *time.Time
	if s.metrics.LastExecution != nil {
		t := *s.metrics.LastExecution
		lastExec = &t
	}

	return SchedulerMetricsSnapshot{
		TotalSchedules:  s.metrics.TotalSchedules,
		ActiveSchedules: s.metrics.ActiveSchedules,
		TotalExecutions: s.metrics.TotalExecutions,
		SuccessfulExecs: s.metrics.SuccessfulExecs,
		FailedExecs:     s.metrics.FailedExecs,
		AverageExecTime: s.metrics.AverageExecTime,
		LastExecution:   lastExec,
	}
}

// metricsCollector runs periodic metrics collection
func (s *WorkflowScheduler) metricsCollector() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.collectMetrics()
		}
	}
}

// collectMetrics updates scheduler metrics
func (s *WorkflowScheduler) collectMetrics() {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var totalDuration time.Duration
	var completedExecutions int64

	for _, history := range s.executions {
		history.mutex.RLock()
		for _, exec := range history.Executions {
			if exec.EndTime != nil {
				totalDuration += exec.Duration
				completedExecutions++
			}
		}
		history.mutex.RUnlock()
	}

	s.metrics.mutex.Lock()
	if completedExecutions > 0 {
		s.metrics.AverageExecTime = totalDuration / time.Duration(completedExecutions)
	}
	s.metrics.mutex.Unlock()
}
