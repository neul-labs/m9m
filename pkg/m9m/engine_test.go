package m9m

import (
	"os"
	"testing"
)

// setupTestEnv sets up the test environment
// SECURITY: These tests run in dev mode to allow auto-generated encryption keys
func setupTestEnv(t *testing.T) {
	t.Helper()
	os.Setenv("M9M_DEV_MODE", "true")
	t.Cleanup(func() {
		os.Unsetenv("M9M_DEV_MODE")
	})
}

func TestNew(t *testing.T) {
	engine := New()
	if engine == nil {
		t.Fatal("New() returned nil")
	}
	if engine.internal == nil {
		t.Fatal("Engine.internal is nil")
	}
}

func TestNewWithOptions(t *testing.T) {
	setupTestEnv(t)
	cm, err := NewCredentialManager()
	if err != nil {
		t.Fatalf("Failed to create credential manager: %v", err)
	}

	engine := NewWithOptions(WithCredentialManager(cm))
	if engine == nil {
		t.Fatal("NewWithOptions() returned nil")
	}
}

func TestEngine_Execute_NilWorkflow(t *testing.T) {
	engine := New()
	_, err := engine.Execute(nil, nil)
	if err != ErrNilWorkflow {
		t.Errorf("Expected ErrNilWorkflow, got %v", err)
	}
}

func TestEngine_Execute_EmptyWorkflow(t *testing.T) {
	engine := New()
	workflow := &Workflow{
		Name:  "test",
		Nodes: []Node{},
	}

	result, err := engine.Execute(workflow, nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result == nil {
		t.Fatal("Execute returned nil result")
	}
}

func TestEngine_Execute_WithInput(t *testing.T) {
	engine := New()
	workflow := &Workflow{
		Name:  "test",
		Nodes: []Node{},
	}

	input := []DataItem{
		{JSON: map[string]interface{}{"key": "value"}},
	}

	result, err := engine.Execute(workflow, input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result == nil {
		t.Fatal("Execute returned nil result")
	}
}

func TestEngine_RegisterNode(t *testing.T) {
	engine := New()

	node := &testNode{
		BaseNode: NewBaseNode(NodeDescription{
			Name:        "Test Node",
			Description: "A test node",
			Category:    "test",
		}),
	}

	engine.RegisterNode("test.node", node)

	retrieved, err := engine.GetNode("test.node")
	if err != nil {
		t.Fatalf("GetNode failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("GetNode returned nil")
	}

	desc := retrieved.Description()
	if desc.Name != "Test Node" {
		t.Errorf("Expected name 'Test Node', got '%s'", desc.Name)
	}
}

func TestEngine_GetNode_NotFound(t *testing.T) {
	engine := New()
	_, err := engine.GetNode("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent node, got nil")
	}
}

func TestEngine_ExecuteParallel_Empty(t *testing.T) {
	engine := New()
	results, err := engine.ExecuteParallel([]*Workflow{}, nil)
	if err != nil {
		t.Fatalf("ExecuteParallel failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestEngine_ExecuteParallel_Multiple(t *testing.T) {
	engine := New()

	workflows := []*Workflow{
		{Name: "test1", Nodes: []Node{}},
		{Name: "test2", Nodes: []Node{}},
	}

	inputs := [][]DataItem{
		{{JSON: map[string]interface{}{"id": 1}}},
		{{JSON: map[string]interface{}{"id": 2}}},
	}

	results, err := engine.ExecuteParallel(workflows, inputs)
	if err != nil {
		t.Fatalf("ExecuteParallel failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

// testNode is a simple node implementation for testing
type testNode struct {
	*BaseNode
}

func (n *testNode) Execute(inputData []DataItem, nodeParams map[string]interface{}) ([]DataItem, error) {
	// Add a marker to show this node was executed
	for i := range inputData {
		if inputData[i].JSON == nil {
			inputData[i].JSON = make(map[string]interface{})
		}
		inputData[i].JSON["testNodeExecuted"] = true
	}
	return inputData, nil
}
