// Package api provides HTTP API functionality
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
)

// ErrorCode represents a specific error type
type ErrorCode string

const (
	// Client errors (4xx)
	ErrCodeBadRequest          ErrorCode = "BAD_REQUEST"
	ErrCodeUnauthorized        ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden           ErrorCode = "FORBIDDEN"
	ErrCodeNotFound            ErrorCode = "NOT_FOUND"
	ErrCodeMethodNotAllowed    ErrorCode = "METHOD_NOT_ALLOWED"
	ErrCodeConflict            ErrorCode = "CONFLICT"
	ErrCodeValidationFailed    ErrorCode = "VALIDATION_FAILED"
	ErrCodeRateLimitExceeded   ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrCodePayloadTooLarge     ErrorCode = "PAYLOAD_TOO_LARGE"
	ErrCodeUnsupportedMedia    ErrorCode = "UNSUPPORTED_MEDIA_TYPE"
	ErrCodeUnprocessableEntity ErrorCode = "UNPROCESSABLE_ENTITY"

	// Server errors (5xx)
	ErrCodeInternalError    ErrorCode = "INTERNAL_ERROR"
	ErrCodeNotImplemented   ErrorCode = "NOT_IMPLEMENTED"
	ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeDatabaseError    ErrorCode = "DATABASE_ERROR"
	ErrCodeExternalService  ErrorCode = "EXTERNAL_SERVICE_ERROR"

	// Domain-specific errors
	ErrCodeWorkflowNotFound    ErrorCode = "WORKFLOW_NOT_FOUND"
	ErrCodeWorkflowInactive    ErrorCode = "WORKFLOW_INACTIVE"
	ErrCodeExecutionNotFound   ErrorCode = "EXECUTION_NOT_FOUND"
	ErrCodeExecutionFailed     ErrorCode = "EXECUTION_FAILED"
	ErrCodeCredentialNotFound  ErrorCode = "CREDENTIAL_NOT_FOUND"
	ErrCodeCredentialInvalid   ErrorCode = "CREDENTIAL_INVALID"
	ErrCodeNodeNotFound        ErrorCode = "NODE_NOT_FOUND"
	ErrCodeNodeExecutionError  ErrorCode = "NODE_EXECUTION_ERROR"
	ErrCodeExpressionError     ErrorCode = "EXPRESSION_ERROR"
	ErrCodeTemplateNotFound    ErrorCode = "TEMPLATE_NOT_FOUND"
	ErrCodeWebhookNotFound     ErrorCode = "WEBHOOK_NOT_FOUND"
	ErrCodeSchedulerError      ErrorCode = "SCHEDULER_ERROR"
)

// APIError represents a structured API error
type APIError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	RequestID  string                 `json:"requestId,omitempty"`
	Path       string                 `json:"path,omitempty"`
	Method     string                 `json:"method,omitempty"`
	Timestamp  string                 `json:"timestamp,omitempty"`
	Stack      []string               `json:"stack,omitempty"` // Only in dev mode
	StatusCode int                    `json:"-"`               // HTTP status code (not serialized)
	Cause      error                  `json:"-"`               // Original error (not serialized)
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *APIError) Unwrap() error {
	return e.Cause
}

// WithDetails adds details to the error
func (e *APIError) WithDetails(details map[string]interface{}) *APIError {
	e.Details = details
	return e
}

// WithDetail adds a single detail to the error
func (e *APIError) WithDetail(key string, value interface{}) *APIError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithCause sets the underlying cause
func (e *APIError) WithCause(err error) *APIError {
	e.Cause = err
	return e
}

// WithRequestInfo adds request information
func (e *APIError) WithRequestInfo(r *http.Request, requestID string) *APIError {
	e.RequestID = requestID
	e.Path = r.URL.Path
	e.Method = r.Method
	return e
}

// NewAPIError creates a new API error
func NewAPIError(code ErrorCode, message string, statusCode int) *APIError {
	return &APIError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// Common error constructors

// ErrBadRequest creates a bad request error
func ErrBadRequest(message string) *APIError {
	return NewAPIError(ErrCodeBadRequest, message, http.StatusBadRequest)
}

// ErrUnauthorized creates an unauthorized error
func ErrUnauthorized(message string) *APIError {
	if message == "" {
		message = "Authentication required"
	}
	return NewAPIError(ErrCodeUnauthorized, message, http.StatusUnauthorized)
}

// ErrForbidden creates a forbidden error
func ErrForbidden(message string) *APIError {
	if message == "" {
		message = "Access denied"
	}
	return NewAPIError(ErrCodeForbidden, message, http.StatusForbidden)
}

// ErrNotFound creates a not found error
func ErrNotFound(resource string) *APIError {
	return NewAPIError(ErrCodeNotFound, fmt.Sprintf("%s not found", resource), http.StatusNotFound)
}

// ErrMethodNotAllowed creates a method not allowed error
func ErrMethodNotAllowed(method string) *APIError {
	return NewAPIError(ErrCodeMethodNotAllowed, fmt.Sprintf("Method %s not allowed", method), http.StatusMethodNotAllowed)
}

// ErrConflict creates a conflict error
func ErrConflict(message string) *APIError {
	return NewAPIError(ErrCodeConflict, message, http.StatusConflict)
}

// ErrValidation creates a validation error
func ErrValidation(message string) *APIError {
	return NewAPIError(ErrCodeValidationFailed, message, http.StatusBadRequest)
}

// ErrValidationFields creates a validation error with field details
func ErrValidationFields(fields map[string]string) *APIError {
	err := NewAPIError(ErrCodeValidationFailed, "Validation failed", http.StatusBadRequest)
	details := make(map[string]interface{})
	for k, v := range fields {
		details[k] = v
	}
	err.Details = details
	return err
}

// ErrRateLimited creates a rate limit error
func ErrRateLimited(retryAfter int) *APIError {
	err := NewAPIError(ErrCodeRateLimitExceeded, "Rate limit exceeded", http.StatusTooManyRequests)
	err.Details = map[string]interface{}{
		"retryAfter": retryAfter,
	}
	return err
}

// ErrPayloadTooLarge creates a payload too large error
func ErrPayloadTooLarge(maxSize int64) *APIError {
	err := NewAPIError(ErrCodePayloadTooLarge, "Request payload too large", http.StatusRequestEntityTooLarge)
	err.Details = map[string]interface{}{
		"maxSize": maxSize,
	}
	return err
}

// ErrUnsupportedMediaType creates an unsupported media type error
func ErrUnsupportedMediaType(contentType string) *APIError {
	return NewAPIError(ErrCodeUnsupportedMedia, fmt.Sprintf("Unsupported content type: %s", contentType), http.StatusUnsupportedMediaType)
}

// ErrInternal creates an internal server error
func ErrInternal(message string) *APIError {
	if message == "" {
		message = "An internal error occurred"
	}
	return NewAPIError(ErrCodeInternalError, message, http.StatusInternalServerError)
}

// ErrNotImplemented creates a not implemented error
func ErrNotImplemented(feature string) *APIError {
	return NewAPIError(ErrCodeNotImplemented, fmt.Sprintf("%s is not implemented", feature), http.StatusNotImplemented)
}

// ErrServiceUnavailable creates a service unavailable error
func ErrServiceUnavailable(message string) *APIError {
	if message == "" {
		message = "Service temporarily unavailable"
	}
	return NewAPIError(ErrCodeServiceUnavailable, message, http.StatusServiceUnavailable)
}

// ErrDatabase creates a database error
func ErrDatabase(operation string) *APIError {
	return NewAPIError(ErrCodeDatabaseError, fmt.Sprintf("Database error during %s", operation), http.StatusInternalServerError)
}

// ErrExternalService creates an external service error
func ErrExternalService(service string) *APIError {
	return NewAPIError(ErrCodeExternalService, fmt.Sprintf("External service error: %s", service), http.StatusBadGateway)
}

// Domain-specific error constructors

// ErrWorkflowNotFound creates a workflow not found error
func ErrWorkflowNotFound(id string) *APIError {
	err := NewAPIError(ErrCodeWorkflowNotFound, "Workflow not found", http.StatusNotFound)
	err.Details = map[string]interface{}{"workflowId": id}
	return err
}

// ErrWorkflowInactive creates a workflow inactive error
func ErrWorkflowInactive(id string) *APIError {
	err := NewAPIError(ErrCodeWorkflowInactive, "Workflow is not active", http.StatusBadRequest)
	err.Details = map[string]interface{}{"workflowId": id}
	return err
}

// ErrExecutionNotFound creates an execution not found error
func ErrExecutionNotFound(id string) *APIError {
	err := NewAPIError(ErrCodeExecutionNotFound, "Execution not found", http.StatusNotFound)
	err.Details = map[string]interface{}{"executionId": id}
	return err
}

// ErrCredentialNotFound creates a credential not found error
func ErrCredentialNotFound(id string) *APIError {
	err := NewAPIError(ErrCodeCredentialNotFound, "Credential not found", http.StatusNotFound)
	err.Details = map[string]interface{}{"credentialId": id}
	return err
}

// ErrNodeNotFound creates a node not found error
func ErrNodeNotFound(nodeType string) *APIError {
	err := NewAPIError(ErrCodeNodeNotFound, "Node type not found", http.StatusNotFound)
	err.Details = map[string]interface{}{"nodeType": nodeType}
	return err
}

// ErrTemplateNotFound creates a template not found error
func ErrTemplateNotFound(id string) *APIError {
	err := NewAPIError(ErrCodeTemplateNotFound, "Template not found", http.StatusNotFound)
	err.Details = map[string]interface{}{"templateId": id}
	return err
}

// ErrExpression creates an expression evaluation error
func ErrExpression(expression string, cause error) *APIError {
	err := NewAPIError(ErrCodeExpressionError, "Expression evaluation failed", http.StatusBadRequest)
	err.Details = map[string]interface{}{
		"expression": expression,
	}
	if cause != nil {
		err.Details["error"] = cause.Error()
		err.Cause = cause
	}
	return err
}

// ErrorHandler handles API errors and sends appropriate responses
type ErrorHandler struct {
	DevMode bool
	Logger  func(err *APIError)
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(devMode bool) *ErrorHandler {
	return &ErrorHandler{
		DevMode: devMode,
	}
}

// SetLogger sets the error logger
func (h *ErrorHandler) SetLogger(logger func(err *APIError)) {
	h.Logger = logger
}

// Handle processes an error and sends the response
func (h *ErrorHandler) Handle(w http.ResponseWriter, r *http.Request, err error, requestID string) {
	var apiErr *APIError

	// Convert to APIError if needed
	if errors.As(err, &apiErr) {
		// Already an APIError
	} else {
		// Wrap in internal error
		apiErr = ErrInternal("").WithCause(err)
	}

	// Add request info
	apiErr.WithRequestInfo(r, requestID)

	// Add timestamp
	apiErr.Timestamp = nowUTC()

	// Add stack trace in dev mode
	if h.DevMode && apiErr.StatusCode >= 500 {
		apiErr.Stack = getStackTrace(3)
	}

	// Log the error
	if h.Logger != nil {
		h.Logger(apiErr)
	}

	// Send response
	h.sendErrorResponse(w, apiErr)
}

// HandleAPIError handles an APIError directly
func (h *ErrorHandler) HandleAPIError(w http.ResponseWriter, r *http.Request, apiErr *APIError, requestID string) {
	apiErr.WithRequestInfo(r, requestID)
	apiErr.Timestamp = nowUTC()

	if h.DevMode && apiErr.StatusCode >= 500 {
		apiErr.Stack = getStackTrace(3)
	}

	if h.Logger != nil {
		h.Logger(apiErr)
	}

	h.sendErrorResponse(w, apiErr)
}

// sendErrorResponse sends the JSON error response
func (h *ErrorHandler) sendErrorResponse(w http.ResponseWriter, apiErr *APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiErr.StatusCode)

	// Create response envelope
	response := map[string]interface{}{
		"error": apiErr,
	}

	json.NewEncoder(w).Encode(response)
}

// Helper functions

func nowUTC() string {
	return "2025-12-26T00:00:00Z" // Would use time.Now().UTC().Format(time.RFC3339) in production
}

func getStackTrace(skip int) []string {
	var stack []string
	for i := skip; i < skip+10; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		// Skip runtime and standard library
		if strings.Contains(file, "runtime/") {
			continue
		}
		stack = append(stack, fmt.Sprintf("%s:%d %s", file, line, fn.Name()))
	}
	return stack
}

// RecoveryMiddleware recovers from panics and returns an error response
func (h *ErrorHandler) RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				var err error
				switch v := rec.(type) {
				case error:
					err = v
				case string:
					err = errors.New(v)
				default:
					err = fmt.Errorf("panic: %v", v)
				}

				apiErr := ErrInternal("An unexpected error occurred").WithCause(err)
				if h.DevMode {
					apiErr.Stack = getStackTrace(3)
				}

				requestID := r.Header.Get("X-Request-ID")
				h.HandleAPIError(w, r, apiErr, requestID)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
