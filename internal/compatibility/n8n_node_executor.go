package compatibility

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
	"github.com/neul-labs/m9m/internal/runtime"
)

type N8nNodeExecutor struct {
	*base.BaseNode
	jsRuntime          *runtime.JavaScriptRuntime
	nodeDefinition     *N8nNodeDefinition
	nodeCode           string
	credentialsManager *CredentialsManager
}

type N8nNodeDefinition struct {
	Name        string                 `json:"name"`
	DisplayName string                 `json:"displayName"`
	Description string                 `json:"description"`
	Version     float64                `json:"version"`
	Icon        string                 `json:"icon"`
	Group       []string               `json:"group"`
	Defaults    map[string]interface{} `json:"defaults"`
	Inputs      []string               `json:"inputs"`
	Outputs     []string               `json:"outputs"`
	Properties  []N8nNodeProperty      `json:"properties"`
	Credentials []N8nCredentialType    `json:"credentials"`
	Hooks       map[string]interface{} `json:"hooks"`
	Methods     map[string]interface{} `json:"methods"`
	Webhooks    []N8nWebhookConfig     `json:"webhooks"`
	Polling     bool                   `json:"polling"`
	TriggerPanel interface{}           `json:"triggerPanel"`
}

type N8nNodeProperty struct {
	DisplayName    string                 `json:"displayName"`
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`
	Default        interface{}            `json:"default"`
	Required       bool                   `json:"required"`
	Description    string                 `json:"description"`
	Options        []N8nPropertyOption    `json:"options"`
	DisplayOptions map[string]interface{} `json:"displayOptions"`
	TypeOptions    map[string]interface{} `json:"typeOptions"`
	Placeholder    string                 `json:"placeholder"`
	ExtractValue   map[string]interface{} `json:"extractValue"`
	LoadOptions    map[string]interface{} `json:"loadOptions"`
}

type N8nPropertyOption struct {
	Name        string      `json:"name"`
	Value       interface{} `json:"value"`
	Description string      `json:"description"`
}

type N8nCredentialType struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	DisplayName string `json:"displayName"`
	TestRequest string `json:"testRequest"`
}

type N8nWebhookConfig struct {
	Name       string `json:"name"`
	HttpMethod string `json:"httpMethod"`
	Path       string `json:"path"`
	IsFullPath bool   `json:"isFullPath"`
}

type N8nExecutionContext struct {
	*runtime.ExecutionContext
	Helpers          *N8nExecutionHelpers
	GetNodeParameter func(string, int) interface{}
	GetCredentials   func(string, string) (interface{}, error)
	GetInputData     func(int) []interface{}
	PrepareOutputData func([]interface{}) []interface{}
}

type N8nExecutionHelpers struct {
	HttpRequest                 func(interface{}) (interface{}, error)
	HttpRequestWithAuth         func(string, interface{}) (interface{}, error)
	RequestOAuth1               func(string, interface{}) (interface{}, error)
	RequestOAuth2               func(string, interface{}) (interface{}, error)
	ReturnJsonArray             func(interface{}) []interface{}
	ConstructExecutionMetaData  func([]interface{}, map[string]interface{}) []interface{}
}

type CredentialsManager struct {
	credentials map[string]map[string]interface{}
}

func NewN8nNodeExecutor(nodePath string, jsRuntime *runtime.JavaScriptRuntime) (*N8nNodeExecutor, error) {
	// Read node definition file
	definitionPath := filepath.Join(nodePath, "*.node.json")
	definitionFiles, err := filepath.Glob(definitionPath)
	if err != nil || len(definitionFiles) == 0 {
		return nil, fmt.Errorf("no node definition found in %s", nodePath)
	}

	definitionData, err := ioutil.ReadFile(definitionFiles[0])
	if err != nil {
		return nil, fmt.Errorf("failed to read node definition: %w", err)
	}

	var definition N8nNodeDefinition
	if err := json.Unmarshal(definitionData, &definition); err != nil {
		return nil, fmt.Errorf("failed to parse node definition: %w", err)
	}

	// Read node implementation file
	jsFiles, err := filepath.Glob(filepath.Join(nodePath, "*.node.js"))
	if err != nil || len(jsFiles) == 0 {
		return nil, fmt.Errorf("no node implementation found in %s", nodePath)
	}

	nodeCode, err := ioutil.ReadFile(jsFiles[0])
	if err != nil {
		return nil, fmt.Errorf("failed to read node implementation: %w", err)
	}

	credentialsManager := &CredentialsManager{
		credentials: make(map[string]map[string]interface{}),
	}

	executor := &N8nNodeExecutor{
		BaseNode: base.NewBaseNode(base.NodeDescription{
			Name:        definition.Name,
			Description: definition.Description,
			Category:    strings.Join(definition.Group, ","),
		}),
		jsRuntime:          jsRuntime,
		nodeDefinition:     &definition,
		nodeCode:           string(nodeCode),
		credentialsManager: credentialsManager,
	}

	return executor, nil
}

// GetNodeDefinition returns the node definition
func (n *N8nNodeExecutor) GetNodeDefinition() *N8nNodeDefinition {
	return n.nodeDefinition
}

// Execute implements the standard node interface
func (n *N8nNodeExecutor) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	// Prepare execution context
	context := &N8nExecutionContext{
		ExecutionContext: &runtime.ExecutionContext{
			WorkflowID:  fmt.Sprintf("workflow_%d", time.Now().Unix()),
			ExecutionID: fmt.Sprintf("execution_%d", time.Now().Unix()),
			NodeID:      n.nodeDefinition.Name,
			ItemIndex:   0,
			RunIndex:    0,
			Mode:        "manual",
			Timezone:    "UTC",
			Variables:   make(map[string]interface{}),
			Credentials: make(map[string]interface{}),
		},
		Helpers: n.createExecutionHelpers(),
	}

	// Set up parameter access function
	context.GetNodeParameter = func(parameterName string, itemIndex int) interface{} {
		if value, exists := nodeParams[parameterName]; exists {
			return value
		}
		return nil
	}

	context.GetCredentials = func(credentialType string, nodeCredentialName string) (interface{}, error) {
		return n.credentialsManager.GetCredentials(credentialType, nodeCredentialName)
	}

	context.GetInputData = func(inputIndex int) []interface{} {
		var result []interface{}
		for _, item := range inputData {
			result = append(result, map[string]interface{}{
				"json": item.JSON,
			})
		}
		return result
	}

	context.PrepareOutputData = func(outputData []interface{}) []interface{} {
		return outputData
	}

	// Set execution context in JavaScript runtime
	n.jsRuntime.SetExecutionContext(context.ExecutionContext)

	// Prepare JavaScript execution environment
	executionCode := n.prepareExecutionCode(context)

	// Execute the node
	result, err := n.jsRuntime.Execute(executionCode, context.ExecutionContext, inputData)
	if err != nil {
		return nil, fmt.Errorf("node execution failed: %w", err)
	}

	// Convert result back to n8n-go format
	outputItems, err := n.convertResultToItems(result)
	if err != nil {
		return nil, fmt.Errorf("result conversion failed: %w", err)
	}

	return outputItems, nil
}

func (n *N8nNodeExecutor) prepareExecutionCode(context *N8nExecutionContext) string {
	// Create the execution wrapper that mimics n8n's execution environment
	executionWrapper := fmt.Sprintf(`
		// Set up n8n execution context
		const context = %s;

		// Helper functions available to the node
		function getNodeParameter(parameterName, itemIndex = 0) {
			return context.getNodeParameter(parameterName, itemIndex);
		}

		function getCredentials(credentialType, nodeCredentialName) {
			return context.getCredentials(credentialType, nodeCredentialName);
		}

		function getInputData(inputIndex = 0) {
			return context.getInputData(inputIndex);
		}

		function prepareOutputData(outputData) {
			return context.prepareOutputData(outputData);
		}

		// Execution helpers
		const helpers = context.helpers;

		// Set up this context for the node
		const thisContext = {
			getNodeParameter: getNodeParameter,
			getCredentials: getCredentials,
			getInputData: getInputData,
			prepareOutputData: prepareOutputData,
			helpers: helpers
		};

		// Load the node implementation
		%s

		// Execute the node
		let nodeResult;
		try {
			if (typeof execute === 'function') {
				nodeResult = execute.call(thisContext);
			} else if (typeof thisContext.execute === 'function') {
				nodeResult = thisContext.execute.call(thisContext);
			} else {
				throw new Error('No execute function found in node');
			}
		} catch (error) {
			throw new Error('Node execution failed: ' + error.message);
		}

		// Return the result
		nodeResult;
	`, n.contextToJSON(context), n.nodeCode)

	return executionWrapper
}

func (n *N8nNodeExecutor) contextToJSON(context *N8nExecutionContext) string {
	// Convert execution context to JSON for JavaScript
	contextData := map[string]interface{}{
		"getNodeParameter":  "function(parameterName, itemIndex) { return getNodeParameter(parameterName, itemIndex); }",
		"getCredentials":    "function(credentialType, nodeCredentialName) { return getCredentials(credentialType, nodeCredentialName); }",
		"getInputData":      "function(inputIndex) { return getInputData(inputIndex); }",
		"prepareOutputData": "function(outputData) { return prepareOutputData(outputData); }",
		"helpers": map[string]string{
			"httpRequest":         "function(options) { return helpers.httpRequest(options); }",
			"httpRequestWithAuth": "function(credentialType, options) { return helpers.httpRequestWithAuth(credentialType, options); }",
			"returnJsonArray":     "function(data) { return helpers.returnJsonArray(data); }",
		},
	}

	data, _ := json.Marshal(contextData)
	return string(data)
}

func (n *N8nNodeExecutor) createExecutionHelpers() *N8nExecutionHelpers {
	return &N8nExecutionHelpers{
		HttpRequest: func(options interface{}) (interface{}, error) {
			// Mock HTTP request implementation
			return map[string]interface{}{
				"status": 200,
				"body":   `{"success": true}`,
				"headers": map[string]string{
					"content-type": "application/json",
				},
			}, nil
		},
		HttpRequestWithAuth: func(credentialType string, options interface{}) (interface{}, error) {
			// Mock authenticated HTTP request
			return map[string]interface{}{
				"status": 200,
				"body":   `{"success": true, "authenticated": true}`,
				"headers": map[string]string{
					"content-type": "application/json",
				},
			}, nil
		},
		RequestOAuth1: func(credentialType string, options interface{}) (interface{}, error) {
			return map[string]interface{}{
				"status": 200,
				"body":   `{"success": true, "oauth1": true}`,
			}, nil
		},
		RequestOAuth2: func(credentialType string, options interface{}) (interface{}, error) {
			return map[string]interface{}{
				"status": 200,
				"body":   `{"success": true, "oauth2": true}`,
			}, nil
		},
		ReturnJsonArray: func(data interface{}) []interface{} {
			if arr, ok := data.([]interface{}); ok {
				return arr
			}
			return []interface{}{data}
		},
		ConstructExecutionMetaData: func(inputData []interface{}, metadata map[string]interface{}) []interface{} {
			// Add metadata to each item
			for i, item := range inputData {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if itemMap["pairedItem"] == nil {
						itemMap["pairedItem"] = map[string]interface{}{
							"item": i,
						}
					}
				}
			}
			return inputData
		},
	}
}

func (n *N8nNodeExecutor) convertResultToItems(result interface{}) ([]model.DataItem, error) {
	var items []model.DataItem

	switch r := result.(type) {
	case []interface{}:
		for _, item := range r {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if jsonData, exists := itemMap["json"]; exists {
					if jsonMap, ok := jsonData.(map[string]interface{}); ok {
						items = append(items, model.DataItem{JSON: jsonMap})
					}
				} else {
					// If no json property, use the item itself
					items = append(items, model.DataItem{JSON: itemMap})
				}
			}
		}
	case map[string]interface{}:
		if jsonData, exists := r["json"]; exists {
			if jsonMap, ok := jsonData.(map[string]interface{}); ok {
				items = append(items, model.DataItem{JSON: jsonMap})
			}
		} else {
			items = append(items, model.DataItem{JSON: r})
		}
	default:
		// Convert single value to item
		items = append(items, model.DataItem{
			JSON: map[string]interface{}{
				"value": result,
			},
		})
	}

	return items, nil
}

// Credentials Management
func (cm *CredentialsManager) SetCredentials(credentialType, name string, credentials map[string]interface{}) {
	if cm.credentials[credentialType] == nil {
		cm.credentials[credentialType] = make(map[string]interface{})
	}
	cm.credentials[credentialType][name] = credentials
}

func (cm *CredentialsManager) GetCredentials(credentialType, name string) (interface{}, error) {
	if typeCredentials, exists := cm.credentials[credentialType]; exists {
		if credentials, exists := typeCredentials[name]; exists {
			return credentials, nil
		}
	}
	return nil, fmt.Errorf("credentials not found: %s/%s", credentialType, name)
}

func (cm *CredentialsManager) ListCredentials() map[string][]string {
	result := make(map[string][]string)
	for credType, credentials := range cm.credentials {
		var names []string
		for name := range credentials {
			names = append(names, name)
		}
		result[credType] = names
	}
	return result
}

// Factory function to create n8n-compatible nodes
func CreateN8nCompatibleNode(nodePath string, jsRuntime *runtime.JavaScriptRuntime) (*N8nNodeExecutor, error) {
	return NewN8nNodeExecutor(nodePath, jsRuntime)
}

// Bulk loader for n8n nodes directory
func LoadN8nNodesFromDirectory(nodesDir string, jsRuntime *runtime.JavaScriptRuntime) (map[string]*N8nNodeExecutor, error) {
	nodes := make(map[string]*N8nNodeExecutor)

	// Walk through directories looking for node definitions
	entries, err := ioutil.ReadDir(nodesDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			nodePath := filepath.Join(nodesDir, entry.Name())
			node, err := NewN8nNodeExecutor(nodePath, jsRuntime)
			if err == nil {
				nodes[node.nodeDefinition.Name] = node
			}
		}
	}

	return nodes, nil
}
