package plugins

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dipankar/n8n-go/internal/engine"
	"github.com/dipankar/n8n-go/internal/nodes/base"
)

// PluginType represents the type of plugin
type PluginType string

const (
	PluginTypeJavaScript PluginType = "javascript"
	PluginTypeGRPC       PluginType = "grpc"
	PluginTypeREST       PluginType = "rest"
)

// Plugin represents a generic plugin interface
type Plugin interface {
	GetDescription() base.NodeDescription
	GetType() PluginType
}

// PluginRegistry manages all loaded plugins
type PluginRegistry struct {
	plugins          map[string]Plugin
	pluginDir        string                  // Directory where plugins are loaded from
	engine           engine.WorkflowEngine   // Reference to workflow engine for re-registration
	jsConfig         *JavaScriptPluginConfig
	grpcConfig       *GRPCPluginConfig
	restConfig       *RESTPluginConfig
	mu               sync.RWMutex
}

// NewPluginRegistry creates a new plugin registry
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins:    make(map[string]Plugin),
		jsConfig:   DefaultJavaScriptPluginConfig(),
		grpcConfig: DefaultGRPCPluginConfig(),
		restConfig: DefaultRESTPluginConfig(),
	}
}

// SetJavaScriptConfig sets the configuration for JavaScript plugins
func (r *PluginRegistry) SetJavaScriptConfig(config *JavaScriptPluginConfig) {
	r.jsConfig = config
}

// SetGRPCConfig sets the configuration for gRPC plugins
func (r *PluginRegistry) SetGRPCConfig(config *GRPCPluginConfig) {
	r.grpcConfig = config
}

// SetRESTConfig sets the configuration for REST plugins
func (r *PluginRegistry) SetRESTConfig(config *RESTPluginConfig) {
	r.restConfig = config
}

// LoadPluginsFromDirectory scans a directory and loads all plugins
func (r *PluginRegistry) LoadPluginsFromDirectory(dir string) error {
	r.mu.Lock()
	r.pluginDir = dir  // Save directory for hot-reload
	r.mu.Unlock()

	log.Printf("Scanning for plugins in: %s", dir)

	// Find all plugin files
	jsFiles, err := filepath.Glob(filepath.Join(dir, "*.js"))
	if err != nil {
		return fmt.Errorf("failed to scan for JavaScript plugins: %w", err)
	}

	grpcFiles, err := filepath.Glob(filepath.Join(dir, "*.grpc.yaml"))
	if err != nil {
		return fmt.Errorf("failed to scan for gRPC plugins: %w", err)
	}

	restFiles, err := filepath.Glob(filepath.Join(dir, "*.rest.yaml"))
	if err != nil {
		return fmt.Errorf("failed to scan for REST plugins: %w", err)
	}

	// Load JavaScript plugins
	for _, file := range jsFiles {
		if err := r.LoadJavaScriptPlugin(file); err != nil {
			log.Printf("Warning: Failed to load JavaScript plugin %s: %v", file, err)
			continue
		}
		log.Printf("✓ Loaded JavaScript plugin: %s", file)
	}

	// Load gRPC plugins
	for _, file := range grpcFiles {
		if err := r.LoadGRPCPlugin(file); err != nil {
			log.Printf("Warning: Failed to load gRPC plugin %s: %v", file, err)
			continue
		}
		log.Printf("✓ Loaded gRPC plugin: %s", file)
	}

	// Load REST API plugins
	for _, file := range restFiles {
		if err := r.LoadRESTPlugin(file); err != nil {
			log.Printf("Warning: Failed to load REST plugin %s: %v", file, err)
			continue
		}
		log.Printf("✓ Loaded REST plugin: %s", file)
	}

	log.Printf("Plugin loading complete. Total plugins: %d", r.Count())

	return nil
}

// LoadJavaScriptPlugin loads a JavaScript plugin from a file
func (r *PluginRegistry) LoadJavaScriptPlugin(filePath string) error {
	plugin, err := LoadJavaScriptPlugin(filePath, r.jsConfig)
	if err != nil {
		return err
	}

	return r.RegisterPlugin(plugin.Name, plugin)
}

// LoadGRPCPlugin loads a gRPC plugin from a configuration file
func (r *PluginRegistry) LoadGRPCPlugin(filePath string) error {
	plugin, err := LoadGRPCPlugin(filePath, r.grpcConfig)
	if err != nil {
		return err
	}

	return r.RegisterPlugin(plugin.Name, plugin)
}

// LoadRESTPlugin loads a REST API plugin from a configuration file
func (r *PluginRegistry) LoadRESTPlugin(filePath string) error {
	plugin, err := LoadRESTPlugin(filePath, r.restConfig)
	if err != nil {
		return err
	}

	return r.RegisterPlugin(plugin.Name, plugin)
}

// RegisterPlugin registers a plugin with the registry
func (r *PluginRegistry) RegisterPlugin(name string, plugin Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Normalize name
	name = normalizePluginName(name)

	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin with name %s already registered", name)
	}

	r.plugins[name] = plugin
	return nil
}

// GetPlugin retrieves a plugin by name
func (r *PluginRegistry) GetPlugin(name string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.plugins[name]
	return plugin, exists
}

// Count returns the number of registered plugins
func (r *PluginRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.plugins)
}

// ListPlugins returns a list of all plugin names
func (r *PluginRegistry) ListPlugins() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}

	return names
}

// RegisterWithEngine registers all plugins with the workflow engine
func (r *PluginRegistry) RegisterWithEngine(eng engine.WorkflowEngine) error {
	r.mu.Lock()
	r.engine = eng  // Save engine reference for hot-reload
	plugins := make(map[string]Plugin)
	for k, v := range r.plugins {
		plugins[k] = v
	}
	r.mu.Unlock()

	for name, plugin := range plugins {
		var executor base.NodeExecutor

		switch p := plugin.(type) {
		case *JavaScriptNodePlugin:
			executor = NewJavaScriptNodeWrapper(p)
		case *GRPCNodePlugin:
			executor = NewGRPCNodeWrapper(p)
		case *RESTNodePlugin:
			executor = NewRESTNodeWrapper(p)
		default:
			log.Printf("Warning: Unknown plugin type for %s", name)
			continue
		}

		// Register with engine using n8n-nodes-base prefix
		nodeName := fmt.Sprintf("n8n-nodes-base.%s", name)
		eng.RegisterNodeExecutor(nodeName, executor)

		log.Printf("Registered plugin node: %s", nodeName)
	}

	return nil
}

// ReloadPlugin reloads a specific plugin (for hot reload)
func (r *PluginRegistry) ReloadPlugin(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	plugin, exists := r.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Get the file path based on plugin type
	var filePath string
	switch p := plugin.(type) {
	case *JavaScriptNodePlugin:
		filePath = p.FilePath
	case *GRPCNodePlugin:
		filePath = p.ConfigPath
	case *RESTNodePlugin:
		filePath = p.ConfigPath
	default:
		return fmt.Errorf("unknown plugin type")
	}

	// Remove old plugin
	delete(r.plugins, name)

	// Reload based on file extension
	if strings.HasSuffix(filePath, ".js") {
		return r.LoadJavaScriptPlugin(filePath)
	} else if strings.HasSuffix(filePath, ".grpc.yaml") {
		return r.LoadGRPCPlugin(filePath)
	} else if strings.HasSuffix(filePath, ".rest.yaml") {
		return r.LoadRESTPlugin(filePath)
	}

	return fmt.Errorf("unknown plugin file type: %s", filePath)
}

// ReloadAllPlugins reloads all plugins from the plugin directory
func (r *PluginRegistry) ReloadAllPlugins() error {
	r.mu.RLock()
	pluginDir := r.pluginDir
	eng := r.engine
	r.mu.RUnlock()

	if pluginDir == "" {
		return fmt.Errorf("no plugin directory configured")
	}

	if eng == nil {
		return fmt.Errorf("no workflow engine registered")
	}

	log.Printf("Reloading all plugins from: %s", pluginDir)

	// Clear old plugins (close gRPC connections if needed)
	r.mu.Lock()
	for name, plugin := range r.plugins {
		if grpcPlugin, ok := plugin.(*GRPCNodePlugin); ok {
			if err := grpcPlugin.Close(); err != nil {
				log.Printf("Warning: Failed to close gRPC plugin %s: %v", name, err)
			}
		}
	}
	r.plugins = make(map[string]Plugin)
	r.mu.Unlock()

	// Reload all plugins from directory
	if err := r.LoadPluginsFromDirectory(pluginDir); err != nil {
		return fmt.Errorf("failed to reload plugins: %w", err)
	}

	// Re-register all plugins with engine
	if err := r.RegisterWithEngine(eng); err != nil {
		return fmt.Errorf("failed to register reloaded plugins: %w", err)
	}

	log.Printf("✅ Successfully reloaded %d plugins", r.Count())
	return nil
}

// GetPluginDirectory returns the current plugin directory
func (r *PluginRegistry) GetPluginDirectory() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.pluginDir
}

// GetStats returns statistics about loaded plugins
func (r *PluginRegistry) GetStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := map[string]interface{}{
		"total":      len(r.plugins),
		"javascript": 0,
		"grpc":       0,
		"rest":       0,
		"directory":  r.pluginDir,
	}

	for _, plugin := range r.plugins {
		switch plugin.GetType() {
		case PluginTypeJavaScript:
			stats["javascript"] = stats["javascript"].(int) + 1
		case PluginTypeGRPC:
			stats["grpc"] = stats["grpc"].(int) + 1
		case PluginTypeREST:
			stats["rest"] = stats["rest"].(int) + 1
		}
	}

	return stats
}

// normalizePluginName normalizes a plugin name to a standard format
func normalizePluginName(name string) string {
	// Remove spaces and convert to lowercase
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")

	return name
}

// GetType returns the plugin type
func (p *JavaScriptNodePlugin) GetType() PluginType {
	return PluginTypeJavaScript
}
