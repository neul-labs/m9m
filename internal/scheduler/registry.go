package scheduler

import (
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

// AddSchedule adds a new scheduled workflow
func (s *WorkflowScheduler) AddSchedule(config *ScheduleConfig) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	schedule, location, err := parseCronSchedule(config.CronExpr, config.Timezone)
	if err != nil {
		return err
	}

	if config.ID == "" {
		config.ID = generateScheduleID()
	}

	now := time.Now()
	config.CreatedAt = now
	config.UpdatedAt = now
	config.NextRun = &[]time.Time{schedule.Next(now.In(location))}[0]

	cronJob, err := s.newCronJob(config.ID, config.CronExpr, location)
	if err != nil {
		return err
	}

	if config.Enabled {
		cronJob.Start()
	}

	s.schedules[config.ID] = config
	s.cronJobs[config.ID] = cronJob
	s.executions[config.ID] = &ExecutionHistory{
		ScheduleID: config.ID,
		Executions: make([]ExecutionRecord, 0),
	}

	s.metrics.mutex.Lock()
	s.metrics.TotalSchedules++
	if config.Enabled {
		s.metrics.ActiveSchedules++
	}
	s.metrics.mutex.Unlock()

	log.Printf("Added schedule: %s for workflow: %s", config.ID, config.WorkflowID)
	return nil
}

// RemoveSchedule removes a scheduled workflow
func (s *WorkflowScheduler) RemoveSchedule(scheduleID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	config, exists := s.schedules[scheduleID]
	if !exists {
		return fmt.Errorf("schedule not found: %s", scheduleID)
	}

	if cronJob, exists := s.cronJobs[scheduleID]; exists {
		cronJob.Stop()
		delete(s.cronJobs, scheduleID)
	}

	delete(s.schedules, scheduleID)
	delete(s.executions, scheduleID)

	s.metrics.mutex.Lock()
	s.metrics.TotalSchedules--
	if config.Enabled {
		s.metrics.ActiveSchedules--
	}
	s.metrics.mutex.Unlock()

	log.Printf("Removed schedule: %s", scheduleID)
	return nil
}

// UpdateSchedule updates an existing schedule
func (s *WorkflowScheduler) UpdateSchedule(scheduleID string, updates *ScheduleConfig) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	config, exists := s.schedules[scheduleID]
	if !exists {
		return fmt.Errorf("schedule not found: %s", scheduleID)
	}

	wasEnabled := config.Enabled

	if updates.CronExpr != "" {
		config.CronExpr = updates.CronExpr
	}
	if updates.Timezone != "" {
		config.Timezone = updates.Timezone
	}
	config.Enabled = updates.Enabled
	if updates.MaxRuns > 0 {
		config.MaxRuns = updates.MaxRuns
	}
	if updates.MaxDuration > 0 {
		config.MaxDuration = updates.MaxDuration
	}
	if updates.InputData != nil {
		config.InputData = updates.InputData
	}
	if updates.Parameters != nil {
		config.Parameters = updates.Parameters
	}
	config.UpdatedAt = time.Now()

	if updates.CronExpr != "" || updates.Timezone != "" {
		location := time.UTC
		if config.Timezone != "" {
			var err error
			location, err = time.LoadLocation(config.Timezone)
			if err != nil {
				return fmt.Errorf("invalid timezone: %w", err)
			}
		}

		if cronJob, exists := s.cronJobs[scheduleID]; exists {
			cronJob.Stop()
		}

		cronJob, err := s.newCronJob(scheduleID, config.CronExpr, location)
		if err != nil {
			return err
		}
		s.cronJobs[scheduleID] = cronJob

		schedule, err := cronParser.Parse(config.CronExpr)
		if err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
		config.NextRun = &[]time.Time{schedule.Next(time.Now().In(location))}[0]
	}

	if cronJob, exists := s.cronJobs[scheduleID]; exists {
		if config.Enabled && !wasEnabled {
			cronJob.Start()
			s.metrics.mutex.Lock()
			s.metrics.ActiveSchedules++
			s.metrics.mutex.Unlock()
		} else if !config.Enabled && wasEnabled {
			cronJob.Stop()
			s.metrics.mutex.Lock()
			s.metrics.ActiveSchedules--
			s.metrics.mutex.Unlock()
		}
	}

	log.Printf("Updated schedule: %s", scheduleID)
	return nil
}

// GetSchedule retrieves a schedule by ID
func (s *WorkflowScheduler) GetSchedule(scheduleID string) (*ScheduleConfig, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	config, exists := s.schedules[scheduleID]
	if !exists {
		return nil, fmt.Errorf("schedule not found: %s", scheduleID)
	}

	configCopy := *config
	return &configCopy, nil
}

// ListSchedules returns all schedules
func (s *WorkflowScheduler) ListSchedules() []*ScheduleConfig {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	schedules := make([]*ScheduleConfig, 0, len(s.schedules))
	for _, config := range s.schedules {
		configCopy := *config
		schedules = append(schedules, &configCopy)
	}

	return schedules
}

func parseCronSchedule(cronExpr, timezone string) (cron.Schedule, *time.Location, error) {
	schedule, err := cronParser.Parse(cronExpr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid cron expression: %w", err)
	}

	location := time.UTC
	if timezone != "" {
		location, err = time.LoadLocation(timezone)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid timezone: %w", err)
		}
	}

	return schedule, location, nil
}

func (s *WorkflowScheduler) newCronJob(scheduleID, cronExpr string, location *time.Location) (*cron.Cron, error) {
	cronJob := cron.New(cron.WithLocation(location), cron.WithChain(
		cron.Recover(cron.DefaultLogger),
		cron.DelayIfStillRunning(cron.DefaultLogger),
	))

	_, err := cronJob.AddFunc(cronExpr, func() {
		s.executeScheduledWorkflow(scheduleID)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add cron job: %w", err)
	}

	return cronJob, nil
}
