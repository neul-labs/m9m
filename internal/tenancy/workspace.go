// Package tenancy provides the server-side workspace concept that scopes
// every workflow, execution, credential, and credential reference to a
// tenant. It is distinct from internal/workspace (which models per-user CLI
// filesystem workspaces).
//
// In m9m Cloud each tenant gets its own Workspace identified by a UUID.
// Self-hosted single-tenant deployments use the DefaultID workspace
// transparently — existing data is migrated to belong to DefaultID and the
// rest of the engine carries on as before.
//
// Multi-tenancy enforcement lives at the storage layer (every query is
// workspace-scoped) and at the API middleware layer (every request carries
// a workspace context). The engine itself is tenant-agnostic: it processes
// a workflow without caring which workspace it came from. This keeps the
// engine simple and the isolation property easy to reason about.
package tenancy

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// DefaultID is the workspace UUID assigned to all data in a single-tenant
// deployment, and to any data that existed before the multi-tenancy refactor.
// It is a fixed UUID (the all-zeros UUID by convention) so existing rows
// always sort to the same workspace and self-hosted users can rely on it.
const DefaultID = "00000000-0000-0000-0000-000000000000"

// IsDefault reports whether the given workspace ID is the default workspace.
func IsDefault(id string) bool { return id == DefaultID }

// Workspace is the server-side tenant identity. One workspace contains many
// workflows, executions, credentials, and members. A user may belong to
// many workspaces via workspace_members.
type Workspace struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// OrganizationID is optional. When set, the workspace belongs to an
	// org (used for multi-workspace billing arrangements). Self-hosted
	// deployments leave this empty.
	OrganizationID string `json:"organizationId,omitempty"`
}

// NewWorkspace constructs a Workspace with a freshly-generated UUID.
func NewWorkspace(name string) *Workspace {
	now := time.Now().UTC()
	return &Workspace{
		ID:        uuid.NewString(),
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Validate returns nil if the workspace fields are well-formed.
func (w *Workspace) Validate() error {
	if w == nil {
		return errors.New("tenancy: workspace is nil")
	}
	if w.ID == "" {
		return errors.New("tenancy: workspace ID is required")
	}
	if _, err := uuid.Parse(w.ID); err != nil {
		return errors.New("tenancy: workspace ID is not a valid UUID")
	}
	if w.Name == "" {
		return errors.New("tenancy: workspace name is required")
	}
	return nil
}

// contextKey is unexported so callers can only put/get a workspace ID via
// the helpers in this package.
type contextKey struct{}

var workspaceKey = contextKey{}

// WithID returns a derived context that carries the given workspace ID.
// The engine, storage layer, and middleware use this to scope every
// operation to a tenant without threading the ID through every function
// signature.
func WithID(parent context.Context, workspaceID string) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, workspaceKey, workspaceID)
}

// FromContext returns the workspace ID associated with ctx, or DefaultID if
// none is set. The second return is true when an explicit workspace was
// found — callers that need to *require* an explicit workspace (e.g. the
// Cloud router) can branch on it.
func FromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return DefaultID, false
	}
	v, ok := ctx.Value(workspaceKey).(string)
	if !ok || v == "" {
		return DefaultID, false
	}
	return v, true
}

// RequireID returns the workspace ID from ctx, or an error if none is set.
// Use this in places (Cloud router, multi-tenant API endpoints) where
// missing workspace context is a programming error rather than a fallback.
func RequireID(ctx context.Context) (string, error) {
	id, ok := FromContext(ctx)
	if !ok {
		return "", ErrMissingWorkspace
	}
	return id, nil
}

// ErrMissingWorkspace is returned by RequireID when no workspace ID is in
// the context.
var ErrMissingWorkspace = errors.New("tenancy: workspace ID missing from context")
