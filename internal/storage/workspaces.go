package storage

import (
	"time"

	"github.com/neul-labs/m9m/internal/tenancy"
)

// bootstrapDefaultWorkspace ensures the default workspace exists in the
// given backend. Called from every storage constructor so existing data and
// new single-tenant deployments always have a workspace to attach to.
func bootstrapDefaultWorkspace(s WorkflowStorage) error {
	existing, err := s.GetWorkspace(tenancy.DefaultID)
	if err == nil && existing != nil {
		return nil
	}
	now := time.Now().UTC()
	return s.SaveWorkspace(&tenancy.Workspace{
		ID:        tenancy.DefaultID,
		Name:      "default",
		CreatedAt: now,
		UpdatedAt: now,
	})
}
