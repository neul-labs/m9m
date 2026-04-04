package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/neul-labs/m9m/internal/model"
)

// Template represents a workflow template.
type Template struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Workflow    *model.Workflow        `json:"workflow"`
	Tags        []string               `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
}

var builtInTemplates = []Template{
	{
		ID:          "http-webhook",
		Name:        "HTTP Webhook",
		Description: "Receive HTTP requests and process them",
		Category:    "trigger",
		Tags:        []string{"webhook", "http", "api"},
		Workflow: &model.Workflow{
			Name:   "HTTP Webhook",
			Active: false,
			Nodes: []model.Node{
				{
					ID:       "webhook-1",
					Name:     "Webhook",
					Type:     "n8n-nodes-base.webhook",
					Position: []int{250, 300},
					Parameters: map[string]interface{}{
						"httpMethod": "POST",
						"path":       "webhook",
					},
				},
			},
			Connections: make(map[string]model.Connections),
		},
	},
	{
		ID:          "scheduled-task",
		Name:        "Scheduled Task",
		Description: "Run tasks on a schedule using cron expressions",
		Category:    "trigger",
		Tags:        []string{"schedule", "cron", "timer"},
		Workflow: &model.Workflow{
			Name:   "Scheduled Task",
			Active: false,
			Nodes: []model.Node{
				{
					ID:       "cron-1",
					Name:     "Schedule Trigger",
					Type:     "n8n-nodes-base.scheduleTrigger",
					Position: []int{250, 300},
					Parameters: map[string]interface{}{
						"rule": map[string]interface{}{
							"interval": []map[string]interface{}{
								{"field": "hours", "hoursInterval": 1},
							},
						},
					},
				},
			},
			Connections: make(map[string]model.Connections),
		},
	},
	{
		ID:          "http-request",
		Name:        "HTTP Request",
		Description: "Make HTTP requests to external APIs",
		Category:    "action",
		Tags:        []string{"http", "api", "request"},
		Workflow: &model.Workflow{
			Name:   "HTTP Request",
			Active: false,
			Nodes: []model.Node{
				{
					ID:       "trigger-1",
					Name:     "Manual Trigger",
					Type:     "n8n-nodes-base.manualTrigger",
					Position: []int{250, 300},
				},
				{
					ID:       "http-1",
					Name:     "HTTP Request",
					Type:     "n8n-nodes-base.httpRequest",
					Position: []int{450, 300},
					Parameters: map[string]interface{}{
						"method": "GET",
						"url":    "https://api.example.com",
					},
				},
			},
			Connections: map[string]model.Connections{
				"Manual Trigger": {
					Main: [][]model.Connection{
						{{Node: "HTTP Request", Type: "main", Index: 0}},
					},
				},
			},
		},
	},
	{
		ID:          "data-transform",
		Name:        "Data Transformation",
		Description: "Transform and manipulate data",
		Category:    "transform",
		Tags:        []string{"transform", "data", "json"},
		Workflow: &model.Workflow{
			Name:   "Data Transformation",
			Active: false,
			Nodes: []model.Node{
				{
					ID:       "trigger-1",
					Name:     "Manual Trigger",
					Type:     "n8n-nodes-base.manualTrigger",
					Position: []int{250, 300},
				},
				{
					ID:       "set-1",
					Name:     "Set Data",
					Type:     "n8n-nodes-base.set",
					Position: []int{450, 300},
					Parameters: map[string]interface{}{
						"values": map[string]interface{}{
							"string": []map[string]interface{}{
								{"name": "example", "value": "Hello World"},
							},
						},
					},
				},
			},
			Connections: map[string]model.Connections{
				"Manual Trigger": {
					Main: [][]model.Connection{
						{{Node: "Set Data", Type: "main", Index: 0}},
					},
				},
			},
		},
	},
}

func (s *APIServer) ListTemplates(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")

	templates := make([]Template, 0)
	for _, template := range builtInTemplates {
		if category == "" || template.Category == category {
			templates = append(templates, template)
		}
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"data":  templates,
		"total": len(templates),
	})
}

func (s *APIServer) GetTemplate(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	for _, template := range builtInTemplates {
		if template.ID == id {
			s.sendJSON(w, http.StatusOK, template)
			return
		}
	}

	s.sendError(w, http.StatusNotFound, "Template not found", nil)
}

func (s *APIServer) ApplyTemplate(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var template *Template
	for _, current := range builtInTemplates {
		if current.ID == id {
			template = &current
			break
		}
	}

	if template == nil {
		s.sendError(w, http.StatusNotFound, "Template not found", nil)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}

	workflow := &model.Workflow{
		Name:        template.Workflow.Name,
		Active:      false,
		Nodes:       make([]model.Node, len(template.Workflow.Nodes)),
		Connections: make(map[string]model.Connections),
		Settings:    template.Workflow.Settings,
		Tags:        template.Tags,
	}

	if req.Name != "" {
		workflow.Name = req.Name
	}

	for i, node := range template.Workflow.Nodes {
		workflow.Nodes[i] = model.Node{
			ID:          fmt.Sprintf("node-%d", time.Now().UnixNano()+int64(i)),
			Name:        node.Name,
			Type:        node.Type,
			TypeVersion: node.TypeVersion,
			Position:    node.Position,
			Parameters:  node.Parameters,
		}
	}

	for key, conn := range template.Workflow.Connections {
		workflow.Connections[key] = conn
	}

	if err := s.storage.SaveWorkflow(workflow); err != nil {
		s.sendError(w, http.StatusInternalServerError, "Failed to create workflow from template", err)
		return
	}

	s.sendJSON(w, http.StatusCreated, workflow)
}
