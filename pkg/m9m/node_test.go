package m9m

import (
	"testing"
)

func TestNewBaseNode(t *testing.T) {
	desc := NodeDescription{
		Name:        "Test Node",
		Description: "A test node",
		Category:    "test",
	}

	node := NewBaseNode(desc)
	if node == nil {
		t.Fatal("NewBaseNode returned nil")
	}

	gotDesc := node.Description()
	if gotDesc.Name != "Test Node" {
		t.Errorf("Expected name 'Test Node', got '%s'", gotDesc.Name)
	}
	if gotDesc.Description != "A test node" {
		t.Errorf("Expected description 'A test node', got '%s'", gotDesc.Description)
	}
	if gotDesc.Category != "test" {
		t.Errorf("Expected category 'test', got '%s'", gotDesc.Category)
	}
}

func TestBaseNode_ValidateParameters(t *testing.T) {
	node := NewBaseNode(NodeDescription{Name: "Test"})

	// Default implementation accepts anything
	if err := node.ValidateParameters(nil); err != nil {
		t.Errorf("Unexpected error for nil params: %v", err)
	}
	if err := node.ValidateParameters(map[string]interface{}{"key": "value"}); err != nil {
		t.Errorf("Unexpected error for valid params: %v", err)
	}
}

func TestBaseNode_Execute(t *testing.T) {
	node := NewBaseNode(NodeDescription{Name: "Test"})
	input := []DataItem{{JSON: map[string]interface{}{"key": "value"}}}

	output, err := node.Execute(input, nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if len(output) != 1 {
		t.Errorf("Expected 1 output item, got %d", len(output))
	}
	if output[0].JSON["key"] != "value" {
		t.Error("Input data not passed through")
	}
}

func TestBaseNode_GetParameter(t *testing.T) {
	node := NewBaseNode(NodeDescription{Name: "Test"})
	params := map[string]interface{}{
		"existing": "value",
	}

	// Test existing parameter
	val := node.GetParameter(params, "existing", "default")
	if val != "value" {
		t.Errorf("Expected 'value', got '%v'", val)
	}

	// Test missing parameter
	val = node.GetParameter(params, "missing", "default")
	if val != "default" {
		t.Errorf("Expected 'default', got '%v'", val)
	}

	// Test nil params
	val = node.GetParameter(nil, "any", "default")
	if val != "default" {
		t.Errorf("Expected 'default' for nil params, got '%v'", val)
	}
}

func TestBaseNode_GetStringParameter(t *testing.T) {
	node := NewBaseNode(NodeDescription{Name: "Test"})
	params := map[string]interface{}{
		"string": "hello",
		"int":    42,
	}

	// Test string parameter
	val := node.GetStringParameter(params, "string", "default")
	if val != "hello" {
		t.Errorf("Expected 'hello', got '%s'", val)
	}

	// Test non-string parameter (should return default)
	val = node.GetStringParameter(params, "int", "default")
	if val != "default" {
		t.Errorf("Expected 'default' for non-string, got '%s'", val)
	}

	// Test missing parameter
	val = node.GetStringParameter(params, "missing", "default")
	if val != "default" {
		t.Errorf("Expected 'default', got '%s'", val)
	}
}

func TestBaseNode_GetIntParameter(t *testing.T) {
	node := NewBaseNode(NodeDescription{Name: "Test"})
	params := map[string]interface{}{
		"int":     42,
		"int64":   int64(100),
		"float":   3.14,
		"string":  "not a number",
	}

	// Test int parameter
	val := node.GetIntParameter(params, "int", 0)
	if val != 42 {
		t.Errorf("Expected 42, got %d", val)
	}

	// Test int64 parameter
	val = node.GetIntParameter(params, "int64", 0)
	if val != 100 {
		t.Errorf("Expected 100, got %d", val)
	}

	// Test float parameter (truncated)
	val = node.GetIntParameter(params, "float", 0)
	if val != 3 {
		t.Errorf("Expected 3, got %d", val)
	}

	// Test string parameter (should return default)
	val = node.GetIntParameter(params, "string", -1)
	if val != -1 {
		t.Errorf("Expected -1 for string, got %d", val)
	}

	// Test missing parameter
	val = node.GetIntParameter(params, "missing", 99)
	if val != 99 {
		t.Errorf("Expected 99, got %d", val)
	}
}

func TestBaseNode_GetFloat64Parameter(t *testing.T) {
	node := NewBaseNode(NodeDescription{Name: "Test"})
	params := map[string]interface{}{
		"float":  3.14,
		"int":    42,
		"int64":  int64(100),
		"string": "not a number",
	}

	// Test float parameter
	val := node.GetFloat64Parameter(params, "float", 0)
	if val != 3.14 {
		t.Errorf("Expected 3.14, got %f", val)
	}

	// Test int parameter
	val = node.GetFloat64Parameter(params, "int", 0)
	if val != 42.0 {
		t.Errorf("Expected 42.0, got %f", val)
	}

	// Test int64 parameter
	val = node.GetFloat64Parameter(params, "int64", 0)
	if val != 100.0 {
		t.Errorf("Expected 100.0, got %f", val)
	}

	// Test missing parameter
	val = node.GetFloat64Parameter(params, "missing", 1.5)
	if val != 1.5 {
		t.Errorf("Expected 1.5, got %f", val)
	}
}

func TestBaseNode_GetBoolParameter(t *testing.T) {
	node := NewBaseNode(NodeDescription{Name: "Test"})
	params := map[string]interface{}{
		"true":   true,
		"false":  false,
		"string": "true",
	}

	// Test true parameter
	val := node.GetBoolParameter(params, "true", false)
	if !val {
		t.Error("Expected true, got false")
	}

	// Test false parameter
	val = node.GetBoolParameter(params, "false", true)
	if val {
		t.Error("Expected false, got true")
	}

	// Test string parameter (should return default)
	val = node.GetBoolParameter(params, "string", false)
	if val {
		t.Error("Expected false for string, got true")
	}

	// Test missing parameter
	val = node.GetBoolParameter(params, "missing", true)
	if !val {
		t.Error("Expected true for missing, got false")
	}
}

func TestBaseNode_GetSliceParameter(t *testing.T) {
	node := NewBaseNode(NodeDescription{Name: "Test"})
	params := map[string]interface{}{
		"slice":  []interface{}{"a", "b", "c"},
		"string": "not a slice",
	}

	// Test slice parameter
	val := node.GetSliceParameter(params, "slice")
	if len(val) != 3 {
		t.Errorf("Expected 3 items, got %d", len(val))
	}

	// Test non-slice parameter
	val = node.GetSliceParameter(params, "string")
	if val != nil {
		t.Error("Expected nil for non-slice")
	}

	// Test missing parameter
	val = node.GetSliceParameter(params, "missing")
	if val != nil {
		t.Error("Expected nil for missing")
	}
}

func TestBaseNode_GetMapParameter(t *testing.T) {
	node := NewBaseNode(NodeDescription{Name: "Test"})
	params := map[string]interface{}{
		"map":    map[string]interface{}{"key": "value"},
		"string": "not a map",
	}

	// Test map parameter
	val := node.GetMapParameter(params, "map")
	if val == nil || val["key"] != "value" {
		t.Error("Map parameter not retrieved correctly")
	}

	// Test non-map parameter
	val = node.GetMapParameter(params, "string")
	if val != nil {
		t.Error("Expected nil for non-map")
	}

	// Test missing parameter
	val = node.GetMapParameter(params, "missing")
	if val != nil {
		t.Error("Expected nil for missing")
	}
}

func TestBaseNode_CreateError(t *testing.T) {
	node := NewBaseNode(NodeDescription{Name: "TestNode"})
	err := node.CreateError("something went wrong")

	expected := "node TestNode error: something went wrong"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}
}

func TestBaseNode_CreateErrorf(t *testing.T) {
	node := NewBaseNode(NodeDescription{Name: "TestNode"})
	err := node.CreateErrorf("value %d is invalid", 42)

	expected := "node TestNode error: value 42 is invalid"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}
}

func TestNewFunctionNode(t *testing.T) {
	executed := false
	fn := func(input []DataItem, params map[string]interface{}) ([]DataItem, error) {
		executed = true
		return input, nil
	}

	node := NewFunctionNode(NodeDescription{Name: "FuncNode"}, fn)
	if node == nil {
		t.Fatal("NewFunctionNode returned nil")
	}

	desc := node.Description()
	if desc.Name != "FuncNode" {
		t.Errorf("Expected name 'FuncNode', got '%s'", desc.Name)
	}

	input := []DataItem{{JSON: map[string]interface{}{}}}
	_, err := node.Execute(input, nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !executed {
		t.Error("Function was not executed")
	}
}

func TestNewFunctionNode_NilFunc(t *testing.T) {
	node := NewFunctionNode(NodeDescription{Name: "NilFunc"}, nil)
	input := []DataItem{{JSON: map[string]interface{}{}}}

	output, err := node.Execute(input, nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if len(output) != 1 {
		t.Errorf("Expected 1 output, got %d", len(output))
	}
}

func TestCredentialData_ToMap(t *testing.T) {
	cred := &CredentialData{
		ID:   "cred-123",
		Name: "My API Key",
		Type: "apiKey",
		Data: map[string]interface{}{
			"apiKey": "secret123",
		},
	}

	m := cred.ToMap()
	if m["id"] != "cred-123" {
		t.Errorf("Expected id 'cred-123', got '%v'", m["id"])
	}
	if m["name"] != "My API Key" {
		t.Errorf("Expected name 'My API Key', got '%v'", m["name"])
	}
	if m["type"] != "apiKey" {
		t.Errorf("Expected type 'apiKey', got '%v'", m["type"])
	}
}
