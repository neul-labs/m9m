package tenancy

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestDefaultID_IsZeroUUID(t *testing.T) {
	if DefaultID != "00000000-0000-0000-0000-000000000000" {
		t.Errorf("DefaultID changed from the all-zeros UUID convention")
	}
	if !IsDefault(DefaultID) {
		t.Error("IsDefault returned false for DefaultID")
	}
	if IsDefault("00000000-0000-0000-0000-000000000001") {
		t.Error("IsDefault returned true for a non-default UUID")
	}
}

func TestNewWorkspace_HasUUIDAndTimestamps(t *testing.T) {
	ws := NewWorkspace("hello")
	if err := ws.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if ws.Name != "hello" {
		t.Errorf("Name = %q, want %q", ws.Name, "hello")
	}
	if ws.CreatedAt.IsZero() {
		t.Error("CreatedAt not set")
	}
	if ws.UpdatedAt.IsZero() {
		t.Error("UpdatedAt not set")
	}
	if strings.Count(ws.ID, "-") != 4 {
		t.Errorf("ID = %q not a UUID", ws.ID)
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		ws      *Workspace
		wantErr bool
	}{
		{"nil", nil, true},
		{"empty id", &Workspace{Name: "x"}, true},
		{"bad uuid", &Workspace{ID: "not-a-uuid", Name: "x"}, true},
		{"empty name", &Workspace{ID: "00000000-0000-0000-0000-000000000000"}, true},
		{"ok", &Workspace{ID: "00000000-0000-0000-0000-000000000000", Name: "x"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.ws.Validate()
			if c.wantErr && err == nil {
				t.Error("expected error")
			}
			if !c.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestContext_RoundTrip(t *testing.T) {
	ctx := WithID(context.Background(), "ws-abc")
	id, ok := FromContext(ctx)
	if !ok {
		t.Error("FromContext returned ok=false after WithID")
	}
	if id != "ws-abc" {
		t.Errorf("FromContext id = %q, want %q", id, "ws-abc")
	}
}

func TestContext_FallsBackToDefault(t *testing.T) {
	id, ok := FromContext(context.Background())
	if ok {
		t.Error("FromContext on a bare context should report ok=false")
	}
	if id != DefaultID {
		t.Errorf("fallback id = %q, want DefaultID", id)
	}
}

func TestContext_NilContextSafe(t *testing.T) {
	//nolint:staticcheck // intentionally passing nil
	id, ok := FromContext(nil)
	if ok || id != DefaultID {
		t.Errorf("nil ctx: id=%q ok=%v, want DefaultID/false", id, ok)
	}
	//nolint:staticcheck
	derived := WithID(nil, "ws-x")
	if derived == nil {
		t.Error("WithID(nil, ...) returned nil context")
	}
	if got, _ := FromContext(derived); got != "ws-x" {
		t.Errorf("after WithID(nil, ws-x), FromContext = %q", got)
	}
}

func TestRequireID(t *testing.T) {
	if _, err := RequireID(context.Background()); !errors.Is(err, ErrMissingWorkspace) {
		t.Errorf("RequireID on bare ctx: err=%v, want ErrMissingWorkspace", err)
	}
	id, err := RequireID(WithID(context.Background(), "ws-1"))
	if err != nil {
		t.Errorf("RequireID with workspace: err=%v", err)
	}
	if id != "ws-1" {
		t.Errorf("RequireID id = %q", id)
	}
}

func TestContext_EmptyStringFallsBackToDefault(t *testing.T) {
	// Defensive: if someone passes WithID(ctx, "") we must not return ""
	// from FromContext, otherwise queries would silently miss the default
	// workspace.
	ctx := WithID(context.Background(), "")
	id, ok := FromContext(ctx)
	if ok {
		t.Error("empty string workspace should be treated as not-set")
	}
	if id != DefaultID {
		t.Errorf("fallback id = %q, want DefaultID", id)
	}
}
