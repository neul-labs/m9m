package aws

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/dipankar/m9m/internal/expressions"
	"github.com/dipankar/m9m/internal/model"
	"github.com/dipankar/m9m/internal/nodes/base"
)

// S3OperationsNode provides comprehensive S3 operations
type S3OperationsNode struct {
	*base.BaseNode
	evaluator *expressions.GojaExpressionEvaluator
	s3Client  *s3.S3
	uploader  *s3manager.Uploader
	downloader *s3manager.Downloader
	session   *session.Session
}

// S3Config holds S3 connection configuration
type S3Config struct {
	Region          string `json:"region"`
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	SessionToken    string `json:"sessionToken,omitempty"`
	Endpoint        string `json:"endpoint,omitempty"` // For S3-compatible services
	S3ForcePathStyle bool  `json:"s3ForcePathStyle"`
	DisableSSL      bool   `json:"disableSSL"`
}

// S3Operation represents an S3 operation configuration
type S3Operation struct {
	Operation    string                 `json:"operation"` // upload, download, delete, list, copy, etc.
	Bucket       string                 `json:"bucket"`
	Key          string                 `json:"key,omitempty"`
	Source       string                 `json:"source,omitempty"`      // For upload operations
	Destination  string                 `json:"destination,omitempty"` // For download operations
	Prefix       string                 `json:"prefix,omitempty"`      // For list operations
	CopySource   *S3CopySource          `json:"copySource,omitempty"`
	Metadata     map[string]string      `json:"metadata,omitempty"`
	ContentType  string                 `json:"contentType,omitempty"`
	CacheControl string                 `json:"cacheControl,omitempty"`
	Encryption   *S3Encryption          `json:"encryption,omitempty"`
	ACL          string                 `json:"acl,omitempty"`
	StorageClass string                 `json:"storageClass,omitempty"`
	Versioning   bool                   `json:"versioning"`
	Lifecycle    *S3LifecycleConfig     `json:"lifecycle,omitempty"`
	Options      map[string]interface{} `json:"options,omitempty"`
}

// S3CopySource defines source for copy operations
type S3CopySource struct {
	Bucket    string `json:"bucket"`
	Key       string `json:"key"`
	VersionID string `json:"versionId,omitempty"`
}

// S3Encryption defines encryption settings
type S3Encryption struct {
	Method    string `json:"method"` // SSE-S3, SSE-KMS, SSE-C
	KMSKeyID  string `json:"kmsKeyId,omitempty"`
	CustomerKey string `json:"customerKey,omitempty"`
}

// S3LifecycleConfig defines lifecycle configuration
type S3LifecycleConfig struct {
	Rules []S3LifecycleRule `json:"rules"`
}

// S3LifecycleRule defines a lifecycle rule
type S3LifecycleRule struct {
	ID                   string                 `json:"id"`
	Status               string                 `json:"status"` // Enabled, Disabled
	Filter               *S3LifecycleFilter     `json:"filter,omitempty"`
	Transitions          []S3LifecycleTransition `json:"transitions,omitempty"`
	Expiration           *S3LifecycleExpiration `json:"expiration,omitempty"`
	AbortIncompleteMultipartUpload *S3AbortIncompleteMultipartUpload `json:"abortIncompleteMultipartUpload,omitempty"`
}

// S3LifecycleFilter defines lifecycle filter
type S3LifecycleFilter struct {
	Prefix string            `json:"prefix,omitempty"`
	Tags   map[string]string `json:"tags,omitempty"`
}

// S3LifecycleTransition defines lifecycle transition
type S3LifecycleTransition struct {
	Days         int    `json:"days"`
	StorageClass string `json:"storageClass"` // STANDARD_IA, GLACIER, DEEP_ARCHIVE
}

// S3LifecycleExpiration defines lifecycle expiration
type S3LifecycleExpiration struct {
	Days int `json:"days"`
}

// S3AbortIncompleteMultipartUpload defines incomplete multipart upload abort
type S3AbortIncompleteMultipartUpload struct {
	DaysAfterInitiation int `json:"daysAfterInitiation"`
}

// S3Object represents an S3 object
type S3Object struct {
	Key          string            `json:"key"`
	Size         int64             `json:"size"`
	LastModified time.Time         `json:"lastModified"`
	ETag         string            `json:"etag"`
	StorageClass string            `json:"storageClass"`
	Owner        string            `json:"owner,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	ContentType  string            `json:"contentType,omitempty"`
	VersionID    string            `json:"versionId,omitempty"`
	IsLatest     bool              `json:"isLatest"`
}

// S3OperationResult represents the result of an S3 operation
type S3OperationResult struct {
	Operation     string                 `json:"operation"`
	Success       bool                   `json:"success"`
	Bucket        string                 `json:"bucket"`
	Key           string                 `json:"key,omitempty"`
	Objects       []S3Object             `json:"objects,omitempty"`
	Data          interface{}            `json:"data,omitempty"`
	Metadata      map[string]string      `json:"metadata,omitempty"`
	ContentType   string                 `json:"contentType,omitempty"`
	Size          int64                  `json:"size,omitempty"`
	ETag          string                 `json:"etag,omitempty"`
	VersionID     string                 `json:"versionId,omitempty"`
	Location      string                 `json:"location,omitempty"`
	ExecutionTime time.Duration          `json:"executionTime"`
	Error         string                 `json:"error,omitempty"`
}

// NewS3OperationsNode creates a new S3 operations node
func NewS3OperationsNode() *S3OperationsNode {
	return &S3OperationsNode{
		BaseNode:  base.NewBaseNode(base.NodeDescription{Name: "AWS S3", Description: "Amazon S3 storage operations", Category: "cloud"}),
		evaluator: expressions.NewGojaExpressionEvaluator(expressions.DefaultEvaluatorConfig()),
	}
}

// Execute performs S3 operations
func (n *S3OperationsNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	var results []model.DataItem

	// Parse S3 configuration
	s3Config, err := n.parseS3Config(nodeParams)
	if err != nil {
		return nil, n.CreateError("invalid S3 configuration", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Initialize S3 client
	if err := n.initializeS3Client(s3Config); err != nil {
		return nil, n.CreateError("failed to initialize S3 client", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Parse operation configuration
	operation, err := n.parseS3Operation(nodeParams)
	if err != nil {
		return nil, n.CreateError("invalid S3 operation", map[string]interface{}{
			"error": err.Error(),
		})
	}

	for index, item := range inputData {
		// Create expression context
		context := &expressions.ExpressionContext{
			ActiveNodeName:      "AWS S3",
			RunIndex:           0,
			ItemIndex:          index,
			Mode:               expressions.ModeManual,
			ConnectionInputData: []model.DataItem{item},
			AdditionalKeys:     &expressions.AdditionalKeys{},
		}

		// Execute S3 operation
		result, err := n.executeS3Operation(operation, context, item)
		if err != nil {
			return nil, n.CreateError("S3 operation failed", map[string]interface{}{
				"operation": operation.Operation,
				"error":     err.Error(),
			})
		}

		// Create result item - convert S3OperationResult to map
		resultJSON := map[string]interface{}{
			"operation":     result.Operation,
			"success":       result.Success,
			"bucket":        result.Bucket,
			"key":           result.Key,
			"size":          result.Size,
			"etag":          result.ETag,
			"versionId":     result.VersionID,
			"location":      result.Location,
			"contentType":   result.ContentType,
			"executionTime": result.ExecutionTime.String(),
		}
		if result.Error != "" {
			resultJSON["error"] = result.Error
		}
		if len(result.Objects) > 0 {
			resultJSON["objects"] = result.Objects
		}
		if result.Metadata != nil {
			resultJSON["metadata"] = result.Metadata
		}

		resultItem := model.DataItem{
			JSON: resultJSON,
		}

		// Add binary data if operation downloaded data
		if operation.Operation == "download" && result.Data != nil {
			if dataBytes, ok := result.Data.([]byte); ok {
				// Store binary data in the proper format
				resultItem.Binary = map[string]model.BinaryData{
					"data": {
						Data:     base64.StdEncoding.EncodeToString(dataBytes),
						MimeType: result.ContentType,
						FileName: filepath.Base(result.Key),
					},
				}
			}
		}

		results = append(results, resultItem)
	}

	return results, nil
}

// executeS3Operation executes the specified S3 operation
func (n *S3OperationsNode) executeS3Operation(operation *S3Operation, context *expressions.ExpressionContext, item model.DataItem) (*S3OperationResult, error) {
	startTime := time.Now()

	result := &S3OperationResult{
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
	case "head":
		err = n.headObject(operation, context, item, result)
	case "create_bucket":
		err = n.createBucket(operation, context, item, result)
	case "delete_bucket":
		err = n.deleteBucket(operation, context, item, result)
	case "set_bucket_policy":
		err = n.setBucketPolicy(operation, context, item, result)
	case "get_bucket_policy":
		err = n.getBucketPolicy(operation, context, item, result)
	case "set_bucket_lifecycle":
		err = n.setBucketLifecycle(operation, context, item, result)
	case "set_bucket_versioning":
		err = n.setBucketVersioning(operation, context, item, result)
	case "generate_presigned_url":
		err = n.generatePresignedURL(operation, context, item, result)
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

// Upload operations
func (n *S3OperationsNode) uploadObject(operation *S3Operation, context *expressions.ExpressionContext, item model.DataItem, result *S3OperationResult) error {
	// Evaluate key
	key, err := n.evaluateExpression(operation.Key, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate key: %w", err)
	}
	keyStr := fmt.Sprintf("%v", key)
	result.Key = keyStr

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

	// Determine content type if not set
	if operation.ContentType != "" {
		contentType = operation.ContentType
	} else if contentType == "" {
		contentType = n.detectContentType(keyStr, data)
	}

	// Prepare upload input
	uploadInput := &s3manager.UploadInput{
		Bucket:      aws.String(operation.Bucket),
		Key:         aws.String(keyStr),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	}

	// Add metadata
	if len(operation.Metadata) > 0 {
		metadata := make(map[string]*string)
		for k, v := range operation.Metadata {
			metadata[k] = aws.String(v)
		}
		uploadInput.Metadata = metadata
	}

	// Add ACL
	if operation.ACL != "" {
		uploadInput.ACL = aws.String(operation.ACL)
	}

	// Add storage class
	if operation.StorageClass != "" {
		uploadInput.StorageClass = aws.String(operation.StorageClass)
	}

	// Add cache control
	if operation.CacheControl != "" {
		uploadInput.CacheControl = aws.String(operation.CacheControl)
	}

	// Add encryption
	if operation.Encryption != nil {
		switch operation.Encryption.Method {
		case "SSE-S3":
			uploadInput.ServerSideEncryption = aws.String("AES256")
		case "SSE-KMS":
			uploadInput.ServerSideEncryption = aws.String("aws:kms")
			if operation.Encryption.KMSKeyID != "" {
				uploadInput.SSEKMSKeyId = aws.String(operation.Encryption.KMSKeyID)
			}
		}
	}

	// Perform upload
	uploadResult, err := n.uploader.Upload(uploadInput)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	result.Location = uploadResult.Location
	result.ETag = *uploadResult.ETag
	if uploadResult.VersionID != nil {
		result.VersionID = *uploadResult.VersionID
	}
	result.Size = int64(len(data))
	result.ContentType = contentType

	return nil
}

// Download operations
func (n *S3OperationsNode) downloadObject(operation *S3Operation, context *expressions.ExpressionContext, item model.DataItem, result *S3OperationResult) error {
	// Evaluate key
	key, err := n.evaluateExpression(operation.Key, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate key: %w", err)
	}
	keyStr := fmt.Sprintf("%v", key)
	result.Key = keyStr

	// Create buffer for download
	buffer := &aws.WriteAtBuffer{}

	// Prepare download input
	downloadInput := &s3.GetObjectInput{
		Bucket: aws.String(operation.Bucket),
		Key:    aws.String(keyStr),
	}

	// Perform download
	_, err = n.downloader.Download(buffer, downloadInput)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	result.Data = buffer.Bytes()
	result.Size = int64(len(buffer.Bytes()))

	// Get object metadata
	headInput := &s3.HeadObjectInput{
		Bucket: aws.String(operation.Bucket),
		Key:    aws.String(keyStr),
	}

	headResult, err := n.s3Client.HeadObject(headInput)
	if err == nil {
		if headResult.ContentType != nil {
			result.ContentType = *headResult.ContentType
		}
		if headResult.ETag != nil {
			result.ETag = *headResult.ETag
		}
		if headResult.VersionId != nil {
			result.VersionID = *headResult.VersionId
		}
		if headResult.Metadata != nil {
			metadata := make(map[string]string)
			for k, v := range headResult.Metadata {
				if v != nil {
					metadata[k] = *v
				}
			}
			result.Metadata = metadata
		}
	}

	return nil
}

// Delete operations
func (n *S3OperationsNode) deleteObject(operation *S3Operation, context *expressions.ExpressionContext, item model.DataItem, result *S3OperationResult) error {
	// Evaluate key
	key, err := n.evaluateExpression(operation.Key, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate key: %w", err)
	}
	keyStr := fmt.Sprintf("%v", key)
	result.Key = keyStr

	// Prepare delete input
	deleteInput := &s3.DeleteObjectInput{
		Bucket: aws.String(operation.Bucket),
		Key:    aws.String(keyStr),
	}

	// Perform delete
	deleteResult, err := n.s3Client.DeleteObject(deleteInput)
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	if deleteResult.VersionId != nil {
		result.VersionID = *deleteResult.VersionId
	}

	return nil
}

// List operations
func (n *S3OperationsNode) listObjects(operation *S3Operation, context *expressions.ExpressionContext, item model.DataItem, result *S3OperationResult) error {
	// Prepare list input
	listInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(operation.Bucket),
	}

	if operation.Prefix != "" {
		prefix, err := n.evaluateExpression(operation.Prefix, context)
		if err != nil {
			return fmt.Errorf("failed to evaluate prefix: %w", err)
		}
		listInput.Prefix = aws.String(fmt.Sprintf("%v", prefix))
	}

	// Handle pagination options
	if maxKeys, ok := operation.Options["maxKeys"].(int); ok {
		listInput.MaxKeys = aws.Int64(int64(maxKeys))
	}

	if delimiter, ok := operation.Options["delimiter"].(string); ok {
		listInput.Delimiter = aws.String(delimiter)
	}

	// Perform list operation
	var objects []S3Object

	err := n.s3Client.ListObjectsV2Pages(listInput, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, obj := range page.Contents {
			s3Obj := S3Object{
				Key:          *obj.Key,
				Size:         *obj.Size,
				LastModified: *obj.LastModified,
				ETag:         strings.Trim(*obj.ETag, "\""),
				StorageClass: *obj.StorageClass,
			}

			if obj.Owner != nil && obj.Owner.DisplayName != nil {
				s3Obj.Owner = *obj.Owner.DisplayName
			}

			objects = append(objects, s3Obj)
		}
		return !lastPage
	})

	if err != nil {
		return fmt.Errorf("list failed: %w", err)
	}

	result.Objects = objects
	return nil
}

// Copy operations
func (n *S3OperationsNode) copyObject(operation *S3Operation, context *expressions.ExpressionContext, item model.DataItem, result *S3OperationResult) error {
	if operation.CopySource == nil {
		return fmt.Errorf("copy source is required for copy operation")
	}

	// Evaluate destination key
	key, err := n.evaluateExpression(operation.Key, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate destination key: %w", err)
	}
	keyStr := fmt.Sprintf("%v", key)
	result.Key = keyStr

	// Build copy source
	copySource := fmt.Sprintf("%s/%s", operation.CopySource.Bucket, operation.CopySource.Key)
	if operation.CopySource.VersionID != "" {
		copySource += "?versionId=" + operation.CopySource.VersionID
	}

	// Prepare copy input
	copyInput := &s3.CopyObjectInput{
		Bucket:     aws.String(operation.Bucket),
		Key:        aws.String(keyStr),
		CopySource: aws.String(copySource),
	}

	// Add metadata directive
	if len(operation.Metadata) > 0 {
		copyInput.MetadataDirective = aws.String("REPLACE")
		metadata := make(map[string]*string)
		for k, v := range operation.Metadata {
			metadata[k] = aws.String(v)
		}
		copyInput.Metadata = metadata
	}

	// Perform copy
	copyResult, err := n.s3Client.CopyObject(copyInput)
	if err != nil {
		return fmt.Errorf("copy failed: %w", err)
	}

	if copyResult.CopyObjectResult.ETag != nil {
		result.ETag = strings.Trim(*copyResult.CopyObjectResult.ETag, "\"")
	}

	if copyResult.VersionId != nil {
		result.VersionID = *copyResult.VersionId
	}

	return nil
}

// Head operations
func (n *S3OperationsNode) headObject(operation *S3Operation, context *expressions.ExpressionContext, item model.DataItem, result *S3OperationResult) error {
	// Evaluate key
	key, err := n.evaluateExpression(operation.Key, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate key: %w", err)
	}
	keyStr := fmt.Sprintf("%v", key)
	result.Key = keyStr

	// Prepare head input
	headInput := &s3.HeadObjectInput{
		Bucket: aws.String(operation.Bucket),
		Key:    aws.String(keyStr),
	}

	// Perform head operation
	headResult, err := n.s3Client.HeadObject(headInput)
	if err != nil {
		return fmt.Errorf("head failed: %w", err)
	}

	// Extract metadata
	if headResult.ContentType != nil {
		result.ContentType = *headResult.ContentType
	}

	if headResult.ContentLength != nil {
		result.Size = *headResult.ContentLength
	}

	if headResult.ETag != nil {
		result.ETag = strings.Trim(*headResult.ETag, "\"")
	}

	if headResult.VersionId != nil {
		result.VersionID = *headResult.VersionId
	}

	if headResult.Metadata != nil {
		metadata := make(map[string]string)
		for k, v := range headResult.Metadata {
			if v != nil {
				metadata[k] = *v
			}
		}
		result.Metadata = metadata
	}

	return nil
}

// Bucket operations
func (n *S3OperationsNode) createBucket(operation *S3Operation, context *expressions.ExpressionContext, item model.DataItem, result *S3OperationResult) error {
	// Prepare create bucket input
	createInput := &s3.CreateBucketInput{
		Bucket: aws.String(operation.Bucket),
	}

	// Add location constraint if region is not us-east-1
	if n.session.Config.Region != nil && *n.session.Config.Region != "us-east-1" {
		createInput.CreateBucketConfiguration = &s3.CreateBucketConfiguration{
			LocationConstraint: n.session.Config.Region,
		}
	}

	// Perform create bucket
	_, err := n.s3Client.CreateBucket(createInput)
	if err != nil {
		return fmt.Errorf("create bucket failed: %w", err)
	}

	// Enable versioning if requested
	if operation.Versioning {
		versioningInput := &s3.PutBucketVersioningInput{
			Bucket: aws.String(operation.Bucket),
			VersioningConfiguration: &s3.VersioningConfiguration{
				Status: aws.String("Enabled"),
			},
		}

		_, err = n.s3Client.PutBucketVersioning(versioningInput)
		if err != nil {
			return fmt.Errorf("failed to enable versioning: %w", err)
		}
	}

	// Set lifecycle configuration if provided
	if operation.Lifecycle != nil {
		if err := n.setBucketLifecycleConfig(operation); err != nil {
			return fmt.Errorf("failed to set lifecycle: %w", err)
		}
	}

	return nil
}

func (n *S3OperationsNode) deleteBucket(operation *S3Operation, context *expressions.ExpressionContext, item model.DataItem, result *S3OperationResult) error {
	// Prepare delete bucket input
	deleteInput := &s3.DeleteBucketInput{
		Bucket: aws.String(operation.Bucket),
	}

	// Perform delete bucket
	_, err := n.s3Client.DeleteBucket(deleteInput)
	if err != nil {
		return fmt.Errorf("delete bucket failed: %w", err)
	}

	return nil
}

func (n *S3OperationsNode) setBucketPolicy(operation *S3Operation, context *expressions.ExpressionContext, item model.DataItem, result *S3OperationResult) error {
	policy, ok := operation.Options["policy"].(string)
	if !ok {
		return fmt.Errorf("bucket policy is required")
	}

	// Evaluate policy as expression
	evaluatedPolicy, err := n.evaluateExpression(policy, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate policy: %w", err)
	}

	policyStr := fmt.Sprintf("%v", evaluatedPolicy)

	// Prepare put bucket policy input
	policyInput := &s3.PutBucketPolicyInput{
		Bucket: aws.String(operation.Bucket),
		Policy: aws.String(policyStr),
	}

	// Perform put bucket policy
	_, err = n.s3Client.PutBucketPolicy(policyInput)
	if err != nil {
		return fmt.Errorf("set bucket policy failed: %w", err)
	}

	return nil
}

func (n *S3OperationsNode) getBucketPolicy(operation *S3Operation, context *expressions.ExpressionContext, item model.DataItem, result *S3OperationResult) error {
	// Prepare get bucket policy input
	policyInput := &s3.GetBucketPolicyInput{
		Bucket: aws.String(operation.Bucket),
	}

	// Perform get bucket policy
	policyResult, err := n.s3Client.GetBucketPolicy(policyInput)
	if err != nil {
		return fmt.Errorf("get bucket policy failed: %w", err)
	}

	if policyResult.Policy != nil {
		result.Data = *policyResult.Policy
	}

	return nil
}

func (n *S3OperationsNode) setBucketLifecycle(operation *S3Operation, context *expressions.ExpressionContext, item model.DataItem, result *S3OperationResult) error {
	if operation.Lifecycle == nil {
		return fmt.Errorf("lifecycle configuration is required")
	}

	return n.setBucketLifecycleConfig(operation)
}

func (n *S3OperationsNode) setBucketVersioning(operation *S3Operation, context *expressions.ExpressionContext, item model.DataItem, result *S3OperationResult) error {
	status := "Enabled"
	if !operation.Versioning {
		status = "Suspended"
	}

	versioningInput := &s3.PutBucketVersioningInput{
		Bucket: aws.String(operation.Bucket),
		VersioningConfiguration: &s3.VersioningConfiguration{
			Status: aws.String(status),
		},
	}

	_, err := n.s3Client.PutBucketVersioning(versioningInput)
	if err != nil {
		return fmt.Errorf("set bucket versioning failed: %w", err)
	}

	return nil
}

func (n *S3OperationsNode) generatePresignedURL(operation *S3Operation, context *expressions.ExpressionContext, item model.DataItem, result *S3OperationResult) error {
	// Evaluate key
	key, err := n.evaluateExpression(operation.Key, context)
	if err != nil {
		return fmt.Errorf("failed to evaluate key: %w", err)
	}
	keyStr := fmt.Sprintf("%v", key)
	result.Key = keyStr

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

	var req *s3.GetObjectInput
	var presignedURL string

	switch method {
	case "GET":
		req = &s3.GetObjectInput{
			Bucket: aws.String(operation.Bucket),
			Key:    aws.String(keyStr),
		}
		request, _ := n.s3Client.GetObjectRequest(req)
		presignedURL, err = request.Presign(expiration)

	case "PUT":
		putReq := &s3.PutObjectInput{
			Bucket: aws.String(operation.Bucket),
			Key:    aws.String(keyStr),
		}
		request, _ := n.s3Client.PutObjectRequest(putReq)
		presignedURL, err = request.Presign(expiration)

	default:
		return fmt.Errorf("unsupported method for presigned URL: %s", method)
	}

	if err != nil {
		return fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	result.Data = presignedURL
	return nil
}

// Helper methods

func (n *S3OperationsNode) setBucketLifecycleConfig(operation *S3Operation) error {
	if operation.Lifecycle == nil || len(operation.Lifecycle.Rules) == 0 {
		return nil
	}

	var rules []*s3.LifecycleRule

	for _, rule := range operation.Lifecycle.Rules {
		s3Rule := &s3.LifecycleRule{
			ID:     aws.String(rule.ID),
			Status: aws.String(rule.Status),
		}

		// Add filter
		if rule.Filter != nil {
			if rule.Filter.Prefix != "" {
				s3Rule.Filter = &s3.LifecycleRuleFilter{
					Prefix: aws.String(rule.Filter.Prefix),
				}
			}
		}

		// Add transitions
		for _, transition := range rule.Transitions {
			s3Rule.Transitions = append(s3Rule.Transitions, &s3.Transition{
				Days:         aws.Int64(int64(transition.Days)),
				StorageClass: aws.String(transition.StorageClass),
			})
		}

		// Add expiration
		if rule.Expiration != nil {
			s3Rule.Expiration = &s3.LifecycleExpiration{
				Days: aws.Int64(int64(rule.Expiration.Days)),
			}
		}

		// Add abort incomplete multipart upload
		if rule.AbortIncompleteMultipartUpload != nil {
			s3Rule.AbortIncompleteMultipartUpload = &s3.AbortIncompleteMultipartUpload{
				DaysAfterInitiation: aws.Int64(int64(rule.AbortIncompleteMultipartUpload.DaysAfterInitiation)),
			}
		}

		rules = append(rules, s3Rule)
	}

	lifecycleInput := &s3.PutBucketLifecycleConfigurationInput{
		Bucket: aws.String(operation.Bucket),
		LifecycleConfiguration: &s3.BucketLifecycleConfiguration{
			Rules: rules,
		},
	}

	_, err := n.s3Client.PutBucketLifecycleConfiguration(lifecycleInput)
	return err
}

func (n *S3OperationsNode) detectContentType(key string, data []byte) string {
	// Basic content type detection based on file extension
	ext := strings.ToLower(filepath.Ext(key))

	contentTypes := map[string]string{
		".json": "application/json",
		".xml":  "application/xml",
		".html": "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".pdf":  "application/pdf",
		".zip":  "application/zip",
		".txt":  "text/plain",
		".csv":  "text/csv",
	}

	if contentType, exists := contentTypes[ext]; exists {
		return contentType
	}

	// Try to detect JSON
	if len(data) > 0 && (data[0] == '{' || data[0] == '[') {
		return "application/json"
	}

	return "application/octet-stream"
}

// Configuration parsing methods

func (n *S3OperationsNode) parseS3Config(nodeParams map[string]interface{}) (*S3Config, error) {
	config := &S3Config{}

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

	if endpoint, ok := nodeParams["endpoint"].(string); ok {
		config.Endpoint = endpoint
	}

	if s3ForcePathStyle, ok := nodeParams["s3ForcePathStyle"].(bool); ok {
		config.S3ForcePathStyle = s3ForcePathStyle
	}

	if disableSSL, ok := nodeParams["disableSSL"].(bool); ok {
		config.DisableSSL = disableSSL
	}

	return config, nil
}

func (n *S3OperationsNode) parseS3Operation(nodeParams map[string]interface{}) (*S3Operation, error) {
	operation := &S3Operation{
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

	if key, ok := nodeParams["key"].(string); ok {
		operation.Key = key
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

	if acl, ok := nodeParams["acl"].(string); ok {
		operation.ACL = acl
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

	// Parse copy source
	if copySource, ok := nodeParams["copySource"].(map[string]interface{}); ok {
		operation.CopySource = &S3CopySource{}
		if bucket, ok := copySource["bucket"].(string); ok {
			operation.CopySource.Bucket = bucket
		}
		if key, ok := copySource["key"].(string); ok {
			operation.CopySource.Key = key
		}
		if versionID, ok := copySource["versionId"].(string); ok {
			operation.CopySource.VersionID = versionID
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

func (n *S3OperationsNode) initializeS3Client(config *S3Config) error {
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

	// Set custom endpoint
	if config.Endpoint != "" {
		awsConfig.Endpoint = aws.String(config.Endpoint)
		awsConfig.S3ForcePathStyle = aws.Bool(config.S3ForcePathStyle)
	}

	// Disable SSL if requested
	if config.DisableSSL {
		awsConfig.DisableSSL = aws.Bool(true)
	}

	// Create session
	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return fmt.Errorf("failed to create AWS session: %w", err)
	}

	n.session = sess
	n.s3Client = s3.New(sess)
	n.uploader = s3manager.NewUploader(sess)
	n.downloader = s3manager.NewDownloader(sess)

	return nil
}

func (n *S3OperationsNode) evaluateExpression(expr string, context *expressions.ExpressionContext) (interface{}, error) {
	return n.evaluator.EvaluateExpression(expr, context)
}

// ValidateParameters validates the node parameters
func (n *S3OperationsNode) ValidateParameters(params map[string]interface{}) error {
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
			"upload", "download", "delete", "list", "copy", "head",
			"create_bucket", "delete_bucket", "set_bucket_policy",
			"get_bucket_policy", "set_bucket_lifecycle", "set_bucket_versioning",
			"generate_presigned_url",
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