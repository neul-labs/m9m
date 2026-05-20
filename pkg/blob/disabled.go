package blob

import (
	"context"
	"errors"
	"io"
)

// Disabled is a Store that refuses all operations. Use this when the
// embedding deployment keeps all per-tenant state inside the engine's
// SQLite backend and does not need separate blob storage.
type Disabled struct{}

// NewDisabled returns a Store that disallows all operations.
func NewDisabled() *Disabled { return &Disabled{} }

// ErrDisabled is returned by all Disabled methods.
var ErrDisabled = errors.New("blob: disabled")

func (Disabled) Put(ctx context.Context, workspaceID, path string, content io.Reader, opts *PutOptions) (*ObjectInfo, error) {
	return nil, ErrDisabled
}

func (Disabled) Get(ctx context.Context, workspaceID, path string) (io.ReadCloser, *ObjectInfo, error) {
	return nil, nil, ErrDisabled
}

func (Disabled) Stat(ctx context.Context, workspaceID, path string) (*ObjectInfo, error) {
	return nil, ErrDisabled
}

func (Disabled) Delete(ctx context.Context, workspaceID, path string) error { return ErrDisabled }

func (Disabled) List(ctx context.Context, workspaceID, prefix string, limit int) ([]ObjectInfo, error) {
	return nil, nil
}

func (Disabled) DeleteWorkspace(ctx context.Context, workspaceID string) error { return nil }

var _ Store = (*Disabled)(nil)
