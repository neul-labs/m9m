// Package transport provides MCP transport implementations.
package transport

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/dipankar/m9m/internal/mcp"
)

// StdioTransport implements MCP transport over stdin/stdout
type StdioTransport struct {
	reader *bufio.Reader
	writer io.Writer
	mu     sync.Mutex
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport(reader io.Reader, writer io.Writer) *StdioTransport {
	return &StdioTransport{
		reader: bufio.NewReader(reader),
		writer: writer,
	}
}

// ReadRequest reads a JSON-RPC request from stdin
func (t *StdioTransport) ReadRequest() (*mcp.Request, error) {
	line, err := t.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	var req mcp.Request
	if err := json.Unmarshal(line, &req); err != nil {
		return nil, fmt.Errorf("failed to parse request: %w", err)
	}

	return &req, nil
}

// WriteResponse writes a JSON-RPC response to stdout
func (t *StdioTransport) WriteResponse(response *mcp.Response) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	_, err = fmt.Fprintf(t.writer, "%s\n", data)
	return err
}

// WriteNotification writes a JSON-RPC notification to stdout
func (t *StdioTransport) WriteNotification(notification *mcp.Notification) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	_, err = fmt.Fprintf(t.writer, "%s\n", data)
	return err
}

// Close closes the transport (no-op for stdio)
func (t *StdioTransport) Close() error {
	return nil
}
