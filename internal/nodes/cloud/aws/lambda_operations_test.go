package aws

import (
	"testing"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Constructor Tests ---

func TestNewLambdaOperationsNode_ReturnsNonNil(t *testing.T) {
	node := NewLambdaOperationsNode()
	require.NotNil(t, node)
	require.NotNil(t, node.BaseNode)
	require.NotNil(t, node.evaluator)
}

// --- Description Tests ---

func TestLambdaOperationsNode_Description_Name(t *testing.T) {
	node := NewLambdaOperationsNode()
	desc := node.Description()
	assert.Equal(t, "AWS Lambda", desc.Name)
}

func TestLambdaOperationsNode_Description_Category(t *testing.T) {
	node := NewLambdaOperationsNode()
	desc := node.Description()
	assert.Equal(t, "cloud", desc.Category)
}

func TestLambdaOperationsNode_Description_DescriptionField(t *testing.T) {
	node := NewLambdaOperationsNode()
	desc := node.Description()
	assert.Equal(t, "AWS Lambda function operations", desc.Description)
}

// --- ValidateParameters Tests ---

func TestLambdaOperationsNode_ValidateParameters_ValidInvoke(t *testing.T) {
	node := NewLambdaOperationsNode()
	params := map[string]interface{}{
		"operation":    "invoke",
		"functionName": "my-function",
	}
	err := node.ValidateParameters(params)
	assert.NoError(t, err)
}

func TestLambdaOperationsNode_ValidateParameters_AllOperations(t *testing.T) {
	node := NewLambdaOperationsNode()

	validOperations := []string{
		"invoke", "create", "update", "delete", "get", "list",
		"update_code", "update_configuration", "create_alias",
		"update_alias", "delete_alias", "list_aliases",
		"publish_version", "list_versions",
	}

	for _, op := range validOperations {
		t.Run(op, func(t *testing.T) {
			params := map[string]interface{}{
				"operation":    op,
				"functionName": "test-function",
			}
			err := node.ValidateParameters(params)
			assert.NoError(t, err, "operation %q should be valid", op)
		})
	}
}

func TestLambdaOperationsNode_ValidateParameters_MissingOperation(t *testing.T) {
	node := NewLambdaOperationsNode()
	params := map[string]interface{}{
		"functionName": "my-function",
	}
	err := node.ValidateParameters(params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "operation parameter is required")
}

func TestLambdaOperationsNode_ValidateParameters_MissingFunctionName(t *testing.T) {
	node := NewLambdaOperationsNode()
	params := map[string]interface{}{
		"operation": "invoke",
	}
	err := node.ValidateParameters(params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "functionName parameter is required")
}

func TestLambdaOperationsNode_ValidateParameters_InvalidOperation(t *testing.T) {
	node := NewLambdaOperationsNode()
	params := map[string]interface{}{
		"operation":    "nonexistent_operation",
		"functionName": "my-function",
	}
	err := node.ValidateParameters(params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid operation")
}

func TestLambdaOperationsNode_ValidateParameters_EmptyParams(t *testing.T) {
	node := NewLambdaOperationsNode()
	params := map[string]interface{}{}
	err := node.ValidateParameters(params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "operation parameter is required")
}

func TestLambdaOperationsNode_ValidateParameters_OperationNotString(t *testing.T) {
	node := NewLambdaOperationsNode()
	// operation exists but is not a string; validation should still pass the presence check
	// but the type assertion for valid operations will not match, so it skips the validation block
	params := map[string]interface{}{
		"operation":    123,
		"functionName": "my-function",
	}
	err := node.ValidateParameters(params)
	// The key exists so the "required" check passes,
	// but the type assertion to string fails so the valid-operations loop is skipped.
	assert.NoError(t, err)
}

// --- parseLambdaConfig Tests ---

func TestLambdaOperationsNode_ParseLambdaConfig_Defaults(t *testing.T) {
	node := NewLambdaOperationsNode()
	params := map[string]interface{}{}
	config, err := node.parseLambdaConfig(params)
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, "us-east-1", config.Region, "default region should be us-east-1")
	assert.Empty(t, config.AccessKeyID)
	assert.Empty(t, config.SecretAccessKey)
	assert.Empty(t, config.SessionToken)
}

func TestLambdaOperationsNode_ParseLambdaConfig_WithCredentials(t *testing.T) {
	node := NewLambdaOperationsNode()
	params := map[string]interface{}{
		"region":          "eu-west-1",
		"accessKeyId":     "AKIAIOSFODNN7EXAMPLE",
		"secretAccessKey": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		"sessionToken":    "FwoGZXIvYXdzEBY",
	}
	config, err := node.parseLambdaConfig(params)
	require.NoError(t, err)
	assert.Equal(t, "eu-west-1", config.Region)
	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", config.AccessKeyID)
	assert.Equal(t, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", config.SecretAccessKey)
	assert.Equal(t, "FwoGZXIvYXdzEBY", config.SessionToken)
}

func TestLambdaOperationsNode_ParseLambdaConfig_PartialCredentials(t *testing.T) {
	node := NewLambdaOperationsNode()
	params := map[string]interface{}{
		"accessKeyId": "AKIAIOSFODNN7EXAMPLE",
	}
	config, err := node.parseLambdaConfig(params)
	require.NoError(t, err)
	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", config.AccessKeyID)
	assert.Empty(t, config.SecretAccessKey)
}

// --- parseLambdaOperation Tests ---

func TestLambdaOperationsNode_ParseLambdaOperation_Basic(t *testing.T) {
	node := NewLambdaOperationsNode()
	params := map[string]interface{}{
		"operation":    "invoke",
		"functionName": "my-function",
	}
	op, err := node.parseLambdaOperation(params)
	require.NoError(t, err)
	require.NotNil(t, op)
	assert.Equal(t, "invoke", op.Operation)
	assert.Equal(t, "my-function", op.FunctionName)
}

func TestLambdaOperationsNode_ParseLambdaOperation_MissingOperation(t *testing.T) {
	node := NewLambdaOperationsNode()
	params := map[string]interface{}{
		"functionName": "my-function",
	}
	op, err := node.parseLambdaOperation(params)
	require.Error(t, err)
	assert.Nil(t, op)
	assert.Contains(t, err.Error(), "operation parameter is required")
}

func TestLambdaOperationsNode_ParseLambdaOperation_MissingFunctionName(t *testing.T) {
	node := NewLambdaOperationsNode()
	params := map[string]interface{}{
		"operation": "invoke",
	}
	op, err := node.parseLambdaOperation(params)
	require.Error(t, err)
	assert.Nil(t, op)
	assert.Contains(t, err.Error(), "functionName parameter is required")
}

func TestLambdaOperationsNode_ParseLambdaOperation_AllFields(t *testing.T) {
	node := NewLambdaOperationsNode()
	params := map[string]interface{}{
		"operation":      "create",
		"functionName":   "new-function",
		"payload":        `{"key":"value"}`,
		"invocationType": "Event",
		"logType":        "Tail",
		"qualifier":      "$LATEST",
		"runtime":        "go1.x",
		"role":           "arn:aws:iam::123456789012:role/lambda-role",
		"handler":        "main",
		"description":    "A test function",
		"timeout":        30,
		"memorySize":     256,
		"environment": map[string]interface{}{
			"ENV_VAR": "value",
		},
		"tags": map[string]interface{}{
			"team": "platform",
		},
		"options": map[string]interface{}{
			"aliasName": "prod",
		},
	}
	op, err := node.parseLambdaOperation(params)
	require.NoError(t, err)
	require.NotNil(t, op)

	assert.Equal(t, "create", op.Operation)
	assert.Equal(t, "new-function", op.FunctionName)
	assert.Equal(t, `{"key":"value"}`, op.Payload)
	assert.Equal(t, "Event", op.InvocationType)
	assert.Equal(t, "Tail", op.LogType)
	assert.Equal(t, "$LATEST", op.Qualifier)
	assert.Equal(t, "go1.x", op.Runtime)
	assert.Equal(t, "arn:aws:iam::123456789012:role/lambda-role", op.Role)
	assert.Equal(t, "main", op.Handler)
	assert.Equal(t, "A test function", op.Description)
	assert.Equal(t, int64(30), op.Timeout)
	assert.Equal(t, int64(256), op.MemorySize)
	assert.Equal(t, "value", op.Environment["ENV_VAR"])
	assert.Equal(t, "platform", op.Tags["team"])
	assert.Equal(t, "prod", op.Options["aliasName"])
}

func TestLambdaOperationsNode_ParseLambdaOperation_OptionsInitialized(t *testing.T) {
	node := NewLambdaOperationsNode()
	params := map[string]interface{}{
		"operation":    "invoke",
		"functionName": "my-function",
	}
	op, err := node.parseLambdaOperation(params)
	require.NoError(t, err)
	require.NotNil(t, op.Options, "Options map should be initialized even when not provided")
}

// --- Execute Error Path Tests ---

func TestLambdaOperationsNode_Execute_MissingOperationParam(t *testing.T) {
	node := NewLambdaOperationsNode()
	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"test": "data"}},
	}
	// Missing "operation" causes parseLambdaOperation to fail
	params := map[string]interface{}{
		"functionName": "my-function",
		"region":       "us-east-1",
	}
	result, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "operation")
}

func TestLambdaOperationsNode_Execute_MissingFunctionNameParam(t *testing.T) {
	node := NewLambdaOperationsNode()
	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"test": "data"}},
	}
	params := map[string]interface{}{
		"operation": "invoke",
		"region":    "us-east-1",
	}
	result, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Nil(t, result)
	// The error is wrapped by CreateError as "invalid Lambda operation"
	assert.Contains(t, err.Error(), "invalid Lambda operation")
}

func TestLambdaOperationsNode_Execute_EmptyInputData(t *testing.T) {
	node := NewLambdaOperationsNode()
	inputData := []model.DataItem{}
	params := map[string]interface{}{
		"operation":    "invoke",
		"functionName": "my-function",
		"region":       "us-east-1",
	}
	// With empty input, the for loop over inputData does not execute,
	// so we get nil results and no error (after client initialization succeeds).
	// However, initializeLambdaClient will try to create a real AWS session.
	// Since we are not making actual API calls, session creation should succeed
	// (the AWS SDK allows creating sessions without valid credentials).
	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	assert.Empty(t, result)
}

// --- LambdaConfig Struct Tests ---

func TestLambdaConfig_FieldsAssignment(t *testing.T) {
	config := &LambdaConfig{
		Region:         "ap-southeast-1",
		AccessKeyID:    "AKID",
		SecretAccessKey: "SECRET",
		SessionToken:   "TOKEN",
	}
	assert.Equal(t, "ap-southeast-1", config.Region)
	assert.Equal(t, "AKID", config.AccessKeyID)
	assert.Equal(t, "SECRET", config.SecretAccessKey)
	assert.Equal(t, "TOKEN", config.SessionToken)
}

// --- LambdaOperation Struct Tests ---

func TestLambdaOperation_StructDefaults(t *testing.T) {
	op := &LambdaOperation{}
	assert.Empty(t, op.Operation)
	assert.Empty(t, op.FunctionName)
	assert.Nil(t, op.Payload)
	assert.Empty(t, op.InvocationType)
	assert.Empty(t, op.LogType)
	assert.Empty(t, op.Qualifier)
	assert.Nil(t, op.VpcConfig)
	assert.Nil(t, op.DeadLetterConfig)
	assert.Nil(t, op.TracingConfig)
}

// --- LambdaResult Struct Tests ---

func TestLambdaResult_StructDefaults(t *testing.T) {
	result := &LambdaResult{}
	assert.Empty(t, result.Operation)
	assert.False(t, result.Success)
	assert.Empty(t, result.FunctionName)
	assert.Equal(t, int64(0), result.StatusCode)
	assert.Nil(t, result.Payload)
	assert.Nil(t, result.Configuration)
}

// --- LambdaConfiguration Struct Tests ---

func TestLambdaConfiguration_Fields(t *testing.T) {
	config := &LambdaConfiguration{
		FunctionName: "test-function",
		FunctionArn:  "arn:aws:lambda:us-east-1:123456789012:function:test-function",
		Runtime:      "go1.x",
		Role:         "arn:aws:iam::123456789012:role/role",
		Handler:      "main",
		CodeSize:     1024,
		Description:  "Test function",
		Timeout:      30,
		MemorySize:   128,
		Version:      "$LATEST",
		State:        "Active",
	}
	assert.Equal(t, "test-function", config.FunctionName)
	assert.Equal(t, "arn:aws:lambda:us-east-1:123456789012:function:test-function", config.FunctionArn)
	assert.Equal(t, int64(1024), config.CodeSize)
	assert.Equal(t, int64(30), config.Timeout)
	assert.Equal(t, int64(128), config.MemorySize)
	assert.Equal(t, "Active", config.State)
}

// --- VPC, DeadLetter, Tracing Config Struct Tests ---

func TestLambdaVpcConfig_Fields(t *testing.T) {
	vpc := &LambdaVpcConfig{
		SubnetIds:        []string{"subnet-1", "subnet-2"},
		SecurityGroupIds: []string{"sg-1"},
	}
	assert.Len(t, vpc.SubnetIds, 2)
	assert.Len(t, vpc.SecurityGroupIds, 1)
}

func TestLambdaDeadLetterConfig_Fields(t *testing.T) {
	dlc := &LambdaDeadLetterConfig{
		TargetArn: "arn:aws:sqs:us-east-1:123456789012:my-dlq",
	}
	assert.Equal(t, "arn:aws:sqs:us-east-1:123456789012:my-dlq", dlc.TargetArn)
}

func TestLambdaTracingConfig_Fields(t *testing.T) {
	tc := &LambdaTracingConfig{
		Mode: "Active",
	}
	assert.Equal(t, "Active", tc.Mode)
}

// --- initializeLambdaClient Tests ---

func TestLambdaOperationsNode_InitializeLambdaClient_WithCredentials(t *testing.T) {
	node := NewLambdaOperationsNode()
	config := &LambdaConfig{
		Region:         "us-west-2",
		AccessKeyID:    "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}
	err := node.initializeLambdaClient(config)
	require.NoError(t, err)
	assert.NotNil(t, node.session)
	assert.NotNil(t, node.lambdaClient)
}

func TestLambdaOperationsNode_InitializeLambdaClient_WithoutCredentials(t *testing.T) {
	node := NewLambdaOperationsNode()
	config := &LambdaConfig{
		Region: "us-east-1",
	}
	// Session creation should succeed even without explicit credentials;
	// the SDK will look for environment/instance credentials.
	err := node.initializeLambdaClient(config)
	require.NoError(t, err)
	assert.NotNil(t, node.session)
	assert.NotNil(t, node.lambdaClient)
}

func TestLambdaOperationsNode_InitializeLambdaClient_WithSessionToken(t *testing.T) {
	node := NewLambdaOperationsNode()
	config := &LambdaConfig{
		Region:         "us-east-1",
		AccessKeyID:    "AKID",
		SecretAccessKey: "SECRET",
		SessionToken:   "TOKEN",
	}
	err := node.initializeLambdaClient(config)
	require.NoError(t, err)
	assert.NotNil(t, node.lambdaClient)
}

// --- ParseLambdaOperation edge cases ---

func TestLambdaOperationsNode_ParseLambdaOperation_EnvironmentConversion(t *testing.T) {
	node := NewLambdaOperationsNode()
	params := map[string]interface{}{
		"operation":    "create",
		"functionName": "fn",
		"environment": map[string]interface{}{
			"NUMBER_VAR": 42,
			"BOOL_VAR":   true,
			"STRING_VAR": "hello",
		},
	}
	op, err := node.parseLambdaOperation(params)
	require.NoError(t, err)
	assert.Equal(t, "42", op.Environment["NUMBER_VAR"])
	assert.Equal(t, "true", op.Environment["BOOL_VAR"])
	assert.Equal(t, "hello", op.Environment["STRING_VAR"])
}

func TestLambdaOperationsNode_ParseLambdaOperation_TagsConversion(t *testing.T) {
	node := NewLambdaOperationsNode()
	params := map[string]interface{}{
		"operation":    "create",
		"functionName": "fn",
		"tags": map[string]interface{}{
			"env":     "prod",
			"version": 3,
		},
	}
	op, err := node.parseLambdaOperation(params)
	require.NoError(t, err)
	assert.Equal(t, "prod", op.Tags["env"])
	assert.Equal(t, "3", op.Tags["version"])
}
