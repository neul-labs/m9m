// Package api provides HTTP API functionality
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// ValidatorConfig configures the request validator
type ValidatorConfig struct {
	// Maximum request body size in bytes
	MaxBodySize int64

	// Allowed content types
	AllowedContentTypes []string

	// Enable strict content type checking
	StrictContentType bool

	// Enable JSON validation
	ValidateJSON bool

	// Custom validators per endpoint
	EndpointValidators map[string]EndpointValidator
}

// EndpointValidator defines validation rules for a specific endpoint
type EndpointValidator struct {
	// Required fields
	RequiredFields []string

	// Field validators
	Fields map[string]FieldValidator

	// Custom validation function
	CustomValidator func(body map[string]interface{}) error
}

// FieldValidator defines validation rules for a field
type FieldValidator struct {
	Type      string // string, number, boolean, array, object
	Required  bool
	MinLength int
	MaxLength int
	MinValue  float64
	MaxValue  float64
	Pattern   string
	Enum      []interface{}
}

// DefaultValidatorConfig returns sensible defaults
func DefaultValidatorConfig() *ValidatorConfig {
	return &ValidatorConfig{
		MaxBodySize: 10 * 1024 * 1024, // 10MB
		AllowedContentTypes: []string{
			"application/json",
			"application/json; charset=utf-8",
			"application/json;charset=utf-8",
			"application/json; charset=UTF-8",
		},
		StrictContentType: true,
		ValidateJSON:      true,
		EndpointValidators: map[string]EndpointValidator{
			"POST /api/v1/workflows": {
				RequiredFields: []string{"name"},
				Fields: map[string]FieldValidator{
					"name": {
						Type:      "string",
						Required:  true,
						MinLength: 1,
						MaxLength: 255,
					},
					"active": {
						Type: "boolean",
					},
				},
			},
			"PUT /api/v1/workflows/{id}": {
				Fields: map[string]FieldValidator{
					"name": {
						Type:      "string",
						MinLength: 1,
						MaxLength: 255,
					},
				},
			},
			"POST /api/v1/credentials": {
				RequiredFields: []string{"name", "type"},
				Fields: map[string]FieldValidator{
					"name": {
						Type:      "string",
						Required:  true,
						MinLength: 1,
						MaxLength: 255,
					},
					"type": {
						Type:     "string",
						Required: true,
					},
				},
			},
			"POST /api/v1/expressions/evaluate": {
				RequiredFields: []string{"expression"},
				Fields: map[string]FieldValidator{
					"expression": {
						Type:     "string",
						Required: true,
					},
				},
			},
		},
	}
}

// RequestValidator provides request validation functionality
type RequestValidator struct {
	config       *ValidatorConfig
	errorHandler *ErrorHandler
}

// NewRequestValidator creates a new request validator
func NewRequestValidator(config *ValidatorConfig, errorHandler *ErrorHandler) *RequestValidator {
	if config == nil {
		config = DefaultValidatorConfig()
	}
	return &RequestValidator{
		config:       config,
		errorHandler: errorHandler,
	}
}

// Middleware returns an HTTP middleware for request validation
func (rv *RequestValidator) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip validation for GET, HEAD, OPTIONS
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// Validate request
			if err := rv.ValidateRequest(r); err != nil {
				requestID := r.Header.Get("X-Request-ID")
				rv.errorHandler.HandleAPIError(w, r, err, requestID)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ValidateRequest validates an HTTP request
func (rv *RequestValidator) ValidateRequest(r *http.Request) *APIError {
	// Check content length
	if r.ContentLength > rv.config.MaxBodySize {
		return ErrPayloadTooLarge(rv.config.MaxBodySize)
	}

	// Check content type
	if rv.config.StrictContentType && r.ContentLength > 0 {
		contentType := r.Header.Get("Content-Type")
		if !rv.isAllowedContentType(contentType) {
			return ErrUnsupportedMediaType(contentType)
		}
	}

	return nil
}

// ValidateBody validates the request body against endpoint rules
func (rv *RequestValidator) ValidateBody(r *http.Request, body map[string]interface{}) *APIError {
	// Build endpoint key
	endpointKey := rv.getEndpointKey(r)

	validator, ok := rv.config.EndpointValidators[endpointKey]
	if !ok {
		// Try without path parameters
		endpointKey = rv.normalizeEndpoint(r)
		validator, ok = rv.config.EndpointValidators[endpointKey]
		if !ok {
			return nil // No validation rules for this endpoint
		}
	}

	// Check required fields
	for _, field := range validator.RequiredFields {
		if _, exists := body[field]; !exists {
			return ErrValidation(fmt.Sprintf("Field '%s' is required", field))
		}
	}

	// Validate fields
	fieldErrors := make(map[string]string)
	for fieldName, fieldValidator := range validator.Fields {
		value, exists := body[fieldName]

		if fieldValidator.Required && !exists {
			fieldErrors[fieldName] = "This field is required"
			continue
		}

		if !exists {
			continue
		}

		if err := rv.validateField(fieldName, value, fieldValidator); err != "" {
			fieldErrors[fieldName] = err
		}
	}

	if len(fieldErrors) > 0 {
		return ErrValidationFields(fieldErrors)
	}

	// Run custom validator
	if validator.CustomValidator != nil {
		if err := validator.CustomValidator(body); err != nil {
			return ErrValidation(err.Error())
		}
	}

	return nil
}

// validateField validates a single field value
func (rv *RequestValidator) validateField(name string, value interface{}, validator FieldValidator) string {
	// Type validation
	switch validator.Type {
	case "string":
		str, ok := value.(string)
		if !ok {
			return "Must be a string"
		}
		if validator.MinLength > 0 && len(str) < validator.MinLength {
			return fmt.Sprintf("Must be at least %d characters", validator.MinLength)
		}
		if validator.MaxLength > 0 && len(str) > validator.MaxLength {
			return fmt.Sprintf("Must be at most %d characters", validator.MaxLength)
		}
		if validator.Pattern != "" {
			matched, _ := regexp.MatchString(validator.Pattern, str)
			if !matched {
				return "Invalid format"
			}
		}

	case "number":
		num, ok := toFloat64(value)
		if !ok {
			return "Must be a number"
		}
		if validator.MinValue != 0 && num < validator.MinValue {
			return fmt.Sprintf("Must be at least %v", validator.MinValue)
		}
		if validator.MaxValue != 0 && num > validator.MaxValue {
			return fmt.Sprintf("Must be at most %v", validator.MaxValue)
		}

	case "boolean":
		if _, ok := value.(bool); !ok {
			return "Must be a boolean"
		}

	case "array":
		if _, ok := value.([]interface{}); !ok {
			return "Must be an array"
		}

	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return "Must be an object"
		}
	}

	// Enum validation
	if len(validator.Enum) > 0 {
		found := false
		for _, enumValue := range validator.Enum {
			if value == enumValue {
				found = true
				break
			}
		}
		if !found {
			return "Invalid value"
		}
	}

	return ""
}

// isAllowedContentType checks if the content type is allowed
func (rv *RequestValidator) isAllowedContentType(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	for _, allowed := range rv.config.AllowedContentTypes {
		if strings.ToLower(allowed) == contentType {
			return true
		}
	}
	// Also check if it starts with application/json
	if strings.HasPrefix(contentType, "application/json") {
		return true
	}
	return false
}

// getEndpointKey builds a key for endpoint lookup
func (rv *RequestValidator) getEndpointKey(r *http.Request) string {
	return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
}

// normalizeEndpoint normalizes the endpoint path (replaces IDs with placeholders)
func (rv *RequestValidator) normalizeEndpoint(r *http.Request) string {
	path := r.URL.Path

	// Replace UUIDs and numeric IDs with {id}
	uuidPattern := regexp.MustCompile(`/[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
	path = uuidPattern.ReplaceAllString(path, "/{id}")

	numericPattern := regexp.MustCompile(`/\d+`)
	path = numericPattern.ReplaceAllString(path, "/{id}")

	return fmt.Sprintf("%s %s", r.Method, path)
}

// toFloat64 converts a value to float64
func toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	default:
		return 0, false
	}
}

// JSONValidator provides JSON-specific validation
type JSONValidator struct {
	maxDepth     int
	maxKeyLen    int
	maxStringLen int
}

// NewJSONValidator creates a new JSON validator
func NewJSONValidator() *JSONValidator {
	return &JSONValidator{
		maxDepth:     32,
		maxKeyLen:    256,
		maxStringLen: 1024 * 1024, // 1MB max string
	}
}

// ValidateJSON validates JSON structure
func (jv *JSONValidator) ValidateJSON(data []byte) error {
	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return jv.validateValue(result, 0)
}

func (jv *JSONValidator) validateValue(value interface{}, depth int) error {
	if depth > jv.maxDepth {
		return fmt.Errorf("JSON nesting too deep (max %d)", jv.maxDepth)
	}

	switch v := value.(type) {
	case map[string]interface{}:
		for key, val := range v {
			if len(key) > jv.maxKeyLen {
				return fmt.Errorf("JSON key too long (max %d)", jv.maxKeyLen)
			}
			if err := jv.validateValue(val, depth+1); err != nil {
				return err
			}
		}
	case []interface{}:
		for _, val := range v {
			if err := jv.validateValue(val, depth+1); err != nil {
				return err
			}
		}
	case string:
		if len(v) > jv.maxStringLen {
			return fmt.Errorf("JSON string too long (max %d)", jv.maxStringLen)
		}
	}

	return nil
}

// LimitedReader wraps a reader with a size limit
type LimitedReader struct {
	R io.Reader
	N int64
}

// Read implements io.Reader with size limiting
func (l *LimitedReader) Read(p []byte) (n int, err error) {
	if l.N <= 0 {
		return 0, ErrPayloadTooLarge(0)
	}
	if int64(len(p)) > l.N {
		p = p[:l.N]
	}
	n, err = l.R.Read(p)
	l.N -= int64(n)
	return
}

// SanitizeInput sanitizes string input to prevent injection attacks
func SanitizeInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Trim whitespace
	input = strings.TrimSpace(input)

	return input
}

// ValidateWorkflowName validates a workflow name
func ValidateWorkflowName(name string) error {
	if name == "" {
		return fmt.Errorf("workflow name is required")
	}
	if len(name) > 255 {
		return fmt.Errorf("workflow name must be at most 255 characters")
	}
	// Check for invalid characters
	if strings.ContainsAny(name, "<>\"'") {
		return fmt.Errorf("workflow name contains invalid characters")
	}
	return nil
}

// ValidateEmail validates an email address format
func ValidateEmail(email string) bool {
	pattern := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return pattern.MatchString(email)
}

// ValidateURL validates a URL format
func ValidateURL(url string) bool {
	pattern := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	return pattern.MatchString(url)
}

// ValidateCronExpression validates a cron expression (basic check)
func ValidateCronExpression(cron string) bool {
	parts := strings.Fields(cron)
	// Standard cron has 5 parts, some implementations have 6 (with seconds)
	return len(parts) >= 5 && len(parts) <= 6
}
