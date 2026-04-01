package versioning

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestVersionManager(t *testing.T) *VersionManager {
	t.Helper()
	dir := t.TempDir()
	vm, err := NewVersionManager(dir)
	require.NoError(t, err)
	// Disable auto-commit to avoid git add ordering issues
	vm.autoCommit = false
	return vm
}

func TestNewVersionManager(t *testing.T) {
	dir := t.TempDir()
	vm, err := NewVersionManager(dir)
	require.NoError(t, err)
	require.NotNil(t, vm)
	assert.Equal(t, dir, vm.storagePath)
	assert.Equal(t, 100, vm.maxVersions)
	assert.True(t, vm.autoCommit)
	assert.NotNil(t, vm.repository)
	assert.NotNil(t, vm.workTree)
}

func TestVersionManager_SaveAndGetVersion(t *testing.T) {
	vm := newTestVersionManager(t)
	var err error

	data := map[string]interface{}{
		"name": "test-workflow",
		"nodes": []interface{}{
			map[string]interface{}{"id": "node1", "type": "start"},
		},
	}

	version, err := vm.SaveVersion("wf-1", data, "Initial version", "test-author")
	require.NoError(t, err)
	require.NotNil(t, version)
	assert.Equal(t, "wf-1", version.WorkflowID)
	assert.Equal(t, 1, version.Version)
	assert.Equal(t, "Initial version", version.Message)
	assert.Equal(t, "test-author", version.Author)
	assert.NotEmpty(t, version.Hash)
	assert.False(t, version.Timestamp.IsZero())
}

func TestVersionManager_SaveMultipleVersions(t *testing.T) {
	vm := newTestVersionManager(t)
	var err error

	data1 := map[string]interface{}{"name": "v1"}
	data2 := map[string]interface{}{"name": "v2"}

	v1, err := vm.SaveVersion("wf-1", data1, "Version 1", "author")
	require.NoError(t, err)
	assert.Equal(t, 1, v1.Version)

	v2, err := vm.SaveVersion("wf-1", data2, "Version 2", "author")
	require.NoError(t, err)
	assert.Equal(t, 2, v2.Version)
}

func TestVersionManager_ListVersions(t *testing.T) {
	vm := newTestVersionManager(t)

	vm.SaveVersion("wf-1", map[string]interface{}{"v": 1}, "v1", "author")
	vm.SaveVersion("wf-1", map[string]interface{}{"v": 2}, "v2", "author")
	vm.SaveVersion("wf-1", map[string]interface{}{"v": 3}, "v3", "author")

	history, err := vm.ListVersions("wf-1")
	require.NoError(t, err)
	assert.Len(t, history, 3)
}

func TestVersionManager_ListVersions_Empty(t *testing.T) {
	vm := newTestVersionManager(t)

	history, err := vm.ListVersions("nonexistent")
	require.NoError(t, err)
	assert.Empty(t, history)
}

func TestVersionManager_GetVersion(t *testing.T) {
	vm := newTestVersionManager(t)
	var err error

	saved, err := vm.SaveVersion("wf-1", map[string]interface{}{"name": "test"}, "msg", "author")
	require.NoError(t, err)

	retrieved, err := vm.GetVersion("wf-1", saved.Version)
	require.NoError(t, err)
	assert.Equal(t, saved.Version, retrieved.Version)
	assert.Equal(t, saved.WorkflowID, retrieved.WorkflowID)
}

func TestVersionManager_CompareVersions(t *testing.T) {
	vm := newTestVersionManager(t)

	vm.SaveVersion("wf-1", map[string]interface{}{"name": "v1", "nodes": 1}, "v1", "author")
	vm.SaveVersion("wf-1", map[string]interface{}{"name": "v2", "nodes": 2}, "v2", "author")

	diff, err := vm.CompareVersions("wf-1", 1, 2)
	require.NoError(t, err)
	assert.NotNil(t, diff)
}

func TestWorkflowVersion_Struct(t *testing.T) {
	v := WorkflowVersion{
		ID:         "v-1",
		WorkflowID: "wf-1",
		Version:    1,
		Hash:       "abc123",
		Message:    "test",
		Author:     "user",
		Status:     "active",
		Tags:       []string{"production"},
		Branch:     "main",
	}

	assert.Equal(t, "v-1", v.ID)
	assert.Equal(t, "wf-1", v.WorkflowID)
	assert.Equal(t, 1, v.Version)
	assert.Equal(t, "active", v.Status)
	assert.Equal(t, "main", v.Branch)
	assert.Contains(t, v.Tags, "production")
}
