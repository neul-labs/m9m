/*
Package engine provides the core workflow execution engine for n8n-go.
*/
package engine

import (
	"fmt"
	"log"
	"runtime/debug"
	"sync"

	"github.com/dipankar/m9m/internal/connections"
	"github.com/dipankar/m9m/internal/credentials"
	"github.com/dipankar/m9m/internal/model"
	"github.com/dipankar/m9m/internal/nodes/base"
)

// ExecutionResult represents the result of a workflow execution
type ExecutionResult struct {
	Data  []model.DataItem `json:"data"`
	Error error            `json:"error,omitempty"`
}

// NodeRegistry maps node types to their executors
type NodeRegistry map[string]base.NodeExecutor

// WorkflowEngine is the interface for workflow execution
type WorkflowEngine interface {
	// ExecuteWorkflow executes a complete workflow
	ExecuteWorkflow(workflow *model.Workflow, inputData []model.DataItem) (*ExecutionResult, error)

	// ExecuteWorkflowParallel executes multiple workflows in parallel
	ExecuteWorkflowParallel(workflows []*model.Workflow, inputData [][]model.DataItem) ([]*ExecutionResult, error)

	// RegisterNodeExecutor registers a node executor for a node type
	RegisterNodeExecutor(nodeType string, executor base.NodeExecutor)

	// GetNodeExecutor retrieves a node executor for a node type
	GetNodeExecutor(nodeType string) (base.NodeExecutor, error)

	// SetCredentialManager sets the credential manager for the engine
	SetCredentialManager(credentialManager *credentials.CredentialManager)

	// SetConnectionRouter sets the connection router for the engine
	SetConnectionRouter(connectionRouter connections.ConnectionRouter)
}

// workflowEngineImpl is the concrete implementation of WorkflowEngine
type workflowEngineImpl struct {
	nodeRegistry      NodeRegistry
	credentialManager *credentials.CredentialManager
	connectionRouter  connections.ConnectionRouter
}

// NewWorkflowEngine creates a new workflow engine
func NewWorkflowEngine() WorkflowEngine {
	return &workflowEngineImpl{
		nodeRegistry:     make(NodeRegistry),
		connectionRouter: connections.NewConnectionRouter(),
	}
}

// RegisterNodeExecutor registers a node executor for a node type
func (e *workflowEngineImpl) RegisterNodeExecutor(nodeType string, executor base.NodeExecutor) {
	e.nodeRegistry[nodeType] = executor
}

// GetNodeExecutor retrieves a node executor for a node type
func (e *workflowEngineImpl) GetNodeExecutor(nodeType string) (base.NodeExecutor, error) {
	executor, exists := e.nodeRegistry[nodeType]
	if !exists {
		return nil, fmt.Errorf("no executor registered for node type: %s", nodeType)
	}
	return executor, nil
}

// SetCredentialManager sets the credential manager for the engine
func (e *workflowEngineImpl) SetCredentialManager(credentialManager *credentials.CredentialManager) {
	e.credentialManager = credentialManager
}

// SetConnectionRouter sets the connection router for the engine
func (e *workflowEngineImpl) SetConnectionRouter(connectionRouter connections.ConnectionRouter) {
	e.connectionRouter = connectionRouter
}

// ExecuteWorkflow executes a complete workflow
func (e *workflowEngineImpl) ExecuteWorkflow(workflow *model.Workflow, inputData []model.DataItem) (*ExecutionResult, error) {
	if workflow == nil {
		return nil, fmt.Errorf("workflow cannot be nil")
	}

	// Handle empty workflow
	if len(workflow.Nodes) == 0 {
		return &ExecutionResult{
			Data: inputData,
		}, nil
	}

	// Validate workflow connections
	if err := e.connectionRouter.ValidateConnections(workflow); err != nil {
		return nil, fmt.Errorf("invalid workflow connections: %w", err)
	}

	// Check for cycles in workflow
	hasCycles, err := e.connectionRouter.HasCycles(workflow)
	if err != nil {
		return nil, fmt.Errorf("error checking for workflow cycles: %w", err)
	}

	if hasCycles {
		return nil, fmt.Errorf("workflow contains cycles - cannot execute")
	}

	// Resolve workflow credentials if credential manager is available
	if e.credentialManager != nil {
		err := e.credentialManager.ResolveWorkflowCredentials(workflow)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve workflow credentials: %w", err)
		}
	}

	// Get execution order for nodes
	executionOrder, err := e.connectionRouter.GetExecutionOrder(workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to determine execution order: %w", err)
	}

	// Execute nodes in order
	nodeResults := make(map[string][]model.DataItem)

	// Build node lookup map for O(1) access (instead of O(n) linear search)
	nodeMap := make(map[string]*model.Node, len(workflow.Nodes))
	for i := range workflow.Nodes {
		nodeMap[workflow.Nodes[i].Name] = &workflow.Nodes[i]
	}

	// Initialize with input data for starting nodes (nodes with no incoming connections)
	startingNodes := e.findStartingNodes(workflow)

	// Set input data for starting nodes
	for _, nodeName := range startingNodes {
		nodeResults[nodeName] = inputData
	}

	// Execute each node in order
	for _, nodeName := range executionOrder {
		// Get the node by name using O(1) lookup
		node := nodeMap[nodeName]
		if node == nil {
			return nil, fmt.Errorf("node %s not found in workflow", nodeName)
		}

		// Get the executor for this node type
		executor, err := e.GetNodeExecutor(node.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to get executor for node %s: %w", node.Name, err)
		}

		// Validate parameters
		if err := executor.ValidateParameters(node.Parameters); err != nil {
			return nil, fmt.Errorf("invalid parameters for node %s: %w", node.Name, err)
		}

		// Get input data for this node
		inputDataForNode := nodeResults[nodeName]
		if inputDataForNode == nil {
			// No specific input data, use empty input
			inputDataForNode = []model.DataItem{{JSON: make(map[string]interface{})}}
		}

		// Prepare node parameters with credentials if credential manager is available
		finalNodeParams := node.Parameters
		if e.credentialManager != nil {
			var err error
			finalNodeParams, err = e.credentialManager.InjectCredentialsIntoNodeParameters(node.ID, node.Parameters)
			if err != nil {
				return &ExecutionResult{
					Data:  nil,
					Error: fmt.Errorf("error injecting credentials for node %s: %w", node.Name, err),
				}, nil // Return as result.Error, not as function error
			}
		}

		// Execute the node, passing the input data and node parameters
		outputData, err := executor.Execute(inputDataForNode, finalNodeParams)
		if err != nil {
			return &ExecutionResult{
				Data:  nil,
				Error: fmt.Errorf("error executing node %s: %w", node.Name, err),
			}, nil // Return as result.Error, not as function error
		}

		// Store output data for this node
		nodeResults[nodeName] = outputData

		// Route data to connected nodes
		routedData, err := e.connectionRouter.RouteData(nodeName, workflow, outputData)
		if err != nil {
			return nil, fmt.Errorf("error routing data from node %s: %w", node.Name, err)
		}

		// Add routed data to node results
		for targetNode, data := range routedData {
			if nodeResults[targetNode] == nil {
				nodeResults[targetNode] = data
			} else {
				// Append data if node already has data
				nodeResults[targetNode] = append(nodeResults[targetNode], data...)
			}
		}
	}

	// Return the result from the last node in execution order
	var finalResult []model.DataItem
	if len(executionOrder) > 0 {
		lastNodeName := executionOrder[len(executionOrder)-1]
		finalResult = nodeResults[lastNodeName]
	}

	return &ExecutionResult{
		Data: finalResult,
	}, nil
}

// ExecuteWorkflowParallel executes multiple workflows in parallel
func (e *workflowEngineImpl) ExecuteWorkflowParallel(workflows []*model.Workflow, inputData [][]model.DataItem) ([]*ExecutionResult, error) {
	if len(workflows) == 0 {
		return []*ExecutionResult{}, nil
	}

	if len(workflows) != len(inputData) {
		return nil, fmt.Errorf("number of workflows (%d) must match number of input data arrays (%d)", len(workflows), len(inputData))
	}

	// Create channel for results
	results := make([]*ExecutionResult, len(workflows))

	// Create wait group to track completion
	var wg sync.WaitGroup
	wg.Add(len(workflows))

	// Execute each workflow in a separate goroutine with panic recovery
	for i, workflow := range workflows {
		go func(index int, wf *model.Workflow, data []model.DataItem) {
			defer wg.Done()

			// Recover from panics to prevent goroutine crashes from blocking WaitGroup
			defer func() {
				if r := recover(); r != nil {
					log.Printf("PANIC in workflow execution %s: %v\n%s", wf.Name, r, debug.Stack())
					results[index] = &ExecutionResult{
						Data:  nil,
						Error: fmt.Errorf("panic during workflow execution: %v", r),
					}
				}
			}()

			// Execute the workflow
			result, err := e.ExecuteWorkflow(wf, data)
			if err != nil {
				results[index] = &ExecutionResult{
					Data:  nil,
					Error: fmt.Errorf("error executing workflow %s: %w", wf.Name, err),
				}
			} else {
				results[index] = result
			}
		}(i, workflow, inputData[i])
	}

	// Wait for all workflows to complete
	wg.Wait()

	return results, nil
}

// findStartingNodes finds nodes that have no incoming connections
func (e *workflowEngineImpl) findStartingNodes(workflow *model.Workflow) []string {
	if workflow == nil || workflow.Connections == nil {
		// If no connections, all nodes are starting nodes
		var allNodes []string
		for _, node := range workflow.Nodes {
			allNodes = append(allNodes, node.Name)
		}
		return allNodes
	}

	// Create a set of all node names
	allNodes := make(map[string]bool)
	for _, node := range workflow.Nodes {
		allNodes[node.Name] = true
	}

	// Find nodes that are targets of connections (have incoming connections)
	nodesWithIncomingConnections := make(map[string]bool)
	for _, connections := range workflow.Connections {
		// Check main connections
		for _, typeConnections := range connections.Main {
			// typeConnections is []Connection
			for _, connection := range typeConnections {
				// connection is a Connection struct
				nodesWithIncomingConnections[connection.Node] = true
			}
		}
	}

	// Starting nodes are those that are not targets of any connection
	var startingNodes []string
	for nodeName := range allNodes {
		if !nodesWithIncomingConnections[nodeName] {
			startingNodes = append(startingNodes, nodeName)
		}
	}

	return startingNodes
}
