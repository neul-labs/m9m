package variables

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testEncryptionKey = "test-encryption-key-32-bytes-xx!"

func newTestVariableManager() *VariableManager {
	return NewVariableManager(NewMemoryVariableStorage(), testEncryptionKey)
}

func createTestVariable(t *testing.T, m *VariableManager, key, value string, varType VariableType) *Variable {
	t.Helper()
	v, err := m.CreateVariable(&VariableCreateRequest{
		Key:   key,
		Value: value,
		Type:  varType,
	})
	require.NoError(t, err)
	require.NotNil(t, v)
	return v
}

func TestVariableManager_CreateVariable(t *testing.T) {
	tests := []struct {
		name      string
		request   *VariableCreateRequest
		expectErr bool
		errMsg    string
	}{
		{
			name: "successful global variable",
			request: &VariableCreateRequest{
				Key:   "API_URL",
				Value: "https://api.example.com",
				Type:  GlobalVariable,
			},
			expectErr: false,
		},
		{
			name: "successful workflow variable",
			request: &VariableCreateRequest{
				Key:         "BATCH_SIZE",
				Value:       "100",
				Type:        WorkflowVariable,
				Description: "Batch size for processing",
			},
			expectErr: false,
		},
		{
			name: "empty key",
			request: &VariableCreateRequest{
				Key:   "",
				Value: "value",
				Type:  GlobalVariable,
			},
			expectErr: true,
			errMsg:    "variable key is required",
		},
		{
			name: "empty value",
			request: &VariableCreateRequest{
				Key:   "key",
				Value: "",
				Type:  GlobalVariable,
			},
			expectErr: true,
			errMsg:    "variable value is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestVariableManager()
			variable, err := m.CreateVariable(tt.request)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, variable)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, variable)
			assert.NotEmpty(t, variable.ID)
			assert.Equal(t, tt.request.Key, variable.Key)
			assert.Equal(t, tt.request.Type, variable.Type)
			assert.False(t, variable.CreatedAt.IsZero())
		})
	}
}

func TestVariableManager_CreateVariable_DuplicateKey(t *testing.T) {
	m := newTestVariableManager()
	createTestVariable(t, m, "DUPLICATE_KEY", "value1", GlobalVariable)

	_, err := m.CreateVariable(&VariableCreateRequest{
		Key:   "DUPLICATE_KEY",
		Value: "value2",
		Type:  GlobalVariable,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestVariableManager_CreateVariable_Encrypted(t *testing.T) {
	m := newTestVariableManager()

	variable, err := m.CreateVariable(&VariableCreateRequest{
		Key:       "SECRET_KEY",
		Value:     "super-secret-value",
		Type:      GlobalVariable,
		Encrypted: true,
	})
	require.NoError(t, err)
	assert.True(t, variable.Encrypted)
	// The stored value should be encrypted (not equal to original)
	assert.NotEqual(t, "super-secret-value", variable.Value)

	// Retrieve without decrypt - should still be encrypted
	retrieved, err := m.GetVariable(variable.ID, false)
	require.NoError(t, err)
	assert.NotEqual(t, "super-secret-value", retrieved.Value)

	// Retrieve with decrypt - should be the original value
	decrypted, err := m.GetVariable(variable.ID, true)
	require.NoError(t, err)
	assert.Equal(t, "super-secret-value", decrypted.Value)
}

func TestVariableManager_GetVariable(t *testing.T) {
	m := newTestVariableManager()
	created := createTestVariable(t, m, "GET_TEST", "value", GlobalVariable)

	t.Run("existing variable", func(t *testing.T) {
		v, err := m.GetVariable(created.ID, false)
		require.NoError(t, err)
		assert.Equal(t, "GET_TEST", v.Key)
		assert.Equal(t, "value", v.Value)
	})

	t.Run("non-existent variable", func(t *testing.T) {
		_, err := m.GetVariable("nonexistent", false)
		assert.Error(t, err)
	})
}

func TestVariableManager_GetVariableByKey(t *testing.T) {
	m := newTestVariableManager()
	createTestVariable(t, m, "BY_KEY", "value", GlobalVariable)

	t.Run("existing key and type", func(t *testing.T) {
		v, err := m.GetVariableByKey("BY_KEY", GlobalVariable, false)
		require.NoError(t, err)
		assert.Equal(t, "BY_KEY", v.Key)
	})

	t.Run("non-existent key", func(t *testing.T) {
		_, err := m.GetVariableByKey("NOPE", GlobalVariable, false)
		assert.Error(t, err)
	})

	t.Run("wrong type", func(t *testing.T) {
		_, err := m.GetVariableByKey("BY_KEY", WorkflowVariable, false)
		assert.Error(t, err)
	})
}

func TestVariableManager_ListVariables(t *testing.T) {
	m := newTestVariableManager()
	createTestVariable(t, m, "GLOBAL_1", "val1", GlobalVariable)
	createTestVariable(t, m, "GLOBAL_2", "val2", GlobalVariable)
	createTestVariable(t, m, "WORKFLOW_1", "val3", WorkflowVariable)

	t.Run("list all", func(t *testing.T) {
		vars, total, err := m.ListVariables(VariableListFilters{})
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, vars, 3)
	})

	t.Run("filter by type", func(t *testing.T) {
		vars, total, err := m.ListVariables(VariableListFilters{Type: GlobalVariable})
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, vars, 2)
	})

	t.Run("filter by search", func(t *testing.T) {
		vars, total, err := m.ListVariables(VariableListFilters{Search: "WORKFLOW"})
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, vars, 1)
	})

	t.Run("pagination", func(t *testing.T) {
		vars, total, err := m.ListVariables(VariableListFilters{Limit: 1, Offset: 0})
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, vars, 1)
	})
}

func TestVariableManager_UpdateVariable(t *testing.T) {
	m := newTestVariableManager()
	original := createTestVariable(t, m, "UPDATE_ME", "old_value", GlobalVariable)

	t.Run("update value", func(t *testing.T) {
		newVal := "new_value"
		updated, err := m.UpdateVariable(original.ID, &VariableUpdateRequest{
			Value: &newVal,
		})
		require.NoError(t, err)
		assert.Equal(t, "new_value", updated.Value)
	})

	t.Run("update description", func(t *testing.T) {
		desc := "new description"
		updated, err := m.UpdateVariable(original.ID, &VariableUpdateRequest{
			Description: &desc,
		})
		require.NoError(t, err)
		assert.Equal(t, "new description", updated.Description)
	})

	t.Run("update tags", func(t *testing.T) {
		tags := []string{"tag1", "tag2"}
		updated, err := m.UpdateVariable(original.ID, &VariableUpdateRequest{
			Tags: tags,
		})
		require.NoError(t, err)
		assert.Equal(t, tags, updated.Tags)
	})

	t.Run("update non-existent", func(t *testing.T) {
		val := "x"
		_, err := m.UpdateVariable("nonexistent", &VariableUpdateRequest{Value: &val})
		assert.Error(t, err)
	})
}

func TestVariableManager_DeleteVariable(t *testing.T) {
	m := newTestVariableManager()
	v := createTestVariable(t, m, "DELETE_ME", "val", GlobalVariable)

	err := m.DeleteVariable(v.ID)
	require.NoError(t, err)

	_, err = m.GetVariable(v.ID, false)
	assert.Error(t, err)
}

func TestVariableManager_DeleteVariable_NonExistent(t *testing.T) {
	m := newTestVariableManager()
	err := m.DeleteVariable("nonexistent")
	assert.Error(t, err)
}

func TestVariableManager_Environment_CRUD(t *testing.T) {
	m := newTestVariableManager()

	t.Run("create environment", func(t *testing.T) {
		env, err := m.CreateEnvironment(&EnvironmentCreateRequest{
			Name:   "Development",
			Key:    "dev",
			Active: true,
			Variables: map[string]string{
				"DB_HOST": "localhost",
			},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, env.ID)
		assert.Equal(t, "Development", env.Name)
		assert.Equal(t, "dev", env.Key)
		assert.True(t, env.Active)
	})

	t.Run("create environment missing key", func(t *testing.T) {
		_, err := m.CreateEnvironment(&EnvironmentCreateRequest{
			Name: "No Key",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "environment key is required")
	})

	t.Run("create environment missing name", func(t *testing.T) {
		_, err := m.CreateEnvironment(&EnvironmentCreateRequest{
			Key: "noname",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "environment name is required")
	})

	t.Run("duplicate key", func(t *testing.T) {
		_, err := m.CreateEnvironment(&EnvironmentCreateRequest{
			Name: "Dev 2",
			Key:  "dev",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

func TestVariableManager_Environment_GetAndList(t *testing.T) {
	m := newTestVariableManager()

	env1, _ := m.CreateEnvironment(&EnvironmentCreateRequest{Name: "Dev", Key: "dev", Active: false})
	_, _ = m.CreateEnvironment(&EnvironmentCreateRequest{Name: "Staging", Key: "staging", Active: false})

	t.Run("get environment", func(t *testing.T) {
		env, err := m.GetEnvironment(env1.ID)
		require.NoError(t, err)
		assert.Equal(t, "Dev", env.Name)
	})

	t.Run("get non-existent environment", func(t *testing.T) {
		_, err := m.GetEnvironment("nonexistent")
		assert.Error(t, err)
	})

	t.Run("list environments", func(t *testing.T) {
		envs, err := m.ListEnvironments()
		require.NoError(t, err)
		assert.Len(t, envs, 2)
	})
}

func TestVariableManager_Environment_Update(t *testing.T) {
	m := newTestVariableManager()
	env, _ := m.CreateEnvironment(&EnvironmentCreateRequest{
		Name:   "Dev",
		Key:    "dev",
		Active: false,
	})

	t.Run("update name", func(t *testing.T) {
		newName := "Development"
		updated, err := m.UpdateEnvironment(env.ID, &EnvironmentUpdateRequest{
			Name: &newName,
		})
		require.NoError(t, err)
		assert.Equal(t, "Development", updated.Name)
	})

	t.Run("update active status", func(t *testing.T) {
		active := true
		updated, err := m.UpdateEnvironment(env.ID, &EnvironmentUpdateRequest{
			Active: &active,
		})
		require.NoError(t, err)
		assert.True(t, updated.Active)
	})

	t.Run("update variables", func(t *testing.T) {
		vars := map[string]string{"NEW_VAR": "new_value"}
		updated, err := m.UpdateEnvironment(env.ID, &EnvironmentUpdateRequest{
			Variables: vars,
		})
		require.NoError(t, err)
		assert.Equal(t, vars, updated.Variables)
	})

	t.Run("update non-existent", func(t *testing.T) {
		name := "x"
		_, err := m.UpdateEnvironment("nonexistent", &EnvironmentUpdateRequest{Name: &name})
		assert.Error(t, err)
	})
}

func TestVariableManager_Environment_Delete(t *testing.T) {
	m := newTestVariableManager()
	env, _ := m.CreateEnvironment(&EnvironmentCreateRequest{
		Name: "Deletable", Key: "del", Active: false,
	})

	err := m.DeleteEnvironment(env.ID)
	require.NoError(t, err)

	_, err = m.GetEnvironment(env.ID)
	assert.Error(t, err)
}

func TestVariableManager_Environment_DeleteActive(t *testing.T) {
	m := newTestVariableManager()
	env, _ := m.CreateEnvironment(&EnvironmentCreateRequest{
		Name: "Active", Key: "active", Active: true,
	})

	err := m.DeleteEnvironment(env.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "active environment")
}

func TestVariableManager_WorkflowVariables(t *testing.T) {
	m := newTestVariableManager()

	t.Run("save and get workflow variables", func(t *testing.T) {
		vars := map[string]string{"KEY1": "val1", "KEY2": "val2"}
		err := m.SaveWorkflowVariables("wf-1", vars)
		require.NoError(t, err)

		retrieved, err := m.GetWorkflowVariables("wf-1")
		require.NoError(t, err)
		assert.Equal(t, vars, retrieved)
	})

	t.Run("get non-existent workflow variables", func(t *testing.T) {
		vars, err := m.GetWorkflowVariables("no-vars")
		require.NoError(t, err)
		assert.Empty(t, vars)
	})
}

func TestVariableManager_GetVariableContext(t *testing.T) {
	m := newTestVariableManager()

	// Create global variables
	createTestVariable(t, m, "SHARED_KEY", "global_value", GlobalVariable)
	createTestVariable(t, m, "GLOBAL_ONLY", "g_value", GlobalVariable)

	// Create environment
	env, _ := m.CreateEnvironment(&EnvironmentCreateRequest{
		Name:   "Dev",
		Key:    "dev",
		Active: true,
		Variables: map[string]string{
			"SHARED_KEY": "env_value",
			"ENV_ONLY":   "e_value",
		},
	})
	require.NotNil(t, env)

	// Create workflow variables
	_ = m.SaveWorkflowVariables("wf-ctx", map[string]string{
		"SHARED_KEY": "workflow_value",
		"WF_ONLY":    "w_value",
	})

	ctx, err := m.GetVariableContext("wf-ctx", "")
	require.NoError(t, err)
	require.NotNil(t, ctx)

	// Verify the context was populated
	assert.NotEmpty(t, ctx.Global)
	assert.NotEmpty(t, ctx.Workflow)
}

func TestVariableContext_GetValue(t *testing.T) {
	vc := &VariableContext{
		Global:      map[string]string{"SHARED": "global", "G_ONLY": "g"},
		Environment: map[string]string{"SHARED": "env", "E_ONLY": "e"},
		Workflow:    map[string]string{"SHARED": "workflow", "W_ONLY": "w"},
	}

	tests := []struct {
		name     string
		key      string
		expected string
		found    bool
	}{
		{"workflow overrides all", "SHARED", "workflow", true},
		{"workflow only", "W_ONLY", "w", true},
		{"environment only", "E_ONLY", "e", true},
		{"global only", "G_ONLY", "g", true},
		{"not found", "MISSING", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, found := vc.GetValue(tt.key)
			assert.Equal(t, tt.expected, val)
			assert.Equal(t, tt.found, found)
		})
	}
}

func TestVariableContext_MergeAll(t *testing.T) {
	vc := &VariableContext{
		Global:      map[string]string{"A": "global_a", "B": "global_b"},
		Environment: map[string]string{"B": "env_b", "C": "env_c"},
		Workflow:    map[string]string{"C": "wf_c", "D": "wf_d"},
	}

	merged := vc.MergeAll()

	assert.Equal(t, "global_a", merged["A"])
	assert.Equal(t, "env_b", merged["B"])
	assert.Equal(t, "wf_c", merged["C"])
	assert.Equal(t, "wf_d", merged["D"])
	assert.Len(t, merged, 4)
}

func TestVariableManager_EncryptDecrypt(t *testing.T) {
	m := newTestVariableManager()

	// Create encrypted variable
	v, err := m.CreateVariable(&VariableCreateRequest{
		Key:       "ENC_VAR",
		Value:     "my secret data",
		Type:      GlobalVariable,
		Encrypted: true,
	})
	require.NoError(t, err)
	assert.NotEqual(t, "my secret data", v.Value)

	// Update the encrypted variable's value
	newVal := "updated secret"
	updated, err := m.UpdateVariable(v.ID, &VariableUpdateRequest{Value: &newVal})
	require.NoError(t, err)
	assert.NotEqual(t, "updated secret", updated.Value)

	// Decrypt to verify
	decrypted, err := m.GetVariable(v.ID, true)
	require.NoError(t, err)
	assert.Equal(t, "updated secret", decrypted.Value)
}

func TestNewVariableManager_DefaultKey(t *testing.T) {
	// Passing empty key should use default
	m := NewVariableManager(NewMemoryVariableStorage(), "")
	require.NotNil(t, m)
	assert.Len(t, m.encryptionKey, 32)
}

func TestNewVariableManager_ShortKey(t *testing.T) {
	m := NewVariableManager(NewMemoryVariableStorage(), "short")
	require.NotNil(t, m)
	assert.Len(t, m.encryptionKey, 32)
}

func TestNewVariableManager_LongKey(t *testing.T) {
	m := NewVariableManager(NewMemoryVariableStorage(), "this-key-is-longer-than-thirty-two-bytes-for-sure")
	require.NotNil(t, m)
	assert.Len(t, m.encryptionKey, 32)
}
