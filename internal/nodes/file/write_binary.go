package file

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dipankar/n8n-go/internal/expressions"
	"github.com/dipankar/n8n-go/internal/model"
	"github.com/dipankar/n8n-go/internal/nodes/base"
)

// WriteBinaryFileNode writes binary data to files on the filesystem with enhanced security
type WriteBinaryFileNode struct {
	*base.BaseNode
	evaluator         *expressions.GojaExpressionEvaluator
	allowedDirectories []string // Security: restrict file access to specific directories
	maxFileSize       int64    // Security: limit maximum file size
}

// NewWriteBinaryFileNode creates a new WriteBinaryFileNode instance
func NewWriteBinaryFileNode() *WriteBinaryFileNode {
	return &WriteBinaryFileNode{
		BaseNode:          base.NewBaseNode(base.NodeDescription{Name: "Write Binary File", Description: "n8n-nodes-base.writeBinaryFile", Category: "core"}),
		evaluator:         expressions.NewGojaExpressionEvaluator(expressions.DefaultEvaluatorConfig()),
		allowedDirectories: []string{"/tmp", "/var/tmp", "./data", "./uploads", "./output"}, // Default allowed directories
		maxFileSize:       100 * 1024 * 1024, // 100MB default limit
	}
}

// Execute writes binary data to files with comprehensive features
func (n *WriteBinaryFileNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	var results []model.DataItem

	for index, item := range inputData {
		// Create expression context
		context := &expressions.ExpressionContext{
			ActiveNodeName:      "Write Binary File",
			RunIndex:           0,
			ItemIndex:          index,
			Mode:               expressions.ModeManual,
			ConnectionInputData: []model.DataItem{item},
			AdditionalKeys: &expressions.AdditionalKeys{},
		}

		// Evaluate file path
		filePathExpr, ok := nodeParams["filePath"].(string)
		if !ok {
			return nil, n.CreateError("filePath parameter is required", map[string]interface{}{
				"nodeParams": nodeParams,
			})
		}

		filePath, err := n.evaluator.EvaluateExpression(filePathExpr, context)
		if err != nil {
			return nil, n.CreateError("failed to evaluate file path", map[string]interface{}{
				"expression": filePathExpr,
				"error":      err.Error(),
			})
		}

		filePathStr, ok := filePath.(string)
		if !ok {
			return nil, n.CreateError("file path must be a string", map[string]interface{}{
				"filePath": filePath,
				"type":     fmt.Sprintf("%T", filePath),
			})
		}

		// Security validation
		if err := n.validateFilePath(filePathStr); err != nil {
			return nil, n.CreateError("invalid file path", map[string]interface{}{
				"path":  filePathStr,
				"error": err.Error(),
			})
		}

		// Get data to write
		data, err := n.extractDataToWrite(item, nodeParams, context)
		if err != nil {
			return nil, n.CreateError("failed to extract data to write", map[string]interface{}{
				"error": err.Error(),
			})
		}

		// Validate file size
		if int64(len(data)) > n.maxFileSize {
			return nil, n.CreateError("data too large to write", map[string]interface{}{
				"dataSize": len(data),
				"maxSize":  n.maxFileSize,
			})
		}

		// Create directory if requested
		createDir, _ := nodeParams["createDirectory"].(bool)
		if createDir {
			dir := filepath.Dir(filePathStr)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, n.CreateError("failed to create directory", map[string]interface{}{
					"directory": dir,
					"error":     err.Error(),
				})
			}
		}

		// Check if file exists and handle overwrite
		overwrite, _ := nodeParams["overwrite"].(bool)
		if !overwrite {
			if _, err := os.Stat(filePathStr); err == nil {
				return nil, n.CreateError("file already exists and overwrite is disabled", map[string]interface{}{
					"path": filePathStr,
				})
			}
		}

		// Write file
		bytesWritten, err := n.writeFile(filePathStr, data)
		if err != nil {
			return nil, n.CreateError("failed to write file", map[string]interface{}{
				"path":  filePathStr,
				"error": err.Error(),
			})
		}

		// Get file info after writing
		fileInfo, err := os.Stat(filePathStr)
		if err != nil {
			return nil, n.CreateError("failed to get file info after writing", map[string]interface{}{
				"path":  filePathStr,
				"error": err.Error(),
			})
		}

		// Create result item
		resultItem := model.DataItem{
			JSON: map[string]interface{}{
				"filePath":     filePathStr,
				"fileName":     filepath.Base(filePathStr),
				"bytesWritten": bytesWritten,
				"size":         fileInfo.Size(),
				"modTime":      fileInfo.ModTime().Format(time.RFC3339),
				"writtenAt":    time.Now().UTC().Format(time.RFC3339),
				"success":      true,
			},
		}

		// Copy original data if requested
		keepOriginalData, _ := nodeParams["keepOriginalData"].(bool)
		if keepOriginalData {
			for k, v := range item.JSON {
				if k != "filePath" && k != "fileName" && k != "bytesWritten" &&
				   k != "size" && k != "modTime" && k != "writtenAt" && k != "success" {
					resultItem.JSON[k] = v
				}
			}
		}

		results = append(results, resultItem)
	}

	return results, nil
}

// validateFilePath ensures the file path is safe and allowed
func (n *WriteBinaryFileNode) validateFilePath(path string) error {
	// Prevent directory traversal attacks
	if strings.Contains(path, "..") {
		return fmt.Errorf("directory traversal not allowed")
	}

	// Get absolute path for validation
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Check if path is within allowed directories
	allowed := false
	for _, allowedDir := range n.allowedDirectories {
		allowedAbs, err := filepath.Abs(allowedDir)
		if err != nil {
			continue
		}
		if strings.HasPrefix(absPath, allowedAbs) {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("file access denied: path not in allowed directories")
	}

	return nil
}

// extractDataToWrite extracts the data to write from various sources
func (n *WriteBinaryFileNode) extractDataToWrite(item model.DataItem, nodeParams map[string]interface{}, context *expressions.ExpressionContext) ([]byte, error) {
	dataSource, _ := nodeParams["dataSource"].(string)
	if dataSource == "" {
		dataSource = "binary" // Default to binary data
	}

	switch dataSource {
	case "binary":
		// Use binary data from the item
		if item.Binary != nil {
			return item.Binary, nil
		}
		return nil, fmt.Errorf("no binary data available in item")

	case "json":
		// Extract data from JSON field
		dataField, _ := nodeParams["dataField"].(string)
		if dataField == "" {
			dataField = "data"
		}

		dataFieldExpr := fmt.Sprintf("{{ $json.%s }}", dataField)
		dataValue, err := n.evaluator.EvaluateExpression(dataFieldExpr, context)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate data field: %w", err)
		}

		return n.convertToBytes(dataValue, nodeParams)

	case "expression":
		// Evaluate expression to get data
		dataExpr, ok := nodeParams["dataExpression"].(string)
		if !ok {
			return nil, fmt.Errorf("dataExpression parameter is required when dataSource is 'expression'")
		}

		dataValue, err := n.evaluator.EvaluateExpression(dataExpr, context)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate data expression: %w", err)
		}

		return n.convertToBytes(dataValue, nodeParams)

	default:
		return nil, fmt.Errorf("unsupported data source: %s", dataSource)
	}
}

// convertToBytes converts various data types to byte array
func (n *WriteBinaryFileNode) convertToBytes(value interface{}, nodeParams map[string]interface{}) ([]byte, error) {
	encoding, _ := nodeParams["encoding"].(string)
	if encoding == "" {
		encoding = "auto"
	}

	switch v := value.(type) {
	case []byte:
		return v, nil
	case string:
		return n.decodeString(v, encoding)
	default:
		// Convert to string first, then to bytes
		str := fmt.Sprintf("%v", v)
		return n.decodeString(str, encoding)
	}
}

// decodeString decodes a string based on the specified encoding
func (n *WriteBinaryFileNode) decodeString(str, encoding string) ([]byte, error) {
	switch strings.ToLower(encoding) {
	case "auto", "utf8", "text":
		return []byte(str), nil
	case "base64":
		decoded, err := base64.StdEncoding.DecodeString(str)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64: %w", err)
		}
		return decoded, nil
	case "hex":
		decoded, err := hex.DecodeString(str)
		if err != nil {
			return nil, fmt.Errorf("failed to decode hex: %w", err)
		}
		return decoded, nil
	default:
		return nil, fmt.Errorf("unsupported encoding: %s", encoding)
	}
}

// writeFile writes data to the specified file
func (n *WriteBinaryFileNode) writeFile(path string, data []byte) (int, error) {
	file, err := os.Create(path)
	if err != nil {
		return 0, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	bytesWritten, err := file.Write(data)
	if err != nil {
		// Try to remove the partially written file
		os.Remove(path)
		return 0, fmt.Errorf("failed to write data: %w", err)
	}

	// Ensure data is written to disk
	if err := file.Sync(); err != nil {
		return bytesWritten, fmt.Errorf("failed to sync file: %w", err)
	}

	return bytesWritten, nil
}

// ValidateParameters validates the node parameters
func (n *WriteBinaryFileNode) ValidateParameters(params map[string]interface{}) error {
	// Check required parameters
	if _, ok := params["filePath"]; !ok {
		return fmt.Errorf("filePath parameter is required")
	}

	// Validate data source
	if dataSource, ok := params["dataSource"]; ok {
		if dataSourceStr, ok := dataSource.(string); ok {
			validSources := []string{"binary", "json", "expression"}
			valid := false
			for _, validSource := range validSources {
				if strings.ToLower(dataSourceStr) == validSource {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid data source: %s. Valid sources: %v", dataSourceStr, validSources)
			}

			// Additional validation based on data source
			if dataSourceStr == "expression" {
				if _, ok := params["dataExpression"]; !ok {
					return fmt.Errorf("dataExpression parameter is required when dataSource is 'expression'")
				}
			}
		}
	}

	// Validate encoding if provided
	if encoding, ok := params["encoding"]; ok {
		if encodingStr, ok := encoding.(string); ok {
			validEncodings := []string{"auto", "utf8", "text", "base64", "hex"}
			valid := false
			for _, validEnc := range validEncodings {
				if strings.ToLower(encodingStr) == validEnc {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid encoding: %s. Valid encodings: %v", encodingStr, validEncodings)
			}
		}
	}

	return nil
}

// SetAllowedDirectories configures which directories are allowed for file access
func (n *WriteBinaryFileNode) SetAllowedDirectories(dirs []string) {
	n.allowedDirectories = dirs
}

// SetMaxFileSize configures the maximum allowed file size
func (n *WriteBinaryFileNode) SetMaxFileSize(size int64) {
	n.maxFileSize = size
}