/*
Package worker implements the worker execution engine for the hybrid architecture.

Workers pull work from the control plane, execute workflows, and push results back.
*/
package worker

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/dipankar/n8n-go/internal/engine"
	"github.com/dipankar/n8n-go/internal/queue"
	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/pull"
	"go.nanomsg.org/mangos/v3/protocol/push"
	"go.nanomsg.org/mangos/v3/protocol/req"
	_ "go.nanomsg.org/mangos/v3/transport/tcp"
)

// WorkerConfig contains worker configuration
type WorkerConfig struct {
	WorkerID        string   // Unique worker ID
	ControlPlane    []string // Control plane addresses (for failover)
	MaxConcurrent   int      // Max concurrent workflow executions
	HeartbeatInterval time.Duration // Heartbeat interval
	WorkTimeout     time.Duration // Timeout for single workflow execution
}

// Worker represents a worker node that executes workflows
type Worker struct {
	config WorkerConfig
	engine engine.WorkflowEngine

	// NNG sockets
	workPull   mangos.Socket // Pull work from control plane
	resultPush mangos.Socket // Push results to control plane
	regReq     mangos.Socket // Registration/heartbeat to control plane

	// Execution tracking
	activeExecutions map[string]*executionState
	executionsMutex  sync.RWMutex

	// Statistics
	stats      *WorkerStats
	statsMutex sync.RWMutex

	// Control
	stopChan chan struct{}
	wg       sync.WaitGroup
	startTime time.Time
}

// executionState tracks an in-progress execution
type executionState struct {
	ExecutionID string
	WorkflowID  string
	StartTime   time.Time
	Cancel      chan struct{}
}

// WorkerStats tracks worker statistics
type WorkerStats struct {
	TotalExecutions   int64
	SuccessfulExecs   int64
	FailedExecs       int64
	ActiveExecutions  int
	AverageDurationMs int64
	UptimeSeconds     int64
}

// NewWorker creates a new worker instance
func NewWorker(config WorkerConfig, eng engine.WorkflowEngine) (*Worker, error) {
	if config.MaxConcurrent <= 0 {
		config.MaxConcurrent = 10 // Default
	}
	if config.HeartbeatInterval == 0 {
		config.HeartbeatInterval = 5 * time.Second
	}
	if config.WorkTimeout == 0 {
		config.WorkTimeout = 5 * time.Minute
	}

	w := &Worker{
		config:           config,
		engine:           eng,
		activeExecutions: make(map[string]*executionState),
		stats:            &WorkerStats{},
		stopChan:         make(chan struct{}),
		startTime:        time.Now(),
	}

	return w, nil
}

// Start initializes and starts the worker
func (w *Worker) Start() error {
	log.Printf("Starting worker: %s", w.config.WorkerID)

	// Connect to control plane
	if err := w.connectToControlPlane(); err != nil {
		return fmt.Errorf("failed to connect to control plane: %w", err)
	}

	// Register with control plane
	if err := w.register(); err != nil {
		return fmt.Errorf("failed to register with control plane: %w", err)
	}

	// Start background goroutines
	w.wg.Add(3)
	go w.workPuller()
	go w.heartbeatLoop()
	go w.statsUpdater()

	log.Printf("Worker %s started successfully (max_concurrent=%d)",
		w.config.WorkerID, w.config.MaxConcurrent)
	return nil
}

// connectToControlPlane establishes connections to control plane
func (w *Worker) connectToControlPlane() error {
	// Try to connect to first available control plane node
	var lastErr error
	for _, addr := range w.config.ControlPlane {
		log.Printf("Connecting to control plane: %s", addr)

		// Create PULL socket for work
		var err error
		if w.workPull, err = pull.NewSocket(); err != nil {
			lastErr = err
			continue
		}

		workAddr := fmt.Sprintf("%s:9000", addr)
		if err := w.workPull.Dial(workAddr); err != nil {
			w.workPull.Close()
			lastErr = err
			log.Printf("Failed to connect to work queue at %s: %v", workAddr, err)
			continue
		}

		// Create PUSH socket for results
		if w.resultPush, err = push.NewSocket(); err != nil {
			w.workPull.Close()
			lastErr = err
			continue
		}

		resultAddr := fmt.Sprintf("%s:9001", addr)
		if err := w.resultPush.Dial(resultAddr); err != nil {
			w.workPull.Close()
			w.resultPush.Close()
			lastErr = err
			log.Printf("Failed to connect to result collector at %s: %v", resultAddr, err)
			continue
		}

		// Create REQ socket for registration
		if w.regReq, err = req.NewSocket(); err != nil {
			w.workPull.Close()
			w.resultPush.Close()
			lastErr = err
			continue
		}

		regAddr := fmt.Sprintf("%s:9002", addr)
		if err := w.regReq.Dial(regAddr); err != nil {
			w.workPull.Close()
			w.resultPush.Close()
			w.regReq.Close()
			lastErr = err
			log.Printf("Failed to connect to registration at %s: %v", regAddr, err)
			continue
		}

		log.Printf("Successfully connected to control plane: %s", addr)
		return nil
	}

	return fmt.Errorf("failed to connect to any control plane node: %w", lastErr)
}

// register registers this worker with the control plane
func (w *Worker) register() error {
	reg := queue.WorkerRegistration{
		Type:           "register",
		WorkerID:       w.config.WorkerID,
		Capabilities:   []string{"*"}, // Support all node types
		MaxConcurrent:  w.config.MaxConcurrent,
		ActiveExecutions: 0,
		UptimeSeconds:  0,
	}

	data, err := json.Marshal(reg)
	if err != nil {
		return fmt.Errorf("failed to marshal registration: %w", err)
	}

	if err := w.regReq.Send(data); err != nil {
		return fmt.Errorf("failed to send registration: %w", err)
	}

	// Wait for response
	respData, err := w.regReq.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive registration response: %w", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(respData, &response); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		return fmt.Errorf("registration failed: %v", response["message"])
	}

	log.Printf("Worker registered successfully: %s", w.config.WorkerID)
	return nil
}

// workPuller continuously pulls and executes work from the queue
func (w *Worker) workPuller() {
	defer w.wg.Done()

	log.Println("Work puller started")

	for {
		select {
		case <-w.stopChan:
			return
		default:
		}

		// Check if we can accept more work
		if !w.canAcceptWork() {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Pull work (blocks until work arrives)
		data, err := w.workPull.Recv()
		if err != nil {
			if err == mangos.ErrClosed {
				return
			}
			log.Printf("Error receiving work: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		// Parse work message
		var work queue.WorkMessage
		if err := json.Unmarshal(data, &work); err != nil {
			log.Printf("Error unmarshaling work: %v", err)
			continue
		}

		log.Printf("Received work: execution_id=%s, workflow_id=%s",
			work.ExecutionID, work.WorkflowID)

		// Execute in goroutine (non-blocking)
		go w.executeWork(&work)
	}
}

// canAcceptWork checks if worker can accept more work
func (w *Worker) canAcceptWork() bool {
	w.executionsMutex.RLock()
	defer w.executionsMutex.RUnlock()

	return len(w.activeExecutions) < w.config.MaxConcurrent
}

// executeWork executes a workflow and sends the result
func (w *Worker) executeWork(work *queue.WorkMessage) {
	startTime := time.Now()

	// Track execution
	execState := &executionState{
		ExecutionID: work.ExecutionID,
		WorkflowID:  work.WorkflowID,
		StartTime:   startTime,
		Cancel:      make(chan struct{}),
	}

	w.executionsMutex.Lock()
	w.activeExecutions[work.ExecutionID] = execState
	w.executionsMutex.Unlock()

	// Remove from tracking when done
	defer func() {
		w.executionsMutex.Lock()
		delete(w.activeExecutions, work.ExecutionID)
		w.executionsMutex.Unlock()
	}()

	// Execute workflow with timeout
	result := w.executeWithTimeout(work, execState.Cancel)

	// Calculate duration
	duration := time.Since(startTime)
	result.DurationMs = duration.Milliseconds()
	result.WorkerID = w.config.WorkerID
	result.Timestamp = time.Now()

	// Update stats
	w.updateStats(result.Status, duration)

	// Send result back to control plane
	if err := w.sendResult(result); err != nil {
		log.Printf("Error sending result for execution %s: %v", work.ExecutionID, err)
	}
}

// executeWithTimeout executes a workflow with timeout
func (w *Worker) executeWithTimeout(work *queue.WorkMessage, cancel chan struct{}) *queue.ResultMessage {
	resultChan := make(chan *queue.ResultMessage, 1)

	// Execute in goroutine
	go func() {
		result := w.executeWorkflow(work)
		resultChan <- result
	}()

	// Wait for result or timeout
	select {
	case result := <-resultChan:
		return result
	case <-time.After(w.config.WorkTimeout):
		log.Printf("Workflow execution timeout: execution_id=%s", work.ExecutionID)
		return &queue.ResultMessage{
			Type:        "execution_result",
			ExecutionID: work.ExecutionID,
			Status:      "timeout",
			Error:       fmt.Sprintf("execution timeout after %v", w.config.WorkTimeout),
		}
	case <-cancel:
		log.Printf("Workflow execution cancelled: execution_id=%s", work.ExecutionID)
		return &queue.ResultMessage{
			Type:        "execution_result",
			ExecutionID: work.ExecutionID,
			Status:      "cancelled",
			Error:       "execution cancelled",
		}
	}
}

// executeWorkflow executes a workflow and returns the result
func (w *Worker) executeWorkflow(work *queue.WorkMessage) *queue.ResultMessage {
	log.Printf("Executing workflow: execution_id=%s, workflow_id=%s",
		work.ExecutionID, work.WorkflowID)

	// Execute workflow using the engine
	resultData, err := w.engine.ExecuteWorkflow(work.Workflow, work.InputData)

	if err != nil {
		log.Printf("Workflow execution failed: execution_id=%s, error=%v",
			work.ExecutionID, err)
		return &queue.ResultMessage{
			Type:        "execution_result",
			ExecutionID: work.ExecutionID,
			Status:      "failed",
			Error:       err.Error(),
		}
	}

	log.Printf("Workflow execution succeeded: execution_id=%s", work.ExecutionID)
	return &queue.ResultMessage{
		Type:        "execution_result",
		ExecutionID: work.ExecutionID,
		Status:      "success",
		ResultData:  resultData.Data,
	}
}

// sendResult sends a result back to the control plane
func (w *Worker) sendResult(result *queue.ResultMessage) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	if err := w.resultPush.Send(data); err != nil {
		return fmt.Errorf("failed to send result: %w", err)
	}

	log.Printf("Sent result: execution_id=%s, status=%s", result.ExecutionID, result.Status)
	return nil
}

// heartbeatLoop sends periodic heartbeats to control plane
func (w *Worker) heartbeatLoop() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			return
		case <-ticker.C:
			if err := w.sendHeartbeat(); err != nil {
				log.Printf("Error sending heartbeat: %v", err)
			}
		}
	}
}

// sendHeartbeat sends a heartbeat to the control plane
func (w *Worker) sendHeartbeat() error {
	w.executionsMutex.RLock()
	activeCount := len(w.activeExecutions)
	w.executionsMutex.RUnlock()

	uptime := time.Since(w.startTime).Seconds()

	heartbeat := queue.WorkerRegistration{
		Type:             "heartbeat",
		WorkerID:         w.config.WorkerID,
		MaxConcurrent:    w.config.MaxConcurrent,
		ActiveExecutions: activeCount,
		UptimeSeconds:    int64(uptime),
	}

	data, err := json.Marshal(heartbeat)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat: %w", err)
	}

	if err := w.regReq.Send(data); err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}

	// Wait for acknowledgment
	if _, err := w.regReq.Recv(); err != nil {
		return fmt.Errorf("failed to receive heartbeat ack: %w", err)
	}

	return nil
}

// updateStats updates worker statistics
func (w *Worker) updateStats(status string, duration time.Duration) {
	w.statsMutex.Lock()
	defer w.statsMutex.Unlock()

	w.stats.TotalExecutions++
	if status == "success" {
		w.stats.SuccessfulExecs++
	} else {
		w.stats.FailedExecs++
	}

	// Update average duration (simple moving average)
	totalMs := w.stats.AverageDurationMs * (w.stats.TotalExecutions - 1)
	totalMs += duration.Milliseconds()
	w.stats.AverageDurationMs = totalMs / w.stats.TotalExecutions
}

// statsUpdater periodically updates stats
func (w *Worker) statsUpdater() {
	defer w.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			return
		case <-ticker.C:
			w.updateRuntimeStats()
		}
	}
}

// updateRuntimeStats updates runtime statistics
func (w *Worker) updateRuntimeStats() {
	w.executionsMutex.RLock()
	activeCount := len(w.activeExecutions)
	w.executionsMutex.RUnlock()

	w.statsMutex.Lock()
	w.stats.ActiveExecutions = activeCount
	w.stats.UptimeSeconds = int64(time.Since(w.startTime).Seconds())
	w.statsMutex.Unlock()
}

// GetStats returns current worker statistics
func (w *Worker) GetStats() WorkerStats {
	w.statsMutex.RLock()
	defer w.statsMutex.RUnlock()

	return *w.stats
}

// Stop gracefully shuts down the worker
func (w *Worker) Stop() error {
	log.Printf("Stopping worker: %s", w.config.WorkerID)

	close(w.stopChan)

	// Wait for active executions to complete (with timeout)
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		w.executionsMutex.RLock()
		activeCount := len(w.activeExecutions)
		w.executionsMutex.RUnlock()

		if activeCount == 0 {
			break
		}

		select {
		case <-timeout:
			log.Printf("Timeout waiting for active executions to complete (%d remaining)", activeCount)
			goto cleanup
		case <-ticker.C:
			log.Printf("Waiting for %d active executions to complete...", activeCount)
		}
	}

cleanup:
	// Wait for goroutines
	w.wg.Wait()

	// Close sockets
	if w.workPull != nil {
		w.workPull.Close()
	}
	if w.resultPush != nil {
		w.resultPush.Close()
	}
	if w.regReq != nil {
		w.regReq.Close()
	}

	log.Printf("Worker %s stopped", w.config.WorkerID)
	return nil
}
