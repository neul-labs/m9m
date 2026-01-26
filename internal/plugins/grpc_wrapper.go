package plugins

import (
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// GRPCNodeWrapper wraps a gRPC plugin to implement the NodeExecutor interface
type GRPCNodeWrapper struct {
	*base.BaseNode
	plugin *GRPCNodePlugin
}

// NewGRPCNodeWrapper creates a new wrapper for a gRPC plugin
func NewGRPCNodeWrapper(plugin *GRPCNodePlugin) *GRPCNodeWrapper {
	return &GRPCNodeWrapper{
		BaseNode: base.NewBaseNode(plugin.Description),
		plugin:   plugin,
	}
}

// Execute implements NodeExecutor.Execute
func (w *GRPCNodeWrapper) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	return w.plugin.Execute(inputData, nodeParams)
}

// Description implements NodeExecutor.Description
func (w *GRPCNodeWrapper) Description() base.NodeDescription {
	return w.plugin.Description
}

// ValidateParameters implements NodeExecutor.ValidateParameters
func (w *GRPCNodeWrapper) ValidateParameters(params map[string]interface{}) error {
	return w.plugin.ValidateParameters(params)
}
