// Package blob defines the extension contract for per-workspace blob
// storage. Workflows, execution logs, encrypted credentials, npm caches, and
// other "content" (as distinct from "metadata") live here.
//
// blob.Store is intentionally distinct from snapshot.Sink:
//   - Store handles ad-hoc read/write of individual objects identified by
//     path, during normal operation. Typical access pattern: many small
//     gets/puts/lists per request.
//   - snapshot.Sink handles atomic save/load of full workspace state at a
//     point in time, for hibernation and disaster recovery.
//
// The engine uses blob.Store for everything that isn't a hot-path index
// lookup. m9m Cloud's cost economics depend on this: Postgres holds only
// metadata (5-10 KB per tenant); content lives in cheap object storage
// (Hetzner OSS at €4.90 / TB / month, zero egress).
//
// This is part of the public embedding surface and commits to semver.
package blob

import (
	"context"
	"errors"
	"io"
	"time"
)

// ObjectInfo is the metadata for a single object.
type ObjectInfo struct {
	WorkspaceID string
	Path        string // workspace-relative path
	Size        int64  // bytes
	UpdatedAt   time.Time
	SHA256      string // hex digest; empty if not computed
	// ContentType is the MIME type recorded with the object. May be empty.
	ContentType string
}

// PutOptions is optional metadata for Put.
type PutOptions struct {
	ContentType string
	// SHA256 is the expected hash, hex-encoded. If set, the Store verifies
	// after upload and returns an error on mismatch.
	SHA256 string
}

// Store is the extension contract for per-workspace blob storage.
//
// All methods are safe for concurrent use. Paths are workspace-relative and
// use forward slashes. Implementations are not required to support
// arbitrarily deep prefixes; embedders should organize paths sensibly (e.g.
// workflows/<id>.json, executions/<id>.log).
//
// Operations are on the worker's hot path. Get/Put should return within
// 50-200ms for typical objects (<1 MB) against a network-attached store.
// Larger objects (npm caches, exec logs) may take longer.
type Store interface {
	// Put writes the object's content. Returns the resulting ObjectInfo.
	// Implementations may compute a SHA-256 hash; if the caller supplied one
	// in opts and it doesn't match, an error is returned and the object is
	// not committed.
	Put(ctx context.Context, workspaceID, path string, content io.Reader, opts *PutOptions) (*ObjectInfo, error)

	// Get returns a reader for the object content. The caller must close the
	// reader. ObjectInfo includes the size and stored metadata.
	Get(ctx context.Context, workspaceID, path string) (io.ReadCloser, *ObjectInfo, error)

	// Stat returns metadata without fetching content.
	Stat(ctx context.Context, workspaceID, path string) (*ObjectInfo, error)

	// Delete removes an object. Returns nil if it does not exist.
	Delete(ctx context.Context, workspaceID, path string) error

	// List returns objects under a prefix, lexically sorted. Limit is a soft
	// cap. For deep listings, callers should paginate by passing a larger
	// limit or by issuing successive calls with a more-specific prefix.
	//
	// List is for admin/maintenance paths. The engine should not List at
	// runtime for hot lookups — those go through the index tables in the
	// configured storage backend.
	List(ctx context.Context, workspaceID, prefix string, limit int) ([]ObjectInfo, error)

	// DeleteWorkspace removes all objects for a workspace. Called on
	// workspace deletion. May be eventually consistent.
	DeleteWorkspace(ctx context.Context, workspaceID string) error
}

// ErrNotFound is returned by Get / Stat when the object does not exist.
var ErrNotFound = errors.New("blob: not found")

// ErrChecksumMismatch is returned by Put when the caller supplied a SHA-256
// in PutOptions and the stored content's hash differs.
var ErrChecksumMismatch = errors.New("blob: checksum mismatch")
