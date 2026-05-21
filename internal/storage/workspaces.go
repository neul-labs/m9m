package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/neul-labs/m9m/internal/tenancy"
)

// ensureColumn adds a column to a table if it does not already exist. Used
// for idempotent migrations of pre-existing tables when columns are added
// in later releases. Errors that indicate the column already exists are
// swallowed; all other errors are returned.
func ensureColumn(db *sql.DB, table, column, columnDef string) error {
	_, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, columnDef))
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "duplicate column") || // SQLite
		strings.Contains(msg, "already exists") || // Postgres
		strings.Contains(msg, "duplicate key") {
		return nil
	}
	return err
}

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

// resolveWorkspaceID returns the workspace ID to use when persisting a row,
// substituting the default workspace for empty inputs. The substitution
// keeps existing data and single-tenant callers working unchanged while
// preserving the NOT NULL invariant on the column.
func resolveWorkspaceID(id string) string {
	if id == "" {
		return tenancy.DefaultID
	}
	return id
}
