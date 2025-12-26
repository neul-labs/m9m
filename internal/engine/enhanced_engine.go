package engine

import (
	"fmt"
	"time"

	"github.com/dipankar/m9m/internal/expressions"
	"github.com/dipankar/m9m/internal/model"
)

// EnhancedWorkflowEngine extends the base engine with expression evaluation
type EnhancedWorkflowEngine struct {
	*workflowEngineImpl
	expressionEvaluator *expressions.GojaExpressionEvaluator
	executionMode       expressions.WorkflowExecuteMode
}

// NewEnhancedWorkflowEngine creates a new enhanced workflow engine with expression support
func NewEnhancedWorkflowEngine() *EnhancedWorkflowEngine {
	baseEngine := NewWorkflowEngine().(*workflowEngineImpl)

	return &EnhancedWorkflowEngine{
		workflowEngineImpl:  baseEngine,
		expressionEvaluator: expressions.NewGojaExpressionEvaluator(expressions.DefaultEvaluatorConfig()),
		executionMode:       expressions.ModeManual,
	}
}

// SetExecutionMode sets the workflow execution mode
func (e *EnhancedWorkflowEngine) SetExecutionMode(mode expressions.WorkflowExecuteMode) {
	e.executionMode = mode
}

// ExecuteWorkflowWithExpressions executes a workflow with full expression support
func (e *EnhancedWorkflowEngine) ExecuteWorkflowWithExpressions(
	workflow *model.Workflow,
	inputData []model.DataItem,
) (*ExecutionResult, error) {

	if workflow == nil {
		return nil, fmt.Errorf("workflow cannot be nil")
	}

	// Handle empty workflow
	if len(workflow.Nodes) == 0 {
		return &ExecutionResult{Data: inputData}, nil
	}

	// Validate workflow connections
	if err := e.connectionRouter.ValidateConnections(workflow); err != nil {
		return nil, fmt.Errorf("invalid workflow connections: %w", err)
	}

	// Check for cycles
	hasCycles, err := e.connectionRouter.HasCycles(workflow)
	if err != nil {
		return nil, fmt.Errorf("error checking for workflow cycles: %w", err)
	}
	if hasCycles {
		return nil, fmt.Errorf("workflow contains cycles - cannot execute")
	}

	// Resolve workflow credentials
	if e.credentialManager != nil {
		err := e.credentialManager.ResolveWorkflowCredentials(workflow)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve workflow credentials: %w", err)
		}
	}

	// Get execution order
	executionOrder, err := e.connectionRouter.GetExecutionOrder(workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to determine execution order: %w", err)
	}

	// Create execution context
	runExecutionData := &expressions.RunExecutionData{
		ExecutionData: &expressions.ExecutionData{
			ContextData:         make(map[string]interface{}),
			NodeExecutionStack: []interface{}{},
			MetaData:           make(map[string]interface{}),
			WaitingExecution:   make(map[string]interface{}),
		},
		ResultData: &expressions.RunData{
			NodeData: make(map[string][]expressions.NodeExecutionResult),
		},
		ExecutionMode: e.executionMode,
		StartedAt:     time.Now(),
	}

	// Execute nodes in order with expression evaluation
	nodeResults := make(map[string][]model.DataItem)
	startingNodes := e.findStartingNodes(workflow)

	// Set input data for starting nodes
	for _, nodeName := range startingNodes {
		nodeResults[nodeName] = inputData
	}

	// Execute each node in order
	for runIndex, nodeName := range executionOrder {
		node := e.findNodeByName(workflow, nodeName)
		if node == nil {
			return nil, fmt.Errorf("node %s not found in workflow", nodeName)
		}

		// Get executor
		executor, err := e.GetNodeExecutor(node.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to get executor for node %s: %w", node.Name, err)
		}

		// Get input data for this node
		inputDataForNode := nodeResults[nodeName]
		if inputDataForNode == nil {
			inputDataForNode = []model.DataItem{{JSON: make(map[string]interface{})}}
		}

		// Execute node with expression evaluation for each input item
		var allOutputData []model.DataItem

		for itemIndex, inputItem := range inputDataForNode {
			// Create expression context for this item
			context := &expressions.ExpressionContext{
				Workflow:            workflow,
				RunExecutionData:    runExecutionData,
				RunIndex:           runIndex,
				ItemIndex:          itemIndex,
				ActiveNodeName:     node.Name,
				ConnectionInputData: []model.DataItem{inputItem},
				Mode:               e.executionMode,
				AdditionalKeys:     e.createAdditionalKeys(),
			}

			// Resolve expressions in node parameters
			resolvedParams, err := e.resolveNodeParameters(node.Parameters, context)
			if err != nil {
				return &ExecutionResult{
					Error: fmt.Errorf("error resolving expressions in node %s: %w", node.Name, err),
				}, nil
			}

			// Inject credentials if available
			finalParams := resolvedParams
			if e.credentialManager != nil {
				finalParams, err = e.credentialManager.InjectCredentialsIntoNodeParameters(node.ID, resolvedParams)
				if err != nil {
					return &ExecutionResult{
						Error: fmt.Errorf("error injecting credentials for node %s: %w", node.Name, err),
					}, nil
				}
			}

			// Validate resolved parameters
			if err := executor.ValidateParameters(finalParams); err != nil {
				return &ExecutionResult{
					Error: fmt.Errorf("invalid resolved parameters for node %s: %w", node.Name, err),
				}, nil
			}

			// Execute the node with resolved parameters
			outputData, err := executor.Execute([]model.DataItem{inputItem}, finalParams)
			if err != nil {
				return &ExecutionResult{
					Error: fmt.Errorf("error executing node %s: %w", node.Name, err),
				}, nil
			}

			allOutputData = append(allOutputData, outputData...)
		}

		// Store results for this node
		nodeResults[nodeName] = allOutputData

		// Update run execution data
		nodeExecutionResult := expressions.NodeExecutionResult{
			Data:        allOutputData,
			StartTime:   time.Now(),
			ExecutionTime: 0, // This would be calculated properly in a real implementation
		}
		runExecutionData.ResultData.NodeData[nodeName] = []expressions.NodeExecutionResult{nodeExecutionResult}

		// Route data to connected nodes
		routedData, err := e.connectionRouter.RouteData(nodeName, workflow, allOutputData)
		if err != nil {
			return nil, fmt.Errorf("error routing data from node %s: %w", node.Name, err)
		}

		// Update node results with routed data
		for connectedNodeName, data := range routedData {
			if nodeResults[connectedNodeName] == nil {
				nodeResults[connectedNodeName] = []model.DataItem{}
			}
			nodeResults[connectedNodeName] = append(nodeResults[connectedNodeName], data...)
		}
	}

	// Find the final output
	finalNodes := e.findFinalNodes(workflow)
	var finalOutput []model.DataItem

	if len(finalNodes) > 0 {
		for _, finalNodeName := range finalNodes {
			if data, exists := nodeResults[finalNodeName]; exists {
				finalOutput = append(finalOutput, data...)
			}
		}
	} else if len(nodeResults) > 0 {
		// If no clear final nodes, return output from last executed node
		lastNodeName := executionOrder[len(executionOrder)-1]
		finalOutput = nodeResults[lastNodeName]
	}

	// Mark execution as completed
	now := time.Now()
	runExecutionData.StoppedAt = &now

	return &ExecutionResult{Data: finalOutput}, nil
}

// resolveNodeParameters resolves expressions in node parameters
func (e *EnhancedWorkflowEngine) resolveNodeParameters(
	parameters map[string]interface{},
	context *expressions.ExpressionContext,
) (map[string]interface{}, error) {

	resolved := make(map[string]interface{})

	for key, value := range parameters {
		resolvedValue, err := e.expressionEvaluator.ResolveParameterValue(value, context)
		if err != nil {
			return nil, fmt.Errorf("error resolving parameter '%s': %w", key, err)
		}
		resolved[key] = resolvedValue
	}

	return resolved, nil
}

// createAdditionalKeys creates additional context keys for expressions
func (e *EnhancedWorkflowEngine) createAdditionalKeys() *expressions.AdditionalKeys {
	return &expressions.AdditionalKeys{
		ExecutionId:           fmt.Sprintf("exec_%d", time.Now().UnixNano()),
		RestApiUrl:           "http://localhost:5678/api/v1",
		InstanceBaseUrl:      "http://localhost:5678",
		WebhookBaseUrl:       "http://localhost:5678/webhook",
		WebhookWaitingBaseUrl: "http://localhost:5678/webhook-waiting",
		WebhookTestBaseUrl:   "http://localhost:5678/webhook-test",
	}
}

// findNodeByName finds a node by name in the workflow
func (e *EnhancedWorkflowEngine) findNodeByName(workflow *model.Workflow, nodeName string) *model.Node {
	for i := range workflow.Nodes {
		if workflow.Nodes[i].Name == nodeName {
			return &workflow.Nodes[i]
		}
	}
	return nil
}

// findStartingNodes finds nodes with no incoming connections
func (e *EnhancedWorkflowEngine) findStartingNodes(workflow *model.Workflow) []string {
	// This is a simplified implementation
	// In a real implementation, this would analyze the connections
	var startingNodes []string
	nodeHasInput := make(map[string]bool)

	// Mark nodes that have incoming connections
	for _, connections := range workflow.Connections {
		for _, mainConnections := range connections.Main {
			for _, connection := range mainConnections {
				nodeHasInput[connection.Node] = true
			}
		}
	}

	// Find nodes without incoming connections
	for _, node := range workflow.Nodes {
		if !nodeHasInput[node.Name] {
			startingNodes = append(startingNodes, node.Name)
		}
	}

	return startingNodes
}

// findFinalNodes finds nodes with no outgoing connections
func (e *EnhancedWorkflowEngine) findFinalNodes(workflow *model.Workflow) []string {
	// This is a simplified implementation
	var finalNodes []string
	nodeHasOutput := make(map[string]bool)

	// Mark nodes that have outgoing connections
	for nodeName := range workflow.Connections {
		nodeHasOutput[nodeName] = true
	}

	// Find nodes without outgoing connections
	for _, node := range workflow.Nodes {
		if !nodeHasOutput[node.Name] {
			finalNodes = append(finalNodes, node.Name)
		}
	}

	return finalNodes
}

// GetExpressionEvaluator returns the expression evaluator for external use
func (e *EnhancedWorkflowEngine) GetExpressionEvaluator() *expressions.GojaExpressionEvaluator {
	return e.expressionEvaluator
}

// ValidateWorkflowExpressions validates all expressions in a workflow
func (e *EnhancedWorkflowEngine) ValidateWorkflowExpressions(workflow *model.Workflow) []ExpressionValidationError {
	var errors []ExpressionValidationError

	for _, node := range workflow.Nodes {
		nodeErrors := e.validateNodeParameters(node.Name, node.Parameters)
		errors = append(errors, nodeErrors...)
	}

	return errors
}

// ExpressionValidationError represents an expression validation error
type ExpressionValidationError struct {
	NodeName   string `json:"nodeName"`
	Parameter  string `json:"parameter"`
	Expression string `json:"expression"`
	Error      string `json:"error"`
}

// validateNodeParameters validates expressions in node parameters
func (e *EnhancedWorkflowEngine) validateNodeParameters(
	nodeName string,
	parameters map[string]interface{},
) []ExpressionValidationError {
	var errors []ExpressionValidationError

	for paramName, paramValue := range parameters {
		paramErrors := e.validateParameterValue(nodeName, paramName, "", paramValue)
		errors = append(errors, paramErrors...)
	}

	return errors
}

// validateParameterValue recursively validates expressions in parameter values
func (e *EnhancedWorkflowEngine) validateParameterValue(
	nodeName, paramName, path string,
	value interface{},
) []ExpressionValidationError {
	var errors []ExpressionValidationError

	switch v := value.(type) {
	case string:
		if err := e.expressionEvaluator.ValidateExpression(v); err != nil {
			fullPath := paramName
			if path != "" {
				fullPath = paramName + "." + path
			}
			errors = append(errors, ExpressionValidationError{
				NodeName:   nodeName,
				Parameter:  fullPath,
				Expression: v,
				Error:      err.Error(),
			})
		}

	case map[string]interface{}:
		for key, nestedValue := range v {
			nestedPath := key
			if path != "" {
				nestedPath = path + "." + key
			}
			nestedErrors := e.validateParameterValue(nodeName, paramName, nestedPath, nestedValue)
			errors = append(errors, nestedErrors...)
		}

	case []interface{}:
		for i, nestedValue := range v {
			nestedPath := fmt.Sprintf("[%d]", i)
			if path != "" {
				nestedPath = path + nestedPath
			}
			nestedErrors := e.validateParameterValue(nodeName, paramName, nestedPath, nestedValue)
			errors = append(errors, nestedErrors...)
		}
	}

	return errors
}

// GetExpressionStats returns statistics about expression evaluation
func (e *EnhancedWorkflowEngine) GetExpressionStats() map[string]interface{} {
	return e.expressionEvaluator.GetStats()
}

// GetFunctionHelp returns help for all available expression functions
func (e *EnhancedWorkflowEngine) GetFunctionHelp() map[string]*expressions.FunctionHelp {
	return e.expressionEvaluator.GetFunctionHelp()
}