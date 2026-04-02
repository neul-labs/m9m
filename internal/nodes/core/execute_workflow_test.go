package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockExecutor struct {
	result *WorkflowResult
	err    error
}

func (m *mockExecutor) ExecuteWorkflow(workflow *model.Workflow, inputData []model.DataItem) (*WorkflowResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func writeTestWorkflow(t *testing.T, dir string) string {
	t.Helper()
	wf := model.Workflow{
		Name: "sub-workflow",
		Nodes: []model.Node{
			{Name: "Start", Type: "n8n-nodes-base.start", Position: []int{0, 0}},
		},
	}
	data, _ := json.Marshal(wf)
	path := filepath.Join(dir, "sub.json")
	require.NoError(t, os.WriteFile(path, data, 0644))
	return path
}

func TestExecuteWorkflowNode_Execute(t *testing.T) {
	t.Run("executes sub-workflow", func(t *testing.T) {
		dir := t.TempDir()
		wfPath := writeTestWorkflow(t, dir)

		executor := &mockExecutor{
			result: &WorkflowResult{
				Data: []model.DataItem{{JSON: map[string]interface{}{"result": "ok"}}},
			},
		}

		node := NewExecuteWorkflowNode(executor)
		params := map[string]interface{}{
			"source":       "localFile",
			"workflowPath": wfPath,
		}

		input := []model.DataItem{{JSON: map[string]interface{}{"input": "data"}}}
		result, err := node.Execute(input, params)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "ok", result[0].JSON["result"])
	})

	t.Run("no executor", func(t *testing.T) {
		node := NewExecuteWorkflowNode(nil)
		_, err := node.Execute(nil, map[string]interface{}{
			"workflowPath": "/tmp/test.json",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no workflow engine")
	})

	t.Run("max depth exceeded", func(t *testing.T) {
		dir := t.TempDir()
		wfPath := writeTestWorkflow(t, dir)

		executor := &mockExecutor{
			result: &WorkflowResult{Data: []model.DataItem{}},
		}

		node := NewExecuteWorkflowNode(executor)
		node.depth = 10
		params := map[string]interface{}{
			"workflowPath": wfPath,
			"maxDepth":     10,
		}

		_, err := node.Execute(nil, params)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "maximum recursion depth")
	})

	t.Run("missing workflow path", func(t *testing.T) {
		executor := &mockExecutor{}
		node := NewExecuteWorkflowNode(executor)
		_, err := node.Execute(nil, map[string]interface{}{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "workflowPath is required")
	})

	t.Run("file not found", func(t *testing.T) {
		executor := &mockExecutor{}
		node := NewExecuteWorkflowNode(executor)
		_, err := node.Execute(nil, map[string]interface{}{
			"workflowPath": "/nonexistent/workflow.json",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot read workflow file")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bad.json")
		require.NoError(t, os.WriteFile(path, []byte("not json"), 0644))

		executor := &mockExecutor{}
		node := NewExecuteWorkflowNode(executor)
		_, err := node.Execute(nil, map[string]interface{}{
			"workflowPath": path,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid workflow JSON")
	})

	t.Run("unsupported source", func(t *testing.T) {
		executor := &mockExecutor{}
		node := NewExecuteWorkflowNode(executor)
		_, err := node.Execute(nil, map[string]interface{}{
			"source":       "remote",
			"workflowPath": "/tmp/test.json",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported source")
	})
}

func TestExecuteWorkflowNode_ValidateParameters(t *testing.T) {
	node := NewExecuteWorkflowNode(nil)

	assert.NoError(t, node.ValidateParameters(nil))
	assert.Error(t, node.ValidateParameters(map[string]interface{}{
		"workflowPath": "",
	}))
	assert.NoError(t, node.ValidateParameters(map[string]interface{}{
		"workflowPath": "/some/path.json",
	}))
	assert.Error(t, node.ValidateParameters(map[string]interface{}{
		"source":       "remote",
		"workflowPath": "/some/path.json",
	}))
}

func TestExecuteWorkflowNode_Description(t *testing.T) {
	node := NewExecuteWorkflowNode(nil)
	desc := node.Description()
	assert.Equal(t, "Execute Workflow", desc.Name)
	assert.Equal(t, "Core", desc.Category)
}
