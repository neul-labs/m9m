/*
Package scheduler provides a distributed scheduler that uses Raft leader election.

The distributed scheduler ensures only one node in the cluster executes scheduled
workflows, preventing duplicate executions when running multiple instances.
*/
package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/dipankar/m9m/internal/consensus"
	"github.com/dipankar/m9m/internal/engine"
	"github.com/dipankar/m9m/internal/model"
	"github.com/dipankar/m9m/internal/storage"
)

// DistributedScheduler wraps WorkflowScheduler with distributed coordination
type DistributedScheduler struct {
	raft             *consensus.RaftNode
	engine           engine.WorkflowEngine
	storage          storage.WorkflowStorage
	scheduler        *WorkflowScheduler
	isLeader         bool
	leadershipMutex  sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
	leadershipTicker *time.Ticker
}

// NewDistributedScheduler creates a new distributed scheduler
func NewDistributedScheduler(raftNode *consensus.RaftNode, eng engine.WorkflowEngine, store storage.WorkflowStorage) *DistributedScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	ds := &DistributedScheduler{
		raft:             raftNode,
		engine:           eng,
		storage:          store,
		scheduler:        NewWorkflowScheduler(eng),
		isLeader:         false,
		ctx:              ctx,
		cancel:           cancel,
		leadershipTicker: time.NewTicker(5 * time.Second),
	}

	return ds
}

// Start initializes the distributed scheduler
func (ds *DistributedScheduler) Start() error {
	log.Println("Starting distributed scheduler...")

	// Start leadership monitor
	go ds.leadershipMonitor()

	log.Println("Distributed scheduler started successfully")
	return nil
}

// Stop gracefully shuts down the distributed scheduler
func (ds *DistributedScheduler) Stop() error {
	log.Println("Stopping distributed scheduler...")

	ds.cancel()
	ds.leadershipTicker.Stop()

	// Stop underlying scheduler if running
	ds.leadershipMutex.Lock()
	if ds.isLeader {
		if err := ds.scheduler.Stop(); err != nil {
			log.Printf("Error stopping scheduler: %v", err)
		}
		ds.isLeader = false
	}
	ds.leadershipMutex.Unlock()

	log.Println("Distributed scheduler stopped")
	return nil
}

// leadershipMonitor watches for leadership changes
func (ds *DistributedScheduler) leadershipMonitor() {
	log.Println("Started leadership monitor")

	for {
		select {
		case <-ds.ctx.Done():
			return
		case <-ds.leadershipTicker.C:
			ds.checkLeadership()
		}
	}
}

// checkLeadership checks if this node is the leader and acts accordingly
func (ds *DistributedScheduler) checkLeadership() {
	currentlyLeader := ds.raft.IsLeader()

	ds.leadershipMutex.Lock()
	defer ds.leadershipMutex.Unlock()

	// Leadership changed from follower to leader
	if currentlyLeader && !ds.isLeader {
		log.Println("🎯 Became Raft leader - starting scheduler")
		ds.isLeader = true

		// Start the scheduler
		if err := ds.scheduler.Start(); err != nil {
			log.Printf("Error starting scheduler: %v", err)
			return
		}

		// Load all active workflows and add schedules
		if err := ds.loadActiveSchedules(); err != nil {
			log.Printf("Error loading active schedules: %v", err)
		}
	}

	// Leadership changed from leader to follower
	if !currentlyLeader && ds.isLeader {
		log.Println("⚠️  Lost Raft leadership - stopping scheduler")
		ds.isLeader = false

		// Stop the scheduler
		if err := ds.scheduler.Stop(); err != nil {
			log.Printf("Error stopping scheduler: %v", err)
		}
	}
}

// loadActiveSchedules loads all active workflow schedules from storage
func (ds *DistributedScheduler) loadActiveSchedules() error {
	log.Println("Loading active workflow schedules...")

	// List all active workflows
	activeTrue := true
	workflows, _, err := ds.storage.ListWorkflows(storage.WorkflowFilters{
		Active: &activeTrue,
		Limit:  1000, // Load up to 1000 workflows
	})

	if err != nil {
		return err
	}

	log.Printf("Found %d active workflows", len(workflows))

	// Add schedules for workflows with cron triggers
	schedulesAdded := 0
	for _, workflow := range workflows {
		if schedule := ds.extractScheduleFromWorkflow(workflow); schedule != nil {
			if err := ds.scheduler.AddSchedule(schedule); err != nil {
				log.Printf("Warning: failed to add schedule for workflow %s: %v", workflow.ID, err)
				continue
			}
			schedulesAdded++
		}
	}

	log.Printf("Added %d workflow schedules", schedulesAdded)
	return nil
}

// extractScheduleFromWorkflow extracts schedule config from a workflow
func (ds *DistributedScheduler) extractScheduleFromWorkflow(workflow *model.Workflow) *ScheduleConfig {
	// This is a simplified implementation
	// In practice, you would parse the workflow to find cron trigger nodes
	// For now, return nil as we need to implement workflow parsing logic
	return nil
}

// AddSchedule adds a schedule (delegates to underlying scheduler if leader)
func (ds *DistributedScheduler) AddSchedule(config *ScheduleConfig) error {
	ds.leadershipMutex.RLock()
	defer ds.leadershipMutex.RUnlock()

	if !ds.isLeader {
		log.Printf("Ignoring AddSchedule request - not leader")
		return nil
	}

	return ds.scheduler.AddSchedule(config)
}

// RemoveSchedule removes a schedule (delegates to underlying scheduler if leader)
func (ds *DistributedScheduler) RemoveSchedule(scheduleID string) error {
	ds.leadershipMutex.RLock()
	defer ds.leadershipMutex.RUnlock()

	if !ds.isLeader {
		log.Printf("Ignoring RemoveSchedule request - not leader")
		return nil
	}

	return ds.scheduler.RemoveSchedule(scheduleID)
}

// UpdateSchedule updates a schedule (delegates to underlying scheduler if leader)
func (ds *DistributedScheduler) UpdateSchedule(scheduleID string, updates *ScheduleConfig) error {
	ds.leadershipMutex.RLock()
	defer ds.leadershipMutex.RUnlock()

	if !ds.isLeader {
		log.Printf("Ignoring UpdateSchedule request - not leader")
		return nil
	}

	return ds.scheduler.UpdateSchedule(scheduleID, updates)
}

// GetSchedule retrieves a schedule
func (ds *DistributedScheduler) GetSchedule(scheduleID string) (*ScheduleConfig, error) {
	ds.leadershipMutex.RLock()
	defer ds.leadershipMutex.RUnlock()

	if !ds.isLeader {
		log.Printf("GetSchedule request - not leader, returning nil")
		return nil, nil
	}

	return ds.scheduler.GetSchedule(scheduleID)
}

// ListSchedules returns all schedules
func (ds *DistributedScheduler) ListSchedules() []*ScheduleConfig {
	ds.leadershipMutex.RLock()
	defer ds.leadershipMutex.RUnlock()

	if !ds.isLeader {
		return []*ScheduleConfig{}
	}

	return ds.scheduler.ListSchedules()
}

// GetExecutionHistory returns execution history for a schedule
func (ds *DistributedScheduler) GetExecutionHistory(scheduleID string, limit int) (*ExecutionHistory, error) {
	ds.leadershipMutex.RLock()
	defer ds.leadershipMutex.RUnlock()

	if !ds.isLeader {
		return nil, nil
	}

	return ds.scheduler.GetExecutionHistory(scheduleID, limit)
}

// GetMetrics returns scheduler metrics as a snapshot
func (ds *DistributedScheduler) GetMetrics() SchedulerMetricsSnapshot {
	ds.leadershipMutex.RLock()
	defer ds.leadershipMutex.RUnlock()

	if !ds.isLeader {
		return SchedulerMetricsSnapshot{}
	}

	return ds.scheduler.GetMetrics()
}

// IsLeader returns true if this node is the leader
func (ds *DistributedScheduler) IsLeader() bool {
	ds.leadershipMutex.RLock()
	defer ds.leadershipMutex.RUnlock()
	return ds.isLeader
}

// GetLeaderAddr returns the current leader address
func (ds *DistributedScheduler) GetLeaderAddr() string {
	return ds.raft.GetLeader()
}
