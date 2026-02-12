package m9m

import (
	"github.com/neul-labs/m9m/internal/connections"
	"github.com/neul-labs/m9m/internal/credentials"
	"github.com/neul-labs/m9m/internal/engine"
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// Engine wraps the internal workflow engine with a public API.
type Engine struct {
	internal engine.WorkflowEngine
}

// New creates a new workflow engine with default settings.
func New() *Engine {
	return &Engine{
		internal: engine.NewWorkflowEngine(),
	}
}

// NewWithOptions creates a new workflow engine with custom options.
func NewWithOptions(opts ...Option) *Engine {
	e := New()
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Option configures an Engine.
type Option func(*Engine)

// WithCredentialManager sets the credential manager for the engine.
func WithCredentialManager(cm *CredentialManager) Option {
	return func(e *Engine) {
		if cm != nil && cm.internal != nil {
			e.internal.SetCredentialManager(cm.internal)
		}
	}
}

// Execute runs a workflow with the given input data.
// If inputData is nil, an empty data item is used as input.
func (e *Engine) Execute(workflow *Workflow, inputData []DataItem) (*ExecutionResult, error) {
	if workflow == nil {
		return nil, ErrNilWorkflow
	}

	// Convert public types to internal types
	internalWorkflow := workflow.toInternal()
	internalInput := dataItemsToInternal(inputData)

	// Execute the workflow
	result, err := e.internal.ExecuteWorkflow(internalWorkflow, internalInput)
	if err != nil {
		return nil, err
	}

	// Convert result back to public types
	return executionResultFromInternal(result), nil
}

// ExecuteParallel runs multiple workflows in parallel and returns all results.
func (e *Engine) ExecuteParallel(workflows []*Workflow, inputData [][]DataItem) ([]*ExecutionResult, error) {
	if len(workflows) == 0 {
		return []*ExecutionResult{}, nil
	}

	// Convert public types to internal types
	internalWorkflows := make([]*model.Workflow, len(workflows))
	internalInputs := make([][]model.DataItem, len(workflows))

	for i, w := range workflows {
		if w == nil {
			return nil, ErrNilWorkflow
		}
		internalWorkflows[i] = w.toInternal()
		if i < len(inputData) {
			internalInputs[i] = dataItemsToInternal(inputData[i])
		} else {
			internalInputs[i] = []model.DataItem{}
		}
	}

	// Execute workflows in parallel
	results, err := e.internal.ExecuteWorkflowParallel(internalWorkflows, internalInputs)
	if err != nil {
		return nil, err
	}

	// Convert results back to public types
	publicResults := make([]*ExecutionResult, len(results))
	for i, r := range results {
		publicResults[i] = executionResultFromInternal(r)
	}

	return publicResults, nil
}

// RegisterNode registers a node executor for a given node type.
// The nodeType should follow the n8n convention (e.g., "n8n-nodes-base.httpRequest").
func (e *Engine) RegisterNode(nodeType string, executor NodeExecutor) {
	e.internal.RegisterNodeExecutor(nodeType, &nodeExecutorAdapter{executor})
}

// GetNode retrieves a registered node executor by type.
func (e *Engine) GetNode(nodeType string) (NodeExecutor, error) {
	internal, err := e.internal.GetNodeExecutor(nodeType)
	if err != nil {
		return nil, err
	}

	// Check if this is an adapter wrapping a public NodeExecutor
	if adapter, ok := internal.(*nodeExecutorAdapter); ok {
		return adapter.public, nil
	}

	// Wrap internal executor
	return &internalNodeAdapter{internal}, nil
}

// SetCredentialManager sets the credential manager for the engine.
func (e *Engine) SetCredentialManager(cm *CredentialManager) {
	if cm != nil && cm.internal != nil {
		e.internal.SetCredentialManager(cm.internal)
	}
}

// SetConnectionRouter sets a custom connection router for the engine.
func (e *Engine) SetConnectionRouter(router connections.ConnectionRouter) {
	e.internal.SetConnectionRouter(router)
}

// nodeExecutorAdapter adapts a public NodeExecutor to the internal interface.
type nodeExecutorAdapter struct {
	public NodeExecutor
}

func (a *nodeExecutorAdapter) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	publicInput := dataItemsFromInternal(inputData)
	result, err := a.public.Execute(publicInput, nodeParams)
	if err != nil {
		return nil, err
	}
	return dataItemsToInternal(result), nil
}

func (a *nodeExecutorAdapter) Description() base.NodeDescription {
	desc := a.public.Description()
	return base.NodeDescription{
		Name:        desc.Name,
		Description: desc.Description,
		Category:    desc.Category,
	}
}

func (a *nodeExecutorAdapter) ValidateParameters(params map[string]interface{}) error {
	return a.public.ValidateParameters(params)
}

// internalNodeAdapter adapts an internal NodeExecutor to the public interface.
type internalNodeAdapter struct {
	internal base.NodeExecutor
}

func (a *internalNodeAdapter) Execute(inputData []DataItem, nodeParams map[string]interface{}) ([]DataItem, error) {
	internalInput := dataItemsToInternal(inputData)
	result, err := a.internal.Execute(internalInput, nodeParams)
	if err != nil {
		return nil, err
	}
	return dataItemsFromInternal(result), nil
}

func (a *internalNodeAdapter) Description() NodeDescription {
	desc := a.internal.Description()
	return NodeDescription{
		Name:        desc.Name,
		Description: desc.Description,
		Category:    desc.Category,
	}
}

func (a *internalNodeAdapter) ValidateParameters(params map[string]interface{}) error {
	return a.internal.ValidateParameters(params)
}

// executionResultFromInternal converts an internal ExecutionResult to a public one.
func executionResultFromInternal(r *engine.ExecutionResult) *ExecutionResult {
	if r == nil {
		return nil
	}
	return &ExecutionResult{
		Data:  dataItemsFromInternal(r.Data),
		Error: r.Error,
	}
}

// NewCredentialManager creates a new credential manager.
func NewCredentialManager() (*CredentialManager, error) {
	internal, err := credentials.NewCredentialManager()
	if err != nil {
		return nil, err
	}
	return &CredentialManager{internal: internal}, nil
}
