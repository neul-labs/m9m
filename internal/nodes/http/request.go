/*
Package http provides HTTP-related node implementations for m9m.
*/
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/neul-labs/m9m/internal/expressions"
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// SSRF protection: blocked IP ranges and hosts
var (
	// blockedIPRanges contains private and special-use IP ranges that should not be accessed
	blockedIPRanges = []struct {
		network *net.IPNet
		name    string
	}{
		{mustParseCIDR("10.0.0.0/8"), "Private (10.x.x.x)"},
		{mustParseCIDR("172.16.0.0/12"), "Private (172.16-31.x.x)"},
		{mustParseCIDR("192.168.0.0/16"), "Private (192.168.x.x)"},
		{mustParseCIDR("127.0.0.0/8"), "Loopback"},
		{mustParseCIDR("169.254.0.0/16"), "Link-local / Cloud Metadata"},
		{mustParseCIDR("0.0.0.0/8"), "Current network"},
		{mustParseCIDR("100.64.0.0/10"), "Shared address space"},
		{mustParseCIDR("192.0.0.0/24"), "IETF Protocol"},
		{mustParseCIDR("192.0.2.0/24"), "TEST-NET-1"},
		{mustParseCIDR("198.51.100.0/24"), "TEST-NET-2"},
		{mustParseCIDR("203.0.113.0/24"), "TEST-NET-3"},
		{mustParseCIDR("224.0.0.0/4"), "Multicast"},
		{mustParseCIDR("240.0.0.0/4"), "Reserved"},
		{mustParseCIDR("255.255.255.255/32"), "Broadcast"},
		// IPv6 blocked ranges
		{mustParseCIDR("::1/128"), "IPv6 Loopback"},
		{mustParseCIDR("fc00::/7"), "IPv6 Unique local"},
		{mustParseCIDR("fe80::/10"), "IPv6 Link-local"},
		{mustParseCIDR("ff00::/8"), "IPv6 Multicast"},
	}

	// blockedHosts contains specific hostnames that should not be accessed
	blockedHosts = map[string]bool{
		"localhost":                          true,
		"metadata.google.internal":           true, // GCP metadata
		"metadata.goog":                      true, // GCP metadata
		"169.254.169.254":                    true, // AWS/Azure/GCP metadata
		"169.254.170.2":                      true, // AWS ECS metadata
		"fd00:ec2::254":                      true, // AWS EC2 IPv6 metadata
		"[fd00:ec2::254]":                    true,
		"instance-data":                      true, // OpenStack metadata
		"metadata":                           true,
		"kubernetes.default":                 true, // Kubernetes API
		"kubernetes.default.svc":             true,
		"kubernetes.default.svc.cluster.local": true,
	}
)

func mustParseCIDR(s string) *net.IPNet {
	_, network, err := net.ParseCIDR(s)
	if err != nil {
		panic(fmt.Sprintf("invalid CIDR: %s", s))
	}
	return network
}

// isBlockedIP checks if an IP address is in a blocked range
func isBlockedIP(ip net.IP) (bool, string) {
	for _, blocked := range blockedIPRanges {
		if blocked.network.Contains(ip) {
			return true, blocked.name
		}
	}
	return false, ""
}

// validateURLForSSRF checks if a URL is safe to request (not internal/metadata)
// SECURITY: Prevents Server-Side Request Forgery attacks
func validateURLForSSRF(urlStr string) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow http and https schemes
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("only http and https schemes are allowed, got: %s", parsedURL.Scheme)
	}

	// Extract hostname (without port)
	hostname := parsedURL.Hostname()

	// Check if hostname is in blocked list
	if blockedHosts[strings.ToLower(hostname)] {
		return fmt.Errorf("access to host '%s' is blocked for security reasons", hostname)
	}

	// Resolve hostname to IP addresses
	ips, err := net.LookupIP(hostname)
	if err != nil {
		// Allow the request to proceed - DNS resolution will fail at request time
		// This handles cases where the hostname is valid but DNS is temporarily unavailable
		log.Printf("SECURITY WARNING: Could not resolve hostname '%s' for SSRF check: %v", hostname, err)
		return nil
	}

	// Check each resolved IP against blocked ranges
	for _, ip := range ips {
		if blocked, reason := isBlockedIP(ip); blocked {
			return fmt.Errorf("access to IP %s (%s) is blocked for security reasons: %s", ip.String(), hostname, reason)
		}
	}

	return nil
}

// HTTPRequestNode implements the HTTP Request node functionality
type HTTPRequestNode struct {
	*base.BaseNode
	client *http.Client
}

// NewHTTPRequestNode creates a new HTTP Request node
func NewHTTPRequestNode() *HTTPRequestNode {
	description := base.NodeDescription{
		Name:        "HTTP Request",
		Description: "Makes HTTP requests to REST APIs and other web services",
		Category:    "HTTP",
	}
	
	return &HTTPRequestNode{
		BaseNode: base.NewBaseNode(description),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Description returns the node description
func (h *HTTPRequestNode) Description() base.NodeDescription {
	return h.BaseNode.Description()
}

// ValidateParameters validates HTTP Request node parameters
func (h *HTTPRequestNode) ValidateParameters(params map[string]interface{}) error {
	if params == nil {
		return h.CreateError("parameters cannot be nil", nil)
	}
	
	// Check if url exists
	url := h.GetStringParameter(params, "url", "")
	if url == "" {
		return h.CreateError("url parameter is required", nil)
	}
	
	// Check if method exists
	method := h.GetStringParameter(params, "method", "GET")
	
	// Allow expressions in method parameter
	if expressions.IsExpression(method) {
		// Method is an expression, so we can't validate it at this stage
		// It will be validated during execution after expression evaluation
		return nil
	}
	
	// Validate method if it's not an expression
	validMethods := map[string]bool{
		"GET":     true,
		"POST":    true,
		"PUT":     true,
		"PATCH":   true,
		"DELETE":  true,
		"HEAD":    true,
		"OPTIONS": true,
	}
	
	if !validMethods[strings.ToUpper(method)] {
		return h.CreateError(fmt.Sprintf("invalid HTTP method: %s", method), nil)
	}
	
	return nil
}

// Execute processes the HTTP Request node operation
func (h *HTTPRequestNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	if len(inputData) == 0 {
		return []model.DataItem{}, nil
	}
	
	// Create execution context for expression evaluation
	execContext := &expressions.ExecutionContext{
		InputData: inputData,
		ItemIndex: 0,
		Variables: make(map[string]interface{}),
	}
	
	// Process each input data item
	result := make([]model.DataItem, len(inputData))
	
	for i, item := range inputData {
		// Update execution context for current item
		execContext.ItemIndex = i
		execContext.Variables["json"] = item.JSON
		
		// Evaluate node parameters with expression context
		evaluatedParams, err := h.evaluateParameters(nodeParams, execContext)
		if err != nil {
			return nil, h.CreateError(fmt.Sprintf("failed to evaluate parameters: %v", err), nil)
		}
		
		// Get required parameters from evaluated parameters
		requestURL := h.GetStringParameter(evaluatedParams, "url", "")
		if requestURL == "" {
			return nil, h.CreateError("url parameter cannot be empty", nil)
		}

		// SECURITY: Validate URL against SSRF attacks
		// Check if allowInternalRequests is explicitly enabled (disabled by default)
		allowInternal := h.GetBoolParameter(evaluatedParams, "allowInternalRequests", false)
		if !allowInternal {
			if err := validateURLForSSRF(requestURL); err != nil {
				return nil, h.CreateError(fmt.Sprintf("SSRF protection: %v", err), nil)
			}
		} else {
			log.Printf("SECURITY WARNING: SSRF protection disabled for request to %s", requestURL)
		}

		method := h.GetStringParameter(evaluatedParams, "method", "GET")
		
		// Validate method after evaluation
		validMethods := map[string]bool{
			"GET":     true,
			"POST":    true,
			"PUT":     true,
			"PATCH":   true,
			"DELETE":  true,
			"HEAD":    true,
			"OPTIONS": true,
		}
		
		if !validMethods[strings.ToUpper(method)] {
			return nil, h.CreateError(fmt.Sprintf("invalid HTTP method: %s", method), nil)
		}
		
		// Create HTTP request
		req, err := http.NewRequest(strings.ToUpper(method), requestURL, nil)
		if err != nil {
			return nil, h.CreateError(fmt.Sprintf("failed to create request: %v", err), nil)
		}
		
		// Add headers if provided
		if headers, ok := evaluatedParams["headers"].(map[string]interface{}); ok {
			for key, value := range headers {
				if strValue, ok := value.(string); ok {
					req.Header.Set(key, strValue)
				}
			}
		}
		
		// Add body for POST, PUT, PATCH requests
		if method == "POST" || method == "PUT" || method == "PATCH" {
			if body, ok := evaluatedParams["body"].(string); ok && body != "" {
				req.Body = io.NopCloser(strings.NewReader(body))
				if req.Header.Get("Content-Type") == "" {
					req.Header.Set("Content-Type", "application/json")
				}
			} else if body, ok := evaluatedParams["body"].(map[string]interface{}); ok {
				// Convert map to JSON
				jsonBody, err := json.Marshal(body)
				if err != nil {
					return nil, h.CreateError(fmt.Sprintf("failed to marshal body: %v", err), nil)
				}
				req.Body = io.NopCloser(bytes.NewReader(jsonBody))
				if req.Header.Get("Content-Type") == "" {
					req.Header.Set("Content-Type", "application/json")
				}
			}
		}
		
		// Set default user agent if not provided
		if req.Header.Get("User-Agent") == "" {
			req.Header.Set("User-Agent", "m9m/1.0")
		}
		
		// Execute request
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		req = req.WithContext(ctx)
		
		resp, err := h.client.Do(req)
		if err != nil {
			return nil, h.CreateError(fmt.Sprintf("request failed: %v", err), nil)
		}
		defer resp.Body.Close()
		
		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, h.CreateError(fmt.Sprintf("failed to read response body: %v", err), nil)
		}
		
		// Parse response headers
		headers := make(map[string]interface{})
		for key, values := range resp.Header {
			if len(values) > 0 {
				headers[key] = values[0] // Take first value
			}
		}
		
		// Create response data item
		responseData := model.DataItem{
			JSON: map[string]interface{}{
				"statusCode": resp.StatusCode,
				"headers":    headers,
				"body":       string(body),
			},
		}
		
		// Try to parse JSON response
		if strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
			var jsonResponse interface{}
			if err := json.Unmarshal(body, &jsonResponse); err == nil {
				responseData.JSON["json"] = jsonResponse
			}
		}
		
		// Copy binary data if present
		if item.Binary != nil {
			responseData.Binary = make(map[string]model.BinaryData)
			for k, v := range item.Binary {
				responseData.Binary[k] = v
			}
		}
		
		// Copy paired item data if present
		if item.PairedItem != nil {
			responseData.PairedItem = item.PairedItem
		}
		
		result[i] = responseData
	}
	
	return result, nil
}

// evaluateParameters evaluates node parameters with expression context
func (h *HTTPRequestNode) evaluateParameters(params map[string]interface{}, context *expressions.ExecutionContext) (map[string]interface{}, error) {
	evaluator := expressions.NewExpressionEvaluator()
	
	evaluatedParams := make(map[string]interface{})
	
	for key, value := range params {
		switch v := value.(type) {
		case string:
			// Check if this is an expression
			if expressions.IsExpression(v) {
				// Evaluate the expression
				evaluatedValue, err := evaluator.Evaluate(v, context)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate expression for parameter %s: %v", key, err)
				}
				evaluatedParams[key] = evaluatedValue
			} else {
				// Use the literal value
				evaluatedParams[key] = v
			}
		case map[string]interface{}:
			// Recursively evaluate nested maps
			nested, err := h.evaluateParameters(v, context)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate nested parameters for %s: %v", key, err)
			}
			evaluatedParams[key] = nested
		default:
			// Use the literal value
			evaluatedParams[key] = v
		}
	}
	
	return evaluatedParams, nil
}