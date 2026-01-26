package gcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/neul-labs/m9m/internal/expressions"
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// CloudStorageNode provides Google Cloud Storage operations
type CloudStorageNode struct {
	*base.BaseNode
	evaluator     *expressions.GojaExpressionEvaluator
	storageClient *storage.Client
	ctx           context.Context
}

// GCPConfig holds GCP authentication configuration
type GCPConfig struct {
	ProjectID           string `json:"projectId"`
	KeyFile             string `json:"keyFile,omitempty"`
	KeyFileContent      string `json:"keyFileContent,omitempty"`
	ServiceAccountEmail string `json:"serviceAccountEmail,omitempty"`
	UseADC              bool   `json:"useADC"` // Use Application Default Credentials
}

// StorageOperation represents a Cloud Storage operation
type StorageOperation struct {
	Operation     string                 `json:"operation"`
	Bucket        string                 `json:"bucket"`
	Object        string                 `json:"object,omitempty"`
	Source        string                 `json:"source,omitempty"`
	Destination   string                 `json:"destination,omitempty"`
	Prefix        string                 `json:"prefix,omitempty"`
	Delimiter     string                 `json:"delimiter,omitempty"`
	ContentType   string                 `json:"contentType,omitempty"`
	Metadata      map[string]string      `json:"metadata,omitempty"`
	ACL           *StorageACL            `json:"acl,omitempty"`
	Lifecycle     *StorageLifecycle      `json:"lifecycle,omitempty"`
	Versioning    bool                   `json:"versioning"`
	Encryption    *StorageEncryption     `json:"encryption,omitempty"`
	StorageClass  string                 `json:"storageClass,omitempty"`
	Options       map[string]interface{} `json:"options,omitempty"`
}

// StorageACL defines access control settings
type StorageACL struct {
	DefaultObjectACL string                `json:"defaultObjectACL,omitempty"`
	BucketACL        string                `json:"bucketACL,omitempty"`
	Permissions      []StoragePermission   `json:"permissions,omitempty"`
}

// StoragePermission defines individual permissions
type StoragePermission struct {
	Entity string `json:"entity"` // user-email, group-email, domain-domain, project-team-projectId, etc.
	Role   string `json:"role"`   // READER, WRITER, OWNER
}

// StorageLifecycle defines lifecycle management rules
type StorageLifecycle struct {
	Rules []StorageLifecycleRule `json:"rules"`
}

// StorageLifecycleRule defines a lifecycle rule
type StorageLifecycleRule struct {
	Action    StorageLifecycleAction    `json:"action"`
	Condition StorageLifecycleCondition `json:"condition"`
}

// StorageLifecycleAction defines lifecycle action
type StorageLifecycleAction struct {
	Type         string `json:"type"`         // Delete, SetStorageClass
	StorageClass string `json:"storageClass,omitempty"`
}

// StorageLifecycleCondition defines lifecycle condition
type StorageLifecycleCondition struct {
	Age                   int      `json:"age,omitempty"`
	CreatedBefore         string   `json:"createdBefore,omitempty"`
	IsLive                *bool    `json:"isLive,omitempty"`
	MatchesStorageClasses []string `json:"matchesStorageClasses,omitempty"`
	MatchesPrefix         []string `json:"matchesPrefix,omitempty"`
	MatchesSuffix         []string `json:"matchesSuffix,omitempty"`
	NumNewerVersions      int      `json:"numNewerVersions,omitempty"`
}

// StorageEncryption defines encryption settings
type StorageEncryption struct {
	KMSKeyName          string `json:"kmsKeyName,omitempty"`
	CustomerSuppliedKey string `json:"customerSuppliedKey,omitempty"`
}

// StorageObject represents a Cloud Storage object
type StorageObject struct {
	Name            string            `json:"name"`
	Bucket          string            `json:"bucket"`
	Size            int64             `json:"size"`
	ContentType     string            `json:"contentType"`
	TimeCreated     time.Time         `json:"timeCreated"`
	Updated         time.Time         `json:"updated"`
	Generation      int64             `json:"generation"`
	Metageneration  int64             `json:"metageneration"`
	StorageClass    string            `json:"storageClass"`
	MD5             string            `json:"md5,omitempty"`
	CRC32C          string            `json:"crc32c,omitempty"`
	ETag            string            `json:"etag"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	ACL             []StorageACLEntry `json:"acl,omitempty"`
	Owner           string            `json:"owner,omitempty"`
	RetentionExpires *time.Time       `json:"retentionExpires,omitempty"`
}

// StorageACLEntry represents an ACL entry
type StorageACLEntry struct {
	Entity string `json:"entity"`
	Role   string `json:"role"`
	Email  string `json:"email,omitempty"`
}

// StorageBucket represents a Cloud Storage bucket
type StorageBucket struct {
	Name               string                     `json:"name"`
	Location           string                     `json:"location"`
	StorageClass       string                     `json:"storageClass"`
	TimeCreated        time.Time                  `json:"timeCreated"`
	Updated            time.Time                  `json:"updated"`
	Metageneration     int64                      `json:"metageneration"`
	Labels             map[string]string          `json:"labels,omitempty"`
	Lifecycle          *StorageLifecycle          `json:"lifecycle,omitempty"`
	Versioning         bool                       `json:"versioning"`
	RequesterPays      bool                       `json:"requesterPays"`
	RetentionPolicy    *StorageRetentionPolicy    `json:"retentionPolicy,omitempty"`
	Encryption         *StorageEncryption         `json:"encryption,omitempty"`
	CORS               []StorageCORSPolicy        `json:"cors,omitempty"`
	Website            *StorageWebsiteConfig      `json:"website,omitempty"`
	Logging            *StorageLoggingConfig      `json:"logging,omitempty"`
}

// StorageRetentionPolicy defines bucket retention policy
type StorageRetentionPolicy struct {
	RetentionPeriod time.Duration `json:"retentionPeriod"`
	EffectiveTime   time.Time     `json:"effectiveTime"`
	IsLocked        bool          `json:"isLocked"`
}

// StorageCORSPolicy defines CORS policy
type StorageCORSPolicy struct {
	Origins         []string `json:"origins"`
	Methods         []string `json:"methods"`
	Headers         []string `json:"headers"`
	ResponseHeaders []string `json:"responseHeaders"`
	MaxAge          int      `json:"maxAge"`
}

// StorageWebsiteConfig defines website configuration
type StorageWebsiteConfig struct {
	MainPageSuffix string `json:"mainPageSuffix"`
	NotFoundPage   string `json:"notFoundPage"`
}

// StorageLoggingConfig defines access logging configuration
type StorageLoggingConfig struct {
	LogBucket       string `json:"logBucket"`
	LogObjectPrefix string `json:"logObjectPrefix"`
}

// StorageOperationResult represents the result of a storage operation
type StorageOperationResult struct {
	Operation     string                 `json:"operation"`
	Success       bool                   `json:"success"`
	Bucket        string                 `json:"bucket"`
	Object        string                 `json:"object,omitempty"`
	Objects       []StorageObject        `json:"objects,omitempty"`
	Buckets       []StorageBucket        `json:"buckets,omitempty"`
	Data          interface{}            `json:"data,omitempty"`
	Size          int64                  `json:"size,omitempty"`
	ContentType   string                 `json:"contentType,omitempty"`
	Generation    int64                  `json:"generation,omitempty"`
	URL           string                 `json:"url,omitempty"`
	ExecutionTime time.Duration          `json:"executionTime"`
	Error         string                 `json:"error,omitempty"`
}

// NewCloudStorageNode creates a new Cloud Storage operations node
func NewCloudStorageNode() *CloudStorageNode {
	return &CloudStorageNode{
		BaseNode:  base.NewBaseNode(base.NodeDescription{Name: "Google Cloud Storage", Description: "Google Cloud Storage operations", Category: "cloud"}),
		evaluator: expressions.NewGojaExpressionEvaluator(expressions.DefaultEvaluatorConfig()),
		ctx:       context.Background(),
	}
}

// Execute performs Cloud Storage operations
func (n *CloudStorageNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	var results []model.DataItem

	// Parse GCP configuration
	config, err := n.parseGCPConfig(nodeParams)
	if err != nil {
		return nil, n.CreateError("invalid GCP configuration", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Initialize Cloud Storage client
	if err := n.initializeStorageClient(config); err != nil {
		return nil, n.CreateError("failed to initialize Cloud Storage client", map[string]interface{}{
			"error": err.Error(),
		})
	}
	defer n.storageClient.Close()

	// Parse operation configuration
	operation, err := n.parseStorageOperation(nodeParams)
	if err != nil {
		return nil, n.CreateError("invalid storage operation", map[string]interface{}{
			"error": err.Error(),
		})
	}

	for index, item := range inputData {
		// Create expression context
		exprContext := &expressions.ExpressionContext{
			ActiveNodeName:      "Google Cloud Storage",
			RunIndex:           0,
			ItemIndex:          index,
			Mode:               expressions.ModeManual,
			ConnectionInputData: []model.DataItem{item},
			AdditionalKeys:     &expressions.AdditionalKeys{},
		}

		// Execute storage operation
		result, err := n.executeStorageOperation(operation, exprContext, item)
		if err != nil {
			return nil, n.CreateError("Cloud Storage operation failed", map[string]interface{}{
				"operation": operation.Operation,
				"error":     err.Error(),
			})
		}

		// Create result item - convert StorageOperationResult to map
		resultJSON := map[string]interface{}{
			"operation":     result.Operation,
			"success":       result.Success,
			"bucket":        result.Bucket,
			"object":        result.Object,
			"size":          result.Size,
			"contentType":   result.ContentType,
			"generation":    result.Generation,
			"url":           result.URL,
			"executionTime": result.ExecutionTime.String(),
		}
		if len(result.Objects) > 0 {
			resultJSON["objects"] = result.Objects
		}
		if len(result.Buckets) > 0 {
			resultJSON["buckets"] = result.Buckets
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
						FileName: filepath.Base(result.Object),
					},
				}
			}
		}

		results = append(results, resultItem)
	}

	return results, nil
}

// executeStorageOperation executes the specified storage operation
func (n *CloudStorageNode) executeStorageOperation(operation *StorageOperation, context *expressions.ExpressionContext, item model.DataItem) (*StorageOperationResult, error) {
	startTime := time.Now()

	result := &StorageOperationResult{
		Operation: operation.Operation,
		Bucket:    operation.Bucket,
	}

	var err error

	switch strings.ToLower(operation.Operation) {
	case "upload":
		err = n.uploadObject(operation, context, item, result)
	case "download":
		err = n.downloadObject(operation, context, item, result)
	case "delete":
		err = n.deleteObject(operation, context, item, result)
	case "list":
		err = n.listObjects(operation, context, item, result)
	case "copy":
		err = n.copyObject(operation, context, item, result)
	case "move":
		err = n.moveObject(operation, context, item, result)
	case "get_metadata":
		err = n.getObjectMetadata(operation, context, item, result)
	case "set_metadata":
		err = n.setObjectMetadata(operation, context, item, result)
	case "create_bucket":
		err = n.createBucket(operation, context, item, result)
	case "delete_bucket":
		err = n.deleteBucket(operation, context, item, result)
	case "list_buckets":
		err = n.listBuckets(operation, context, item, result)
	case "get_bucket":
		err = n.getBucket(operation, context, item, result)
	case "set_bucket_lifecycle":
		err = n.setBucketLifecycle(operation, context, item, result)
	case "generate_signed_url":
		err = n.generateSignedURL(operation, context, item, result)
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

// Object operations
func (n *CloudStorageNode) uploadObject(operation *StorageOperation, context *expressions.ExpressionContext, item model.DataItem, result *StorageOperationResult) error {
	// Evaluate object name
	objectName, err := n.evaluateExpression(operation.Object, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate object name: %w", err)
	}
	result.Object = fmt.Sprintf("%v", objectName)

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
		contentType = n.detectContentType(result.Object)
	}

	// Get bucket and object handles
	bucket := n.storageClient.Bucket(operation.Bucket)
	obj := bucket.Object(result.Object)

	// Create writer
	writer := obj.NewWriter(n.ctx)
	writer.ContentType = contentType

	// Set metadata
	if len(operation.Metadata) > 0 {
		writer.Metadata = operation.Metadata
	}

	// Set storage class
	if operation.StorageClass != "" {
		writer.StorageClass = operation.StorageClass
	}

	// Set encryption
	if operation.Encryption != nil {
		if operation.Encryption.KMSKeyName != "" {
			writer.KMSKeyName = operation.Encryption.KMSKeyName
		}
	}

	// Write data
	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return fmt.Errorf("failed to write data: %w", err)
	}

	// Close writer to complete upload
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to complete upload: %w", err)
	}

	// Get object attributes after upload
	attrs, err := obj.Attrs(n.ctx)
	if err == nil {
		result.Size = attrs.Size
		result.ContentType = attrs.ContentType
		result.Generation = attrs.Generation
	}

	return nil
}

func (n *CloudStorageNode) downloadObject(operation *StorageOperation, context *expressions.ExpressionContext, item model.DataItem, result *StorageOperationResult) error {
	// Evaluate object name
	objectName, err := n.evaluateExpression(operation.Object, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate object name: %w", err)
	}
	result.Object = fmt.Sprintf("%v", objectName)

	// Get bucket and object handles
	bucket := n.storageClient.Bucket(operation.Bucket)
	obj := bucket.Object(result.Object)

	// Create reader
	reader, err := obj.NewReader(n.ctx)
	if err != nil {
		return fmt.Errorf("failed to create reader: %w", err)
	}
	defer reader.Close()

	// Read all data
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	result.Data = data
	result.Size = int64(len(data))
	result.ContentType = reader.Attrs.ContentType
	result.Generation = reader.Attrs.Generation

	return nil
}

func (n *CloudStorageNode) deleteObject(operation *StorageOperation, context *expressions.ExpressionContext, item model.DataItem, result *StorageOperationResult) error {
	// Evaluate object name
	objectName, err := n.evaluateExpression(operation.Object, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate object name: %w", err)
	}
	result.Object = fmt.Sprintf("%v", objectName)

	// Get bucket and object handles
	bucket := n.storageClient.Bucket(operation.Bucket)
	obj := bucket.Object(result.Object)

	// Delete object
	if err := obj.Delete(n.ctx); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

func (n *CloudStorageNode) listObjects(operation *StorageOperation, context *expressions.ExpressionContext, item model.DataItem, result *StorageOperationResult) error {
	// Get bucket handle
	bucket := n.storageClient.Bucket(operation.Bucket)

	// Create query
	query := &storage.Query{}

	if operation.Prefix != "" {
		prefix, err := n.evaluateExpression(operation.Prefix, context)
		if err != nil {
			return fmt.Errorf("failed to evaluate prefix: %w", err)
		}
		query.Prefix = fmt.Sprintf("%v", prefix)
	}

	if operation.Delimiter != "" {
		query.Delimiter = operation.Delimiter
	}

	// Handle pagination
	if maxResults, ok := operation.Options["maxResults"].(int); ok {
		// Note: Cloud Storage client doesn't have a direct MaxResults field
		// We'll limit results manually after fetching
		_ = maxResults
	}

	// List objects
	it := bucket.Objects(n.ctx, query)
	var objects []StorageObject

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to iterate objects: %w", err)
		}

		obj := StorageObject{
			Name:           attrs.Name,
			Bucket:         attrs.Bucket,
			Size:           attrs.Size,
			ContentType:    attrs.ContentType,
			TimeCreated:    attrs.Created,
			Updated:        attrs.Updated,
			Generation:     attrs.Generation,
			Metageneration: attrs.Metageneration,
			StorageClass:   attrs.StorageClass,
			ETag:           attrs.Etag,
		}

		// Add metadata if present
		if len(attrs.Metadata) > 0 {
			obj.Metadata = attrs.Metadata
		}

		// Add checksums
		if len(attrs.MD5) > 0 {
			obj.MD5 = string(attrs.MD5)
		}
		if attrs.CRC32C != 0 {
			obj.CRC32C = fmt.Sprintf("%d", attrs.CRC32C)
		}

		objects = append(objects, obj)
	}

	result.Objects = objects
	return nil
}

func (n *CloudStorageNode) copyObject(operation *StorageOperation, context *expressions.ExpressionContext, item model.DataItem, result *StorageOperationResult) error {
	// Source and destination parsing
	sourceBucket, sourceObject, err := n.parseSourceDestination(operation.Source, context)
	if err != nil {
		return fmt.Errorf("failed to parse source: %w", err)
	}

	destBucket, destObject, err := n.parseSourceDestination(operation.Destination, context)
	if err != nil {
		return fmt.Errorf("failed to parse destination: %w", err)
	}

	result.Object = destObject

	// Get source and destination handles
	srcBucket := n.storageClient.Bucket(sourceBucket)
	srcObj := srcBucket.Object(sourceObject)

	dstBucket := n.storageClient.Bucket(destBucket)
	dstObj := dstBucket.Object(destObject)

	// Copy object
	copier := dstObj.CopierFrom(srcObj)

	// Set metadata if provided
	if len(operation.Metadata) > 0 {
		copier.Metadata = operation.Metadata
	}

	// Set storage class if provided
	if operation.StorageClass != "" {
		copier.StorageClass = operation.StorageClass
	}

	// Perform copy
	attrs, err := copier.Run(n.ctx)
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	result.Size = attrs.Size
	result.ContentType = attrs.ContentType
	result.Generation = attrs.Generation

	return nil
}

func (n *CloudStorageNode) moveObject(operation *StorageOperation, context *expressions.ExpressionContext, item model.DataItem, result *StorageOperationResult) error {
	// Copy the object first
	if err := n.copyObject(operation, context, item, result); err != nil {
		return err
	}

	// Then delete the source
	sourceBucket, sourceObject, err := n.parseSourceDestination(operation.Source, context)
	if err != nil {
		return fmt.Errorf("failed to parse source for deletion: %w", err)
	}

	srcBucket := n.storageClient.Bucket(sourceBucket)
	srcObj := srcBucket.Object(sourceObject)

	if err := srcObj.Delete(n.ctx); err != nil {
		return fmt.Errorf("failed to delete source object after copy: %w", err)
	}

	return nil
}

func (n *CloudStorageNode) getObjectMetadata(operation *StorageOperation, context *expressions.ExpressionContext, item model.DataItem, result *StorageOperationResult) error {
	// Evaluate object name
	objectName, err := n.evaluateExpression(operation.Object, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate object name: %w", err)
	}
	result.Object = fmt.Sprintf("%v", objectName)

	// Get bucket and object handles
	bucket := n.storageClient.Bucket(operation.Bucket)
	obj := bucket.Object(result.Object)

	// Get object attributes
	attrs, err := obj.Attrs(n.ctx)
	if err != nil {
		return fmt.Errorf("failed to get object metadata: %w", err)
	}

	// Convert to our object format
	storageObj := StorageObject{
		Name:           attrs.Name,
		Bucket:         attrs.Bucket,
		Size:           attrs.Size,
		ContentType:    attrs.ContentType,
		TimeCreated:    attrs.Created,
		Updated:        attrs.Updated,
		Generation:     attrs.Generation,
		Metageneration: attrs.Metageneration,
		StorageClass:   attrs.StorageClass,
		ETag:           attrs.Etag,
		Metadata:       attrs.Metadata,
	}

	result.Data = storageObj
	return nil
}

func (n *CloudStorageNode) setObjectMetadata(operation *StorageOperation, context *expressions.ExpressionContext, item model.DataItem, result *StorageOperationResult) error {
	// Evaluate object name
	objectName, err := n.evaluateExpression(operation.Object, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate object name: %w", err)
	}
	result.Object = fmt.Sprintf("%v", objectName)

	// Get bucket and object handles
	bucket := n.storageClient.Bucket(operation.Bucket)
	obj := bucket.Object(result.Object)

	// Prepare updates
	objAttrsToUpdate := storage.ObjectAttrsToUpdate{}

	// Set metadata
	if len(operation.Metadata) > 0 {
		objAttrsToUpdate.Metadata = operation.Metadata
	}

	// Set content type if provided
	if operation.ContentType != "" {
		objAttrsToUpdate.ContentType = operation.ContentType
	}

	// Update object attributes
	attrs, err := obj.Update(n.ctx, objAttrsToUpdate)
	if err != nil {
		return fmt.Errorf("failed to update object metadata: %w", err)
	}

	result.Generation = attrs.Generation
	return nil
}

// Bucket operations
func (n *CloudStorageNode) createBucket(operation *StorageOperation, context *expressions.ExpressionContext, item model.DataItem, result *StorageOperationResult) error {
	// Get project ID
	projectID, ok := operation.Options["projectId"].(string)
	if !ok {
		return fmt.Errorf("projectId is required for bucket creation")
	}

	// Get bucket handle
	bucket := n.storageClient.Bucket(operation.Bucket)

	// Prepare bucket attributes
	bucketAttrs := &storage.BucketAttrs{}

	// Set location
	if location, ok := operation.Options["location"].(string); ok {
		bucketAttrs.Location = location
	}

	// Set storage class
	if operation.StorageClass != "" {
		bucketAttrs.StorageClass = operation.StorageClass
	}

	// Set versioning
	if operation.Versioning {
		bucketAttrs.VersioningEnabled = true
	}

	// Set lifecycle
	if operation.Lifecycle != nil {
		bucketAttrs.Lifecycle = n.convertToGCSLifecycle(operation.Lifecycle)
	}

	// Create bucket
	if err := bucket.Create(n.ctx, projectID, bucketAttrs); err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	return nil
}

func (n *CloudStorageNode) deleteBucket(operation *StorageOperation, context *expressions.ExpressionContext, item model.DataItem, result *StorageOperationResult) error {
	// Get bucket handle
	bucket := n.storageClient.Bucket(operation.Bucket)

	// Delete bucket
	if err := bucket.Delete(n.ctx); err != nil {
		return fmt.Errorf("failed to delete bucket: %w", err)
	}

	return nil
}

func (n *CloudStorageNode) listBuckets(operation *StorageOperation, context *expressions.ExpressionContext, item model.DataItem, result *StorageOperationResult) error {
	// Get project ID
	projectID, ok := operation.Options["projectId"].(string)
	if !ok {
		return fmt.Errorf("projectId is required for listing buckets")
	}

	// List buckets
	it := n.storageClient.Buckets(n.ctx, projectID)
	var buckets []StorageBucket

	for {
		bucketAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to iterate buckets: %w", err)
		}

		bucket := StorageBucket{
			Name:           bucketAttrs.Name,
			Location:       bucketAttrs.Location,
			StorageClass:   bucketAttrs.StorageClass,
			TimeCreated:    bucketAttrs.Created,
			Updated:        bucketAttrs.Updated,
			Metageneration: bucketAttrs.MetaGeneration,
			Labels:         bucketAttrs.Labels,
			Versioning:     bucketAttrs.VersioningEnabled,
			RequesterPays:  bucketAttrs.RequesterPays,
		}

		buckets = append(buckets, bucket)
	}

	result.Buckets = buckets
	return nil
}

func (n *CloudStorageNode) getBucket(operation *StorageOperation, context *expressions.ExpressionContext, item model.DataItem, result *StorageOperationResult) error {
	// Get bucket handle
	bucket := n.storageClient.Bucket(operation.Bucket)

	// Get bucket attributes
	attrs, err := bucket.Attrs(n.ctx)
	if err != nil {
		return fmt.Errorf("failed to get bucket attributes: %w", err)
	}

	// Convert to our bucket format
	storageBucket := StorageBucket{
		Name:           attrs.Name,
		Location:       attrs.Location,
		StorageClass:   attrs.StorageClass,
		TimeCreated:    attrs.Created,
		Updated:        attrs.Updated,
		Metageneration: attrs.MetaGeneration,
		Labels:         attrs.Labels,
		Versioning:     attrs.VersioningEnabled,
		RequesterPays:  attrs.RequesterPays,
	}

	result.Data = storageBucket
	return nil
}

func (n *CloudStorageNode) setBucketLifecycle(operation *StorageOperation, context *expressions.ExpressionContext, item model.DataItem, result *StorageOperationResult) error {
	if operation.Lifecycle == nil {
		return fmt.Errorf("lifecycle configuration is required")
	}

	// Get bucket handle
	bucket := n.storageClient.Bucket(operation.Bucket)

	// Prepare bucket attributes update
	lifecycle := n.convertToGCSLifecycle(operation.Lifecycle)
	bucketAttrsToUpdate := storage.BucketAttrsToUpdate{
		Lifecycle: &lifecycle,
	}

	// Update bucket attributes
	_, err := bucket.Update(n.ctx, bucketAttrsToUpdate)
	if err != nil {
		return fmt.Errorf("failed to set bucket lifecycle: %w", err)
	}

	return nil
}

func (n *CloudStorageNode) generateSignedURL(operation *StorageOperation, context *expressions.ExpressionContext, item model.DataItem, result *StorageOperationResult) error {
	// Evaluate object name
	objectName, err := n.evaluateExpression(operation.Object, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate object name: %w", err)
	}
	result.Object = fmt.Sprintf("%v", objectName)

	// Get expiration time
	expiration := 15 * time.Minute // Default 15 minutes
	if exp, ok := operation.Options["expiration"].(int); ok {
		expiration = time.Duration(exp) * time.Minute
	}

	// Get HTTP method
	method := "GET"
	if m, ok := operation.Options["method"].(string); ok {
		method = strings.ToUpper(m)
	}

	// Get bucket handle
	bucket := n.storageClient.Bucket(operation.Bucket)

	// Generate signed URL
	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  method,
		Expires: time.Now().Add(expiration),
	}

	signedURL, err := bucket.SignedURL(result.Object, opts)
	if err != nil {
		return fmt.Errorf("failed to generate signed URL: %w", err)
	}

	result.URL = signedURL
	return nil
}

// Helper methods
func (n *CloudStorageNode) convertToGCSLifecycle(lifecycle *StorageLifecycle) storage.Lifecycle {
	var rules []storage.LifecycleRule

	for _, rule := range lifecycle.Rules {
		gcsRule := storage.LifecycleRule{
			Action: storage.LifecycleAction{
				Type: rule.Action.Type,
			},
			Condition: storage.LifecycleCondition{
				AgeInDays:             int64(rule.Condition.Age),
				MatchesStorageClasses: rule.Condition.MatchesStorageClasses,
				MatchesPrefix:         rule.Condition.MatchesPrefix,
				MatchesSuffix:         rule.Condition.MatchesSuffix,
				NumNewerVersions:      int64(rule.Condition.NumNewerVersions),
			},
		}

		if rule.Action.StorageClass != "" {
			gcsRule.Action.StorageClass = rule.Action.StorageClass
		}

		if rule.Condition.CreatedBefore != "" {
			if createdBefore, err := time.Parse("2006-01-02", rule.Condition.CreatedBefore); err == nil {
				gcsRule.Condition.CreatedBefore = createdBefore
			}
		}

		if rule.Condition.IsLive != nil {
			gcsRule.Condition.Liveness = storage.Live
			if !*rule.Condition.IsLive {
				gcsRule.Condition.Liveness = storage.Archived
			}
		}

		rules = append(rules, gcsRule)
	}

	return storage.Lifecycle{Rules: rules}
}

func (n *CloudStorageNode) parseSourceDestination(path string, context *expressions.ExpressionContext) (bucket, object string, err error) {
	// Evaluate path
	pathValue, err := n.evaluateExpression(path, context)
	if err != nil {
		return "", "", err
	}

	pathStr := fmt.Sprintf("%v", pathValue)

	// Parse gs://bucket/object format or bucket/object
	if strings.HasPrefix(pathStr, "gs://") {
		pathStr = pathStr[5:] // Remove gs:// prefix
	}

	parts := strings.SplitN(pathStr, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid path format: %s (expected bucket/object)", pathStr)
	}

	return parts[0], parts[1], nil
}

func (n *CloudStorageNode) detectContentType(objectName string) string {
	// Basic content type detection based on file extension
	if strings.HasSuffix(objectName, ".json") {
		return "application/json"
	} else if strings.HasSuffix(objectName, ".xml") {
		return "application/xml"
	} else if strings.HasSuffix(objectName, ".html") {
		return "text/html"
	} else if strings.HasSuffix(objectName, ".css") {
		return "text/css"
	} else if strings.HasSuffix(objectName, ".js") {
		return "application/javascript"
	} else if strings.HasSuffix(objectName, ".jpg") || strings.HasSuffix(objectName, ".jpeg") {
		return "image/jpeg"
	} else if strings.HasSuffix(objectName, ".png") {
		return "image/png"
	} else if strings.HasSuffix(objectName, ".pdf") {
		return "application/pdf"
	} else if strings.HasSuffix(objectName, ".txt") {
		return "text/plain"
	}

	return "application/octet-stream"
}

// Configuration parsing methods
func (n *CloudStorageNode) parseGCPConfig(nodeParams map[string]interface{}) (*GCPConfig, error) {
	config := &GCPConfig{}

	if projectID, ok := nodeParams["projectId"].(string); ok {
		config.ProjectID = projectID
	}

	if keyFile, ok := nodeParams["keyFile"].(string); ok {
		config.KeyFile = keyFile
	}

	if keyFileContent, ok := nodeParams["keyFileContent"].(string); ok {
		config.KeyFileContent = keyFileContent
	}

	if serviceAccountEmail, ok := nodeParams["serviceAccountEmail"].(string); ok {
		config.ServiceAccountEmail = serviceAccountEmail
	}

	if useADC, ok := nodeParams["useADC"].(bool); ok {
		config.UseADC = useADC
	}

	return config, nil
}

func (n *CloudStorageNode) parseStorageOperation(nodeParams map[string]interface{}) (*StorageOperation, error) {
	operation := &StorageOperation{
		Options: make(map[string]interface{}),
	}

	if op, ok := nodeParams["operation"].(string); ok {
		operation.Operation = op
	} else {
		return nil, fmt.Errorf("operation parameter is required")
	}

	if bucket, ok := nodeParams["bucket"].(string); ok {
		operation.Bucket = bucket
	} else {
		return nil, fmt.Errorf("bucket parameter is required")
	}

	if object, ok := nodeParams["object"].(string); ok {
		operation.Object = object
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

	if delimiter, ok := nodeParams["delimiter"].(string); ok {
		operation.Delimiter = delimiter
	}

	if contentType, ok := nodeParams["contentType"].(string); ok {
		operation.ContentType = contentType
	}

	if storageClass, ok := nodeParams["storageClass"].(string); ok {
		operation.StorageClass = storageClass
	}

	if versioning, ok := nodeParams["versioning"].(bool); ok {
		operation.Versioning = versioning
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

func (n *CloudStorageNode) initializeStorageClient(config *GCPConfig) error {
	var opts []option.ClientOption

	// Set credentials
	if config.KeyFile != "" {
		opts = append(opts, option.WithCredentialsFile(config.KeyFile))
	} else if config.KeyFileContent != "" {
		opts = append(opts, option.WithCredentialsJSON([]byte(config.KeyFileContent)))
	} else if !config.UseADC {
		// If not using ADC and no credentials provided, return error
		return fmt.Errorf("credentials must be provided: either keyFile, keyFileContent, or set useADC to true")
	}

	// Create client
	client, err := storage.NewClient(n.ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to create storage client: %w", err)
	}

	n.storageClient = client
	return nil
}

func (n *CloudStorageNode) evaluateExpression(expr string, context *expressions.ExpressionContext) (interface{}, error) {
	return n.evaluator.EvaluateExpression(expr, context)
}

// ValidateParameters validates the node parameters
func (n *CloudStorageNode) ValidateParameters(params map[string]interface{}) error {
	// Validate required parameters
	if _, ok := params["operation"]; !ok {
		return fmt.Errorf("operation parameter is required")
	}

	if _, ok := params["bucket"]; !ok {
		return fmt.Errorf("bucket parameter is required")
	}

	// Validate operation
	if operation, ok := params["operation"].(string); ok {
		validOperations := []string{
			"upload", "download", "delete", "list", "copy", "move",
			"get_metadata", "set_metadata", "create_bucket", "delete_bucket",
			"list_buckets", "get_bucket", "set_bucket_lifecycle",
			"generate_signed_url",
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