package expressions

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/yourusername/n8n-go/internal/model"
)

// WorkflowExecuteMode represents the execution mode of the workflow
type WorkflowExecuteMode string

const (
	ModeCLI        WorkflowExecuteMode = "cli"
	ModeError      WorkflowExecuteMode = "error"
	ModeIntegrated WorkflowExecuteMode = "integrated"
	ModeInternal   WorkflowExecuteMode = "internal"
	ModeManual     WorkflowExecuteMode = "manual"
	ModeRetry      WorkflowExecuteMode = "retry"
	ModeTrigger    WorkflowExecuteMode = "trigger"
	ModeWebhook    WorkflowExecuteMode = "webhook"
	ModeEvaluation WorkflowExecuteMode = "evaluation"
)

// RunExecutionData represents data for a workflow run
type RunExecutionData struct {
	ExecutionData   *ExecutionData
	ResultData      *RunData
	ExecutionMode   WorkflowExecuteMode
	StartedAt       time.Time
	StoppedAt       *time.Time
	WorkflowData    interface{}
}

// ExecutionData represents execution context data
type ExecutionData struct {
	ContextData     map[string]interface{}
	NodeExecutionStack []interface{}
	MetaData        map[string]interface{}
	WaitingExecution map[string]interface{}
}

// RunData represents data from previous node runs
type RunData struct {
	NodeData map[string][]NodeExecutionResult
}

// NodeExecutionResult represents the result of executing a node
type NodeExecutionResult struct {
	Data        []model.DataItem  `json:"data"`
	Error       *ExecutionError   `json:"error,omitempty"`
	StartTime   time.Time         `json:"startTime"`
	ExecutionTime time.Duration   `json:"executionTime"`
	Source      []interface{}     `json:"source,omitempty"`
}

// ExecutionError represents an error during execution
type ExecutionError struct {
	Name        string      `json:"name"`
	Message     string      `json:"message"`
	Description string      `json:"description"`
	Context     interface{} `json:"context"`
	Cause       interface{} `json:"cause"`
	Timestamp   time.Time   `json:"timestamp"`
	NodeName    string      `json:"nodeName"`
	NodeType    string      `json:"nodeType"`
}

// ExecutionContext represents the context for expression evaluation
type ExpressionContext struct {
	Workflow            *model.Workflow
	RunExecutionData    *RunExecutionData
	RunIndex            int
	ItemIndex           int
	ActiveNodeName      string
	ConnectionInputData []model.DataItem
	SiblingParameters   map[string]interface{}
	Mode                WorkflowExecuteMode
	AdditionalKeys      *AdditionalKeys
	ExecuteData         *ExecuteData
	ContextNodeName     *string
}

// AdditionalKeys represents additional context keys
type AdditionalKeys struct {
	ExecutionId    string                 `json:"executionId"`
	CurrentNodeParameters map[string]interface{} `json:"currentNodeParameters"`
	RestApiUrl     string                 `json:"restApiUrl"`
	InstanceBaseUrl string                `json:"instanceBaseUrl"`
	WebhookBaseUrl  string                `json:"webhookBaseUrl"`
	WebhookWaitingBaseUrl string          `json:"webhookWaitingBaseUrl"`
	WebhookTestBaseUrl string             `json:"webhookTestBaseUrl"`
}

// ExecuteData represents additional execution data
type ExecuteData struct {
	Data        interface{}
	Source      interface{}
	Metadata    map[string]interface{}
}

// WorkflowDataProxy provides access to workflow data contexts ($json, $input, etc.)
type WorkflowDataProxy struct {
	// Context
	workflow            *model.Workflow
	runExecutionData    *RunExecutionData
	runIndex            int
	itemIndex           int
	activeNodeName      string
	connectionInputData []model.DataItem
	siblingParameters   map[string]interface{}
	mode                WorkflowExecuteMode

	// Additional context
	additionalKeys      *AdditionalKeys
	executeData         *ExecuteData
	contextNodeName     *string

	// Caching
	dataCache           map[string]interface{}
	cacheMutex          sync.RWMutex

	// JavaScript VM reference
	vm                  *goja.Runtime
}

// NewWorkflowDataProxy creates a new WorkflowDataProxy instance
func NewWorkflowDataProxy(context *ExpressionContext, vm *goja.Runtime) *WorkflowDataProxy {
	return &WorkflowDataProxy{
		workflow:            context.Workflow,
		runExecutionData:    context.RunExecutionData,
		runIndex:           context.RunIndex,
		itemIndex:          context.ItemIndex,
		activeNodeName:     context.ActiveNodeName,
		connectionInputData: context.ConnectionInputData,
		siblingParameters:   context.SiblingParameters,
		mode:               context.Mode,
		additionalKeys:     context.AdditionalKeys,
		executeData:        context.ExecuteData,
		contextNodeName:    context.ContextNodeName,
		dataCache:          make(map[string]interface{}),
		vm:                 vm,
	}
}

// CreateJavaScriptProxy creates a JavaScript proxy object with all n8n context variables
func (p *WorkflowDataProxy) CreateJavaScriptProxy() goja.Value {
	proxy := p.vm.NewObject()

	// Core n8n variables
	proxy.Set("$json", p.createJsonProxy())
	proxy.Set("$input", p.createInputProxy())
	proxy.Set("$node", p.createNodeProxy())
	proxy.Set("$parameter", p.createParameterProxy())
	proxy.Set("$workflow", p.createWorkflowProxy())
	proxy.Set("$execution", p.createExecutionProxy())
	proxy.Set("$env", p.createEnvProxy())
	proxy.Set("$binary", p.createBinaryProxy())
	proxy.Set("$vars", p.createVarsProxy())

	// Shorthand for $node function
	proxy.Set("$", p.createNodeProxy())

	// Legacy support
	proxy.Set("$evaluateExpression", p.createEvaluateExpressionProxy())

	return proxy
}

// createJsonProxy creates the $json context variable
func (p *WorkflowDataProxy) createJsonProxy() goja.Value {
	if p.itemIndex >= len(p.connectionInputData) {
		return goja.Undefined()
	}

	item := p.connectionInputData[p.itemIndex]
	return p.vm.ToValue(item.JSON)
}

// createInputProxy creates the $input context variable
func (p *WorkflowDataProxy) createInputProxy() goja.Value {
	inputProxy := p.vm.NewObject()

	// $input.all() - all items from all connections
	inputProxy.Set("all", p.vm.ToValue(func(call goja.FunctionCall) goja.Value {
		var connectionIndex int = 0
		if len(call.Arguments) > 0 {
			connectionIndex = int(call.Arguments[0].ToInteger())
		}

		inputData := p.getInputConnectionData(connectionIndex)
		jsonData := make([]interface{}, len(inputData))
		for i, item := range inputData {
			jsonData[i] = item.JSON
		}
		return p.vm.ToValue(jsonData)
	}))

	// $input.first() - first item from connection
	inputProxy.Set("first", p.vm.ToValue(func(call goja.FunctionCall) goja.Value {
		var connectionIndex int = 0
		if len(call.Arguments) > 0 {
			connectionIndex = int(call.Arguments[0].ToInteger())
		}

		inputData := p.getInputConnectionData(connectionIndex)
		if len(inputData) > 0 {
			return p.vm.ToValue(inputData[0].JSON)
		}
		return goja.Undefined()
	}))

	// $input.last() - last item from connection
	inputProxy.Set("last", p.vm.ToValue(func(call goja.FunctionCall) goja.Value {
		var connectionIndex int = 0
		if len(call.Arguments) > 0 {
			connectionIndex = int(call.Arguments[0].ToInteger())
		}

		inputData := p.getInputConnectionData(connectionIndex)
		if len(inputData) > 0 {
			return p.vm.ToValue(inputData[len(inputData)-1].JSON)
		}
		return goja.Undefined()
	}))

	// $input.item - specific item by index
	inputProxy.Set("item", p.vm.ToValue(func(call goja.FunctionCall) goja.Value {
		var itemIndex int = p.itemIndex
		var connectionIndex int = 0

		if len(call.Arguments) > 0 {
			itemIndex = int(call.Arguments[0].ToInteger())
		}
		if len(call.Arguments) > 1 {
			connectionIndex = int(call.Arguments[1].ToInteger())
		}

		inputData := p.getInputConnectionData(connectionIndex)
		if itemIndex >= 0 && itemIndex < len(inputData) {
			return p.vm.ToValue(inputData[itemIndex].JSON)
		}
		return goja.Undefined()
	}))

	return inputProxy
}

// createNodeProxy creates the $node context variable and $() function
func (p *WorkflowDataProxy) createNodeProxy() goja.Value {
	return p.vm.ToValue(func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			panic(p.vm.NewTypeError("Node name is required"))
		}

		nodeName := call.Arguments[0].String()

		// Get node execution data
		nodeExecutionData := p.getNodeExecutionData(nodeName)

		nodeProxy := p.vm.NewObject()

		// Current item from node (for current run/item index)
		if len(nodeExecutionData) > 0 && p.itemIndex < len(nodeExecutionData) {
			nodeProxy.Set("json", p.vm.ToValue(nodeExecutionData[p.itemIndex].JSON))
			nodeProxy.Set("binary", p.vm.ToValue(nodeExecutionData[p.itemIndex].Binary))
		} else {
			nodeProxy.Set("json", goja.Undefined())
			nodeProxy.Set("binary", goja.Undefined())
		}

		// Array-like access methods
		nodeProxy.Set("first", p.vm.ToValue(func(call goja.FunctionCall) goja.Value {
			if len(nodeExecutionData) > 0 {
				return p.vm.ToValue(nodeExecutionData[0].JSON)
			}
			return goja.Undefined()
		}))

		nodeProxy.Set("last", p.vm.ToValue(func(call goja.FunctionCall) goja.Value {
			if len(nodeExecutionData) > 0 {
				return p.vm.ToValue(nodeExecutionData[len(nodeExecutionData)-1].JSON)
			}
			return goja.Undefined()
		}))

		nodeProxy.Set("all", p.vm.ToValue(func(call goja.FunctionCall) goja.Value {
			jsonData := make([]interface{}, len(nodeExecutionData))
			for i, item := range nodeExecutionData {
				jsonData[i] = item.JSON
			}
			return p.vm.ToValue(jsonData)
		}))

		nodeProxy.Set("item", p.vm.ToValue(func(call goja.FunctionCall) goja.Value {
			var itemIndex int = 0
			if len(call.Arguments) > 0 {
				itemIndex = int(call.Arguments[0].ToInteger())
			}

			if itemIndex >= 0 && itemIndex < len(nodeExecutionData) {
				return p.vm.ToValue(nodeExecutionData[itemIndex].JSON)
			}
			return goja.Undefined()
		}))

		// Paired item support for data lineage
		nodeProxy.Set("pairedItem", p.vm.ToValue(func(call goja.FunctionCall) goja.Value {
			var itemIndex int = p.itemIndex
			if len(call.Arguments) > 0 {
				itemIndex = int(call.Arguments[0].ToInteger())
			}

			return p.getPairedItemData(nodeName, itemIndex)
		}))

		return nodeProxy
	})
}

// createParameterProxy creates the $parameter context variable
func (p *WorkflowDataProxy) createParameterProxy() goja.Value {
	// Get current node parameters
	currentNode := p.getCurrentNode()
	if currentNode == nil {
		return p.vm.ToValue(map[string]interface{}{})
	}

	// Merge with sibling parameters if available
	parameters := make(map[string]interface{})
	for key, value := range currentNode.Parameters {
		parameters[key] = value
	}

	// Add sibling parameters
	if p.siblingParameters != nil {
		for key, value := range p.siblingParameters {
			parameters[key] = value
		}
	}

	// Add additional keys if available
	if p.additionalKeys != nil && p.additionalKeys.CurrentNodeParameters != nil {
		for key, value := range p.additionalKeys.CurrentNodeParameters {
			parameters[key] = value
		}
	}

	return p.vm.ToValue(parameters)
}

// createWorkflowProxy creates the $workflow context variable
func (p *WorkflowDataProxy) createWorkflowProxy() goja.Value {
	workflowProxy := p.vm.NewObject()

	if p.workflow != nil {
		workflowProxy.Set("id", p.vm.ToValue(p.workflow.ID))
		workflowProxy.Set("name", p.vm.ToValue(p.workflow.Name))
		workflowProxy.Set("active", p.vm.ToValue(p.workflow.Active))
		workflowProxy.Set("versionId", p.vm.ToValue(p.workflow.VersionID))

		if p.workflow.Settings != nil {
			settingsProxy := p.vm.NewObject()
			settingsProxy.Set("timezone", p.vm.ToValue(p.workflow.Settings.Timezone))
			settingsProxy.Set("executionOrder", p.vm.ToValue(p.workflow.Settings.ExecutionOrder))
			workflowProxy.Set("settings", settingsProxy)
		}
	}

	return workflowProxy
}

// createExecutionProxy creates the $execution context variable
func (p *WorkflowDataProxy) createExecutionProxy() goja.Value {
	executionProxy := p.vm.NewObject()

	if p.runExecutionData != nil {
		executionProxy.Set("mode", p.vm.ToValue(string(p.runExecutionData.ExecutionMode)))
		executionProxy.Set("startedAt", p.vm.ToValue(p.runExecutionData.StartedAt.Unix()*1000))

		if p.runExecutionData.StoppedAt != nil {
			executionProxy.Set("stoppedAt", p.vm.ToValue(p.runExecutionData.StoppedAt.Unix()*1000))
		}
	}

	// Add additional execution keys
	if p.additionalKeys != nil {
		executionProxy.Set("id", p.vm.ToValue(p.additionalKeys.ExecutionId))
		executionProxy.Set("restApiUrl", p.vm.ToValue(p.additionalKeys.RestApiUrl))
		executionProxy.Set("instanceBaseUrl", p.vm.ToValue(p.additionalKeys.InstanceBaseUrl))
		executionProxy.Set("webhookBaseUrl", p.vm.ToValue(p.additionalKeys.WebhookBaseUrl))
		executionProxy.Set("webhookWaitingBaseUrl", p.vm.ToValue(p.additionalKeys.WebhookWaitingBaseUrl))
		executionProxy.Set("webhookTestBaseUrl", p.vm.ToValue(p.additionalKeys.WebhookTestBaseUrl))
	}

	return executionProxy
}

// createEnvProxy creates the $env context variable
func (p *WorkflowDataProxy) createEnvProxy() goja.Value {
	// Create a proxy that dynamically reads environment variables
	return p.vm.ToValue(func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			// Return all environment variables
			envMap := make(map[string]string)
			for _, env := range os.Environ() {
				if pair := strings.SplitN(env, "=", 2); len(pair) == 2 {
					envMap[pair[0]] = pair[1]
				}
			}
			return p.vm.ToValue(envMap)
		}

		// Return specific environment variable
		varName := call.Arguments[0].String()
		value := os.Getenv(varName)
		if value == "" {
			return goja.Undefined()
		}
		return p.vm.ToValue(value)
	})
}

// createBinaryProxy creates the $binary context variable
func (p *WorkflowDataProxy) createBinaryProxy() goja.Value {
	if p.itemIndex >= len(p.connectionInputData) {
		return p.vm.ToValue(map[string]interface{}{})
	}

	item := p.connectionInputData[p.itemIndex]
	return p.vm.ToValue(item.Binary)
}

// createVarsProxy creates the $vars context variable (workflow variables)
func (p *WorkflowDataProxy) createVarsProxy() goja.Value {
	// Return static data as workflow variables
	if p.workflow != nil && p.workflow.StaticData != nil {
		return p.vm.ToValue(p.workflow.StaticData)
	}
	return p.vm.ToValue(map[string]interface{}{})
}

// createEvaluateExpressionProxy creates the legacy $evaluateExpression function
func (p *WorkflowDataProxy) createEvaluateExpressionProxy() goja.Value {
	return p.vm.ToValue(func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(p.vm.NewTypeError("$evaluateExpression requires 1 argument"))
		}

		expression := call.Arguments[0].String()

		// Create a new evaluator for nested expression
		evaluator := NewGojaExpressionEvaluator(DefaultEvaluatorConfig())
		context := &ExpressionContext{
			Workflow:            p.workflow,
			RunExecutionData:    p.runExecutionData,
			RunIndex:           p.runIndex,
			ItemIndex:          p.itemIndex,
			ActiveNodeName:     p.activeNodeName,
			ConnectionInputData: p.connectionInputData,
			Mode:               p.mode,
			AdditionalKeys:     p.additionalKeys,
			ExecuteData:        p.executeData,
		}

		result, err := evaluator.EvaluateExpression(expression, context)
		if err != nil {
			panic(p.vm.ToValue(fmt.Sprintf("ExpressionError: %s", err.Error())))
		}

		return p.vm.ToValue(result)
	})
}

// Helper methods

// getInputConnectionData gets input data for a specific connection index
func (p *WorkflowDataProxy) getInputConnectionData(connectionIndex int) []model.DataItem {
	// For now, return the main connection input data
	// In a full implementation, this would handle multiple connection indices
	if connectionIndex == 0 {
		return p.connectionInputData
	}
	return []model.DataItem{}
}

// getNodeExecutionData gets execution data for a specific node
func (p *WorkflowDataProxy) getNodeExecutionData(nodeName string) []model.DataItem {
	p.cacheMutex.RLock()
	if cached, exists := p.dataCache[nodeName]; exists {
		p.cacheMutex.RUnlock()
		return cached.([]model.DataItem)
	}
	p.cacheMutex.RUnlock()

	// Load data from run execution data
	var data []model.DataItem
	if p.runExecutionData != nil && p.runExecutionData.ResultData != nil {
		if nodeResults, exists := p.runExecutionData.ResultData.NodeData[nodeName]; exists {
			if len(nodeResults) > p.runIndex {
				data = nodeResults[p.runIndex].Data
			}
		}
	}

	// Cache the result
	p.cacheMutex.Lock()
	p.dataCache[nodeName] = data
	p.cacheMutex.Unlock()

	return data
}

// getCurrentNode gets the current node being executed
func (p *WorkflowDataProxy) getCurrentNode() *model.Node {
	if p.workflow == nil {
		return nil
	}

	for _, node := range p.workflow.Nodes {
		if node.Name == p.activeNodeName {
			return &node
		}
	}

	return nil
}

// getPairedItemData gets paired item data for lineage tracking
func (p *WorkflowDataProxy) getPairedItemData(nodeName string, itemIndex int) goja.Value {
	// Get the execution data for the node
	nodeData := p.getNodeExecutionData(nodeName)

	if itemIndex >= 0 && itemIndex < len(nodeData) {
		item := nodeData[itemIndex]
		if item.PairedItem != nil {
			return p.vm.ToValue(item.PairedItem)
		}
	}

	return goja.Undefined()
}

// GetCacheKey returns a cache key for this proxy context
func (p *WorkflowDataProxy) GetCacheKey() string {
	return fmt.Sprintf("%s:%d:%d:%s",
		p.activeNodeName, p.runIndex, p.itemIndex, string(p.mode))
}

// Setup configures all n8n context variables in the JavaScript runtime
func (p *WorkflowDataProxy) Setup(vm *goja.Runtime) error {
	// Set up $json variable
	err := vm.Set("$json", p.createJsonProxy())
	if err != nil {
		return fmt.Errorf("failed to set $json: %w", err)
	}

	// Set up $input variable
	err = vm.Set("$input", p.createInputProxy())
	if err != nil {
		return fmt.Errorf("failed to set $input: %w", err)
	}

	// Set up $node variable
	err = vm.Set("$node", p.createNodeProxy())
	if err != nil {
		return fmt.Errorf("failed to set $node: %w", err)
	}

	// Set up $parameter variable
	err = vm.Set("$parameter", p.createParameterProxy())
	if err != nil {
		return fmt.Errorf("failed to set $parameter: %w", err)
	}

	// Set up $workflow variable
	err = vm.Set("$workflow", p.createWorkflowProxy())
	if err != nil {
		return fmt.Errorf("failed to set $workflow: %w", err)
	}

	// Set up $execution variable
	err = vm.Set("$execution", p.createExecutionProxy())
	if err != nil {
		return fmt.Errorf("failed to set $execution: %w", err)
	}

	// Set up $env variable
	err = vm.Set("$env", p.createEnvProxy())
	if err != nil {
		return fmt.Errorf("failed to set $env: %w", err)
	}

	// Set up $binary variable
	err = vm.Set("$binary", p.createBinaryProxy())
	if err != nil {
		return fmt.Errorf("failed to set $binary: %w", err)
	}

	// Set up $vars variable
	err = vm.Set("$vars", p.createVarsProxy())
	if err != nil {
		return fmt.Errorf("failed to set $vars: %w", err)
	}

	// Set up $evaluateExpression function
	err = vm.Set("$evaluateExpression", p.createEvaluateExpressionProxy())
	if err != nil {
		return fmt.Errorf("failed to set $evaluateExpression: %w", err)
	}

	return nil
}

// Reset clears the data cache
func (p *WorkflowDataProxy) Reset() {
	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()
	p.dataCache = make(map[string]interface{})
}