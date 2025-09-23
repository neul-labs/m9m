package timer

import (
	"testing"
	
	"github.com/yourusername/n8n-go/internal/model"
	"github.com/yourusername/n8n-go/internal/nodes/base"
)

func TestCronNodeCreation(t *testing.T) {
	node := NewCronNode()
	if node == nil {
		t.Fatal("Expected node to be created, got nil")
	}
	
	desc := node.Description()
	if desc.Name != "Cron" {
		t.Errorf("Expected name 'Cron', got '%s'", desc.Name)
	}
}

func TestCronNodeValidateParameters(t *testing.T) {
	node := NewCronNode()
	
	// Test with nil params
	err := node.ValidateParameters(nil)
	if err == nil {
		t.Error("Expected error with nil params, got nil")
	}
	
	// Test with missing cronExpression
	params := map[string]interface{}{}
	err = node.ValidateParameters(params)
	if err == nil {
		t.Error("Expected error with missing cronExpression, got nil")
	}
	
	// Test with valid cronExpression
	validParams := map[string]interface{}{
		"cronExpression": "* * * * *",
	}
	err = node.ValidateParameters(validParams)
	if err != nil {
		t.Errorf("Expected no error with valid cronExpression, got %v", err)
	}
}

func TestCronNodeExecute(t *testing.T) {
	node := NewCronNode()
	
	inputData := []model.DataItem{
		{
			JSON: map[string]interface{}{
				"existing": "data",
			},
		},
	}
	
	nodeParams := map[string]interface{}{
		"cronExpression": "* * * * *",
	}
	
	result, err := node.Execute(inputData, nodeParams)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result) != 1 {
		t.Fatalf("Expected 1 result item, got %d", len(result))
	}
	
	// Check that the result contains the expected fields
	if _, ok := result[0].JSON["cronExpression"]; !ok {
		t.Error("Expected cronExpression field in result")
	}
	
	if _, ok := result[0].JSON["triggeredAt"]; !ok {
		t.Error("Expected triggeredAt field in result")
	}
	
	if trigger, ok := result[0].JSON["trigger"].(bool); !ok || !trigger {
		t.Error("Expected trigger field to be true")
	}
}

func TestCronNodeParseCronExpression(t *testing.T) {
	node := NewCronNode()
	
	nextTime, err := node.ParseCronExpression("* * * * *")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	// Check that we got a valid time
	if nextTime.IsZero() {
		t.Error("Expected valid next execution time")
	}
}

func TestCronNodeImplementsNodeExecutor(t *testing.T) {
	var _ base.NodeExecutor = (*CronNode)(nil)
}