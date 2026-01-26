/*
Package queue implements NNG-based work distribution for the hybrid architecture.

This package provides:
- Work distribution from control plane to workers (PUSH/PULL)
- Result collection from workers to control plane (PUSH/PULL)
- Worker registration and heartbeat (REQ/REP)
*/
package queue

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/neul-labs/m9m/internal/model"
	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/pull"
	"go.nanomsg.org/mangos/v3/protocol/push"
	"go.nanomsg.org/mangos/v3/protocol/rep"
	_ "go.nanomsg.org/mangos/v3/transport/tcp"
)

// WorkMessage represents a work item to be executed by a worker
type WorkMessage struct {
	Type        string                `json:"type"`         // "execute_workflow"
	ExecutionID string                `json:"execution_id"` // Unique execution ID
	WorkflowID  string                `json:"workflow_id"`  // Workflow ID
	Workflow    *model.Workflow       `json:"workflow"`     // Full workflow definition
	InputData   []model.DataItem      `json:"input_data"`   // Initial input data
	Priority    string                `json:"priority"`     // "high", "normal", "low"
	Timestamp   time.Time             `json:"timestamp"`    // When work was queued
	RetryCount  int                   `json:"retry_count"`  // Number of retries
}

// ResultMessage represents a result from a worker
type ResultMessage struct {
	Type        string           `json:"type"`         // "execution_result"
	ExecutionID string           `json:"execution_id"` // Execution ID
	Status      string           `json:"status"`       // "success", "failed", "timeout"
	ResultData  []model.DataItem `json:"result_data"`  // Output data
	Error       string           `json:"error"`        // Error message if failed
	DurationMs  int64            `json:"duration_ms"`  // Execution duration
	WorkerID    string           `json:"worker_id"`    // Which worker executed it
	Timestamp   time.Time        `json:"timestamp"`    // When result was generated
}

// WorkerRegistration represents a worker registration message
type WorkerRegistration struct {
	Type           string   `json:"type"`            // "register" or "heartbeat"
	WorkerID       string   `json:"worker_id"`       // Unique worker ID
	Capabilities   []string `json:"capabilities"`    // Supported node types
	MaxConcurrent  int      `json:"max_concurrent"`  // Max concurrent executions
	ActiveExecutions int    `json:"active_executions"` // Current active count
	UptimeSeconds  int64    `json:"uptime_seconds"`  // Worker uptime
}

// WorkerStatus represents the status of a worker
type WorkerStatus struct {
	WorkerID         string    `json:"worker_id"`
	Status           string    `json:"status"` // "active", "offline"
	LastHeartbeat    time.Time `json:"last_heartbeat"`
	ActiveExecutions int       `json:"active_executions"`
	TotalExecutions  int64     `json:"total_executions"`
	MaxConcurrent    int       `json:"max_concurrent"`
	RegisteredAt     time.Time `json:"registered_at"`
}

// WorkQueue manages work distribution via NNG
type WorkQueue struct {
	// Work distribution (control plane → workers)
	workPush mangos.Socket

	// Result collection (workers → control plane)
	resultPull mangos.Socket

	// Worker registration (workers ↔ control plane)
	workerRep mangos.Socket

	// Worker tracking
	workers      map[string]*WorkerStatus
	workersMutex sync.RWMutex

	// Handlers
	resultHandler ResultHandler

	// Control
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// ResultHandler is called when a result is received
type ResultHandler func(result *ResultMessage) error

// NewWorkQueue creates a new work queue for the control plane
func NewWorkQueue(workAddr, resultAddr, registrationAddr string) (*WorkQueue, error) {
	wq := &WorkQueue{
		workers:  make(map[string]*WorkerStatus),
		stopChan: make(chan struct{}),
	}

	// Create PUSH socket for work distribution
	var err error
	if wq.workPush, err = push.NewSocket(); err != nil {
		return nil, fmt.Errorf("failed to create push socket: %w", err)
	}

	if err := wq.workPush.Listen(workAddr); err != nil {
		return nil, fmt.Errorf("failed to listen on work address %s: %w", workAddr, err)
	}
	log.Printf("Work queue listening on %s (PUSH)", workAddr)

	// Create PULL socket for result collection
	if wq.resultPull, err = pull.NewSocket(); err != nil {
		wq.workPush.Close()
		return nil, fmt.Errorf("failed to create pull socket: %w", err)
	}

	if err := wq.resultPull.Listen(resultAddr); err != nil {
		wq.workPush.Close()
		wq.resultPull.Close()
		return nil, fmt.Errorf("failed to listen on result address %s: %w", resultAddr, err)
	}
	log.Printf("Result collector listening on %s (PULL)", resultAddr)

	// Create REP socket for worker registration
	if wq.workerRep, err = rep.NewSocket(); err != nil {
		wq.workPush.Close()
		wq.resultPull.Close()
		return nil, fmt.Errorf("failed to create rep socket: %w", err)
	}

	if err := wq.workerRep.Listen(registrationAddr); err != nil {
		wq.workPush.Close()
		wq.resultPull.Close()
		wq.workerRep.Close()
		return nil, fmt.Errorf("failed to listen on registration address %s: %w", registrationAddr, err)
	}
	log.Printf("Worker registration listening on %s (REP)", registrationAddr)

	// Start background goroutines
	wq.wg.Add(3)
	go wq.resultCollector()
	go wq.registrationHandler()
	go wq.workerMonitor()

	return wq, nil
}

// PushWork pushes a work item to the queue
func (wq *WorkQueue) PushWork(work *WorkMessage) error {
	data, err := json.Marshal(work)
	if err != nil {
		return fmt.Errorf("failed to marshal work: %w", err)
	}

	if err := wq.workPush.Send(data); err != nil {
		return fmt.Errorf("failed to send work: %w", err)
	}

	log.Printf("Pushed work to queue: execution_id=%s, workflow_id=%s", work.ExecutionID, work.WorkflowID)
	return nil
}

// SetResultHandler sets the handler for received results
func (wq *WorkQueue) SetResultHandler(handler ResultHandler) {
	wq.resultHandler = handler
}

// resultCollector continuously pulls results from workers
func (wq *WorkQueue) resultCollector() {
	defer wq.wg.Done()

	log.Println("Started result collector")

	for {
		select {
		case <-wq.stopChan:
			return
		default:
		}

		// Receive result (blocks until message arrives)
		data, err := wq.resultPull.Recv()
		if err != nil {
			if err == mangos.ErrClosed {
				return
			}
			log.Printf("Error receiving result: %v", err)
			continue
		}

		// Parse result
		var result ResultMessage
		if err := json.Unmarshal(data, &result); err != nil {
			log.Printf("Error unmarshaling result: %v", err)
			continue
		}

		log.Printf("Received result: execution_id=%s, status=%s, worker=%s",
			result.ExecutionID, result.Status, result.WorkerID)

		// Update worker stats
		wq.updateWorkerStats(result.WorkerID)

		// Call handler
		if wq.resultHandler != nil {
			if err := wq.resultHandler(&result); err != nil {
				log.Printf("Error handling result: %v", err)
			}
		}
	}
}

// registrationHandler handles worker registration and heartbeat
func (wq *WorkQueue) registrationHandler() {
	defer wq.wg.Done()

	log.Println("Started registration handler")

	for {
		select {
		case <-wq.stopChan:
			return
		default:
		}

		// Receive registration message (REQ)
		data, err := wq.workerRep.Recv()
		if err != nil {
			if err == mangos.ErrClosed {
				return
			}
			log.Printf("Error receiving registration: %v", err)
			continue
		}

		// Parse registration
		var reg WorkerRegistration
		if err := json.Unmarshal(data, &reg); err != nil {
			log.Printf("Error unmarshaling registration: %v", err)
			wq.sendRegistrationResponse(false, "Invalid message format")
			continue
		}

		// Handle registration or heartbeat
		if reg.Type == "register" {
			wq.registerWorker(&reg)
			wq.sendRegistrationResponse(true, "Registered successfully")
		} else if reg.Type == "heartbeat" {
			wq.updateHeartbeat(&reg)
			wq.sendRegistrationResponse(true, "Heartbeat acknowledged")
		} else {
			wq.sendRegistrationResponse(false, "Unknown message type")
		}
	}
}

// sendRegistrationResponse sends a response to worker registration/heartbeat
func (wq *WorkQueue) sendRegistrationResponse(success bool, message string) {
	response := map[string]interface{}{
		"success": success,
		"message": message,
	}

	data, _ := json.Marshal(response)
	if err := wq.workerRep.Send(data); err != nil {
		log.Printf("Error sending registration response: %v", err)
	}
}

// registerWorker registers a new worker
func (wq *WorkQueue) registerWorker(reg *WorkerRegistration) {
	wq.workersMutex.Lock()
	defer wq.workersMutex.Unlock()

	worker := &WorkerStatus{
		WorkerID:         reg.WorkerID,
		Status:           "active",
		LastHeartbeat:    time.Now(),
		ActiveExecutions: reg.ActiveExecutions,
		MaxConcurrent:    reg.MaxConcurrent,
		RegisteredAt:     time.Now(),
	}

	wq.workers[reg.WorkerID] = worker
	log.Printf("Worker registered: %s (max_concurrent=%d)", reg.WorkerID, reg.MaxConcurrent)
}

// updateHeartbeat updates a worker's heartbeat
func (wq *WorkQueue) updateHeartbeat(reg *WorkerRegistration) {
	wq.workersMutex.Lock()
	defer wq.workersMutex.Unlock()

	if worker, exists := wq.workers[reg.WorkerID]; exists {
		worker.LastHeartbeat = time.Now()
		worker.ActiveExecutions = reg.ActiveExecutions
		worker.Status = "active"
	}
}

// updateWorkerStats updates worker execution stats
func (wq *WorkQueue) updateWorkerStats(workerID string) {
	wq.workersMutex.Lock()
	defer wq.workersMutex.Unlock()

	if worker, exists := wq.workers[workerID]; exists {
		worker.TotalExecutions++
	}
}

// workerMonitor monitors worker health and marks offline workers
func (wq *WorkQueue) workerMonitor() {
	defer wq.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-wq.stopChan:
			return
		case <-ticker.C:
			wq.checkWorkerHealth()
		}
	}
}

// checkWorkerHealth checks for offline workers (no heartbeat for 30 seconds)
func (wq *WorkQueue) checkWorkerHealth() {
	wq.workersMutex.Lock()
	defer wq.workersMutex.Unlock()

	timeout := 30 * time.Second
	now := time.Now()

	for workerID, worker := range wq.workers {
		if now.Sub(worker.LastHeartbeat) > timeout {
			if worker.Status != "offline" {
				log.Printf("Worker marked offline: %s (last heartbeat: %v ago)",
					workerID, now.Sub(worker.LastHeartbeat))
				worker.Status = "offline"
			}
		}
	}
}

// GetWorkers returns the current worker pool status
func (wq *WorkQueue) GetWorkers() []*WorkerStatus {
	wq.workersMutex.RLock()
	defer wq.workersMutex.RUnlock()

	workers := make([]*WorkerStatus, 0, len(wq.workers))
	for _, worker := range wq.workers {
		workerCopy := *worker
		workers = append(workers, &workerCopy)
	}

	return workers
}

// GetWorkerCount returns the number of active workers
func (wq *WorkQueue) GetWorkerCount() (active int, total int) {
	wq.workersMutex.RLock()
	defer wq.workersMutex.RUnlock()

	total = len(wq.workers)
	for _, worker := range wq.workers {
		if worker.Status == "active" {
			active++
		}
	}

	return active, total
}

// Close gracefully shuts down the work queue
func (wq *WorkQueue) Close() error {
	close(wq.stopChan)
	wq.wg.Wait()

	if err := wq.workPush.Close(); err != nil {
		return err
	}
	if err := wq.resultPull.Close(); err != nil {
		return err
	}
	if err := wq.workerRep.Close(); err != nil {
		return err
	}

	log.Println("Work queue shut down successfully")
	return nil
}
