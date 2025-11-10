package tags

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// TagManager handles tag business logic
type TagManager struct {
	storage TagStorage
}

// NewTagManager creates a new tag manager
func NewTagManager(storage TagStorage) *TagManager {
	return &TagManager{
		storage: storage,
	}
}

// CreateTag creates a new tag
func (m *TagManager) CreateTag(request *TagCreateRequest) (*Tag, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	tag := &Tag{
		ID:        generateTagID(),
		Name:      request.Name,
		Color:     request.Color,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := m.storage.SaveTag(tag); err != nil {
		return nil, err
	}

	return tag, nil
}

// GetTag retrieves a tag by ID
func (m *TagManager) GetTag(id string) (*Tag, error) {
	return m.storage.GetTag(id)
}

// GetTagByName retrieves a tag by name
func (m *TagManager) GetTagByName(name string) (*Tag, error) {
	return m.storage.GetTagByName(name)
}

// ListTags lists all tags with optional filtering
func (m *TagManager) ListTags(filters TagListFilters) ([]*Tag, int, error) {
	return m.storage.ListTags(filters)
}

// UpdateTag updates an existing tag
func (m *TagManager) UpdateTag(id string, request *TagUpdateRequest) (*Tag, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	tag, err := m.storage.GetTag(id)
	if err != nil {
		return nil, err
	}

	// Update fields
	if request.Name != "" {
		tag.Name = request.Name
	}
	if request.Color != "" {
		tag.Color = request.Color
	}
	tag.UpdatedAt = time.Now()

	if err := m.storage.SaveTag(tag); err != nil {
		return nil, err
	}

	return tag, nil
}

// DeleteTag deletes a tag
func (m *TagManager) DeleteTag(id string) error {
	return m.storage.DeleteTag(id)
}

// GetWorkflowTags gets all tags for a workflow
func (m *TagManager) GetWorkflowTags(workflowID string) ([]*Tag, error) {
	return m.storage.GetWorkflowTags(workflowID)
}

// SetWorkflowTags sets the tags for a workflow
func (m *TagManager) SetWorkflowTags(workflowID string, tagIDs []string) error {
	return m.storage.SetWorkflowTags(workflowID, tagIDs)
}

// AddWorkflowTag adds a tag to a workflow
func (m *TagManager) AddWorkflowTag(workflowID, tagID string) error {
	return m.storage.AddWorkflowTag(workflowID, tagID)
}

// RemoveWorkflowTag removes a tag from a workflow
func (m *TagManager) RemoveWorkflowTag(workflowID, tagID string) error {
	return m.storage.RemoveWorkflowTag(workflowID, tagID)
}

// GetTagWorkflows gets all workflows that have a specific tag
func (m *TagManager) GetTagWorkflows(tagID string) ([]string, error) {
	return m.storage.GetTagWorkflows(tagID)
}

// generateTagID generates a unique tag ID
func generateTagID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "tag_" + hex.EncodeToString(b)
}
