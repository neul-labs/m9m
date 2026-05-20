package snapshot

import (
	"context"
	"errors"
	"io"
)

// Disabled is a Sink that refuses all operations. Use this when you do not
// want hibernation/revive — e.g. single-tenant self-hosters who keep the
// process running.
//
// Save returns an error so the engine knows hibernation is not supported and
// can keep the process alive instead.
type Disabled struct{}

// NewDisabled returns a Sink that disallows hibernation.
func NewDisabled() *Disabled { return &Disabled{} }

// ErrDisabled is returned by all Disabled methods.
var ErrDisabled = errors.New("snapshot: disabled")

func (Disabled) Save(ctx context.Context, workspaceID string, content io.Reader) (*Snapshot, error) {
	return nil, ErrDisabled
}

func (Disabled) Load(ctx context.Context, workspaceID, version string) (io.ReadCloser, *Snapshot, error) {
	return nil, nil, ErrDisabled
}

func (Disabled) List(ctx context.Context, workspaceID string, limit int) ([]Snapshot, error) {
	return nil, nil
}

func (Disabled) Delete(ctx context.Context, workspaceID, version string) error { return nil }

func (Disabled) DeleteWorkspace(ctx context.Context, workspaceID string) error { return nil }

var _ Sink = (*Disabled)(nil)
