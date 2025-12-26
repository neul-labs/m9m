package plugins

import (
	"github.com/dipankar/m9m/internal/model"
	"github.com/dipankar/m9m/internal/nodes/base"
)

// RESTNodeWrapper wraps a REST plugin to implement the NodeExecutor interface
type RESTNodeWrapper struct {
	*base.BaseNode
	plugin *RESTNodePlugin
}

// NewRESTNodeWrapper creates a new wrapper for a REST plugin
func NewRESTNodeWrapper(plugin *RESTNodePlugin) *RESTNodeWrapper {
	return &RESTNodeWrapper{
		BaseNode: base.NewBaseNode(plugin.Description),
		plugin:   plugin,
	}
}

// Execute implements NodeExecutor.Execute
func (w *RESTNodeWrapper) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	return w.plugin.Execute(inputData, nodeParams)
}

// Description implements NodeExecutor.Description
func (w *RESTNodeWrapper) Description() base.NodeDescription {
	return w.plugin.Description
}

// ValidateParameters implements NodeExecutor.ValidateParameters
func (w *RESTNodeWrapper) ValidateParameters(params map[string]interface{}) error {
	return w.plugin.ValidateParameters(params)
}
