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
	r.mu.RLock()
	defer r.mu.RUnlock()

	for name, plugin := range r.plugins {
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
