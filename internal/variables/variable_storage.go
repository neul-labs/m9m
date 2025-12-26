package variables

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dipankar/m9m/internal/storage"
)

// VariableStorage defines the interface for variable persistence
type VariableStorage interface {
	// Variable operations
	SaveVariable(variable *Variable) error
	GetVariable(id string) (*Variable, error)
	GetVariableByKey(key string, varType VariableType) (*Variable, error)
	ListVariables(filters VariableListFilters) ([]*Variable, int, error)
	DeleteVariable(id string) error

	// Environment operations
	SaveEnvironment(environment *Environment) error
	GetEnvironment(id string) (*Environment, error)
	GetEnvironmentByKey(key string) (*Environment, error)
	ListEnvironments() ([]*Environment, error)
	DeleteEnvironment(id string) error
	GetActiveEnvironment() (*Environment, error)

	// Workflow variables operations
	SaveWorkflowVariables(workflowID string, variables map[string]string) error
	GetWorkflowVariables(workflowID string) (map[string]string, error)
	DeleteWorkflowVariables(workflowID string) error
}

// MemoryVariableStorage implements VariableStorage using in-memory storage
type MemoryVariableStorage struct {
	variables         map[string]*Variable
	variablesByKey    map[string]*Variable // key -> variable (for quick lookup)
	environments      map[string]*Environment
	environmentsByKey map[string]*Environment
	workflowVariables map[string]map[string]string
	mu                sync.RWMutex
}

// NewMemoryVariableStorage creates a new in-memory variable storage
func NewMemoryVariableStorage() *MemoryVariableStorage {
	return &MemoryVariableStorage{
		variables:         make(map[string]*Variable),
		variablesByKey:    make(map[string]*Variable),
		environments:      make(map[string]*Environment),
		environmentsByKey: make(map[string]*Environment),
		workflowVariables: make(map[string]map[string]string),
	}
}

// SaveVariable saves a variable
func (s *MemoryVariableStorage) SaveVariable(variable *Variable) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if variable.ID == "" {
		return fmt.Errorf("variable ID cannot be empty")
	}

	if variable.Key == "" {
		return fmt.Errorf("variable key cannot be empty")
	}

	if variable.CreatedAt.IsZero() {
		variable.CreatedAt = time.Now()
	}
	variable.UpdatedAt = time.Now()

	s.variables[variable.ID] = variable
	s.variablesByKey[variable.Key] = variable

	return nil
}

// GetVariable retrieves a variable by ID
func (s *MemoryVariableStorage) GetVariable(id string) (*Variable, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	variable, exists := s.variables[id]
	if !exists {
		return nil, fmt.Errorf("variable not found: %s", id)
	}

	return variable, nil
}

// GetVariableByKey retrieves a variable by key and type
func (s *MemoryVariableStorage) GetVariableByKey(key string, varType VariableType) (*Variable, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, variable := range s.variables {
		if variable.Key == key && variable.Type == varType {
			return variable, nil
		}
	}

	return nil, fmt.Errorf("variable not found: %s", key)
}

// ListVariables lists variables with optional filters
func (s *MemoryVariableStorage) ListVariables(filters VariableListFilters) ([]*Variable, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*Variable

	for _, variable := range s.variables {
		// Apply type filter
		if filters.Type != "" && variable.Type != filters.Type {
			continue
		}

		// Apply search filter
		if filters.Search != "" {
			searchLower := strings.ToLower(filters.Search)
			if !strings.Contains(strings.ToLower(variable.Key), searchLower) &&
				!strings.Contains(strings.ToLower(variable.Description), searchLower) {
				continue
			}
		}

		// Apply tags filter
		if len(filters.Tags) > 0 {
			hasTag := false
			for _, filterTag := range filters.Tags {
				for _, varTag := range variable.Tags {
					if varTag == filterTag {
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

		results = append(results, variable)
	}

	// Sort by creation time (newest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

	total := len(results)

	// Apply pagination
	if filters.Offset > 0 {
		if filters.Offset >= len(results) {
			return []*Variable{}, total, nil
		}
		results = results[filters.Offset:]
	}

	if filters.Limit > 0 && filters.Limit < len(results) {
		results = results[:filters.Limit]
	}

	return results, total, nil
}

// DeleteVariable deletes a variable
func (s *MemoryVariableStorage) DeleteVariable(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	variable, exists := s.variables[id]
	if !exists {
		return fmt.Errorf("variable not found: %s", id)
	}

	delete(s.variables, id)
	delete(s.variablesByKey, variable.Key)

	return nil
}

// SaveEnvironment saves an environment
func (s *MemoryVariableStorage) SaveEnvironment(environment *Environment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if environment.ID == "" {
		return fmt.Errorf("environment ID cannot be empty")
	}

	if environment.Key == "" {
		return fmt.Errorf("environment key cannot be empty")
	}

	if environment.CreatedAt.IsZero() {
		environment.CreatedAt = time.Now()
	}
	environment.UpdatedAt = time.Now()

	// If this environment is active, deactivate others
	if environment.Active {
		for _, env := range s.environments {
			env.Active = false
		}
	}

	s.environments[environment.ID] = environment
	s.environmentsByKey[environment.Key] = environment

	return nil
}

// GetEnvironment retrieves an environment by ID
func (s *MemoryVariableStorage) GetEnvironment(id string) (*Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	environment, exists := s.environments[id]
	if !exists {
		return nil, fmt.Errorf("environment not found: %s", id)
	}

	return environment, nil
}

// GetEnvironmentByKey retrieves an environment by key
func (s *MemoryVariableStorage) GetEnvironmentByKey(key string) (*Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	environment, exists := s.environmentsByKey[key]
	if !exists {
		return nil, fmt.Errorf("environment not found: %s", key)
	}

	return environment, nil
}

// ListEnvironments lists all environments
func (s *MemoryVariableStorage) ListEnvironments() ([]*Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*Environment
	for _, env := range s.environments {
		results = append(results, env)
	}

	// Sort by creation time
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.Before(results[j].CreatedAt)
	})

	return results, nil
}

// DeleteEnvironment deletes an environment
func (s *MemoryVariableStorage) DeleteEnvironment(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	environment, exists := s.environments[id]
	if !exists {
		return fmt.Errorf("environment not found: %s", id)
	}

	// Don't allow deleting the active environment
	if environment.Active {
		return fmt.Errorf("cannot delete the active environment")
	}

	delete(s.environments, id)
	delete(s.environmentsByKey, environment.Key)

	return nil
}

// GetActiveEnvironment retrieves the currently active environment
func (s *MemoryVariableStorage) GetActiveEnvironment() (*Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, env := range s.environments {
		if env.Active {
			return env, nil
		}
	}

	return nil, fmt.Errorf("no active environment found")
}

// SaveWorkflowVariables saves variables for a specific workflow
func (s *MemoryVariableStorage) SaveWorkflowVariables(workflowID string, variables map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if workflowID == "" {
		return fmt.Errorf("workflow ID cannot be empty")
	}

	s.workflowVariables[workflowID] = variables

	return nil
}

// GetWorkflowVariables retrieves variables for a specific workflow
func (s *MemoryVariableStorage) GetWorkflowVariables(workflowID string) (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	variables, exists := s.workflowVariables[workflowID]
	if !exists {
		return make(map[string]string), nil // Return empty map instead of error
	}

	return variables, nil
}

// DeleteWorkflowVariables deletes variables for a specific workflow
func (s *MemoryVariableStorage) DeleteWorkflowVariables(workflowID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.workflowVariables, workflowID)

	return nil
}

// PersistentVariableStorage implements VariableStorage using persistent storage
type PersistentVariableStorage struct {
	workflowStorage storage.WorkflowStorage
	mu              sync.RWMutex
}

// NewPersistentVariableStorage creates a persistent variable storage
func NewPersistentVariableStorage(workflowStorage storage.WorkflowStorage) *PersistentVariableStorage {
	return &PersistentVariableStorage{
		workflowStorage: workflowStorage,
	}
}

// SaveVariable saves a variable
func (s *PersistentVariableStorage) SaveVariable(variable *Variable) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if variable.ID == "" {
		return fmt.Errorf("variable ID cannot be empty")
	}

	if variable.CreatedAt.IsZero() {
		variable.CreatedAt = time.Now()
	}
	variable.UpdatedAt = time.Now()

	data, err := json.Marshal(variable)
	if err != nil {
		return fmt.Errorf("failed to marshal variable: %w", err)
	}

	key := fmt.Sprintf("variable:%s", variable.ID)
	return s.workflowStorage.SaveRaw(key, data)
}

// GetVariable retrieves a variable by ID
func (s *PersistentVariableStorage) GetVariable(id string) (*Variable, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("variable:%s", id)
	data, err := s.workflowStorage.GetRaw(key)
	if err != nil {
		return nil, fmt.Errorf("variable not found: %s", id)
	}

	var variable Variable
	if err := json.Unmarshal(data, &variable); err != nil {
		return nil, fmt.Errorf("failed to unmarshal variable: %w", err)
	}

	return &variable, nil
}

// GetVariableByKey retrieves a variable by key
func (s *PersistentVariableStorage) GetVariableByKey(key string, varType VariableType) (*Variable, error) {
	variables, _, err := s.ListVariables(VariableListFilters{Type: varType})
	if err != nil {
		return nil, err
	}

	for _, v := range variables {
		if v.Key == key {
			return v, nil
		}
	}

	return nil, fmt.Errorf("variable not found: %s", key)
}

// ListVariables lists variables with filters
func (s *PersistentVariableStorage) ListVariables(filters VariableListFilters) ([]*Variable, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys, err := s.workflowStorage.ListKeys("variable:")
	if err != nil {
		return nil, 0, err
	}

	var variables []*Variable
	for _, key := range keys {
		data, err := s.workflowStorage.GetRaw(key)
		if err != nil {
			continue
		}

		var variable Variable
		if err := json.Unmarshal(data, &variable); err != nil {
			continue
		}

		// Apply filters (similar to memory implementation)
		if filters.Type != "" && variable.Type != filters.Type {
			continue
		}

		variables = append(variables, &variable)
	}

	sort.Slice(variables, func(i, j int) bool {
		return variables[i].CreatedAt.After(variables[j].CreatedAt)
	})

	total := len(variables)

	// Apply pagination
	if filters.Offset > 0 {
		if filters.Offset >= len(variables) {
			return []*Variable{}, total, nil
		}
		variables = variables[filters.Offset:]
	}

	if filters.Limit > 0 && filters.Limit < len(variables) {
		variables = variables[:filters.Limit]
	}

	return variables, total, nil
}

// DeleteVariable deletes a variable
func (s *PersistentVariableStorage) DeleteVariable(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("variable:%s", id)
	return s.workflowStorage.DeleteRaw(key)
}

// SaveEnvironment saves an environment
func (s *PersistentVariableStorage) SaveEnvironment(environment *Environment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if environment.ID == "" {
		return fmt.Errorf("environment ID cannot be empty")
	}

	if environment.CreatedAt.IsZero() {
		environment.CreatedAt = time.Now()
	}
	environment.UpdatedAt = time.Now()

	// If active, deactivate others
	if environment.Active {
		envs, _ := s.ListEnvironments()
		for _, env := range envs {
			if env.ID != environment.ID {
				env.Active = false
				s.SaveEnvironment(env)
			}
		}
	}

	data, err := json.Marshal(environment)
	if err != nil {
		return fmt.Errorf("failed to marshal environment: %w", err)
	}

	key := fmt.Sprintf("environment:%s", environment.ID)
	return s.workflowStorage.SaveRaw(key, data)
}

// GetEnvironment retrieves an environment by ID
func (s *PersistentVariableStorage) GetEnvironment(id string) (*Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("environment:%s", id)
	data, err := s.workflowStorage.GetRaw(key)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %s", id)
	}

	var environment Environment
	if err := json.Unmarshal(data, &environment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal environment: %w", err)
	}

	return &environment, nil
}

// GetEnvironmentByKey retrieves an environment by key
func (s *PersistentVariableStorage) GetEnvironmentByKey(key string) (*Environment, error) {
	environments, err := s.ListEnvironments()
	if err != nil {
		return nil, err
	}

	for _, env := range environments {
		if env.Key == key {
			return env, nil
		}
	}

	return nil, fmt.Errorf("environment not found: %s", key)
}

// ListEnvironments lists all environments
func (s *PersistentVariableStorage) ListEnvironments() ([]*Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys, err := s.workflowStorage.ListKeys("environment:")
	if err != nil {
		return nil, err
	}

	var environments []*Environment
	for _, key := range keys {
		data, err := s.workflowStorage.GetRaw(key)
		if err != nil {
			continue
		}

		var environment Environment
		if err := json.Unmarshal(data, &environment); err != nil {
			continue
		}

		environments = append(environments, &environment)
	}

	sort.Slice(environments, func(i, j int) bool {
		return environments[i].CreatedAt.Before(environments[j].CreatedAt)
	})

	return environments, nil
}

// DeleteEnvironment deletes an environment
func (s *PersistentVariableStorage) DeleteEnvironment(id string) error {
	environment, err := s.GetEnvironment(id)
	if err != nil {
		return err
	}

	if environment.Active {
		return fmt.Errorf("cannot delete the active environment")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("environment:%s", id)
	return s.workflowStorage.DeleteRaw(key)
}

// GetActiveEnvironment retrieves the currently active environment
func (s *PersistentVariableStorage) GetActiveEnvironment() (*Environment, error) {
	environments, err := s.ListEnvironments()
	if err != nil {
		return nil, err
	}

	for _, env := range environments {
		if env.Active {
			return env, nil
		}
	}

	return nil, fmt.Errorf("no active environment found")
}

// SaveWorkflowVariables saves workflow variables
func (s *PersistentVariableStorage) SaveWorkflowVariables(workflowID string, variables map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(variables)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow variables: %w", err)
	}

	key := fmt.Sprintf("workflow_vars:%s", workflowID)
	return s.workflowStorage.SaveRaw(key, data)
}

// GetWorkflowVariables retrieves workflow variables
func (s *PersistentVariableStorage) GetWorkflowVariables(workflowID string) (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("workflow_vars:%s", workflowID)
	data, err := s.workflowStorage.GetRaw(key)
	if err != nil {
		return make(map[string]string), nil // Return empty map if not found
	}

	var variables map[string]string
	if err := json.Unmarshal(data, &variables); err != nil {
		return nil, fmt.Errorf("failed to unmarshal workflow variables: %w", err)
	}

	return variables, nil
}

// DeleteWorkflowVariables deletes workflow variables
func (s *PersistentVariableStorage) DeleteWorkflowVariables(workflowID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("workflow_vars:%s", workflowID)
	return s.workflowStorage.DeleteRaw(key)
}
