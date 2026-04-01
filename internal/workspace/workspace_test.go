package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	dir := t.TempDir()
	workspacesDir := filepath.Join(dir, "workspaces")
	err := os.MkdirAll(workspacesDir, 0755)
	require.NoError(t, err)
	return &Manager{baseDir: dir}
}

func TestManager_Create(t *testing.T) {
	m := newTestManager(t)

	ws, err := m.Create("test-ws", nil)
	require.NoError(t, err)
	require.NotNil(t, ws)
	assert.Equal(t, "test-ws", ws.Name)
	assert.NotEmpty(t, ws.Path)
	assert.False(t, ws.CreatedAt.IsZero())
	// Default config
	assert.Equal(t, "sqlite", ws.Config.StorageType)
}

func TestManager_Create_CustomConfig(t *testing.T) {
	m := newTestManager(t)

	config := &WorkspaceConfig{
		StorageType:   "postgres",
		IdleTimeout:   300,
		MaxExecutions: 500,
		DefaultTags:   []string{"production"},
	}

	ws, err := m.Create("custom-ws", config)
	require.NoError(t, err)
	assert.Equal(t, "postgres", ws.Config.StorageType)
	assert.Equal(t, 300, ws.Config.IdleTimeout)
	assert.Equal(t, 500, ws.Config.MaxExecutions)
	assert.Equal(t, []string{"production"}, ws.Config.DefaultTags)
}

func TestManager_Create_EmptyName(t *testing.T) {
	m := newTestManager(t)
	_, err := m.Create("", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestManager_Create_Duplicate(t *testing.T) {
	m := newTestManager(t)

	_, err := m.Create("dup-ws", nil)
	require.NoError(t, err)

	_, err = m.Create("dup-ws", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestManager_Get(t *testing.T) {
	m := newTestManager(t)

	_, err := m.Create("get-test", nil)
	require.NoError(t, err)

	ws, err := m.Get("get-test")
	require.NoError(t, err)
	assert.Equal(t, "get-test", ws.Name)
}

func TestManager_Get_NotFound(t *testing.T) {
	m := newTestManager(t)
	_, err := m.Get("nonexistent")
	require.Error(t, err)
}

func TestManager_List(t *testing.T) {
	m := newTestManager(t)

	m.Create("ws-a", nil)
	m.Create("ws-b", nil)

	workspaces, err := m.List()
	require.NoError(t, err)
	assert.Len(t, workspaces, 2)
}

func TestManager_Delete(t *testing.T) {
	m := newTestManager(t)

	_, err := m.Create("del-ws", nil)
	require.NoError(t, err)

	err = m.Delete("del-ws")
	require.NoError(t, err)

	_, err = m.Get("del-ws")
	require.Error(t, err)
}

func TestManager_Delete_NotFound(t *testing.T) {
	m := newTestManager(t)
	err := m.Delete("nonexistent")
	require.Error(t, err)
}

func TestWorkspaceConfig_Defaults(t *testing.T) {
	config := WorkspaceConfig{}
	assert.Empty(t, config.StorageType)
	assert.Equal(t, 0, config.IdleTimeout)
	assert.Equal(t, 0, config.MaxExecutions)
	assert.Nil(t, config.DefaultTags)
}
