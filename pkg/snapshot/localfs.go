package snapshot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// LocalFS stores snapshots as files on the local filesystem. Useful for
// self-hosted deployments and tests.
//
// Layout: <Root>/<workspaceID>/<version>.tar.gz with a sibling .meta file
// holding metadata. The "latest" snapshot is whichever has the largest
// version (versions are zero-padded unix-nanos to sort lexically).
type LocalFS struct {
	Root string
}

// NewLocalFS returns a Sink that stores snapshots under root. The directory
// is created if it does not exist.
func NewLocalFS(root string) (*LocalFS, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &LocalFS{Root: root}, nil
}

func (l *LocalFS) workspaceDir(workspaceID string) string {
	return filepath.Join(l.Root, workspaceID)
}

// Save writes the snapshot to disk and returns its metadata.
func (l *LocalFS) Save(ctx context.Context, workspaceID string, content io.Reader) (*Snapshot, error) {
	dir := l.workspaceDir(workspaceID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	version := strconv.FormatInt(time.Now().UnixNano(), 10)
	path := filepath.Join(dir, version+".tar.gz")
	tmp := path + ".tmp"

	f, err := os.Create(tmp)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
		_ = os.Remove(tmp)
	}()

	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), content)
	if err != nil {
		return nil, err
	}
	if err := f.Sync(); err != nil {
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	if err := os.Rename(tmp, path); err != nil {
		return nil, err
	}

	snap := &Snapshot{
		WorkspaceID: workspaceID,
		Version:     version,
		Size:        n,
		CreatedAt:   time.Now(),
		SHA256:      hex.EncodeToString(h.Sum(nil)),
	}
	return snap, nil
}

// Load returns the snapshot content. If version is empty, returns the latest.
func (l *LocalFS) Load(ctx context.Context, workspaceID, version string) (io.ReadCloser, *Snapshot, error) {
	if version == "" {
		list, err := l.List(ctx, workspaceID, 1)
		if err != nil {
			return nil, nil, err
		}
		if len(list) == 0 {
			return nil, nil, ErrNotFound
		}
		version = list[0].Version
	}
	path := filepath.Join(l.workspaceDir(workspaceID), version+".tar.gz")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, err
	}
	fi, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, nil, err
	}
	snap := &Snapshot{
		WorkspaceID: workspaceID,
		Version:     version,
		Size:        fi.Size(),
		CreatedAt:   fi.ModTime(),
	}
	return f, snap, nil
}

// List returns snapshots newest first, up to limit.
func (l *LocalFS) List(ctx context.Context, workspaceID string, limit int) ([]Snapshot, error) {
	dir := l.workspaceDir(workspaceID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]Snapshot, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".tar.gz") {
			continue
		}
		ver := strings.TrimSuffix(name, ".tar.gz")
		fi, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, Snapshot{
			WorkspaceID: workspaceID,
			Version:     ver,
			Size:        fi.Size(),
			CreatedAt:   fi.ModTime(),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version > out[j].Version })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// Delete removes a single snapshot version.
func (l *LocalFS) Delete(ctx context.Context, workspaceID, version string) error {
	path := filepath.Join(l.workspaceDir(workspaceID), version+".tar.gz")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// DeleteWorkspace removes the entire workspace directory.
func (l *LocalFS) DeleteWorkspace(ctx context.Context, workspaceID string) error {
	dir := l.workspaceDir(workspaceID)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("snapshot: remove workspace dir: %w", err)
	}
	return nil
}

var _ Sink = (*LocalFS)(nil)
