package storage

import (
	"strings"
	"testing"

	"github.com/neul-labs/m9m/internal/tenancy"
)

// TestMemoryStorage_DefaultWorkspaceBootstrap is the cross-backend
// invariant: every freshly-opened storage instance has the default
// workspace populated. We exercise it on MemoryStorage (the only backend
// safe to construct in a unit test); the SQLite and Postgres backends
// follow the same bootstrap path via bootstrapDefaultWorkspace.
func TestMemoryStorage_DefaultWorkspaceBootstrap(t *testing.T) {
	s := NewMemoryStorage()

	ws, err := s.GetWorkspace(tenancy.DefaultID)
	if err != nil {
		t.Fatalf("default workspace missing after construction: %v", err)
	}
	if ws.ID != tenancy.DefaultID {
		t.Errorf("default workspace ID = %q, want %q", ws.ID, tenancy.DefaultID)
	}
	if ws.Name == "" {
		t.Error("default workspace has empty Name")
	}
	if ws.CreatedAt.IsZero() {
		t.Error("default workspace CreatedAt is zero")
	}
}

func TestMemoryStorage_WorkspaceCRUD(t *testing.T) {
	s := NewMemoryStorage()

	ws := tenancy.NewWorkspace("acme")
	if err := s.SaveWorkspace(ws); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := s.GetWorkspace(ws.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "acme" {
		t.Errorf("Name = %q, want %q", got.Name, "acme")
	}

	// Listing includes default + the new one.
	list, err := s.ListWorkspaces()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Errorf("ListWorkspaces returned %d entries, want 2", len(list))
	}

	// Delete the new one (not the default).
	if err := s.DeleteWorkspace(ws.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.GetWorkspace(ws.ID); err == nil {
		t.Error("expected error getting deleted workspace")
	}

	// Deleting the default must fail.
	err = s.DeleteWorkspace(tenancy.DefaultID)
	if err == nil || !strings.Contains(err.Error(), "default") {
		t.Errorf("DeleteWorkspace(DefaultID) error = %v, want a 'default' guard error", err)
	}
}

func TestMemoryStorage_GetWorkspace_NotFound(t *testing.T) {
	s := NewMemoryStorage()
	_, err := s.GetWorkspace("nonexistent")
	if err == nil {
		t.Error("expected not-found error")
	}
}

func TestMemoryStorage_SaveWorkspace_RejectsInvalid(t *testing.T) {
	s := NewMemoryStorage()
	// Missing ID.
	if err := s.SaveWorkspace(&tenancy.Workspace{Name: "x"}); err == nil {
		t.Error("expected validation error on missing ID")
	}
	// Non-UUID ID.
	if err := s.SaveWorkspace(&tenancy.Workspace{ID: "abc", Name: "x"}); err == nil {
		t.Error("expected validation error on non-UUID ID")
	}
}

func TestMemoryStorage_SaveWorkspace_ClonesInput(t *testing.T) {
	// Defensive: the storage should not retain a pointer to the caller's
	// workspace; otherwise mutating the local copy would change stored
	// state.
	s := NewMemoryStorage()
	ws := tenancy.NewWorkspace("acme")
	if err := s.SaveWorkspace(ws); err != nil {
		t.Fatal(err)
	}
	ws.Name = "mutated locally"
	got, _ := s.GetWorkspace(ws.ID)
	if got.Name != "acme" {
		t.Errorf("storage retained alias to caller's workspace: name = %q", got.Name)
	}
}
