/*
Package storage provides BadgerDB-based persistent storage with Raft replication.

This implementation uses BadgerDB as an embedded LSM-tree key-value store and
integrates with Raft for distributed consensus and replication across cluster nodes.
*/
package storage

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/dipankar/n8n-go/internal/consensus"
	"github.com/dipankar/n8n-go/internal/model"
)

// BadgerStorage implements WorkflowStorage using BadgerDB with Raft replication
type BadgerStorage struct {
	db   *badger.DB
	raft *consensus.RaftNode
}

// NewBadgerStorage creates a new BadgerDB storage instance
func NewBadgerStorage(dataDir string, raftNode *consensus.RaftNode) (*BadgerStorage, error) {
	// Configure BadgerDB
	opts := badger.DefaultOptions(filepath.Join(dataDir, "badger"))
	opts.Logger = nil // Disable verbose logging

	// Open database
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger database: %w", err)
	}

	return &BadgerStorage{
		db:   db,
		raft: raftNode,
	}, nil
}

// Workflow Operations

func (s *BadgerStorage) SaveWorkflow(workflow *model.Workflow) error {
	// Set timestamps
	if workflow.CreatedAt.IsZero() {
		workflow.CreatedAt = time.Now()
	}
	workflow.UpdatedAt = time.Now()

	data, err := json.Marshal(workflow)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}

	// Apply through Raft for replication
	cmd := consensus.RaftCommand{
		Type:  "put",
		Key:   "workflow:" + workflow.ID,
		Value: json.RawMessage(data),
	}

	return s.raft.Apply(cmd)
}

func (s *BadgerStorage) GetWorkflow(id string) (*model.Workflow, error) {
	var workflow model.Workflow

	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("workflow:" + id))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &workflow)
		})
	})

	if err == badger.ErrKeyNotFound {
		return nil, fmt.Errorf("workflow not found: %s", id)
	}

	return &workflow, err
}

func (s *BadgerStorage) ListWorkflows(filters WorkflowFilters) ([]*model.Workflow, int, error) {
	var workflows []*model.Workflow

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("workflow:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var workflow model.Workflow
				if err := json.Unmarshal(val, &workflow); err != nil {
					return err
				}

				// Apply filters
				if filters.Active != nil && workflow.Active != *filters.Active {
					return nil
				}

				if filters.Search != "" {
					if !strings.Contains(strings.ToLower(workflow.Name), strings.ToLower(filters.Search)) {
						return nil
					}
				}

				workflows = append(workflows, &workflow)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, 0, err
	}

	// Apply pagination
	total := len(workflows)
	if filters.Offset >= len(workflows) {
		return []*model.Workflow{}, total, nil
	}

	end := filters.Offset + filters.Limit
	if filters.Limit == 0 || end > len(workflows) {
		end = len(workflows)
	}

	return workflows[filters.Offset:end], total, nil
}

func (s *BadgerStorage) UpdateWorkflow(id string, workflow *model.Workflow) error {
	// Get existing workflow to preserve created_at
	existing, err := s.GetWorkflow(id)
	if err != nil {
		return err
	}

	workflow.ID = id
	workflow.CreatedAt = existing.CreatedAt
	workflow.UpdatedAt = time.Now()

	return s.SaveWorkflow(workflow)
}

func (s *BadgerStorage) DeleteWorkflow(id string) error {
	cmd := consensus.RaftCommand{
		Type: "delete",
		Key:  "workflow:" + id,
	}

	return s.raft.Apply(cmd)
}

func (s *BadgerStorage) ActivateWorkflow(id string) error {
	workflow, err := s.GetWorkflow(id)
	if err != nil {
		return err
	}

	workflow.Active = true
	return s.SaveWorkflow(workflow)
}

func (s *BadgerStorage) DeactivateWorkflow(id string) error {
	workflow, err := s.GetWorkflow(id)
	if err != nil {
		return err
	}

	workflow.Active = false
	return s.SaveWorkflow(workflow)
}

// Execution Operations

func (s *BadgerStorage) SaveExecution(execution *model.WorkflowExecution) error {
	if execution.StartedAt.IsZero() {
		execution.StartedAt = time.Now()
	}

	data, err := json.Marshal(execution)
	if err != nil {
		return fmt.Errorf("failed to marshal execution: %w", err)
	}

	cmd := consensus.RaftCommand{
		Type:  "put",
		Key:   "execution:" + execution.ID,
		Value: json.RawMessage(data),
	}

	return s.raft.Apply(cmd)
}

func (s *BadgerStorage) GetExecution(id string) (*model.WorkflowExecution, error) {
	var execution model.WorkflowExecution

	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("execution:" + id))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &execution)
		})
	})

	if err == badger.ErrKeyNotFound {
		return nil, fmt.Errorf("execution not found: %s", id)
	}

	return &execution, err
}

func (s *BadgerStorage) ListExecutions(filters ExecutionFilters) ([]*model.WorkflowExecution, int, error) {
	var executions []*model.WorkflowExecution

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("execution:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var execution model.WorkflowExecution
				if err := json.Unmarshal(val, &execution); err != nil {
					return err
				}

				// Apply filters
				if filters.WorkflowID != "" && execution.WorkflowID != filters.WorkflowID {
					return nil
				}

				if filters.Status != "" && execution.Status != filters.Status {
					return nil
				}

				executions = append(executions, &execution)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, 0, err
	}

	// Apply pagination
	total := len(executions)
	if filters.Offset >= len(executions) {
		return []*model.WorkflowExecution{}, total, nil
	}

	end := filters.Offset + filters.Limit
	if filters.Limit == 0 || end > len(executions) {
		end = len(executions)
	}

	return executions[filters.Offset:end], total, nil
}

func (s *BadgerStorage) DeleteExecution(id string) error {
	cmd := consensus.RaftCommand{
		Type: "delete",
		Key:  "execution:" + id,
	}

	return s.raft.Apply(cmd)
}

// Credential Operations

func (s *BadgerStorage) SaveCredential(credential *Credential) error {
	if credential.CreatedAt.IsZero() {
		credential.CreatedAt = time.Now()
	}
	credential.UpdatedAt = time.Now()

	data, err := json.Marshal(credential)
	if err != nil {
		return fmt.Errorf("failed to marshal credential: %w", err)
	}

	cmd := consensus.RaftCommand{
		Type:  "put",
		Key:   "credential:" + credential.ID,
		Value: json.RawMessage(data),
	}

	return s.raft.Apply(cmd)
}

func (s *BadgerStorage) GetCredential(id string) (*Credential, error) {
	var credential Credential

	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("credential:" + id))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &credential)
		})
	})

	if err == badger.ErrKeyNotFound {
		return nil, fmt.Errorf("credential not found: %s", id)
	}

	return &credential, err
}

func (s *BadgerStorage) ListCredentials() ([]*Credential, error) {
	var credentials []*Credential

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("credential:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var credential Credential
				if err := json.Unmarshal(val, &credential); err != nil {
					return err
				}

				credentials = append(credentials, &credential)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	return credentials, err
}

func (s *BadgerStorage) UpdateCredential(id string, credential *Credential) error {
	existing, err := s.GetCredential(id)
	if err != nil {
		return err
	}

	credential.ID = id
	credential.CreatedAt = existing.CreatedAt
	credential.UpdatedAt = time.Now()

	return s.SaveCredential(credential)
}

func (s *BadgerStorage) DeleteCredential(id string) error {
	cmd := consensus.RaftCommand{
		Type: "delete",
		Key:  "credential:" + id,
	}

	return s.raft.Apply(cmd)
}

// Tag Operations

func (s *BadgerStorage) SaveTag(tag *Tag) error {
	if tag.CreatedAt.IsZero() {
		tag.CreatedAt = time.Now()
	}
	tag.UpdatedAt = time.Now()

	data, err := json.Marshal(tag)
	if err != nil {
		return fmt.Errorf("failed to marshal tag: %w", err)
	}

	cmd := consensus.RaftCommand{
		Type:  "put",
		Key:   "tag:" + tag.ID,
		Value: json.RawMessage(data),
	}

	return s.raft.Apply(cmd)
}

func (s *BadgerStorage) GetTag(id string) (*Tag, error) {
	var tag Tag

	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("tag:" + id))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &tag)
		})
	})

	if err == badger.ErrKeyNotFound {
		return nil, fmt.Errorf("tag not found: %s", id)
	}

	return &tag, err
}

func (s *BadgerStorage) ListTags() ([]*Tag, error) {
	var tags []*Tag

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("tag:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var tag Tag
				if err := json.Unmarshal(val, &tag); err != nil {
					return err
				}

				tags = append(tags, &tag)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	return tags, err
}

func (s *BadgerStorage) UpdateTag(id string, tag *Tag) error {
	existing, err := s.GetTag(id)
	if err != nil {
		return err
	}

	tag.ID = id
	tag.CreatedAt = existing.CreatedAt
	tag.UpdatedAt = time.Now()

	return s.SaveTag(tag)
}

func (s *BadgerStorage) DeleteTag(id string) error {
	cmd := consensus.RaftCommand{
		Type: "delete",
		Key:  "tag:" + id,
	}

	return s.raft.Apply(cmd)
}

// Raw key-value operations

func (s *BadgerStorage) SaveRaw(key string, value []byte) error {
	cmd := consensus.RaftCommand{
		Type:  "put",
		Key:   key,
		Value: json.RawMessage(value),
	}

	return s.raft.Apply(cmd)
}

func (s *BadgerStorage) GetRaw(key string) ([]byte, error) {
	var value []byte

	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			value = append([]byte{}, val...)
			return nil
		})
	})

	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, fmt.Errorf("key not found: %s", key)
		}
		return nil, fmt.Errorf("failed to get raw value: %w", err)
	}

	return value, nil
}

func (s *BadgerStorage) ListKeys(prefix string) ([]string, error) {
	var keys []string

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false // Only need keys
		it := txn.NewIterator(opts)
		defer it.Close()

		prefixBytes := []byte(prefix)
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			key := string(item.Key())
			keys = append(keys, key)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	return keys, nil
}

func (s *BadgerStorage) DeleteRaw(key string) error {
	cmd := consensus.RaftCommand{
		Type: "delete",
		Key:  key,
	}

	return s.raft.Apply(cmd)
}

// Close gracefully shuts down the storage
func (s *BadgerStorage) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("failed to close badger database: %w", err)
	}
	return nil
}
