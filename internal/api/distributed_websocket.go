/*
Package api provides distributed WebSocket management using NNG messaging.

This allows WebSocket clients connected to any node in the cluster to receive
execution updates and real-time events from workflows running on any other node.
*/
package api

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/dipankar/n8n-go/internal/messaging"
	"github.com/dipankar/n8n-go/internal/model"
)

// DistributedWebSocketManager manages WebSocket connections across cluster nodes
type DistributedWebSocketManager struct {
	localClients    map[string]*websocket.Conn
	clientsMutex    sync.RWMutex
	messaging       *messaging.NNGMessaging
	messageHandlers map[string]func(map[string]interface{})
}

// NewDistributedWebSocketManager creates a new distributed WebSocket manager
func NewDistributedWebSocketManager(nng *messaging.NNGMessaging) *DistributedWebSocketManager {
	manager := &DistributedWebSocketManager{
		localClients:    make(map[string]*websocket.Conn),
		messaging:       nng,
		messageHandlers: make(map[string]func(map[string]interface{})),
	}

	// Subscribe to cluster-wide WebSocket broadcasts
	nng.Subscribe("websocket.broadcast", manager.handleWebSocketBroadcast)
	nng.Subscribe("execution.update", manager.handleExecutionUpdate)
	nng.Subscribe("node.execution", manager.handleNodeExecution)

	log.Println("Distributed WebSocket manager initialized")
	return manager
}

// AddClient registers a new WebSocket connection
func (m *DistributedWebSocketManager) AddClient(clientID string, conn *websocket.Conn) {
	m.clientsMutex.Lock()
	defer m.clientsMutex.Unlock()

	m.localClients[clientID] = conn
	log.Printf("WebSocket client connected: %s (total local clients: %d)", clientID, len(m.localClients))
}

// RemoveClient unregisters a WebSocket connection
func (m *DistributedWebSocketManager) RemoveClient(clientID string) {
	m.clientsMutex.Lock()
	defer m.clientsMutex.Unlock()

	if conn, exists := m.localClients[clientID]; exists {
		conn.Close()
		delete(m.localClients, clientID)
		log.Printf("WebSocket client disconnected: %s (remaining local clients: %d)", clientID, len(m.localClients))
	}
}

// BroadcastToCluster broadcasts a message to all nodes in the cluster
func (m *DistributedWebSocketManager) BroadcastToCluster(msgType string, data map[string]interface{}) error {
	return m.messaging.BroadcastWebSocketMessage(msgType, data)
}

// BroadcastExecutionUpdate broadcasts a workflow execution update to all clients
func (m *DistributedWebSocketManager) BroadcastExecutionUpdate(execution *model.WorkflowExecution) error {
	data := map[string]interface{}{
		"executionId": execution.ID,
		"workflowId":  execution.WorkflowID,
		"status":      execution.Status,
		"startedAt":   execution.StartedAt,
		"mode":        execution.Mode,
	}

	if execution.FinishedAt != nil && !execution.FinishedAt.IsZero() {
		data["finishedAt"] = execution.FinishedAt
	}

	if execution.Error != nil {
		data["error"] = execution.Error.Error()
	}

	// Broadcast to cluster
	return m.messaging.BroadcastExecutionUpdate(
		execution.ID,
		execution.Status,
		data,
	)
}

// BroadcastNodeExecution broadcasts a node execution event
func (m *DistributedWebSocketManager) BroadcastNodeExecution(execID, nodeID, status string, data map[string]interface{}) error {
	if data == nil {
		data = make(map[string]interface{})
	}

	data["executionId"] = execID
	data["nodeId"] = nodeID
	data["status"] = status

	return m.messaging.BroadcastNodeExecution(execID, nodeID, status)
}

// handleWebSocketBroadcast handles incoming websocket.broadcast messages
func (m *DistributedWebSocketManager) handleWebSocketBroadcast(msg messaging.Message) error {
	msgType, ok := msg.Payload["messageType"].(string)
	if !ok {
		return nil
	}

	data, ok := msg.Payload["data"].(map[string]interface{})
	if !ok {
		return nil
	}

	// Send to all local WebSocket clients
	m.sendToLocalClients(msgType, data)
	return nil
}

// handleExecutionUpdate handles incoming execution.update messages
func (m *DistributedWebSocketManager) handleExecutionUpdate(msg messaging.Message) error {
	// Extract execution data
	execID, _ := msg.Payload["executionId"].(string)
	status, _ := msg.Payload["status"].(string)

	// Prepare WebSocket message
	wsMessage := map[string]interface{}{
		"type": "executionUpdate",
		"data": map[string]interface{}{
			"executionId": execID,
			"status":      status,
		},
	}

	// Add all other payload fields
	for k, v := range msg.Payload {
		if k != "executionId" && k != "status" {
			wsMessage["data"].(map[string]interface{})[k] = v
		}
	}

	// Send to local clients
	m.sendToLocalClients("executionUpdate", wsMessage["data"].(map[string]interface{}))
	return nil
}

// handleNodeExecution handles incoming node.execution messages
func (m *DistributedWebSocketManager) handleNodeExecution(msg messaging.Message) error {
	execID, _ := msg.Payload["executionId"].(string)
	nodeID, _ := msg.Payload["nodeId"].(string)
	status, _ := msg.Payload["status"].(string)

	wsMessage := map[string]interface{}{
		"type": "nodeExecution",
		"data": map[string]interface{}{
			"executionId": execID,
			"nodeId":      nodeID,
			"status":      status,
		},
	}

	m.sendToLocalClients("nodeExecution", wsMessage["data"].(map[string]interface{}))
	return nil
}

// sendToLocalClients sends a message to all local WebSocket clients
func (m *DistributedWebSocketManager) sendToLocalClients(msgType string, data map[string]interface{}) {
	m.clientsMutex.RLock()
	defer m.clientsMutex.RUnlock()

	if len(m.localClients) == 0 {
		return
	}

	// Prepare message
	message := map[string]interface{}{
		"type": msgType,
		"data": data,
		"timestamp": time.Now().Unix(),
	}

	messageJSON, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling WebSocket message: %v", err)
		return
	}

	// Send to all local clients
	var disconnected []string
	for clientID, conn := range m.localClients {
		if err := conn.WriteMessage(websocket.TextMessage, messageJSON); err != nil {
			log.Printf("Error sending message to client %s: %v", clientID, err)
			disconnected = append(disconnected, clientID)
		}
	}

	// Clean up disconnected clients
	if len(disconnected) > 0 {
		m.clientsMutex.RUnlock()
		m.clientsMutex.Lock()
		for _, clientID := range disconnected {
			if conn, exists := m.localClients[clientID]; exists {
				conn.Close()
				delete(m.localClients, clientID)
			}
		}
		m.clientsMutex.Unlock()
		m.clientsMutex.RLock()
	}
}

// SendToClient sends a message to a specific local client
func (m *DistributedWebSocketManager) SendToClient(clientID string, msgType string, data map[string]interface{}) error {
	m.clientsMutex.RLock()
	conn, exists := m.localClients[clientID]
	m.clientsMutex.RUnlock()

	if !exists {
		return nil // Client not on this node
	}

	message := map[string]interface{}{
		"type": msgType,
		"data": data,
		"timestamp": time.Now().Unix(),
	}

	messageJSON, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, messageJSON)
}

// GetLocalClientCount returns the number of clients connected to this node
func (m *DistributedWebSocketManager) GetLocalClientCount() int {
	m.clientsMutex.RLock()
	defer m.clientsMutex.RUnlock()
	return len(m.localClients)
}

// GetLocalClients returns a list of local client IDs
func (m *DistributedWebSocketManager) GetLocalClients() []string {
	m.clientsMutex.RLock()
	defer m.clientsMutex.RUnlock()

	clients := make([]string, 0, len(m.localClients))
	for clientID := range m.localClients {
		clients = append(clients, clientID)
	}
	return clients
}

// PingClients sends ping messages to all local clients to keep connections alive
func (m *DistributedWebSocketManager) PingClients() {
	m.clientsMutex.RLock()
	defer m.clientsMutex.RUnlock()

	var disconnected []string
	for clientID, conn := range m.localClients {
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			log.Printf("Error pinging client %s: %v", clientID, err)
			disconnected = append(disconnected, clientID)
		}
	}

	// Clean up disconnected clients
	if len(disconnected) > 0 {
		m.clientsMutex.RUnlock()
		m.clientsMutex.Lock()
		for _, clientID := range disconnected {
			if conn, exists := m.localClients[clientID]; exists {
				conn.Close()
				delete(m.localClients, clientID)
			}
		}
		m.clientsMutex.Unlock()
		m.clientsMutex.RLock()
	}
}

// StartPingRoutine starts a goroutine that pings clients periodically
func (m *DistributedWebSocketManager) StartPingRoutine() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			m.PingClients()
		}
	}()
}

// Stats returns statistics about WebSocket connections
func (m *DistributedWebSocketManager) Stats() map[string]interface{} {
	m.clientsMutex.RLock()
	defer m.clientsMutex.RUnlock()

	return map[string]interface{}{
		"local_clients":      len(m.localClients),
		"messaging_stats":    m.messaging.Stats(),
	}
}
