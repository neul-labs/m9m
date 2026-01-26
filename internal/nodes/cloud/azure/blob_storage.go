package azure

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-storage-blob-go/azblob"

	"github.com/neul-labs/m9m/internal/expressions"
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// BlobStorageNode provides Azure Blob Storage operations
type BlobStorageNode struct {
	*base.BaseNode
	evaluator         *expressions.GojaExpressionEvaluator
	serviceURL        *azblob.ServiceURL
	containerURLs     map[string]*azblob.ContainerURL
	credential        azblob.Credential
	ctx               context.Context
}

// AzureConfig holds Azure authentication configuration
type AzureConfig struct {
	AccountName   string `json:"accountName"`
	AccountKey    string `json:"accountKey,omitempty"`
	SASToken      string `json:"sasToken,omitempty"`
	ConnectionString string `json:"connectionString,omitempty"`
	TenantID      string `json:"tenantId,omitempty"`
	ClientID      string `json:"clientId,omitempty"`
	ClientSecret  string `json:"clientSecret,omitempty"`
	UseManagedIdentity bool `json:"useManagedIdentity"`
}

// BlobOperation represents a Blob Storage operation
type BlobOperation struct {
	Operation     string                 `json:"operation"`
	Container     string                 `json:"container"`
	Blob          string                 `json:"blob,omitempty"`
	Source        string                 `json:"source,omitempty"`
	Destination   string                 `json:"destination,omitempty"`
	Prefix        string                 `json:"prefix,omitempty"`
	ContentType   string                 `json:"contentType,omitempty"`
	Metadata      map[string]string      `json:"metadata,omitempty"`
	Properties    *BlobProperties        `json:"properties,omitempty"`
	AccessTier    string                 `json:"accessTier,omitempty"`
	LeaseOptions  *BlobLeaseOptions      `json:"leaseOptions,omitempty"`
	CopyOptions   *BlobCopyOptions       `json:"copyOptions,omitempty"`
	Options       map[string]interface{} `json:"options,omitempty"`
}

// BlobProperties defines blob properties
type BlobProperties struct {
	ContentType        string            `json:"contentType,omitempty"`
	ContentEncoding    string            `json:"contentEncoding,omitempty"`
	ContentLanguage    string            `json:"contentLanguage,omitempty"`
	ContentDisposition string            `json:"contentDisposition,omitempty"`
	CacheControl       string            `json:"cacheControl,omitempty"`
	ContentMD5         []byte            `json:"contentMD5,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
}

// BlobLeaseOptions defines blob lease options
type BlobLeaseOptions struct {
	Action   string `json:"action"` // acquire, renew, change, release, break
	Duration int32  `json:"duration,omitempty"`
	LeaseID  string `json:"leaseId,omitempty"`
}

// BlobCopyOptions defines blob copy options
type BlobCopyOptions struct {
	SourceURL      string            `json:"sourceUrl"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	AccessTier     string            `json:"accessTier,omitempty"`
	RehydratePriority string         `json:"rehydratePriority,omitempty"`
}

// BlobInfo represents Azure Blob information
type BlobInfo struct {
	Name               string            `json:"name"`
	Container          string            `json:"container"`
	Size               int64             `json:"size"`
	ContentType        string            `json:"contentType"`
	LastModified       time.Time         `json:"lastModified"`
	ETag               string            `json:"etag"`
	ContentMD5         string            `json:"contentMD5,omitempty"`
	ContentEncoding    string            `json:"contentEncoding,omitempty"`
	ContentLanguage    string            `json:"contentLanguage,omitempty"`
	ContentDisposition string            `json:"contentDisposition,omitempty"`
	CacheControl       string            `json:"cacheControl,omitempty"`
	BlobType           string            `json:"blobType"`
	AccessTier         string            `json:"accessTier,omitempty"`
	AccessTierInferred bool              `json:"accessTierInferred"`
	ArchiveStatus      string            `json:"archiveStatus,omitempty"`
	CreationTime       time.Time         `json:"creationTime"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	LeaseStatus        string            `json:"leaseStatus,omitempty"`
	LeaseState         string            `json:"leaseState,omitempty"`
	LeaseDuration      string            `json:"leaseDuration,omitempty"`
	ServerEncrypted    bool              `json:"serverEncrypted"`
	DeletedTime        *time.Time        `json:"deletedTime,omitempty"`
	RemainingRetentionDays *int32        `json:"remainingRetentionDays,omitempty"`
}

// ContainerInfo represents Azure Container information
type ContainerInfo struct {
	Name                   string            `json:"name"`
	LastModified           time.Time         `json:"lastModified"`
	ETag                   string            `json:"etag"`
	LeaseStatus            string            `json:"leaseStatus"`
	LeaseState             string            `json:"leaseState"`
	LeaseDuration          string            `json:"leaseDuration,omitempty"`
	PublicAccess           string            `json:"publicAccess"`
	HasImmutabilityPolicy  bool              `json:"hasImmutabilityPolicy"`
	HasLegalHold           bool              `json:"hasLegalHold"`
	Metadata               map[string]string `json:"metadata,omitempty"`
	DefaultEncryptionScope string            `json:"defaultEncryptionScope,omitempty"`
	DenyEncryptionScopeOverride bool         `json:"denyEncryptionScopeOverride"`
}

// BlobOperationResult represents the result of a blob operation
type BlobOperationResult struct {
	Operation     string                 `json:"operation"`
	Success       bool                   `json:"success"`
	Container     string                 `json:"container"`
	Blob          string                 `json:"blob,omitempty"`
	Blobs         []BlobInfo             `json:"blobs,omitempty"`
	Containers    []ContainerInfo        `json:"containers,omitempty"`
	Data          interface{}            `json:"data,omitempty"`
	Size          int64                  `json:"size,omitempty"`
	ContentType   string                 `json:"contentType,omitempty"`
	ETag          string                 `json:"etag,omitempty"`
	URL           string                 `json:"url,omitempty"`
	LeaseID       string                 `json:"leaseId,omitempty"`
	ExecutionTime time.Duration          `json:"executionTime"`
	Error         string                 `json:"error,omitempty"`
}

// NewBlobStorageNode creates a new Blob Storage operations node
func NewBlobStorageNode() *BlobStorageNode {
	return &BlobStorageNode{
		BaseNode:      base.NewBaseNode(base.NodeDescription{Name: "Azure Blob Storage", Description: "Azure Blob Storage operations", Category: "cloud"}),
		evaluator:     expressions.NewGojaExpressionEvaluator(expressions.DefaultEvaluatorConfig()),
		containerURLs: make(map[string]*azblob.ContainerURL),
		ctx:           context.Background(),
	}
}

// Execute performs Blob Storage operations
func (n *BlobStorageNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	var results []model.DataItem

	// Parse Azure configuration
	config, err := n.parseAzureConfig(nodeParams)
	if err != nil {
		return nil, n.CreateError("invalid Azure configuration", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Initialize Blob Storage client
	if err := n.initializeBlobClient(config); err != nil {
		return nil, n.CreateError("failed to initialize Blob Storage client", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Parse operation configuration
	operation, err := n.parseBlobOperation(nodeParams)
	if err != nil {
		return nil, n.CreateError("invalid blob operation", map[string]interface{}{
			"error": err.Error(),
		})
	}

	for index, item := range inputData {
		// Create expression context
		exprContext := &expressions.ExpressionContext{
			ActiveNodeName:      "Azure Blob Storage",
			RunIndex:           0,
			ItemIndex:          index,
			Mode:               expressions.ModeManual,
			ConnectionInputData: []model.DataItem{item},
			AdditionalKeys:     &expressions.AdditionalKeys{},
		}

		// Execute blob operation
		result, err := n.executeBlobOperation(operation, exprContext, item)
		if err != nil {
			return nil, n.CreateError("Blob Storage operation failed", map[string]interface{}{
				"operation": operation.Operation,
				"error":     err.Error(),
			})
		}

		// Create result item - convert BlobOperationResult to map
		resultJSON := map[string]interface{}{
			"operation":     result.Operation,
			"success":       result.Success,
			"container":     result.Container,
			"blob":          result.Blob,
			"size":          result.Size,
			"contentType":   result.ContentType,
			"etag":          result.ETag,
			"url":           result.URL,
			"executionTime": result.ExecutionTime.String(),
		}
		if result.LeaseID != "" {
			resultJSON["leaseId"] = result.LeaseID
		}
		if len(result.Blobs) > 0 {
			resultJSON["blobs"] = result.Blobs
		}
		if len(result.Containers) > 0 {
			resultJSON["containers"] = result.Containers
		}
		if result.Error != "" {
			resultJSON["error"] = result.Error
		}

		resultItem := model.DataItem{
			JSON: resultJSON,
		}

		// Add binary data if operation downloaded data
		if operation.Operation == "download" && result.Data != nil {
			if dataBytes, ok := result.Data.([]byte); ok {
				resultItem.Binary = map[string]model.BinaryData{
					"data": {
						Data:     base64.StdEncoding.EncodeToString(dataBytes),
						MimeType: result.ContentType,
						FileName: filepath.Base(result.Blob),
					},
				}
			}
		}

		results = append(results, resultItem)
	}

	return results, nil
}

// executeBlobOperation executes the specified blob operation
func (n *BlobStorageNode) executeBlobOperation(operation *BlobOperation, context *expressions.ExpressionContext, item model.DataItem) (*BlobOperationResult, error) {
	startTime := time.Now()

	result := &BlobOperationResult{
		Operation: operation.Operation,
		Container: operation.Container,
	}

	var err error

	switch strings.ToLower(operation.Operation) {
	case "upload":
		err = n.uploadBlob(operation, context, item, result)
	case "download":
		err = n.downloadBlob(operation, context, item, result)
	case "delete":
		err = n.deleteBlob(operation, context, item, result)
	case "list":
		err = n.listBlobs(operation, context, item, result)
	case "copy":
		err = n.copyBlob(operation, context, item, result)
	case "get_properties":
		err = n.getBlobProperties(operation, context, item, result)
	case "set_properties":
		err = n.setBlobProperties(operation, context, item, result)
	case "get_metadata":
		err = n.getBlobMetadata(operation, context, item, result)
	case "set_metadata":
		err = n.setBlobMetadata(operation, context, item, result)
	case "create_container":
		err = n.createContainer(operation, context, item, result)
	case "delete_container":
		err = n.deleteContainer(operation, context, item, result)
	case "list_containers":
		err = n.listContainers(operation, context, item, result)
	case "acquire_lease":
		err = n.acquireLease(operation, context, item, result)
	case "renew_lease":
		err = n.renewLease(operation, context, item, result)
	case "release_lease":
		err = n.releaseLease(operation, context, item, result)
	case "break_lease":
		err = n.breakLease(operation, context, item, result)
	case "generate_sas_url":
		err = n.generateSASURL(operation, context, item, result)
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

// Blob operations
func (n *BlobStorageNode) uploadBlob(operation *BlobOperation, context *expressions.ExpressionContext, item model.DataItem, result *BlobOperationResult) error {
	// Evaluate blob name
	blobName, err := n.evaluateExpression(operation.Blob, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate blob name: %w", err)
	}
	result.Blob = fmt.Sprintf("%v", blobName)

	// Get data to upload
	var data []byte
	var contentType string

	if operation.Source != "" {
		// Source is an expression or field reference
		sourceData, err := n.evaluateExpression(operation.Source, context)
		if err != nil {
			return fmt.Errorf("failed to evaluate source: %w", err)
		}

		switch sd := sourceData.(type) {
		case []byte:
			data = sd
		case string:
			data = []byte(sd)
		default:
			// Convert to JSON
			jsonData, err := json.Marshal(sd)
			if err != nil {
				return fmt.Errorf("failed to serialize data: %w", err)
			}
			data = jsonData
			contentType = "application/json"
		}
	} else if item.Binary != nil && len(item.Binary) > 0 {
		// Use binary data from item - get first binary entry
		for _, bd := range item.Binary {
			decoded, err := base64.StdEncoding.DecodeString(bd.Data)
			if err != nil {
				return fmt.Errorf("failed to decode binary data: %w", err)
			}
			data = decoded
			if bd.MimeType != "" && contentType == "" {
				contentType = bd.MimeType
			}
			break
		}
	} else {
		// Use JSON data from item
		jsonData, err := json.Marshal(item.JSON)
		if err != nil {
			return fmt.Errorf("failed to serialize item data: %w", err)
		}
		data = jsonData
		contentType = "application/json"
	}

	// Set content type
	if operation.ContentType != "" {
		contentType = operation.ContentType
	} else if contentType == "" {
		contentType = n.detectContentType(result.Blob)
	}

	// Get container URL
	containerURL, err := n.getContainerURL(operation.Container)
	if err != nil {
		return fmt.Errorf("failed to get container URL: %w", err)
	}

	// Get blob URL
	blobURL := containerURL.NewBlockBlobURL(result.Blob)

	// Prepare upload options
	uploadOptions := azblob.UploadToBlockBlobOptions{}

	// Set properties
	if operation.Properties != nil {
		uploadOptions.BlobHTTPHeaders = azblob.BlobHTTPHeaders{
			ContentType:        operation.Properties.ContentType,
			ContentEncoding:    operation.Properties.ContentEncoding,
			ContentLanguage:    operation.Properties.ContentLanguage,
			ContentDisposition: operation.Properties.ContentDisposition,
			CacheControl:       operation.Properties.CacheControl,
			ContentMD5:         operation.Properties.ContentMD5,
		}
		uploadOptions.Metadata = operation.Properties.Metadata
	} else {
		uploadOptions.BlobHTTPHeaders = azblob.BlobHTTPHeaders{
			ContentType: contentType,
		}
	}

	// Set metadata
	if len(operation.Metadata) > 0 {
		uploadOptions.Metadata = operation.Metadata
	}

	// Upload blob
	_, err = azblob.UploadBufferToBlockBlob(n.ctx, data, blobURL, uploadOptions)
	if err != nil {
		return fmt.Errorf("failed to upload blob: %w", err)
	}

	// Get blob properties after upload
	props, err := blobURL.GetProperties(n.ctx, azblob.BlobAccessConditions{}, azblob.ClientProvidedKeyOptions{})
	if err == nil {
		result.Size = props.ContentLength()
		result.ContentType = props.ContentType()
		result.ETag = string(props.ETag())
	}

	result.Size = int64(len(data))
	return nil
}

func (n *BlobStorageNode) downloadBlob(operation *BlobOperation, context *expressions.ExpressionContext, item model.DataItem, result *BlobOperationResult) error {
	// Evaluate blob name
	blobName, err := n.evaluateExpression(operation.Blob, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate blob name: %w", err)
	}
	result.Blob = fmt.Sprintf("%v", blobName)

	// Get container URL
	containerURL, err := n.getContainerURL(operation.Container)
	if err != nil {
		return fmt.Errorf("failed to get container URL: %w", err)
	}

	// Get blob URL
	blobURL := containerURL.NewBlobURL(result.Blob)

	// Download blob
	downloadResponse, err := blobURL.Download(n.ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false, azblob.ClientProvidedKeyOptions{})
	if err != nil {
		return fmt.Errorf("failed to download blob: %w", err)
	}

	// Read body
	bodyStream := downloadResponse.Body(azblob.RetryReaderOptions{MaxRetryRequests: 20})
	defer bodyStream.Close()

	data, err := io.ReadAll(bodyStream)
	if err != nil {
		return fmt.Errorf("failed to read blob data: %w", err)
	}

	result.Data = data
	result.Size = int64(len(data))
	result.ContentType = downloadResponse.ContentType()
	result.ETag = string(downloadResponse.ETag())

	return nil
}

func (n *BlobStorageNode) deleteBlob(operation *BlobOperation, context *expressions.ExpressionContext, item model.DataItem, result *BlobOperationResult) error {
	// Evaluate blob name
	blobName, err := n.evaluateExpression(operation.Blob, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate blob name: %w", err)
	}
	result.Blob = fmt.Sprintf("%v", blobName)

	// Get container URL
	containerURL, err := n.getContainerURL(operation.Container)
	if err != nil {
		return fmt.Errorf("failed to get container URL: %w", err)
	}

	// Get blob URL
	blobURL := containerURL.NewBlobURL(result.Blob)

	// Delete blob
	deleteOptions := azblob.DeleteSnapshotsOptionInclude
	_, err = blobURL.Delete(n.ctx, deleteOptions, azblob.BlobAccessConditions{})
	if err != nil {
		return fmt.Errorf("failed to delete blob: %w", err)
	}

	return nil
}

func (n *BlobStorageNode) listBlobs(operation *BlobOperation, context *expressions.ExpressionContext, item model.DataItem, result *BlobOperationResult) error {
	// Get container URL
	containerURL, err := n.getContainerURL(operation.Container)
	if err != nil {
		return fmt.Errorf("failed to get container URL: %w", err)
	}

	// Prepare list options
	listOptions := azblob.ListBlobsSegmentOptions{}

	if operation.Prefix != "" {
		prefix, err := n.evaluateExpression(operation.Prefix, context)
		if err != nil {
			return fmt.Errorf("failed to evaluate prefix: %w", err)
		}
		listOptions.Prefix = fmt.Sprintf("%v", prefix)
	}

	// Handle pagination
	if maxResults, ok := operation.Options["maxResults"].(int); ok {
		listOptions.MaxResults = int32(maxResults)
	}

	listOptions.Details = azblob.BlobListingDetails{
		Metadata: true,
		Snapshots: false,
		UncommittedBlobs: false,
		Deleted: false,
	}

	// List blobs
	var blobs []BlobInfo
	marker := azblob.Marker{}

	for marker.NotDone() {
		listResponse, err := containerURL.ListBlobsFlatSegment(n.ctx, marker, listOptions)
		if err != nil {
			return fmt.Errorf("failed to list blobs: %w", err)
		}

		for _, blobInfo := range listResponse.Segment.BlobItems {
			blob := BlobInfo{
				Name:               blobInfo.Name,
				Container:          operation.Container,
				Size:               *blobInfo.Properties.ContentLength,
				ContentType:        *blobInfo.Properties.ContentType,
				LastModified:       blobInfo.Properties.LastModified,
				ETag:               string(blobInfo.Properties.Etag),
				BlobType:           string(blobInfo.Properties.BlobType),
				CreationTime:       *blobInfo.Properties.CreationTime,
				ServerEncrypted:    *blobInfo.Properties.ServerEncrypted,
				AccessTierInferred: *blobInfo.Properties.AccessTierInferred,
			}

			// Add optional properties
			if blobInfo.Properties.ContentEncoding != nil {
				blob.ContentEncoding = *blobInfo.Properties.ContentEncoding
			}
			if blobInfo.Properties.ContentLanguage != nil {
				blob.ContentLanguage = *blobInfo.Properties.ContentLanguage
			}
			if blobInfo.Properties.ContentDisposition != nil {
				blob.ContentDisposition = *blobInfo.Properties.ContentDisposition
			}
			if blobInfo.Properties.CacheControl != nil {
				blob.CacheControl = *blobInfo.Properties.CacheControl
			}
			if blobInfo.Properties.AccessTier != "" {
				blob.AccessTier = string(blobInfo.Properties.AccessTier)
			}
			if blobInfo.Properties.ArchiveStatus != "" {
				blob.ArchiveStatus = string(blobInfo.Properties.ArchiveStatus)
			}
			if blobInfo.Properties.LeaseStatus != "" {
				blob.LeaseStatus = string(blobInfo.Properties.LeaseStatus)
			}
			if blobInfo.Properties.LeaseState != "" {
				blob.LeaseState = string(blobInfo.Properties.LeaseState)
			}
			if blobInfo.Properties.LeaseDuration != "" {
				blob.LeaseDuration = string(blobInfo.Properties.LeaseDuration)
			}

			// Add metadata
			if len(blobInfo.Metadata) > 0 {
				blob.Metadata = blobInfo.Metadata
			}

			blobs = append(blobs, blob)
		}

		marker = listResponse.NextMarker
	}

	result.Blobs = blobs
	return nil
}

func (n *BlobStorageNode) copyBlob(operation *BlobOperation, context *expressions.ExpressionContext, item model.DataItem, result *BlobOperationResult) error {
	if operation.CopyOptions == nil {
		return fmt.Errorf("copy options are required for copy operation")
	}

	// Evaluate destination blob name
	destBlobName, err := n.evaluateExpression(operation.Blob, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate destination blob name: %w", err)
	}
	result.Blob = fmt.Sprintf("%v", destBlobName)

	// Get destination container URL
	containerURL, err := n.getContainerURL(operation.Container)
	if err != nil {
		return fmt.Errorf("failed to get container URL: %w", err)
	}

	// Get destination blob URL
	destBlobURL := containerURL.NewBlobURL(result.Blob)

	// Parse source URL
	sourceURL, err := url.Parse(operation.CopyOptions.SourceURL)
	if err != nil {
		return fmt.Errorf("failed to parse source URL: %w", err)
	}

	// Start copy operation
	copyResponse, err := destBlobURL.StartCopyFromURL(n.ctx, *sourceURL, operation.CopyOptions.Metadata, azblob.ModifiedAccessConditions{}, azblob.BlobAccessConditions{}, azblob.DefaultAccessTier, nil)
	if err != nil {
		return fmt.Errorf("failed to start copy operation: %w", err)
	}

	result.ETag = string(copyResponse.ETag())

	// Wait for copy to complete (polling)
	for {
		props, err := destBlobURL.GetProperties(n.ctx, azblob.BlobAccessConditions{}, azblob.ClientProvidedKeyOptions{})
		if err != nil {
			return fmt.Errorf("failed to get blob properties during copy: %w", err)
		}

		copyStatus := props.CopyStatus()
		if copyStatus == azblob.CopyStatusSuccess {
			result.Size = props.ContentLength()
			result.ContentType = props.ContentType()
			break
		} else if copyStatus == azblob.CopyStatusFailed || copyStatus == azblob.CopyStatusAborted {
			return fmt.Errorf("copy operation failed with status: %s", copyStatus)
		}

		// Wait before polling again
		time.Sleep(1 * time.Second)
	}

	return nil
}

func (n *BlobStorageNode) getBlobProperties(operation *BlobOperation, context *expressions.ExpressionContext, item model.DataItem, result *BlobOperationResult) error {
	// Evaluate blob name
	blobName, err := n.evaluateExpression(operation.Blob, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate blob name: %w", err)
	}
	result.Blob = fmt.Sprintf("%v", blobName)

	// Get container URL
	containerURL, err := n.getContainerURL(operation.Container)
	if err != nil {
		return fmt.Errorf("failed to get container URL: %w", err)
	}

	// Get blob URL
	blobURL := containerURL.NewBlobURL(result.Blob)

	// Get blob properties
	props, err := blobURL.GetProperties(n.ctx, azblob.BlobAccessConditions{}, azblob.ClientProvidedKeyOptions{})
	if err != nil {
		return fmt.Errorf("failed to get blob properties: %w", err)
	}

	// Convert to our format
	blobInfo := BlobInfo{
		Name:            result.Blob,
		Container:       operation.Container,
		Size:            props.ContentLength(),
		ContentType:     props.ContentType(),
		LastModified:    props.LastModified(),
		ETag:            string(props.ETag()),
		ContentEncoding: props.ContentEncoding(),
		ContentLanguage: props.ContentLanguage(),
		CacheControl:    props.CacheControl(),
		BlobType:        string(props.BlobType()),
		CreationTime:    props.CreationTime(),
		ServerEncrypted: props.IsServerEncrypted() == "true",
	}

	result.Data = blobInfo
	return nil
}

func (n *BlobStorageNode) setBlobProperties(operation *BlobOperation, context *expressions.ExpressionContext, item model.DataItem, result *BlobOperationResult) error {
	// Evaluate blob name
	blobName, err := n.evaluateExpression(operation.Blob, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate blob name: %w", err)
	}
	result.Blob = fmt.Sprintf("%v", blobName)

	if operation.Properties == nil {
		return fmt.Errorf("properties are required for set_properties operation")
	}

	// Get container URL
	containerURL, err := n.getContainerURL(operation.Container)
	if err != nil {
		return fmt.Errorf("failed to get container URL: %w", err)
	}

	// Get blob URL
	blobURL := containerURL.NewBlobURL(result.Blob)

	// Set properties
	headers := azblob.BlobHTTPHeaders{
		ContentType:        operation.Properties.ContentType,
		ContentEncoding:    operation.Properties.ContentEncoding,
		ContentLanguage:    operation.Properties.ContentLanguage,
		ContentDisposition: operation.Properties.ContentDisposition,
		CacheControl:       operation.Properties.CacheControl,
		ContentMD5:         operation.Properties.ContentMD5,
	}

	_, err = blobURL.SetHTTPHeaders(n.ctx, headers, azblob.BlobAccessConditions{})
	if err != nil {
		return fmt.Errorf("failed to set blob properties: %w", err)
	}

	return nil
}

func (n *BlobStorageNode) getBlobMetadata(operation *BlobOperation, context *expressions.ExpressionContext, item model.DataItem, result *BlobOperationResult) error {
	// Evaluate blob name
	blobName, err := n.evaluateExpression(operation.Blob, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate blob name: %w", err)
	}
	result.Blob = fmt.Sprintf("%v", blobName)

	// Get container URL
	containerURL, err := n.getContainerURL(operation.Container)
	if err != nil {
		return fmt.Errorf("failed to get container URL: %w", err)
	}

	// Get blob URL
	blobURL := containerURL.NewBlobURL(result.Blob)

	// Get blob properties (which includes metadata)
	props, err := blobURL.GetProperties(n.ctx, azblob.BlobAccessConditions{}, azblob.ClientProvidedKeyOptions{})
	if err != nil {
		return fmt.Errorf("failed to get blob metadata: %w", err)
	}

	result.Data = props.NewMetadata()
	return nil
}

func (n *BlobStorageNode) setBlobMetadata(operation *BlobOperation, context *expressions.ExpressionContext, item model.DataItem, result *BlobOperationResult) error {
	// Evaluate blob name
	blobName, err := n.evaluateExpression(operation.Blob, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate blob name: %w", err)
	}
	result.Blob = fmt.Sprintf("%v", blobName)

	if len(operation.Metadata) == 0 {
		return fmt.Errorf("metadata is required for set_metadata operation")
	}

	// Get container URL
	containerURL, err := n.getContainerURL(operation.Container)
	if err != nil {
		return fmt.Errorf("failed to get container URL: %w", err)
	}

	// Get blob URL
	blobURL := containerURL.NewBlobURL(result.Blob)

	// Set metadata
	_, err = blobURL.SetMetadata(n.ctx, operation.Metadata, azblob.BlobAccessConditions{}, azblob.ClientProvidedKeyOptions{})
	if err != nil {
		return fmt.Errorf("failed to set blob metadata: %w", err)
	}

	return nil
}

// Container operations
func (n *BlobStorageNode) createContainer(operation *BlobOperation, context *expressions.ExpressionContext, item model.DataItem, result *BlobOperationResult) error {
	// Get container URL
	containerURL, err := n.getContainerURL(operation.Container)
	if err != nil {
		return fmt.Errorf("failed to get container URL: %w", err)
	}

	// Set public access type
	publicAccessType := azblob.PublicAccessNone
	if accessType, ok := operation.Options["publicAccess"].(string); ok {
		switch strings.ToLower(accessType) {
		case "blob":
			publicAccessType = azblob.PublicAccessBlob
		case "container":
			publicAccessType = azblob.PublicAccessContainer
		}
	}

	// Create container
	_, err = containerURL.Create(n.ctx, operation.Metadata, publicAccessType)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	return nil
}

func (n *BlobStorageNode) deleteContainer(operation *BlobOperation, context *expressions.ExpressionContext, item model.DataItem, result *BlobOperationResult) error {
	// Get container URL
	containerURL, err := n.getContainerURL(operation.Container)
	if err != nil {
		return fmt.Errorf("failed to get container URL: %w", err)
	}

	// Delete container
	_, err = containerURL.Delete(n.ctx, azblob.ContainerAccessConditions{})
	if err != nil {
		return fmt.Errorf("failed to delete container: %w", err)
	}

	return nil
}

func (n *BlobStorageNode) listContainers(operation *BlobOperation, context *expressions.ExpressionContext, item model.DataItem, result *BlobOperationResult) error {
	// Prepare list options
	listOptions := azblob.ListContainersSegmentOptions{}

	// Handle pagination
	if maxResults, ok := operation.Options["maxResults"].(int); ok {
		listOptions.MaxResults = int32(maxResults)
	}

	// List containers
	var containers []ContainerInfo
	marker := azblob.Marker{}

	for marker.NotDone() {
		listResponse, err := n.serviceURL.ListContainersSegment(n.ctx, marker, listOptions)
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}

		for _, containerInfo := range listResponse.ContainerItems {
			container := ContainerInfo{
				Name:         containerInfo.Name,
				LastModified: containerInfo.Properties.LastModified,
				ETag:         string(containerInfo.Properties.Etag),
				LeaseStatus:  string(containerInfo.Properties.LeaseStatus),
				LeaseState:   string(containerInfo.Properties.LeaseState),
				PublicAccess: string(containerInfo.Properties.PublicAccess),
			}

			// Add optional properties (handle pointer fields)
			if containerInfo.Properties.HasImmutabilityPolicy != nil {
				container.HasImmutabilityPolicy = *containerInfo.Properties.HasImmutabilityPolicy
			}
			if containerInfo.Properties.HasLegalHold != nil {
				container.HasLegalHold = *containerInfo.Properties.HasLegalHold
			}
			if containerInfo.Properties.PreventEncryptionScopeOverride != nil {
				container.DenyEncryptionScopeOverride = *containerInfo.Properties.PreventEncryptionScopeOverride
			}
			if containerInfo.Properties.LeaseDuration != "" {
				container.LeaseDuration = string(containerInfo.Properties.LeaseDuration)
			}
			if containerInfo.Properties.DefaultEncryptionScope != nil {
				container.DefaultEncryptionScope = *containerInfo.Properties.DefaultEncryptionScope
			}

			// Add metadata
			if len(containerInfo.Metadata) > 0 {
				container.Metadata = containerInfo.Metadata
			}

			containers = append(containers, container)
		}

		marker = listResponse.NextMarker
	}

	result.Containers = containers
	return nil
}

// Lease operations
func (n *BlobStorageNode) acquireLease(operation *BlobOperation, context *expressions.ExpressionContext, item model.DataItem, result *BlobOperationResult) error {
	if operation.LeaseOptions == nil {
		return fmt.Errorf("lease options are required for acquire_lease operation")
	}

	// Evaluate blob name
	blobName, err := n.evaluateExpression(operation.Blob, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate blob name: %w", err)
	}
	result.Blob = fmt.Sprintf("%v", blobName)

	// Get container URL
	containerURL, err := n.getContainerURL(operation.Container)
	if err != nil {
		return fmt.Errorf("failed to get container URL: %w", err)
	}

	// Get blob URL
	blobURL := containerURL.NewBlobURL(result.Blob)

	// Acquire lease
	leaseResponse, err := blobURL.AcquireLease(n.ctx, "", operation.LeaseOptions.Duration, azblob.ModifiedAccessConditions{})
	if err != nil {
		return fmt.Errorf("failed to acquire lease: %w", err)
	}

	result.LeaseID = leaseResponse.LeaseID()
	return nil
}

func (n *BlobStorageNode) renewLease(operation *BlobOperation, context *expressions.ExpressionContext, item model.DataItem, result *BlobOperationResult) error {
	if operation.LeaseOptions == nil || operation.LeaseOptions.LeaseID == "" {
		return fmt.Errorf("lease ID is required for renew_lease operation")
	}

	// Evaluate blob name
	blobName, err := n.evaluateExpression(operation.Blob, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate blob name: %w", err)
	}
	result.Blob = fmt.Sprintf("%v", blobName)

	// Get container URL
	containerURL, err := n.getContainerURL(operation.Container)
	if err != nil {
		return fmt.Errorf("failed to get container URL: %w", err)
	}

	// Get blob URL
	blobURL := containerURL.NewBlobURL(result.Blob)

	// Renew lease
	_, err = blobURL.RenewLease(n.ctx, operation.LeaseOptions.LeaseID, azblob.ModifiedAccessConditions{})
	if err != nil {
		return fmt.Errorf("failed to renew lease: %w", err)
	}

	result.LeaseID = operation.LeaseOptions.LeaseID
	return nil
}

func (n *BlobStorageNode) releaseLease(operation *BlobOperation, context *expressions.ExpressionContext, item model.DataItem, result *BlobOperationResult) error {
	if operation.LeaseOptions == nil || operation.LeaseOptions.LeaseID == "" {
		return fmt.Errorf("lease ID is required for release_lease operation")
	}

	// Evaluate blob name
	blobName, err := n.evaluateExpression(operation.Blob, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate blob name: %w", err)
	}
	result.Blob = fmt.Sprintf("%v", blobName)

	// Get container URL
	containerURL, err := n.getContainerURL(operation.Container)
	if err != nil {
		return fmt.Errorf("failed to get container URL: %w", err)
	}

	// Get blob URL
	blobURL := containerURL.NewBlobURL(result.Blob)

	// Release lease
	_, err = blobURL.ReleaseLease(n.ctx, operation.LeaseOptions.LeaseID, azblob.ModifiedAccessConditions{})
	if err != nil {
		return fmt.Errorf("failed to release lease: %w", err)
	}

	return nil
}

func (n *BlobStorageNode) breakLease(operation *BlobOperation, context *expressions.ExpressionContext, item model.DataItem, result *BlobOperationResult) error {
	// Evaluate blob name
	blobName, err := n.evaluateExpression(operation.Blob, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate blob name: %w", err)
	}
	result.Blob = fmt.Sprintf("%v", blobName)

	// Get container URL
	containerURL, err := n.getContainerURL(operation.Container)
	if err != nil {
		return fmt.Errorf("failed to get container URL: %w", err)
	}

	// Get blob URL
	blobURL := containerURL.NewBlobURL(result.Blob)

	// Break lease
	breakPeriod := int32(0) // Immediate break
	if operation.LeaseOptions != nil && operation.LeaseOptions.Duration > 0 {
		breakPeriod = operation.LeaseOptions.Duration
	}

	_, err = blobURL.BreakLease(n.ctx, breakPeriod, azblob.ModifiedAccessConditions{})
	if err != nil {
		return fmt.Errorf("failed to break lease: %w", err)
	}

	return nil
}

func (n *BlobStorageNode) generateSASURL(operation *BlobOperation, context *expressions.ExpressionContext, item model.DataItem, result *BlobOperationResult) error {
	// Evaluate blob name
	blobName, err := n.evaluateExpression(operation.Blob, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate blob name: %w", err)
	}
	result.Blob = fmt.Sprintf("%v", blobName)

	// Get expiration time
	expiration := 15 * time.Minute // Default 15 minutes
	if exp, ok := operation.Options["expiration"].(int); ok {
		expiration = time.Duration(exp) * time.Minute
	}

	// Get permissions
	permissions := "r" // Default read permission
	if perms, ok := operation.Options["permissions"].(string); ok {
		permissions = perms
	}

	// Get container URL
	containerURL, err := n.getContainerURL(operation.Container)
	if err != nil {
		return fmt.Errorf("failed to get container URL: %w", err)
	}

	// Get blob URL
	blobURL := containerURL.NewBlobURL(result.Blob)

	// Check if credential supports SAS generation
	storageAccountCred, ok := n.credential.(azblob.StorageAccountCredential)
	if !ok {
		return fmt.Errorf("SAS URL generation requires a storage account credential with account key")
	}

	// Generate SAS token
	sasQueryParams, err := azblob.BlobSASSignatureValues{
		Protocol:      azblob.SASProtocolHTTPS,
		ExpiryTime:    time.Now().Add(expiration),
		ContainerName: operation.Container,
		BlobName:      result.Blob,
		Permissions:   permissions,
	}.NewSASQueryParameters(storageAccountCred)

	if err != nil {
		return fmt.Errorf("failed to generate SAS token: %w", err)
	}

	// Build URL with SAS token
	blobURLValue := blobURL.URL()
	urlParts := azblob.NewBlobURLParts(blobURLValue)
	urlParts.SAS = sasQueryParams
	sasURL := urlParts.URL()
	result.URL = sasURL.String()

	return nil
}

// Helper methods
func (n *BlobStorageNode) getContainerURL(containerName string) (*azblob.ContainerURL, error) {
	if containerURL, exists := n.containerURLs[containerName]; exists {
		return containerURL, nil
	}

	containerURL := n.serviceURL.NewContainerURL(containerName)
	n.containerURLs[containerName] = &containerURL
	return &containerURL, nil
}

func (n *BlobStorageNode) detectContentType(blobName string) string {
	// Basic content type detection based on file extension
	if strings.HasSuffix(blobName, ".json") {
		return "application/json"
	} else if strings.HasSuffix(blobName, ".xml") {
		return "application/xml"
	} else if strings.HasSuffix(blobName, ".html") {
		return "text/html"
	} else if strings.HasSuffix(blobName, ".css") {
		return "text/css"
	} else if strings.HasSuffix(blobName, ".js") {
		return "application/javascript"
	} else if strings.HasSuffix(blobName, ".jpg") || strings.HasSuffix(blobName, ".jpeg") {
		return "image/jpeg"
	} else if strings.HasSuffix(blobName, ".png") {
		return "image/png"
	} else if strings.HasSuffix(blobName, ".pdf") {
		return "application/pdf"
	} else if strings.HasSuffix(blobName, ".txt") {
		return "text/plain"
	}

	return "application/octet-stream"
}

// Configuration parsing methods
func (n *BlobStorageNode) parseAzureConfig(nodeParams map[string]interface{}) (*AzureConfig, error) {
	config := &AzureConfig{}

	if accountName, ok := nodeParams["accountName"].(string); ok {
		config.AccountName = accountName
	} else {
		return nil, fmt.Errorf("accountName is required")
	}

	if accountKey, ok := nodeParams["accountKey"].(string); ok {
		config.AccountKey = accountKey
	}

	if sasToken, ok := nodeParams["sasToken"].(string); ok {
		config.SASToken = sasToken
	}

	if connectionString, ok := nodeParams["connectionString"].(string); ok {
		config.ConnectionString = connectionString
	}

	if tenantID, ok := nodeParams["tenantId"].(string); ok {
		config.TenantID = tenantID
	}

	if clientID, ok := nodeParams["clientId"].(string); ok {
		config.ClientID = clientID
	}

	if clientSecret, ok := nodeParams["clientSecret"].(string); ok {
		config.ClientSecret = clientSecret
	}

	if useManagedIdentity, ok := nodeParams["useManagedIdentity"].(bool); ok {
		config.UseManagedIdentity = useManagedIdentity
	}

	return config, nil
}

func (n *BlobStorageNode) parseBlobOperation(nodeParams map[string]interface{}) (*BlobOperation, error) {
	operation := &BlobOperation{
		Options: make(map[string]interface{}),
	}

	if op, ok := nodeParams["operation"].(string); ok {
		operation.Operation = op
	} else {
		return nil, fmt.Errorf("operation parameter is required")
	}

	if container, ok := nodeParams["container"].(string); ok {
		operation.Container = container
	} else {
		return nil, fmt.Errorf("container parameter is required")
	}

	if blob, ok := nodeParams["blob"].(string); ok {
		operation.Blob = blob
	}

	if source, ok := nodeParams["source"].(string); ok {
		operation.Source = source
	}

	if destination, ok := nodeParams["destination"].(string); ok {
		operation.Destination = destination
	}

	if prefix, ok := nodeParams["prefix"].(string); ok {
		operation.Prefix = prefix
	}

	if contentType, ok := nodeParams["contentType"].(string); ok {
		operation.ContentType = contentType
	}

	if accessTier, ok := nodeParams["accessTier"].(string); ok {
		operation.AccessTier = accessTier
	}

	// Parse metadata
	if metadata, ok := nodeParams["metadata"].(map[string]interface{}); ok {
		operation.Metadata = make(map[string]string)
		for k, v := range metadata {
			operation.Metadata[k] = fmt.Sprintf("%v", v)
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

func (n *BlobStorageNode) initializeBlobClient(config *AzureConfig) error {
	var serviceURL azblob.ServiceURL

	if config.ConnectionString != "" {
		// Parse connection string manually
		connStr := config.ConnectionString
		accountName, accountKey, endpoint := parseConnectionString(connStr)
		if accountName != "" && accountKey != "" {
			credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
			if err != nil {
				return fmt.Errorf("failed to create shared key credential from connection string: %w", err)
			}
			n.credential = credential

			if endpoint == "" {
				endpoint = fmt.Sprintf("https://%s.blob.core.windows.net", accountName)
			}

			u, err := url.Parse(endpoint)
			if err != nil {
				return fmt.Errorf("failed to parse endpoint URL: %w", err)
			}

			p := azblob.NewPipeline(credential, azblob.PipelineOptions{})
			serviceURL = azblob.NewServiceURL(*u, p)
		} else {
			return fmt.Errorf("invalid connection string: missing account name or key")
		}
	} else if config.AccountKey != "" {
		// Use account key
		credential, err := azblob.NewSharedKeyCredential(config.AccountName, config.AccountKey)
		if err != nil {
			return fmt.Errorf("failed to create shared key credential: %w", err)
		}
		n.credential = credential

		// Create service URL
		serviceURL = azblob.NewServiceURL(
			url.URL{Scheme: "https", Host: fmt.Sprintf("%s.blob.core.windows.net", config.AccountName)},
			azblob.NewPipeline(credential, azblob.PipelineOptions{}),
		)
	} else if config.SASToken != "" {
		// Use SAS token
		credential := azblob.NewAnonymousCredential()
		n.credential = credential

		// Create service URL with SAS token
		u, err := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/?%s", config.AccountName, config.SASToken))
		if err != nil {
			return fmt.Errorf("failed to parse SAS URL: %w", err)
		}

		serviceURL = azblob.NewServiceURL(*u, azblob.NewPipeline(credential, azblob.PipelineOptions{}))
	} else {
		return fmt.Errorf("authentication method required: provide accountKey, sasToken, or connectionString")
	}

	n.serviceURL = &serviceURL
	return nil
}

func (n *BlobStorageNode) evaluateExpression(expr string, context *expressions.ExpressionContext) (interface{}, error) {
	return n.evaluator.EvaluateExpression(expr, context)
}

// parseConnectionString parses an Azure Storage connection string and extracts AccountName, AccountKey, and BlobEndpoint
func parseConnectionString(connStr string) (accountName, accountKey, endpoint string) {
	parts := strings.Split(connStr, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		idx := strings.Index(part, "=")
		if idx == -1 {
			continue
		}

		key := strings.TrimSpace(part[:idx])
		value := strings.TrimSpace(part[idx+1:])

		switch strings.ToLower(key) {
		case "accountname":
			accountName = value
		case "accountkey":
			accountKey = value
		case "blobendpoint":
			endpoint = value
		}
	}
	return
}

// ValidateParameters validates the node parameters
func (n *BlobStorageNode) ValidateParameters(params map[string]interface{}) error {
	// Validate required parameters
	if _, ok := params["operation"]; !ok {
		return fmt.Errorf("operation parameter is required")
	}

	if _, ok := params["container"]; !ok {
		return fmt.Errorf("container parameter is required")
	}

	// Validate operation
	if operation, ok := params["operation"].(string); ok {
		validOperations := []string{
			"upload", "download", "delete", "list", "copy",
			"get_properties", "set_properties", "get_metadata", "set_metadata",
			"create_container", "delete_container", "list_containers",
			"acquire_lease", "renew_lease", "release_lease", "break_lease",
			"generate_sas_url",
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