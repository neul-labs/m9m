package plugins

import (
	"testing"

	"github.com/neul-labs/m9m/internal/nodes/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPlugin implements the Plugin interface for testing
type mockPlugin struct {
	name        string
	pluginType  PluginType
	description base.NodeDescription
}

func (m *mockPlugin) GetDescription() base.NodeDescription {
	return m.description
}

func (m *mockPlugin) GetType() PluginType {
	return m.pluginType
}

func newMockPlugin(name string, pType PluginType) *mockPlugin {
	return &mockPlugin{
		name:       name,
		pluginType: pType,
		description: base.NodeDescription{
			Name:        name,
			Description: "A test plugin",
			Category:    "test",
		},
	}
}

func TestNewPluginRegistry(t *testing.T) {
	r := NewPluginRegistry()
	require.NotNil(t, r)
	assert.Equal(t, 0, r.Count())
	assert.NotNil(t, r.plugins)
	assert.NotNil(t, r.jsConfig)
	assert.NotNil(t, r.grpcConfig)
	assert.NotNil(t, r.restConfig)
}

func TestPluginRegistry_RegisterPlugin(t *testing.T) {
	r := NewPluginRegistry()
	p := newMockPlugin("test-plugin", PluginTypeJavaScript)

	err := r.RegisterPlugin("test-plugin", p)
	require.NoError(t, err)
	assert.Equal(t, 1, r.Count())
}

func TestPluginRegistry_RegisterPlugin_Duplicate(t *testing.T) {
	r := NewPluginRegistry()
	p := newMockPlugin("test-plugin", PluginTypeJavaScript)

	err := r.RegisterPlugin("test-plugin", p)
	require.NoError(t, err)

	err = r.RegisterPlugin("test-plugin", p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestPluginRegistry_RegisterPlugin_NormalizesName(t *testing.T) {
	r := NewPluginRegistry()

	err := r.RegisterPlugin("My Plugin", newMockPlugin("My Plugin", PluginTypeJavaScript))
	require.NoError(t, err)

	// Should be normalized to lowercase with hyphens
	p, ok := r.GetPlugin("my-plugin")
	assert.True(t, ok)
	assert.NotNil(t, p)
}

func TestPluginRegistry_GetPlugin(t *testing.T) {
	r := NewPluginRegistry()
	p := newMockPlugin("test-plugin", PluginTypeREST)
	r.RegisterPlugin("test-plugin", p)

	found, ok := r.GetPlugin("test-plugin")
	assert.True(t, ok)
	assert.NotNil(t, found)
	assert.Equal(t, PluginTypeREST, found.GetType())
}

func TestPluginRegistry_GetPlugin_NotFound(t *testing.T) {
	r := NewPluginRegistry()
	_, ok := r.GetPlugin("nonexistent")
	assert.False(t, ok)
}

func TestPluginRegistry_ListPlugins(t *testing.T) {
	r := NewPluginRegistry()
	r.RegisterPlugin("alpha", newMockPlugin("alpha", PluginTypeJavaScript))
	r.RegisterPlugin("beta", newMockPlugin("beta", PluginTypeGRPC))
	r.RegisterPlugin("gamma", newMockPlugin("gamma", PluginTypeREST))

	names := r.ListPlugins()
	assert.Len(t, names, 3)
	assert.Contains(t, names, "alpha")
	assert.Contains(t, names, "beta")
	assert.Contains(t, names, "gamma")
}

func TestPluginRegistry_Count(t *testing.T) {
	r := NewPluginRegistry()
	assert.Equal(t, 0, r.Count())

	r.RegisterPlugin("p1", newMockPlugin("p1", PluginTypeJavaScript))
	assert.Equal(t, 1, r.Count())

	r.RegisterPlugin("p2", newMockPlugin("p2", PluginTypeGRPC))
	assert.Equal(t, 2, r.Count())
}

func TestPluginRegistry_SetConfigs(t *testing.T) {
	r := NewPluginRegistry()

	jsConfig := &JavaScriptPluginConfig{MaxExecutionTime: 10}
	r.SetJavaScriptConfig(jsConfig)
	assert.Equal(t, jsConfig, r.jsConfig)

	grpcConfig := &GRPCPluginConfig{MaxMessageSize: 1024}
	r.SetGRPCConfig(grpcConfig)
	assert.Equal(t, grpcConfig, r.grpcConfig)

	restConfig := &RESTPluginConfig{DefaultTimeout: 5}
	r.SetRESTConfig(restConfig)
	assert.Equal(t, restConfig, r.restConfig)
}

func TestPluginTypeConstants(t *testing.T) {
	assert.Equal(t, PluginType("javascript"), PluginTypeJavaScript)
	assert.Equal(t, PluginType("grpc"), PluginTypeGRPC)
	assert.Equal(t, PluginType("rest"), PluginTypeREST)
}

func TestNormalizePluginName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"my-plugin", "my-plugin"},
		{"My Plugin", "my-plugin"},
		{"MY_PLUGIN", "my-plugin"},
		{"MixedCase_with Spaces", "mixedcase-with-spaces"},
		{"already-normalized", "already-normalized"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizePluginName(tt.input))
		})
	}
}

func TestPluginRegistry_LoadPluginsFromDirectory_Empty(t *testing.T) {
	r := NewPluginRegistry()
	// Load from a temp empty directory
	dir := t.TempDir()
	err := r.LoadPluginsFromDirectory(dir)
	assert.NoError(t, err)
	assert.Equal(t, 0, r.Count())
}
