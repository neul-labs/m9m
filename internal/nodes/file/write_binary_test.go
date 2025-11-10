package file

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	
	"github.com/dipankar/n8n-go/internal/model"
	"github.com/dipankar/n8n-go/internal/nodes/base"
)

func TestWriteBinaryFileNodeCreation(t *testing.T) {
	node := NewWriteBinaryFileNode()
	if node == nil {
		t.Fatal("Expected node to be created, got nil")
	}
	
	desc := node.Description()
	if desc.Name != "Write Binary File" {
		t.Errorf("Expected name 'Write Binary File', got '%s'", desc.Name)
	}
}

func TestWriteBinaryFileNodeValidateParameters(t *testing.T) {
	node := NewWriteBinaryFileNode()
	
	// Test with nil params
	err := node.ValidateParameters(nil)
	if err == nil {
		t.Error("Expected error with nil params, got nil")
	}
	
	// Test with missing filePath
	params := map[string]interface{}{}
	err = node.ValidateParameters(params)
	if err == nil {
		t.Error("Expected error with missing filePath, got nil")
	}
	
	// Test with valid filePath
	validParams := map[string]interface{}{
		"filePath": "/path/to/file.txt",
	}
	err = node.ValidateParameters(validParams)
	if err != nil {
		t.Errorf("Expected no error with valid filePath, got %v", err)
	}
}

func TestWriteBinaryFileNodeExecuteWithEmptyInput(t *testing.T) {
	node := NewWriteBinaryFileNode()
	
	inputData := []model.DataItem{}
	nodeParams := map[string]interface{}{
		"filePath": "/path/to/file.txt",
	}
	
	result, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d items", len(result))
	}
}

func TestWriteBinaryFileNodeWriteFile(t *testing.T) {
	node := NewWriteBinaryFileNode()
	
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "test-write-file-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Test data
	content := "Hello, World!"
	encodedContent := base64.StdEncoding.EncodeToString([]byte(content))
	filePath := filepath.Join(tmpDir, "test-file.txt")
	
	// Test writing the file
	err = node.writeFile(filePath, encodedContent)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	
	// Verify the file was written correctly
	writtenContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}
	
	if string(writtenContent) != content {
		t.Errorf("Expected content '%s', got '%s'", content, string(writtenContent))
	}
}

func TestWriteBinaryFileNodeExecute(t *testing.T) {
	node := NewWriteBinaryFileNode()
	
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "test-write-file-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"data": base64.StdEncoding.EncodeToString([]byte("Hello, World!")),
			},
		},
	}
	
	filePath := filepath.Join(tmpDir, "output-file.txt")
	nodeParams := map[string]interface{}{
		"filePath": filePath,
	}
	
	result, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result) != 1 {
		t.Fatalf("Expected 1 result item, got %d", len(result))
	}
	
	// Check success field
	if success, ok := result[0].JSON["success"].(bool); !ok || !success {
		t.Error("Expected success field to be true")
	}
	
	// Check filePath field
	if path, ok := result[0].JSON["filePath"].(string); !ok || path != filePath {
		t.Errorf("Expected filePath field to be '%s', got '%v'", filePath, result[0].JSON["filePath"])
	}
	
	// Verify the file was written correctly
	writtenContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}
	
	if string(writtenContent) != "Hello, World!" {
		t.Errorf("Expected content 'Hello, World!', got '%s'", string(writtenContent))
	}
}

func TestWriteBinaryFileNodeImplementsNodeExecutor(t *testing.T) {
	var _ base.NodeExecutor = (*WriteBinaryFileNode)(nil)
}