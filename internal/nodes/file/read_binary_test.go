package file

import (
	"encoding/base64"
	"os"
	"testing"
	
	"github.com/dipankar/n8n-go/internal/model"
	"github.com/dipankar/n8n-go/internal/nodes/base"
)

func TestReadBinaryFileNodeCreation(t *testing.T) {
	node := NewReadBinaryFileNode()
	if node == nil {
		t.Fatal("Expected node to be created, got nil")
	}
	
	desc := node.Description()
	if desc.Name != "Read Binary File" {
		t.Errorf("Expected name 'Read Binary File', got '%s'", desc.Name)
	}
}

func TestReadBinaryFileNodeValidateParameters(t *testing.T) {
	node := NewReadBinaryFileNode()
	
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

func TestReadBinaryFileNodeExecuteWithEmptyInput(t *testing.T) {
	node := NewReadBinaryFileNode()
	
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

func TestReadBinaryFileNodeReadFile(t *testing.T) {
	node := NewReadBinaryFileNode()
	
	// Create a temporary file for testing
	content := "Hello, World!"
	tmpFile, err := os.CreateTemp("", "test-file-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.WriteString(content)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()
	
	// Test reading the file
	encodedContent, err := node.readFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	
	// Decode and verify content
	decodedContent, err := base64.StdEncoding.DecodeString(encodedContent)
	if err != nil {
		t.Fatalf("Failed to decode content: %v", err)
	}
	
	if string(decodedContent) != content {
		t.Errorf("Expected content '%s', got '%s'", content, string(decodedContent))
	}
}

func TestReadBinaryFileNodeImplementsNodeExecutor(t *testing.T) {
	var _ base.NodeExecutor = (*ReadBinaryFileNode)(nil)
}