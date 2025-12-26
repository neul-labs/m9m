package versioning

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// WorkflowVersion represents a version of a workflow
type WorkflowVersion struct {
	ID          string                 `json:"id"`
	WorkflowID  string                 `json:"workflowId"`
	Version     int                    `json:"version"`
	Hash        string                 `json:"hash"`
	Message     string                 `json:"message"`
	Author      string                 `json:"author"`
	Timestamp   time.Time              `json:"timestamp"`
	Data        map[string]interface{} `json:"data"`
	Tags        []string               `json:"tags"`
	Branch      string                 `json:"branch"`
	ParentHash  string                 `json:"parentHash,omitempty"`
	Environment string                 `json:"environment,omitempty"`
	Status      string                 `json:"status"` // "draft", "active", "archived"
}

// VersionManager manages workflow versions
type VersionManager struct {
	storagePath string
	repository  *git.Repository
	workTree    *git.Worktree
	maxVersions int
	autoCommit  bool
}

// NewVersionManager creates a new version manager
func NewVersionManager(storagePath string) (*VersionManager, error) {
	vm := &VersionManager{
		storagePath: storagePath,
		maxVersions: 100,
		autoCommit:  true,
	}

	// Ensure storage directory exists
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Initialize or open git repository
	if err := vm.initGitRepo(); err != nil {
		return nil, fmt.Errorf("failed to initialize git repository: %w", err)
	}

	return vm, nil
}

// initGitRepo initializes or opens the git repository
func (vm *VersionManager) initGitRepo() error {
	repoPath := filepath.Join(vm.storagePath, ".git")

	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		// Initialize new repository
		repo, err := git.PlainInit(vm.storagePath, false)
		if err != nil {
			return fmt.Errorf("failed to initialize git repository: %w", err)
		}
		vm.repository = repo

		// Create initial commit
		w, err := repo.Worktree()
		if err != nil {
			return fmt.Errorf("failed to get worktree: %w", err)
		}
		vm.workTree = w

		// Create README
		readmePath := filepath.Join(vm.storagePath, "README.md")
		readmeContent := `# Workflow Version Control\n\nThis repository tracks workflow versions for n8n-go.`
		if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
			return err
		}

		// Add and commit
		_, err = w.Add("README.md")
		if err != nil {
			return err
		}

		_, err = w.Commit("Initial commit", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "n8n-go",
				Email: "system@n8n-go.local",
				When:  time.Now(),
			},
		})
		if err != nil {
			return err
		}
	} else {
		// Open existing repository
		repo, err := git.PlainOpen(vm.storagePath)
		if err != nil {
			return fmt.Errorf("failed to open git repository: %w", err)
		}
		vm.repository = repo

		w, err := repo.Worktree()
		if err != nil {
			return fmt.Errorf("failed to get worktree: %w", err)
		}
		vm.workTree = w
	}

	return nil
}

// SaveVersion saves a new version of a workflow
func (vm *VersionManager) SaveVersion(workflowID string, data map[string]interface{}, message string, author string) (*WorkflowVersion, error) {
	// Calculate hash of workflow data
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal workflow data: %w", err)
	}

	hash := sha256.Sum256(dataJSON)
	hashStr := hex.EncodeToString(hash[:])

	// Get current version number
	versions, _ := vm.ListVersions(workflowID)
	nextVersion := len(versions) + 1

	// Create version object
	version := &WorkflowVersion{
		ID:         fmt.Sprintf("%s-v%d", workflowID, nextVersion),
		WorkflowID: workflowID,
		Version:    nextVersion,
		Hash:       hashStr,
		Message:    message,
		Author:     author,
		Timestamp:  time.Now(),
		Data:       data,
		Status:     "active",
	}

	// Save to file
	filePath := vm.getWorkflowFilePath(workflowID)
	if err := vm.saveWorkflowFile(filePath, version); err != nil {
		return nil, err
	}

	// Git commit if auto-commit is enabled
	if vm.autoCommit {
		if err := vm.commitVersion(workflowID, version); err != nil {
			return nil, fmt.Errorf("failed to commit version: %w", err)
		}
	}

	// Save version metadata
	if err := vm.saveVersionMetadata(version); err != nil {
		return nil, err
	}

	return version, nil
}

// GetVersion retrieves a specific version of a workflow
func (vm *VersionManager) GetVersion(workflowID string, versionNumber int) (*WorkflowVersion, error) {
	versions, err := vm.ListVersions(workflowID)
	if err != nil {
		return nil, err
	}

	for _, v := range versions {
		if v.Version == versionNumber {
			return v, nil
		}
	}

	return nil, fmt.Errorf("version %d not found for workflow %s", versionNumber, workflowID)
}

// GetLatestVersion retrieves the latest version of a workflow
func (vm *VersionManager) GetLatestVersion(workflowID string) (*WorkflowVersion, error) {
	versions, err := vm.ListVersions(workflowID)
	if err != nil {
		return nil, err
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions found for workflow %s", workflowID)
	}

	return versions[len(versions)-1], nil
}

// ListVersions lists all versions of a workflow
func (vm *VersionManager) ListVersions(workflowID string) ([]*WorkflowVersion, error) {
	versionDir := filepath.Join(vm.storagePath, "versions", workflowID)
	if _, err := os.Stat(versionDir); os.IsNotExist(err) {
		return []*WorkflowVersion{}, nil
	}

	files, err := os.ReadDir(versionDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read version directory: %w", err)
	}

	var versions []*WorkflowVersion
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") {
			filePath := filepath.Join(versionDir, file.Name())
			version, err := vm.loadVersionFromFile(filePath)
			if err != nil {
				continue
			}
			versions = append(versions, version)
		}
	}

	// Sort by version number
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version < versions[j].Version
	})

	return versions, nil
}

// Rollback rolls back to a specific version
func (vm *VersionManager) Rollback(workflowID string, targetVersion int, message string, author string) (*WorkflowVersion, error) {
	// Get target version
	targetVer, err := vm.GetVersion(workflowID, targetVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get target version: %w", err)
	}

	// Create new version with rollback data
	rollbackMessage := fmt.Sprintf("Rollback to v%d: %s", targetVersion, message)
	newVersion, err := vm.SaveVersion(workflowID, targetVer.Data, rollbackMessage, author)
	if err != nil {
		return nil, fmt.Errorf("failed to create rollback version: %w", err)
	}

	// Mark as rollback
	newVersion.ParentHash = targetVer.Hash
	if err := vm.saveVersionMetadata(newVersion); err != nil {
		return nil, err
	}

	return newVersion, nil
}

// CompareVersions compares two versions and returns the differences
func (vm *VersionManager) CompareVersions(workflowID string, version1, version2 int) (map[string]interface{}, error) {
	v1, err := vm.GetVersion(workflowID, version1)
	if err != nil {
		return nil, err
	}

	v2, err := vm.GetVersion(workflowID, version2)
	if err != nil {
		return nil, err
	}

	diff := map[string]interface{}{
		"version1":   v1.Version,
		"version2":   v2.Version,
		"timestamp1": v1.Timestamp,
		"timestamp2": v2.Timestamp,
		"author1":    v1.Author,
		"author2":    v2.Author,
		"changes":    vm.calculateChanges(v1.Data, v2.Data),
	}

	return diff, nil
}

// CreateBranch creates a new branch for a workflow
func (vm *VersionManager) CreateBranch(workflowID string, branchName string, fromVersion int) error {
	// Get base version
	baseVersion, err := vm.GetVersion(workflowID, fromVersion)
	if err != nil {
		return fmt.Errorf("failed to get base version: %w", err)
	}

	// Create git branch
	headRef, err := vm.repository.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	branchRef := plumbing.NewBranchReferenceName(branchName)
	ref := plumbing.NewHashReference(branchRef, headRef.Hash())

	if err := vm.repository.Storer.SetReference(ref); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// Save branch metadata
	baseVersion.Branch = branchName
	if err := vm.saveVersionMetadata(baseVersion); err != nil {
		return err
	}

	return nil
}

// MergeBranch merges a branch back to main
func (vm *VersionManager) MergeBranch(workflowID string, branchName string, message string, author string) (*WorkflowVersion, error) {
	// This is a simplified merge - in production you'd want conflict resolution
	branches, err := vm.repository.Branches()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	var branchFound bool
	branches.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().Short() == branchName {
			branchFound = true
		}
		return nil
	})

	if !branchFound {
		return nil, fmt.Errorf("branch %s not found", branchName)
	}

	// Get latest version from branch
	latestVersion, err := vm.GetLatestVersion(workflowID)
	if err != nil {
		return nil, err
	}

	// Create merge version
	mergeMessage := fmt.Sprintf("Merge branch '%s': %s", branchName, message)
	mergedVersion, err := vm.SaveVersion(workflowID, latestVersion.Data, mergeMessage, author)
	if err != nil {
		return nil, err
	}

	return mergedVersion, nil
}

// TagVersion adds a tag to a version
func (vm *VersionManager) TagVersion(workflowID string, versionNumber int, tag string) error {
	version, err := vm.GetVersion(workflowID, versionNumber)
	if err != nil {
		return err
	}

	if version.Tags == nil {
		version.Tags = []string{}
	}
	version.Tags = append(version.Tags, tag)

	return vm.saveVersionMetadata(version)
}

// SetEnvironment sets the environment for a version
func (vm *VersionManager) SetEnvironment(workflowID string, versionNumber int, environment string) error {
	version, err := vm.GetVersion(workflowID, versionNumber)
	if err != nil {
		return err
	}

	version.Environment = environment
	return vm.saveVersionMetadata(version)
}

// Helper methods

func (vm *VersionManager) getWorkflowFilePath(workflowID string) string {
	return filepath.Join(vm.storagePath, "workflows", fmt.Sprintf("%s.json", workflowID))
}

func (vm *VersionManager) saveWorkflowFile(filePath string, version *WorkflowVersion) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(version.Data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal workflow data: %w", err)
	}

	return os.WriteFile(filePath, data, 0644)
}

func (vm *VersionManager) saveVersionMetadata(version *WorkflowVersion) error {
	versionDir := filepath.Join(vm.storagePath, "versions", version.WorkflowID)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return fmt.Errorf("failed to create version directory: %w", err)
	}

	filePath := filepath.Join(versionDir, fmt.Sprintf("v%d.json", version.Version))
	data, err := json.MarshalIndent(version, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal version metadata: %w", err)
	}

	return os.WriteFile(filePath, data, 0644)
}

func (vm *VersionManager) loadVersionFromFile(filePath string) (*WorkflowVersion, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read version file: %w", err)
	}

	var version WorkflowVersion
	if err := json.Unmarshal(data, &version); err != nil {
		return nil, fmt.Errorf("failed to unmarshal version: %w", err)
	}

	return &version, nil
}

func (vm *VersionManager) commitVersion(workflowID string, version *WorkflowVersion) error {
	// Add workflow file
	workflowPath := fmt.Sprintf("workflows/%s.json", workflowID)
	_, err := vm.workTree.Add(workflowPath)
	if err != nil {
		return err
	}

	// Add version metadata
	versionPath := fmt.Sprintf("versions/%s/v%d.json", workflowID, version.Version)
	_, err = vm.workTree.Add(versionPath)
	if err != nil {
		return err
	}

	// Commit
	_, err = vm.workTree.Commit(version.Message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  version.Author,
			Email: fmt.Sprintf("%s@n8n-go.local", strings.ReplaceAll(version.Author, " ", "")),
			When:  version.Timestamp,
		},
	})

	return err
}

func (vm *VersionManager) calculateChanges(data1, data2 map[string]interface{}) []map[string]interface{} {
	changes := []map[string]interface{}{}

	// Check for added/modified fields
	for key, value2 := range data2 {
		if value1, exists := data1[key]; !exists {
			changes = append(changes, map[string]interface{}{
				"type":  "added",
				"field": key,
				"value": value2,
			})
		} else if !reflect.DeepEqual(value1, value2) {
			changes = append(changes, map[string]interface{}{
				"type":     "modified",
				"field":    key,
				"oldValue": value1,
				"newValue": value2,
			})
		}
	}

	// Check for deleted fields
	for key, value1 := range data1 {
		if _, exists := data2[key]; !exists {
			changes = append(changes, map[string]interface{}{
				"type":  "deleted",
				"field": key,
				"value": value1,
			})
		}
	}

	return changes
}

// CleanupOldVersions removes old versions beyond the max limit
func (vm *VersionManager) CleanupOldVersions(workflowID string) error {
	versions, err := vm.ListVersions(workflowID)
	if err != nil {
		return err
	}

	if len(versions) <= vm.maxVersions {
		return nil
	}

	// Remove oldest versions
	toRemove := len(versions) - vm.maxVersions
	for i := 0; i < toRemove; i++ {
		versionPath := filepath.Join(vm.storagePath, "versions", workflowID, fmt.Sprintf("v%d.json", versions[i].Version))
		if err := os.Remove(versionPath); err != nil {
			return fmt.Errorf("failed to remove version file: %w", err)
		}
	}

	return nil
}

// ExportVersion exports a version as a standalone file
func (vm *VersionManager) ExportVersion(workflowID string, versionNumber int, exportPath string) error {
	version, err := vm.GetVersion(workflowID, versionNumber)
	if err != nil {
		return err
	}

	exportData := map[string]interface{}{
		"workflow":   version.Data,
		"metadata":   version,
		"exportedAt": time.Now(),
	}

	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal export data: %w", err)
	}

	return os.WriteFile(exportPath, data, 0644)
}

// ImportVersion imports a version from an exported file
func (vm *VersionManager) ImportVersion(exportPath string, workflowID string, message string, author string) (*WorkflowVersion, error) {
	data, err := os.ReadFile(exportPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read export file: %w", err)
	}

	var exportData map[string]interface{}
	if err := json.Unmarshal(data, &exportData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal export data: %w", err)
	}

	workflowData, ok := exportData["workflow"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid export format: workflow data not found")
	}

	importMessage := fmt.Sprintf("Import: %s", message)
	return vm.SaveVersion(workflowID, workflowData, importMessage, author)
}