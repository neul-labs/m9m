// Package snapshot defines the extension contract for saving and restoring
// the complete state of a workspace at a point in time. This is the
// foundation of m9m Cloud's hibernate/revive lifecycle: idle tenants are
// snapshotted to object storage and the worker container exits; on the next
// request the snapshot is restored and execution resumes.
//
// snapshot.Sink is intentionally distinct from blob.Store:
//   - Sink handles atomic save/load of full workspace state. Versioned by
//     timestamp; primarily for hibernation and disaster recovery.
//   - blob.Store handles ad-hoc read/write of individual objects (workflow
//     JSON, execution logs, etc.) during normal operation.
//
// Self-hosters who don't care about hibernation can use the Disabled
// implementation. Self-hosters running multi-tenant on shared storage use
// LocalFS. m9m Cloud wires S3Compatible to Hetzner Object Storage.
//
// This is part of the public embedding surface and commits to semver.
package snapshot

import (
	"context"
	"errors"
	"io"
	"time"
)

// Snapshot is the metadata for a single snapshot.
type Snapshot struct {
	WorkspaceID string
	Version     string    // implementation-defined; typically a timestamp or generation
	Size        int64     // bytes, may be -1 if unknown
	CreatedAt   time.Time
	SHA256      string    // hex digest of payload; empty if not computed
}

// Sink is the extension contract for snapshot storage.
//
// All methods are safe for concurrent use. Implementations may parallelize
// internally (multipart upload, async checksum, etc.).
//
// The engine calls these methods rarely (per hibernation, per wake) so
// latency budgets are looser than blob.Store — a hundred milliseconds is
// fine, a few seconds is acceptable for very large snapshots.
type Sink interface {
	// Save uploads a snapshot for the given workspace. The reader contains
	// the workspace's serialized state (typically a gzipped tar of the
	// workspace SQLite + ancillary files). Returns the resulting Snapshot
	// metadata including the assigned Version.
	Save(ctx context.Context, workspaceID string, content io.Reader) (*Snapshot, error)

	// Load fetches the latest snapshot for a workspace, or the specified
	// Version if non-empty. Returns a ReadCloser the caller must drain and
	// close.
	Load(ctx context.Context, workspaceID string, version string) (io.ReadCloser, *Snapshot, error)

	// List returns snapshot metadata for a workspace, newest first. May be
	// truncated; the limit parameter is a soft cap.
	List(ctx context.Context, workspaceID string, limit int) ([]Snapshot, error)

	// Delete removes a specific snapshot version. Implementations may
	// soft-delete (preserving for some retention window).
	Delete(ctx context.Context, workspaceID string, version string) error

	// DeleteWorkspace removes all snapshots for a workspace. Called on
	// workspace deletion. Implementations may be eventually consistent.
	DeleteWorkspace(ctx context.Context, workspaceID string) error
}

// ErrNotFound is returned by Load when no snapshot exists for the given
// workspace/version.
var ErrNotFound = errors.New("snapshot: not found")
