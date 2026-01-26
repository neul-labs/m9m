package plugins

import (
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// JavaScriptNodeWrapper wraps a JavaScript plugin to implement the NodeExecutor interface
type JavaScriptNodeWrapper struct {
	*base.BaseNode
	plugin *JavaScriptNodePlugin
}

// NewJavaScriptNodeWrapper creates a new wrapper for a JavaScript plugin
func NewJavaScriptNodeWrapper(plugin *JavaScriptNodePlugin) *JavaScriptNodeWrapper {
	return &JavaScriptNodeWrapper{
		BaseNode: base.NewBaseNode(plugin.Description),
		plugin:   plugin,
	}
}

// Execute implements NodeExecutor.Execute
func (w *JavaScriptNodeWrapper) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	return w.plugin.Execute(inputData, nodeParams)
}

// Description implements NodeExecutor.Description
func (w *JavaScriptNodeWrapper) Description() base.NodeDescription {
	return w.plugin.Description
}

// ValidateParameters implements NodeExecutor.ValidateParameters
func (w *JavaScriptNodeWrapper) ValidateParameters(params map[string]interface{}) error {
	return w.plugin.ValidateParameters(params)
}
