package tags

import (
	"time"
)

// Tag represents a workflow tag for organization
type Tag struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color,omitempty"`     // Hex color code (e.g., "#FF5733")
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// WorkflowTag represents the association between a workflow and a tag
type WorkflowTag struct {
	WorkflowID string    `json:"workflowId"`
	TagID      string    `json:"tagId"`
	CreatedAt  time.Time `json:"createdAt"`
}

// TagCreateRequest is the request to create a new tag
type TagCreateRequest struct {
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

// TagUpdateRequest is the request to update a tag
type TagUpdateRequest struct {
	Name  string `json:"name,omitempty"`
	Color string `json:"color,omitempty"`
}

// TagListFilters defines filters for listing tags
type TagListFilters struct {
	Search string
	Limit  int
	Offset int
}

// WorkflowTagsRequest is the request to set workflow tags
type WorkflowTagsRequest struct {
	TagIDs []string `json:"tagIds"`
}

// Validate validates a tag create request
func (r *TagCreateRequest) Validate() error {
	if r.Name == "" {
		return ErrTagNameRequired
	}
	if len(r.Name) > 100 {
		return ErrTagNameTooLong
	}
	return nil
}

// Validate validates a tag update request
func (r *TagUpdateRequest) Validate() error {
	if r.Name != "" && len(r.Name) > 100 {
		return ErrTagNameTooLong
	}
	return nil
}
