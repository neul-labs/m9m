package blob

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// LocalFS stores blobs as files on the local filesystem. Useful for
// self-hosted deployments and tests.
//
// Layout: <Root>/<workspaceID>/<path>. Paths are joined safely (no traversal
// outside the workspace root).
type LocalFS struct {
	Root string
}

// NewLocalFS returns a Store rooted at the given directory. The directory is
// created if it does not exist.
func NewLocalFS(root string) (*LocalFS, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &LocalFS{Root: root}, nil
}

func (l *LocalFS) workspaceDir(workspaceID string) string {
	return filepath.Join(l.Root, workspaceID)
}

// pathFor joins workspaceID and path, rejecting traversal attempts.
func (l *LocalFS) pathFor(workspaceID, path string) (string, error) {
	if strings.Contains(path, "..") {
		return "", errors.New("blob: path must not contain ..")
	}
	clean := filepath.Clean(path)
	if strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", errors.New("blob: path resolves outside workspace")
	}
	return filepath.Join(l.workspaceDir(workspaceID), clean), nil
}

// Put writes the object atomically (write-then-rename).
func (l *LocalFS) Put(ctx context.Context, workspaceID, path string, content io.Reader, opts *PutOptions) (*ObjectInfo, error) {
	dest, err := l.pathFor(workspaceID, path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return nil, err
	}
	tmp := dest + ".tmp"
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
	digest := hex.EncodeToString(h.Sum(nil))
	if opts != nil && opts.SHA256 != "" && opts.SHA256 != digest {
		return nil, ErrChecksumMismatch
	}
	if err := os.Rename(tmp, dest); err != nil {
		return nil, err
	}
	contentType := ""
	if opts != nil {
		contentType = opts.ContentType
	}
	return &ObjectInfo{
		WorkspaceID: workspaceID,
		Path:        path,
		Size:        n,
		UpdatedAt:   time.Now(),
		SHA256:      digest,
		ContentType: contentType,
	}, nil
}

// Get opens the file for reading.
func (l *LocalFS) Get(ctx context.Context, workspaceID, path string) (io.ReadCloser, *ObjectInfo, error) {
	dest, err := l.pathFor(workspaceID, path)
	if err != nil {
		return nil, nil, err
	}
	f, err := os.Open(dest)
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
	return f, &ObjectInfo{
		WorkspaceID: workspaceID,
		Path:        path,
		Size:        fi.Size(),
		UpdatedAt:   fi.ModTime(),
	}, nil
}

// Stat returns metadata without opening the file content.
func (l *LocalFS) Stat(ctx context.Context, workspaceID, path string) (*ObjectInfo, error) {
	dest, err := l.pathFor(workspaceID, path)
	if err != nil {
		return nil, err
	}
	fi, err := os.Stat(dest)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &ObjectInfo{
		WorkspaceID: workspaceID,
		Path:        path,
		Size:        fi.Size(),
		UpdatedAt:   fi.ModTime(),
	}, nil
}

// Delete removes the file.
func (l *LocalFS) Delete(ctx context.Context, workspaceID, path string) error {
	dest, err := l.pathFor(workspaceID, path)
	if err != nil {
		return err
	}
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// List walks the workspace dir under prefix, returning lexically sorted info.
func (l *LocalFS) List(ctx context.Context, workspaceID, prefix string, limit int) ([]ObjectInfo, error) {
	root := l.workspaceDir(workspaceID)
	prefixDir, err := l.pathFor(workspaceID, prefix)
	if err != nil {
		return nil, err
	}
	var out []ObjectInfo
	err = filepath.WalkDir(prefixDir, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		fi, err := d.Info()
		if err != nil {
			return err
		}
		out = append(out, ObjectInfo{
			WorkspaceID: workspaceID,
			Path:        rel,
			Size:        fi.Size(),
			UpdatedAt:   fi.ModTime(),
		})
		if limit > 0 && len(out) >= limit*2 {
			// soft early-exit on very deep trees; final sort + truncate below
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// DeleteWorkspace removes the entire workspace directory.
func (l *LocalFS) DeleteWorkspace(ctx context.Context, workspaceID string) error {
	dir := l.workspaceDir(workspaceID)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("blob: remove workspace dir: %w", err)
	}
	return nil
}

var _ Store = (*LocalFS)(nil)
