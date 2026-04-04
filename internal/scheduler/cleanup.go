package scheduler

import (
	"log"
	"time"
)

// cleanupRoutine performs periodic cleanup of old execution records
func (s *WorkflowScheduler) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.cleanupOldExecutions()
		}
	}
}

// cleanupOldExecutions removes old execution records
func (s *WorkflowScheduler) cleanupOldExecutions() {
	cutoff := time.Now().AddDate(0, 0, -30)

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, history := range s.executions {
		history.mutex.Lock()

		var kept []ExecutionRecord
		for _, exec := range history.Executions {
			if exec.StartTime.After(cutoff) {
				kept = append(kept, exec)
			}
		}
		history.Executions = kept

		history.mutex.Unlock()
	}

	log.Println("Cleaned up old execution records")
}
