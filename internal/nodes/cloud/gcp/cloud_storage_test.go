package gcp

import (
	"testing"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------
// Construction and metadata tests
// ---------------------------------------------------------------

func TestNewCloudStorageNode(t *testing.T) {
	node := NewCloudStorageNode()
	require.NotNil(t, node, "NewCloudStorageNode should return a non-nil node")
	require.NotNil(t, node.BaseNode, "BaseNode should be initialised")
	require.NotNil(t, node.evaluator, "evaluator should be initialised")
	require.NotNil(t, node.ctx, "context should be initialised")
}

func TestCloudStorageNode_ImplementsNodeExecutor(t *testing.T) {
	// Compile-time check that CloudStorageNode satisfies the NodeExecutor interface.
	var _ base.NodeExecutor = (*CloudStorageNode)(nil)
}

func TestCloudStorageNode_Description(t *testing.T) {
	node := NewCloudStorageNode()
	desc := node.Description()

	assert.Equal(t, "Google Cloud Storage", desc.Name)
	assert.Equal(t, "cloud", desc.Category)
	assert.NotEmpty(t, desc.Description, "Description should not be empty")
}

// ---------------------------------------------------------------
// ValidateParameters tests
// ---------------------------------------------------------------

func TestCloudStorageNode_ValidateParameters(t *testing.T) {
	node := NewCloudStorageNode()

	tests := []struct {
		name        string
		params      map[string]interface{}
		expectErr   bool
		errContains string
	}{
		{
			name: "valid params with upload operation",
			params: map[string]interface{}{
				"operation": "upload",
				"bucket":    "my-bucket",
				"object":    "path/to/file.txt",
			},
			expectErr: false,
		},
		{
			name: "valid params with download operation",
			params: map[string]interface{}{
				"operation": "download",
				"bucket":    "my-bucket",
				"object":    "path/to/file.txt",
			},
			expectErr: false,
		},
		{
			name: "valid params with delete operation",
			params: map[string]interface{}{
				"operation": "delete",
				"bucket":    "my-bucket",
			},
			expectErr: false,
		},
		{
			name: "valid params with list operation",
			params: map[string]interface{}{
				"operation": "list",
				"bucket":    "my-bucket",
			},
			expectErr: false,
		},
		{
			name: "valid params with copy operation",
			params: map[string]interface{}{
				"operation": "copy",
				"bucket":    "my-bucket",
			},
			expectErr: false,
		},
		{
			name: "valid params with move operation",
			params: map[string]interface{}{
				"operation": "move",
				"bucket":    "my-bucket",
			},
			expectErr: false,
		},
		{
			name: "valid params with get_metadata operation",
			params: map[string]interface{}{
				"operation": "get_metadata",
				"bucket":    "my-bucket",
			},
			expectErr: false,
		},
		{
			name: "valid params with set_metadata operation",
			params: map[string]interface{}{
				"operation": "set_metadata",
				"bucket":    "my-bucket",
			},
			expectErr: false,
		},
		{
			name: "valid params with create_bucket operation",
			params: map[string]interface{}{
				"operation": "create_bucket",
				"bucket":    "my-new-bucket",
			},
			expectErr: false,
		},
		{
			name: "valid params with delete_bucket operation",
			params: map[string]interface{}{
				"operation": "delete_bucket",
				"bucket":    "my-bucket",
			},
			expectErr: false,
		},
		{
			name: "valid params with list_buckets operation",
			params: map[string]interface{}{
				"operation": "list_buckets",
				"bucket":    "my-bucket",
			},
			expectErr: false,
		},
		{
			name: "valid params with get_bucket operation",
			params: map[string]interface{}{
				"operation": "get_bucket",
				"bucket":    "my-bucket",
			},
			expectErr: false,
		},
		{
			name: "valid params with set_bucket_lifecycle operation",
			params: map[string]interface{}{
				"operation": "set_bucket_lifecycle",
				"bucket":    "my-bucket",
			},
			expectErr: false,
		},
		{
			name: "valid params with generate_signed_url operation",
			params: map[string]interface{}{
				"operation": "generate_signed_url",
				"bucket":    "my-bucket",
			},
			expectErr: false,
		},
		{
			name: "missing operation parameter",
			params: map[string]interface{}{
				"bucket": "my-bucket",
			},
			expectErr:   true,
			errContains: "operation parameter is required",
		},
		{
			name: "missing bucket parameter",
			params: map[string]interface{}{
				"operation": "upload",
			},
			expectErr:   true,
			errContains: "bucket parameter is required",
		},
		{
			name:        "missing both operation and bucket",
			params:      map[string]interface{}{},
			expectErr:   true,
			errContains: "operation parameter is required",
		},
		{
			name: "invalid operation",
			params: map[string]interface{}{
				"operation": "invalid_operation",
				"bucket":    "my-bucket",
			},
			expectErr:   true,
			errContains: "invalid operation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := node.ValidateParameters(tt.params)
			if tt.expectErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------
// Execute – credential validation (no GCP call)
// ---------------------------------------------------------------

func TestCloudStorageNode_Execute_MissingCredentials(t *testing.T) {
	node := NewCloudStorageNode()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"key": "value"}},
	}

	// No credential-related params: no keyFile, no keyFileContent, useADC is false by default
	params := map[string]interface{}{
		"operation": "list",
		"bucket":    "my-bucket",
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize Cloud Storage client")
}

func TestCloudStorageNode_Execute_InvalidKeyFileContent(t *testing.T) {
	node := NewCloudStorageNode()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"key": "value"}},
	}

	// Provide invalid JSON as key file content - this should fail when creating the client
	params := map[string]interface{}{
		"operation":      "list",
		"bucket":         "my-bucket",
		"keyFileContent": "this-is-not-valid-json",
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	// The error should come from failing to initialize the storage client
	assert.Contains(t, err.Error(), "failed to initialize Cloud Storage client")
}

func TestCloudStorageNode_Execute_InvalidKeyFile(t *testing.T) {
	node := NewCloudStorageNode()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"key": "value"}},
	}

	// Provide a non-existent key file path
	params := map[string]interface{}{
		"operation": "list",
		"bucket":    "my-bucket",
		"keyFile":   "/nonexistent/path/to/credentials.json",
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize Cloud Storage client")
}

// ---------------------------------------------------------------
// parseStorageOperation tests (indirectly via Execute)
// ---------------------------------------------------------------

func TestCloudStorageNode_Execute_MissingOperationParam(t *testing.T) {
	node := NewCloudStorageNode()

	// Missing operation should be caught by ValidateParameters.
	params := map[string]interface{}{
		"bucket": "my-bucket",
	}

	err := node.ValidateParameters(params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "operation parameter is required")
}

func TestCloudStorageNode_Execute_MissingBucketParam(t *testing.T) {
	node := NewCloudStorageNode()

	params := map[string]interface{}{
		"operation": "upload",
	}

	err := node.ValidateParameters(params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bucket parameter is required")
}

// ---------------------------------------------------------------
// parseGCPConfig tests (indirectly via construction)
// ---------------------------------------------------------------

func TestCloudStorageNode_ParseGCPConfig_AllFields(t *testing.T) {
	node := NewCloudStorageNode()

	params := map[string]interface{}{
		"projectId":           "my-project",
		"keyFile":             "/path/to/key.json",
		"keyFileContent":      `{"type":"service_account"}`,
		"serviceAccountEmail": "test@my-project.iam.gserviceaccount.com",
		"useADC":              true,
	}

	config, err := node.parseGCPConfig(params)
	require.NoError(t, err)
	assert.Equal(t, "my-project", config.ProjectID)
	assert.Equal(t, "/path/to/key.json", config.KeyFile)
	assert.Equal(t, `{"type":"service_account"}`, config.KeyFileContent)
	assert.Equal(t, "test@my-project.iam.gserviceaccount.com", config.ServiceAccountEmail)
	assert.True(t, config.UseADC)
}

func TestCloudStorageNode_ParseGCPConfig_EmptyParams(t *testing.T) {
	node := NewCloudStorageNode()

	params := map[string]interface{}{}

	config, err := node.parseGCPConfig(params)
	require.NoError(t, err)
	assert.Empty(t, config.ProjectID)
	assert.Empty(t, config.KeyFile)
	assert.Empty(t, config.KeyFileContent)
	assert.Empty(t, config.ServiceAccountEmail)
	assert.False(t, config.UseADC)
}

func TestCloudStorageNode_ParseGCPConfig_PartialFields(t *testing.T) {
	node := NewCloudStorageNode()

	params := map[string]interface{}{
		"projectId": "my-project",
		"useADC":    true,
	}

	config, err := node.parseGCPConfig(params)
	require.NoError(t, err)
	assert.Equal(t, "my-project", config.ProjectID)
	assert.Empty(t, config.KeyFile)
	assert.Empty(t, config.KeyFileContent)
	assert.True(t, config.UseADC)
}

func TestCloudStorageNode_ParseGCPConfig_WrongTypes(t *testing.T) {
	node := NewCloudStorageNode()

	// Params with wrong types - these should be silently ignored by type assertions
	params := map[string]interface{}{
		"projectId": 12345,    // should be string
		"keyFile":   true,     // should be string
		"useADC":    "true",   // should be bool
	}

	config, err := node.parseGCPConfig(params)
	require.NoError(t, err)
	// All fields should remain at zero values since type assertions fail
	assert.Empty(t, config.ProjectID)
	assert.Empty(t, config.KeyFile)
	assert.False(t, config.UseADC)
}

// ---------------------------------------------------------------
// parseStorageOperation tests
// ---------------------------------------------------------------

func TestCloudStorageNode_ParseStorageOperation_ValidFullParams(t *testing.T) {
	node := NewCloudStorageNode()

	params := map[string]interface{}{
		"operation":    "upload",
		"bucket":       "my-bucket",
		"object":       "path/to/file.txt",
		"source":       "gs://source-bucket/source-object",
		"destination":  "gs://dest-bucket/dest-object",
		"prefix":       "logs/",
		"delimiter":    "/",
		"contentType":  "application/json",
		"storageClass": "NEARLINE",
		"versioning":   true,
		"metadata": map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		},
		"options": map[string]interface{}{
			"projectId": "my-project",
		},
	}

	operation, err := node.parseStorageOperation(params)
	require.NoError(t, err)
	assert.Equal(t, "upload", operation.Operation)
	assert.Equal(t, "my-bucket", operation.Bucket)
	assert.Equal(t, "path/to/file.txt", operation.Object)
	assert.Equal(t, "gs://source-bucket/source-object", operation.Source)
	assert.Equal(t, "gs://dest-bucket/dest-object", operation.Destination)
	assert.Equal(t, "logs/", operation.Prefix)
	assert.Equal(t, "/", operation.Delimiter)
	assert.Equal(t, "application/json", operation.ContentType)
	assert.Equal(t, "NEARLINE", operation.StorageClass)
	assert.True(t, operation.Versioning)
	assert.Equal(t, "value1", operation.Metadata["key1"])
	assert.Equal(t, "value2", operation.Metadata["key2"])
	assert.Equal(t, "my-project", operation.Options["projectId"])
}

func TestCloudStorageNode_ParseStorageOperation_MinimalParams(t *testing.T) {
	node := NewCloudStorageNode()

	params := map[string]interface{}{
		"operation": "list",
		"bucket":    "my-bucket",
	}

	operation, err := node.parseStorageOperation(params)
	require.NoError(t, err)
	assert.Equal(t, "list", operation.Operation)
	assert.Equal(t, "my-bucket", operation.Bucket)
	assert.Empty(t, operation.Object)
	assert.Empty(t, operation.Source)
	assert.Empty(t, operation.Destination)
	assert.Empty(t, operation.Prefix)
	assert.Empty(t, operation.Delimiter)
	assert.Empty(t, operation.ContentType)
	assert.Empty(t, operation.StorageClass)
	assert.False(t, operation.Versioning)
	assert.Nil(t, operation.Metadata)
	assert.NotNil(t, operation.Options)
	assert.Empty(t, operation.Options)
}

func TestCloudStorageNode_ParseStorageOperation_MissingOperation(t *testing.T) {
	node := NewCloudStorageNode()

	params := map[string]interface{}{
		"bucket": "my-bucket",
	}

	_, err := node.parseStorageOperation(params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "operation parameter is required")
}

func TestCloudStorageNode_ParseStorageOperation_MissingBucket(t *testing.T) {
	node := NewCloudStorageNode()

	params := map[string]interface{}{
		"operation": "upload",
	}

	_, err := node.parseStorageOperation(params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bucket parameter is required")
}

func TestCloudStorageNode_ParseStorageOperation_MetadataConversion(t *testing.T) {
	node := NewCloudStorageNode()

	params := map[string]interface{}{
		"operation": "upload",
		"bucket":    "my-bucket",
		"metadata": map[string]interface{}{
			"stringVal":  "hello",
			"intVal":     42,
			"floatVal":   3.14,
			"boolVal":    true,
		},
	}

	operation, err := node.parseStorageOperation(params)
	require.NoError(t, err)
	assert.Equal(t, "hello", operation.Metadata["stringVal"])
	assert.Equal(t, "42", operation.Metadata["intVal"])
	assert.Equal(t, "3.14", operation.Metadata["floatVal"])
	assert.Equal(t, "true", operation.Metadata["boolVal"])
}

// ---------------------------------------------------------------
// detectContentType tests
// ---------------------------------------------------------------

func TestCloudStorageNode_DetectContentType(t *testing.T) {
	node := NewCloudStorageNode()

	tests := []struct {
		objectName  string
		expected    string
	}{
		{"file.json", "application/json"},
		{"file.xml", "application/xml"},
		{"file.html", "text/html"},
		{"file.css", "text/css"},
		{"file.js", "application/javascript"},
		{"file.jpg", "image/jpeg"},
		{"file.jpeg", "image/jpeg"},
		{"file.png", "image/png"},
		{"file.pdf", "application/pdf"},
		{"file.txt", "text/plain"},
		{"file.unknown", "application/octet-stream"},
		{"file", "application/octet-stream"},
		{"path/to/file.json", "application/json"},
		{"path/to/file.tar.gz", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.objectName, func(t *testing.T) {
			result := node.detectContentType(tt.objectName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------
// initializeStorageClient credential validation tests
// ---------------------------------------------------------------

func TestCloudStorageNode_InitializeStorageClient_NoCredentials(t *testing.T) {
	node := NewCloudStorageNode()

	config := &GCPConfig{
		ProjectID: "my-project",
		UseADC:    false,
		// No keyFile or keyFileContent
	}

	err := node.initializeStorageClient(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "credentials must be provided")
}

func TestCloudStorageNode_InitializeStorageClient_InvalidKeyFileContent(t *testing.T) {
	node := NewCloudStorageNode()

	config := &GCPConfig{
		ProjectID:      "my-project",
		KeyFileContent: "not-valid-json",
	}

	err := node.initializeStorageClient(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create storage client")
}

func TestCloudStorageNode_InitializeStorageClient_NonexistentKeyFile(t *testing.T) {
	node := NewCloudStorageNode()

	config := &GCPConfig{
		ProjectID: "my-project",
		KeyFile:   "/nonexistent/path/to/key.json",
	}

	err := node.initializeStorageClient(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create storage client")
}

// ---------------------------------------------------------------
// GCPConfig struct tests
// ---------------------------------------------------------------

func TestGCPConfig_Defaults(t *testing.T) {
	config := &GCPConfig{}
	assert.Empty(t, config.ProjectID)
	assert.Empty(t, config.KeyFile)
	assert.Empty(t, config.KeyFileContent)
	assert.Empty(t, config.ServiceAccountEmail)
	assert.False(t, config.UseADC)
}

// ---------------------------------------------------------------
// StorageOperation struct tests
// ---------------------------------------------------------------

func TestStorageOperation_Defaults(t *testing.T) {
	op := &StorageOperation{}
	assert.Empty(t, op.Operation)
	assert.Empty(t, op.Bucket)
	assert.Empty(t, op.Object)
	assert.Empty(t, op.Source)
	assert.Empty(t, op.Destination)
	assert.Empty(t, op.Prefix)
	assert.Empty(t, op.Delimiter)
	assert.Empty(t, op.ContentType)
	assert.Nil(t, op.Metadata)
	assert.Nil(t, op.ACL)
	assert.Nil(t, op.Lifecycle)
	assert.False(t, op.Versioning)
	assert.Nil(t, op.Encryption)
	assert.Empty(t, op.StorageClass)
	assert.Nil(t, op.Options)
}

// ---------------------------------------------------------------
// Execute with empty input data
// ---------------------------------------------------------------

func TestCloudStorageNode_Execute_EmptyInputData(t *testing.T) {
	node := NewCloudStorageNode()

	// Even with credentials missing, if input data is empty and we can get
	// past client initialization, we return empty results. However, since
	// client init happens before the loop, it will still fail on credentials.
	// This test verifies the credential check still applies with empty input.
	params := map[string]interface{}{
		"operation": "list",
		"bucket":    "my-bucket",
	}

	_, err := node.Execute([]model.DataItem{}, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize Cloud Storage client")
}

// ---------------------------------------------------------------
// ValidateParameters – edge cases
// ---------------------------------------------------------------

func TestCloudStorageNode_ValidateParameters_NilParams(t *testing.T) {
	node := NewCloudStorageNode()

	// nil params - the CloudStorageNode checks for "operation" key, which won't exist
	err := node.ValidateParameters(nil)
	// ValidateParameters checks params["operation"] which will panic on nil map
	// unless the implementation handles it. Let's see what happens:
	// Looking at the code: if _, ok := params["operation"]; !ok { return error }
	// Accessing a nil map returns zero value without panic, so ok will be false.
	require.Error(t, err)
	assert.Contains(t, err.Error(), "operation parameter is required")
}

func TestCloudStorageNode_ValidateParameters_OperationWrongType(t *testing.T) {
	node := NewCloudStorageNode()

	// Operation is present but not a string - the type assertion in the
	// validation section will fail, so the invalid operation check is skipped,
	// and no error is returned (the existence check passes).
	params := map[string]interface{}{
		"operation": 123,
		"bucket":    "my-bucket",
	}

	err := node.ValidateParameters(params)
	// The first check `if _, ok := params["operation"]; !ok` passes (key exists).
	// The second check `if operation, ok := params["operation"].(string); ok` fails
	// since 123 is not a string, so the valid operations check is skipped.
	assert.NoError(t, err)
}

// ---------------------------------------------------------------
// All valid operations validate successfully
// ---------------------------------------------------------------

func TestCloudStorageNode_ValidateParameters_AllValidOperations(t *testing.T) {
	node := NewCloudStorageNode()

	validOps := []string{
		"upload", "download", "delete", "list", "copy", "move",
		"get_metadata", "set_metadata", "create_bucket", "delete_bucket",
		"list_buckets", "get_bucket", "set_bucket_lifecycle",
		"generate_signed_url",
	}

	for _, op := range validOps {
		t.Run(op, func(t *testing.T) {
			params := map[string]interface{}{
				"operation": op,
				"bucket":    "test-bucket",
			}
			err := node.ValidateParameters(params)
			assert.NoError(t, err)
		})
	}
}

// ---------------------------------------------------------------
// convertToGCSLifecycle tests
// ---------------------------------------------------------------

func TestCloudStorageNode_ConvertToGCSLifecycle_BasicRule(t *testing.T) {
	node := NewCloudStorageNode()

	lifecycle := &StorageLifecycle{
		Rules: []StorageLifecycleRule{
			{
				Action: StorageLifecycleAction{
					Type: "Delete",
				},
				Condition: StorageLifecycleCondition{
					Age: 30,
				},
			},
		},
	}

	result := node.convertToGCSLifecycle(lifecycle)
	require.Len(t, result.Rules, 1)
	assert.Equal(t, "Delete", result.Rules[0].Action.Type)
	assert.Equal(t, int64(30), result.Rules[0].Condition.AgeInDays)
}

func TestCloudStorageNode_ConvertToGCSLifecycle_SetStorageClass(t *testing.T) {
	node := NewCloudStorageNode()

	lifecycle := &StorageLifecycle{
		Rules: []StorageLifecycleRule{
			{
				Action: StorageLifecycleAction{
					Type:         "SetStorageClass",
					StorageClass: "COLDLINE",
				},
				Condition: StorageLifecycleCondition{
					Age:                   90,
					MatchesStorageClasses: []string{"STANDARD"},
				},
			},
		},
	}

	result := node.convertToGCSLifecycle(lifecycle)
	require.Len(t, result.Rules, 1)
	assert.Equal(t, "SetStorageClass", result.Rules[0].Action.Type)
	assert.Equal(t, "COLDLINE", result.Rules[0].Action.StorageClass)
	assert.Equal(t, int64(90), result.Rules[0].Condition.AgeInDays)
	assert.Equal(t, []string{"STANDARD"}, result.Rules[0].Condition.MatchesStorageClasses)
}

func TestCloudStorageNode_ConvertToGCSLifecycle_WithCreatedBefore(t *testing.T) {
	node := NewCloudStorageNode()

	lifecycle := &StorageLifecycle{
		Rules: []StorageLifecycleRule{
			{
				Action: StorageLifecycleAction{
					Type: "Delete",
				},
				Condition: StorageLifecycleCondition{
					CreatedBefore: "2024-01-01",
				},
			},
		},
	}

	result := node.convertToGCSLifecycle(lifecycle)
	require.Len(t, result.Rules, 1)
	assert.False(t, result.Rules[0].Condition.CreatedBefore.IsZero(),
		"CreatedBefore should be parsed as a valid date")
	assert.Equal(t, 2024, result.Rules[0].Condition.CreatedBefore.Year())
	assert.Equal(t, 1, result.Rules[0].Condition.CreatedBefore.Day())
}

func TestCloudStorageNode_ConvertToGCSLifecycle_WithInvalidCreatedBefore(t *testing.T) {
	node := NewCloudStorageNode()

	lifecycle := &StorageLifecycle{
		Rules: []StorageLifecycleRule{
			{
				Action: StorageLifecycleAction{
					Type: "Delete",
				},
				Condition: StorageLifecycleCondition{
					CreatedBefore: "not-a-date",
				},
			},
		},
	}

	result := node.convertToGCSLifecycle(lifecycle)
	require.Len(t, result.Rules, 1)
	// Invalid date should result in zero time (silently skipped)
	assert.True(t, result.Rules[0].Condition.CreatedBefore.IsZero(),
		"Invalid date should result in zero time")
}

func TestCloudStorageNode_ConvertToGCSLifecycle_WithIsLive(t *testing.T) {
	node := NewCloudStorageNode()

	boolTrue := true
	boolFalse := false

	tests := []struct {
		name     string
		isLive   *bool
	}{
		{"isLive true", &boolTrue},
		{"isLive false", &boolFalse},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lifecycle := &StorageLifecycle{
				Rules: []StorageLifecycleRule{
					{
						Action: StorageLifecycleAction{Type: "Delete"},
						Condition: StorageLifecycleCondition{
							IsLive: tt.isLive,
						},
					},
				},
			}

			result := node.convertToGCSLifecycle(lifecycle)
			require.Len(t, result.Rules, 1)
			// Liveness should be set to a non-zero value
			assert.NotZero(t, result.Rules[0].Condition.Liveness)
		})
	}
}

func TestCloudStorageNode_ConvertToGCSLifecycle_MultipleRules(t *testing.T) {
	node := NewCloudStorageNode()

	lifecycle := &StorageLifecycle{
		Rules: []StorageLifecycleRule{
			{
				Action:    StorageLifecycleAction{Type: "Delete"},
				Condition: StorageLifecycleCondition{Age: 30},
			},
			{
				Action:    StorageLifecycleAction{Type: "SetStorageClass", StorageClass: "ARCHIVE"},
				Condition: StorageLifecycleCondition{Age: 365},
			},
			{
				Action:    StorageLifecycleAction{Type: "Delete"},
				Condition: StorageLifecycleCondition{NumNewerVersions: 3},
			},
		},
	}

	result := node.convertToGCSLifecycle(lifecycle)
	require.Len(t, result.Rules, 3)
	assert.Equal(t, "Delete", result.Rules[0].Action.Type)
	assert.Equal(t, "SetStorageClass", result.Rules[1].Action.Type)
	assert.Equal(t, "ARCHIVE", result.Rules[1].Action.StorageClass)
	assert.Equal(t, int64(3), result.Rules[2].Condition.NumNewerVersions)
}

func TestCloudStorageNode_ConvertToGCSLifecycle_EmptyRules(t *testing.T) {
	node := NewCloudStorageNode()

	lifecycle := &StorageLifecycle{
		Rules: []StorageLifecycleRule{},
	}

	result := node.convertToGCSLifecycle(lifecycle)
	assert.Empty(t, result.Rules)
}

func TestCloudStorageNode_ConvertToGCSLifecycle_WithMatchesPrefixAndSuffix(t *testing.T) {
	node := NewCloudStorageNode()

	lifecycle := &StorageLifecycle{
		Rules: []StorageLifecycleRule{
			{
				Action: StorageLifecycleAction{Type: "Delete"},
				Condition: StorageLifecycleCondition{
					MatchesPrefix: []string{"logs/", "tmp/"},
					MatchesSuffix: []string{".log", ".tmp"},
				},
			},
		},
	}

	result := node.convertToGCSLifecycle(lifecycle)
	require.Len(t, result.Rules, 1)
	assert.Equal(t, []string{"logs/", "tmp/"}, result.Rules[0].Condition.MatchesPrefix)
	assert.Equal(t, []string{".log", ".tmp"}, result.Rules[0].Condition.MatchesSuffix)
}

// ---------------------------------------------------------------
// StorageOperationResult struct tests
// ---------------------------------------------------------------

func TestStorageOperationResult_Defaults(t *testing.T) {
	result := &StorageOperationResult{}
	assert.Empty(t, result.Operation)
	assert.False(t, result.Success)
	assert.Empty(t, result.Bucket)
	assert.Empty(t, result.Object)
	assert.Nil(t, result.Objects)
	assert.Nil(t, result.Buckets)
	assert.Nil(t, result.Data)
	assert.Equal(t, int64(0), result.Size)
	assert.Empty(t, result.ContentType)
	assert.Equal(t, int64(0), result.Generation)
	assert.Empty(t, result.URL)
	assert.Empty(t, result.Error)
}
