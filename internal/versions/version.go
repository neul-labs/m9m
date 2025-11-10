package versions

import (
	"time"

	"github.com/dipankar/n8n-go/internal/model"
)

// WorkflowVersion represents a snapshot of a workflow at a point in time
type WorkflowVersion struct {
	ID          string           `json:"id"`
	WorkflowID  string           `json:"workflowId"`
	VersionTag  string           `json:"versionTag"`      // e.g., "v1.0.0", "2023-11-10-fix"
	VersionNum  int              `json:"versionNum"`      // Sequential version number
	Workflow    *model.Workflow  `json:"workflow"`        // Full workflow snapshot
	Author      string           `json:"author"`          // User ID who created this version
	Description string           `json:"description"`     // Version description/notes
	Changes     []string         `json:"changes"`         // List of changes from previous version
	IsCurrent   bool             `json:"isCurrent"`       // Is this the current active version
	CreatedAt   time.Time        `json:"createdAt"`
	Tags        []string         `json:"tags,omitempty"`  // Custom tags
}

// VersionComparison represents a comparison between two workflow versions
type VersionComparison struct {
	FromVersion *WorkflowVersion `json:"fromVersion"`
	ToVersion   *WorkflowVersion `json:"toVersion"`
	Changes     *VersionChanges  `json:"changes"`
}

// VersionChanges details the differences between versions
type VersionChanges struct {
	NodesAdded    []string `json:"nodesAdded"`    // Node names added
	NodesRemoved  []string `json:"nodesRemoved"`  // Node names removed
	NodesModified []string `json:"nodesModified"` // Node names modified
	ConnectionsChanged bool `json:"connectionsChanged"`
	SettingsChanged    bool `json:"settingsChanged"`
	Summary            string `json:"summary"`
}

// VersionCreateRequest represents a request to create a new version
type VersionCreateRequest struct {
	VersionTag  string   `json:"versionTag,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// VersionRestoreRequest represents a request to restore a version
type VersionRestoreRequest struct {
	CreateBackup bool   `json:"createBackup"` // Create backup before restoring
	Description  string `json:"description,omitempty"`
}

// VersionListFilters defines filters for listing versions
type VersionListFilters struct {
	WorkflowID string
	Author     string
	Tags       []string
	Limit      int
	Offset     int
}
