package email

import (
	"testing"
	
	"github.com/dipankar/n8n-go/internal/model"
	"github.com/dipankar/n8n-go/internal/nodes/base"
)

func TestSendEmailNodeCreation(t *testing.T) {
	node := NewSendEmailNode()
	if node == nil {
		t.Fatal("Expected node to be created, got nil")
	}
	
	desc := node.Description()
	if desc.Name != "Send Email" {
		t.Errorf("Expected name 'Send Email', got '%s'", desc.Name)
	}
}

func TestSendEmailNodeValidateParameters(t *testing.T) {
	node := NewSendEmailNode()
	
	// Test with nil params
	err := node.ValidateParameters(nil)
	if err == nil {
		t.Error("Expected error with nil params, got nil")
	}
	
	// Test with missing required parameters
	params := map[string]interface{}{}
	err = node.ValidateParameters(params)
	if err == nil {
		t.Error("Expected error with missing parameters, got nil")
	}
	
	// Test with missing smtpHost
	partialParams := map[string]interface{}{
		"smtpHost": "",
		"smtpPort": 587,
		"fromEmail": "sender@example.com",
		"toEmail":   "recipient@example.com",
		"subject":   "Test Subject",
	}
	err = node.ValidateParameters(partialParams)
	if err == nil {
		t.Error("Expected error with missing smtpHost, got nil")
	}
	
	// Test with missing smtpPort
	partialParams2 := map[string]interface{}{
		"smtpHost":  "smtp.example.com",
		"smtpPort":  0,
		"fromEmail": "sender@example.com",
		"toEmail":   "recipient@example.com",
		"subject":   "Test Subject",
	}
	err = node.ValidateParameters(partialParams2)
	if err == nil {
		t.Error("Expected error with missing smtpPort, got nil")
	}
	
	// Test with missing fromEmail
	partialParams3 := map[string]interface{}{
		"smtpHost":  "smtp.example.com",
		"smtpPort":  587,
		"fromEmail": "",
		"toEmail":   "recipient@example.com",
		"subject":   "Test Subject",
	}
	err = node.ValidateParameters(partialParams3)
	if err == nil {
		t.Error("Expected error with missing fromEmail, got nil")
	}
	
	// Test with missing toEmail
	partialParams4 := map[string]interface{}{
		"smtpHost":  "smtp.example.com",
		"smtpPort":  587,
		"fromEmail": "sender@example.com",
		"toEmail":   "",
		"subject":   "Test Subject",
	}
	err = node.ValidateParameters(partialParams4)
	if err == nil {
		t.Error("Expected error with missing toEmail, got nil")
	}
	
	// Test with missing subject
	partialParams5 := map[string]interface{}{
		"smtpHost":  "smtp.example.com",
		"smtpPort":  587,
		"fromEmail": "sender@example.com",
		"toEmail":   "recipient@example.com",
		"subject":   "",
	}
	err = node.ValidateParameters(partialParams5)
	if err == nil {
		t.Error("Expected error with missing subject, got nil")
	}
	
	// Test with valid parameters
	validParams := map[string]interface{}{
		"smtpHost":  "smtp.example.com",
		"smtpPort":  587,
		"fromEmail": "sender@example.com",
		"toEmail":   "recipient@example.com",
		"subject":   "Test Subject",
	}
	err = node.ValidateParameters(validParams)
	if err != nil {
		t.Errorf("Expected no error with valid parameters, got %v", err)
	}
}

func TestSendEmailNodeExecuteWithEmptyInput(t *testing.T) {
	node := NewSendEmailNode()
	
	inputData := []model.DataItem{}
	nodeParams := map[string]interface{}{
		"smtpHost":  "smtp.example.com",
		"smtpPort":  587,
		"username":  "testuser",
		"password":  "testpass",
		"fromEmail": "sender@example.com",
		"toEmail":   "recipient@example.com",
		"subject":   "Test Subject",
	}
	
	result, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d items", len(result))
	}
}

func TestSendEmailNodeImplementsNodeExecutor(t *testing.T) {
	var _ base.NodeExecutor = (*SendEmailNode)(nil)
}