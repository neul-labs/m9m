package blob

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestLocalFS_PutGet(t *testing.T) {
	s, err := NewLocalFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	content := []byte("hello, world")
	info, err := s.Put(ctx, "ws1", "workflows/abc.json", bytes.NewReader(content), nil)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if info.Size != int64(len(content)) {
		t.Errorf("size = %d, want %d", info.Size, len(content))
	}
	if info.SHA256 == "" {
		t.Error("SHA256 not computed")
	}

	r, gotInfo, err := s.Get(ctx, "ws1", "workflows/abc.json")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	got, _ := io.ReadAll(r)
	_ = r.Close()
	if !bytes.Equal(got, content) {
		t.Errorf("content mismatch: got %q want %q", got, content)
	}
	if gotInfo.Size != int64(len(content)) {
		t.Errorf("get info size = %d", gotInfo.Size)
	}
}

func TestLocalFS_ChecksumMismatch(t *testing.T) {
	s, err := NewLocalFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Put(context.Background(), "ws", "f", strings.NewReader("data"), &PutOptions{SHA256: "not-a-match"})
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Errorf("expected ErrChecksumMismatch, got %v", err)
	}
}

func TestLocalFS_NotFound(t *testing.T) {
	s, err := NewLocalFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = s.Get(context.Background(), "ws", "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestLocalFS_PathTraversal(t *testing.T) {
	s, err := NewLocalFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Put(context.Background(), "ws", "../escape.txt", strings.NewReader("x"), nil)
	if err == nil {
		t.Error("expected path-traversal rejection")
	}
}

func TestLocalFS_List(t *testing.T) {
	s, err := NewLocalFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, p := range []string{"a/1.json", "a/2.json", "b/1.json"} {
		if _, err := s.Put(ctx, "ws", p, strings.NewReader("x"), nil); err != nil {
			t.Fatal(err)
		}
	}
	list, err := s.List(ctx, "ws", "a", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Errorf("list under 'a' = %d, want 2", len(list))
	}
}

func TestLocalFS_DeleteWorkspace(t *testing.T) {
	s, err := NewLocalFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	_, _ = s.Put(ctx, "ws", "x", strings.NewReader("data"), nil)
	if err := s.DeleteWorkspace(ctx, "ws"); err != nil {
		t.Fatal(err)
	}
	_, _, err = s.Get(ctx, "ws", "x")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound after DeleteWorkspace, got %v", err)
	}
}
