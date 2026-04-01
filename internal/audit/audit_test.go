package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- DefaultAuditLoggerConfig ---

func TestDefaultAuditLoggerConfig(t *testing.T) {
	cfg := DefaultAuditLoggerConfig()

	assert.True(t, cfg.Enabled)
	assert.Equal(t, os.Stdout, cfg.Output)
	assert.Equal(t, "json", cfg.Format)
	assert.Equal(t, SeverityInfo, cfg.MinSeverity)
	assert.False(t, cfg.IncludeSensitive)
	assert.True(t, cfg.AsyncEnabled)
	assert.Equal(t, 1000, cfg.BufferSize)
	assert.Equal(t, 90, cfg.RetentionDays)
	assert.Contains(t, cfg.ExcludedEvents, EventTypeExecutionStarted)
	assert.Contains(t, cfg.ExcludedEvents, EventTypeExecutionCompleted)
}

// --- NewAuditLogger ---

func TestNewAuditLogger_NilConfig(t *testing.T) {
	logger, err := NewAuditLogger(nil)
	require.NoError(t, err)
	require.NotNil(t, logger)
	defer logger.Close()

	assert.True(t, logger.config.Enabled)
	assert.Equal(t, "json", logger.config.Format)
}

func TestNewAuditLogger_CustomConfig(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &AuditLoggerConfig{
		Enabled:      true,
		Output:       buf,
		Format:       "text",
		MinSeverity:  SeverityWarning,
		AsyncEnabled: false,
		BufferSize:   100,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	require.NotNil(t, logger)
	defer logger.Close()

	assert.Equal(t, "text", logger.config.Format)
	assert.Equal(t, SeverityWarning, logger.config.MinSeverity)
}

func TestNewAuditLogger_WithFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "audit.log")

	cfg := &AuditLoggerConfig{
		Enabled:      true,
		FilePath:     logFile,
		Format:       "json",
		MinSeverity:  SeverityInfo,
		AsyncEnabled: false,
		BufferSize:   10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	require.NotNil(t, logger)
	defer logger.Close()

	// Log an event
	logger.Log(&AuditEvent{
		EventType:   EventTypeSystemStartup,
		Severity:    SeverityInfo,
		Action:      "startup",
		Description: "test startup",
		Success:     true,
	})

	// Verify file was written
	data, err := os.ReadFile(logFile)
	require.NoError(t, err)
	assert.Contains(t, string(data), "system.startup")
}

func TestNewAuditLogger_InvalidFilePath(t *testing.T) {
	cfg := &AuditLoggerConfig{
		Enabled:      true,
		FilePath:     "/nonexistent/dir/audit.log",
		Format:       "json",
		AsyncEnabled: false,
		BufferSize:   10,
	}

	logger, err := NewAuditLogger(cfg)
	assert.Error(t, err)
	assert.Nil(t, logger)
	assert.Contains(t, err.Error(), "failed to open audit log file")
}

// --- Log ---

func TestLog_DisabledLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &AuditLoggerConfig{
		Enabled:      false,
		Output:       buf,
		Format:       "json",
		AsyncEnabled: false,
		BufferSize:   10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	logger.Log(&AuditEvent{
		EventType: EventTypeLogin,
		Severity:  SeverityInfo,
		Success:   true,
	})

	assert.Empty(t, buf.String())
}

func TestLog_ExcludedEvent(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &AuditLoggerConfig{
		Enabled:        true,
		Output:         buf,
		Format:         "json",
		ExcludedEvents: []EventType{EventTypeLogin},
		MinSeverity:    SeverityInfo,
		AsyncEnabled:   false,
		BufferSize:     10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	logger.Log(&AuditEvent{
		EventType: EventTypeLogin,
		Severity:  SeverityInfo,
		Success:   true,
	})

	assert.Empty(t, buf.String())
}

func TestLog_EnabledEventsFilter(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &AuditLoggerConfig{
		Enabled:       true,
		Output:        buf,
		Format:        "json",
		EnabledEvents: []EventType{EventTypeLogin},
		MinSeverity:   SeverityInfo,
		AsyncEnabled:  false,
		BufferSize:    10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	// This should NOT appear (not in EnabledEvents)
	logger.Log(&AuditEvent{
		EventType: EventTypeLogout,
		Severity:  SeverityInfo,
		Success:   true,
	})
	assert.Empty(t, buf.String())

	// This SHOULD appear (in EnabledEvents)
	logger.Log(&AuditEvent{
		EventType: EventTypeLogin,
		Severity:  SeverityInfo,
		Success:   true,
	})
	assert.NotEmpty(t, buf.String())
	assert.Contains(t, buf.String(), "auth.login")
}

func TestLog_MinSeverityFilter(t *testing.T) {
	tests := []struct {
		name        string
		minSeverity EventSeverity
		eventSev    EventSeverity
		shouldLog   bool
	}{
		{"info passes info", SeverityInfo, SeverityInfo, true},
		{"warning passes warning", SeverityWarning, SeverityWarning, true},
		{"error passes error", SeverityError, SeverityError, true},
		{"critical passes critical", SeverityCritical, SeverityCritical, true},
		{"warning passes error", SeverityWarning, SeverityError, true},
		{"error blocks info", SeverityError, SeverityInfo, false},
		{"critical blocks warning", SeverityCritical, SeverityWarning, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			cfg := &AuditLoggerConfig{
				Enabled:      true,
				Output:       buf,
				Format:       "json",
				MinSeverity:  tt.minSeverity,
				AsyncEnabled: false,
				BufferSize:   10,
			}

			logger, err := NewAuditLogger(cfg)
			require.NoError(t, err)
			defer logger.Close()

			logger.Log(&AuditEvent{
				EventType: EventTypeSystemStartup,
				Severity:  tt.eventSev,
				Success:   true,
			})

			if tt.shouldLog {
				assert.NotEmpty(t, buf.String())
			} else {
				assert.Empty(t, buf.String())
			}
		})
	}
}

func TestLog_SetsIDAndTimestamp(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &AuditLoggerConfig{
		Enabled:      true,
		Output:       buf,
		Format:       "json",
		MinSeverity:  SeverityInfo,
		AsyncEnabled: false,
		BufferSize:   10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	event := &AuditEvent{
		EventType: EventTypeSystemStartup,
		Severity:  SeverityInfo,
		Success:   true,
	}

	logger.Log(event)

	assert.NotEmpty(t, event.ID)
	assert.True(t, strings.HasPrefix(event.ID, "audit-"))
	assert.False(t, event.Timestamp.IsZero())
}

func TestLog_PreservesExistingTimestamp(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &AuditLoggerConfig{
		Enabled:      true,
		Output:       buf,
		Format:       "json",
		MinSeverity:  SeverityInfo,
		AsyncEnabled: false,
		BufferSize:   10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	specificTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	event := &AuditEvent{
		EventType: EventTypeSystemStartup,
		Severity:  SeverityInfo,
		Timestamp: specificTime,
		Success:   true,
	}

	logger.Log(event)

	assert.Equal(t, specificTime, event.Timestamp)
}

func TestLog_SanitizesEvent(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &AuditLoggerConfig{
		Enabled:          true,
		Output:           buf,
		Format:           "json",
		MinSeverity:      SeverityInfo,
		IncludeSensitive: false,
		AsyncEnabled:     false,
		BufferSize:       10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	logger.Log(&AuditEvent{
		EventType: EventTypeLogin,
		Severity:  SeverityInfo,
		Success:   true,
		Metadata: map[string]interface{}{
			"password":  "secret123",
			"token":     "mytoken",
			"username":  "john",
			"apiKey":    "key123",
			"safeField": "visible",
		},
	})

	output := buf.String()
	assert.NotContains(t, output, "secret123")
	assert.NotContains(t, output, "mytoken")
	assert.NotContains(t, output, "key123")
	assert.Contains(t, output, "john")
	assert.Contains(t, output, "visible")
}

func TestLog_IncludeSensitive(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &AuditLoggerConfig{
		Enabled:          true,
		Output:           buf,
		Format:           "json",
		MinSeverity:      SeverityInfo,
		IncludeSensitive: true,
		AsyncEnabled:     false,
		BufferSize:       10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	logger.Log(&AuditEvent{
		EventType: EventTypeLogin,
		Severity:  SeverityInfo,
		Success:   true,
		Metadata: map[string]interface{}{
			"password": "secret123",
		},
	})

	assert.Contains(t, buf.String(), "secret123")
}

// --- Format Tests ---

func TestLog_JSONFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &AuditLoggerConfig{
		Enabled:      true,
		Output:       buf,
		Format:       "json",
		MinSeverity:  SeverityInfo,
		AsyncEnabled: false,
		BufferSize:   10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	logger.Log(&AuditEvent{
		EventType:   EventTypeLogin,
		Severity:    SeverityInfo,
		UserID:      "user1",
		Description: "test login",
		Success:     true,
	})

	var parsed AuditEvent
	err = json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &parsed)
	require.NoError(t, err)
	assert.Equal(t, EventTypeLogin, parsed.EventType)
	assert.Equal(t, SeverityInfo, parsed.Severity)
	assert.Equal(t, "user1", parsed.UserID)
	assert.Equal(t, "test login", parsed.Description)
	assert.True(t, parsed.Success)
}

func TestLog_TextFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &AuditLoggerConfig{
		Enabled:      true,
		Output:       buf,
		Format:       "text",
		MinSeverity:  SeverityInfo,
		AsyncEnabled: false,
		BufferSize:   10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	logger.Log(&AuditEvent{
		EventType:   EventTypeLogin,
		Severity:    SeverityInfo,
		UserID:      "user1",
		Action:      "login",
		Description: "test login",
		Success:     true,
	})

	output := buf.String()
	assert.Contains(t, output, "info")
	assert.Contains(t, output, "auth.login")
	assert.Contains(t, output, "user=user1")
	assert.Contains(t, output, "success=true")
}

func TestLog_DefaultFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &AuditLoggerConfig{
		Enabled:      true,
		Output:       buf,
		Format:       "unknown_format",
		MinSeverity:  SeverityInfo,
		AsyncEnabled: false,
		BufferSize:   10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	logger.Log(&AuditEvent{
		EventType: EventTypeLogin,
		Severity:  SeverityInfo,
		Success:   true,
	})

	// Default format is JSON
	var parsed AuditEvent
	err = json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &parsed)
	require.NoError(t, err)
}

// --- Async logging ---

func TestLog_AsyncLogging(t *testing.T) {
	buf := &safeBuffer{}
	cfg := &AuditLoggerConfig{
		Enabled:      true,
		Output:       buf,
		Format:       "json",
		MinSeverity:  SeverityInfo,
		AsyncEnabled: true,
		BufferSize:   100,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)

	logger.Log(&AuditEvent{
		EventType:   EventTypeLogin,
		Severity:    SeverityInfo,
		Description: "async test",
		Success:     true,
	})

	logger.Flush()
	// Allow async writer to process
	time.Sleep(100 * time.Millisecond)

	assert.Contains(t, buf.String(), "async test")

	err = logger.Close()
	require.NoError(t, err)
}

// --- LogContext ---

func TestLogContext(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &AuditLoggerConfig{
		Enabled:      true,
		Output:       buf,
		Format:       "json",
		MinSeverity:  SeverityInfo,
		AsyncEnabled: false,
		BufferSize:   10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.WithValue(context.Background(), "userID", "ctx-user")
	ctx = context.WithValue(ctx, "requestID", "req-123")

	event := &AuditEvent{
		EventType: EventTypeLogin,
		Severity:  SeverityInfo,
		Success:   true,
	}

	logger.LogContext(ctx, event)

	assert.Equal(t, "ctx-user", event.UserID)
	assert.Equal(t, "req-123", event.RequestID)
}

func TestLogContext_DoesNotOverrideExisting(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &AuditLoggerConfig{
		Enabled:      true,
		Output:       buf,
		Format:       "json",
		MinSeverity:  SeverityInfo,
		AsyncEnabled: false,
		BufferSize:   10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.WithValue(context.Background(), "userID", "ctx-user")

	event := &AuditEvent{
		EventType: EventTypeLogin,
		Severity:  SeverityInfo,
		UserID:    "existing-user",
		Success:   true,
	}

	logger.LogContext(ctx, event)

	assert.Equal(t, "existing-user", event.UserID)
}

// --- Helper methods ---

func TestLogLogin(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &AuditLoggerConfig{
		Enabled:      true,
		Output:       buf,
		Format:       "json",
		MinSeverity:  SeverityInfo,
		AsyncEnabled: false,
		BufferSize:   10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	logger.LogLogin("user1", "user@test.com", "127.0.0.1", "TestAgent")

	output := buf.String()
	assert.Contains(t, output, "auth.login")
	assert.Contains(t, output, "user1")
	assert.Contains(t, output, "user@test.com")
}

func TestLogLoginFailed(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &AuditLoggerConfig{
		Enabled:      true,
		Output:       buf,
		Format:       "json",
		MinSeverity:  SeverityInfo,
		AsyncEnabled: false,
		BufferSize:   10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	logger.LogLoginFailed("user@test.com", "127.0.0.1", "TestAgent", "invalid password")

	output := buf.String()
	assert.Contains(t, output, "auth.login_failed")
	assert.Contains(t, output, "invalid password")
}

func TestLogWorkflowCreated(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &AuditLoggerConfig{
		Enabled:      true,
		Output:       buf,
		Format:       "json",
		MinSeverity:  SeverityInfo,
		AsyncEnabled: false,
		BufferSize:   10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	logger.LogWorkflowCreated("user1", "wf-1", "My Workflow")

	output := buf.String()
	assert.Contains(t, output, "workflow.created")
	assert.Contains(t, output, "wf-1")
	assert.Contains(t, output, "My Workflow")
}

func TestLogWorkflowUpdated(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &AuditLoggerConfig{
		Enabled:      true,
		Output:       buf,
		Format:       "json",
		MinSeverity:  SeverityInfo,
		AsyncEnabled: false,
		BufferSize:   10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	changes := map[string]interface{}{"active": true}
	logger.LogWorkflowUpdated("user1", "wf-1", "My Workflow", changes)

	output := buf.String()
	assert.Contains(t, output, "workflow.updated")
}

func TestLogWorkflowDeleted(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &AuditLoggerConfig{
		Enabled:      true,
		Output:       buf,
		Format:       "json",
		MinSeverity:  SeverityInfo,
		AsyncEnabled: false,
		BufferSize:   10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	logger.LogWorkflowDeleted("user1", "wf-1", "My Workflow")

	output := buf.String()
	assert.Contains(t, output, "workflow.deleted")
}

func TestLogExecutionFailed(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &AuditLoggerConfig{
		Enabled:      true,
		Output:       buf,
		Format:       "json",
		MinSeverity:  SeverityInfo,
		AsyncEnabled: false,
		BufferSize:   10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	logger.LogExecutionFailed("wf-1", "exec-1", "node failed")

	output := buf.String()
	assert.Contains(t, output, "execution.failed")
	assert.Contains(t, output, "node failed")
}

func TestLogCredentialAccessed(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &AuditLoggerConfig{
		Enabled:      true,
		Output:       buf,
		Format:       "json",
		MinSeverity:  SeverityInfo,
		AsyncEnabled: false,
		BufferSize:   10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	logger.LogCredentialAccessed("user1", "cred-1", "My API Key", "workflow execution")

	output := buf.String()
	assert.Contains(t, output, "credential.accessed")
	assert.Contains(t, output, "My API Key")
}

func TestLogRateLimited(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &AuditLoggerConfig{
		Enabled:      true,
		Output:       buf,
		Format:       "json",
		MinSeverity:  SeverityInfo,
		AsyncEnabled: false,
		BufferSize:   10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	logger.LogRateLimited("10.0.0.1", "/api/execute")

	output := buf.String()
	assert.Contains(t, output, "security.rate_limited")
	assert.Contains(t, output, "/api/execute")
}

func TestLogSystemStartup(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &AuditLoggerConfig{
		Enabled:      true,
		Output:       buf,
		Format:       "json",
		MinSeverity:  SeverityInfo,
		AsyncEnabled: false,
		BufferSize:   10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	logger.LogSystemStartup("1.0.0", map[string]interface{}{"port": 8080})

	output := buf.String()
	assert.Contains(t, output, "system.startup")
}

func TestLogSystemShutdown(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &AuditLoggerConfig{
		Enabled:      true,
		Output:       buf,
		Format:       "json",
		MinSeverity:  SeverityInfo,
		AsyncEnabled: false,
		BufferSize:   10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	logger.LogSystemShutdown("graceful shutdown")

	output := buf.String()
	assert.Contains(t, output, "system.shutdown")
	assert.Contains(t, output, "graceful shutdown")
}

func TestLogSystemError(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &AuditLoggerConfig{
		Enabled:      true,
		Output:       buf,
		Format:       "json",
		MinSeverity:  SeverityInfo,
		AsyncEnabled: false,
		BufferSize:   10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	logger.LogSystemError(assert.AnError, "database connection lost")

	output := buf.String()
	assert.Contains(t, output, "system.error")
	assert.Contains(t, output, "database connection lost")
}

// --- Close and Flush ---

func TestClose(t *testing.T) {
	cfg := &AuditLoggerConfig{
		Enabled:      true,
		Output:       &bytes.Buffer{},
		Format:       "json",
		MinSeverity:  SeverityInfo,
		AsyncEnabled: true,
		BufferSize:   100,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)

	err = logger.Close()
	assert.NoError(t, err)
}

func TestFlush_SyncLogger(t *testing.T) {
	cfg := &AuditLoggerConfig{
		Enabled:      true,
		Output:       &bytes.Buffer{},
		Format:       "json",
		MinSeverity:  SeverityInfo,
		AsyncEnabled: false,
		BufferSize:   10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	// Flush on sync logger should be a no-op
	logger.Flush()
}

// --- isSensitiveKey ---

func TestIsSensitiveKey(t *testing.T) {
	tests := []struct {
		key       string
		sensitive bool
	}{
		{"password", true},
		{"secret", true},
		{"token", true},
		{"apiKey", true},
		{"api_key", true},
		{"credential", true},
		{"privateKey", true},
		{"private_key", true},
		{"accessToken", true},
		{"access_token", true},
		{"refreshToken", true},
		{"refresh_token", true},
		{"username", false},
		{"email", false},
		{"workflowId", false},
		{"status", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			assert.Equal(t, tt.sensitive, isSensitiveKey(tt.key))
		})
	}
}

// --- Concurrent Access ---

func TestLog_ConcurrentAccess(t *testing.T) {
	buf := &safeBuffer{}
	cfg := &AuditLoggerConfig{
		Enabled:      true,
		Output:       buf,
		Format:       "json",
		MinSeverity:  SeverityInfo,
		AsyncEnabled: false,
		BufferSize:   10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			logger.Log(&AuditEvent{
				EventType: EventTypeLogin,
				Severity:  SeverityInfo,
				Success:   true,
			})
		}()
	}

	wg.Wait()

	// All 50 events should have been logged
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Equal(t, 50, len(lines))
}

// --- nextEventID ---

func TestNextEventID_Unique(t *testing.T) {
	cfg := &AuditLoggerConfig{
		Enabled:      true,
		Output:       &bytes.Buffer{},
		Format:       "json",
		AsyncEnabled: false,
		BufferSize:   10,
	}

	logger, err := NewAuditLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := logger.nextEventID()
		assert.False(t, ids[id], "duplicate event ID: %s", id)
		ids[id] = true
	}
}

// --- AuditQuery struct ---

func TestAuditQuery_Struct(t *testing.T) {
	query := &AuditQuery{
		StartTime:    time.Now().Add(-time.Hour),
		EndTime:      time.Now(),
		EventTypes:   []EventType{EventTypeLogin},
		UserID:       "user1",
		ResourceType: "workflow",
		ResourceID:   "wf-1",
		Severity:     SeverityInfo,
		Limit:        10,
		Offset:       0,
	}

	assert.Equal(t, "user1", query.UserID)
	assert.Equal(t, 10, query.Limit)
	assert.Equal(t, 1, len(query.EventTypes))
}

// safeBuffer is a thread-safe bytes.Buffer for use in concurrent tests.
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (sb *safeBuffer) Write(p []byte) (n int, err error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

func (sb *safeBuffer) String() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.String()
}
