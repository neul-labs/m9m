package compatibility

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/neul-labs/m9m/internal/model"
)

type N8nWorkflowImporter struct {
	nodeRegistry       map[string]*N8nNodeExecutor
	credentialsManager *CredentialsManager
	expressionConverter *ExpressionConverter
}

type N8nWorkflowData struct {
	ID          interface{}                           `json:"id"`
	Name        string                                `json:"name"`
	Active      bool                                  `json:"active"`
	Nodes       []N8nNodeData                         `json:"nodes"`
	Connections map[string]N8nConnectionGroup         `json:"connections"`
	Settings    map[string]interface{}                `json:"settings"`
	StaticData  map[string]interface{}                `json:"staticData"`
	Tags        []interface{}                         `json:"tags"`
	TriggerCount int                                  `json:"triggerCount"`
	Meta        map[string]interface{}                `json:"meta"`
	PinData     map[string][]interface{}              `json:"pinData"`
}

type N8nConnectionGroup struct {
	Main [][]N8nConnection `json:"main"`
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
		Connections: make(map[string]model.Connections),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		VersionID:   "1.0.0",
		Tags:        importer.convertTags(n8nWorkflow.Tags),
		Settings:    importer.convertSettings(n8nWorkflow.Settings),
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
	for sourceNodeId, connectionGroup := range n8nWorkflow.Connections {
		// Convert connection group for this source node
		convertedGroup := model.Connections{
			Main: make([][]model.Connection, len(connectionGroup.Main)),
		}

		for outputIndex, outputConnections := range connectionGroup.Main {
			convertedGroup.Main[outputIndex] = make([]model.Connection, 0)
			for _, connection := range outputConnections {
				convertedConnection, issues := importer.convertConnection(
					sourceNodeId, outputIndex, connection, nodeIdMap,
				)
				if convertedConnection != nil {
					convertedGroup.Main[outputIndex] = append(convertedGroup.Main[outputIndex], *convertedConnection)
					result.Statistics.ConvertedConnections++
				}
				result.ConversionIssues = append(result.ConversionIssues, issues...)
			}
		}

		workflow.Connections[sourceNodeId] = convertedGroup
	}

	result.Workflow = workflow
	return result, nil
}

func (importer *N8nWorkflowImporter) convertNode(n8nNode N8nNodeData) (*model.Node, []ConversionIssue) {
	var issues []ConversionIssue

	// Check if we have a registered executor for this node type
	_, exists := importer.nodeRegistry[n8nNode.Type]
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

	// Convert disabled flag to *bool
	var disabled *bool
	if n8nNode.Disabled {
		disabled = &n8nNode.Disabled
	}

	node := &model.Node{
		ID:          n8nNode.ID,
		Name:        n8nNode.Name,
		Type:        n8nNode.Type,
		TypeVersion: int(n8nNode.TypeVersion),
		Position:    importer.convertPosition(n8nNode.Position),
		Parameters:  convertedParams,
		Disabled:    disabled,
		Notes:       n8nNode.Notes,
		Credentials: importer.convertCredentials(n8nNode.Credentials),
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

func (importer *N8nWorkflowImporter) convertConnection(sourceNodeId string, outputIndex int, connection N8nConnection, nodeIdMap map[string]string) (*model.Connection, []ConversionIssue) {
	var issues []ConversionIssue

	// Check if source node exists
	_, sourceExists := nodeIdMap[sourceNodeId]
	_, targetExists := nodeIdMap[connection.Node]

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
		Node:  connection.Node,
		Type:  connection.Type,
		Index: connection.Index,
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

func (importer *N8nWorkflowImporter) convertCredentials(creds map[string]string) map[string]model.Credential {
	result := make(map[string]model.Credential)
	for credType, credName := range creds {
		result[credType] = model.Credential{
			Name: credName,
			Type: credType,
		}
	}
	return result
}

func (importer *N8nWorkflowImporter) convertSettings(settings map[string]interface{}) *model.WorkflowSettings {
	if settings == nil {
		return nil
	}

	ws := &model.WorkflowSettings{}

	if order, ok := settings["executionOrder"].(string); ok {
		ws.ExecutionOrder = order
	}
	if tz, ok := settings["timezone"].(string); ok {
		ws.Timezone = tz
	}
	if saveErr, ok := settings["saveDataError"]; ok {
		ws.SaveDataError = saveErr
	}
	if saveSuccess, ok := settings["saveDataSuccess"]; ok {
		ws.SaveDataSuccess = saveSuccess
	}
	if saveManual, ok := settings["saveManualExecutions"]; ok {
		ws.SaveManualExecutions = saveManual
	}

	return ws
}

func (importer *N8nWorkflowImporter) countConnections(connections map[string]N8nConnectionGroup) int {
	count := 0
	for _, connectionGroup := range connections {
		for _, outputConnections := range connectionGroup.Main {
			count += len(outputConnections)
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
	// Convert settings to map
	var settingsMap map[string]interface{}
	if workflow.Settings != nil {
		settingsMap = map[string]interface{}{
			"executionOrder":       workflow.Settings.ExecutionOrder,
			"timezone":             workflow.Settings.Timezone,
			"saveDataError":        workflow.Settings.SaveDataError,
			"saveDataSuccess":      workflow.Settings.SaveDataSuccess,
			"saveManualExecutions": workflow.Settings.SaveManualExecutions,
		}
	}

	n8nWorkflow := N8nWorkflowData{
		ID:          workflow.ID,
		Name:        workflow.Name,
		Active:      workflow.Active,
		Nodes:       []N8nNodeData{},
		Connections: make(map[string]N8nConnectionGroup),
		Settings:    settingsMap,
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
		// Convert credentials
		credMap := make(map[string]string)
		for credType, cred := range node.Credentials {
			credMap[credType] = cred.Name
		}

		// Convert disabled
		disabled := false
		if node.Disabled != nil {
			disabled = *node.Disabled
		}

		n8nNode := N8nNodeData{
			ID:          node.ID,
			Name:        node.Name,
			Type:        node.Type,
			TypeVersion: float64(node.TypeVersion),
			Position:    []float64{float64(node.Position[0]), float64(node.Position[1])},
			Parameters:  node.Parameters,
			Credentials: credMap,
			Disabled:    disabled,
			Notes:       node.Notes,
		}

		n8nWorkflow.Nodes = append(n8nWorkflow.Nodes, n8nNode)
	}

	// Convert connections
	for sourceNodeId, connectionGroup := range workflow.Connections {
		n8nGroup := N8nConnectionGroup{
			Main: make([][]N8nConnection, len(connectionGroup.Main)),
		}

		for outputIndex, outputConnections := range connectionGroup.Main {
			n8nGroup.Main[outputIndex] = make([]N8nConnection, len(outputConnections))
			for i, conn := range outputConnections {
				n8nGroup.Main[outputIndex][i] = N8nConnection{
					Node:        conn.Node,
					Type:        conn.Type,
					Index:       conn.Index,
					OutputIndex: outputIndex,
				}
			}
		}

		n8nWorkflow.Connections[sourceNodeId] = n8nGroup
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

	for sourceId, connectionGroup := range workflow.Connections {
		if !nodeIds[sourceId] {
			return fmt.Errorf("connection references non-existent source node: %s", sourceId)
		}

		for _, outputConnections := range connectionGroup.Main {
			for _, connection := range outputConnections {
				if !nodeIds[connection.Node] {
					return fmt.Errorf("connection references non-existent target node: %s", connection.Node)
				}
			}
		}
	}

	return nil
}