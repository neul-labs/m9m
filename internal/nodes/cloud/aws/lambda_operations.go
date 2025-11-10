package aws

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"

	"github.com/dipankar/n8n-go/internal/expressions"
	"github.com/dipankar/n8n-go/internal/model"
	"github.com/dipankar/n8n-go/internal/nodes/base"
)

// LambdaOperationsNode provides AWS Lambda operations
type LambdaOperationsNode struct {
	*base.BaseNode
	evaluator    *expressions.GojaExpressionEvaluator
	lambdaClient *lambda.Lambda
	session      *session.Session
}

// LambdaConfig holds Lambda configuration
type LambdaConfig struct {
	Region          string `json:"region"`
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	SessionToken    string `json:"sessionToken,omitempty"`
}

// LambdaOperation represents a Lambda operation
type LambdaOperation struct {
	Operation        string                 `json:"operation"`
	FunctionName     string                 `json:"functionName"`
	Payload          interface{}            `json:"payload,omitempty"`
	InvocationType   string                 `json:"invocationType"` // Event, RequestResponse, DryRun
	LogType          string                 `json:"logType"`        // None, Tail
	Qualifier        string                 `json:"qualifier,omitempty"`
	ZipFile          []byte                 `json:"zipFile,omitempty"`
	S3Bucket         string                 `json:"s3Bucket,omitempty"`
	S3Key            string                 `json:"s3Key,omitempty"`
	S3ObjectVersion  string                 `json:"s3ObjectVersion,omitempty"`
	Runtime          string                 `json:"runtime,omitempty"`
	Role             string                 `json:"role,omitempty"`
	Handler          string                 `json:"handler,omitempty"`
	Description      string                 `json:"description,omitempty"`
	Timeout          int64                  `json:"timeout,omitempty"`
	MemorySize       int64                  `json:"memorySize,omitempty"`
	Environment      map[string]string      `json:"environment,omitempty"`
	Tags             map[string]string      `json:"tags,omitempty"`
	VpcConfig        *LambdaVpcConfig       `json:"vpcConfig,omitempty"`
	DeadLetterConfig *LambdaDeadLetterConfig `json:"deadLetterConfig,omitempty"`
	TracingConfig    *LambdaTracingConfig   `json:"tracingConfig,omitempty"`
	Options          map[string]interface{} `json:"options,omitempty"`
}

// LambdaVpcConfig defines VPC configuration
type LambdaVpcConfig struct {
	SubnetIds        []string `json:"subnetIds"`
	SecurityGroupIds []string `json:"securityGroupIds"`
}

// LambdaDeadLetterConfig defines dead letter queue configuration
type LambdaDeadLetterConfig struct {
	TargetArn string `json:"targetArn"`
}

// LambdaTracingConfig defines tracing configuration
type LambdaTracingConfig struct {
	Mode string `json:"mode"` // Active, PassThrough
}

// LambdaResult represents the result of a Lambda operation
type LambdaResult struct {
	Operation     string                 `json:"operation"`
	Success       bool                   `json:"success"`
	FunctionName  string                 `json:"functionName"`
	StatusCode    int64                  `json:"statusCode,omitempty"`
	ExecutedVersion string               `json:"executedVersion,omitempty"`
	Payload       interface{}            `json:"payload,omitempty"`
	LogResult     string                 `json:"logResult,omitempty"`
	FunctionError string                 `json:"functionError,omitempty"`
	Configuration *LambdaConfiguration   `json:"configuration,omitempty"`
	ExecutionTime time.Duration          `json:"executionTime"`
	Error         string                 `json:"error,omitempty"`
}

// LambdaConfiguration represents Lambda function configuration
type LambdaConfiguration struct {
	FunctionName     string            `json:"functionName"`
	FunctionArn      string            `json:"functionArn"`
	Runtime          string            `json:"runtime"`
	Role             string            `json:"role"`
	Handler          string            `json:"handler"`
	CodeSize         int64             `json:"codeSize"`
	Description      string            `json:"description"`
	Timeout          int64             `json:"timeout"`
	MemorySize       int64             `json:"memorySize"`
	LastModified     string            `json:"lastModified"`
	CodeSha256       string            `json:"codeSha256"`
	Version          string            `json:"version"`
	Environment      map[string]string `json:"environment,omitempty"`
	DeadLetterConfig *LambdaDeadLetterConfig `json:"deadLetterConfig,omitempty"`
	TracingConfig    *LambdaTracingConfig `json:"tracingConfig,omitempty"`
	VpcConfig        *LambdaVpcConfig  `json:"vpcConfig,omitempty"`
	Layers           []string          `json:"layers,omitempty"`
	State            string            `json:"state"`
	StateReason      string            `json:"stateReason,omitempty"`
	Tags             map[string]string `json:"tags,omitempty"`
}

// NewLambdaOperationsNode creates a new Lambda operations node
func NewLambdaOperationsNode() *LambdaOperationsNode {
	return &LambdaOperationsNode{
		BaseNode:  base.NewBaseNode("AWS Lambda", "n8n-nodes-base.awsLambda"),
		evaluator: expressions.NewGojaExpressionEvaluator(),
	}
}

// Execute performs Lambda operations
func (n *LambdaOperationsNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	var results []model.DataItem

	// Parse Lambda configuration
	config, err := n.parseLambdaConfig(nodeParams)
	if err != nil {
		return nil, n.CreateError("invalid Lambda configuration", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Initialize Lambda client
	if err := n.initializeLambdaClient(config); err != nil {
		return nil, n.CreateError("failed to initialize Lambda client", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Parse operation configuration
	operation, err := n.parseLambdaOperation(nodeParams)
	if err != nil {
		return nil, n.CreateError("invalid Lambda operation", map[string]interface{}{
			"error": err.Error(),
		})
	}

	for index, item := range inputData {
		// Create expression context
		context := &expressions.ExpressionContext{
			ActiveNodeName:      "AWS Lambda",
			RunIndex:           0,
			ItemIndex:          index,
			Mode:               expressions.ModeManual,
			ConnectionInputData: []model.DataItem{item},
			AdditionalKeys:     make(map[string]interface{}),
		}

		// Execute Lambda operation
		result, err := n.executeLambdaOperation(operation, context, item)
		if err != nil {
			return nil, n.CreateError("Lambda operation failed", map[string]interface{}{
				"operation": operation.Operation,
				"error":     err.Error(),
			})
		}

		// Create result item
		resultItem := model.DataItem{
			JSON: result,
		}

		results = append(results, resultItem)
	}

	return results, nil
}

// executeLambdaOperation executes the specified Lambda operation
func (n *LambdaOperationsNode) executeLambdaOperation(operation *LambdaOperation, context *expressions.ExpressionContext, item model.DataItem) (*LambdaResult, error) {
	startTime := time.Now()

	result := &LambdaResult{
		Operation:    operation.Operation,
		FunctionName: operation.FunctionName,
	}

	var err error

	switch strings.ToLower(operation.Operation) {
	case "invoke":
		err = n.invokeFunction(operation, context, item, result)
	case "create":
		err = n.createFunction(operation, context, item, result)
	case "update":
		err = n.updateFunction(operation, context, item, result)
	case "delete":
		err = n.deleteFunction(operation, context, item, result)
	case "get":
		err = n.getFunction(operation, context, item, result)
	case "list":
		err = n.listFunctions(operation, context, item, result)
	case "update_code":
		err = n.updateFunctionCode(operation, context, item, result)
	case "update_configuration":
		err = n.updateFunctionConfiguration(operation, context, item, result)
	case "create_alias":
		err = n.createAlias(operation, context, item, result)
	case "update_alias":
		err = n.updateAlias(operation, context, item, result)
	case "delete_alias":
		err = n.deleteAlias(operation, context, item, result)
	case "list_aliases":
		err = n.listAliases(operation, context, item, result)
	case "publish_version":
		err = n.publishVersion(operation, context, item, result)
	case "list_versions":
		err = n.listVersions(operation, context, item, result)
	default:
		err = fmt.Errorf("unsupported operation: %s", operation.Operation)
	}

	result.ExecutionTime = time.Since(startTime)
	result.Success = err == nil

	if err != nil {
		result.Error = err.Error()
	}

	return result, err
}

// Invoke operations
func (n *LambdaOperationsNode) invokeFunction(operation *LambdaOperation, context *expressions.ExpressionContext, item model.DataItem, result *LambdaResult) error {
	// Evaluate function name
	functionName, err := n.evaluateExpression(operation.FunctionName, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate function name: %w", err)
	}
	result.FunctionName = fmt.Sprintf("%v", functionName)

	// Prepare payload
	var payloadBytes []byte
	if operation.Payload != nil {
		payload, err := n.evaluateExpression(fmt.Sprintf("%v", operation.Payload), context)
		if err != nil {
			return fmt.Errorf("failed to evaluate payload: %w", err)
		}

		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
	} else {
		// Use item data as payload
		payloadBytes, err = json.Marshal(item.JSON)
		if err != nil {
			return fmt.Errorf("failed to marshal item data: %w", err)
		}
	}

	// Prepare invoke input
	invokeInput := &lambda.InvokeInput{
		FunctionName: aws.String(result.FunctionName),
		Payload:      payloadBytes,
	}

	// Set invocation type
	if operation.InvocationType != "" {
		invokeInput.InvocationType = aws.String(operation.InvocationType)
	} else {
		invokeInput.InvocationType = aws.String("RequestResponse")
	}

	// Set log type
	if operation.LogType != "" {
		invokeInput.LogType = aws.String(operation.LogType)
	}

	// Set qualifier (version or alias)
	if operation.Qualifier != "" {
		invokeInput.Qualifier = aws.String(operation.Qualifier)
	}

	// Invoke function
	invokeResult, err := n.lambdaClient.Invoke(invokeInput)
	if err != nil {
		return fmt.Errorf("function invocation failed: %w", err)
	}

	// Process result
	if invokeResult.StatusCode != nil {
		result.StatusCode = *invokeResult.StatusCode
	}

	if invokeResult.ExecutedVersion != nil {
		result.ExecutedVersion = *invokeResult.ExecutedVersion
	}

	if invokeResult.FunctionError != nil {
		result.FunctionError = *invokeResult.FunctionError
	}

	// Process payload
	if invokeResult.Payload != nil {
		var payload interface{}
		if err := json.Unmarshal(invokeResult.Payload, &payload); err != nil {
			// If JSON unmarshal fails, return as string
			result.Payload = string(invokeResult.Payload)
		} else {
			result.Payload = payload
		}
	}

	// Process log result
	if invokeResult.LogResult != nil {
		logData, err := base64.StdEncoding.DecodeString(*invokeResult.LogResult)
		if err == nil {
			result.LogResult = string(logData)
		} else {
			result.LogResult = *invokeResult.LogResult
		}
	}

	return nil
}

// Function management operations
func (n *LambdaOperationsNode) createFunction(operation *LambdaOperation, context *expressions.ExpressionContext, item model.DataItem, result *LambdaResult) error {
	// Evaluate function name
	functionName, err := n.evaluateExpression(operation.FunctionName, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate function name: %w", err)
	}
	result.FunctionName = fmt.Sprintf("%v", functionName)

	// Prepare create function input
	createInput := &lambda.CreateFunctionInput{
		FunctionName: aws.String(result.FunctionName),
		Runtime:      aws.String(operation.Runtime),
		Role:         aws.String(operation.Role),
		Handler:      aws.String(operation.Handler),
	}

	// Set code source
	if len(operation.ZipFile) > 0 {
		createInput.Code = &lambda.FunctionCode{
			ZipFile: operation.ZipFile,
		}
	} else if operation.S3Bucket != "" && operation.S3Key != "" {
		createInput.Code = &lambda.FunctionCode{
			S3Bucket: aws.String(operation.S3Bucket),
			S3Key:    aws.String(operation.S3Key),
		}
		if operation.S3ObjectVersion != "" {
			createInput.Code.S3ObjectVersion = aws.String(operation.S3ObjectVersion)
		}
	} else {
		return fmt.Errorf("either zipFile or S3 location must be provided")
	}

	// Set optional parameters
	if operation.Description != "" {
		createInput.Description = aws.String(operation.Description)
	}

	if operation.Timeout > 0 {
		createInput.Timeout = aws.Int64(operation.Timeout)
	}

	if operation.MemorySize > 0 {
		createInput.MemorySize = aws.Int64(operation.MemorySize)
	}

	// Set environment variables
	if len(operation.Environment) > 0 {
		env := &lambda.Environment{
			Variables: make(map[string]*string),
		}
		for key, value := range operation.Environment {
			env.Variables[key] = aws.String(value)
		}
		createInput.Environment = env
	}

	// Set VPC configuration
	if operation.VpcConfig != nil {
		vpcConfig := &lambda.VpcConfig{
			SubnetIds:        aws.StringSlice(operation.VpcConfig.SubnetIds),
			SecurityGroupIds: aws.StringSlice(operation.VpcConfig.SecurityGroupIds),
		}
		createInput.VpcConfig = vpcConfig
	}

	// Set dead letter configuration
	if operation.DeadLetterConfig != nil {
		createInput.DeadLetterConfig = &lambda.DeadLetterConfig{
			TargetArn: aws.String(operation.DeadLetterConfig.TargetArn),
		}
	}

	// Set tracing configuration
	if operation.TracingConfig != nil {
		createInput.TracingConfig = &lambda.TracingConfig{
			Mode: aws.String(operation.TracingConfig.Mode),
		}
	}

	// Set tags
	if len(operation.Tags) > 0 {
		tags := make(map[string]*string)
		for key, value := range operation.Tags {
			tags[key] = aws.String(value)
		}
		createInput.Tags = tags
	}

	// Create function
	createResult, err := n.lambdaClient.CreateFunction(createInput)
	if err != nil {
		return fmt.Errorf("function creation failed: %w", err)
	}

	// Convert result to our configuration format
	result.Configuration = n.convertToLambdaConfiguration(createResult)

	return nil
}

func (n *LambdaOperationsNode) updateFunction(operation *LambdaOperation, context *expressions.ExpressionContext, item model.DataItem, result *LambdaResult) error {
	// First update code if provided
	if len(operation.ZipFile) > 0 || (operation.S3Bucket != "" && operation.S3Key != "") {
		if err := n.updateFunctionCode(operation, context, item, result); err != nil {
			return err
		}
	}

	// Then update configuration
	return n.updateFunctionConfiguration(operation, context, item, result)
}

func (n *LambdaOperationsNode) updateFunctionCode(operation *LambdaOperation, context *expressions.ExpressionContext, item model.DataItem, result *LambdaResult) error {
	// Evaluate function name
	functionName, err := n.evaluateExpression(operation.FunctionName, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate function name: %w", err)
	}
	result.FunctionName = fmt.Sprintf("%v", functionName)

	// Prepare update code input
	updateInput := &lambda.UpdateFunctionCodeInput{
		FunctionName: aws.String(result.FunctionName),
	}

	// Set code source
	if len(operation.ZipFile) > 0 {
		updateInput.ZipFile = operation.ZipFile
	} else if operation.S3Bucket != "" && operation.S3Key != "" {
		updateInput.S3Bucket = aws.String(operation.S3Bucket)
		updateInput.S3Key = aws.String(operation.S3Key)
		if operation.S3ObjectVersion != "" {
			updateInput.S3ObjectVersion = aws.String(operation.S3ObjectVersion)
		}
	} else {
		return fmt.Errorf("either zipFile or S3 location must be provided")
	}

	// Update function code
	updateResult, err := n.lambdaClient.UpdateFunctionCode(updateInput)
	if err != nil {
		return fmt.Errorf("function code update failed: %w", err)
	}

	// Convert result to our configuration format
	result.Configuration = n.convertToLambdaConfiguration(updateResult)

	return nil
}

func (n *LambdaOperationsNode) updateFunctionConfiguration(operation *LambdaOperation, context *expressions.ExpressionContext, item model.DataItem, result *LambdaResult) error {
	// Evaluate function name
	functionName, err := n.evaluateExpression(operation.FunctionName, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate function name: %w", err)
	}
	result.FunctionName = fmt.Sprintf("%v", functionName)

	// Prepare update configuration input
	updateInput := &lambda.UpdateFunctionConfigurationInput{
		FunctionName: aws.String(result.FunctionName),
	}

	// Set optional parameters
	if operation.Description != "" {
		updateInput.Description = aws.String(operation.Description)
	}

	if operation.Handler != "" {
		updateInput.Handler = aws.String(operation.Handler)
	}

	if operation.Runtime != "" {
		updateInput.Runtime = aws.String(operation.Runtime)
	}

	if operation.Role != "" {
		updateInput.Role = aws.String(operation.Role)
	}

	if operation.Timeout > 0 {
		updateInput.Timeout = aws.Int64(operation.Timeout)
	}

	if operation.MemorySize > 0 {
		updateInput.MemorySize = aws.Int64(operation.MemorySize)
	}

	// Set environment variables
	if len(operation.Environment) > 0 {
		env := &lambda.Environment{
			Variables: make(map[string]*string),
		}
		for key, value := range operation.Environment {
			env.Variables[key] = aws.String(value)
		}
		updateInput.Environment = env
	}

	// Set VPC configuration
	if operation.VpcConfig != nil {
		vpcConfig := &lambda.VpcConfig{
			SubnetIds:        aws.StringSlice(operation.VpcConfig.SubnetIds),
			SecurityGroupIds: aws.StringSlice(operation.VpcConfig.SecurityGroupIds),
		}
		updateInput.VpcConfig = vpcConfig
	}

	// Set dead letter configuration
	if operation.DeadLetterConfig != nil {
		updateInput.DeadLetterConfig = &lambda.DeadLetterConfig{
			TargetArn: aws.String(operation.DeadLetterConfig.TargetArn),
		}
	}

	// Set tracing configuration
	if operation.TracingConfig != nil {
		updateInput.TracingConfig = &lambda.TracingConfig{
			Mode: aws.String(operation.TracingConfig.Mode),
		}
	}

	// Update function configuration
	updateResult, err := n.lambdaClient.UpdateFunctionConfiguration(updateInput)
	if err != nil {
		return fmt.Errorf("function configuration update failed: %w", err)
	}

	// Convert result to our configuration format
	result.Configuration = n.convertToLambdaConfiguration(updateResult)

	return nil
}

func (n *LambdaOperationsNode) deleteFunction(operation *LambdaOperation, context *expressions.ExpressionContext, item model.DataItem, result *LambdaResult) error {
	// Evaluate function name
	functionName, err := n.evaluateExpression(operation.FunctionName, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate function name: %w", err)
	}
	result.FunctionName = fmt.Sprintf("%v", functionName)

	// Prepare delete input
	deleteInput := &lambda.DeleteFunctionInput{
		FunctionName: aws.String(result.FunctionName),
	}

	if operation.Qualifier != "" {
		deleteInput.Qualifier = aws.String(operation.Qualifier)
	}

	// Delete function
	_, err = n.lambdaClient.DeleteFunction(deleteInput)
	if err != nil {
		return fmt.Errorf("function deletion failed: %w", err)
	}

	return nil
}

func (n *LambdaOperationsNode) getFunction(operation *LambdaOperation, context *expressions.ExpressionContext, item model.DataItem, result *LambdaResult) error {
	// Evaluate function name
	functionName, err := n.evaluateExpression(operation.FunctionName, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate function name: %w", err)
	}
	result.FunctionName = fmt.Sprintf("%v", functionName)

	// Prepare get function input
	getInput := &lambda.GetFunctionInput{
		FunctionName: aws.String(result.FunctionName),
	}

	if operation.Qualifier != "" {
		getInput.Qualifier = aws.String(operation.Qualifier)
	}

	// Get function
	getResult, err := n.lambdaClient.GetFunction(getInput)
	if err != nil {
		return fmt.Errorf("get function failed: %w", err)
	}

	// Convert result to our configuration format
	if getResult.Configuration != nil {
		result.Configuration = n.convertToLambdaConfiguration(getResult.Configuration)
	}

	return nil
}

func (n *LambdaOperationsNode) listFunctions(operation *LambdaOperation, context *expressions.ExpressionContext, item model.DataItem, result *LambdaResult) error {
	// Prepare list functions input
	listInput := &lambda.ListFunctionsInput{}

	// Handle pagination
	if maxItems, ok := operation.Options["maxItems"].(int); ok {
		listInput.MaxItems = aws.Int64(int64(maxItems))
	}

	if marker, ok := operation.Options["marker"].(string); ok {
		listInput.Marker = aws.String(marker)
	}

	if masterRegion, ok := operation.Options["masterRegion"].(string); ok {
		listInput.MasterRegion = aws.String(masterRegion)
	}

	// List functions
	var functions []*LambdaConfiguration

	err := n.lambdaClient.ListFunctionsPages(listInput, func(page *lambda.ListFunctionsOutput, lastPage bool) bool {
		for _, function := range page.Functions {
			functions = append(functions, n.convertToLambdaConfiguration(function))
		}
		return !lastPage
	})

	if err != nil {
		return fmt.Errorf("list functions failed: %w", err)
	}

	result.Payload = functions
	return nil
}

// Alias operations
func (n *LambdaOperationsNode) createAlias(operation *LambdaOperation, context *expressions.ExpressionContext, item model.DataItem, result *LambdaResult) error {
	aliasName, ok := operation.Options["aliasName"].(string)
	if !ok {
		return fmt.Errorf("aliasName is required for create alias operation")
	}

	functionVersion, ok := operation.Options["functionVersion"].(string)
	if !ok {
		return fmt.Errorf("functionVersion is required for create alias operation")
	}

	// Evaluate function name
	functionName, err := n.evaluateExpression(operation.FunctionName, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate function name: %w", err)
	}
	result.FunctionName = fmt.Sprintf("%v", functionName)

	// Prepare create alias input
	createInput := &lambda.CreateAliasInput{
		FunctionName:    aws.String(result.FunctionName),
		Name:            aws.String(aliasName),
		FunctionVersion: aws.String(functionVersion),
	}

	if description, ok := operation.Options["description"].(string); ok {
		createInput.Description = aws.String(description)
	}

	// Create alias
	createResult, err := n.lambdaClient.CreateAlias(createInput)
	if err != nil {
		return fmt.Errorf("create alias failed: %w", err)
	}

	result.Payload = map[string]interface{}{
		"aliasArn":         *createResult.AliasArn,
		"name":             *createResult.Name,
		"functionVersion":  *createResult.FunctionVersion,
		"description":      aws.StringValue(createResult.Description),
		"revisionId":       aws.StringValue(createResult.RevisionId),
	}

	return nil
}

func (n *LambdaOperationsNode) updateAlias(operation *LambdaOperation, context *expressions.ExpressionContext, item model.DataItem, result *LambdaResult) error {
	aliasName, ok := operation.Options["aliasName"].(string)
	if !ok {
		return fmt.Errorf("aliasName is required for update alias operation")
	}

	// Evaluate function name
	functionName, err := n.evaluateExpression(operation.FunctionName, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate function name: %w", err)
	}
	result.FunctionName = fmt.Sprintf("%v", functionName)

	// Prepare update alias input
	updateInput := &lambda.UpdateAliasInput{
		FunctionName: aws.String(result.FunctionName),
		Name:         aws.String(aliasName),
	}

	if functionVersion, ok := operation.Options["functionVersion"].(string); ok {
		updateInput.FunctionVersion = aws.String(functionVersion)
	}

	if description, ok := operation.Options["description"].(string); ok {
		updateInput.Description = aws.String(description)
	}

	// Update alias
	updateResult, err := n.lambdaClient.UpdateAlias(updateInput)
	if err != nil {
		return fmt.Errorf("update alias failed: %w", err)
	}

	result.Payload = map[string]interface{}{
		"aliasArn":         *updateResult.AliasArn,
		"name":             *updateResult.Name,
		"functionVersion":  *updateResult.FunctionVersion,
		"description":      aws.StringValue(updateResult.Description),
		"revisionId":       aws.StringValue(updateResult.RevisionId),
	}

	return nil
}

func (n *LambdaOperationsNode) deleteAlias(operation *LambdaOperation, context *expressions.ExpressionContext, item model.DataItem, result *LambdaResult) error {
	aliasName, ok := operation.Options["aliasName"].(string)
	if !ok {
		return fmt.Errorf("aliasName is required for delete alias operation")
	}

	// Evaluate function name
	functionName, err := n.evaluateExpression(operation.FunctionName, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate function name: %w", err)
	}
	result.FunctionName = fmt.Sprintf("%v", functionName)

	// Prepare delete alias input
	deleteInput := &lambda.DeleteAliasInput{
		FunctionName: aws.String(result.FunctionName),
		Name:         aws.String(aliasName),
	}

	// Delete alias
	_, err = n.lambdaClient.DeleteAlias(deleteInput)
	if err != nil {
		return fmt.Errorf("delete alias failed: %w", err)
	}

	return nil
}

func (n *LambdaOperationsNode) listAliases(operation *LambdaOperation, context *expressions.ExpressionContext, item model.DataItem, result *LambdaResult) error {
	// Evaluate function name
	functionName, err := n.evaluateExpression(operation.FunctionName, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate function name: %w", err)
	}
	result.FunctionName = fmt.Sprintf("%v", functionName)

	// Prepare list aliases input
	listInput := &lambda.ListAliasesInput{
		FunctionName: aws.String(result.FunctionName),
	}

	if functionVersion, ok := operation.Options["functionVersion"].(string); ok {
		listInput.FunctionVersion = aws.String(functionVersion)
	}

	// List aliases
	listResult, err := n.lambdaClient.ListAliases(listInput)
	if err != nil {
		return fmt.Errorf("list aliases failed: %w", err)
	}

	var aliases []map[string]interface{}
	for _, alias := range listResult.Aliases {
		aliases = append(aliases, map[string]interface{}{
			"aliasArn":         *alias.AliasArn,
			"name":             *alias.Name,
			"functionVersion":  *alias.FunctionVersion,
			"description":      aws.StringValue(alias.Description),
			"revisionId":       aws.StringValue(alias.RevisionId),
		})
	}

	result.Payload = aliases
	return nil
}

// Version operations
func (n *LambdaOperationsNode) publishVersion(operation *LambdaOperation, context *expressions.ExpressionContext, item model.DataItem, result *LambdaResult) error {
	// Evaluate function name
	functionName, err := n.evaluateExpression(operation.FunctionName, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate function name: %w", err)
	}
	result.FunctionName = fmt.Sprintf("%v", functionName)

	// Prepare publish version input
	publishInput := &lambda.PublishVersionInput{
		FunctionName: aws.String(result.FunctionName),
	}

	if description, ok := operation.Options["description"].(string); ok {
		publishInput.Description = aws.String(description)
	}

	// Publish version
	publishResult, err := n.lambdaClient.PublishVersion(publishInput)
	if err != nil {
		return fmt.Errorf("publish version failed: %w", err)
	}

	// Convert result to our configuration format
	result.Configuration = n.convertToLambdaConfiguration(publishResult)

	return nil
}

func (n *LambdaOperationsNode) listVersions(operation *LambdaOperation, context *expressions.ExpressionContext, item model.DataItem, result *LambdaResult) error {
	// Evaluate function name
	functionName, err := n.evaluateExpression(operation.FunctionName, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate function name: %w", err)
	}
	result.FunctionName = fmt.Sprintf("%v", functionName)

	// Prepare list versions input
	listInput := &lambda.ListVersionsByFunctionInput{
		FunctionName: aws.String(result.FunctionName),
	}

	// List versions
	var versions []*LambdaConfiguration

	err = n.lambdaClient.ListVersionsByFunctionPages(listInput, func(page *lambda.ListVersionsByFunctionOutput, lastPage bool) bool {
		for _, version := range page.Versions {
			versions = append(versions, n.convertToLambdaConfiguration(version))
		}
		return !lastPage
	})

	if err != nil {
		return fmt.Errorf("list versions failed: %w", err)
	}

	result.Payload = versions
	return nil
}

// Helper methods
func (n *LambdaOperationsNode) convertToLambdaConfiguration(function *lambda.FunctionConfiguration) *LambdaConfiguration {
	config := &LambdaConfiguration{
		FunctionName: aws.StringValue(function.FunctionName),
		FunctionArn:  aws.StringValue(function.FunctionArn),
		Runtime:      aws.StringValue(function.Runtime),
		Role:         aws.StringValue(function.Role),
		Handler:      aws.StringValue(function.Handler),
		CodeSize:     aws.Int64Value(function.CodeSize),
		Description:  aws.StringValue(function.Description),
		Timeout:      aws.Int64Value(function.Timeout),
		MemorySize:   aws.Int64Value(function.MemorySize),
		LastModified: aws.StringValue(function.LastModified),
		CodeSha256:   aws.StringValue(function.CodeSha256),
		Version:      aws.StringValue(function.Version),
		State:        aws.StringValue(function.State),
		StateReason:  aws.StringValue(function.StateReason),
	}

	// Convert environment variables
	if function.Environment != nil && function.Environment.Variables != nil {
		config.Environment = make(map[string]string)
		for key, value := range function.Environment.Variables {
			config.Environment[key] = aws.StringValue(value)
		}
	}

	// Convert VPC configuration
	if function.VpcConfig != nil {
		config.VpcConfig = &LambdaVpcConfig{
			SubnetIds:        aws.StringValueSlice(function.VpcConfig.SubnetIds),
			SecurityGroupIds: aws.StringValueSlice(function.VpcConfig.SecurityGroupIds),
		}
	}

	// Convert dead letter configuration
	if function.DeadLetterConfig != nil {
		config.DeadLetterConfig = &LambdaDeadLetterConfig{
			TargetArn: aws.StringValue(function.DeadLetterConfig.TargetArn),
		}
	}

	// Convert tracing configuration
	if function.TracingConfig != nil {
		config.TracingConfig = &LambdaTracingConfig{
			Mode: aws.StringValue(function.TracingConfig.Mode),
		}
	}

	// Convert layers
	if function.Layers != nil {
		config.Layers = make([]string, len(function.Layers))
		for i, layer := range function.Layers {
			config.Layers[i] = aws.StringValue(layer.Arn)
		}
	}

	return config
}

// Configuration parsing methods
func (n *LambdaOperationsNode) parseLambdaConfig(nodeParams map[string]interface{}) (*LambdaConfig, error) {
	config := &LambdaConfig{}

	if region, ok := nodeParams["region"].(string); ok {
		config.Region = region
	} else {
		config.Region = "us-east-1" // Default region
	}

	if accessKeyID, ok := nodeParams["accessKeyId"].(string); ok {
		config.AccessKeyID = accessKeyID
	}

	if secretAccessKey, ok := nodeParams["secretAccessKey"].(string); ok {
		config.SecretAccessKey = secretAccessKey
	}

	if sessionToken, ok := nodeParams["sessionToken"].(string); ok {
		config.SessionToken = sessionToken
	}

	return config, nil
}

func (n *LambdaOperationsNode) parseLambdaOperation(nodeParams map[string]interface{}) (*LambdaOperation, error) {
	operation := &LambdaOperation{
		Options: make(map[string]interface{}),
	}

	if op, ok := nodeParams["operation"].(string); ok {
		operation.Operation = op
	} else {
		return nil, fmt.Errorf("operation parameter is required")
	}

	if functionName, ok := nodeParams["functionName"].(string); ok {
		operation.FunctionName = functionName
	} else {
		return nil, fmt.Errorf("functionName parameter is required")
	}

	if payload, ok := nodeParams["payload"]; ok {
		operation.Payload = payload
	}

	if invocationType, ok := nodeParams["invocationType"].(string); ok {
		operation.InvocationType = invocationType
	}

	if logType, ok := nodeParams["logType"].(string); ok {
		operation.LogType = logType
	}

	if qualifier, ok := nodeParams["qualifier"].(string); ok {
		operation.Qualifier = qualifier
	}

	if runtime, ok := nodeParams["runtime"].(string); ok {
		operation.Runtime = runtime
	}

	if role, ok := nodeParams["role"].(string); ok {
		operation.Role = role
	}

	if handler, ok := nodeParams["handler"].(string); ok {
		operation.Handler = handler
	}

	if description, ok := nodeParams["description"].(string); ok {
		operation.Description = description
	}

	if timeout, ok := nodeParams["timeout"].(int); ok {
		operation.Timeout = int64(timeout)
	}

	if memorySize, ok := nodeParams["memorySize"].(int); ok {
		operation.MemorySize = int64(memorySize)
	}

	// Parse environment variables
	if environment, ok := nodeParams["environment"].(map[string]interface{}); ok {
		operation.Environment = make(map[string]string)
		for k, v := range environment {
			operation.Environment[k] = fmt.Sprintf("%v", v)
		}
	}

	// Parse tags
	if tags, ok := nodeParams["tags"].(map[string]interface{}); ok {
		operation.Tags = make(map[string]string)
		for k, v := range tags {
			operation.Tags[k] = fmt.Sprintf("%v", v)
		}
	}

	// Parse options
	if options, ok := nodeParams["options"].(map[string]interface{}); ok {
		for k, v := range options {
			operation.Options[k] = v
		}
	}

	return operation, nil
}

func (n *LambdaOperationsNode) initializeLambdaClient(config *LambdaConfig) error {
	// Create AWS config
	awsConfig := &aws.Config{
		Region: aws.String(config.Region),
	}

	// Set credentials
	if config.AccessKeyID != "" && config.SecretAccessKey != "" {
		awsConfig.Credentials = credentials.NewStaticCredentials(
			config.AccessKeyID,
			config.SecretAccessKey,
			config.SessionToken,
		)
	}

	// Create session
	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return fmt.Errorf("failed to create AWS session: %w", err)
	}

	n.session = sess
	n.lambdaClient = lambda.New(sess)

	return nil
}

func (n *LambdaOperationsNode) evaluateExpression(expr string, context *expressions.ExpressionContext) (interface{}, error) {
	return n.evaluator.EvaluateExpression(expr, context)
}

// ValidateParameters validates the node parameters
func (n *LambdaOperationsNode) ValidateParameters(params map[string]interface{}) error {
	// Validate required parameters
	if _, ok := params["operation"]; !ok {
		return fmt.Errorf("operation parameter is required")
	}

	if _, ok := params["functionName"]; !ok {
		return fmt.Errorf("functionName parameter is required")
	}

	// Validate operation
	if operation, ok := params["operation"].(string); ok {
		validOperations := []string{
			"invoke", "create", "update", "delete", "get", "list",
			"update_code", "update_configuration", "create_alias",
			"update_alias", "delete_alias", "list_aliases",
			"publish_version", "list_versions",
		}

		valid := false
		for _, validOp := range validOperations {
			if operation == validOp {
				valid = true
				break
			}
		}

		if !valid {
			return fmt.Errorf("invalid operation: %s. Valid operations: %v", operation, validOperations)
		}
	}

	return nil
}