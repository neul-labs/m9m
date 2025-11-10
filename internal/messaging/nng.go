/*
Package messaging implements peer-to-peer messaging for distributed n8n-go using NNG.

This package provides lightweight messaging between cluster nodes without requiring
external message brokers. It supports broadcasting execution updates, WebSocket
messages, and other cluster events.
*/
package messaging

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/pub"
	"go.nanomsg.org/mangos/v3/protocol/sub"
	_ "go.nanomsg.org/mangos/v3/transport/tcp" // Register TCP transport
)

// Message represents a message exchanged between cluster nodes
type Message struct {
	Type      string                 `json:"type"`      // Message type (e.g., "execution.update", "websocket.broadcast")
	NodeID    string                 `json:"node_id"`   // Originating node ID
	Payload   map[string]interface{} `json:"payload"`   // Message payload
	Timestamp time.Time              `json:"timestamp"` // Message timestamp
}

// MessageHandler is a function that processes incoming messages
type MessageHandler func(msg Message) error

// NNGMessaging implements peer-to-peer messaging using NNG
type NNGMessaging struct {
	nodeID      string
	pubSocket   mangos.Socket
	subSocket   mangos.Socket
	handlers    map[string][]MessageHandler
	handlersMux sync.RWMutex
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

// NewNNGMessaging creates a new NNG messaging instance
func NewNNGMessaging(nodeID string, pubAddr string, subAddrs []string) (*NNGMessaging, error) {
	nm := &NNGMessaging{
		nodeID:   nodeID,
		handlers: make(map[string][]MessageHandler),
		stopChan: make(chan struct{}),
	}

	// Create publisher socket
	var err error
	if nm.pubSocket, err = pub.NewSocket(); err != nil {
		return nil, fmt.Errorf("failed to create pub socket: %w", err)
	}

	// Set socket options
	if err := nm.pubSocket.SetOption(mangos.OptionSendDeadline, 5*time.Second); err != nil {
		return nil, fmt.Errorf("failed to set send deadline: %w", err)
	}

	// Listen on publisher address
	if err := nm.pubSocket.Listen(pubAddr); err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", pubAddr, err)
	}
	log.Printf("NNG publisher listening on %s", pubAddr)

	// Create subscriber socket
	if nm.subSocket, err = sub.NewSocket(); err != nil {
		nm.pubSocket.Close()
		return nil, fmt.Errorf("failed to create sub socket: %w", err)
	}

	// Subscribe to all topics (empty string means all)
	if err := nm.subSocket.SetOption(mangos.OptionSubscribe, ""); err != nil {
		nm.pubSocket.Close()
		nm.subSocket.Close()
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	// Set socket options
	if err := nm.subSocket.SetOption(mangos.OptionRecvDeadline, 100*time.Millisecond); err != nil {
		nm.pubSocket.Close()
		nm.subSocket.Close()
		return nil, fmt.Errorf("failed to set recv deadline: %w", err)
	}

	// Dial to all peer publishers
	for _, addr := range subAddrs {
		if err := nm.subSocket.Dial(addr); err != nil {
			log.Printf("Warning: failed to dial %s: %v (will retry in background)", addr, err)
		} else {
			log.Printf("NNG subscriber connected to %s", addr)
		}
	}

	// Start message receiver
	nm.wg.Add(1)
	go nm.receiveLoop()

	return nm, nil
}

// Publish broadcasts a message to all cluster nodes
func (nm *NNGMessaging) Publish(msgType string, payload map[string]interface{}) error {
	msg := Message{
		Type:      msgType,
		NodeID:    nm.nodeID,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := nm.pubSocket.Send(data); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// Subscribe registers a handler for messages of a specific type
func (nm *NNGMessaging) Subscribe(msgType string, handler MessageHandler) {
	nm.handlersMux.Lock()
	defer nm.handlersMux.Unlock()

	nm.handlers[msgType] = append(nm.handlers[msgType], handler)
	log.Printf("Registered handler for message type: %s", msgType)
}

// BroadcastExecutionUpdate broadcasts a workflow execution update
func (nm *NNGMessaging) BroadcastExecutionUpdate(execID, status string, data map[string]interface{}) error {
	payload := map[string]interface{}{
		"executionId": execID,
		"status":      status,
	}

	// Merge additional data
	for k, v := range data {
		payload[k] = v
	}

	return nm.Publish("execution.update", payload)
}

// BroadcastWebSocketMessage broadcasts a WebSocket message to all nodes
func (nm *NNGMessaging) BroadcastWebSocketMessage(msgType string, data map[string]interface{}) error {
	payload := map[string]interface{}{
		"messageType": msgType,
		"data":        data,
	}

	return nm.Publish("websocket.broadcast", payload)
}

// BroadcastNodeExecution broadcasts a node execution event
func (nm *NNGMessaging) BroadcastNodeExecution(execID, nodeID, status string) error {
	payload := map[string]interface{}{
		"executionId": execID,
		"nodeId":      nodeID,
		"status":      status,
	}

	return nm.Publish("node.execution", payload)
}

// receiveLoop continuously receives and processes messages
func (nm *NNGMessaging) receiveLoop() {
	defer nm.wg.Done()

	for {
		select {
		case <-nm.stopChan:
			return
		default:
		}

		// Receive message (with timeout from socket options)
		data, err := nm.subSocket.Recv()
		if err != nil {
			// Timeout is expected (OptionRecvDeadline), continue
			continue
		}

		// Parse message
		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		// Ignore messages from self
		if msg.NodeID == nm.nodeID {
			continue
		}

		// Dispatch to handlers
		nm.dispatchMessage(msg)
	}
}

// dispatchMessage dispatches a message to registered handlers
func (nm *NNGMessaging) dispatchMessage(msg Message) {
	nm.handlersMux.RLock()
	handlers, exists := nm.handlers[msg.Type]
	nm.handlersMux.RUnlock()

	if !exists {
		return
	}

	// Call all handlers for this message type
	for _, handler := range handlers {
		go func(h MessageHandler) {
			if err := h(msg); err != nil {
				log.Printf("Error handling message type %s: %v", msg.Type, err)
			}
		}(handler)
	}
}

// AddPeer dynamically adds a new peer to subscribe to
func (nm *NNGMessaging) AddPeer(addr string) error {
	if err := nm.subSocket.Dial(addr); err != nil {
		return fmt.Errorf("failed to dial peer %s: %w", addr, err)
	}
	log.Printf("Added peer: %s", addr)
	return nil
}

// Close gracefully shuts down the messaging system
func (nm *NNGMessaging) Close() error {
	close(nm.stopChan)
	nm.wg.Wait()

	if err := nm.pubSocket.Close(); err != nil {
		return fmt.Errorf("failed to close pub socket: %w", err)
	}

	if err := nm.subSocket.Close(); err != nil {
		return fmt.Errorf("failed to close sub socket: %w", err)
	}

	log.Println("NNG messaging shut down successfully")
	return nil
}

// Stats returns messaging statistics
func (nm *NNGMessaging) Stats() map[string]interface{} {
	nm.handlersMux.RLock()
	defer nm.handlersMux.RUnlock()

	return map[string]interface{}{
		"node_id":        nm.nodeID,
		"handler_types":  len(nm.handlers),
		"total_handlers": nm.totalHandlers(),
	}
}

// totalHandlers counts total number of registered handlers
func (nm *NNGMessaging) totalHandlers() int {
	total := 0
	for _, handlers := range nm.handlers {
		total += len(handlers)
	}
	return total
}
