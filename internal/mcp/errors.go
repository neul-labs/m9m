package mcp

import "fmt"

// Predefined errors for common scenarios
var (
	ErrParseError     = NewError(ErrCodeParseError, "Parse error", nil)
	ErrInvalidRequest = NewError(ErrCodeInvalidRequest, "Invalid request", nil)
	ErrMethodNotFound = NewError(ErrCodeMethodNotFound, "Method not found", nil)
	ErrInvalidParams  = NewError(ErrCodeInvalidParams, "Invalid params", nil)
	ErrInternal       = NewError(ErrCodeInternal, "Internal error", nil)
)

// WorkflowNotFoundError creates a workflow not found error
func WorkflowNotFoundError(id string) *Error {
	return NewError(ErrCodeWorkflowNotFound, fmt.Sprintf("Workflow not found: %s", id), map[string]string{"workflowId": id})
}

// ExecutionFailedError creates an execution failed error
func ExecutionFailedError(id string, reason string) *Error {
	return NewError(ErrCodeExecutionFailed, fmt.Sprintf("Execution failed: %s", reason), map[string]string{"executionId": id, "reason": reason})
}

// CredentialDeniedError creates a credential access denied error
func CredentialDeniedError(id string) *Error {
	return NewError(ErrCodeCredentialDenied, "Credential access denied", map[string]string{"credentialId": id})
}

// ValidationFailedError creates a validation error
func ValidationFailedError(field string, message string) *Error {
	return NewError(ErrCodeValidationFailed, fmt.Sprintf("Validation failed for %s: %s", field, message), map[string]string{"field": field, "message": message})
}

// PluginError creates a plugin-related error
func PluginError(name string, message string) *Error {
	return NewError(ErrCodePluginError, fmt.Sprintf("Plugin error for %s: %s", name, message), map[string]string{"plugin": name, "message": message})
}

// ResourceNotFoundError creates a resource not found error
func ResourceNotFoundError(uri string) *Error {
	return NewError(ErrCodeResourceNotFound, fmt.Sprintf("Resource not found: %s", uri), map[string]string{"uri": uri})
}

// ScheduleError creates a schedule-related error
func ScheduleError(id string, message string) *Error {
	return NewError(ErrCodeScheduleError, fmt.Sprintf("Schedule error: %s", message), map[string]string{"scheduleId": id, "message": message})
}

// NodeNotFoundError creates a node not found error
func NodeNotFoundError(nodeType string) *Error {
	return NewError(ErrCodeNodeNotFound, fmt.Sprintf("Node type not found: %s", nodeType), map[string]string{"nodeType": nodeType})
}

// TimeoutError creates a timeout error
func TimeoutError(operation string) *Error {
	return NewError(ErrCodeTimeout, fmt.Sprintf("Operation timed out: %s", operation), map[string]string{"operation": operation})
}

// InvalidParamsError creates an invalid params error with details
func InvalidParamsError(message string) *Error {
	return NewError(ErrCodeInvalidParams, message, nil)
}
