package showcase

import (
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/neul-labs/m9m/internal/engine"
	"github.com/neul-labs/m9m/internal/expressions"
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/core"
	"github.com/neul-labs/m9m/internal/nodes/transform"
)

// BenchmarkOptions controls benchmark execution.
type BenchmarkOptions struct {
	Quick    bool
	Category string // "expressions", "workflows", "concurrency", "memory", "startup", or ""
}

// RunBenchmark executes benchmarks and returns a report.
func RunBenchmark(opts BenchmarkOptions, registerNodes func(engine.WorkflowEngine)) *BenchmarkReport {
	report := &BenchmarkReport{
		System: GetSystemInfo(),
	}

	categories := map[string]bool{
		"expressions": opts.Category == "" || opts.Category == "expressions",
		"workflows":   opts.Category == "" || opts.Category == "workflows",
		"concurrency": opts.Category == "" || opts.Category == "concurrency",
		"memory":      opts.Category == "" || opts.Category == "memory",
		"startup":     opts.Category == "" || opts.Category == "startup",
	}

	if categories["expressions"] {
		report.Categories = append(report.Categories, benchmarkExpressions(opts))
	}
	if categories["workflows"] {
		report.Categories = append(report.Categories, benchmarkWorkflows(opts, registerNodes))
	}
	if categories["concurrency"] {
		report.Categories = append(report.Categories, benchmarkConcurrency(opts))
	}
	if categories["memory"] {
		report.Categories = append(report.Categories, benchmarkMemory(opts, registerNodes))
	}
	if categories["startup"] {
		report.Categories = append(report.Categories, benchmarkStartup(opts, registerNodes))
	}

	return report
}

func benchmarkExpressions(opts BenchmarkOptions) BenchmarkCategory {
	evaluator := expressions.NewGojaExpressionEvaluator(expressions.DefaultEvaluatorConfig())

	ctx := &expressions.ExpressionContext{
		ActiveNodeName: "bench",
		Mode:           expressions.ModeManual,
		AdditionalKeys: &expressions.AdditionalKeys{ExecutionId: "bench"},
		Workflow:       &model.Workflow{ID: "bench", Name: "Bench"},
		ConnectionInputData: []model.DataItem{
			{JSON: map[string]interface{}{
				"name": "John Doe", "age": 25, "email": "john@example.com",
				"items": []interface{}{"a", "b", "c"},
			}},
		},
	}

	tests := []struct {
		name string
		expr string
	}{
		{"Simple Math", "{{ 2 + 3 * 4 - 1 }}"},
		{"String Functions", "{{ upper(trim('  hello world  ')) }}"},
		{"Array Functions", "{{ join(split('a,b,c', ','), ' | ') }}"},
		{"Conditional Logic", "{{ if(5 > 3, 'yes', 'no') }}"},
		{"Complex Nested", "{{ upper(first(split(trim('  john doe  '), ' '))) }}"},
		{"Hashing", "{{ md5('benchmark-test-string') }}"},
		{"Variable Access", "{{ $json.name }}"},
	}

	iters := 5000
	if opts.Quick {
		iters = 500
	}

	var results []BenchmarkResult
	for _, tc := range tests {
		// Warmup
		for i := 0; i < 100; i++ {
			evaluator.EvaluateExpression(tc.expr, ctx)
		}

		// Measure
		opsPerSec := measureOpsPerSec(iters, func() {
			evaluator.EvaluateExpression(tc.expr, ctx)
		})

		ref := int64(500)
		if r, ok := ExpressionRef[tc.name]; ok {
			ref = r
		}
		speedup := float64(opsPerSec) / float64(ref)

		results = append(results, BenchmarkResult{
			Name:    tc.name,
			Value:   float64(opsPerSec),
			Unit:    "ops/sec",
			Ref:     float64(ref),
			Speedup: speedup,
		})
	}

	return BenchmarkCategory{Name: "Expression Performance", Results: results}
}

func benchmarkWorkflows(opts BenchmarkOptions, registerNodes func(engine.WorkflowEngine)) BenchmarkCategory {
	// 2-node workflow: Start -> Set
	twoNodeWf := &model.Workflow{
		Name:   "2-node bench",
		Active: true,
		Nodes: []model.Node{
			{Name: "Start", Type: "n8n-nodes-base.start", Position: []int{250, 300}, Parameters: map[string]interface{}{}},
			{Name: "Set", Type: "n8n-nodes-base.set", Position: []int{450, 300}, Parameters: map[string]interface{}{
				"assignments": []interface{}{
					map[string]interface{}{"name": "processed", "value": "{{ upper($json.name) }}"},
				},
			}},
		},
		Connections: map[string]model.Connections{
			"Start": {Main: [][]model.Connection{{{Node: "Set", Type: "main", Index: 0}}}},
		},
	}

	// 5-node workflow: Start -> Set -> Set -> Filter -> Set
	fiveNodeWf := &model.Workflow{
		Name:   "5-node bench",
		Active: true,
		Nodes: []model.Node{
			{Name: "Start", Type: "n8n-nodes-base.start", Position: []int{100, 300}, Parameters: map[string]interface{}{}},
			{Name: "Set1", Type: "n8n-nodes-base.set", Position: []int{250, 300}, Parameters: map[string]interface{}{
				"assignments": []interface{}{
					map[string]interface{}{"name": "step", "value": "1"},
				},
			}},
			{Name: "Set2", Type: "n8n-nodes-base.set", Position: []int{400, 300}, Parameters: map[string]interface{}{
				"assignments": []interface{}{
					map[string]interface{}{"name": "step", "value": "2"},
				},
			}},
			{Name: "Filter", Type: "n8n-nodes-base.filter", Position: []int{550, 300}, Parameters: map[string]interface{}{
				"conditions": map[string]interface{}{
					"string": []interface{}{
						map[string]interface{}{"value1": "={{ $json.step }}", "operation": "isNotEmpty"},
					},
				},
			}},
			{Name: "Set3", Type: "n8n-nodes-base.set", Position: []int{700, 300}, Parameters: map[string]interface{}{
				"assignments": []interface{}{
					map[string]interface{}{"name": "done", "value": "true"},
				},
			}},
		},
		Connections: map[string]model.Connections{
			"Start":  {Main: [][]model.Connection{{{Node: "Set1", Type: "main", Index: 0}}}},
			"Set1":   {Main: [][]model.Connection{{{Node: "Set2", Type: "main", Index: 0}}}},
			"Set2":   {Main: [][]model.Connection{{{Node: "Filter", Type: "main", Index: 0}}}},
			"Filter": {Main: [][]model.Connection{{{Node: "Set3", Type: "main", Index: 0}}}},
		},
	}

	testData := []model.DataItem{
		{JSON: map[string]interface{}{"name": "John", "age": 25}},
	}

	iters := 500
	if opts.Quick {
		iters = 50
	}

	var results []BenchmarkResult

	for _, tc := range []struct {
		name string
		wf   *model.Workflow
	}{
		{"2-node (Set)", twoNodeWf},
		{"5-node (transform)", fiveNodeWf},
	} {
		eng := engine.NewWorkflowEngine()
		registerNodes(eng)

		// Warmup
		for i := 0; i < 10; i++ {
			eng.ExecuteWorkflow(tc.wf, testData)
		}

		opsPerSec := measureOpsPerSec(iters, func() {
			eng.ExecuteWorkflow(tc.wf, testData)
		})

		ref := float64(100)
		if r, ok := WorkflowRef[tc.name]; ok {
			ref = r
		}
		speedup := float64(opsPerSec) / ref

		results = append(results, BenchmarkResult{
			Name:    tc.name,
			Value:   float64(opsPerSec),
			Unit:    "wf/sec",
			Ref:     ref,
			Speedup: speedup,
		})
	}

	return BenchmarkCategory{Name: "Workflow Throughput", Results: results}
}

func benchmarkConcurrency(opts BenchmarkOptions) BenchmarkCategory {
	evaluator := expressions.NewGojaExpressionEvaluator(expressions.DefaultEvaluatorConfig())

	ctx := &expressions.ExpressionContext{
		ActiveNodeName: "bench",
		Mode:           expressions.ModeManual,
		AdditionalKeys: &expressions.AdditionalKeys{ExecutionId: "bench"},
		ConnectionInputData: []model.DataItem{
			{JSON: map[string]interface{}{"value": 42}},
		},
	}

	expr := "{{ multiply($json.value, 2) + 8 }}"

	levels := []int{1, 5, 10, 25, 50, 100}
	if opts.Quick {
		levels = []int{1, 5, 10, 25}
	}

	opsPerLevel := 1000
	if opts.Quick {
		opsPerLevel = 200
	}

	var results []BenchmarkResult
	for _, conc := range levels {
		start := time.Now()
		var wg sync.WaitGroup
		wg.Add(conc)
		for i := 0; i < conc; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < opsPerLevel; j++ {
					evaluator.EvaluateExpression(expr, ctx)
				}
			}()
		}
		wg.Wait()
		elapsed := time.Since(start)

		totalOps := int64(conc * opsPerLevel)
		opsPerSec := float64(totalOps) / elapsed.Seconds()

		ref := int64(500)
		if r, ok := ConcurrencyRef[conc]; ok {
			ref = r
		}
		speedup := opsPerSec / float64(ref)

		results = append(results, BenchmarkResult{
			Name:    fmt.Sprintf("%d", conc),
			Value:   opsPerSec,
			Unit:    "ops/sec",
			Ref:     float64(ref),
			Speedup: speedup,
		})
	}

	return BenchmarkCategory{Name: "Concurrency Scaling", Results: results}
}

func benchmarkMemory(opts BenchmarkOptions, registerNodes func(engine.WorkflowEngine)) BenchmarkCategory {
	var results []BenchmarkResult

	// Baseline
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	baselineMB := float64(m1.Alloc) / (1024 * 1024)

	results = append(results, BenchmarkResult{
		Name:    "Baseline (idle)",
		Value:   baselineMB,
		Unit:    "MB",
		Ref:     MemoryRef["Baseline (idle)"],
		Savings: (1 - baselineMB/MemoryRef["Baseline (idle)"]) * 100,
	})

	// Per 1K expressions
	evaluator := expressions.NewGojaExpressionEvaluator(expressions.DefaultEvaluatorConfig())
	ctx := &expressions.ExpressionContext{
		ActiveNodeName: "bench",
		Mode:           expressions.ModeManual,
		AdditionalKeys: &expressions.AdditionalKeys{ExecutionId: "bench"},
		ConnectionInputData: []model.DataItem{
			{JSON: map[string]interface{}{"value": 42}},
		},
	}
	exprCount := 1000
	if opts.Quick {
		exprCount = 200
	}

	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	before := m2.TotalAlloc

	for i := 0; i < exprCount; i++ {
		evaluator.EvaluateExpression("{{ $json.value + 1 }}", ctx)
	}
	runtime.GC()
	var m3 runtime.MemStats
	runtime.ReadMemStats(&m3)
	exprDeltaMB := float64(m3.TotalAlloc-before) / (1024 * 1024)

	results = append(results, BenchmarkResult{
		Name:    "Per 1K expressions",
		Value:   exprDeltaMB,
		Unit:    "MB",
		Ref:     MemoryRef["Per 1K expressions"],
		Savings: (1 - exprDeltaMB/MemoryRef["Per 1K expressions"]) * 100,
	})

	// Per 1K workflows
	eng := engine.NewWorkflowEngine()
	registerNodes(eng)
	wf := &model.Workflow{
		Name:   "mem-bench",
		Active: true,
		Nodes: []model.Node{
			{Name: "Start", Type: "n8n-nodes-base.start", Position: []int{250, 300}, Parameters: map[string]interface{}{}},
			{Name: "Set", Type: "n8n-nodes-base.set", Position: []int{450, 300}, Parameters: map[string]interface{}{
				"assignments": []interface{}{
					map[string]interface{}{"name": "x", "value": "1"},
				},
			}},
		},
		Connections: map[string]model.Connections{
			"Start": {Main: [][]model.Connection{{{Node: "Set", Type: "main", Index: 0}}}},
		},
	}
	testData := []model.DataItem{{JSON: map[string]interface{}{"v": 1}}}

	wfCount := 1000
	if opts.Quick {
		wfCount = 200
	}

	runtime.GC()
	var m4 runtime.MemStats
	runtime.ReadMemStats(&m4)
	before = m4.TotalAlloc

	for i := 0; i < wfCount; i++ {
		eng.ExecuteWorkflow(wf, testData)
	}
	runtime.GC()
	var m5 runtime.MemStats
	runtime.ReadMemStats(&m5)
	wfDeltaMB := float64(m5.TotalAlloc-before) / (1024 * 1024)

	results = append(results, BenchmarkResult{
		Name:    "Per 1K workflows",
		Value:   wfDeltaMB,
		Unit:    "MB",
		Ref:     MemoryRef["Per 1K workflows"],
		Savings: (1 - wfDeltaMB/MemoryRef["Per 1K workflows"]) * 100,
	})

	return BenchmarkCategory{Name: "Memory Usage", Results: results}
}

func benchmarkStartup(opts BenchmarkOptions, registerNodes func(engine.WorkflowEngine)) BenchmarkCategory {
	var results []BenchmarkResult

	// Engine init
	iters := 100
	if opts.Quick {
		iters = 20
	}
	initMs := measureAvgMs(iters, func() {
		engine.NewWorkflowEngine()
	})
	results = append(results, BenchmarkResult{
		Name:    "Engine init",
		Value:   initMs,
		Unit:    "ms",
		Ref:     StartupRef["Engine init"],
		Speedup: safeSpeedup(StartupRef["Engine init"], initMs),
	})

	// Node registration
	regMs := measureAvgMs(iters, func() {
		eng := engine.NewWorkflowEngine()
		registerNodes(eng)
	})
	// Subtract engine init to get just registration time
	regOnlyMs := regMs - initMs
	if regOnlyMs < 0.01 {
		regOnlyMs = 0.01
	}
	results = append(results, BenchmarkResult{
		Name:    "Node registration",
		Value:   regOnlyMs,
		Unit:    "ms",
		Ref:     StartupRef["Node registration"],
		Speedup: safeSpeedup(StartupRef["Node registration"], regOnlyMs),
	})

	// First execution
	eng := engine.NewWorkflowEngine()
	eng.RegisterNodeExecutor("n8n-nodes-base.start", core.NewStartNode())
	eng.RegisterNodeExecutor("n8n-nodes-base.set", transform.NewSetNode())

	wf := &model.Workflow{
		Name:   "startup-bench",
		Active: true,
		Nodes: []model.Node{
			{Name: "Start", Type: "n8n-nodes-base.start", Position: []int{250, 300}, Parameters: map[string]interface{}{}},
			{Name: "Set", Type: "n8n-nodes-base.set", Position: []int{450, 300}, Parameters: map[string]interface{}{
				"assignments": []interface{}{
					map[string]interface{}{"name": "x", "value": "1"},
				},
			}},
		},
		Connections: map[string]model.Connections{
			"Start": {Main: [][]model.Connection{{{Node: "Set", Type: "main", Index: 0}}}},
		},
	}
	testData := []model.DataItem{{JSON: map[string]interface{}{"v": 1}}}

	execMs := measureAvgMs(iters, func() {
		eng.ExecuteWorkflow(wf, testData)
	})
	results = append(results, BenchmarkResult{
		Name:    "First execution",
		Value:   execMs,
		Unit:    "ms",
		Ref:     StartupRef["First execution"],
		Speedup: safeSpeedup(StartupRef["First execution"], execMs),
	})

	return BenchmarkCategory{Name: "Startup Time", Results: results}
}

// measureOpsPerSec runs fn iters times and returns median ops/sec.
func measureOpsPerSec(iters int, fn func()) int64 {
	// Run in batches of 100 to get multiple samples
	batchSize := 100
	if iters < batchSize {
		batchSize = iters
	}
	batches := iters / batchSize
	if batches < 1 {
		batches = 1
	}

	var samples []float64
	for b := 0; b < batches; b++ {
		start := time.Now()
		for i := 0; i < batchSize; i++ {
			fn()
		}
		elapsed := time.Since(start)
		samples = append(samples, float64(batchSize)/elapsed.Seconds())
	}

	sort.Float64s(samples)
	// Return median
	return int64(samples[len(samples)/2])
}

// measureAvgMs runs fn iters times and returns average duration in ms.
func measureAvgMs(iters int, fn func()) float64 {
	start := time.Now()
	for i := 0; i < iters; i++ {
		fn()
	}
	elapsed := time.Since(start)
	return float64(elapsed.Nanoseconds()) / float64(iters) / 1e6
}

// safeSpeedup computes ref/value, capping at a max to avoid Inf in JSON.
func safeSpeedup(ref, value float64) float64 {
	if value <= 0 {
		return 99999
	}
	s := ref / value
	if s > 99999 {
		return 99999
	}
	return s
}

