package plugins

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// RESTNodePlugin represents a node that calls a REST API endpoint
type RESTNodePlugin struct {
	Name        string
	Description base.NodeDescription
	ConfigPath  string
	Endpoint    string
	Method      string
	Headers     map[string]string
	Timeout     time.Duration
	httpClient  *http.Client
}

// RESTPluginConfig holds configuration for REST plugins
type RESTPluginConfig struct {
	DefaultTimeout time.Duration
	MaxRetries     int
}

// DefaultRESTPluginConfig returns default REST configuration
func DefaultRESTPluginConfig() *RESTPluginConfig {
	return &RESTPluginConfig{
		DefaultTimeout: 30 * time.Second,
		MaxRetries:     3,
	}
}

// RESTPluginYAML represents the YAML configuration for a REST plugin
type RESTPluginYAML struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Category    string            `yaml:"category"`
	Endpoint    string            `yaml:"endpoint"`
	Method      string            `yaml:"method"`
	Headers     map[string]string `yaml:"headers"`
	Timeout     string            `yaml:"timeout"`
	Parameters  map[string]ParamSpec `yaml:"parameters"`
}

// LoadRESTPlugin loads a REST plugin from a YAML configuration file
func LoadRESTPlugin(configPath string, config *RESTPluginConfig) (*RESTNodePlugin, error) {
	if config == nil {
		config = DefaultRESTPluginConfig()
	}

	// Read configuration file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var pluginConfig RESTPluginYAML
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

	// Default method to POST
	method := pluginConfig.Method
	if method == "" {
		method = "POST"
	}

	plugin := &RESTNodePlugin{
		Name: pluginConfig.Name,
		Description: base.NodeDescription{
			Name:        pluginConfig.Name,
			Description: pluginConfig.Description,
			Category:    pluginConfig.Category,
		},
		ConfigPath: configPath,
		Endpoint:   pluginConfig.Endpoint,
		Method:     method,
		Headers:    pluginConfig.Headers,
		Timeout:    timeout,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}

	return plugin, nil
}

// Execute calls the REST API endpoint to execute the node
func (p *RESTNodePlugin) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	// Prepare request payload
	payload := map[string]interface{}{
		"inputData":  inputData,
		"parameters": nodeParams,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest(p.Method, p.Endpoint, bytes.NewBuffer(payloadJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range p.Headers {
		req.Header.Set(key, value)
	}

	// Make request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var response RESTExecuteResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("execution failed: %s", response.Error)
	}

	return response.OutputData, nil
}

// ValidateParameters validates node parameters
func (p *RESTNodePlugin) ValidateParameters(params map[string]interface{}) error {
	// Basic validation - external service can provide more detailed validation
	return nil
}

// GetDescription returns the node description
func (p *RESTNodePlugin) GetDescription() base.NodeDescription {
	return p.Description
}

// GetType returns the plugin type
func (p *RESTNodePlugin) GetType() PluginType {
	return PluginTypeREST
}

// RESTExecuteResponse represents the response from a REST API execution
type RESTExecuteResponse struct {
	Success    bool              `json:"success"`
	OutputData []model.DataItem  `json:"outputData"`
	Error      string            `json:"error,omitempty"`
}
