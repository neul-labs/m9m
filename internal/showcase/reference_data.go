package showcase

// n8n reference benchmark numbers.
// These are documented reference values from n8n's Node.js-based architecture.
// Source: n8n community benchmarks and architecture documentation.
// n8n uses a single-threaded Node.js event loop for expression evaluation
// and workflow execution.

// ExpressionRef holds n8n reference ops/sec for expression types.
var ExpressionRef = map[string]int64{
	"Simple Math":       500,
	"String Functions":  400,
	"Array Functions":   350,
	"Conditional Logic": 450,
	"Complex Nested":    200,
	"Hashing":           300,
	"Variable Access":   500,
}

// WorkflowRef holds n8n reference throughput (workflows/sec).
var WorkflowRef = map[string]float64{
	"2-node (Set)":         100,
	"5-node (transform)":   40,
}

// MemoryRef holds n8n reference memory values in MB.
var MemoryRef = map[string]float64{
	"Baseline (idle)":    512,
	"Per 1K expressions": 8,
	"Per 1K workflows":   25,
}

// StartupRef holds n8n reference startup times in ms.
var StartupRef = map[string]float64{
	"Engine init":       3000,
	"Node registration": 500,
	"First execution":   200,
}

// ConcurrencyRef holds n8n reference ops/sec at concurrency levels.
// n8n is single-threaded so performance doesn't scale with concurrency.
var ConcurrencyRef = map[int]int64{
	1:   500,
	5:   500,
	10:  500,
	25:  500,
	50:  500,
	100: 500,
}

// ArchComparison holds architecture comparison data.
type ArchComparison struct {
	Feature string
	M9m     string
	N8n     string
}

// ArchData is the architecture comparison table.
var ArchData = []ArchComparison{
	{"Language", "Go 1.21+", "Node.js (TypeScript)"},
	{"Concurrency", "Goroutines (true parallel)", "Single-threaded event loop"},
	{"Binary", "Single static binary", "npm install + node_modules"},
	{"Container Size", "~300 MB", "~1.2 GB"},
	{"Startup Time", "<500 ms", "~3 seconds"},
	{"Memory (idle)", "~150 MB", "~512 MB"},
	{"Expression Engine", "Goja (compiled JS)", "vm2 / Node.js eval"},
	{"Scaling", "Horizontal + vertical", "Vertical only (queue mode)"},
}

// EnterpriseFeature holds an enterprise feature comparison entry.
type EnterpriseFeature struct {
	Feature string
	M9m     string
	N8n     string
}

// EnterpriseData lists enterprise features with support status.
var EnterpriseData = []EnterpriseFeature{
	{"Circuit Breaker", "Built-in", "Not available"},
	{"Dead Letter Queue", "Built-in", "Not available"},
	{"Prometheus Metrics", "Built-in", "Community plugin"},
	{"OpenTelemetry Tracing", "Built-in", "Not available"},
	{"AI Copilot (MCP)", "Built-in", "Not available"},
	{"Hot-Reload Plugins", "Built-in", "Restart required"},
	{"Workflow Versioning", "Built-in", "Enterprise only"},
	{"Multi-Workspace", "Built-in", "Enterprise only"},
	{"Audit Logging", "Built-in", "Enterprise only"},
	{"REST API", "Built-in", "Built-in"},
	{"Webhook Triggers", "Built-in", "Built-in"},
	{"Cron Scheduling", "Built-in", "Built-in"},
}

// ExpressionCategory holds expression function category info.
type ExpressionCategory struct {
	Category string
	Count    int
	Examples string
}

// ExpressionCategories lists expression function categories.
var ExpressionCategories = []ExpressionCategory{
	{"String", 20, "upper, lower, trim, split, replace, includes, startsWith"},
	{"Math", 15, "sum, min, max, average, round, ceil, floor, abs, multiply"},
	{"Date/Time", 12, "now, today, formatDate, dateAdd, dateDiff, moment"},
	{"Array", 10, "first, last, join, unique, compact, chunk, flatten"},
	{"Type Check", 8, "isEmpty, isEmail, isUrl, isNumeric, isDefined, isNull"},
	{"Hash/Crypto", 6, "md5, sha256, base64Encode, base64Decode, hmac"},
	{"Control", 5, "if, switch, default, ifEmpty, coalesce"},
}
