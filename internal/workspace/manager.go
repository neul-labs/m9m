package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Workspace represents a named workspace for tenancy
type Workspace struct {
	Name      string          `json:"name"`
	Path      string          `json:"path"`
	Config    WorkspaceConfig `json:"config"`
	CreatedAt time.Time       `json:"createdAt"`
	LastUsed  time.Time       `json:"lastUsed"`
}

// WorkspaceConfig holds workspace-specific settings
type WorkspaceConfig struct {
	StorageType   string   `json:"storageType"`   // sqlite, postgres, badger
	IdleTimeout   int      `json:"idleTimeout"`   // seconds, 0 = no timeout
	MaxExecutions int      `json:"maxExecutions"` // history limit
	DefaultTags   []string `json:"defaultTags"`
}

// Manager handles workspace operations
type Manager struct {
	baseDir string // ~/.m9m
}

// NewManager creates a new workspace manager
func NewManager() (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	baseDir := filepath.Join(homeDir, ".m9m")

	// Ensure base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create m9m directory: %w", err)
	}

	// Ensure workspaces directory exists
	workspacesDir := filepath.Join(baseDir, "workspaces")
	if err := os.MkdirAll(workspacesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspaces directory: %w", err)
	}

	return &Manager{baseDir: baseDir}, nil
}

// Create creates a new workspace
func (m *Manager) Create(name string, config *WorkspaceConfig) (*Workspace, error) {
	if name == "" {
		return nil, fmt.Errorf("workspace name cannot be empty")
	}

	// Check if workspace already exists
	wsPath := m.getWorkspacePath(name)
	if _, err := os.Stat(wsPath); err == nil {
		return nil, fmt.Errorf("workspace '%s' already exists", name)
	}

	// Create workspace directory
	if err := os.MkdirAll(wsPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// Create logs directory
	logsDir := filepath.Join(wsPath, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Set default config if not provided
	if config == nil {
		config = &WorkspaceConfig{
			StorageType:   "sqlite",
			IdleTimeout:   300, // 5 minutes
			MaxExecutions: 1000,
			DefaultTags:   []string{},
		}
	}

	workspace := &Workspace{
		Name:      name,
		Path:      wsPath,
		Config:    *config,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}

	// Save workspace config
	if err := m.saveWorkspaceConfig(workspace); err != nil {
		// Cleanup on failure
		os.RemoveAll(wsPath)
		return nil, fmt.Errorf("failed to save workspace config: %w", err)
	}

	return workspace, nil
}

// Get retrieves a workspace by name
func (m *Manager) Get(name string) (*Workspace, error) {
	wsPath := m.getWorkspacePath(name)
	configPath := filepath.Join(wsPath, "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("workspace '%s' not found", name)
		}
		return nil, fmt.Errorf("failed to read workspace config: %w", err)
	}

	var workspace Workspace
	if err := json.Unmarshal(data, &workspace); err != nil {
		return nil, fmt.Errorf("failed to parse workspace config: %w", err)
	}

	workspace.Path = wsPath
	return &workspace, nil
}

// List returns all workspaces
func (m *Manager) List() ([]*Workspace, error) {
	workspacesDir := filepath.Join(m.baseDir, "workspaces")
	entries, err := os.ReadDir(workspacesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspaces directory: %w", err)
	}

	var workspaces []*Workspace
	for _, entry := range entries {
		if entry.IsDir() {
			ws, err := m.Get(entry.Name())
			if err != nil {
				// Skip invalid workspaces
				continue
			}
			workspaces = append(workspaces, ws)
		}
	}

	return workspaces, nil
}

// Delete removes a workspace
func (m *Manager) Delete(name string) error {
	wsPath := m.getWorkspacePath(name)
	if _, err := os.Stat(wsPath); os.IsNotExist(err) {
		return fmt.Errorf("workspace '%s' not found", name)
	}

	// Check if it's the current workspace
	current, _ := m.GetCurrent()
	if current == name {
		// Unset current workspace
		m.SetCurrent("")
	}

	if err := os.RemoveAll(wsPath); err != nil {
		return fmt.Errorf("failed to delete workspace: %w", err)
	}

	return nil
}

// SetCurrent sets the current workspace
func (m *Manager) SetCurrent(name string) error {
	currentFile := filepath.Join(m.baseDir, "current")

	if name == "" {
		// Remove current file
		os.Remove(currentFile)
		return nil
	}

	// Verify workspace exists
	if _, err := m.Get(name); err != nil {
		return err
	}

	if err := os.WriteFile(currentFile, []byte(name), 0644); err != nil {
		return fmt.Errorf("failed to set current workspace: %w", err)
	}

	// Update last used timestamp
	ws, _ := m.Get(name)
	if ws != nil {
		ws.LastUsed = time.Now()
		m.saveWorkspaceConfig(ws)
	}

	return nil
}

// GetCurrent returns the current workspace name
func (m *Manager) GetCurrent() (string, error) {
	currentFile := filepath.Join(m.baseDir, "current")
	data, err := os.ReadFile(currentFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read current workspace: %w", err)
	}
	return string(data), nil
}

// GetCurrentWorkspace returns the current workspace
func (m *Manager) GetCurrentWorkspace() (*Workspace, error) {
	name, err := m.GetCurrent()
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, fmt.Errorf("no workspace selected, run 'm9m init' or 'm9m workspace use <name>'")
	}
	return m.Get(name)
}

// GetStoragePath returns the path to the storage database for a workspace
func (m *Manager) GetStoragePath(name string) (string, error) {
	ws, err := m.Get(name)
	if err != nil {
		return "", err
	}
	return filepath.Join(ws.Path, "m9m.db"), nil
}

// GetBaseDir returns the base m9m directory
func (m *Manager) GetBaseDir() string {
	return m.baseDir
}

// saveWorkspaceConfig saves workspace configuration to disk
func (m *Manager) saveWorkspaceConfig(ws *Workspace) error {
	configPath := filepath.Join(ws.Path, "config.json")
	data, err := json.MarshalIndent(ws, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

// getWorkspacePath returns the path for a workspace
func (m *Manager) getWorkspacePath(name string) string {
	return filepath.Join(m.baseDir, "workspaces", name)
}

// WorkspaceExists checks if a workspace exists
func (m *Manager) WorkspaceExists(name string) bool {
	wsPath := m.getWorkspacePath(name)
	_, err := os.Stat(wsPath)
	return err == nil
}
