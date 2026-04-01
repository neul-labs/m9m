package showcase

import "github.com/neul-labs/m9m/internal/model"

// DemoWorkflow describes a demo with its workflow and input data.
type DemoWorkflow struct {
	Name        string
	Description string
	Category    string // "transform", "business", "expressions", "enterprise"
	Nodes       string // short summary of pipeline
	Workflow    *model.Workflow
	Input       []model.DataItem
}

// AllDemos returns all demo workflows.
func AllDemos() []DemoWorkflow {
	return []DemoWorkflow{
		buildOrderProcessingDemo(),
		buildCustomerSegmentationDemo(),
		buildMultiBranchRoutingDemo(),
		buildExpressionShowcaseDemo(),
		buildDataValidationDemo(),
		buildBatchProcessingDemo(),
	}
}

func buildOrderProcessingDemo() DemoWorkflow {
	wf := &model.Workflow{
		Name:   "E-Commerce Order Processing",
		Active: true,
		Nodes: []model.Node{
			{Name: "Start", Type: "n8n-nodes-base.start", Position: []int{100, 300}, Parameters: map[string]interface{}{}},
			{Name: "SetOrders", Type: "n8n-nodes-base.set", Position: []int{250, 300}, Parameters: map[string]interface{}{
				"assignments": []interface{}{
					map[string]interface{}{"name": "orderId", "value": "{{ 'ORD-' + $json.id }}"},
					map[string]interface{}{"name": "total", "value": "{{ multiply($json.price, $json.quantity) }}"},
					map[string]interface{}{"name": "customerName", "value": "{{ upper($json.customer) }}"},
				},
			}},
			{Name: "AddTax", Type: "n8n-nodes-base.set", Position: []int{400, 300}, Parameters: map[string]interface{}{
				"assignments": []interface{}{
					map[string]interface{}{"name": "tax", "value": "{{ round(multiply($json.total, 0.08), 2) }}"},
					map[string]interface{}{"name": "grandTotal", "value": "{{ round(multiply($json.total, 1.08), 2) }}"},
				},
			}},
			{Name: "HighValue", Type: "n8n-nodes-base.filter", Position: []int{550, 300}, Parameters: map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{"leftValue": "$json.grandTotal", "operator": "greaterThan", "rightValue": 50},
				},
			}},
		},
		Connections: map[string]model.Connections{
			"Start":     {Main: [][]model.Connection{{{Node: "SetOrders", Type: "main", Index: 0}}}},
			"SetOrders": {Main: [][]model.Connection{{{Node: "AddTax", Type: "main", Index: 0}}}},
			"AddTax":    {Main: [][]model.Connection{{{Node: "HighValue", Type: "main", Index: 0}}}},
		},
	}

	input := []model.DataItem{
		{JSON: map[string]interface{}{"id": "1001", "customer": "alice smith", "product": "Widget", "price": 29.99, "quantity": 3}},
		{JSON: map[string]interface{}{"id": "1002", "customer": "bob jones", "product": "Gadget", "price": 9.99, "quantity": 1}},
		{JSON: map[string]interface{}{"id": "1003", "customer": "carol white", "product": "Gizmo", "price": 49.99, "quantity": 2}},
	}

	return DemoWorkflow{
		Name:        "E-Commerce Order Processing",
		Description: "Calculates order totals with tax, filters high-value orders",
		Category:    "business",
		Nodes:       "Start -> Set -> Set -> Filter",
		Workflow:    wf,
		Input:       input,
	}
}

func buildCustomerSegmentationDemo() DemoWorkflow {
	wf := &model.Workflow{
		Name:   "Customer Segmentation",
		Active: true,
		Nodes: []model.Node{
			{Name: "Start", Type: "n8n-nodes-base.start", Position: []int{100, 300}, Parameters: map[string]interface{}{}},
			{Name: "Enrich", Type: "n8n-nodes-base.set", Position: []int{250, 300}, Parameters: map[string]interface{}{
				"assignments": []interface{}{
					map[string]interface{}{"name": "fullName", "value": "{{ upper($json.firstName) + ' ' + upper($json.lastName) }}"},
					map[string]interface{}{"name": "segment", "value": "{{ if($json.totalSpent > 1000, 'VIP', if($json.totalSpent > 500, 'Premium', 'Standard')) }}"},
				},
			}},
			{Name: "FilterActive", Type: "n8n-nodes-base.filter", Position: []int{400, 300}, Parameters: map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{"leftValue": "$json.active", "operator": "equals", "rightValue": true},
				},
			}},
			{Name: "FormatOutput", Type: "n8n-nodes-base.set", Position: []int{550, 300}, Parameters: map[string]interface{}{
				"assignments": []interface{}{
					map[string]interface{}{"name": "summary", "value": "{{ $json.fullName + ' [' + $json.segment + ']' }}"},
				},
			}},
		},
		Connections: map[string]model.Connections{
			"Start":        {Main: [][]model.Connection{{{Node: "Enrich", Type: "main", Index: 0}}}},
			"Enrich":       {Main: [][]model.Connection{{{Node: "FilterActive", Type: "main", Index: 0}}}},
			"FilterActive": {Main: [][]model.Connection{{{Node: "FormatOutput", Type: "main", Index: 0}}}},
		},
	}

	input := []model.DataItem{
		{JSON: map[string]interface{}{"firstName": "alice", "lastName": "smith", "totalSpent": 1500.0, "active": true}},
		{JSON: map[string]interface{}{"firstName": "bob", "lastName": "jones", "totalSpent": 200.0, "active": false}},
		{JSON: map[string]interface{}{"firstName": "carol", "lastName": "white", "totalSpent": 750.0, "active": true}},
	}

	return DemoWorkflow{
		Name:        "Customer Segmentation",
		Description: "Segments customers by spend level, filters active only",
		Category:    "business",
		Nodes:       "Start -> Set -> Filter -> Set",
		Workflow:    wf,
		Input:       input,
	}
}

func buildMultiBranchRoutingDemo() DemoWorkflow {
	wf := &model.Workflow{
		Name:   "Multi-Branch Routing",
		Active: true,
		Nodes: []model.Node{
			{Name: "Start", Type: "n8n-nodes-base.start", Position: []int{100, 300}, Parameters: map[string]interface{}{}},
			{Name: "Classify", Type: "n8n-nodes-base.set", Position: []int{250, 300}, Parameters: map[string]interface{}{
				"assignments": []interface{}{
					map[string]interface{}{"name": "priority", "value": "{{ if($json.severity == 'critical', 'P1', if($json.severity == 'high', 'P2', 'P3')) }}"},
					map[string]interface{}{"name": "label", "value": "{{ upper($json.severity) + ': ' + $json.title }}"},
				},
			}},
			{Name: "Route", Type: "n8n-nodes-base.switch", Position: []int{400, 300}, Parameters: map[string]interface{}{
				"rules": []interface{}{
					map[string]interface{}{"field": "$json.priority", "operation": "equal", "value": "P1"},
					map[string]interface{}{"field": "$json.priority", "operation": "equal", "value": "P2"},
				},
				"fallbackToLast": true,
			}},
			{Name: "P1-Critical", Type: "n8n-nodes-base.set", Position: []int{600, 150}, Parameters: map[string]interface{}{
				"assignments": []interface{}{
					map[string]interface{}{"name": "action", "value": "ESCALATE IMMEDIATELY"},
				},
			}},
			{Name: "P2-High", Type: "n8n-nodes-base.set", Position: []int{600, 300}, Parameters: map[string]interface{}{
				"assignments": []interface{}{
					map[string]interface{}{"name": "action", "value": "Queue for review"},
				},
			}},
			{Name: "P3-Normal", Type: "n8n-nodes-base.set", Position: []int{600, 450}, Parameters: map[string]interface{}{
				"assignments": []interface{}{
					map[string]interface{}{"name": "action", "value": "Add to backlog"},
				},
			}},
		},
		Connections: map[string]model.Connections{
			"Start":   {Main: [][]model.Connection{{{Node: "Classify", Type: "main", Index: 0}}}},
			"Classify": {Main: [][]model.Connection{{{Node: "Route", Type: "main", Index: 0}}}},
			"Route": {Main: [][]model.Connection{
				{{Node: "P1-Critical", Type: "main", Index: 0}},
				{{Node: "P2-High", Type: "main", Index: 0}},
				{{Node: "P3-Normal", Type: "main", Index: 0}},
			}},
		},
	}

	input := []model.DataItem{
		{JSON: map[string]interface{}{"title": "Server down", "severity": "critical"}},
		{JSON: map[string]interface{}{"title": "Slow queries", "severity": "high"}},
		{JSON: map[string]interface{}{"title": "Update docs", "severity": "low"}},
	}

	return DemoWorkflow{
		Name:        "Multi-Branch Routing",
		Description: "Routes items to different outputs based on priority via Switch node",
		Category:    "transform",
		Nodes:       "Start -> Set -> Switch -> Set(x3)",
		Workflow:    wf,
		Input:       input,
	}
}

func buildExpressionShowcaseDemo() DemoWorkflow {
	wf := &model.Workflow{
		Name:   "Expression Showcase",
		Active: true,
		Nodes: []model.Node{
			{Name: "Start", Type: "n8n-nodes-base.start", Position: []int{100, 300}, Parameters: map[string]interface{}{}},
			{Name: "StringOps", Type: "n8n-nodes-base.set", Position: []int{300, 300}, Parameters: map[string]interface{}{
				"assignments": []interface{}{
					map[string]interface{}{"name": "upper", "value": "{{ upper($json.name) }}"},
					map[string]interface{}{"name": "lower", "value": "{{ lower($json.name) }}"},
					map[string]interface{}{"name": "trimmed", "value": "{{ trim($json.padded) }}"},
					map[string]interface{}{"name": "replaced", "value": "{{ replace($json.name, 'World', 'Go') }}"},
					map[string]interface{}{"name": "length", "value": "{{ length($json.name) }}"},
				},
			}},
			{Name: "MathOps", Type: "n8n-nodes-base.set", Position: []int{500, 300}, Parameters: map[string]interface{}{
				"assignments": []interface{}{
					map[string]interface{}{"name": "sum", "value": "{{ $json.a + $json.b }}"},
					map[string]interface{}{"name": "product", "value": "{{ multiply($json.a, $json.b) }}"},
					map[string]interface{}{"name": "rounded", "value": "{{ round(3.14159, 2) }}"},
					map[string]interface{}{"name": "abs", "value": "{{ abs(-42) }}"},
					map[string]interface{}{"name": "max", "value": "{{ max($json.a, $json.b) }}"},
					map[string]interface{}{"name": "hash", "value": "{{ md5($json.name) }}"},
				},
			}},
		},
		Connections: map[string]model.Connections{
			"Start":     {Main: [][]model.Connection{{{Node: "StringOps", Type: "main", Index: 0}}}},
			"StringOps": {Main: [][]model.Connection{{{Node: "MathOps", Type: "main", Index: 0}}}},
		},
	}

	input := []model.DataItem{
		{JSON: map[string]interface{}{
			"name":   "Hello World",
			"padded": "  spaces  ",
			"a":      10,
			"b":      25,
		}},
	}

	return DemoWorkflow{
		Name:        "Expression Showcase",
		Description: "Demonstrates 15+ expression types: string, math, hash, array, date",
		Category:    "expressions",
		Nodes:       "Start -> Set -> Set",
		Workflow:    wf,
		Input:       input,
	}
}

func buildDataValidationDemo() DemoWorkflow {
	wf := &model.Workflow{
		Name:   "Data Validation Pipeline",
		Active: true,
		Nodes: []model.Node{
			{Name: "Start", Type: "n8n-nodes-base.start", Position: []int{100, 300}, Parameters: map[string]interface{}{}},
			{Name: "Validate", Type: "n8n-nodes-base.set", Position: []int{250, 300}, Parameters: map[string]interface{}{
				"assignments": []interface{}{
					map[string]interface{}{"name": "emailValid", "value": "{{ isEmail($json.email) }}"},
					map[string]interface{}{"name": "urlValid", "value": "{{ isUrl($json.website) }}"},
					map[string]interface{}{"name": "namePresent", "value": "{{ isNotEmpty($json.name) }}"},
				},
			}},
			{Name: "Enrich", Type: "n8n-nodes-base.set", Position: []int{400, 300}, Parameters: map[string]interface{}{
				"assignments": []interface{}{
					map[string]interface{}{"name": "domain", "value": "{{ last(split($json.email, '@')) }}"},
					map[string]interface{}{"name": "displayName", "value": "{{ upper(first(split($json.name, ' '))) }}"},
				},
			}},
			{Name: "ValidOnly", Type: "n8n-nodes-base.filter", Position: []int{550, 300}, Parameters: map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{"leftValue": "$json.emailValid", "operator": "equals", "rightValue": true},
				},
			}},
		},
		Connections: map[string]model.Connections{
			"Start":    {Main: [][]model.Connection{{{Node: "Validate", Type: "main", Index: 0}}}},
			"Validate": {Main: [][]model.Connection{{{Node: "Enrich", Type: "main", Index: 0}}}},
			"Enrich":   {Main: [][]model.Connection{{{Node: "ValidOnly", Type: "main", Index: 0}}}},
		},
	}

	input := []model.DataItem{
		{JSON: map[string]interface{}{"name": "Alice Smith", "email": "alice@example.com", "website": "https://example.com"}},
		{JSON: map[string]interface{}{"name": "Bob Jones", "email": "not-an-email", "website": "invalid"}},
		{JSON: map[string]interface{}{"name": "Carol White", "email": "carol@test.org", "website": "https://test.org"}},
	}

	return DemoWorkflow{
		Name:        "Data Validation Pipeline",
		Description: "Validates emails, URLs, enriches data, filters valid entries",
		Category:    "transform",
		Nodes:       "Start -> Set -> Set -> Filter",
		Workflow:    wf,
		Input:       input,
	}
}

func buildBatchProcessingDemo() DemoWorkflow {
	wf := &model.Workflow{
		Name:   "Batch Processing",
		Active: true,
		Nodes: []model.Node{
			{Name: "Start", Type: "n8n-nodes-base.start", Position: []int{100, 300}, Parameters: map[string]interface{}{}},
			{Name: "BatchSplit", Type: "n8n-nodes-base.splitInBatches", Position: []int{250, 300}, Parameters: map[string]interface{}{
				"batchSize": 2,
			}},
			{Name: "Process", Type: "n8n-nodes-base.set", Position: []int{400, 300}, Parameters: map[string]interface{}{
				"assignments": []interface{}{
					map[string]interface{}{"name": "processed", "value": "true"},
					map[string]interface{}{"name": "label", "value": "{{ upper($json.item) + ' [DONE]' }}"},
				},
			}},
		},
		Connections: map[string]model.Connections{
			"Start":      {Main: [][]model.Connection{{{Node: "BatchSplit", Type: "main", Index: 0}}}},
			"BatchSplit": {Main: [][]model.Connection{{{Node: "Process", Type: "main", Index: 0}}}},
		},
	}

	input := []model.DataItem{
		{JSON: map[string]interface{}{"item": "alpha", "seq": 1}},
		{JSON: map[string]interface{}{"item": "beta", "seq": 2}},
		{JSON: map[string]interface{}{"item": "gamma", "seq": 3}},
		{JSON: map[string]interface{}{"item": "delta", "seq": 4}},
		{JSON: map[string]interface{}{"item": "epsilon", "seq": 5}},
	}

	return DemoWorkflow{
		Name:        "Batch Processing",
		Description: "Splits items into batches of 2 for iterative processing",
		Category:    "enterprise",
		Nodes:       "Start -> SplitInBatches -> Set",
		Workflow:    wf,
		Input:       input,
	}
}
