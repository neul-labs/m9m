package service

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

const (
	// DefaultConnectTimeout is the timeout for connecting to the daemon
	DefaultConnectTimeout = 5 * time.Second

	// DefaultRequestTimeout is the timeout for a request
	DefaultRequestTimeout = 60 * time.Second
)

// Client communicates with the m9m daemon
type Client struct {
	socketPath     string
	connectTimeout time.Duration
	requestTimeout time.Duration
}

// ClientConfig holds client configuration
type ClientConfig struct {
	SocketPath     string
	ConnectTimeout time.Duration
	RequestTimeout time.Duration
}

// NewClient creates a new daemon client
func NewClient(config *ClientConfig) *Client {
	if config == nil {
		config = &ClientConfig{}
	}

	socketPath := config.SocketPath
	if socketPath == "" {
		socketPath = GetSocketPath()
	}

	connectTimeout := config.ConnectTimeout
	if connectTimeout == 0 {
		connectTimeout = DefaultConnectTimeout
	}

	requestTimeout := config.RequestTimeout
	if requestTimeout == 0 {
		requestTimeout = DefaultRequestTimeout
	}

	return &Client{
		socketPath:     socketPath,
		connectTimeout: connectTimeout,
		requestTimeout: requestTimeout,
	}
}

// IsRunning checks if the daemon is running
func (c *Client) IsRunning() bool {
	_, err := os.Stat(c.socketPath)
	if err != nil {
		return false
	}

	// Try to connect
	conn, err := net.DialTimeout("unix", c.socketPath, time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// EnsureRunning ensures the daemon is running, starting it if necessary
func (c *Client) EnsureRunning() error {
	if c.IsRunning() {
		return nil
	}

	// Start the daemon
	if err := c.StartDaemon(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Wait for daemon to be ready
	deadline := time.Now().Add(c.connectTimeout)
	for time.Now().Before(deadline) {
		if c.IsRunning() {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("daemon did not start within %v", c.connectTimeout)
}

// StartDaemon starts the daemon in the background
func (c *Client) StartDaemon() error {
	// Find the m9m binary
	execPath, err := os.Executable()
	if err != nil {
		execPath = "m9m"
	}

	// Start daemon in background
	cmd := exec.Command(execPath, "service", "start", "--background")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Create new session
	}

	// Redirect output to files for debugging
	home, _ := os.UserHomeDir()
	logDir := filepath.Join(home, ".m9m", "logs")
	os.MkdirAll(logDir, 0755)

	logFile, err := os.OpenFile(filepath.Join(logDir, "daemon.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Write PID file
	pidFile := GetPidFile()
	os.MkdirAll(filepath.Dir(pidFile), 0755)
	os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644)

	return nil
}

// StopDaemon stops the running daemon
func (c *Client) StopDaemon() error {
	// Send shutdown request
	resp, err := c.Call(&Request{Method: "shutdown"})
	if err != nil {
		// Try to kill by PID if we can't connect
		pidFile := GetPidFile()
		data, err := os.ReadFile(pidFile)
		if err == nil {
			pid, err := strconv.Atoi(string(data))
			if err == nil {
				proc, err := os.FindProcess(pid)
				if err == nil {
					proc.Signal(syscall.SIGTERM)
				}
			}
		}
		os.Remove(pidFile)
		os.Remove(c.socketPath)
		return nil
	}

	if !resp.Success {
		return fmt.Errorf("failed to stop daemon: %s", resp.Error)
	}

	os.Remove(GetPidFile())
	return nil
}

// Call makes a request to the daemon
func (c *Client) Call(req *Request) (*Response, error) {
	conn, err := net.DialTimeout("unix", c.socketPath, c.connectTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer conn.Close()

	// Set deadline
	conn.SetDeadline(time.Now().Add(c.requestTimeout))

	// Send request
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(req); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Read response
	decoder := json.NewDecoder(conn)
	var resp Response
	if err := decoder.Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return &resp, nil
}

// CallWithRetry calls the daemon, starting it if necessary
func (c *Client) CallWithRetry(req *Request) (*Response, error) {
	if err := c.EnsureRunning(); err != nil {
		return nil, err
	}
	return c.Call(req)
}

// Ping checks if the daemon is responding
func (c *Client) Ping() error {
	resp, err := c.Call(&Request{Method: "ping"})
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("ping failed: %s", resp.Error)
	}
	return nil
}

// Status gets the daemon status
func (c *Client) Status() (map[string]interface{}, error) {
	resp, err := c.Call(&Request{Method: "status"})
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("status failed: %s", resp.Error)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid status response")
	}
	return data, nil
}

// Workflow operations

// ListWorkflows lists workflows in the specified workspace
func (c *Client) ListWorkflows(workspace string, search string, limit int) (*Response, error) {
	params := map[string]interface{}{}
	if search != "" {
		params["search"] = search
	}
	if limit > 0 {
		params["limit"] = limit
	}

	return c.CallWithRetry(&Request{
		Method:    "workflow.list",
		Workspace: workspace,
		Params:    params,
	})
}

// GetWorkflow gets a workflow by ID or name
func (c *Client) GetWorkflow(workspace, idOrName string) (*Response, error) {
	// Try as ID first, then as name
	return c.CallWithRetry(&Request{
		Method:    "workflow.get",
		Workspace: workspace,
		Params: map[string]interface{}{
			"id":   idOrName,
			"name": idOrName,
		},
	})
}

// CreateWorkflow creates a new workflow
func (c *Client) CreateWorkflow(workspace string, workflow map[string]interface{}) (*Response, error) {
	return c.CallWithRetry(&Request{
		Method:    "workflow.create",
		Workspace: workspace,
		Params: map[string]interface{}{
			"workflow": workflow,
		},
	})
}

// RunWorkflow executes a workflow
func (c *Client) RunWorkflow(workspace, idOrName string, input map[string]interface{}) (*Response, error) {
	return c.CallWithRetry(&Request{
		Method:    "workflow.run",
		Workspace: workspace,
		Params: map[string]interface{}{
			"id":    idOrName,
			"name":  idOrName,
			"input": input,
		},
	})
}

// DeleteWorkflow deletes a workflow
func (c *Client) DeleteWorkflow(workspace, id string) (*Response, error) {
	return c.CallWithRetry(&Request{
		Method:    "workflow.delete",
		Workspace: workspace,
		Params: map[string]interface{}{
			"id": id,
		},
	})
}

// Execution operations

// ListExecutions lists workflow executions
func (c *Client) ListExecutions(workspace, workflowID string, limit int) (*Response, error) {
	params := map[string]interface{}{}
	if workflowID != "" {
		params["workflowId"] = workflowID
	}
	if limit > 0 {
		params["limit"] = limit
	}

	return c.CallWithRetry(&Request{
		Method:    "execution.list",
		Workspace: workspace,
		Params:    params,
	})
}

// GetExecution gets an execution by ID
func (c *Client) GetExecution(workspace, id string) (*Response, error) {
	return c.CallWithRetry(&Request{
		Method:    "execution.get",
		Workspace: workspace,
		Params: map[string]interface{}{
			"id": id,
		},
	})
}
