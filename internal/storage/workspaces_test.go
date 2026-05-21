package storage

import (
	"strings"
	"testing"

	"github.com/neul-labs/m9m/internal/model"
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

func TestSaveWorkflow_PopulatesWorkspaceID(t *testing.T) {
	// SaveWorkflow with an empty WorkspaceID should be stamped with
	// DefaultID so the NOT NULL invariant on the workspaces column holds.
	s := NewMemoryStorage()
	wf := &model.Workflow{Name: "wf", Nodes: nil, Connections: nil}
	if err := s.SaveWorkflow(wf); err != nil {
		t.Fatal(err)
	}
	if wf.WorkspaceID != tenancy.DefaultID {
		t.Errorf("empty WorkspaceID was not defaulted: got %q, want %q", wf.WorkspaceID, tenancy.DefaultID)
	}

	got, err := s.GetWorkflow(wf.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.WorkspaceID != tenancy.DefaultID {
		t.Errorf("stored WorkspaceID = %q, want %q", got.WorkspaceID, tenancy.DefaultID)
	}
}

func TestListWorkflows_ScopedByWorkspace(t *testing.T) {
	s := NewMemoryStorage()
	wsA := tenancy.NewWorkspace("A")
	wsB := tenancy.NewWorkspace("B")
	_ = s.SaveWorkspace(wsA)
	_ = s.SaveWorkspace(wsB)

	wfA := &model.Workflow{Name: "in A", WorkspaceID: wsA.ID}
	wfB := &model.Workflow{Name: "in B", WorkspaceID: wsB.ID}
	if err := s.SaveWorkflow(wfA); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveWorkflow(wfB); err != nil {
		t.Fatal(err)
	}

	listA, _, err := s.ListWorkflows(WorkflowFilters{WorkspaceID: wsA.ID, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(listA) != 1 || listA[0].Name != "in A" {
		t.Errorf("workspace-A list = %v, want [in A]", listA)
	}

	// Unfiltered (admin-style) list returns both.
	all, _, _ := s.ListWorkflows(WorkflowFilters{Limit: 100})
	if len(all) != 2 {
		t.Errorf("unfiltered list returned %d, want 2", len(all))
	}
}

func TestSaveExecution_PopulatesWorkspaceID(t *testing.T) {
	s := NewMemoryStorage()
	exec := &model.WorkflowExecution{WorkflowID: "wf-1", Status: "running"}
	if err := s.SaveExecution(exec); err != nil {
		t.Fatal(err)
	}
	if exec.WorkspaceID != tenancy.DefaultID {
		t.Errorf("empty WorkspaceID was not defaulted: got %q", exec.WorkspaceID)
	}
	got, err := s.GetExecution(exec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.WorkspaceID != tenancy.DefaultID {
		t.Errorf("stored execution WorkspaceID = %q", got.WorkspaceID)
	}
}

func TestListExecutions_ScopedByWorkspace(t *testing.T) {
	s := NewMemoryStorage()
	wsA := tenancy.NewWorkspace("A")
	wsB := tenancy.NewWorkspace("B")
	_ = s.SaveWorkspace(wsA)
	_ = s.SaveWorkspace(wsB)

	execA := &model.WorkflowExecution{WorkflowID: "wf-1", Status: "completed", WorkspaceID: wsA.ID}
	execB := &model.WorkflowExecution{WorkflowID: "wf-2", Status: "completed", WorkspaceID: wsB.ID}
	if err := s.SaveExecution(execA); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveExecution(execB); err != nil {
		t.Fatal(err)
	}

	scopedA, _, err := s.ListExecutions(ExecutionFilters{WorkspaceID: wsA.ID, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(scopedA) != 1 || scopedA[0].WorkflowID != "wf-1" {
		t.Errorf("workspace-A executions = %v, want [wf-1]", scopedA)
	}

	all, _, _ := s.ListExecutions(ExecutionFilters{Limit: 100})
	if len(all) != 2 {
		t.Errorf("unfiltered execution list returned %d, want 2", len(all))
	}
}
