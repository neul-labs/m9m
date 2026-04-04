package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/neul-labs/m9m/internal/model"
)

func (s *APIServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade to WebSocket", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	clientID := fmt.Sprintf("client_%d", time.Now().UnixNano())
	s.wsClients[clientID] = conn
	defer delete(s.wsClients, clientID)

	conn.WriteJSON(map[string]interface{}{
		"type": "connected",
		"data": map[string]interface{}{
			"clientId": clientID,
			"time":     time.Now().UTC().Format(time.RFC3339),
		},
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err == nil {
			conn.WriteJSON(map[string]interface{}{
				"type": "response",
				"data": msg,
			})
		}
	}
}

func (s *APIServer) BroadcastExecutionUpdate(execution *model.WorkflowExecution) {
	message := map[string]interface{}{
		"type": "executionUpdate",
		"data": execution,
	}

	for _, conn := range s.wsClients {
		conn.WriteJSON(message)
	}
}
