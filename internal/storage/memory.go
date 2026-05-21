package storage

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/tenancy"
)

// MemoryStorage provides in-memory workflow storage (data will not persist)
type MemoryStorage struct {
	workflows   map[string]*model.Workflow
	executions  map[string]*model.WorkflowExecution
	credentials map[string]*Credential
	tags        map[string]*Tag
	workspaces  map[string]*tenancy.Workspace
	rawData     map[string][]byte // For webhooks and other extensibility
	mu          sync.RWMutex
}

// NewMemoryStorage creates a new in-memory storage instance
func NewMemoryStorage() *MemoryStorage {
	s := &MemoryStorage{
		workflows:   make(map[string]*model.Workflow),
		executions:  make(map[string]*model.WorkflowExecution),
		credentials: make(map[string]*Credential),
		tags:        make(map[string]*Tag),
		workspaces:  make(map[string]*tenancy.Workspace),
		rawData:     make(map[string][]byte),
	}
	// Bootstrap the default workspace. Errors here are not possible for
	// MemoryStorage so we ignore the return.
	_ = bootstrapDefaultWorkspace(s)
	return s
}

// Workflow operations

func (s *MemoryStorage) SaveWorkflow(workflow *model.Workflow) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if workflow.ID == "" {
		workflow.ID = generateID("workflow")
	}
	workflow.WorkspaceID = resolveWorkspaceID(workflow.WorkspaceID)

	now := time.Now()
	if workflow.CreatedAt.IsZero() {
		workflow.CreatedAt = now
	}
	workflow.UpdatedAt = now

	s.workflows[workflow.ID] = workflow
	return nil
}

func (s *MemoryStorage) GetWorkflow(id string) (*model.Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	workflow, exists := s.workflows[id]
	if !exists {
		return nil, fmt.Errorf("workflow not found: %s", id)
	}

	return workflow, nil
}

func (s *MemoryStorage) ListWorkflows(filters WorkflowFilters) ([]*model.Workflow, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*model.Workflow

	for _, workflow := range s.workflows {
		// Apply filters
		if filters.WorkspaceID != "" && workflow.WorkspaceID != filters.WorkspaceID {
			continue
		}

		if filters.Active != nil && workflow.Active != *filters.Active {
			continue
		}

		if filters.Search != "" {
			searchLower := strings.ToLower(filters.Search)
			nameLower := strings.ToLower(workflow.Name)
			if !strings.Contains(nameLower, searchLower) {
				continue
			}
		}

		if len(filters.Tags) > 0 {
			hasTag := false
			for _, filterTag := range filters.Tags {
				for _, workflowTag := range workflow.Tags {
					if strings.EqualFold(filterTag, workflowTag) {
						hasTag = true
						break
					}
				}
				if hasTag {
					break
				}
			}
			if !hasTag {
				continue
			}
		}

		results = append(results, workflow)
	}

	total := len(results)

	// Apply pagination
	if filters.Limit > 0 {
		start := filters.Offset
		end := filters.Offset + filters.Limit

		if start >= len(results) {
			results = []*model.Workflow{}
		} else {
			if end > len(results) {
				end = len(results)
			}
			results = results[start:end]
		}
	}

	return results, total, nil
}

func (s *MemoryStorage) UpdateWorkflow(id string, workflow *model.Workflow) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.workflows[id]; !exists {
		return fmt.Errorf("workflow not found: %s", id)
	}

	workflow.ID = id
	workflow.UpdatedAt = time.Now()
	s.workflows[id] = workflow

	return nil
}

func (s *MemoryStorage) DeleteWorkflow(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.workflows[id]; !exists {
		return fmt.Errorf("workflow not found: %s", id)
	}

	delete(s.workflows, id)
	return nil
}

func (s *MemoryStorage) ActivateWorkflow(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workflow, exists := s.workflows[id]
	if !exists {
		return fmt.Errorf("workflow not found: %s", id)
	}

	workflow.Active = true
	workflow.UpdatedAt = time.Now()

	return nil
}

func (s *MemoryStorage) DeactivateWorkflow(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workflow, exists := s.workflows[id]
	if !exists {
		return fmt.Errorf("workflow not found: %s", id)
	}

	workflow.Active = false
	workflow.UpdatedAt = time.Now()

	return nil
}

// Execution operations

func (s *MemoryStorage) SaveExecution(execution *model.WorkflowExecution) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if execution.ID == "" {
		execution.ID = generateID("exec")
	}
	execution.WorkspaceID = resolveWorkspaceID(execution.WorkspaceID)

	s.executions[execution.ID] = execution
	return nil
}

func (s *MemoryStorage) GetExecution(id string) (*model.WorkflowExecution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	execution, exists := s.executions[id]
	if !exists {
		return nil, fmt.Errorf("execution not found: %s", id)
	}

	return execution, nil
}

func (s *MemoryStorage) ListExecutions(filters ExecutionFilters) ([]*model.WorkflowExecution, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*model.WorkflowExecution

	for _, execution := range s.executions {
		// Apply filters
		if filters.WorkspaceID != "" && execution.WorkspaceID != filters.WorkspaceID {
			continue
		}

		if filters.WorkflowID != "" && execution.WorkflowID != filters.WorkflowID {
			continue
		}

		if filters.Status != "" && execution.Status != filters.Status {
			continue
		}

		results = append(results, execution)
	}

	total := len(results)

	// Apply pagination
	if filters.Limit > 0 {
		start := filters.Offset
		end := filters.Offset + filters.Limit

		if start >= len(results) {
			results = []*model.WorkflowExecution{}
		} else {
			if end > len(results) {
				end = len(results)
			}
			results = results[start:end]
		}
	}

	return results, total, nil
}

func (s *MemoryStorage) DeleteExecution(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.executions[id]; !exists {
		return fmt.Errorf("execution not found: %s", id)
	}

	delete(s.executions, id)
	return nil
}

// Credential operations

func (s *MemoryStorage) SaveCredential(credential *Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if credential.ID == "" {
		credential.ID = generateID("cred")
	}

	now := time.Now()
	if credential.CreatedAt.IsZero() {
		credential.CreatedAt = now
	}
	credential.UpdatedAt = now

	s.credentials[credential.ID] = credential
	return nil
}

func (s *MemoryStorage) GetCredential(id string) (*Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	credential, exists := s.credentials[id]
	if !exists {
		return nil, fmt.Errorf("credential not found: %s", id)
	}

	return credential, nil
}

func (s *MemoryStorage) ListCredentials() ([]*Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]*Credential, 0, len(s.credentials))
	for _, credential := range s.credentials {
		results = append(results, credential)
	}

	return results, nil
}

func (s *MemoryStorage) UpdateCredential(id string, credential *Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.credentials[id]; !exists {
		return fmt.Errorf("credential not found: %s", id)
	}

	credential.ID = id
	credential.UpdatedAt = time.Now()
	s.credentials[id] = credential

	return nil
}

func (s *MemoryStorage) DeleteCredential(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.credentials[id]; !exists {
		return fmt.Errorf("credential not found: %s", id)
	}

	delete(s.credentials, id)
	return nil
}

// Tag operations

func (s *MemoryStorage) SaveTag(tag *Tag) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if tag.ID == "" {
		tag.ID = generateID("tag")
	}

	now := time.Now()
	if tag.CreatedAt.IsZero() {
		tag.CreatedAt = now
	}
	tag.UpdatedAt = now

	s.tags[tag.ID] = tag
	return nil
}

func (s *MemoryStorage) GetTag(id string) (*Tag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tag, exists := s.tags[id]
	if !exists {
		return nil, fmt.Errorf("tag not found: %s", id)
	}

	return tag, nil
}

func (s *MemoryStorage) ListTags() ([]*Tag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]*Tag, 0, len(s.tags))
	for _, tag := range s.tags {
		results = append(results, tag)
	}

	return results, nil
}

func (s *MemoryStorage) UpdateTag(id string, tag *Tag) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tags[id]; !exists {
		return fmt.Errorf("tag not found: %s", id)
	}

	tag.ID = id
	tag.UpdatedAt = time.Now()
	s.tags[id] = tag

	return nil
}

func (s *MemoryStorage) DeleteTag(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tags[id]; !exists {
		return fmt.Errorf("tag not found: %s", id)
	}

	delete(s.tags, id)
	return nil
}

// Workspace operations

func (s *MemoryStorage) SaveWorkspace(ws *tenancy.Workspace) error {
	if err := ws.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if ws.CreatedAt.IsZero() {
		ws.CreatedAt = now
	}
	ws.UpdatedAt = now
	// Store a copy so callers can't mutate after Save.
	clone := *ws
	s.workspaces[ws.ID] = &clone
	return nil
}

func (s *MemoryStorage) GetWorkspace(id string) (*tenancy.Workspace, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ws, ok := s.workspaces[id]
	if !ok {
		return nil, fmt.Errorf("workspace not found: %s", id)
	}
	clone := *ws
	return &clone, nil
}

func (s *MemoryStorage) ListWorkspaces() ([]*tenancy.Workspace, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*tenancy.Workspace, 0, len(s.workspaces))
	for _, ws := range s.workspaces {
		clone := *ws
		out = append(out, &clone)
	}
	return out, nil
}

func (s *MemoryStorage) DeleteWorkspace(id string) error {
	if id == tenancy.DefaultID {
		return fmt.Errorf("cannot delete default workspace")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.workspaces[id]; !ok {
		return fmt.Errorf("workspace not found: %s", id)
	}
	delete(s.workspaces, id)
	return nil
}

// Raw key-value operations

func (s *MemoryStorage) SaveRaw(key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Make a copy to avoid external mutations
	data := make([]byte, len(value))
	copy(data, value)
	s.rawData[key] = data

	return nil
}

func (s *MemoryStorage) GetRaw(key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, exists := s.rawData[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	// Return a copy to avoid external mutations
	result := make([]byte, len(data))
	copy(result, data)

	return result, nil
}

func (s *MemoryStorage) ListKeys(prefix string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var keys []string
	for key := range s.rawData {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

func (s *MemoryStorage) DeleteRaw(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.rawData[key]; !exists {
		return fmt.Errorf("key not found: %s", key)
	}

	delete(s.rawData, key)
	return nil
}

// Close closes the storage (no-op for memory storage)
func (s *MemoryStorage) Close() error {
	return nil
}

// Helper function to generate unique IDs.
//
// We combine a UnixNano timestamp with an atomic counter to remain unique
// under rapid successive calls (e.g. bulk imports, tightly-looped test
// fixtures). The previous implementation used time.Now().UnixNano() alone
// and collided when called twice within the same nanosecond — observable
// on macOS where time resolution is coarser than a nanosecond.
func generateID(prefix string) string {
	n := atomic.AddUint64(&idCounter, 1)
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixNano(), n)
}

var idCounter uint64

