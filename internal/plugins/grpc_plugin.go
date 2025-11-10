package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v3"

	"github.com/dipankar/n8n-go/internal/model"
	"github.com/dipankar/n8n-go/internal/nodes/base"
)

// GRPCNodePlugin represents a node that communicates with an external gRPC service
type GRPCNodePlugin struct {
	Name        string
	Description base.NodeDescription
	ConfigPath  string
	Address     string
	Timeout     time.Duration
	conn        *grpc.ClientConn
	client      NodeServiceClient
}

// GRPCPluginConfig holds configuration for gRPC plugins
type GRPCPluginConfig struct {
	DefaultTimeout time.Duration
	MaxMessageSize int
	EnableRetry    bool
}

// DefaultGRPCPluginConfig returns default gRPC configuration
func DefaultGRPCPluginConfig() *GRPCPluginConfig {
	return &GRPCPluginConfig{
		DefaultTimeout: 30 * time.Second,
		MaxMessageSize: 100 * 1024 * 1024, // 100MB
		EnableRetry:    true,
	}
}

// GRPCPluginYAML represents the YAML configuration for a gRPC plugin
type GRPCPluginYAML struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Category    string            `yaml:"category"`
	Address     string            `yaml:"address"`
	Timeout     string            `yaml:"timeout"`
	Parameters  map[string]ParamSpec `yaml:"parameters"`
}

// ParamSpec describes a parameter
type ParamSpec struct {
	Type     string      `yaml:"type"`
	Required bool        `yaml:"required"`
	Default  interface{} `yaml:"default"`
}

// LoadGRPCPlugin loads a gRPC plugin from a YAML configuration file
func LoadGRPCPlugin(configPath string, config *GRPCPluginConfig) (*GRPCNodePlugin, error) {
	if config == nil {
		config = DefaultGRPCPluginConfig()
	}

	// Read configuration file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var pluginConfig GRPCPluginYAML
	if err := yaml.Unmarshal(data, &pluginConfig); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// Parse timeout
	timeout := config.DefaultTimeout
	if pluginConfig.Timeout != "" {
		if parsedTimeout, err := time.ParseDuration(pluginConfig.Timeout); err == nil {
			timeout = parsedTimeout
		}
	}

	// Create gRPC connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(config.MaxMessageSize),
			grpc.MaxCallSendMsgSize(config.MaxMessageSize),
		),
	}

	conn, err := grpc.DialContext(ctx, pluginConfig.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC service: %w", err)
	}

	plugin := &GRPCNodePlugin{
		Name: pluginConfig.Name,
		Description: base.NodeDescription{
			Name:        pluginConfig.Name,
			Description: pluginConfig.Description,
			Category:    pluginConfig.Category,
		},
		ConfigPath: configPath,
		Address:    pluginConfig.Address,
		Timeout:    timeout,
		conn:       conn,
		client:     NewNodeServiceClient(conn),
	}

	return plugin, nil
}

// Execute calls the external gRPC service to execute the node
func (p *GRPCNodePlugin) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.Timeout)
	defer cancel()

	// Convert input data to JSON
	inputJSON, err := json.Marshal(inputData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input data: %w", err)
	}

	paramsJSON, err := json.Marshal(nodeParams)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	// Call gRPC service
	request := &ExecuteRequest{
		InputData:  string(inputJSON),
		Parameters: string(paramsJSON),
	}

	response, err := p.client.Execute(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("gRPC call failed: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("execution failed: %s", response.Error)
	}

	// Parse output data
	var outputData []model.DataItem
	if err := json.Unmarshal([]byte(response.OutputData), &outputData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal output data: %w", err)
	}

	return outputData, nil
}

// ValidateParameters calls the external service to validate parameters (optional)
func (p *GRPCNodePlugin) ValidateParameters(params map[string]interface{}) error {
	// Basic validation - external service can provide more detailed validation
	return nil
}

// GetDescription returns the node description
func (p *GRPCNodePlugin) GetDescription() base.NodeDescription {
	return p.Description
}

// GetType returns the plugin type
func (p *GRPCNodePlugin) GetType() PluginType {
	return PluginTypeGRPC
}

// Close closes the gRPC connection
func (p *GRPCNodePlugin) Close() error {
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

// Simplified gRPC interface (can be replaced with proper protobuf definitions)

// NodeServiceClient is the client API for NodeService
type NodeServiceClient interface {
	Execute(ctx context.Context, in *ExecuteRequest, opts ...grpc.CallOption) (*ExecuteResponse, error)
	Describe(ctx context.Context, in *DescribeRequest, opts ...grpc.CallOption) (*DescribeResponse, error)
}

type nodeServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewNodeServiceClient(cc grpc.ClientConnInterface) NodeServiceClient {
	return &nodeServiceClient{cc}
}

// ExecuteRequest represents the execution request
type ExecuteRequest struct {
	InputData  string `json:"inputData"`
	Parameters string `json:"parameters"`
}

// ExecuteResponse represents the execution response
type ExecuteResponse struct {
	Success    bool   `json:"success"`
	OutputData string `json:"outputData"`
	Error      string `json:"error"`
}

// DescribeRequest represents the describe request
type DescribeRequest struct{}

// DescribeResponse represents the describe response
type DescribeResponse struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// Execute implements the gRPC Execute call
func (c *nodeServiceClient) Execute(ctx context.Context, in *ExecuteRequest, opts ...grpc.CallOption) (*ExecuteResponse, error) {
	out := new(ExecuteResponse)
	err := c.cc.Invoke(ctx, "/nodeservice.NodeService/Execute", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Describe implements the gRPC Describe call
func (c *nodeServiceClient) Describe(ctx context.Context, in *DescribeRequest, opts ...grpc.CallOption) (*DescribeResponse, error) {
	out := new(DescribeResponse)
	err := c.cc.Invoke(ctx, "/nodeservice.NodeService/Describe", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
