package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/dipankar/m9m/internal/engine"
	"github.com/dipankar/m9m/internal/model"
	"github.com/dipankar/m9m/internal/storage"
	"github.com/robfig/cron/v3"
)

// WorkflowScheduler manages scheduled workflow executions
type WorkflowScheduler struct {
	engine       engine.WorkflowEngine
	storage      storage.WorkflowStorage
	schedules    map[string]*ScheduleConfig
	cronJobs     map[string]*cron.Cron
	executions   map[string]*ExecutionHistory
	mutex        sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	metrics      *SchedulerMetrics
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
	MaxRuns      int                    `json:"maxRuns"`       // 0 = unlimited
	RunCount     int                    `json:"runCount"`
	MaxDuration  time.Duration          `json:"maxDuration"`   // Maximum execution time
	InputData    []model.DataItem       `json:"inputData,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
	CreatedBy    string                 `json:"createdBy,omitempty"`
}

// ExecutionHistory tracks execution history for a schedule
type ExecutionHistory struct {
	ScheduleID    string              `json:"scheduleId"`
	Executions    []ExecutionRecord   `json:"executions"`
	SuccessCount  int                 `json:"successCount"`
	FailureCount  int                 `json:"failureCount"`
	LastSuccess   *time.Time          `json:"lastSuccess,omitempty"`
	LastFailure   *time.Time          `json:"lastFailure,omitempty"`
	AverageTime   time.Duration       `json:"averageTime"`
	mutex         sync.RWMutex
}

// ExecutionRecord represents a single scheduled execution
type ExecutionRecord struct {
	ID          string        `json:"id"`
	ScheduleID  string        `json:"scheduleId"`
	WorkflowID  string        `json:"workflowId"`
	StartTime   time.Time     `json:"startTime"`
	EndTime     *time.Time    `json:"endTime,omitempty"`
	Duration    time.Duration `json:"duration"`
	Status      string        `json:"status"` // pending, running, success, failed, timeout
	Error       string        `json:"error,omitempty"`
	ResultData  interface{}   `json:"resultData,omitempty"`
	Metrics     *ExecutionMetrics `json:"metrics,omitempty"`
}

// ExecutionMetrics contains execution performance metrics
type ExecutionMetrics struct {
	NodesExecuted    int           `json:"nodesExecuted"`
	DataProcessed    int           `json:"dataProcessed"`
	MemoryUsed      int64         `json:"memoryUsed"`
	CPUTime         time.Duration `json:"cpuTime"`
}

// SchedulerMetrics tracks overall scheduler performance
type SchedulerMetrics struct {
	TotalSchedules    int           `json:"totalSchedules"`
	ActiveSchedules   int           `json:"activeSchedules"`
	TotalExecutions   int64         `json:"totalExecutions"`
	SuccessfulExecs   int64         `json:"successfulExecutions"`
	FailedExecs       int64         `json:"failedExecutions"`
	AverageExecTime   time.Duration `json:"averageExecutionTime"`
	LastExecution     *time.Time    `json:"lastExecution,omitempty"`
	mutex             sync.RWMutex
}

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

	// Start metrics collection
	go s.metricsCollector()

	// Start cleanup routine
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

	// Stop all cron jobs
	for _, cronJob := range s.cronJobs {
		cronJob.Stop()
	}

	log.Println("Workflow scheduler stopped")
	return nil
}

// AddSchedule adds a new scheduled workflow
func (s *WorkflowScheduler) AddSchedule(config *ScheduleConfig) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Validate cron expression
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	schedule, err := parser.Parse(config.CronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Set timezone
	location := time.UTC
	if config.Timezone != "" {
		location, err = time.LoadLocation(config.Timezone)
		if err != nil {
			return fmt.Errorf("invalid timezone: %w", err)
		}
	}

	// Generate ID if not provided
	if config.ID == "" {
		config.ID = generateScheduleID()
	}

	// Set timestamps
	now := time.Now()
	config.CreatedAt = now
	config.UpdatedAt = now
	config.NextRun = &[]time.Time{schedule.Next(now.In(location))}[0]

	// Create cron job
	cronJob := cron.New(cron.WithLocation(location), cron.WithChain(
		cron.Recover(cron.DefaultLogger),
		cron.DelayIfStillRunning(cron.DefaultLogger),
	))

	// Add job to cron
	_, err = cronJob.AddFunc(config.CronExpr, func() {
		s.executeScheduledWorkflow(config.ID)
	})
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	// Start cron job if enabled
	if config.Enabled {
		cronJob.Start()
	}

	// Store schedule and cron job
	s.schedules[config.ID] = config
	s.cronJobs[config.ID] = cronJob
	s.executions[config.ID] = &ExecutionHistory{
		ScheduleID: config.ID,
		Executions: make([]ExecutionRecord, 0),
	}

	// Update metrics
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

	// Stop and remove cron job
	if cronJob, exists := s.cronJobs[scheduleID]; exists {
		cronJob.Stop()
		delete(s.cronJobs, scheduleID)
	}

	// Remove schedule and execution history
	delete(s.schedules, scheduleID)
	delete(s.executions, scheduleID)

	// Update metrics
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

	// Store original enabled state
	wasEnabled := config.Enabled

	// Update fields
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

	// Recreate cron job if expression or timezone changed
	if updates.CronExpr != "" || updates.Timezone != "" {
		// Stop old cron job
		if cronJob, exists := s.cronJobs[scheduleID]; exists {
			cronJob.Stop()
		}

		// Create new cron job
		location := time.UTC
		if config.Timezone != "" {
			var err error
			location, err = time.LoadLocation(config.Timezone)
			if err != nil {
				return fmt.Errorf("invalid timezone: %w", err)
			}
		}

		cronJob := cron.New(cron.WithLocation(location), cron.WithChain(
			cron.Recover(cron.DefaultLogger),
			cron.DelayIfStillRunning(cron.DefaultLogger),
		))

		_, err := cronJob.AddFunc(config.CronExpr, func() {
			s.executeScheduledWorkflow(scheduleID)
		})
		if err != nil {
			return fmt.Errorf("failed to update cron job: %w", err)
		}

		s.cronJobs[scheduleID] = cronJob

		// Update next run time
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
		schedule, err := parser.Parse(config.CronExpr)
		if err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
		config.NextRun = &[]time.Time{schedule.Next(time.Now().In(location))}[0]
	}

	// Handle enabled state changes
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

	// Return a copy to prevent external modification
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

	// Create copy with limited executions
	historyCopy := &ExecutionHistory{
		ScheduleID:   history.ScheduleID,
		SuccessCount: history.SuccessCount,
		FailureCount: history.FailureCount,
		LastSuccess:  history.LastSuccess,
		LastFailure:  history.LastFailure,
		AverageTime:  history.AverageTime,
	}

	// Apply limit
	if limit > 0 && limit < len(history.Executions) {
		historyCopy.Executions = make([]ExecutionRecord, limit)
		copy(historyCopy.Executions, history.Executions[:limit])
	} else {
		historyCopy.Executions = make([]ExecutionRecord, len(history.Executions))
		copy(historyCopy.Executions, history.Executions)
	}

	return historyCopy, nil
}

// GetMetrics returns scheduler metrics
func (s *WorkflowScheduler) GetMetrics() *SchedulerMetrics {
	s.metrics.mutex.RLock()
	defer s.metrics.mutex.RUnlock()

	metricsCopy := *s.metrics
	return &metricsCopy
}

// executeScheduledWorkflow executes a scheduled workflow
func (s *WorkflowScheduler) executeScheduledWorkflow(scheduleID string) {
	s.mutex.RLock()
	config, exists := s.schedules[scheduleID]
	if !exists {
		s.mutex.RUnlock()
		log.Printf("Schedule not found during execution: %s", scheduleID)
		return
	}

	// Check if we've reached max runs
	if config.MaxRuns > 0 && config.RunCount >= config.MaxRuns {
		s.mutex.RUnlock()
		log.Printf("Schedule %s has reached maximum runs (%d)", scheduleID, config.MaxRuns)
		return
	}
	s.mutex.RUnlock()

	// Create execution record
	execution := ExecutionRecord{
		ID:         generateExecutionID(),
		ScheduleID: scheduleID,
		WorkflowID: config.WorkflowID,
		StartTime:  time.Now(),
		Status:     "running",
	}

	// Add to execution history
	s.addExecutionRecord(scheduleID, execution)

	log.Printf("Starting scheduled execution: %s for workflow: %s", execution.ID, config.WorkflowID)

	// Execute workflow with timeout
	ctx := s.ctx
	if config.MaxDuration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(s.ctx, config.MaxDuration)
		defer cancel()
	}

	// Load workflow
	workflow, err := s.loadWorkflow(config.WorkflowID)
	if err != nil {
		s.completeExecution(scheduleID, execution.ID, "failed", err.Error(), nil)
		log.Printf("Failed to load workflow %s: %v", config.WorkflowID, err)
		return
	}

	// Prepare input data
	inputData := config.InputData
	if inputData == nil {
		inputData = []model.DataItem{{JSON: make(map[string]interface{})}}
	}

	// Execute workflow
	startTime := time.Now()
	// Note: Using ExecuteWorkflow as ExecuteWorkflowWithContext is not available
	// TODO: Add context support to WorkflowEngine interface and use ctx for timeout
	_ = ctx // Silence unused warning until context support is added
	result, err := s.engine.ExecuteWorkflow(workflow, inputData)
	duration := time.Since(startTime)

	if err != nil {
		s.completeExecution(scheduleID, execution.ID, "failed", err.Error(), nil)
		log.Printf("Scheduled execution %s failed: %v", execution.ID, err)
	} else {
		s.completeExecution(scheduleID, execution.ID, "success", "", result)
		log.Printf("Scheduled execution %s completed successfully in %v", execution.ID, duration)
	}

	// Update schedule run count and last run time
	s.mutex.Lock()
	if config, exists := s.schedules[scheduleID]; exists {
		config.RunCount++
		now := time.Now()
		config.LastRun = &now

		// Calculate next run time
		if cronJob, exists := s.cronJobs[scheduleID]; exists {
			entries := cronJob.Entries()
			if len(entries) > 0 {
				config.NextRun = &[]time.Time{entries[0].Next}[0]
			}
		}
	}
	s.mutex.Unlock()

	// Update metrics
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

	// Add to beginning of slice (most recent first)
	history.Executions = append([]ExecutionRecord{execution}, history.Executions...)

	// Keep only last 100 executions
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

	// Find and update execution record
	for i := range history.Executions {
		if history.Executions[i].ID == executionID {
			now := time.Now()
			history.Executions[i].EndTime = &now
			history.Executions[i].Duration = now.Sub(history.Executions[i].StartTime)
			history.Executions[i].Status = status
			history.Executions[i].Error = errorMsg
			history.Executions[i].ResultData = result

			// Update counters
			if status == "success" {
				history.SuccessCount++
				history.LastSuccess = &now
			} else {
				history.FailureCount++
				history.LastFailure = &now
			}

			// Recalculate average execution time
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

// loadWorkflow loads a workflow by ID from storage
func (s *WorkflowScheduler) loadWorkflow(workflowID string) (*model.Workflow, error) {
	if s.storage == nil {
		return nil, fmt.Errorf("storage not configured - call SetStorage first")
	}
	return s.storage.GetWorkflow(workflowID)
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
	cutoff := time.Now().AddDate(0, 0, -30) // Keep 30 days of history

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, history := range s.executions {
		history.mutex.Lock()

		// Remove executions older than cutoff
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

// generateScheduleID generates a unique schedule ID
func generateScheduleID() string {
	return fmt.Sprintf("sched_%d", time.Now().UnixNano())
}

// generateExecutionID generates a unique execution ID
func generateExecutionID() string {
	return fmt.Sprintf("exec_%d", time.Now().UnixNano())
}

// ToJSON converts schedule config to JSON
func (c *ScheduleConfig) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

// FromJSON creates schedule config from JSON
func (c *ScheduleConfig) FromJSON(data []byte) error {
	return json.Unmarshal(data, c)
}