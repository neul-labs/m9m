package aws

import (
	"testing"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Constructor Tests ---

func TestNewS3OperationsNode_ReturnsNonNil(t *testing.T) {
	node := NewS3OperationsNode()
	require.NotNil(t, node)
	require.NotNil(t, node.BaseNode)
	require.NotNil(t, node.evaluator)
}

// --- Description Tests ---

func TestS3OperationsNode_Description_Name(t *testing.T) {
	node := NewS3OperationsNode()
	desc := node.Description()
	assert.Equal(t, "AWS S3", desc.Name)
}

func TestS3OperationsNode_Description_Category(t *testing.T) {
	node := NewS3OperationsNode()
	desc := node.Description()
	assert.Equal(t, "cloud", desc.Category)
}

func TestS3OperationsNode_Description_DescriptionField(t *testing.T) {
	node := NewS3OperationsNode()
	desc := node.Description()
	assert.Equal(t, "Amazon S3 storage operations", desc.Description)
}

// --- ValidateParameters Tests ---

func TestS3OperationsNode_ValidateParameters_ValidUpload(t *testing.T) {
	node := NewS3OperationsNode()
	params := map[string]interface{}{
		"operation": "upload",
		"bucket":    "my-bucket",
	}
	err := node.ValidateParameters(params)
	assert.NoError(t, err)
}

func TestS3OperationsNode_ValidateParameters_AllOperations(t *testing.T) {
	node := NewS3OperationsNode()

	validOperations := []string{
		"upload", "download", "delete", "list", "copy", "head",
		"create_bucket", "delete_bucket", "set_bucket_policy",
		"get_bucket_policy", "set_bucket_lifecycle", "set_bucket_versioning",
		"generate_presigned_url",
	}

	for _, op := range validOperations {
		t.Run(op, func(t *testing.T) {
			params := map[string]interface{}{
				"operation": op,
				"bucket":    "test-bucket",
			}
			err := node.ValidateParameters(params)
			assert.NoError(t, err, "operation %q should be valid", op)
		})
	}
}

func TestS3OperationsNode_ValidateParameters_MissingOperation(t *testing.T) {
	node := NewS3OperationsNode()
	params := map[string]interface{}{
		"bucket": "my-bucket",
	}
	err := node.ValidateParameters(params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "operation parameter is required")
}

func TestS3OperationsNode_ValidateParameters_MissingBucket(t *testing.T) {
	node := NewS3OperationsNode()
	params := map[string]interface{}{
		"operation": "upload",
	}
	err := node.ValidateParameters(params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bucket parameter is required")
}

func TestS3OperationsNode_ValidateParameters_InvalidOperation(t *testing.T) {
	node := NewS3OperationsNode()
	params := map[string]interface{}{
		"operation": "nonexistent_operation",
		"bucket":    "my-bucket",
	}
	err := node.ValidateParameters(params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid operation")
}

func TestS3OperationsNode_ValidateParameters_EmptyParams(t *testing.T) {
	node := NewS3OperationsNode()
	params := map[string]interface{}{}
	err := node.ValidateParameters(params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "operation parameter is required")
}

func TestS3OperationsNode_ValidateParameters_OperationNotString(t *testing.T) {
	node := NewS3OperationsNode()
	// operation exists but is not a string; the type assertion for valid operations is skipped
	params := map[string]interface{}{
		"operation": 42,
		"bucket":    "my-bucket",
	}
	err := node.ValidateParameters(params)
	// The key "operation" exists so the required check passes;
	// the type assertion to string fails so the valid-operations check is skipped.
	assert.NoError(t, err)
}

func TestS3OperationsNode_ValidateParameters_BucketNotString(t *testing.T) {
	node := NewS3OperationsNode()
	// bucket exists but is not a string; key presence check still passes
	params := map[string]interface{}{
		"operation": "upload",
		"bucket":    123,
	}
	err := node.ValidateParameters(params)
	// "bucket" key exists, so the required check passes.
	assert.NoError(t, err)
}

// --- parseS3Config Tests ---

func TestS3OperationsNode_ParseS3Config_Defaults(t *testing.T) {
	node := NewS3OperationsNode()
	params := map[string]interface{}{}
	config, err := node.parseS3Config(params)
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, "us-east-1", config.Region, "default region should be us-east-1")
	assert.Empty(t, config.AccessKeyID)
	assert.Empty(t, config.SecretAccessKey)
	assert.Empty(t, config.SessionToken)
	assert.Empty(t, config.Endpoint)
	assert.False(t, config.S3ForcePathStyle)
	assert.False(t, config.DisableSSL)
}

func TestS3OperationsNode_ParseS3Config_WithCredentials(t *testing.T) {
	node := NewS3OperationsNode()
	params := map[string]interface{}{
		"region":          "eu-central-1",
		"accessKeyId":     "AKIAIOSFODNN7EXAMPLE",
		"secretAccessKey": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		"sessionToken":    "FwoGZXIvYXdzEBY",
	}
	config, err := node.parseS3Config(params)
	require.NoError(t, err)
	assert.Equal(t, "eu-central-1", config.Region)
	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", config.AccessKeyID)
	assert.Equal(t, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", config.SecretAccessKey)
	assert.Equal(t, "FwoGZXIvYXdzEBY", config.SessionToken)
}

func TestS3OperationsNode_ParseS3Config_CustomEndpoint(t *testing.T) {
	node := NewS3OperationsNode()
	params := map[string]interface{}{
		"endpoint":         "http://localhost:9000",
		"s3ForcePathStyle": true,
		"disableSSL":       true,
	}
	config, err := node.parseS3Config(params)
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:9000", config.Endpoint)
	assert.True(t, config.S3ForcePathStyle)
	assert.True(t, config.DisableSSL)
}

func TestS3OperationsNode_ParseS3Config_PartialCredentials(t *testing.T) {
	node := NewS3OperationsNode()
	params := map[string]interface{}{
		"accessKeyId": "AKIAIOSFODNN7EXAMPLE",
	}
	config, err := node.parseS3Config(params)
	require.NoError(t, err)
	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", config.AccessKeyID)
	assert.Empty(t, config.SecretAccessKey)
}

// --- parseS3Operation Tests ---

func TestS3OperationsNode_ParseS3Operation_Basic(t *testing.T) {
	node := NewS3OperationsNode()
	params := map[string]interface{}{
		"operation": "upload",
		"bucket":    "my-bucket",
	}
	op, err := node.parseS3Operation(params)
	require.NoError(t, err)
	require.NotNil(t, op)
	assert.Equal(t, "upload", op.Operation)
	assert.Equal(t, "my-bucket", op.Bucket)
}

func TestS3OperationsNode_ParseS3Operation_MissingOperation(t *testing.T) {
	node := NewS3OperationsNode()
	params := map[string]interface{}{
		"bucket": "my-bucket",
	}
	op, err := node.parseS3Operation(params)
	require.Error(t, err)
	assert.Nil(t, op)
	assert.Contains(t, err.Error(), "operation parameter is required")
}

func TestS3OperationsNode_ParseS3Operation_MissingBucket(t *testing.T) {
	node := NewS3OperationsNode()
	params := map[string]interface{}{
		"operation": "upload",
	}
	op, err := node.parseS3Operation(params)
	require.Error(t, err)
	assert.Nil(t, op)
	assert.Contains(t, err.Error(), "bucket parameter is required")
}

func TestS3OperationsNode_ParseS3Operation_AllFields(t *testing.T) {
	node := NewS3OperationsNode()
	params := map[string]interface{}{
		"operation":    "upload",
		"bucket":       "my-bucket",
		"key":          "path/to/file.json",
		"source":       "data-field",
		"destination":  "/tmp/output",
		"prefix":       "logs/",
		"contentType":  "application/json",
		"acl":          "public-read",
		"storageClass": "STANDARD_IA",
		"versioning":   true,
		"metadata": map[string]interface{}{
			"author": "test",
		},
		"copySource": map[string]interface{}{
			"bucket":    "source-bucket",
			"key":       "source/key.txt",
			"versionId": "v1",
		},
		"options": map[string]interface{}{
			"maxKeys":   100,
			"delimiter": "/",
		},
	}
	op, err := node.parseS3Operation(params)
	require.NoError(t, err)
	require.NotNil(t, op)

	assert.Equal(t, "upload", op.Operation)
	assert.Equal(t, "my-bucket", op.Bucket)
	assert.Equal(t, "path/to/file.json", op.Key)
	assert.Equal(t, "data-field", op.Source)
	assert.Equal(t, "/tmp/output", op.Destination)
	assert.Equal(t, "logs/", op.Prefix)
	assert.Equal(t, "application/json", op.ContentType)
	assert.Equal(t, "public-read", op.ACL)
	assert.Equal(t, "STANDARD_IA", op.StorageClass)
	assert.True(t, op.Versioning)
	assert.Equal(t, "test", op.Metadata["author"])

	require.NotNil(t, op.CopySource)
	assert.Equal(t, "source-bucket", op.CopySource.Bucket)
	assert.Equal(t, "source/key.txt", op.CopySource.Key)
	assert.Equal(t, "v1", op.CopySource.VersionID)

	assert.Equal(t, 100, op.Options["maxKeys"])
	assert.Equal(t, "/", op.Options["delimiter"])
}

func TestS3OperationsNode_ParseS3Operation_OptionsInitialized(t *testing.T) {
	node := NewS3OperationsNode()
	params := map[string]interface{}{
		"operation": "list",
		"bucket":    "my-bucket",
	}
	op, err := node.parseS3Operation(params)
	require.NoError(t, err)
	require.NotNil(t, op.Options, "Options map should be initialized even when not provided in params")
}

func TestS3OperationsNode_ParseS3Operation_MetadataConversion(t *testing.T) {
	node := NewS3OperationsNode()
	params := map[string]interface{}{
		"operation": "upload",
		"bucket":    "my-bucket",
		"metadata": map[string]interface{}{
			"count":   42,
			"active":  true,
			"name":    "test",
		},
	}
	op, err := node.parseS3Operation(params)
	require.NoError(t, err)
	assert.Equal(t, "42", op.Metadata["count"])
	assert.Equal(t, "true", op.Metadata["active"])
	assert.Equal(t, "test", op.Metadata["name"])
}

func TestS3OperationsNode_ParseS3Operation_CopySourcePartial(t *testing.T) {
	node := NewS3OperationsNode()
	params := map[string]interface{}{
		"operation": "copy",
		"bucket":    "dest-bucket",
		"copySource": map[string]interface{}{
			"bucket": "src-bucket",
		},
	}
	op, err := node.parseS3Operation(params)
	require.NoError(t, err)
	require.NotNil(t, op.CopySource)
	assert.Equal(t, "src-bucket", op.CopySource.Bucket)
	assert.Empty(t, op.CopySource.Key)
	assert.Empty(t, op.CopySource.VersionID)
}

// --- Execute Error Path Tests ---

func TestS3OperationsNode_Execute_MissingOperationParam(t *testing.T) {
	node := NewS3OperationsNode()
	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"test": "data"}},
	}
	params := map[string]interface{}{
		"bucket": "my-bucket",
		"region": "us-east-1",
	}
	result, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "operation")
}

func TestS3OperationsNode_Execute_MissingBucketParam(t *testing.T) {
	node := NewS3OperationsNode()
	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"test": "data"}},
	}
	params := map[string]interface{}{
		"operation": "upload",
		"region":    "us-east-1",
	}
	result, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Nil(t, result)
	// The error is wrapped by CreateError as "invalid S3 operation"
	assert.Contains(t, err.Error(), "invalid S3 operation")
}

func TestS3OperationsNode_Execute_EmptyInputData(t *testing.T) {
	node := NewS3OperationsNode()
	inputData := []model.DataItem{}
	params := map[string]interface{}{
		"operation": "list",
		"bucket":    "my-bucket",
		"region":    "us-east-1",
	}
	// With empty input, the for loop does not execute.
	// initializeS3Client will create a real AWS session (which succeeds without real creds).
	result, err := node.Execute(inputData, params)
	require.NoError(t, err)
	assert.Empty(t, result)
}

// --- S3Config Struct Tests ---

func TestS3Config_FieldsAssignment(t *testing.T) {
	config := &S3Config{
		Region:           "ap-northeast-1",
		AccessKeyID:      "AKID",
		SecretAccessKey:  "SECRET",
		SessionToken:     "TOKEN",
		Endpoint:         "http://minio:9000",
		S3ForcePathStyle: true,
		DisableSSL:       true,
	}
	assert.Equal(t, "ap-northeast-1", config.Region)
	assert.Equal(t, "AKID", config.AccessKeyID)
	assert.Equal(t, "SECRET", config.SecretAccessKey)
	assert.Equal(t, "TOKEN", config.SessionToken)
	assert.Equal(t, "http://minio:9000", config.Endpoint)
	assert.True(t, config.S3ForcePathStyle)
	assert.True(t, config.DisableSSL)
}

// --- S3Operation Struct Tests ---

func TestS3Operation_StructDefaults(t *testing.T) {
	op := &S3Operation{}
	assert.Empty(t, op.Operation)
	assert.Empty(t, op.Bucket)
	assert.Empty(t, op.Key)
	assert.Empty(t, op.Source)
	assert.Empty(t, op.Destination)
	assert.Empty(t, op.Prefix)
	assert.Nil(t, op.CopySource)
	assert.Nil(t, op.Metadata)
	assert.Empty(t, op.ContentType)
	assert.Empty(t, op.ACL)
	assert.Empty(t, op.StorageClass)
	assert.False(t, op.Versioning)
	assert.Nil(t, op.Lifecycle)
	assert.Nil(t, op.Encryption)
	assert.Nil(t, op.Options)
}

// --- S3OperationResult Struct Tests ---

func TestS3OperationResult_StructDefaults(t *testing.T) {
	result := &S3OperationResult{}
	assert.Empty(t, result.Operation)
	assert.False(t, result.Success)
	assert.Empty(t, result.Bucket)
	assert.Empty(t, result.Key)
	assert.Nil(t, result.Objects)
	assert.Nil(t, result.Data)
	assert.Nil(t, result.Metadata)
	assert.Empty(t, result.ContentType)
	assert.Equal(t, int64(0), result.Size)
	assert.Empty(t, result.ETag)
	assert.Empty(t, result.VersionID)
	assert.Empty(t, result.Location)
}

// --- S3Object Struct Tests ---

func TestS3Object_Fields(t *testing.T) {
	obj := S3Object{
		Key:          "docs/readme.md",
		Size:         2048,
		ETag:         "d41d8cd98f00b204e9800998ecf8427e",
		StorageClass: "STANDARD",
		Owner:        "owner-display-name",
		ContentType:  "text/markdown",
		VersionID:    "v123",
		IsLatest:     true,
	}
	assert.Equal(t, "docs/readme.md", obj.Key)
	assert.Equal(t, int64(2048), obj.Size)
	assert.Equal(t, "STANDARD", obj.StorageClass)
	assert.True(t, obj.IsLatest)
}

// --- S3CopySource Struct Tests ---

func TestS3CopySource_Fields(t *testing.T) {
	cs := &S3CopySource{
		Bucket:    "source-bucket",
		Key:       "source/key.txt",
		VersionID: "abc123",
	}
	assert.Equal(t, "source-bucket", cs.Bucket)
	assert.Equal(t, "source/key.txt", cs.Key)
	assert.Equal(t, "abc123", cs.VersionID)
}

// --- S3Encryption Struct Tests ---

func TestS3Encryption_Fields(t *testing.T) {
	enc := &S3Encryption{
		Method:      "SSE-KMS",
		KMSKeyID:    "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
		CustomerKey: "",
	}
	assert.Equal(t, "SSE-KMS", enc.Method)
	assert.NotEmpty(t, enc.KMSKeyID)
	assert.Empty(t, enc.CustomerKey)
}

// --- S3Lifecycle Struct Tests ---

func TestS3LifecycleConfig_Fields(t *testing.T) {
	lifecycle := &S3LifecycleConfig{
		Rules: []S3LifecycleRule{
			{
				ID:     "archive-rule",
				Status: "Enabled",
				Filter: &S3LifecycleFilter{
					Prefix: "logs/",
				},
				Transitions: []S3LifecycleTransition{
					{Days: 30, StorageClass: "GLACIER"},
				},
				Expiration: &S3LifecycleExpiration{
					Days: 365,
				},
				AbortIncompleteMultipartUpload: &S3AbortIncompleteMultipartUpload{
					DaysAfterInitiation: 7,
				},
			},
		},
	}
	require.Len(t, lifecycle.Rules, 1)
	rule := lifecycle.Rules[0]
	assert.Equal(t, "archive-rule", rule.ID)
	assert.Equal(t, "Enabled", rule.Status)
	assert.Equal(t, "logs/", rule.Filter.Prefix)
	require.Len(t, rule.Transitions, 1)
	assert.Equal(t, 30, rule.Transitions[0].Days)
	assert.Equal(t, "GLACIER", rule.Transitions[0].StorageClass)
	assert.Equal(t, 365, rule.Expiration.Days)
	assert.Equal(t, 7, rule.AbortIncompleteMultipartUpload.DaysAfterInitiation)
}

// --- initializeS3Client Tests ---

func TestS3OperationsNode_InitializeS3Client_WithCredentials(t *testing.T) {
	node := NewS3OperationsNode()
	config := &S3Config{
		Region:         "us-west-2",
		AccessKeyID:    "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}
	err := node.initializeS3Client(config)
	require.NoError(t, err)
	assert.NotNil(t, node.session)
	assert.NotNil(t, node.s3Client)
	assert.NotNil(t, node.uploader)
	assert.NotNil(t, node.downloader)
}

func TestS3OperationsNode_InitializeS3Client_WithoutCredentials(t *testing.T) {
	node := NewS3OperationsNode()
	config := &S3Config{
		Region: "us-east-1",
	}
	err := node.initializeS3Client(config)
	require.NoError(t, err)
	assert.NotNil(t, node.session)
	assert.NotNil(t, node.s3Client)
	assert.NotNil(t, node.uploader)
	assert.NotNil(t, node.downloader)
}

func TestS3OperationsNode_InitializeS3Client_CustomEndpoint(t *testing.T) {
	node := NewS3OperationsNode()
	config := &S3Config{
		Region:           "us-east-1",
		Endpoint:         "http://localhost:9000",
		S3ForcePathStyle: true,
		DisableSSL:       true,
	}
	err := node.initializeS3Client(config)
	require.NoError(t, err)
	assert.NotNil(t, node.s3Client)
}

// --- detectContentType Tests ---

func TestS3OperationsNode_DetectContentType_ByExtension(t *testing.T) {
	node := NewS3OperationsNode()

	tests := []struct {
		key      string
		expected string
	}{
		{"file.json", "application/json"},
		{"file.xml", "application/xml"},
		{"file.html", "text/html"},
		{"file.css", "text/css"},
		{"file.js", "application/javascript"},
		{"file.jpg", "image/jpeg"},
		{"file.jpeg", "image/jpeg"},
		{"file.png", "image/png"},
		{"file.gif", "image/gif"},
		{"file.pdf", "application/pdf"},
		{"file.zip", "application/zip"},
		{"file.txt", "text/plain"},
		{"file.csv", "text/csv"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			contentType := node.detectContentType(tt.key, nil)
			assert.Equal(t, tt.expected, contentType)
		})
	}
}

func TestS3OperationsNode_DetectContentType_JSONData(t *testing.T) {
	node := NewS3OperationsNode()

	// When extension is unknown but data starts with '{', detect as JSON
	contentType := node.detectContentType("file.unknown", []byte(`{"key":"value"}`))
	assert.Equal(t, "application/json", contentType)

	// When data starts with '['
	contentType = node.detectContentType("file.dat", []byte(`[1,2,3]`))
	assert.Equal(t, "application/json", contentType)
}

func TestS3OperationsNode_DetectContentType_UnknownFallback(t *testing.T) {
	node := NewS3OperationsNode()

	// Unknown extension and non-JSON data
	contentType := node.detectContentType("file.bin", []byte{0x00, 0x01, 0x02})
	assert.Equal(t, "application/octet-stream", contentType)
}

func TestS3OperationsNode_DetectContentType_EmptyData(t *testing.T) {
	node := NewS3OperationsNode()

	// Unknown extension with empty data
	contentType := node.detectContentType("file.unknown", []byte{})
	assert.Equal(t, "application/octet-stream", contentType)
}

func TestS3OperationsNode_DetectContentType_NilData(t *testing.T) {
	node := NewS3OperationsNode()

	// Unknown extension with nil data
	contentType := node.detectContentType("file.unknown", nil)
	assert.Equal(t, "application/octet-stream", contentType)
}

func TestS3OperationsNode_DetectContentType_PathWithDirectories(t *testing.T) {
	node := NewS3OperationsNode()

	// Key with directory prefix
	contentType := node.detectContentType("uploads/images/photo.png", nil)
	assert.Equal(t, "image/png", contentType)
}

func TestS3OperationsNode_DetectContentType_UppercaseExtension(t *testing.T) {
	node := NewS3OperationsNode()

	// The implementation lowercases the extension, so uppercase should work
	contentType := node.detectContentType("file.JSON", nil)
	assert.Equal(t, "application/json", contentType)
}
