package snapshot

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

func TestLocalFS_SaveLoadLatest(t *testing.T) {
	s, err := NewLocalFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	snap, err := s.Save(ctx, "ws", bytes.NewReader([]byte("snap1")))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if snap.Version == "" {
		t.Error("Save returned empty Version")
	}
	if snap.Size != int64(len("snap1")) {
		t.Errorf("size = %d", snap.Size)
	}
	if snap.SHA256 == "" {
		t.Error("Save did not compute SHA256")
	}

	r, gotSnap, err := s.Load(ctx, "ws", "")
	if err != nil {
		t.Fatalf("Load latest: %v", err)
	}
	defer r.Close()
	got, _ := io.ReadAll(r)
	if string(got) != "snap1" {
		t.Errorf("content = %q want %q", got, "snap1")
	}
	if gotSnap.Version != snap.Version {
		t.Errorf("latest version mismatch")
	}
}

func TestLocalFS_NewestFirst(t *testing.T) {
	s, err := NewLocalFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	first, _ := s.Save(ctx, "ws", bytes.NewReader([]byte("v1")))
	time.Sleep(2 * time.Millisecond)
	second, _ := s.Save(ctx, "ws", bytes.NewReader([]byte("v2")))

	list, err := s.List(ctx, "ws", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) < 2 {
		t.Fatalf("expected ≥2 snapshots, got %d", len(list))
	}
	if list[0].Version != second.Version {
		t.Errorf("List[0] = %s, want newest %s", list[0].Version, second.Version)
	}
	if list[1].Version != first.Version {
		t.Errorf("List[1] = %s, want oldest %s", list[1].Version, first.Version)
	}
}

func TestLocalFS_LoadNotFound(t *testing.T) {
	s, err := NewLocalFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = s.Load(context.Background(), "no-such-ws", "")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestLocalFS_Delete(t *testing.T) {
	s, err := NewLocalFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	snap, _ := s.Save(ctx, "ws", bytes.NewReader([]byte("x")))
	if err := s.Delete(ctx, "ws", snap.Version); err != nil {
		t.Fatal(err)
	}
	_, _, err = s.Load(ctx, "ws", snap.Version)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound after Delete, got %v", err)
	}
}

func TestLocalFS_DeleteWorkspace(t *testing.T) {
	s, err := NewLocalFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	_, _ = s.Save(ctx, "ws", bytes.NewReader([]byte("x")))
	if err := s.DeleteWorkspace(ctx, "ws"); err != nil {
		t.Fatal(err)
	}
	list, _ := s.List(ctx, "ws", 0)
	if len(list) != 0 {
		t.Errorf("expected empty list after DeleteWorkspace, got %d entries", len(list))
	}
}
