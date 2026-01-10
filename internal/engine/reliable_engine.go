/*
Package engine provides the core workflow execution engine for m9m.
This file adds reliability features (retries, circuit breakers) to the engine.
*/
package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dipankar/m9m/internal/connections"
	"github.com/dipankar/m9m/internal/credentials"
	"github.com/dipankar/m9m/internal/model"
	"github.com/dipankar/m9m/internal/nodes/base"
	"github.com/dipankar/m9m/internal/reliability"
)

// ReliableEngineConfig configures the reliable engine
type ReliableEngineConfig struct {
	// EnableRetries enables automatic retries for failed nodes
	EnableRetries bool

	// RetryConfig configures the retry behavior
	RetryConfig *reliability.RetryConfig

	// EnableCircuitBreaker enables circuit breakers per node type
	EnableCircuitBreaker bool

	// CircuitBreakerConfig configures the circuit breaker behavior
	CircuitBreakerConfig *reliability.CircuitBreakerConfig

	// EnableBulkhead enables bulkhead pattern for concurrency isolation
	EnableBulkhead bool

	// BulkheadConfig configures the bulkhead behavior
	BulkheadConfig *reliability.BulkheadConfig

	// ExecutionTimeout timeout for individual node execution
	ExecutionTimeout time.Duration

	// OnNodeRetry callback when a node is being retried
	OnNodeRetry func(nodeName string, attempt int, err error)

	// OnCircuitOpen callback when a circuit breaker opens
	OnCircuitOpen func(nodeType string)
}

// DefaultReliableEngineConfig returns default configuration
func DefaultReliableEngineConfig() *ReliableEngineConfig {
	return &ReliableEngineConfig{
		EnableRetries:        true,
		RetryConfig:          reliability.DefaultRetryConfig(),
		EnableCircuitBreaker: true,
		CircuitBreakerConfig: reliability.DefaultCircuitBreakerConfig(),
		EnableBulkhead:       true,
		BulkheadConfig:       &reliability.BulkheadConfig{MaxConcurrent: 10, MaxWaiting: 100, Timeout: 30 * time.Second},
		ExecutionTimeout:     30 * time.Second,
	}
}

// ReliableWorkflowEngine wraps the base engine with reliability features
type ReliableWorkflowEngine struct {
	baseEngine       *workflowEngineImpl
	config           *ReliableEngineConfig
	circuitBreakers  map[string]*reliability.CircuitBreaker
	bulkheads        map[string]*reliability.Bulkhead
	mu               sync.RWMutex
	executionMetrics *ExecutionMetrics
}

// ExecutionMetrics tracks execution statistics
type ExecutionMetrics struct {
	TotalExecutions     int64
	SuccessfulNodes     int64
	FailedNodes         int64
	RetriedNodes        int64
	CircuitBreakerTrips int64
	AverageLatencyMs    float64
	mu                  sync.RWMutex // protects all fields above
}

// ExecutionMetricsSnapshot is a copy of ExecutionMetrics without the mutex (safe to return by value)
type ExecutionMetricsSnapshot struct {
	TotalExecutions     int64   `json:"totalExecutions"`
	SuccessfulNodes     int64   `json:"successfulNodes"`
	FailedNodes         int64   `json:"failedNodes"`
	RetriedNodes        int64   `json:"retriedNodes"`
	CircuitBreakerTrips int64   `json:"circuitBreakerTrips"`
	AverageLatencyMs    float64 `json:"averageLatencyMs"`
}

// NewReliableWorkflowEngine creates a new reliable workflow engine
func NewReliableWorkflowEngine(config *ReliableEngineConfig) *ReliableWorkflowEngine {
	if config == nil {
		config = DefaultReliableEngineConfig()
	}

	return &ReliableWorkflowEngine{
		baseEngine: &workflowEngineImpl{
			nodeRegistry:     make(NodeRegistry),
			connectionRouter: connections.NewConnectionRouter(),
		},
		config:           config,
		circuitBreakers:  make(map[string]*reliability.CircuitBreaker),
		bulkheads:        make(map[string]*reliability.Bulkhead),
		executionMetrics: &ExecutionMetrics{},
	}
}

// RegisterNodeExecutor registers a node executor
func (e *ReliableWorkflowEngine) RegisterNodeExecutor(nodeType string, executor base.NodeExecutor) {
	e.baseEngine.RegisterNodeExecutor(nodeType, executor)

	// Initialize circuit breaker for this node type
	if e.config.EnableCircuitBreaker {
		e.mu.Lock()
		e.circuitBreakers[nodeType] = reliability.NewCircuitBreaker(e.config.CircuitBreakerConfig)
		e.mu.Unlock()
	}

	// Initialize bulkhead for this node type
	if e.config.EnableBulkhead {
		e.mu.Lock()
		e.bulkheads[nodeType] = reliability.NewBulkhead(e.config.BulkheadConfig)
		e.mu.Unlock()
	}
}

// GetNodeExecutor retrieves a node executor
func (e *ReliableWorkflowEngine) GetNodeExecutor(nodeType string) (base.NodeExecutor, error) {
	return e.baseEngine.GetNodeExecutor(nodeType)
}

// SetCredentialManager sets the credential manager
func (e *ReliableWorkflowEngine) SetCredentialManager(credentialManager *credentials.CredentialManager) {
	e.baseEngine.SetCredentialManager(credentialManager)
}

// SetConnectionRouter sets the connection router
func (e *ReliableWorkflowEngine) SetConnectionRouter(connectionRouter connections.ConnectionRouter) {
	e.baseEngine.SetConnectionRouter(connectionRouter)
}

// ExecuteWorkflow executes a workflow with reliability features
func (e *ReliableWorkflowEngine) ExecuteWorkflow(workflow *model.Workflow, inputData []model.DataItem) (*ExecutionResult, error) {
	if workflow == nil {
		return nil, fmt.Errorf("workflow cannot be nil")
	}

	e.executionMetrics.mu.Lock()
	e.executionMetrics.TotalExecutions++
	e.executionMetrics.mu.Unlock()

	// Handle empty workflow
	if len(workflow.Nodes) == 0 {
		return &ExecutionResult{Data: inputData}, nil
	}

	// Validate workflow
	if err := e.baseEngine.connectionRouter.ValidateConnections(workflow); err != nil {
		return nil, fmt.Errorf("invalid workflow connections: %w", err)
	}

	hasCycles, err := e.baseEngine.connectionRouter.HasCycles(workflow)
	if err != nil {
		return nil, fmt.Errorf("error checking for workflow cycles: %w", err)
	}
	if hasCycles {
		return nil, fmt.Errorf("workflow contains cycles - cannot execute")
	}

	// Resolve credentials
	if e.baseEngine.credentialManager != nil {
		if err := e.baseEngine.credentialManager.ResolveWorkflowCredentials(workflow); err != nil {
			return nil, fmt.Errorf("failed to resolve workflow credentials: %w", err)
		}
	}

	// Get execution order
	executionOrder, err := e.baseEngine.connectionRouter.GetExecutionOrder(workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to determine execution order: %w", err)
	}

	// Execute nodes
	nodeResults := make(map[string][]model.DataItem)
	startingNodes := e.baseEngine.findStartingNodes(workflow)
	for _, nodeName := range startingNodes {
		nodeResults[nodeName] = inputData
	}

	// Execute each node with reliability features
	for _, nodeName := range executionOrder {
		var node *model.Node
		for i := range workflow.Nodes {
			if workflow.Nodes[i].Name == nodeName {
				node = &workflow.Nodes[i]
				break
			}
		}

		if node == nil {
			return nil, fmt.Errorf("node %s not found in workflow", nodeName)
		}

		executor, err := e.GetNodeExecutor(node.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to get executor for node %s: %w", node.Name, err)
		}

		if err := executor.ValidateParameters(node.Parameters); err != nil {
			return nil, fmt.Errorf("invalid parameters for node %s: %w", node.Name, err)
		}

		inputDataForNode := nodeResults[nodeName]
		if inputDataForNode == nil {
			inputDataForNode = []model.DataItem{{JSON: make(map[string]interface{})}}
		}

		finalNodeParams := node.Parameters
		if e.baseEngine.credentialManager != nil {
			finalNodeParams, err = e.baseEngine.credentialManager.InjectCredentialsIntoNodeParameters(node.ID, node.Parameters)
			if err != nil {
				return &ExecutionResult{
					Error: fmt.Errorf("error injecting credentials for node %s: %w", node.Name, err),
				}, nil
			}
		}

		// Execute with reliability features
		outputData, err := e.executeNodeWithReliability(node, executor, inputDataForNode, finalNodeParams)
		if err != nil {
			e.executionMetrics.mu.Lock()
			e.executionMetrics.FailedNodes++
			e.executionMetrics.mu.Unlock()

			return &ExecutionResult{
				Error: fmt.Errorf("error executing node %s: %w", node.Name, err),
			}, nil
		}

		e.executionMetrics.mu.Lock()
		e.executionMetrics.SuccessfulNodes++
		e.executionMetrics.mu.Unlock()

		nodeResults[nodeName] = outputData

		routedData, err := e.baseEngine.connectionRouter.RouteData(nodeName, workflow, outputData)
		if err != nil {
			return nil, fmt.Errorf("error routing data from node %s: %w", node.Name, err)
		}

		for targetNode, data := range routedData {
			if nodeResults[targetNode] == nil {
				nodeResults[targetNode] = data
			} else {
				nodeResults[targetNode] = append(nodeResults[targetNode], data...)
			}
		}
	}

	var finalResult []model.DataItem
	if len(executionOrder) > 0 {
		lastNodeName := executionOrder[len(executionOrder)-1]
		finalResult = nodeResults[lastNodeName]
	}

	return &ExecutionResult{Data: finalResult}, nil
}

// executeNodeWithReliability executes a node with retry, circuit breaker, and bulkhead
func (e *ReliableWorkflowEngine) executeNodeWithReliability(
	node *model.Node,
	executor base.NodeExecutor,
	inputData []model.DataItem,
	params map[string]interface{},
) ([]model.DataItem, error) {
	ctx := context.Background()
	if e.config.ExecutionTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.config.ExecutionTimeout)
		defer cancel()
	}

	// Get or create circuit breaker and bulkhead
	e.mu.RLock()
	cb := e.circuitBreakers[node.Type]
	bulkhead := e.bulkheads[node.Type]
	e.mu.RUnlock()

	// Capture result from execution to avoid double execution
	var result []model.DataItem
	var resultErr error

	// Core execution function - captures result in closure
	coreExec := func(execCtx context.Context) error {
		select {
		case <-execCtx.Done():
			return execCtx.Err()
		default:
		}
		result, resultErr = executor.Execute(inputData, params)
		return resultErr
	}

	// Wrap with bulkhead if enabled
	var execWithBulkhead func(context.Context) error
	if bulkhead != nil && e.config.EnableBulkhead {
		execWithBulkhead = func(execCtx context.Context) error {
			return bulkhead.Execute(execCtx, coreExec)
		}
	} else {
		execWithBulkhead = coreExec
	}

	// Wrap with circuit breaker if enabled
	var execWithCB func(context.Context) error
	if cb != nil && e.config.EnableCircuitBreaker {
		execWithCB = func(execCtx context.Context) error {
			err := cb.Execute(execCtx, execWithBulkhead)
			if err == reliability.ErrCircuitOpen {
				e.executionMetrics.mu.Lock()
				e.executionMetrics.CircuitBreakerTrips++
				e.executionMetrics.mu.Unlock()

				if e.config.OnCircuitOpen != nil {
					e.config.OnCircuitOpen(node.Type)
				}
			}
			return err
		}
	} else {
		execWithCB = execWithBulkhead
	}

	// Execute with retries if enabled
	if e.config.EnableRetries && e.config.RetryConfig != nil {
		retryPolicy := reliability.NewRetryPolicy(e.config.RetryConfig)

		err := retryPolicy.Execute(ctx, func(retryCtx context.Context) error {
			execErr := execWithCB(retryCtx)
			if execErr != nil {
				e.executionMetrics.mu.Lock()
				e.executionMetrics.RetriedNodes++
				e.executionMetrics.mu.Unlock()

				if e.config.OnNodeRetry != nil {
					e.config.OnNodeRetry(node.Name, 0, execErr)
				}
			}
			return execErr
		})

		if err != nil {
			return nil, err
		}
		// Return the captured result from the successful execution
		return result, nil
	}

	// Execute without retries - check context first
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	err := execWithCB(ctx)
	if err != nil {
		return nil, err
	}
	// Return the captured result from the successful execution
	return result, nil
}

// ExecuteWorkflowParallel executes multiple workflows in parallel
func (e *ReliableWorkflowEngine) ExecuteWorkflowParallel(workflows []*model.Workflow, inputData [][]model.DataItem) ([]*ExecutionResult, error) {
	if len(workflows) == 0 {
		return []*ExecutionResult{}, nil
	}

	if len(workflows) != len(inputData) {
		return nil, fmt.Errorf("number of workflows (%d) must match number of input data arrays (%d)", len(workflows), len(inputData))
	}

	results := make([]*ExecutionResult, len(workflows))
	var wg sync.WaitGroup
	wg.Add(len(workflows))

	for i, workflow := range workflows {
		go func(index int, wf *model.Workflow, data []model.DataItem) {
			defer wg.Done()
			result, err := e.ExecuteWorkflow(wf, data)
			if err != nil {
				results[index] = &ExecutionResult{
					Error: fmt.Errorf("error executing workflow %s: %w", wf.Name, err),
				}
			} else {
				results[index] = result
			}
		}(i, workflow, inputData[i])
	}

	wg.Wait()
	return results, nil
}

// GetMetrics returns execution metrics as a snapshot (safe to use without mutex concerns)
func (e *ReliableWorkflowEngine) GetMetrics() ExecutionMetricsSnapshot {
	e.executionMetrics.mu.RLock()
	defer e.executionMetrics.mu.RUnlock()
	return ExecutionMetricsSnapshot{
		TotalExecutions:     e.executionMetrics.TotalExecutions,
		SuccessfulNodes:     e.executionMetrics.SuccessfulNodes,
		FailedNodes:         e.executionMetrics.FailedNodes,
		RetriedNodes:        e.executionMetrics.RetriedNodes,
		CircuitBreakerTrips: e.executionMetrics.CircuitBreakerTrips,
		AverageLatencyMs:    e.executionMetrics.AverageLatencyMs,
	}
}

// GetCircuitBreakerStatus returns circuit breaker status for all node types
func (e *ReliableWorkflowEngine) GetCircuitBreakerStatus() map[string]string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	status := make(map[string]string)
	for nodeType, cb := range e.circuitBreakers {
		status[nodeType] = cb.State().String()
	}
	return status
}

// ResetCircuitBreaker resets a circuit breaker for a node type
func (e *ReliableWorkflowEngine) ResetCircuitBreaker(nodeType string) error {
	e.mu.RLock()
	cb, exists := e.circuitBreakers[nodeType]
	e.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no circuit breaker for node type: %s", nodeType)
	}

	cb.Reset()
	return nil
}
