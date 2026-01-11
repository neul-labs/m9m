package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dipankar/m9m/internal/engine"
	"github.com/dipankar/m9m/internal/mcp"
	"github.com/dipankar/m9m/internal/plugins"
)

// PluginCreateJSTool creates a JavaScript plugin
type PluginCreateJSTool struct {
	*BaseTool
	registry  *plugins.PluginRegistry
	pluginDir string
	engine    engine.WorkflowEngine
}

// NewPluginCreateJSTool creates a new JavaScript plugin creation tool
func NewPluginCreateJSTool(registry *plugins.PluginRegistry, pluginDir string, eng engine.WorkflowEngine) *PluginCreateJSTool {
	return &PluginCreateJSTool{
		BaseTool: NewBaseTool(
			"plugin_create_js",
			"Create a custom JavaScript node using the Goja runtime. The node will be immediately available for use in workflows.",
			ObjectSchema(map[string]interface{}{
				"name":        StringProp("Node name (will become n8n-nodes-base.<name>)"),
				"description": StringProp("Description of what the node does"),
				"category":    StringEnumProp("Node category", []string{"transform", "trigger", "action", "integration"}),
				"code": StringProp(`JavaScript code with module.exports = { description, execute, validateParameters }.
Example:
module.exports = {
  description: { name: 'My Node', description: 'Does something', category: 'transform' },
  execute: function(inputData, nodeParams) {
    return inputData.map(item => ({ json: { ...item.json, processed: true } }));
  }
};`),
				"parameters": ArrayProp("Parameter definitions", map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name":     map[string]interface{}{"type": "string"},
						"type":     map[string]interface{}{"type": "string"},
						"required": map[string]interface{}{"type": "boolean"},
						"default":  map[string]interface{}{},
					},
				}),
			}, []string{"name", "code"}),
		),
		registry:  registry,
		pluginDir: pluginDir,
		engine:    eng,
	}
}

// Execute creates a JavaScript plugin
func (t *PluginCreateJSTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	name := GetString(args, "name")
	description := GetString(args, "description")
	category := GetStringOr(args, "category", "transform")
	code := GetString(args, "code")

	// Validate name
	if name == "" {
		return mcp.ErrorContent("Plugin name is required"), nil
	}

	// Sanitize name
	safeName := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	safeName = strings.ReplaceAll(safeName, "_", "-")

	// If no code provided, generate template
	if code == "" {
		code = fmt.Sprintf(`// Plugin: %s
// Description: %s
// Category: %s

module.exports = {
  description: {
    name: '%s',
    description: '%s',
    category: '%s'
  },

  execute: function(inputData, nodeParams) {
    // Process input data
    return inputData.map(function(item) {
      return {
        json: {
          ...item.json,
          processed: true,
          processedAt: new Date().toISOString()
        }
      };
    });
  },

  validateParameters: function(params) {
    // Optional parameter validation
    return null; // Return error string if validation fails
  }
};
`, name, description, category, name, description, category)
	}

	// Ensure plugin directory exists
	if t.pluginDir != "" {
		if err := os.MkdirAll(t.pluginDir, 0755); err != nil {
			return mcp.ErrorContent(fmt.Sprintf("Failed to create plugin directory: %v", err)), nil
		}

		// Write plugin file
		filename := filepath.Join(t.pluginDir, safeName+".js")
		if err := os.WriteFile(filename, []byte(code), 0644); err != nil {
			return mcp.ErrorContent(fmt.Sprintf("Failed to write plugin file: %v", err)), nil
		}

		// If registry exists, try to reload
		if t.registry != nil && t.engine != nil {
			if err := t.registry.ReloadAllPlugins(); err != nil {
				// Non-fatal, plugin is saved but not loaded
				return mcp.SuccessJSON(map[string]interface{}{
					"success":  true,
					"name":     safeName,
					"nodeType": "n8n-nodes-base." + safeName,
					"filename": filename,
					"warning":  fmt.Sprintf("Plugin saved but hot-reload failed: %v", err),
					"message":  "Plugin created. Restart server to load.",
				}), nil
			}
		}

		return mcp.SuccessJSON(map[string]interface{}{
			"success":  true,
			"name":     safeName,
			"nodeType": "n8n-nodes-base." + safeName,
			"filename": filename,
			"message":  fmt.Sprintf("Plugin '%s' created and loaded. Use it in workflows as 'n8n-nodes-base.%s'", name, safeName),
		}), nil
	}

	// No plugin directory - return code only
	return mcp.SuccessJSON(map[string]interface{}{
		"success":  true,
		"name":     safeName,
		"nodeType": "n8n-nodes-base." + safeName,
		"code":     code,
		"message":  "Plugin code generated. Save to plugins directory to use.",
	}), nil
}

// PluginCreateRESTTool creates a REST API wrapper plugin
type PluginCreateRESTTool struct {
	*BaseTool
	pluginDir string
}

// NewPluginCreateRESTTool creates a new REST plugin creation tool
func NewPluginCreateRESTTool(pluginDir string) *PluginCreateRESTTool {
	return &PluginCreateRESTTool{
		BaseTool: NewBaseTool(
			"plugin_create_rest",
			"Create a node that wraps a REST API endpoint. The node will be available as a reusable integration.",
			ObjectSchema(map[string]interface{}{
				"name":        StringProp("Node name"),
				"description": StringProp("Description of the API"),
				"endpoint":    StringProp("Base URL of the API"),
				"method":      StringEnumProp("HTTP method", []string{"GET", "POST", "PUT", "DELETE", "PATCH"}),
				"headers":     ObjectProp("Default headers to include"),
				"timeout":     StringPropWithDefault("Request timeout", "30s"),
				"authType":    StringEnumProp("Authentication type", []string{"none", "bearer", "basic", "apiKey"}),
			}, []string{"name", "endpoint"}),
		),
		pluginDir: pluginDir,
	}
}

// Execute creates a REST plugin
func (t *PluginCreateRESTTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	name := GetString(args, "name")
	description := GetString(args, "description")
	endpoint := GetString(args, "endpoint")
	method := GetStringOr(args, "method", "GET")
	timeout := GetStringOr(args, "timeout", "30s")
	authType := GetStringOr(args, "authType", "none")
	headers := GetMap(args, "headers")

	// Sanitize name
	safeName := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	safeName = strings.ReplaceAll(safeName, "_", "-")

	// Build YAML config
	yaml := fmt.Sprintf(`# REST API Plugin: %s
name: %s
description: %s
category: integration
endpoint: %s
method: %s
timeout: %s
auth_type: %s
`, name, safeName, description, endpoint, method, timeout, authType)

	if headers != nil {
		yaml += "headers:\n"
		for k, v := range headers {
			if s, ok := v.(string); ok {
				yaml += fmt.Sprintf("  %s: %s\n", k, s)
			}
		}
	}

	// Save if plugin directory exists
	if t.pluginDir != "" {
		if err := os.MkdirAll(t.pluginDir, 0755); err != nil {
			return mcp.ErrorContent(fmt.Sprintf("Failed to create plugin directory: %v", err)), nil
		}

		filename := filepath.Join(t.pluginDir, safeName+".rest.yaml")
		if err := os.WriteFile(filename, []byte(yaml), 0644); err != nil {
			return mcp.ErrorContent(fmt.Sprintf("Failed to write plugin file: %v", err)), nil
		}

		return mcp.SuccessJSON(map[string]interface{}{
			"success":  true,
			"name":     safeName,
			"nodeType": "n8n-nodes-base." + safeName,
			"filename": filename,
			"config":   yaml,
			"message":  fmt.Sprintf("REST plugin '%s' created. Restart server to load.", name),
		}), nil
	}

	return mcp.SuccessJSON(map[string]interface{}{
		"success":  true,
		"name":     safeName,
		"nodeType": "n8n-nodes-base." + safeName,
		"config":   yaml,
		"message":  "REST plugin config generated. Save as .rest.yaml in plugins directory.",
	}), nil
}

// PluginListTool lists all plugins
type PluginListTool struct {
	*BaseTool
	registry *plugins.PluginRegistry
}

// NewPluginListTool creates a new plugin list tool
func NewPluginListTool(registry *plugins.PluginRegistry) *PluginListTool {
	return &PluginListTool{
		BaseTool: NewBaseTool(
			"plugin_list",
			"List all installed custom plugins with their types and status.",
			ObjectSchema(map[string]interface{}{
				"type": StringEnumProp("Filter by plugin type", []string{"javascript", "grpc", "rest"}),
			}, nil),
		),
		registry: registry,
	}
}

// Execute lists plugins
func (t *PluginListTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if t.registry == nil {
		return mcp.ErrorContent("Plugin registry not initialized"), nil
	}

	// ListPlugins returns []string (plugin names)
	pluginNames := t.registry.ListPlugins()

	// Build response with plugin details
	pluginInfos := make([]map[string]interface{}, 0, len(pluginNames))
	for _, name := range pluginNames {
		plugin, exists := t.registry.GetPlugin(name)
		if !exists {
			continue
		}
		desc := plugin.GetDescription()
		pluginInfos = append(pluginInfos, map[string]interface{}{
			"name":        desc.Name,
			"nodeType":    "n8n-nodes-base." + strings.ToLower(desc.Name),
			"description": desc.Description,
			"category":    desc.Category,
		})
	}

	stats := t.registry.GetStats()

	return mcp.SuccessJSON(map[string]interface{}{
		"plugins": pluginInfos,
		"count":   len(pluginInfos),
		"stats":   stats,
	}), nil
}

// PluginGetTool gets plugin details
type PluginGetTool struct {
	*BaseTool
	registry  *plugins.PluginRegistry
	pluginDir string
}

// NewPluginGetTool creates a new plugin get tool
func NewPluginGetTool(registry *plugins.PluginRegistry, pluginDir string) *PluginGetTool {
	return &PluginGetTool{
		BaseTool: NewBaseTool(
			"plugin_get",
			"Get details about a specific plugin including its source code.",
			ObjectSchema(map[string]interface{}{
				"name": StringProp("Plugin name to get"),
			}, []string{"name"}),
		),
		registry:  registry,
		pluginDir: pluginDir,
	}
}

// Execute gets plugin details
func (t *PluginGetTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	name := GetString(args, "name")

	// Try to find plugin
	var pluginInfo map[string]interface{}

	if t.registry != nil {
		plugin, exists := t.registry.GetPlugin(name)
		if exists && plugin != nil {
			desc := plugin.GetDescription()
			pluginInfo = map[string]interface{}{
				"name":        desc.Name,
				"nodeType":    "n8n-nodes-base." + strings.ToLower(desc.Name),
				"description": desc.Description,
				"category":    desc.Category,
				"status":      "loaded",
			}
		}
	}

	if pluginInfo == nil {
		return mcp.ErrorContent(fmt.Sprintf("Plugin not found: %s", name)), nil
	}

	// Try to read source code
	if t.pluginDir != "" {
		safeName := strings.ToLower(strings.ReplaceAll(name, " ", "-"))

		// Try JavaScript
		jsPath := filepath.Join(t.pluginDir, safeName+".js")
		if data, err := os.ReadFile(jsPath); err == nil {
			pluginInfo["type"] = "javascript"
			pluginInfo["sourceCode"] = string(data)
			pluginInfo["filePath"] = jsPath
		}

		// Try REST YAML
		restPath := filepath.Join(t.pluginDir, safeName+".rest.yaml")
		if data, err := os.ReadFile(restPath); err == nil {
			pluginInfo["type"] = "rest"
			pluginInfo["config"] = string(data)
			pluginInfo["filePath"] = restPath
		}
	}

	return mcp.SuccessJSON(pluginInfo), nil
}

// PluginReloadTool reloads plugins
type PluginReloadTool struct {
	*BaseTool
	registry *plugins.PluginRegistry
	engine   engine.WorkflowEngine
}

// NewPluginReloadTool creates a new plugin reload tool
func NewPluginReloadTool(registry *plugins.PluginRegistry, eng engine.WorkflowEngine) *PluginReloadTool {
	return &PluginReloadTool{
		BaseTool: NewBaseTool(
			"plugin_reload",
			"Hot-reload all plugins without restarting the server.",
			ObjectSchema(map[string]interface{}{}, nil),
		),
		registry: registry,
		engine:   eng,
	}
}

// Execute reloads plugins
func (t *PluginReloadTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	if t.registry == nil {
		return mcp.ErrorContent("Plugin registry not initialized"), nil
	}

	startTime := time.Now()

	if err := t.registry.ReloadAllPlugins(); err != nil {
		return mcp.ErrorContent(fmt.Sprintf("Failed to reload plugins: %v", err)), nil
	}

	duration := time.Since(startTime)
	stats := t.registry.GetStats()

	return mcp.SuccessJSON(map[string]interface{}{
		"success":  true,
		"message":  "Plugins reloaded successfully",
		"duration": duration.String(),
		"stats":    stats,
	}), nil
}

// PluginDeleteTool deletes a plugin
type PluginDeleteTool struct {
	*BaseTool
	registry  *plugins.PluginRegistry
	pluginDir string
	engine    engine.WorkflowEngine
}

// NewPluginDeleteTool creates a new plugin delete tool
func NewPluginDeleteTool(registry *plugins.PluginRegistry, pluginDir string, eng engine.WorkflowEngine) *PluginDeleteTool {
	return &PluginDeleteTool{
		BaseTool: NewBaseTool(
			"plugin_delete",
			"Delete a custom plugin. The plugin file will be removed and it will no longer be available.",
			ObjectSchema(map[string]interface{}{
				"name": StringProp("Plugin name to delete"),
			}, []string{"name"}),
		),
		registry:  registry,
		pluginDir: pluginDir,
		engine:    eng,
	}
}

// Execute deletes a plugin
func (t *PluginDeleteTool) Execute(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	name := GetString(args, "name")

	if t.pluginDir == "" {
		return mcp.ErrorContent("Plugin directory not configured"), nil
	}

	safeName := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	var deleted []string

	// Try to delete JavaScript plugin
	jsPath := filepath.Join(t.pluginDir, safeName+".js")
	if _, err := os.Stat(jsPath); err == nil {
		if err := os.Remove(jsPath); err != nil {
			return mcp.ErrorContent(fmt.Sprintf("Failed to delete plugin file: %v", err)), nil
		}
		deleted = append(deleted, jsPath)
	}

	// Try to delete REST plugin
	restPath := filepath.Join(t.pluginDir, safeName+".rest.yaml")
	if _, err := os.Stat(restPath); err == nil {
		if err := os.Remove(restPath); err != nil {
			return mcp.ErrorContent(fmt.Sprintf("Failed to delete plugin file: %v", err)), nil
		}
		deleted = append(deleted, restPath)
	}

	if len(deleted) == 0 {
		return mcp.ErrorContent(fmt.Sprintf("No plugin files found for: %s", name)), nil
	}

	// Reload to unregister the plugin
	if t.registry != nil {
		t.registry.ReloadAllPlugins()
	}

	return mcp.SuccessJSON(map[string]interface{}{
		"success":      true,
		"name":         name,
		"deletedFiles": deleted,
		"message":      fmt.Sprintf("Plugin '%s' deleted", name),
	}), nil
}

// RegisterPluginTools registers all plugin management tools with a registry
func RegisterPluginTools(registry *Registry, pluginRegistry *plugins.PluginRegistry, pluginDir string, eng engine.WorkflowEngine) {
	registry.Register(NewPluginCreateJSTool(pluginRegistry, pluginDir, eng))
	registry.Register(NewPluginCreateRESTTool(pluginDir))
	registry.Register(NewPluginListTool(pluginRegistry))
	registry.Register(NewPluginGetTool(pluginRegistry, pluginDir))
	registry.Register(NewPluginReloadTool(pluginRegistry, eng))
	registry.Register(NewPluginDeleteTool(pluginRegistry, pluginDir, eng))
}
