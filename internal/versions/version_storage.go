package versions

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/dipankar/m9m/internal/storage"
)

// VersionStorage defines the interface for version persistence
type VersionStorage interface {
	SaveVersion(version *WorkflowVersion) error
	GetVersion(versionID string) (*WorkflowVersion, error)
	GetWorkflowVersion(workflowID string, versionNum int) (*WorkflowVersion, error)
	ListVersions(filters VersionListFilters) ([]*WorkflowVersion, int, error)
	DeleteVersion(versionID string) error
	GetLatestVersion(workflowID string) (*WorkflowVersion, error)
	GetCurrentVersion(workflowID string) (*WorkflowVersion, error)
	MarkAsCurrent(versionID string) error
}

// MemoryVersionStorage implements VersionStorage using in-memory storage
type MemoryVersionStorage struct {
	workflowStorage storage.WorkflowStorage
	versions        map[string]*WorkflowVersion
	workflowIndex   map[string][]*WorkflowVersion // workflowID -> versions
	mu              sync.RWMutex
}

// NewMemoryVersionStorage creates a new in-memory version storage
func NewMemoryVersionStorage(workflowStorage storage.WorkflowStorage) *MemoryVersionStorage {
	return &MemoryVersionStorage{
		workflowStorage: workflowStorage,
		versions:        make(map[string]*WorkflowVersion),
		workflowIndex:   make(map[string][]*WorkflowVersion),
	}
}

func (s *MemoryVersionStorage) SaveVersion(version *WorkflowVersion) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if version.ID == "" {
		return fmt.Errorf("version ID cannot be empty")
	}

	if version.CreatedAt.IsZero() {
		version.CreatedAt = time.Now()
	}

	s.versions[version.ID] = version

	// Update workflow index
	workflowVersions := s.workflowIndex[version.WorkflowID]

	// Check if version already exists in index
	found := false
	for i, v := range workflowVersions {
		if v.ID == version.ID {
			workflowVersions[i] = version
			found = true
			break
		}
	}

	if !found {
		workflowVersions = append(workflowVersions, version)
	}

	// Sort by version number (descending)
	sort.Slice(workflowVersions, func(i, j int) bool {
		return workflowVersions[i].VersionNum > workflowVersions[j].VersionNum
	})

	s.workflowIndex[version.WorkflowID] = workflowVersions

	return nil
}

func (s *MemoryVersionStorage) GetVersion(versionID string) (*WorkflowVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	version, exists := s.versions[versionID]
	if !exists {
		return nil, fmt.Errorf("version not found: %s", versionID)
	}

	return version, nil
}

func (s *MemoryVersionStorage) GetWorkflowVersion(workflowID string, versionNum int) (*WorkflowVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	workflowVersions, exists := s.workflowIndex[workflowID]
	if !exists {
		return nil, fmt.Errorf("no versions found for workflow: %s", workflowID)
	}

	for _, version := range workflowVersions {
		if version.VersionNum == versionNum {
			return version, nil
		}
	}

	return nil, fmt.Errorf("version %d not found for workflow %s", versionNum, workflowID)
}

func (s *MemoryVersionStorage) ListVersions(filters VersionListFilters) ([]*WorkflowVersion, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*WorkflowVersion

	// Get versions for specific workflow
	if filters.WorkflowID != "" {
		workflowVersions, exists := s.workflowIndex[filters.WorkflowID]
		if !exists {
			return []*WorkflowVersion{}, 0, nil
		}
		results = append(results, workflowVersions...)
	} else {
		// Get all versions
		for _, version := range s.versions {
			results = append(results, version)
		}
		// Sort by creation time (newest first)
		sort.Slice(results, func(i, j int) bool {
			return results[i].CreatedAt.After(results[j].CreatedAt)
		})
	}

	// Apply filters
	filtered := make([]*WorkflowVersion, 0)
	for _, version := range results {
		if filters.Author != "" && version.Author != filters.Author {
			continue
		}

		if len(filters.Tags) > 0 {
			hasTag := false
			for _, filterTag := range filters.Tags {
				for _, versionTag := range version.Tags {
					if versionTag == filterTag {
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

		filtered = append(filtered, version)
	}

	total := len(filtered)

	// Apply pagination
	if filters.Offset > 0 {
		if filters.Offset >= len(filtered) {
			return []*WorkflowVersion{}, total, nil
		}
		filtered = filtered[filters.Offset:]
	}

	if filters.Limit > 0 && filters.Limit < len(filtered) {
		filtered = filtered[:filters.Limit]
	}

	return filtered, total, nil
}

func (s *MemoryVersionStorage) DeleteVersion(versionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	version, exists := s.versions[versionID]
	if !exists {
		return fmt.Errorf("version not found: %s", versionID)
	}

	// Don't allow deleting the current version
	if version.IsCurrent {
		return fmt.Errorf("cannot delete the current version")
	}

	// Remove from main map
	delete(s.versions, versionID)

	// Remove from workflow index
	workflowVersions := s.workflowIndex[version.WorkflowID]
	for i, v := range workflowVersions {
		if v.ID == versionID {
			s.workflowIndex[version.WorkflowID] = append(workflowVersions[:i], workflowVersions[i+1:]...)
			break
		}
	}

	return nil
}

func (s *MemoryVersionStorage) GetLatestVersion(workflowID string) (*WorkflowVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	workflowVersions, exists := s.workflowIndex[workflowID]
	if !exists || len(workflowVersions) == 0 {
		return nil, fmt.Errorf("no versions found for workflow: %s", workflowID)
	}

	// Versions are sorted by version number (descending), so first is latest
	return workflowVersions[0], nil
}

func (s *MemoryVersionStorage) GetCurrentVersion(workflowID string) (*WorkflowVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	workflowVersions, exists := s.workflowIndex[workflowID]
	if !exists {
		return nil, fmt.Errorf("no versions found for workflow: %s", workflowID)
	}

	for _, version := range workflowVersions {
		if version.IsCurrent {
			return version, nil
		}
	}

	return nil, fmt.Errorf("no current version found for workflow: %s", workflowID)
}

func (s *MemoryVersionStorage) MarkAsCurrent(versionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	version, exists := s.versions[versionID]
	if !exists {
		return fmt.Errorf("version not found: %s", versionID)
	}

	// Unmark all other versions for this workflow
	workflowVersions := s.workflowIndex[version.WorkflowID]
	for _, v := range workflowVersions {
		v.IsCurrent = false
	}

	// Mark this version as current
	version.IsCurrent = true

	return nil
}

// PersistentVersionStorage implements VersionStorage using WorkflowStorage backend
type PersistentVersionStorage struct {
	workflowStorage storage.WorkflowStorage
	mu              sync.RWMutex
}

// NewPersistentVersionStorage creates a persistent version storage
func NewPersistentVersionStorage(workflowStorage storage.WorkflowStorage) *PersistentVersionStorage {
	return &PersistentVersionStorage{
		workflowStorage: workflowStorage,
	}
}

func (s *PersistentVersionStorage) SaveVersion(version *WorkflowVersion) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if version.ID == "" {
		return fmt.Errorf("version ID cannot be empty")
	}

	if version.CreatedAt.IsZero() {
		version.CreatedAt = time.Now()
	}

	data, err := json.Marshal(version)
	if err != nil {
		return fmt.Errorf("failed to marshal version: %w", err)
	}

	key := fmt.Sprintf("version:%s", version.ID)
	if err := s.workflowStorage.SaveRaw(key, data); err != nil {
		return fmt.Errorf("failed to save version: %w", err)
	}

	// Save workflow index
	indexKey := fmt.Sprintf("version_index:%s:%d", version.WorkflowID, version.VersionNum)
	if err := s.workflowStorage.SaveRaw(indexKey, []byte(version.ID)); err != nil {
		return fmt.Errorf("failed to save version index: %w", err)
	}

	return nil
}

func (s *PersistentVersionStorage) GetVersion(versionID string) (*WorkflowVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("version:%s", versionID)
	data, err := s.workflowStorage.GetRaw(key)
	if err != nil {
		return nil, fmt.Errorf("version not found: %s", versionID)
	}

	var version WorkflowVersion
	if err := json.Unmarshal(data, &version); err != nil {
		return nil, fmt.Errorf("failed to unmarshal version: %w", err)
	}

	return &version, nil
}

func (s *PersistentVersionStorage) GetWorkflowVersion(workflowID string, versionNum int) (*WorkflowVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	indexKey := fmt.Sprintf("version_index:%s:%d", workflowID, versionNum)
	versionID, err := s.workflowStorage.GetRaw(indexKey)
	if err != nil {
		return nil, fmt.Errorf("version %d not found for workflow %s", versionNum, workflowID)
	}

	return s.GetVersion(string(versionID))
}

func (s *PersistentVersionStorage) ListVersions(filters VersionListFilters) ([]*WorkflowVersion, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys, err := s.workflowStorage.ListKeys("version:")
	if err != nil {
		return nil, 0, err
	}

	var versions []*WorkflowVersion
	for _, key := range keys {
		data, err := s.workflowStorage.GetRaw(key)
		if err != nil {
			continue
		}

		var version WorkflowVersion
		if err := json.Unmarshal(data, &version); err != nil {
			continue
		}

		// Apply filters
		if filters.WorkflowID != "" && version.WorkflowID != filters.WorkflowID {
			continue
		}
		if filters.Author != "" && version.Author != filters.Author {
			continue
		}

		versions = append(versions, &version)
	}

	// Sort by creation time (newest first)
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].CreatedAt.After(versions[j].CreatedAt)
	})

	total := len(versions)

	// Apply pagination
	if filters.Offset > 0 {
		if filters.Offset >= len(versions) {
			return []*WorkflowVersion{}, total, nil
		}
		versions = versions[filters.Offset:]
	}

	if filters.Limit > 0 && filters.Limit < len(versions) {
		versions = versions[:filters.Limit]
	}

	return versions, total, nil
}

func (s *PersistentVersionStorage) DeleteVersion(versionID string) error {
	version, err := s.GetVersion(versionID)
	if err != nil {
		return err
	}

	if version.IsCurrent {
		return fmt.Errorf("cannot delete the current version")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("version:%s", versionID)
	if err := s.workflowStorage.DeleteRaw(key); err != nil {
		return err
	}

	// Delete index
	indexKey := fmt.Sprintf("version_index:%s:%d", version.WorkflowID, version.VersionNum)
	s.workflowStorage.DeleteRaw(indexKey)

	return nil
}

func (s *PersistentVersionStorage) GetLatestVersion(workflowID string) (*WorkflowVersion, error) {
	versions, _, err := s.ListVersions(VersionListFilters{
		WorkflowID: workflowID,
		Limit:      1,
	})
	if err != nil {
		return nil, err
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions found for workflow: %s", workflowID)
	}

	return versions[0], nil
}

func (s *PersistentVersionStorage) GetCurrentVersion(workflowID string) (*WorkflowVersion, error) {
	versions, _, err := s.ListVersions(VersionListFilters{
		WorkflowID: workflowID,
	})
	if err != nil {
		return nil, err
	}

	for _, version := range versions {
		if version.IsCurrent {
			return version, nil
		}
	}

	return nil, fmt.Errorf("no current version found for workflow: %s", workflowID)
}

func (s *PersistentVersionStorage) MarkAsCurrent(versionID string) error {
	version, err := s.GetVersion(versionID)
	if err != nil {
		return err
	}

	// Get all versions for this workflow and update them
	versions, _, err := s.ListVersions(VersionListFilters{
		WorkflowID: version.WorkflowID,
	})
	if err != nil {
		return err
	}

	for _, v := range versions {
		v.IsCurrent = (v.ID == versionID)
		if err := s.SaveVersion(v); err != nil {
			return err
		}
	}

	return nil
}
