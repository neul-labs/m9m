// Package audit provides audit logging functionality for tracking user actions and system events
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// EventType represents the type of audit event
type EventType string

const (
	// Authentication events
	EventTypeLogin          EventType = "auth.login"
	EventTypeLoginFailed    EventType = "auth.login_failed"
	EventTypeLogout         EventType = "auth.logout"
	EventTypeTokenRefresh   EventType = "auth.token_refresh"
	EventTypePasswordChange EventType = "auth.password_change"
	EventTypeAPIKeyCreated  EventType = "auth.api_key_created"
	EventTypeAPIKeyDeleted  EventType = "auth.api_key_deleted"

	// User management events
	EventTypeUserCreated  EventType = "user.created"
	EventTypeUserUpdated  EventType = "user.updated"
	EventTypeUserDeleted  EventType = "user.deleted"
	EventTypeUserDisabled EventType = "user.disabled"

	// Workflow events
	EventTypeWorkflowCreated   EventType = "workflow.created"
	EventTypeWorkflowUpdated   EventType = "workflow.updated"
	EventTypeWorkflowDeleted   EventType = "workflow.deleted"
	EventTypeWorkflowActivated EventType = "workflow.activated"
	EventTypeWorkflowDeactivated EventType = "workflow.deactivated"
	EventTypeWorkflowDuplicated  EventType = "workflow.duplicated"
	EventTypeWorkflowExported    EventType = "workflow.exported"
	EventTypeWorkflowImported    EventType = "workflow.imported"

	// Execution events
	EventTypeExecutionStarted   EventType = "execution.started"
	EventTypeExecutionCompleted EventType = "execution.completed"
	EventTypeExecutionFailed    EventType = "execution.failed"
	EventTypeExecutionCancelled EventType = "execution.cancelled"

	// Credential events
	EventTypeCredentialCreated EventType = "credential.created"
	EventTypeCredentialUpdated EventType = "credential.updated"
	EventTypeCredentialDeleted EventType = "credential.deleted"
	EventTypeCredentialAccessed EventType = "credential.accessed"

	// Settings events
	EventTypeSettingsUpdated EventType = "settings.updated"
	EventTypeLicenseUpdated  EventType = "settings.license_updated"

	// System events
	EventTypeSystemStartup  EventType = "system.startup"
	EventTypeSystemShutdown EventType = "system.shutdown"
	EventTypeSystemError    EventType = "system.error"
	EventTypeMaintenanceMode EventType = "system.maintenance_mode"

	// Webhook events
	EventTypeWebhookRegistered   EventType = "webhook.registered"
	EventTypeWebhookUnregistered EventType = "webhook.unregistered"
	EventTypeWebhookTriggered    EventType = "webhook.triggered"

	// Security events
	EventTypeSecurityRateLimited EventType = "security.rate_limited"
	EventTypeSecurityBlocked     EventType = "security.blocked"
	EventTypeSecuritySuspicious  EventType = "security.suspicious"
)

// EventSeverity represents the severity of an audit event
type EventSeverity string

const (
	SeverityInfo     EventSeverity = "info"
	SeverityWarning  EventSeverity = "warning"
	SeverityError    EventSeverity = "error"
	SeverityCritical EventSeverity = "critical"
)

// AuditEvent represents a single audit log entry
type AuditEvent struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	EventType   EventType              `json:"eventType"`
	Severity    EventSeverity          `json:"severity"`
	UserID      string                 `json:"userId,omitempty"`
	UserEmail   string                 `json:"userEmail,omitempty"`
	UserIP      string                 `json:"userIp,omitempty"`
	UserAgent   string                 `json:"userAgent,omitempty"`
	RequestID   string                 `json:"requestId,omitempty"`
	ResourceType string                `json:"resourceType,omitempty"`
	ResourceID   string                `json:"resourceId,omitempty"`
	Action       string                `json:"action,omitempty"`
	Description  string                `json:"description,omitempty"`
	OldValue     interface{}           `json:"oldValue,omitempty"`
	NewValue     interface{}           `json:"newValue,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Success      bool                  `json:"success"`
	ErrorMessage string                `json:"errorMessage,omitempty"`
}

// AuditLoggerConfig configures the audit logger
type AuditLoggerConfig struct {
	// Enable/disable audit logging
	Enabled bool

	// Output destination
	Output io.Writer

	// File path for file-based logging
	FilePath string

	// Log format (json, text)
	Format string

	// Events to log (empty = all events)
	EnabledEvents []EventType

	// Events to exclude from logging
	ExcludedEvents []EventType

	// Minimum severity to log
	MinSeverity EventSeverity

	// Include sensitive data in logs
	IncludeSensitive bool

	// Async logging
	AsyncEnabled bool
	BufferSize   int

	// Retention settings
	RetentionDays int
}

// DefaultAuditLoggerConfig returns sensible defaults
func DefaultAuditLoggerConfig() *AuditLoggerConfig {
	return &AuditLoggerConfig{
		Enabled:          true,
		Output:           os.Stdout,
		Format:           "json",
		MinSeverity:      SeverityInfo,
		IncludeSensitive: false,
		AsyncEnabled:     true,
		BufferSize:       1000,
		RetentionDays:    90,
		ExcludedEvents: []EventType{
			EventTypeExecutionStarted,   // Too verbose
			EventTypeExecutionCompleted, // Too verbose
		},
	}
}

// AuditLogger provides audit logging functionality
type AuditLogger struct {
	config      *AuditLoggerConfig
	output      io.Writer
	file        *os.File
	eventBuffer chan *AuditEvent
	wg          sync.WaitGroup
	stopCh      chan struct{}
	mu          sync.Mutex
	eventID     int64
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(config *AuditLoggerConfig) (*AuditLogger, error) {
	if config == nil {
		config = DefaultAuditLoggerConfig()
	}

	logger := &AuditLogger{
		config:      config,
		output:      config.Output,
		eventBuffer: make(chan *AuditEvent, config.BufferSize),
		stopCh:      make(chan struct{}),
	}

	// Open file if specified
	if config.FilePath != "" {
		file, err := os.OpenFile(config.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open audit log file: %w", err)
		}
		logger.file = file
		logger.output = file
	}

	// Start async writer if enabled
	if config.AsyncEnabled {
		logger.wg.Add(1)
		go logger.asyncWriter()
	}

	return logger, nil
}

// Log logs an audit event
func (l *AuditLogger) Log(event *AuditEvent) {
	if !l.config.Enabled {
		return
	}

	// Check if event type is excluded
	if l.isExcluded(event.EventType) {
		return
	}

	// Check minimum severity
	if !l.meetsSeverity(event.Severity) {
		return
	}

	// Set ID and timestamp
	event.ID = l.nextEventID()
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Remove sensitive data if configured
	if !l.config.IncludeSensitive {
		event = l.sanitizeEvent(event)
	}

	// Log synchronously or asynchronously
	if l.config.AsyncEnabled {
		select {
		case l.eventBuffer <- event:
		default:
			// Buffer full, log synchronously
			l.writeEvent(event)
		}
	} else {
		l.writeEvent(event)
	}
}

// LogContext logs an audit event with context
func (l *AuditLogger) LogContext(ctx context.Context, event *AuditEvent) {
	// Extract user info from context if available
	if userID, ok := ctx.Value("userID").(string); ok && event.UserID == "" {
		event.UserID = userID
	}
	if requestID, ok := ctx.Value("requestID").(string); ok && event.RequestID == "" {
		event.RequestID = requestID
	}

	l.Log(event)
}

// Helper methods for common events

// LogLogin logs a successful login
func (l *AuditLogger) LogLogin(userID, userEmail, userIP, userAgent string) {
	l.Log(&AuditEvent{
		EventType:   EventTypeLogin,
		Severity:    SeverityInfo,
		UserID:      userID,
		UserEmail:   userEmail,
		UserIP:      userIP,
		UserAgent:   userAgent,
		Action:      "login",
		Description: "User logged in successfully",
		Success:     true,
	})
}

// LogLoginFailed logs a failed login attempt
func (l *AuditLogger) LogLoginFailed(userEmail, userIP, userAgent, reason string) {
	l.Log(&AuditEvent{
		EventType:    EventTypeLoginFailed,
		Severity:     SeverityWarning,
		UserEmail:    userEmail,
		UserIP:       userIP,
		UserAgent:    userAgent,
		Action:       "login_failed",
		Description:  "Login attempt failed",
		Success:      false,
		ErrorMessage: reason,
	})
}

// LogWorkflowCreated logs workflow creation
func (l *AuditLogger) LogWorkflowCreated(userID, workflowID, workflowName string) {
	l.Log(&AuditEvent{
		EventType:    EventTypeWorkflowCreated,
		Severity:     SeverityInfo,
		UserID:       userID,
		ResourceType: "workflow",
		ResourceID:   workflowID,
		Action:       "create",
		Description:  fmt.Sprintf("Workflow '%s' created", workflowName),
		Success:      true,
		Metadata: map[string]interface{}{
			"workflowName": workflowName,
		},
	})
}

// LogWorkflowUpdated logs workflow update
func (l *AuditLogger) LogWorkflowUpdated(userID, workflowID, workflowName string, changes map[string]interface{}) {
	l.Log(&AuditEvent{
		EventType:    EventTypeWorkflowUpdated,
		Severity:     SeverityInfo,
		UserID:       userID,
		ResourceType: "workflow",
		ResourceID:   workflowID,
		Action:       "update",
		Description:  fmt.Sprintf("Workflow '%s' updated", workflowName),
		NewValue:     changes,
		Success:      true,
	})
}

// LogWorkflowDeleted logs workflow deletion
func (l *AuditLogger) LogWorkflowDeleted(userID, workflowID, workflowName string) {
	l.Log(&AuditEvent{
		EventType:    EventTypeWorkflowDeleted,
		Severity:     SeverityInfo,
		UserID:       userID,
		ResourceType: "workflow",
		ResourceID:   workflowID,
		Action:       "delete",
		Description:  fmt.Sprintf("Workflow '%s' deleted", workflowName),
		Success:      true,
	})
}

// LogExecutionFailed logs execution failure
func (l *AuditLogger) LogExecutionFailed(workflowID, executionID, errorMsg string) {
	l.Log(&AuditEvent{
		EventType:    EventTypeExecutionFailed,
		Severity:     SeverityError,
		ResourceType: "execution",
		ResourceID:   executionID,
		Action:       "execute",
		Description:  "Workflow execution failed",
		Success:      false,
		ErrorMessage: errorMsg,
		Metadata: map[string]interface{}{
			"workflowId": workflowID,
		},
	})
}

// LogCredentialAccessed logs credential access
func (l *AuditLogger) LogCredentialAccessed(userID, credentialID, credentialName, purpose string) {
	l.Log(&AuditEvent{
		EventType:    EventTypeCredentialAccessed,
		Severity:     SeverityInfo,
		UserID:       userID,
		ResourceType: "credential",
		ResourceID:   credentialID,
		Action:       "access",
		Description:  fmt.Sprintf("Credential '%s' accessed", credentialName),
		Success:      true,
		Metadata: map[string]interface{}{
			"purpose": purpose,
		},
	})
}

// LogRateLimited logs rate limiting events
func (l *AuditLogger) LogRateLimited(userIP, endpoint string) {
	l.Log(&AuditEvent{
		EventType:   EventTypeSecurityRateLimited,
		Severity:    SeverityWarning,
		UserIP:      userIP,
		Action:      "rate_limit",
		Description: fmt.Sprintf("Rate limit exceeded for endpoint %s", endpoint),
		Success:     false,
		Metadata: map[string]interface{}{
			"endpoint": endpoint,
		},
	})
}

// LogSystemStartup logs system startup
func (l *AuditLogger) LogSystemStartup(version string, config map[string]interface{}) {
	l.Log(&AuditEvent{
		EventType:   EventTypeSystemStartup,
		Severity:    SeverityInfo,
		Action:      "startup",
		Description: "System started",
		Success:     true,
		Metadata: map[string]interface{}{
			"version": version,
			"config":  config,
		},
	})
}

// LogSystemShutdown logs system shutdown
func (l *AuditLogger) LogSystemShutdown(reason string) {
	l.Log(&AuditEvent{
		EventType:   EventTypeSystemShutdown,
		Severity:    SeverityInfo,
		Action:      "shutdown",
		Description: fmt.Sprintf("System shutdown: %s", reason),
		Success:     true,
	})
}

// LogSystemError logs system errors
func (l *AuditLogger) LogSystemError(err error, context string) {
	l.Log(&AuditEvent{
		EventType:    EventTypeSystemError,
		Severity:     SeverityCritical,
		Action:       "error",
		Description:  context,
		Success:      false,
		ErrorMessage: err.Error(),
	})
}

// Internal methods

func (l *AuditLogger) nextEventID() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.eventID++
	return fmt.Sprintf("audit-%d-%d", time.Now().Unix(), l.eventID)
}

func (l *AuditLogger) isExcluded(eventType EventType) bool {
	for _, excluded := range l.config.ExcludedEvents {
		if excluded == eventType {
			return true
		}
	}
	// Check if enabled events is specified
	if len(l.config.EnabledEvents) > 0 {
		for _, enabled := range l.config.EnabledEvents {
			if enabled == eventType {
				return false
			}
		}
		return true
	}
	return false
}

func (l *AuditLogger) meetsSeverity(severity EventSeverity) bool {
	severityOrder := map[EventSeverity]int{
		SeverityInfo:     0,
		SeverityWarning:  1,
		SeverityError:    2,
		SeverityCritical: 3,
	}
	return severityOrder[severity] >= severityOrder[l.config.MinSeverity]
}

func (l *AuditLogger) sanitizeEvent(event *AuditEvent) *AuditEvent {
	// Create a copy
	sanitized := *event

	// Remove sensitive fields from metadata
	if sanitized.Metadata != nil {
		sanitized.Metadata = make(map[string]interface{})
		for k, v := range event.Metadata {
			if !isSensitiveKey(k) {
				sanitized.Metadata[k] = v
			}
		}
	}

	return &sanitized
}

func isSensitiveKey(key string) bool {
	sensitiveKeys := []string{
		"password", "secret", "token", "apiKey", "api_key",
		"credential", "privateKey", "private_key", "accessToken",
		"access_token", "refreshToken", "refresh_token",
	}
	for _, sensitive := range sensitiveKeys {
		if key == sensitive {
			return true
		}
	}
	return false
}

func (l *AuditLogger) asyncWriter() {
	defer l.wg.Done()

	for {
		select {
		case event := <-l.eventBuffer:
			l.writeEvent(event)
		case <-l.stopCh:
			// Drain remaining events
			for {
				select {
				case event := <-l.eventBuffer:
					l.writeEvent(event)
				default:
					return
				}
			}
		}
	}
}

func (l *AuditLogger) writeEvent(event *AuditEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()

	var output []byte
	var err error

	switch l.config.Format {
	case "json":
		output, err = json.Marshal(event)
		if err != nil {
			return
		}
		output = append(output, '\n')
	case "text":
		output = []byte(fmt.Sprintf(
			"[%s] %s %s user=%s resource=%s/%s action=%s success=%t %s\n",
			event.Timestamp.Format(time.RFC3339),
			event.Severity,
			event.EventType,
			event.UserID,
			event.ResourceType,
			event.ResourceID,
			event.Action,
			event.Success,
			event.Description,
		))
	default:
		output, _ = json.Marshal(event)
		output = append(output, '\n')
	}

	l.output.Write(output)
}

// Close closes the audit logger
func (l *AuditLogger) Close() error {
	// Signal async writer to stop
	close(l.stopCh)

	// Wait for async writer to finish
	l.wg.Wait()

	// Close file if opened
	if l.file != nil {
		return l.file.Close()
	}

	return nil
}

// Flush ensures all buffered events are written
func (l *AuditLogger) Flush() {
	if !l.config.AsyncEnabled {
		return
	}

	// Wait for buffer to drain
	for len(l.eventBuffer) > 0 {
		time.Sleep(10 * time.Millisecond)
	}
}

// Query searches audit logs (for storage backends that support it)
type AuditQuery struct {
	StartTime    time.Time
	EndTime      time.Time
	EventTypes   []EventType
	UserID       string
	ResourceType string
	ResourceID   string
	Severity     EventSeverity
	Success      *bool
	Limit        int
	Offset       int
}

// AuditStorage interface for persistent audit storage
type AuditStorage interface {
	Store(event *AuditEvent) error
	Query(query *AuditQuery) ([]*AuditEvent, error)
	Delete(before time.Time) error
}
