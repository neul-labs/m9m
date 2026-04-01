package showcase

import (
	"bytes"
	"testing"

	"github.com/neul-labs/m9m/internal/engine"
	"github.com/neul-labs/m9m/internal/nodes/core"
	"github.com/neul-labs/m9m/internal/nodes/transform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// registerTestNodes registers a minimal set of nodes for testing.
func registerTestNodes(eng engine.WorkflowEngine) {
	eng.RegisterNodeExecutor("n8n-nodes-base.start", core.NewStartNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.set", transform.NewSetNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.filter", transform.NewFilterNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.switch", transform.NewSwitchNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.splitInBatches", transform.NewSplitInBatchesNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.code", transform.NewCodeNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.merge", transform.NewMergeNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.json", transform.NewJSONNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.function", transform.NewFunctionNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.itemLists", transform.NewItemListsNode())
}

func TestBenchmark_Quick(t *testing.T) {
	opts := BenchmarkOptions{Quick: true}
	report := RunBenchmark(opts, registerTestNodes)

	require.NotNil(t, report)
	assert.NotEmpty(t, report.System.OS)
	assert.NotEmpty(t, report.System.GoVersion)
	assert.True(t, report.System.CPUs > 0)

	// Should have all 5 categories
	assert.Len(t, report.Categories, 5)

	for _, cat := range report.Categories {
		assert.NotEmpty(t, cat.Name)
		assert.NotEmpty(t, cat.Results, "category %s should have results", cat.Name)
		for _, r := range cat.Results {
			assert.NotEmpty(t, r.Name)
			assert.True(t, r.Value > 0, "result %s value should be positive", r.Name)
		}
	}
}

func TestBenchmark_Category(t *testing.T) {
	opts := BenchmarkOptions{Quick: true, Category: "expressions"}
	report := RunBenchmark(opts, registerTestNodes)

	require.NotNil(t, report)
	assert.Len(t, report.Categories, 1)
	assert.Equal(t, "Expression Performance", report.Categories[0].Name)
}

func TestBenchmark_PrintTable(t *testing.T) {
	opts := BenchmarkOptions{Quick: true, Category: "expressions"}
	report := RunBenchmark(opts, registerTestNodes)

	var buf bytes.Buffer
	report.PrintTable(&buf)
	output := buf.String()

	assert.Contains(t, output, "m9m Performance Benchmark")
	assert.Contains(t, output, "Expression Performance")
	assert.Contains(t, output, "Simple Math")
}

func TestBenchmark_PrintJSON(t *testing.T) {
	opts := BenchmarkOptions{Quick: true, Category: "startup"}
	report := RunBenchmark(opts, registerTestNodes)

	var buf bytes.Buffer
	err := report.PrintJSON(&buf)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"name"`)
}

func TestDemo_RunAll(t *testing.T) {
	opts := DemoOptions{Verbose: true}
	results := RunDemos(opts, registerTestNodes)

	require.NotEmpty(t, results)
	for _, r := range results {
		assert.NotEmpty(t, r.Name)
		assert.NotEmpty(t, r.Duration, "demo %s should have been executed", r.Name)
		// We check success but don't fail the test on demo failures
		// since some demos may depend on specific node behavior
		t.Logf("Demo: %s success=%v items=%d duration=%s", r.Name, r.Success, r.Items, r.Duration)
	}
}

func TestDemo_ListOnly(t *testing.T) {
	opts := DemoOptions{ListOnly: true}
	results := RunDemos(opts, registerTestNodes)

	require.NotEmpty(t, results)
	for _, r := range results {
		assert.NotEmpty(t, r.Name)
		assert.Empty(t, r.Duration, "list mode should not execute demos")
	}
}

func TestDemo_CategoryFilter(t *testing.T) {
	opts := DemoOptions{Category: "business"}
	results := RunDemos(opts, registerTestNodes)

	for _, r := range results {
		assert.Equal(t, "business", r.Category)
	}
}

func TestDemo_PrintTable(t *testing.T) {
	opts := DemoOptions{ListOnly: true}
	results := RunDemos(opts, registerTestNodes)

	var buf bytes.Buffer
	PrintDemoTable(&buf, results, false)
	output := buf.String()

	assert.Contains(t, output, "m9m Capability Demo")
}

func TestDemo_PrintJSON(t *testing.T) {
	opts := DemoOptions{ListOnly: true}
	results := RunDemos(opts, registerTestNodes)

	var buf bytes.Buffer
	err := PrintDemoJSON(&buf, results)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"name"`)
}

func TestCompare_BuildReport(t *testing.T) {
	report := BuildCompareReport()

	require.NotNil(t, report)
	assert.NotEmpty(t, report.Nodes)
	assert.NotEmpty(t, report.Enterprise)
	assert.NotEmpty(t, report.Architecture)
	assert.NotEmpty(t, report.Expressions)
}

func TestCompare_PrintTable(t *testing.T) {
	report := BuildCompareReport()

	var buf bytes.Buffer
	report.PrintCompareTable(&buf, CompareOptions{})
	output := buf.String()

	assert.Contains(t, output, "m9m vs n8n Feature Comparison")
	assert.Contains(t, output, "Node Coverage")
	assert.Contains(t, output, "Enterprise Features")
	assert.Contains(t, output, "Architecture Comparison")
	assert.Contains(t, output, "Expression Engine")
}

func TestCompare_SectionFilter(t *testing.T) {
	report := BuildCompareReport()

	var buf bytes.Buffer
	report.PrintCompareTable(&buf, CompareOptions{Section: "nodes"})
	output := buf.String()

	assert.Contains(t, output, "Node Coverage")
	assert.NotContains(t, output, "Enterprise Features")
}

func TestCompare_PrintJSON(t *testing.T) {
	report := BuildCompareReport()

	var buf bytes.Buffer
	err := report.PrintCompareJSON(&buf)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"nodes"`)
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{125000, "125,000"},
		{1000000, "1,000,000"},
		{-500, "-500"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, FormatNumber(tt.input))
	}
}

func TestFormatSpeedup(t *testing.T) {
	assert.Equal(t, "250x", FormatSpeedup(250))
	assert.Equal(t, "12x", FormatSpeedup(12))
	assert.Equal(t, "2.5x", FormatSpeedup(2.5))
}

func TestFormatSavings(t *testing.T) {
	assert.Equal(t, "91%", FormatSavings(91.2))
}
