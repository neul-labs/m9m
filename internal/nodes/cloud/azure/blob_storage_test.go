package azure

import (
	"testing"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------
// Construction tests
// ---------------------------------------------------------------

func TestNewBlobStorageNode(t *testing.T) {
	node := NewBlobStorageNode()
	require.NotNil(t, node, "NewBlobStorageNode should return a non-nil node")
	require.NotNil(t, node.BaseNode, "BaseNode should be initialised")
	require.NotNil(t, node.evaluator, "evaluator should be initialised")
	require.NotNil(t, node.containerURLs, "containerURLs map should be initialised")
	require.NotNil(t, node.ctx, "context should be initialised")
}

func TestBlobStorageNode_ImplementsNodeExecutor(t *testing.T) {
	// Compile-time check that BlobStorageNode satisfies the NodeExecutor interface.
	var _ base.NodeExecutor = (*BlobStorageNode)(nil)
}

// ---------------------------------------------------------------
// Description tests
// ---------------------------------------------------------------

func TestBlobStorageNode_Description(t *testing.T) {
	node := NewBlobStorageNode()
	desc := node.Description()

	assert.Equal(t, "Azure Blob Storage", desc.Name)
	assert.Equal(t, "cloud", desc.Category)
	assert.Contains(t, desc.Description, "Azure Blob Storage", "Description should mention Azure Blob Storage")
}

// ---------------------------------------------------------------
// ValidateParameters tests
// ---------------------------------------------------------------

func TestBlobStorageNode_ValidateParameters(t *testing.T) {
	node := NewBlobStorageNode()

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
				"container": "my-container",
			},
			expectErr: false,
		},
		{
			name: "valid params with download operation",
			params: map[string]interface{}{
				"operation": "download",
				"container": "my-container",
			},
			expectErr: false,
		},
		{
			name: "valid params with list operation",
			params: map[string]interface{}{
				"operation": "list",
				"container": "my-container",
			},
			expectErr: false,
		},
		{
			name: "valid params with delete operation",
			params: map[string]interface{}{
				"operation": "delete",
				"container": "my-container",
			},
			expectErr: false,
		},
		{
			name: "valid params with copy operation",
			params: map[string]interface{}{
				"operation": "copy",
				"container": "my-container",
			},
			expectErr: false,
		},
		{
			name: "valid params with get_properties operation",
			params: map[string]interface{}{
				"operation": "get_properties",
				"container": "my-container",
			},
			expectErr: false,
		},
		{
			name: "valid params with set_properties operation",
			params: map[string]interface{}{
				"operation": "set_properties",
				"container": "my-container",
			},
			expectErr: false,
		},
		{
			name: "valid params with get_metadata operation",
			params: map[string]interface{}{
				"operation": "get_metadata",
				"container": "my-container",
			},
			expectErr: false,
		},
		{
			name: "valid params with set_metadata operation",
			params: map[string]interface{}{
				"operation": "set_metadata",
				"container": "my-container",
			},
			expectErr: false,
		},
		{
			name: "valid params with create_container operation",
			params: map[string]interface{}{
				"operation": "create_container",
				"container": "my-container",
			},
			expectErr: false,
		},
		{
			name: "valid params with delete_container operation",
			params: map[string]interface{}{
				"operation": "delete_container",
				"container": "my-container",
			},
			expectErr: false,
		},
		{
			name: "valid params with list_containers operation",
			params: map[string]interface{}{
				"operation": "list_containers",
				"container": "my-container",
			},
			expectErr: false,
		},
		{
			name: "valid params with acquire_lease operation",
			params: map[string]interface{}{
				"operation": "acquire_lease",
				"container": "my-container",
			},
			expectErr: false,
		},
		{
			name: "valid params with generate_sas_url operation",
			params: map[string]interface{}{
				"operation": "generate_sas_url",
				"container": "my-container",
			},
			expectErr: false,
		},
		{
			name: "missing operation parameter",
			params: map[string]interface{}{
				"container": "my-container",
			},
			expectErr:   true,
			errContains: "operation parameter is required",
		},
		{
			name: "missing container parameter",
			params: map[string]interface{}{
				"operation": "upload",
			},
			expectErr:   true,
			errContains: "container parameter is required",
		},
		{
			name:        "missing both operation and container",
			params:      map[string]interface{}{},
			expectErr:   true,
			errContains: "operation parameter is required",
		},
		{
			name: "invalid operation",
			params: map[string]interface{}{
				"operation": "invalid_op",
				"container": "my-container",
			},
			expectErr:   true,
			errContains: "invalid operation: invalid_op",
		},
		{
			name: "another invalid operation",
			params: map[string]interface{}{
				"operation": "move",
				"container": "my-container",
			},
			expectErr:   true,
			errContains: "invalid operation: move",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := node.ValidateParameters(tt.params)
			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------
// Execute – missing/invalid credentials (no real Azure calls)
// ---------------------------------------------------------------

func TestBlobStorageNode_Execute_MissingAccountName(t *testing.T) {
	node := NewBlobStorageNode()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"test": "data"}},
	}

	params := map[string]interface{}{
		"operation": "list",
		"container": "my-container",
		// accountName is intentionally missing
		"accountKey": "dGVzdGtleQ==",
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid Azure configuration")
}

func TestBlobStorageNode_Execute_MissingAuthMethod(t *testing.T) {
	node := NewBlobStorageNode()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"test": "data"}},
	}

	params := map[string]interface{}{
		"operation":   "list",
		"container":   "my-container",
		"accountName": "testaccount",
		// No accountKey, sasToken, or connectionString provided
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize Blob Storage client")
}

func TestBlobStorageNode_Execute_InvalidAccountKey(t *testing.T) {
	node := NewBlobStorageNode()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"test": "data"}},
	}

	// An invalid base64 account key should cause credential initialization to fail
	params := map[string]interface{}{
		"operation":   "list",
		"container":   "my-container",
		"accountName": "testaccount",
		"accountKey":  "not-valid-base64!!!",
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize Blob Storage client")
}

func TestBlobStorageNode_Execute_EmptyInputData(t *testing.T) {
	node := NewBlobStorageNode()

	// With valid credentials (using SAS token to avoid key decode errors)
	// and empty input data, the loop body never executes so we get empty results.
	params := map[string]interface{}{
		"operation":   "list",
		"container":   "my-container",
		"accountName": "testaccount",
		"sasToken":    "sv=2020-08-04&ss=b&srt=sco&sp=rwdlac&se=2099-01-01T00:00:00Z&st=2020-01-01T00:00:00Z&spr=https&sig=fakesig",
	}

	result, err := node.Execute([]model.DataItem{}, params)
	require.NoError(t, err)
	assert.Empty(t, result, "Empty input should produce nil/empty output")
}

func TestBlobStorageNode_Execute_MissingOperationParam(t *testing.T) {
	node := NewBlobStorageNode()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"test": "data"}},
	}

	params := map[string]interface{}{
		"accountName": "testaccount",
		"sasToken":    "sv=2020-08-04&ss=b&srt=sco&sp=rwdlac&se=2099-01-01T00:00:00Z&st=2020-01-01T00:00:00Z&spr=https&sig=fakesig",
		"container":   "my-container",
		// operation is intentionally missing
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid blob operation")
}

func TestBlobStorageNode_Execute_MissingContainerParam(t *testing.T) {
	node := NewBlobStorageNode()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"test": "data"}},
	}

	params := map[string]interface{}{
		"accountName": "testaccount",
		"sasToken":    "sv=2020-08-04&ss=b&srt=sco&sp=rwdlac&se=2099-01-01T00:00:00Z&st=2020-01-01T00:00:00Z&spr=https&sig=fakesig",
		"operation":   "list",
		// container is intentionally missing
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid blob operation")
}

// ---------------------------------------------------------------
// parseAzureConfig tests
// ---------------------------------------------------------------

func TestBlobStorageNode_ParseAzureConfig_AllFields(t *testing.T) {
	node := NewBlobStorageNode()

	params := map[string]interface{}{
		"accountName":        "myaccount",
		"accountKey":         "mykey",
		"sasToken":           "mysas",
		"connectionString":   "DefaultEndpointsProtocol=https;AccountName=myaccount;AccountKey=mykey;EndpointSuffix=core.windows.net",
		"tenantId":           "my-tenant",
		"clientId":           "my-client",
		"clientSecret":       "my-secret",
		"useManagedIdentity": true,
	}

	config, err := node.parseAzureConfig(params)
	require.NoError(t, err)
	assert.Equal(t, "myaccount", config.AccountName)
	assert.Equal(t, "mykey", config.AccountKey)
	assert.Equal(t, "mysas", config.SASToken)
	assert.Contains(t, config.ConnectionString, "AccountName=myaccount")
	assert.Equal(t, "my-tenant", config.TenantID)
	assert.Equal(t, "my-client", config.ClientID)
	assert.Equal(t, "my-secret", config.ClientSecret)
	assert.True(t, config.UseManagedIdentity)
}

func TestBlobStorageNode_ParseAzureConfig_OnlyAccountName(t *testing.T) {
	node := NewBlobStorageNode()

	params := map[string]interface{}{
		"accountName": "myaccount",
	}

	config, err := node.parseAzureConfig(params)
	require.NoError(t, err)
	assert.Equal(t, "myaccount", config.AccountName)
	assert.Empty(t, config.AccountKey)
	assert.Empty(t, config.SASToken)
	assert.Empty(t, config.ConnectionString)
	assert.False(t, config.UseManagedIdentity)
}

func TestBlobStorageNode_ParseAzureConfig_MissingAccountName(t *testing.T) {
	node := NewBlobStorageNode()

	params := map[string]interface{}{
		"accountKey": "mykey",
	}

	_, err := node.parseAzureConfig(params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accountName is required")
}

func TestBlobStorageNode_ParseAzureConfig_EmptyParams(t *testing.T) {
	node := NewBlobStorageNode()

	_, err := node.parseAzureConfig(map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accountName is required")
}

func TestBlobStorageNode_ParseAzureConfig_WrongTypeAccountName(t *testing.T) {
	node := NewBlobStorageNode()

	params := map[string]interface{}{
		"accountName": 12345, // wrong type
	}

	_, err := node.parseAzureConfig(params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accountName is required")
}

// ---------------------------------------------------------------
// parseBlobOperation tests
// ---------------------------------------------------------------

func TestBlobStorageNode_ParseBlobOperation_ValidFullParams(t *testing.T) {
	node := NewBlobStorageNode()

	params := map[string]interface{}{
		"operation":   "upload",
		"container":   "my-container",
		"blob":        "path/to/blob.txt",
		"source":      "myfield",
		"destination": "dest-path",
		"prefix":      "logs/",
		"contentType": "text/plain",
		"accessTier":  "Hot",
		"metadata": map[string]interface{}{
			"author": "test-user",
			"env":    "production",
		},
		"options": map[string]interface{}{
			"maxResults": 100,
		},
	}

	op, err := node.parseBlobOperation(params)
	require.NoError(t, err)
	assert.Equal(t, "upload", op.Operation)
	assert.Equal(t, "my-container", op.Container)
	assert.Equal(t, "path/to/blob.txt", op.Blob)
	assert.Equal(t, "myfield", op.Source)
	assert.Equal(t, "dest-path", op.Destination)
	assert.Equal(t, "logs/", op.Prefix)
	assert.Equal(t, "text/plain", op.ContentType)
	assert.Equal(t, "Hot", op.AccessTier)
	assert.Equal(t, "test-user", op.Metadata["author"])
	assert.Equal(t, "production", op.Metadata["env"])
	assert.Equal(t, 100, op.Options["maxResults"])
}

func TestBlobStorageNode_ParseBlobOperation_MinimalParams(t *testing.T) {
	node := NewBlobStorageNode()

	params := map[string]interface{}{
		"operation": "list",
		"container": "my-container",
	}

	op, err := node.parseBlobOperation(params)
	require.NoError(t, err)
	assert.Equal(t, "list", op.Operation)
	assert.Equal(t, "my-container", op.Container)
	assert.Empty(t, op.Blob)
	assert.Empty(t, op.Source)
	assert.Empty(t, op.Prefix)
	assert.Empty(t, op.ContentType)
	assert.NotNil(t, op.Options, "Options map should be initialised even when not provided")
}

func TestBlobStorageNode_ParseBlobOperation_MissingOperation(t *testing.T) {
	node := NewBlobStorageNode()

	params := map[string]interface{}{
		"container": "my-container",
	}

	_, err := node.parseBlobOperation(params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "operation parameter is required")
}

func TestBlobStorageNode_ParseBlobOperation_MissingContainer(t *testing.T) {
	node := NewBlobStorageNode()

	params := map[string]interface{}{
		"operation": "upload",
	}

	_, err := node.parseBlobOperation(params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "container parameter is required")
}

func TestBlobStorageNode_ParseBlobOperation_WrongTypeOperation(t *testing.T) {
	node := NewBlobStorageNode()

	params := map[string]interface{}{
		"operation": 42,
		"container": "my-container",
	}

	_, err := node.parseBlobOperation(params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "operation parameter is required")
}

func TestBlobStorageNode_ParseBlobOperation_WrongTypeContainer(t *testing.T) {
	node := NewBlobStorageNode()

	params := map[string]interface{}{
		"operation": "upload",
		"container": 42,
	}

	_, err := node.parseBlobOperation(params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "container parameter is required")
}

func TestBlobStorageNode_ParseBlobOperation_MetadataConvertedToStrings(t *testing.T) {
	node := NewBlobStorageNode()

	params := map[string]interface{}{
		"operation": "upload",
		"container": "my-container",
		"metadata": map[string]interface{}{
			"count":   42,
			"enabled": true,
			"name":    "test",
		},
	}

	op, err := node.parseBlobOperation(params)
	require.NoError(t, err)
	assert.Equal(t, "42", op.Metadata["count"])
	assert.Equal(t, "true", op.Metadata["enabled"])
	assert.Equal(t, "test", op.Metadata["name"])
}

// ---------------------------------------------------------------
// detectContentType tests
// ---------------------------------------------------------------

func TestBlobStorageNode_DetectContentType(t *testing.T) {
	node := NewBlobStorageNode()

	tests := []struct {
		blobName    string
		expected    string
	}{
		{"data.json", "application/json"},
		{"config.xml", "application/xml"},
		{"page.html", "text/html"},
		{"style.css", "text/css"},
		{"app.js", "application/javascript"},
		{"photo.jpg", "image/jpeg"},
		{"photo.jpeg", "image/jpeg"},
		{"image.png", "image/png"},
		{"document.pdf", "application/pdf"},
		{"readme.txt", "text/plain"},
		{"unknown.bin", "application/octet-stream"},
		{"noextension", "application/octet-stream"},
		{"path/to/data.json", "application/json"},
		{"UPPERCASE.JSON", "application/octet-stream"}, // case-sensitive detection
		{"", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.blobName, func(t *testing.T) {
			result := node.detectContentType(tt.blobName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------
// parseConnectionString tests
// ---------------------------------------------------------------

func TestParseConnectionString(t *testing.T) {
	tests := []struct {
		name              string
		connStr           string
		expectedAccount   string
		expectedKey       string
		expectedEndpoint  string
	}{
		{
			name:             "full connection string",
			connStr:          "AccountName=myaccount;AccountKey=mykey;BlobEndpoint=https://myaccount.blob.core.windows.net",
			expectedAccount:  "myaccount",
			expectedKey:      "mykey",
			expectedEndpoint: "https://myaccount.blob.core.windows.net",
		},
		{
			name:             "standard Azure connection string",
			connStr:          "DefaultEndpointsProtocol=https;AccountName=testaccount;AccountKey=dGVzdGtleQ==;EndpointSuffix=core.windows.net",
			expectedAccount:  "testaccount",
			expectedKey:      "dGVzdGtleQ==",
			expectedEndpoint: "",
		},
		{
			name:             "only account name and key",
			connStr:          "AccountName=myaccount;AccountKey=mykey",
			expectedAccount:  "myaccount",
			expectedKey:      "mykey",
			expectedEndpoint: "",
		},
		{
			name:             "empty string",
			connStr:          "",
			expectedAccount:  "",
			expectedKey:      "",
			expectedEndpoint: "",
		},
		{
			name:             "connection string with extra spaces",
			connStr:          " AccountName = myaccount ; AccountKey = mykey ; BlobEndpoint = https://ep.test ",
			expectedAccount:  "myaccount",
			expectedKey:      "mykey",
			expectedEndpoint: "https://ep.test",
		},
		{
			name:             "connection string with no recognized keys",
			connStr:          "SomeOtherKey=value;AnotherKey=anothervalue",
			expectedAccount:  "",
			expectedKey:      "",
			expectedEndpoint: "",
		},
		{
			name:             "connection string with only account name",
			connStr:          "AccountName=onlyname",
			expectedAccount:  "onlyname",
			expectedKey:      "",
			expectedEndpoint: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accountName, accountKey, endpoint := parseConnectionString(tt.connStr)
			assert.Equal(t, tt.expectedAccount, accountName)
			assert.Equal(t, tt.expectedKey, accountKey)
			assert.Equal(t, tt.expectedEndpoint, endpoint)
		})
	}
}

// ---------------------------------------------------------------
// initializeBlobClient tests (validation only, no real Azure calls)
// ---------------------------------------------------------------

func TestBlobStorageNode_InitializeBlobClient_WithSASToken(t *testing.T) {
	node := NewBlobStorageNode()

	config := &AzureConfig{
		AccountName: "testaccount",
		SASToken:    "sv=2020-08-04&ss=b&srt=sco&sp=rwdlac&se=2099-01-01T00:00:00Z&st=2020-01-01T00:00:00Z&spr=https&sig=fakesig",
	}

	err := node.initializeBlobClient(config)
	require.NoError(t, err)
	assert.NotNil(t, node.serviceURL, "serviceURL should be set after initialization with SAS token")
	assert.NotNil(t, node.credential, "credential should be set after initialization with SAS token")
}

func TestBlobStorageNode_InitializeBlobClient_NoAuthMethod(t *testing.T) {
	node := NewBlobStorageNode()

	config := &AzureConfig{
		AccountName: "testaccount",
		// No accountKey, sasToken, or connectionString
	}

	err := node.initializeBlobClient(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication method required")
}

func TestBlobStorageNode_InitializeBlobClient_InvalidConnectionString(t *testing.T) {
	node := NewBlobStorageNode()

	config := &AzureConfig{
		AccountName:      "testaccount",
		ConnectionString: "InvalidConnectionString=nothing",
	}

	err := node.initializeBlobClient(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid connection string")
}

func TestBlobStorageNode_InitializeBlobClient_ValidConnectionString(t *testing.T) {
	node := NewBlobStorageNode()

	// The connection string parser expects AccountName and AccountKey parts.
	// The key must be valid base64 for azblob.NewSharedKeyCredential.
	config := &AzureConfig{
		AccountName:      "testaccount",
		ConnectionString: "AccountName=testaccount;AccountKey=dGVzdGtleWRhdGE=",
	}

	err := node.initializeBlobClient(config)
	require.NoError(t, err)
	assert.NotNil(t, node.serviceURL)
	assert.NotNil(t, node.credential)
}

func TestBlobStorageNode_InitializeBlobClient_ValidAccountKey(t *testing.T) {
	node := NewBlobStorageNode()

	// accountKey must be valid base64 for the Azure SDK
	config := &AzureConfig{
		AccountName: "testaccount",
		AccountKey:  "dGVzdGtleWRhdGE=",
	}

	err := node.initializeBlobClient(config)
	require.NoError(t, err)
	assert.NotNil(t, node.serviceURL)
	assert.NotNil(t, node.credential)
}

func TestBlobStorageNode_InitializeBlobClient_InvalidAccountKey(t *testing.T) {
	node := NewBlobStorageNode()

	config := &AzureConfig{
		AccountName: "testaccount",
		AccountKey:  "not-valid-base64!!!",
	}

	err := node.initializeBlobClient(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create shared key credential")
}

// ---------------------------------------------------------------
// AzureConfig struct tests
// ---------------------------------------------------------------

func TestAzureConfig_DefaultValues(t *testing.T) {
	config := AzureConfig{}
	assert.Empty(t, config.AccountName)
	assert.Empty(t, config.AccountKey)
	assert.Empty(t, config.SASToken)
	assert.Empty(t, config.ConnectionString)
	assert.Empty(t, config.TenantID)
	assert.Empty(t, config.ClientID)
	assert.Empty(t, config.ClientSecret)
	assert.False(t, config.UseManagedIdentity)
}

// ---------------------------------------------------------------
// BlobOperation struct tests
// ---------------------------------------------------------------

func TestBlobOperation_DefaultValues(t *testing.T) {
	op := BlobOperation{}
	assert.Empty(t, op.Operation)
	assert.Empty(t, op.Container)
	assert.Empty(t, op.Blob)
	assert.Nil(t, op.Metadata)
	assert.Nil(t, op.Properties)
	assert.Nil(t, op.LeaseOptions)
	assert.Nil(t, op.CopyOptions)
	assert.Nil(t, op.Options)
}

// ---------------------------------------------------------------
// ValidateParameters – all valid operations
// ---------------------------------------------------------------

func TestBlobStorageNode_ValidateParameters_AllValidOperations(t *testing.T) {
	node := NewBlobStorageNode()

	validOperations := []string{
		"upload", "download", "delete", "list", "copy",
		"get_properties", "set_properties", "get_metadata", "set_metadata",
		"create_container", "delete_container", "list_containers",
		"acquire_lease", "renew_lease", "release_lease", "break_lease",
		"generate_sas_url",
	}

	for _, op := range validOperations {
		t.Run(op, func(t *testing.T) {
			params := map[string]interface{}{
				"operation": op,
				"container": "test-container",
			}
			err := node.ValidateParameters(params)
			assert.NoError(t, err, "operation %q should be valid", op)
		})
	}
}

// ---------------------------------------------------------------
// Multiple node instances are independent
// ---------------------------------------------------------------

func TestBlobStorageNode_IndependentInstances(t *testing.T) {
	node1 := NewBlobStorageNode()
	node2 := NewBlobStorageNode()

	// Verify they are distinct instances
	require.NotSame(t, node1, node2)
	require.NotSame(t, node1.BaseNode, node2.BaseNode)

	// Descriptions should be equal but not the same pointer
	assert.Equal(t, node1.Description(), node2.Description())
}

// ---------------------------------------------------------------
// Execute with connection string containing BlobEndpoint
// ---------------------------------------------------------------

func TestBlobStorageNode_InitializeBlobClient_ConnectionStringWithEndpoint(t *testing.T) {
	node := NewBlobStorageNode()

	config := &AzureConfig{
		AccountName:      "testaccount",
		ConnectionString: "AccountName=testaccount;AccountKey=dGVzdGtleWRhdGE=;BlobEndpoint=https://custom.endpoint.net",
	}

	err := node.initializeBlobClient(config)
	require.NoError(t, err)
	assert.NotNil(t, node.serviceURL)
}

// ---------------------------------------------------------------
// getContainerURL caching behavior
// ---------------------------------------------------------------

func TestBlobStorageNode_GetContainerURL_Caching(t *testing.T) {
	node := NewBlobStorageNode()

	// Initialize with SAS token to set up serviceURL
	config := &AzureConfig{
		AccountName: "testaccount",
		SASToken:    "sv=2020-08-04&ss=b&srt=sco&sp=rwdlac&se=2099-01-01T00:00:00Z&st=2020-01-01T00:00:00Z&spr=https&sig=fakesig",
	}
	err := node.initializeBlobClient(config)
	require.NoError(t, err)

	// Get container URL twice for the same container
	url1, err := node.getContainerURL("test-container")
	require.NoError(t, err)
	require.NotNil(t, url1)

	url2, err := node.getContainerURL("test-container")
	require.NoError(t, err)
	require.NotNil(t, url2)

	// Should return the same cached pointer
	assert.Same(t, url1, url2, "Same container name should return cached URL")

	// Different container name should return a different URL
	url3, err := node.getContainerURL("other-container")
	require.NoError(t, err)
	require.NotNil(t, url3)
	assert.NotSame(t, url1, url3, "Different container name should return different URL")
}
