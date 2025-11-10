/*
Package consensus implements Raft-based distributed consensus for n8n-go.

This package provides leader election and log replication to ensure consistency
across multiple n8n-go instances in a cluster. Only the leader runs scheduled
workflows to prevent duplicate executions.
*/
package consensus

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

// RaftCommand represents a command to be replicated via Raft
type RaftCommand struct {
	Type  string          `json:"type"`  // "put" or "delete"
	Key   string          `json:"key"`   // Key to operate on
	Value json.RawMessage `json:"value"` // Value for put operations
}

// RaftNode represents a node in the Raft cluster
type RaftNode struct {
	raft          *raft.Raft
	fsm           *FSM
	transport     *raft.NetworkTransport
	nodeID        string
	config        *raft.Config
	logStore      *raftboltdb.BoltStore
	stableStore   *raftboltdb.BoltStore
	snapshotStore raft.SnapshotStore
}

// FSM implements the Raft Finite State Machine that applies commands to BadgerDB
type FSM struct {
	db *badger.DB
}

// NewRaftNode creates a new Raft node
func NewRaftNode(nodeID string, raftAddr string, dataDir string, db *badger.DB, peers []string) (*RaftNode, error) {
	// Create Raft configuration
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(nodeID)
	config.SnapshotInterval = 120 * time.Second
	config.SnapshotThreshold = 8192
	config.HeartbeatTimeout = 1000 * time.Millisecond
	config.ElectionTimeout = 1000 * time.Millisecond
	config.CommitTimeout = 50 * time.Millisecond
	config.LeaderLeaseTimeout = 500 * time.Millisecond

	// Create data directory
	raftDir := filepath.Join(dataDir, "raft")
	if err := os.MkdirAll(raftDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create raft directory: %w", err)
	}

	// Create log store
	logStore, err := raftboltdb.NewBoltStore(filepath.Join(raftDir, "raft-log.db"))
	if err != nil {
		return nil, fmt.Errorf("failed to create log store: %w", err)
	}

	// Create stable store
	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(raftDir, "raft-stable.db"))
	if err != nil {
		return nil, fmt.Errorf("failed to create stable store: %w", err)
	}

	// Create snapshot store
	snapshotStore, err := raft.NewFileSnapshotStore(raftDir, 3, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot store: %w", err)
	}

	// Create FSM
	fsm := &FSM{db: db}

	// Create TCP transport
	addr, err := net.ResolveTCPAddr("tcp", raftAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve raft address: %w", err)
	}

	transport, err := raft.NewTCPTransport(raftAddr, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}

	// Create Raft instance
	r, err := raft.NewRaft(config, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		transport.Close()
		return nil, fmt.Errorf("failed to create raft: %w", err)
	}

	node := &RaftNode{
		raft:          r,
		fsm:           fsm,
		transport:     transport,
		nodeID:        nodeID,
		config:        config,
		logStore:      logStore,
		stableStore:   stableStore,
		snapshotStore: snapshotStore,
	}

	// Bootstrap cluster if this is the first node
	if len(peers) == 0 {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      config.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}

		f := r.BootstrapCluster(configuration)
		if err := f.Error(); err != nil && err != raft.ErrCantBootstrap {
			return nil, fmt.Errorf("failed to bootstrap cluster: %w", err)
		}

		log.Printf("Bootstrapped Raft cluster with node %s", nodeID)
	} else {
		log.Printf("Started Raft node %s, will join peers: %v", nodeID, peers)
	}

	return node, nil
}

// Apply applies a command through Raft consensus
func (rn *RaftNode) Apply(cmd RaftCommand) error {
	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	f := rn.raft.Apply(data, 10*time.Second)
	if err := f.Error(); err != nil {
		return fmt.Errorf("failed to apply command: %w", err)
	}

	// Check response
	if resp := f.Response(); resp != nil {
		if err, ok := resp.(error); ok {
			return err
		}
	}

	return nil
}

// IsLeader returns true if this node is the Raft leader
func (rn *RaftNode) IsLeader() bool {
	return rn.raft.State() == raft.Leader
}

// WaitForLeader waits for a leader to be elected
func (rn *RaftNode) WaitForLeader(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if rn.raft.Leader() != "" {
				return nil
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for leader")
			}
		}
	}
}

// GetLeader returns the current leader address
func (rn *RaftNode) GetLeader() string {
	return string(rn.raft.Leader())
}

// AddVoter adds a new voting member to the Raft cluster
func (rn *RaftNode) AddVoter(nodeID string, addr string) error {
	f := rn.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(addr), 0, 0)
	if err := f.Error(); err != nil {
		return fmt.Errorf("failed to add voter: %w", err)
	}
	return nil
}

// RemoveServer removes a server from the Raft cluster
func (rn *RaftNode) RemoveServer(nodeID string) error {
	f := rn.raft.RemoveServer(raft.ServerID(nodeID), 0, 0)
	if err := f.Error(); err != nil {
		return fmt.Errorf("failed to remove server: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the Raft node
func (rn *RaftNode) Shutdown() error {
	if err := rn.raft.Shutdown().Error(); err != nil {
		return fmt.Errorf("failed to shutdown raft: %w", err)
	}

	if err := rn.transport.Close(); err != nil {
		return fmt.Errorf("failed to close transport: %w", err)
	}

	if err := rn.logStore.Close(); err != nil {
		return fmt.Errorf("failed to close log store: %w", err)
	}

	if err := rn.stableStore.Close(); err != nil {
		return fmt.Errorf("failed to close stable store: %w", err)
	}

	return nil
}

// Stats returns Raft statistics
func (rn *RaftNode) Stats() map[string]string {
	return rn.raft.Stats()
}

// FSM Implementation

// Apply applies a Raft log entry to the FSM
func (f *FSM) Apply(log *raft.Log) interface{} {
	var cmd RaftCommand
	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal command: %w", err)
	}

	switch cmd.Type {
	case "put":
		err := f.db.Update(func(txn *badger.Txn) error {
			return txn.Set([]byte(cmd.Key), []byte(cmd.Value))
		})
		return err

	case "delete":
		err := f.db.Update(func(txn *badger.Txn) error {
			return txn.Delete([]byte(cmd.Key))
		})
		return err

	default:
		return fmt.Errorf("unknown command type: %s", cmd.Type)
	}
}

// Snapshot returns an FSM snapshot
func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	return &fsmSnapshot{db: f.db}, nil
}

// Restore restores the FSM from a snapshot
func (f *FSM) Restore(rc io.ReadCloser) error {
	defer rc.Close()

	// Read snapshot data
	data := make(map[string][]byte)
	decoder := json.NewDecoder(rc)
	if err := decoder.Decode(&data); err != nil {
		return fmt.Errorf("failed to decode snapshot: %w", err)
	}

	// Restore data to BadgerDB
	return f.db.Update(func(txn *badger.Txn) error {
		for key, value := range data {
			if err := txn.Set([]byte(key), value); err != nil {
				return err
			}
		}
		return nil
	})
}

// fsmSnapshot implements raft.FSMSnapshot
type fsmSnapshot struct {
	db *badger.DB
}

// Persist writes the snapshot to the given sink
func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	// Collect all data from BadgerDB
	data := make(map[string][]byte)

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := item.Key()
			value, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			data[string(key)] = value
		}
		return nil
	})

	if err != nil {
		sink.Cancel()
		return fmt.Errorf("failed to read database: %w", err)
	}

	// Encode and write to sink
	encoder := json.NewEncoder(sink)
	if err := encoder.Encode(data); err != nil {
		sink.Cancel()
		return fmt.Errorf("failed to encode snapshot: %w", err)
	}

	return sink.Close()
}

// Release releases the snapshot
func (s *fsmSnapshot) Release() {}
