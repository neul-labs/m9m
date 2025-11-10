package connections

import (
	"testing"
	"github.com/dipankar/n8n-go/internal/model"
)

func TestConnectionRouterCreation(t *testing.T) {
	router := NewConnectionRouter()
	if router == nil {
		t.Fatal("Expected router to be created, got nil")
	}
}

func TestRouteDataWithNilWorkflow(t *testing.T) {
	router := NewConnectionRouter()
	
	_, err := router.RouteData("source", nil, nil)
	if err == nil {
		t.Error("Expected error with nil workflow, got nil")
	}
}

func TestRouteDataWithNoConnections(t *testing.T) {
	router := NewConnectionRouter()
	
	workflow := &model.Workflow{
		Connections: make(map[string]model.Connections),
	}
	
	data := []model.DataItem{
		{JSON: map[string]interface{}{"test": "data"}},
	}
	
	routedData, err := router.RouteData("source-node", workflow, data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(routedData) != 0 {
		t.Errorf("Expected empty routed data, got %d entries", len(routedData))
	}
}

func TestRouteDataWithValidConnections(t *testing.T) {
	router := NewConnectionRouter()
	
	workflow := &model.Workflow{
		Nodes: []model.Node{
			{Name: "source", Type: "type1"},
			{Name: "target", Type: "type2"},
		},
		Connections: map[string]model.Connections{
			"source": {
				Main: [][]model.Connection{
					{
						{
							Node:  "target",
							Type:  "main",
							Index: 0,
						},
					},
				},
			},
		},
	}
	
	data := []model.DataItem{
		{JSON: map[string]interface{}{"test": "data"}},
	}
	
	routedData, err := router.RouteData("source", workflow, data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(routedData) != 1 {
		t.Errorf("Expected 1 routed data entry, got %d", len(routedData))
	}
	
	targetData, exists := routedData["target"]
	if !exists {
		t.Error("Expected data routed to 'target' node")
	}
	
	if len(targetData) != len(data) {
		t.Errorf("Expected %d data items, got %d", len(data), len(targetData))
	}
}

func TestGetConnectionsForNonExistentNode(t *testing.T) {
	router := NewConnectionRouter()
	
	workflow := &model.Workflow{
		Connections: make(map[string]model.Connections),
	}
	
	connections := router.GetConnections("non-existent", workflow)
	if connections != nil {
		t.Error("Expected nil connections for non-existent node, got connections")
	}
}

func TestGetConnectionsForNodeWithoutConnections(t *testing.T) {
	router := NewConnectionRouter()
	
	workflow := &model.Workflow{
		Connections: map[string]model.Connections{
			"node1": {}, // Empty connections
		},
	}
	
	connections := router.GetConnections("node1", workflow)
	if connections == nil {
		t.Fatal("Expected empty connections, got nil")
	}
	
	// Should be empty but not nil
	if len(connections.Main) != 0 {
		t.Errorf("Expected empty main connections, got %d", len(connections.Main))
	}
}

func TestValidateConnectionsWithNilWorkflow(t *testing.T) {
	router := NewConnectionRouter()
	
	err := router.ValidateConnections(nil)
	if err == nil {
		t.Error("Expected error for nil workflow, got nil")
	}
}

func TestValidateConnectionsWithValidWorkflow(t *testing.T) {
	router := NewConnectionRouter()
	
	workflow := &model.Workflow{
		Nodes: []model.Node{
			{Name: "node1", Type: "type1"},
			{Name: "node2", Type: "type2"},
		},
		Connections: map[string]model.Connections{
			"node1": {
				Main: [][]model.Connection{
					{
						{
							Node:  "node2",
							Type:  "main",
							Index: 0,
						},
					},
				},
			},
		},
	}
	
	err := router.ValidateConnections(workflow)
	if err != nil {
		t.Errorf("Expected valid connections, got error: %v", err)
	}
}

func TestValidateConnectionsWithNonExistentSourceNode(t *testing.T) {
	router := NewConnectionRouter()
	
	workflow := &model.Workflow{
		Nodes: []model.Node{
			{Name: "node2", Type: "type2"},
		},
		Connections: map[string]model.Connections{
			"node1": { // node1 doesn't exist
				Main: [][]model.Connection{
					{
						{
							Node:  "node2",
							Type:  "main",
							Index: 0,
						},
					},
				},
			},
		},
	}
	
	err := router.ValidateConnections(workflow)
	if err == nil {
		t.Error("Expected error for non-existent source node, got nil")
	}
}

func TestValidateConnectionsWithNonExistentTargetNode(t *testing.T) {
	router := NewConnectionRouter()
	
	workflow := &model.Workflow{
		Nodes: []model.Node{
			{Name: "node1", Type: "type1"},
		},
		Connections: map[string]model.Connections{
			"node1": {
				Main: [][]model.Connection{
					{
						{
							Node:  "node2",
							Type:  "main",
							Index: 0, // node2 doesn't exist
						},
					},
				},
			},
		},
	}
	
	err := router.ValidateConnections(workflow)
	if err == nil {
		t.Error("Expected error for non-existent target node, got nil")
	}
}

func TestGetExecutionOrderWithNilWorkflow(t *testing.T) {
	router := NewConnectionRouter()
	
	_, err := router.GetExecutionOrder(nil)
	if err == nil {
		t.Error("Expected error for nil workflow, got nil")
	}
}

func TestGetExecutionOrderWithNoNodes(t *testing.T) {
	router := NewConnectionRouter()
	
	workflow := &model.Workflow{
		Nodes: []model.Node{}, // No nodes
	}
	
	order, err := router.GetExecutionOrder(workflow)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(order) != 0 {
		t.Errorf("Expected empty execution order, got %d nodes", len(order))
	}
}

func TestGetExecutionOrderWithSingleNode(t *testing.T) {
	router := NewConnectionRouter()
	
	workflow := &model.Workflow{
		Nodes: []model.Node{
			{Name: "node1", Type: "type1"},
		},
	}
	
	order, err := router.GetExecutionOrder(workflow)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(order) != 1 {
		t.Fatalf("Expected 1 node in execution order, got %d", len(order))
	}
	
	if order[0] != "node1" {
		t.Errorf("Expected 'node1', got '%s'", order[0])
	}
}

func TestGetExecutionOrderWithLinearDependencies(t *testing.T) {
	router := NewConnectionRouter()
	
	workflow := &model.Workflow{
		Nodes: []model.Node{
			{Name: "node1", Type: "type1"},
			{Name: "node2", Type: "type2"},
			{Name: "node3", Type: "type3"},
		},
		Connections: map[string]model.Connections{
			"node1": {
				Main: [][]model.Connection{
					{
						{
							Node:  "node2",
							Type:  "main",
							Index: 0,
						},
					},
				},
			},
			"node2": {
				Main: [][]model.Connection{
					{
						{
							Node:  "node3",
							Type:  "main",
							Index: 0,
						},
					},
				},
			},
		},
	}
	
	order, err := router.GetExecutionOrder(workflow)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(order) != 3 {
		t.Fatalf("Expected 3 nodes in execution order, got %d", len(order))
	}
	
	// Check that dependencies are respected
	// node1 -> node2 -> node3
	node1Index := -1
	node2Index := -1
	node3Index := -1
	
	for i, node := range order {
		switch node {
		case "node1":
			node1Index = i
		case "node2":
			node2Index = i
		case "node3":
			node3Index = i
		}
	}
	
	if node1Index == -1 || node2Index == -1 || node3Index == -1 {
		t.Fatal("Not all nodes found in execution order")
	}
	
	// node1 should come before node2
	if node1Index >= node2Index {
		t.Error("Expected node1 to come before node2")
	}
	
	// node2 should come before node3
	if node2Index >= node3Index {
		t.Error("Expected node2 to come before node3")
	}
}

func TestHasCyclesWithNilWorkflow(t *testing.T) {
	router := NewConnectionRouter()
	
	_, err := router.HasCycles(nil)
	if err == nil {
		t.Error("Expected error for nil workflow, got nil")
	}
}

func TestHasCyclesWithAcyclicWorkflow(t *testing.T) {
	router := NewConnectionRouter()
	
	workflow := &model.Workflow{
		Nodes: []model.Node{
			{Name: "node1", Type: "type1"},
			{Name: "node2", Type: "type2"},
		},
		Connections: map[string]model.Connections{
			"node1": {
				Main: [][]model.Connection{
					{
						{
							Node:  "node2",
							Type:  "main",
							Index: 0,
						},
					},
				},
			},
		},
	}
	
	hasCycles, err := router.HasCycles(workflow)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if hasCycles {
		t.Error("Expected acyclic workflow, but cycles detected")
	}
}