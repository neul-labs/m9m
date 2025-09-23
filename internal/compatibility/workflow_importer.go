package compatibility

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/n8n-go/n8n-go/pkg/model"
)

type N8nWorkflowImporter struct {
	nodeRegistry       map[string]*N8nNodeExecutor
	credentialsManager *CredentialsManager
	expressionConverter *ExpressionConverter
}

type N8nWorkflowData struct {
	ID          interface{}          `json:"id"`
	Name        string               `json:"name"`
	Active      bool                 `json:"active"`
	Nodes       []N8nNodeData        `json:"nodes"`
	Connections map[string][]N8nConnection `json:"connections"`
	Settings    map[string]interface{} `json:"settings"`
	StaticData  map[string]interface{} `json:"staticData"`
	Tags        []interface{}        `json:"tags"`
	TriggerCount int                 `json:"triggerCount"`
	Meta        map[string]interface{} `json:"meta"`
	PinData     map[string][]interface{} `json:"pinData"`
}

type N8nNodeData struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	TypeVersion   float64                `json:"typeVersion"`
	Position      []float64              `json:"position"`
	Parameters    map[string]interface{} `json:"parameters"`
	Credentials   map[string]string      `json:"credentials"`
	Disabled      bool                   `json:"disabled"`
	Notes         string                 `json:"notes"`
	Color         string                 `json:"color"`
	OnError       string                 `json:"onError"`
	RetryOnFail   bool                   `json:"retryOnFail"`
	MaxTries      int                    `json:"maxTries"`
	WaitBetween   int                    `json:"waitBetween"`
	AlwaysOutputData bool                `json:"alwaysOutputData"`
	ExecuteOnce   bool                   `json:"executeOnce"`
	ContinueOnFail bool                  `json:"continueOnFail"`
}

type N8nConnection struct {
	Node       string `json:"node"`
	Type       string `json:"type"`
	Index      int    `json:"index"`
	OutputIndex int   `json:"outputIndex"`
}

type ExpressionConverter struct {
	// Handles conversion between n8n and n8n-go expression formats
}

type ImportResult struct {
	Workflow        *model.Workflow        `json:"workflow"`
	ConversionIssues []ConversionIssue     `json:"conversion_issues"`
	MissingNodes    []string              `json:"missing_nodes"`
	Statistics      *ImportStatistics     `json:"statistics"`
}

type ConversionIssue struct {
	Type        string `json:"type"`        // "warning", "error", "info"
	NodeID      string `json:"node_id"`
	Field       string `json:"field"`
	Message     string `json:"message"`
	Suggestion  string `json:"suggestion"`
	AutoFixed   bool   `json:"auto_fixed"`
}

type ImportStatistics struct {
	TotalNodes      int    `json:"total_nodes"`
	ConvertedNodes  int    `json:"converted_nodes"`
	SkippedNodes    int    `json:"skipped_nodes"`
	TotalConnections int   `json:"total_connections"`
	ConvertedConnections int `json:"converted_connections"`
	ExpressionsConverted int `json:"expressions_converted"`
	CredentialsFound     int `json:"credentials_found"`
}

func NewN8nWorkflowImporter() *N8nWorkflowImporter {
	return &N8nWorkflowImporter{
		nodeRegistry:        make(map[string]*N8nNodeExecutor),
		credentialsManager:  &CredentialsManager{credentials: make(map[string]map[string]interface{})},
		expressionConverter: &ExpressionConverter{},
	}
}

func (importer *N8nWorkflowImporter) RegisterNode(nodeType string, executor *N8nNodeExecutor) {
	importer.nodeRegistry[nodeType] = executor
}

func (importer *N8nWorkflowImporter) ImportWorkflow(n8nJson []byte) (*ImportResult, error) {
	var n8nWorkflow N8nWorkflowData
	if err := json.Unmarshal(n8nJson, &n8nWorkflow); err != nil {
		return nil, fmt.Errorf("failed to parse n8n workflow JSON: %w", err)
	}

	result := &ImportResult{
		ConversionIssues: []ConversionIssue{},
		MissingNodes:     []string{},
		Statistics: &ImportStatistics{
			TotalNodes:       len(n8nWorkflow.Nodes),
			TotalConnections: importer.countConnections(n8nWorkflow.Connections),
		},
	}

	// Convert workflow metadata
	workflow := &model.Workflow{
		ID:          importer.convertWorkflowID(n8nWorkflow.ID),
		Name:        n8nWorkflow.Name,
		Description: importer.extractDescription(n8nWorkflow),
		Active:      n8nWorkflow.Active,
		Nodes:       []model.Node{},
		Connections: []model.Connection{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     "1.0.0",
		Tags:        importer.convertTags(n8nWorkflow.Tags),
		Settings:    n8nWorkflow.Settings,
	}

	// Convert nodes
	nodeIdMap := make(map[string]string) // n8n ID -> n8n-go ID mapping
	for _, n8nNode := range n8nWorkflow.Nodes {
		convertedNode, issues := importer.convertNode(n8nNode)
		if convertedNode != nil {
			workflow.Nodes = append(workflow.Nodes, *convertedNode)
			nodeIdMap[n8nNode.ID] = convertedNode.ID
			result.Statistics.ConvertedNodes++
		} else {
			result.Statistics.SkippedNodes++
		}
		result.ConversionIssues = append(result.ConversionIssues, issues...)
	}

	// Convert connections
	for sourceNodeId, connectionGroups := range n8nWorkflow.Connections {
		for outputName, connections := range connectionGroups {
			for _, connection := range connections {
				convertedConnection, issues := importer.convertConnection(
					sourceNodeId, outputName, connection, nodeIdMap,
				)
				if convertedConnection != nil {
					workflow.Connections = append(workflow.Connections, *convertedConnection)
					result.Statistics.ConvertedConnections++
				}
				result.ConversionIssues = append(result.ConversionIssues, issues...)
			}
		}
	}

	result.Workflow = workflow
	return result, nil
}

func (importer *N8nWorkflowImporter) convertNode(n8nNode N8nNodeData) (*model.Node, []ConversionIssue) {
	var issues []ConversionIssue

	// Check if we have a registered executor for this node type
	executor, exists := importer.nodeRegistry[n8nNode.Type]
	if !exists {
		issues = append(issues, ConversionIssue{
			Type:       "warning",
			NodeID:     n8nNode.ID,
			Message:    fmt.Sprintf("Node type '%s' not found in registry", n8nNode.Type),
			Suggestion: "Register the node type or use a compatible alternative",
		})

		// Check for built-in equivalents
		if equivalent := importer.findEquivalentNodeType(n8nNode.Type); equivalent != "" {
			issues = append(issues, ConversionIssue{
				Type:       "info",
				NodeID:     n8nNode.ID,
				Message:    fmt.Sprintf("Consider using '%s' as an alternative", equivalent),
				AutoFixed:  false,
			})
		}
	}

	// Convert parameters with expression conversion
	convertedParams, paramIssues := importer.convertParameters(n8nNode.Parameters, n8nNode.ID)
	issues = append(issues, paramIssues...)

	// Convert configuration
	config, _ := json.Marshal(convertedParams)

	node := &model.Node{
		ID:          n8nNode.ID,
		Name:        n8nNode.Name,
		Type:        n8nNode.Type,
		Position:    importer.convertPosition(n8nNode.Position),
		Config:      config,
		Disabled:    n8nNode.Disabled,
		Notes:       n8nNode.Notes,
		Credentials: importer.convertCredentials(n8nNode.Credentials),
		RetryPolicy: &model.RetryPolicy{
			Enabled:      n8nNode.RetryOnFail,
			MaxAttempts:  n8nNode.MaxTries,
			DelayMs:      n8nNode.WaitBetween,
			BackoffType:  "fixed",
		},
		ErrorHandling: &model.ErrorHandling{
			ContinueOnFail: n8nNode.ContinueOnFail,
			OnError:        n8nNode.OnError,
		},
	}

	return node, issues
}

func (importer *N8nWorkflowImporter) convertParameters(params map[string]interface{}, nodeId string) (map[string]interface{}, []ConversionIssue) {
	var issues []ConversionIssue
	converted := make(map[string]interface{})

	for key, value := range params {
		convertedValue, paramIssues := importer.convertParameterValue(value, fmt.Sprintf("%s.%s", nodeId, key))
		converted[key] = convertedValue
		issues = append(issues, paramIssues...)
	}

	return converted, issues
}

func (importer *N8nWorkflowImporter) convertParameterValue(value interface{}, context string) (interface{}, []ConversionIssue) {
	var issues []ConversionIssue

	switch v := value.(type) {
	case string:
		// Check if it's an n8n expression
		if importer.isN8nExpression(v) {
			converted, err := importer.expressionConverter.ConvertExpression(v)
			if err != nil {
				issues = append(issues, ConversionIssue{
					Type:    "warning",
					Field:   context,
					Message: fmt.Sprintf("Expression conversion failed: %s", err.Error()),
					Suggestion: "Manual review required for expression",
				})
				return v, issues // Return original on conversion failure
			}
			return converted, issues
		}
		return v, issues

	case map[string]interface{}:
		converted := make(map[string]interface{})
		for k, subValue := range v {
			convertedSubValue, subIssues := importer.convertParameterValue(subValue, fmt.Sprintf("%s.%s", context, k))
			converted[k] = convertedSubValue
			issues = append(issues, subIssues...)
		}
		return converted, issues

	case []interface{}:
		var converted []interface{}
		for i, item := range v {
			convertedItem, itemIssues := importer.convertParameterValue(item, fmt.Sprintf("%s[%d]", context, i))
			converted = append(converted, convertedItem)
			issues = append(issues, itemIssues...)
		}
		return converted, issues

	default:
		return v, issues
	}
}

func (importer *N8nWorkflowImporter) convertConnection(sourceNodeId, outputName string, connection N8nConnection, nodeIdMap map[string]string) (*model.Connection, []ConversionIssue) {
	var issues []ConversionIssue

	// Map n8n node IDs to n8n-go node IDs
	sourceId, sourceExists := nodeIdMap[sourceNodeId]
	targetId, targetExists := nodeIdMap[connection.Node]

	if !sourceExists {
		issues = append(issues, ConversionIssue{
			Type:    "error",
			Message: fmt.Sprintf("Source node '%s' not found in converted nodes", sourceNodeId),
		})
		return nil, issues
	}

	if !targetExists {
		issues = append(issues, ConversionIssue{
			Type:    "error",
			Message: fmt.Sprintf("Target node '%s' not found in converted nodes", connection.Node),
		})
		return nil, issues
	}

	conn := &model.Connection{
		Source: sourceId,
		Target: targetId,
		Output: outputName,
		Input:  connection.Type,
	}

	return conn, issues
}

func (importer *N8nWorkflowImporter) isN8nExpression(value string) bool {
	// Check for common n8n expression patterns
	patterns := []string{
		"={{",      // Standard expression
		"$json",    // JSON data access
		"$node",    // Node data access
		"$parameter", // Parameter access
		"$workflow", // Workflow data
		"$execution", // Execution data
		"$binary",   // Binary data
		"$env",     // Environment variables
	}

	for _, pattern := range patterns {
		if strings.Contains(value, pattern) {
			return true
		}
	}

	return false
}

func (importer *N8nWorkflowImporter) findEquivalentNodeType(nodeType string) string {
	// Map common n8n node types to n8n-go equivalents
	equivalents := map[string]string{
		"n8n-nodes-base.httpRequest":     "http-request",
		"n8n-nodes-base.set":             "set-data",
		"n8n-nodes-base.if":              "condition",
		"n8n-nodes-base.switch":          "switch",
		"n8n-nodes-base.merge":           "merge-data",
		"n8n-nodes-base.function":        "javascript-function",
		"n8n-nodes-base.webhook":         "webhook-trigger",
		"n8n-nodes-base.schedule":        "schedule-trigger",
		"n8n-nodes-base.manualTrigger":   "manual-trigger",
		"n8n-nodes-base.start":           "start",
		"n8n-nodes-base.executeWorkflow": "workflow-trigger",
		"n8n-nodes-base.wait":            "wait",
		"n8n-nodes-base.stopAndError":    "stop-error",
		"n8n-nodes-base.noOp":            "no-operation",
	}

	if equivalent, exists := equivalents[nodeType]; exists {
		return equivalent
	}

	// Try prefix matching for similar node types
	for n8nType, goType := range equivalents {
		if strings.Contains(nodeType, strings.TrimPrefix(n8nType, "n8n-nodes-base.")) {
			return goType
		}
	}

	return ""
}

func (importer *N8nWorkflowImporter) convertWorkflowID(id interface{}) string {
	switch v := id.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	case int:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("workflow_%d", time.Now().Unix())
	}
}

func (importer *N8nWorkflowImporter) extractDescription(workflow N8nWorkflowData) string {
	if settings, ok := workflow.Settings["description"]; ok {
		if desc, ok := settings.(string); ok {
			return desc
		}
	}
	return ""
}

func (importer *N8nWorkflowImporter) convertTags(tags []interface{}) []string {
	var result []string
	for _, tag := range tags {
		if tagStr, ok := tag.(string); ok {
			result = append(result, tagStr)
		}
	}
	return result
}

func (importer *N8nWorkflowImporter) convertPosition(position []float64) []int {
	if len(position) >= 2 {
		return []int{int(position[0]), int(position[1])}
	}
	return []int{0, 0}
}

func (importer *N8nWorkflowImporter) convertCredentials(creds map[string]string) map[string]string {
	// Direct mapping for now - could add credential type mapping
	return creds
}

func (importer *N8nWorkflowImporter) countConnections(connections map[string][]N8nConnection) int {
	count := 0
	for _, connectionGroups := range connections {
		for _, connectionList := range connectionGroups {
			count += len(connectionList)
		}
	}
	return count
}

// Expression converter methods
func (ec *ExpressionConverter) ConvertExpression(n8nExpression string) (string, error) {
	// Remove n8n expression wrapper if present
	expression := strings.TrimSpace(n8nExpression)
	if strings.HasPrefix(expression, "={{") && strings.HasSuffix(expression, "}}") {
		expression = strings.TrimSuffix(strings.TrimPrefix(expression, "={{"), "}}")
	}

	// Convert common n8n expression patterns to n8n-go format
	conversions := map[string]string{
		"$json":               "$item.json",
		"$node[":              "$nodes[",
		"$parameter[":         "$params[",
		"$workflow.id":        "$workflow.id",
		"$workflow.name":      "$workflow.name",
		"$execution.id":       "$execution.id",
		"$execution.mode":     "$execution.mode",
		"$binary":             "$binary",
		"$env":                "$env",
		"$now":                "$now",
		"$today":              "$today",
		"moment()":            "moment()",
		"luxon.DateTime.now()": "moment()",
	}

	converted := expression
	for n8nPattern, goPattern := range conversions {
		converted = strings.ReplaceAll(converted, n8nPattern, goPattern)
	}

	// Handle complex patterns with regex if needed
	converted = ec.convertComplexPatterns(converted)

	return fmt.Sprintf("{{%s}}", converted), nil
}

func (ec *ExpressionConverter) convertComplexPatterns(expression string) string {
	// Convert $node["NodeName"].json to $nodes["NodeName"].json
	// Convert $parameter["paramName"] to $params["paramName"]
	// Add more complex pattern conversions as needed

	// For now, return as-is
	return expression
}

// Export workflow to n8n format
func (importer *N8nWorkflowImporter) ExportWorkflow(workflow *model.Workflow) ([]byte, error) {
	n8nWorkflow := N8nWorkflowData{
		ID:          workflow.ID,
		Name:        workflow.Name,
		Active:      workflow.Active,
		Nodes:       []N8nNodeData{},
		Connections: make(map[string][]N8nConnection),
		Settings:    workflow.Settings,
		StaticData:  make(map[string]interface{}),
		Tags:        []interface{}{},
		TriggerCount: 0,
		Meta:        make(map[string]interface{}),
		PinData:     make(map[string][]interface{}),
	}

	// Convert tags
	for _, tag := range workflow.Tags {
		n8nWorkflow.Tags = append(n8nWorkflow.Tags, tag)
	}

	// Convert nodes
	for _, node := range workflow.Nodes {
		n8nNode := N8nNodeData{
			ID:          node.ID,
			Name:        node.Name,
			Type:        node.Type,
			TypeVersion: 1.0,
			Position:    []float64{float64(node.Position[0]), float64(node.Position[1])},
			Parameters:  make(map[string]interface{}),
			Credentials: node.Credentials,
			Disabled:    node.Disabled,
			Notes:       node.Notes,
		}

		// Convert parameters back
		if len(node.Config) > 0 {
			json.Unmarshal(node.Config, &n8nNode.Parameters)
		}

		// Convert retry policy
		if node.RetryPolicy != nil {
			n8nNode.RetryOnFail = node.RetryPolicy.Enabled
			n8nNode.MaxTries = node.RetryPolicy.MaxAttempts
			n8nNode.WaitBetween = node.RetryPolicy.DelayMs
		}

		// Convert error handling
		if node.ErrorHandling != nil {
			n8nNode.ContinueOnFail = node.ErrorHandling.ContinueOnFail
			n8nNode.OnError = node.ErrorHandling.OnError
		}

		n8nWorkflow.Nodes = append(n8nWorkflow.Nodes, n8nNode)
	}

	// Convert connections
	for _, connection := range workflow.Connections {
		if n8nWorkflow.Connections[connection.Source] == nil {
			n8nWorkflow.Connections[connection.Source] = []N8nConnection{}
		}

		n8nConnection := N8nConnection{
			Node:        connection.Target,
			Type:        connection.Input,
			Index:       0,
			OutputIndex: 0,
		}

		n8nWorkflow.Connections[connection.Source] = append(
			n8nWorkflow.Connections[connection.Source],
			n8nConnection,
		)
	}

	return json.MarshalIndent(n8nWorkflow, "", "  ")
}

// Validation functions
func (importer *N8nWorkflowImporter) ValidateN8nWorkflow(n8nJson []byte) error {
	var workflow N8nWorkflowData
	if err := json.Unmarshal(n8nJson, &workflow); err != nil {
		return fmt.Errorf("invalid n8n workflow JSON: %w", err)
	}

	if workflow.Name == "" {
		return fmt.Errorf("workflow name is required")
	}

	if len(workflow.Nodes) == 0 {
		return fmt.Errorf("workflow must contain at least one node")
	}

	// Validate node references in connections
	nodeIds := make(map[string]bool)
	for _, node := range workflow.Nodes {
		nodeIds[node.ID] = true
	}

	for sourceId, connectionGroups := range workflow.Connections {
		if !nodeIds[sourceId] {
			return fmt.Errorf("connection references non-existent source node: %s", sourceId)
		}

		for _, connections := range connectionGroups {
			for _, connection := range connections {
				if !nodeIds[connection.Node] {
					return fmt.Errorf("connection references non-existent target node: %s", connection.Node)
				}
			}
		}
	}

	return nil
}