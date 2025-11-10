package versions

import (
	"fmt"
	"log"
	"time"

	"github.com/dipankar/n8n-go/internal/model"
	"github.com/dipankar/n8n-go/internal/storage"
)

// VersionManager manages workflow versions
type VersionManager struct {
	versionStorage  VersionStorage
	workflowStorage storage.WorkflowStorage
}

// NewVersionManager creates a new version manager
func NewVersionManager(versionStorage VersionStorage, workflowStorage storage.WorkflowStorage) *VersionManager {
	return &VersionManager{
		versionStorage:  versionStorage,
		workflowStorage: workflowStorage,
	}
}

// CreateVersion creates a new version from the current workflow state
func (m *VersionManager) CreateVersion(workflowID, author string, request *VersionCreateRequest) (*WorkflowVersion, error) {
	// Get current workflow
	workflow, err := m.workflowStorage.GetWorkflow(workflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	// Get latest version to determine version number
	latestVersion, err := m.versionStorage.GetLatestVersion(workflowID)
	versionNum := 1
	if err == nil {
		versionNum = latestVersion.VersionNum + 1
	}

	// Create version tag if not provided
	versionTag := request.VersionTag
	if versionTag == "" {
		versionTag = fmt.Sprintf("v%d", versionNum)
	}

	// Calculate changes from previous version
	var changes []string
	if latestVersion != nil {
		changes = m.calculateChanges(latestVersion.Workflow, workflow)
	} else {
		changes = []string{"Initial version"}
	}

	// Create version
	version := &WorkflowVersion{
		ID:          generateID("ver"),
		WorkflowID:  workflowID,
		VersionTag:  versionTag,
		VersionNum:  versionNum,
		Workflow:    workflow,
		Author:      author,
		Description: request.Description,
		Changes:     changes,
		IsCurrent:   true, // New version becomes current
		CreatedAt:   time.Now(),
		Tags:        request.Tags,
	}

	// Unmark previous current version
	if currentVersion, err := m.versionStorage.GetCurrentVersion(workflowID); err == nil {
		currentVersion.IsCurrent = false
		m.versionStorage.SaveVersion(currentVersion)
	}

	// Save version
	if err := m.versionStorage.SaveVersion(version); err != nil {
		return nil, fmt.Errorf("failed to save version: %w", err)
	}

	log.Printf("✅ Version created: %s (workflow=%s, version=%d, author=%s)",
		version.ID, workflowID, versionNum, author)

	return version, nil
}

// GetVersion retrieves a version by ID
func (m *VersionManager) GetVersion(versionID string) (*WorkflowVersion, error) {
	return m.versionStorage.GetVersion(versionID)
}

// GetWorkflowVersion retrieves a specific version number for a workflow
func (m *VersionManager) GetWorkflowVersion(workflowID string, versionNum int) (*WorkflowVersion, error) {
	return m.versionStorage.GetWorkflowVersion(workflowID, versionNum)
}

// ListVersions lists all versions with optional filters
func (m *VersionManager) ListVersions(filters VersionListFilters) ([]*WorkflowVersion, int, error) {
	return m.versionStorage.ListVersions(filters)
}

// CompareVersions compares two versions and returns the differences
func (m *VersionManager) CompareVersions(fromVersionID, toVersionID string) (*VersionComparison, error) {
	fromVersion, err := m.versionStorage.GetVersion(fromVersionID)
	if err != nil {
		return nil, fmt.Errorf("from version not found: %w", err)
	}

	toVersion, err := m.versionStorage.GetVersion(toVersionID)
	if err != nil {
		return nil, fmt.Errorf("to version not found: %w", err)
	}

	if fromVersion.WorkflowID != toVersion.WorkflowID {
		return nil, fmt.Errorf("versions are from different workflows")
	}

	changes := m.compareWorkflows(fromVersion.Workflow, toVersion.Workflow)

	return &VersionComparison{
		FromVersion: fromVersion,
		ToVersion:   toVersion,
		Changes:     changes,
	}, nil
}

// RestoreVersion restores a workflow to a specific version
func (m *VersionManager) RestoreVersion(versionID, author string, request *VersionRestoreRequest) (*WorkflowVersion, error) {
	// Get version to restore
	version, err := m.versionStorage.GetVersion(versionID)
	if err != nil {
		return nil, fmt.Errorf("version not found: %w", err)
	}

	// Verify workflow exists
	_, err = m.workflowStorage.GetWorkflow(version.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	// Create backup if requested
	if request.CreateBackup {
		backupRequest := &VersionCreateRequest{
			VersionTag:  fmt.Sprintf("backup-before-restore-%d", time.Now().Unix()),
			Description: fmt.Sprintf("Automatic backup before restoring to version %d", version.VersionNum),
			Tags:        []string{"backup", "auto"},
		}
		if _, err := m.CreateVersion(version.WorkflowID, author, backupRequest); err != nil {
			log.Printf("⚠️  Failed to create backup: %v", err)
		}
	}

	// Restore workflow from version
	restoredWorkflow := version.Workflow
	restoredWorkflow.UpdatedAt = time.Now()

	// Update workflow in storage
	if err := m.workflowStorage.UpdateWorkflow(version.WorkflowID, restoredWorkflow); err != nil {
		return nil, fmt.Errorf("failed to restore workflow: %w", err)
	}

	// Create a new version for the restoration
	description := request.Description
	if description == "" {
		description = fmt.Sprintf("Restored from version %d (%s)", version.VersionNum, version.VersionTag)
	}

	newVersionRequest := &VersionCreateRequest{
		VersionTag:  fmt.Sprintf("restored-from-v%d", version.VersionNum),
		Description: description,
		Tags:        []string{"restore"},
	}

	_, err = m.CreateVersion(version.WorkflowID, author, newVersionRequest)
	if err != nil {
		log.Printf("⚠️  Failed to create restore version: %v", err)
	}

	log.Printf("✅ Workflow restored: workflow=%s, restored-from=v%d, author=%s",
		version.WorkflowID, version.VersionNum, author)

	// Return the version that was restored from
	return version, nil
}

// DeleteVersion deletes a version
func (m *VersionManager) DeleteVersion(versionID string) error {
	return m.versionStorage.DeleteVersion(versionID)
}

// GetCurrentVersion gets the current active version
func (m *VersionManager) GetCurrentVersion(workflowID string) (*WorkflowVersion, error) {
	return m.versionStorage.GetCurrentVersion(workflowID)
}

// calculateChanges calculates a summary of changes between two workflows
func (m *VersionManager) calculateChanges(oldWorkflow, newWorkflow *model.Workflow) []string {
	changes := []string{}

	// Check name change
	if oldWorkflow.Name != newWorkflow.Name {
		changes = append(changes, fmt.Sprintf("Renamed from '%s' to '%s'", oldWorkflow.Name, newWorkflow.Name))
	}

	// Check description change
	if oldWorkflow.Description != newWorkflow.Description {
		changes = append(changes, "Description updated")
	}

	// Check active status change
	if oldWorkflow.Active != newWorkflow.Active {
		if newWorkflow.Active {
			changes = append(changes, "Activated workflow")
		} else {
			changes = append(changes, "Deactivated workflow")
		}
	}

	// Detailed node changes
	oldNodes := make(map[string]*model.Node)
	for i := range oldWorkflow.Nodes {
		oldNodes[oldWorkflow.Nodes[i].Name] = &oldWorkflow.Nodes[i]
	}

	newNodes := make(map[string]*model.Node)
	for i := range newWorkflow.Nodes {
		newNodes[newWorkflow.Nodes[i].Name] = &newWorkflow.Nodes[i]
	}

	// Check for added nodes
	for name := range newNodes {
		if _, exists := oldNodes[name]; !exists {
			changes = append(changes, fmt.Sprintf("Added node: %s", name))
		}
	}

	// Check for removed nodes
	for name := range oldNodes {
		if _, exists := newNodes[name]; !exists {
			changes = append(changes, fmt.Sprintf("Removed node: %s", name))
		}
	}

	// Check for modified nodes
	for name, newNode := range newNodes {
		if oldNode, exists := oldNodes[name]; exists {
			if oldNode.Type != newNode.Type {
				changes = append(changes, fmt.Sprintf("Changed node type: %s", name))
			}
			// Could do deeper parameter comparison here
		}
	}

	// Check connections
	if len(oldWorkflow.Connections) != len(newWorkflow.Connections) {
		changes = append(changes, "Connections modified")
	}

	// Check settings
	if oldWorkflow.Settings != newWorkflow.Settings {
		changes = append(changes, "Settings updated")
	}

	if len(changes) == 0 {
		changes = append(changes, "Minor updates")
	}

	return changes
}

// compareWorkflows performs detailed comparison between two workflows
func (m *VersionManager) compareWorkflows(oldWorkflow, newWorkflow *model.Workflow) *VersionChanges {
	changes := &VersionChanges{
		NodesAdded:    []string{},
		NodesRemoved:  []string{},
		NodesModified: []string{},
	}

	oldNodes := make(map[string]*model.Node)
	for i := range oldWorkflow.Nodes {
		oldNodes[oldWorkflow.Nodes[i].Name] = &oldWorkflow.Nodes[i]
	}

	newNodes := make(map[string]*model.Node)
	for i := range newWorkflow.Nodes {
		newNodes[newWorkflow.Nodes[i].Name] = &newWorkflow.Nodes[i]
	}

	// Find added nodes
	for name := range newNodes {
		if _, exists := oldNodes[name]; !exists {
			changes.NodesAdded = append(changes.NodesAdded, name)
		}
	}

	// Find removed nodes
	for name := range oldNodes {
		if _, exists := newNodes[name]; !exists {
			changes.NodesRemoved = append(changes.NodesRemoved, name)
		}
	}

	// Find modified nodes
	for name, newNode := range newNodes {
		if oldNode, exists := oldNodes[name]; exists {
			if oldNode.Type != newNode.Type || !compareParameters(oldNode.Parameters, newNode.Parameters) {
				changes.NodesModified = append(changes.NodesModified, name)
			}
		}
	}

	// Check connections
	changes.ConnectionsChanged = len(oldWorkflow.Connections) != len(newWorkflow.Connections)

	// Check settings
	changes.SettingsChanged = oldWorkflow.Settings != newWorkflow.Settings

	// Generate summary
	summary := fmt.Sprintf("%d nodes added, %d removed, %d modified",
		len(changes.NodesAdded), len(changes.NodesRemoved), len(changes.NodesModified))
	if changes.ConnectionsChanged {
		summary += ", connections changed"
	}
	if changes.SettingsChanged {
		summary += ", settings changed"
	}
	changes.Summary = summary

	return changes
}

// compareParameters compares node parameters (simplified)
func compareParameters(old, new map[string]interface{}) bool {
	if len(old) != len(new) {
		return false
	}

	for key := range old {
		if _, exists := new[key]; !exists {
			return false
		}
		// Deep comparison would go here
	}

	return true
}

// generateID generates a unique ID
func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}
