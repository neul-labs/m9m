package tags

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/dipankar/n8n-go/internal/storage"
)

// TagStorage defines the interface for tag persistence
type TagStorage interface {
	SaveTag(tag *Tag) error
	GetTag(id string) (*Tag, error)
	GetTagByName(name string) (*Tag, error)
	ListTags(filters TagListFilters) ([]*Tag, int, error)
	DeleteTag(id string) error

	// Workflow tag associations
	AddWorkflowTag(workflowID, tagID string) error
	RemoveWorkflowTag(workflowID, tagID string) error
	GetWorkflowTags(workflowID string) ([]*Tag, error)
	GetTagWorkflows(tagID string) ([]string, error)
	SetWorkflowTags(workflowID string, tagIDs []string) error
}

// MemoryTagStorage implements TagStorage using in-memory storage
type MemoryTagStorage struct {
	tags         map[string]*Tag
	workflowTags map[string]map[string]bool // workflowID -> tagID -> exists
	mu           sync.RWMutex
}

// NewMemoryTagStorage creates a new in-memory tag storage
func NewMemoryTagStorage() *MemoryTagStorage {
	return &MemoryTagStorage{
		tags:         make(map[string]*Tag),
		workflowTags: make(map[string]map[string]bool),
	}
}

func (s *MemoryTagStorage) SaveTag(tag *Tag) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate name (excluding the tag being updated)
	for _, t := range s.tags {
		if t.Name == tag.Name && t.ID != tag.ID {
			return ErrTagNameExists
		}
	}

	s.tags[tag.ID] = tag
	return nil
}

func (s *MemoryTagStorage) GetTag(id string) (*Tag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tag, exists := s.tags[id]
	if !exists {
		return nil, ErrTagNotFound
	}

	return tag, nil
}

func (s *MemoryTagStorage) GetTagByName(name string) (*Tag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, tag := range s.tags {
		if tag.Name == name {
			return tag, nil
		}
	}

	return nil, ErrTagNotFound
}

func (s *MemoryTagStorage) ListTags(filters TagListFilters) ([]*Tag, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Tag
	for _, tag := range s.tags {
		// Apply search filter
		if filters.Search != "" {
			if !strings.Contains(strings.ToLower(tag.Name), strings.ToLower(filters.Search)) {
				continue
			}
		}
		result = append(result, tag)
	}

	total := len(result)

	// Apply pagination
	if filters.Offset >= len(result) {
		return []*Tag{}, total, nil
	}

	end := filters.Offset + filters.Limit
	if end > len(result) || filters.Limit == 0 {
		end = len(result)
	}

	return result[filters.Offset:end], total, nil
}

func (s *MemoryTagStorage) DeleteTag(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tags[id]; !exists {
		return ErrTagNotFound
	}

	// Check if tag is in use
	for _, workflowTagMap := range s.workflowTags {
		if workflowTagMap[id] {
			return ErrTagInUse
		}
	}

	delete(s.tags, id)
	return nil
}

func (s *MemoryTagStorage) AddWorkflowTag(workflowID, tagID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify tag exists
	if _, exists := s.tags[tagID]; !exists {
		return ErrTagNotFound
	}

	if s.workflowTags[workflowID] == nil {
		s.workflowTags[workflowID] = make(map[string]bool)
	}

	s.workflowTags[workflowID][tagID] = true
	return nil
}

func (s *MemoryTagStorage) RemoveWorkflowTag(workflowID, tagID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.workflowTags[workflowID] != nil {
		delete(s.workflowTags[workflowID], tagID)
		if len(s.workflowTags[workflowID]) == 0 {
			delete(s.workflowTags, workflowID)
		}
	}

	return nil
}

func (s *MemoryTagStorage) GetWorkflowTags(workflowID string) ([]*Tag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Tag
	if tagMap, exists := s.workflowTags[workflowID]; exists {
		for tagID := range tagMap {
			if tag, exists := s.tags[tagID]; exists {
				result = append(result, tag)
			}
		}
	}

	return result, nil
}

func (s *MemoryTagStorage) GetTagWorkflows(tagID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []string
	for workflowID, tagMap := range s.workflowTags {
		if tagMap[tagID] {
			result = append(result, workflowID)
		}
	}

	return result, nil
}

func (s *MemoryTagStorage) SetWorkflowTags(workflowID string, tagIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify all tags exist
	for _, tagID := range tagIDs {
		if _, exists := s.tags[tagID]; !exists {
			return fmt.Errorf("tag %s not found", tagID)
		}
	}

	// Clear existing tags
	delete(s.workflowTags, workflowID)

	// Set new tags
	if len(tagIDs) > 0 {
		s.workflowTags[workflowID] = make(map[string]bool)
		for _, tagID := range tagIDs {
			s.workflowTags[workflowID][tagID] = true
		}
	}

	return nil
}

// PersistentTagStorage implements TagStorage using persistent storage
type PersistentTagStorage struct {
	store storage.WorkflowStorage
}

// NewPersistentTagStorage creates a new persistent tag storage
func NewPersistentTagStorage(store storage.WorkflowStorage) *PersistentTagStorage {
	return &PersistentTagStorage{
		store: store,
	}
}

func (s *PersistentTagStorage) SaveTag(tag *Tag) error {
	// Check for duplicate name
	existingTag, err := s.GetTagByName(tag.Name)
	if err == nil && existingTag.ID != tag.ID {
		return ErrTagNameExists
	}

	data, err := json.Marshal(tag)
	if err != nil {
		return fmt.Errorf("failed to marshal tag: %w", err)
	}

	key := fmt.Sprintf("tag:%s", tag.ID)
	if err := s.store.SaveRaw(key, data); err != nil {
		return fmt.Errorf("failed to save tag: %w", err)
	}

	// Update name index
	nameKey := fmt.Sprintf("tag_name:%s", tag.Name)
	if err := s.store.SaveRaw(nameKey, []byte(tag.ID)); err != nil {
		return fmt.Errorf("failed to save tag name index: %w", err)
	}

	return nil
}

func (s *PersistentTagStorage) GetTag(id string) (*Tag, error) {
	key := fmt.Sprintf("tag:%s", id)
	data, err := s.store.GetRaw(key)
	if err != nil {
		return nil, ErrTagNotFound
	}

	var tag Tag
	if err := json.Unmarshal(data, &tag); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tag: %w", err)
	}

	return &tag, nil
}

func (s *PersistentTagStorage) GetTagByName(name string) (*Tag, error) {
	nameKey := fmt.Sprintf("tag_name:%s", name)
	tagIDData, err := s.store.GetRaw(nameKey)
	if err != nil {
		return nil, ErrTagNotFound
	}

	return s.GetTag(string(tagIDData))
}

func (s *PersistentTagStorage) ListTags(filters TagListFilters) ([]*Tag, int, error) {
	// Get all tag keys
	prefix := "tag:"
	keys, err := s.store.ListKeys(prefix)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tags: %w", err)
	}

	var result []*Tag
	for _, key := range keys {
		data, err := s.store.GetRaw(key)
		if err != nil {
			continue
		}

		var tag Tag
		if err := json.Unmarshal(data, &tag); err != nil {
			continue
		}

		// Apply search filter
		if filters.Search != "" {
			if !strings.Contains(strings.ToLower(tag.Name), strings.ToLower(filters.Search)) {
				continue
			}
		}

		result = append(result, &tag)
	}

	total := len(result)

	// Apply pagination
	if filters.Offset >= len(result) {
		return []*Tag{}, total, nil
	}

	end := filters.Offset + filters.Limit
	if end > len(result) || filters.Limit == 0 {
		end = len(result)
	}

	return result[filters.Offset:end], total, nil
}

func (s *PersistentTagStorage) DeleteTag(id string) error {
	// Check if tag exists
	_, err := s.GetTag(id)
	if err != nil {
		return err
	}

	// Check if tag is in use
	workflows, err := s.GetTagWorkflows(id)
	if err != nil {
		return err
	}
	if len(workflows) > 0 {
		return ErrTagInUse
	}

	// Get tag to remove name index
	tag, err := s.GetTag(id)
	if err != nil {
		return err
	}

	// Delete tag
	key := fmt.Sprintf("tag:%s", id)
	if err := s.store.DeleteRaw(key); err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	// Delete name index
	nameKey := fmt.Sprintf("tag_name:%s", tag.Name)
	s.store.DeleteRaw(nameKey) // Ignore error

	return nil
}

func (s *PersistentTagStorage) AddWorkflowTag(workflowID, tagID string) error {
	// Verify tag exists
	if _, err := s.GetTag(tagID); err != nil {
		return err
	}

	key := fmt.Sprintf("workflow_tag:%s:%s", workflowID, tagID)
	data, err := json.Marshal(WorkflowTag{
		WorkflowID: workflowID,
		TagID:      tagID,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal workflow tag: %w", err)
	}

	return s.store.SaveRaw(key, data)
}

func (s *PersistentTagStorage) RemoveWorkflowTag(workflowID, tagID string) error {
	key := fmt.Sprintf("workflow_tag:%s:%s", workflowID, tagID)
	return s.store.DeleteRaw(key)
}

func (s *PersistentTagStorage) GetWorkflowTags(workflowID string) ([]*Tag, error) {
	prefix := fmt.Sprintf("workflow_tag:%s:", workflowID)
	keys, err := s.store.ListKeys(prefix)
	if err != nil {
		return []*Tag{}, nil
	}

	var result []*Tag
	for _, key := range keys {
		data, err := s.store.GetRaw(key)
		if err != nil {
			continue
		}

		var wt WorkflowTag
		if err := json.Unmarshal(data, &wt); err != nil {
			continue
		}

		tag, err := s.GetTag(wt.TagID)
		if err != nil {
			continue
		}

		result = append(result, tag)
	}

	return result, nil
}

func (s *PersistentTagStorage) GetTagWorkflows(tagID string) ([]string, error) {
	// This requires scanning all workflow_tag keys
	// In production, consider maintaining a reverse index
	prefix := "workflow_tag:"
	keys, err := s.store.ListKeys(prefix)
	if err != nil {
		return []string{}, nil
	}

	var result []string
	for _, key := range keys {
		data, err := s.store.GetRaw(key)
		if err != nil {
			continue
		}

		var wt WorkflowTag
		if err := json.Unmarshal(data, &wt); err != nil {
			continue
		}

		if wt.TagID == tagID {
			result = append(result, wt.WorkflowID)
		}
	}

	return result, nil
}

func (s *PersistentTagStorage) SetWorkflowTags(workflowID string, tagIDs []string) error {
	// Verify all tags exist
	for _, tagID := range tagIDs {
		if _, err := s.GetTag(tagID); err != nil {
			return fmt.Errorf("tag %s not found", tagID)
		}
	}

	// Remove all existing tags
	prefix := fmt.Sprintf("workflow_tag:%s:", workflowID)
	keys, err := s.store.ListKeys(prefix)
	if err == nil {
		for _, key := range keys {
			s.store.DeleteRaw(key)
		}
	}

	// Add new tags
	for _, tagID := range tagIDs {
		if err := s.AddWorkflowTag(workflowID, tagID); err != nil {
			return err
		}
	}

	return nil
}
