package showcase

import (
	"encoding/json"
	"fmt"
	"io"
)

// CompareOptions controls comparison output.
type CompareOptions struct {
	Section string // "nodes", "features", "architecture", "expressions", or ""
}

// CompareReport holds the full comparison report.
type CompareReport struct {
	Nodes       []NodeCategoryCount `json:"nodes"`
	Enterprise  []EnterpriseFeature `json:"enterprise"`
	Architecture []ArchComparison   `json:"architecture"`
	Expressions []ExpressionCategory `json:"expressions"`
}

// NodeCategoryCount summarizes node coverage per category.
type NodeCategoryCount struct {
	Category    string `json:"category"`
	Count       int    `json:"count"`
	Description string `json:"description"`
}

// NodeCoverage returns node counts by category.
var NodeCoverage = []NodeCategoryCount{
	{"Core", 1, "Start, trigger entry points"},
	{"Transform", 9, "Set, Filter, Code, Merge, Switch, Function, JSON, SplitInBatches, ItemLists"},
	{"HTTP", 1, "HTTP Request (GET, POST, PUT, DELETE, etc.)"},
	{"Trigger", 2, "Webhook, Cron"},
	{"Messaging", 2, "Slack, Discord"},
	{"Database", 3, "PostgreSQL, MySQL, SQLite"},
	{"Email", 1, "SMTP Send Email"},
	{"Cloud (AWS)", 2, "Lambda, S3"},
	{"Cloud (Azure)", 1, "Blob Storage"},
	{"Cloud (GCP)", 1, "Cloud Storage"},
	{"AI/LLM", 2, "OpenAI, Anthropic (Claude)"},
	{"File", 2, "Read Binary, Write Binary"},
	{"VCS", 2, "GitHub, GitLab"},
	{"Productivity", 1, "Google Sheets"},
	{"CLI/Agents", 1, "CLI Execute (Claude Code, Codex, Aider)"},
	{"Code", 1, "Python Code execution"},
}

// BuildCompareReport builds the comparison report.
func BuildCompareReport() *CompareReport {
	return &CompareReport{
		Nodes:        NodeCoverage,
		Enterprise:   EnterpriseData,
		Architecture: ArchData,
		Expressions:  ExpressionCategories,
	}
}

// PrintCompareTable prints the comparison as ASCII tables.
func (r *CompareReport) PrintCompareTable(w io.Writer, opts CompareOptions) {
	SectionHeader(w, "m9m vs n8n Feature Comparison")

	sections := map[string]bool{
		"nodes":        opts.Section == "" || opts.Section == "nodes",
		"features":     opts.Section == "" || opts.Section == "features",
		"architecture": opts.Section == "" || opts.Section == "architecture",
		"expressions":  opts.Section == "" || opts.Section == "expressions",
	}

	if sections["nodes"] {
		r.printNodeSection(w)
	}
	if sections["features"] {
		r.printEnterpriseSection(w)
	}
	if sections["architecture"] {
		r.printArchSection(w)
	}
	if sections["expressions"] {
		r.printExpressionSection(w)
	}
}

func (r *CompareReport) printNodeSection(w io.Writer) {
	SubHeader(w, "Node Coverage")
	tw := NewTable(w)
	fmt.Fprintln(tw, "Category\tCount\tNodes")
	PrintDivider(w, 70)

	total := 0
	for _, nc := range r.Nodes {
		fmt.Fprintf(tw, "%s\t%d\t%s\n", nc.Category, nc.Count, nc.Description)
		total += nc.Count
	}
	tw.Flush()
	fmt.Fprintf(w, "\nTotal: %d nodes across %d categories\n", total, len(r.Nodes))
	fmt.Fprintln(w)
}

func (r *CompareReport) printEnterpriseSection(w io.Writer) {
	SubHeader(w, "Enterprise Features")
	tw := NewTable(w)
	fmt.Fprintln(tw, "Feature\tm9m\tn8n")
	PrintDivider(w, 60)

	for _, ef := range r.Enterprise {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", ef.Feature, ef.M9m, ef.N8n)
	}
	tw.Flush()
	fmt.Fprintln(w)
}

func (r *CompareReport) printArchSection(w io.Writer) {
	SubHeader(w, "Architecture Comparison")
	tw := NewTable(w)
	fmt.Fprintln(tw, "Aspect\tm9m\tn8n")
	PrintDivider(w, 70)

	for _, ac := range r.Architecture {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", ac.Feature, ac.M9m, ac.N8n)
	}
	tw.Flush()
	fmt.Fprintln(w)
}

func (r *CompareReport) printExpressionSection(w io.Writer) {
	SubHeader(w, "Expression Engine (100+ functions)")
	tw := NewTable(w)
	fmt.Fprintln(tw, "Category\tCount\tExamples")
	PrintDivider(w, 70)

	total := 0
	for _, ec := range r.Expressions {
		fmt.Fprintf(tw, "%s\t%d\t%s\n", ec.Category, ec.Count, ec.Examples)
		total += ec.Count
	}
	tw.Flush()
	fmt.Fprintf(w, "\nTotal: %d expression functions across %d categories\n", total, len(r.Expressions))
	fmt.Fprintln(w)
}

// PrintCompareJSON prints the report as JSON.
func (r *CompareReport) PrintCompareJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}
