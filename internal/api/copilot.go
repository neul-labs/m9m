package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/neul-labs/m9m/internal/model"
)

func (s *APIServer) CopilotGenerate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Description string                 `json:"description"`
		Context     map[string]interface{} `json:"context,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	if req.Description == "" {
		s.sendError(w, http.StatusBadRequest, "Description is required", nil)
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"workflow": map[string]interface{}{
			"name":   "Generated Workflow",
			"active": false,
			"nodes": []map[string]interface{}{
				{
					"id":       "trigger-1",
					"name":     "Manual Trigger",
					"type":     "n8n-nodes-base.manualTrigger",
					"position": []int{250, 300},
				},
			},
			"connections": map[string]interface{}{},
		},
		"explanation": "This is a basic workflow. Connect the copilot to an AI provider for full generation.",
		"suggestions": []string{
			"Configure M9M_COPILOT_API_KEY for AI-powered generation",
			"Set M9M_COPILOT_PROVIDER to 'openai', 'anthropic', or 'ollama'",
		},
	})
}

func (s *APIServer) CopilotSuggest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CurrentWorkflow interface{} `json:"currentWorkflow,omitempty"`
		SelectedNode    string      `json:"selectedNode,omitempty"`
		UserQuery       string      `json:"userQuery"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"suggestions": []map[string]interface{}{
			{
				"type":        "n8n-nodes-base.httpRequest",
				"name":        "HTTP Request",
				"description": "Make HTTP requests to external APIs",
				"reason":      "Commonly used for API integrations",
				"confidence":  0.9,
			},
			{
				"type":        "n8n-nodes-base.set",
				"name":        "Set",
				"description": "Set field values",
				"reason":      "Transform data between nodes",
				"confidence":  0.85,
			},
			{
				"type":        "n8n-nodes-base.if",
				"name":        "IF",
				"description": "Conditional branching",
				"reason":      "Add logic to your workflow",
				"confidence":  0.8,
			},
		},
	})
}

func (s *APIServer) CopilotExplain(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Workflow *model.Workflow `json:"workflow"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	if req.Workflow == nil {
		s.sendError(w, http.StatusBadRequest, "Workflow is required", nil)
		return
	}

	nodeCount := len(req.Workflow.Nodes)
	connectionCount := len(req.Workflow.Connections)

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"summary":  fmt.Sprintf("This workflow '%s' contains %d nodes with %d connections.", req.Workflow.Name, nodeCount, connectionCount),
		"dataFlow": "Data flows from trigger nodes through processing nodes to output.",
		"suggestions": []string{
			"Configure copilot AI for detailed explanations",
		},
	})
}

func (s *APIServer) CopilotFix(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Workflow     *model.Workflow `json:"workflow"`
		ErrorMessage string          `json:"errorMessage"`
		FailedNode   string          `json:"failedNode"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"diagnosis": fmt.Sprintf("Error in node '%s': %s", req.FailedNode, req.ErrorMessage),
		"fixes": []map[string]interface{}{
			{
				"description": "Check node parameters and credentials",
				"confidence":  0.8,
				"autoApply":   false,
			},
			{
				"description": "Verify input data format matches expected schema",
				"confidence":  0.7,
				"autoApply":   false,
			},
		},
		"prevention": "Add validation nodes before critical operations",
	})
}

func (s *APIServer) CopilotChat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		CurrentWorkflow *model.Workflow `json:"currentWorkflow,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	lastMessage := ""
	if len(req.Messages) > 0 {
		lastMessage = req.Messages[len(req.Messages)-1].Content
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("I understand you want to: '%s'. To enable full AI chat, configure M9M_COPILOT_API_KEY.", lastMessage),
		"actions": []map[string]interface{}{},
	})
}
