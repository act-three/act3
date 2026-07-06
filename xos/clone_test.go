package xos

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// newSrc creates a small source file in dir and returns it open
// for reading.
func newSrc(t *testing.T, dir string) *os.File {
	t.Helper()
	path := filepath.Join(dir, "src")
	if err := os.WriteFile(path, []byte("clone me"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func TestClone(t *testing.T) {
	dir := t.TempDir()
	src := newSrc(t, dir)
	dst := filepath.Join(dir, "dst")

	if err := Clone(dst, src); err != nil {
		t.Skipf("cloning unsupported on this filesystem: %v", err)
	}
	b, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "clone me" {
		t.Errorf("clone content = %q, want %q", b, "clone me")
	}

	// dst must not already exist.
	if err := Clone(dst, src); err == nil {
		t.Error("cloning onto an existing file succeeded")
	}
}

func TestCloneUnlinkedSource(t *testing.T) {
	dir := t.TempDir()
	src := newSrc(t, dir)
	if err := os.Remove(src.Name()); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(dir, "dst")
	if err := Clone(dst, src); err != nil {
		t.Skipf("cloning unsupported on this filesystem: %v", err)
	}
	b, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "clone me" {
		t.Errorf("clone content = %q, want %q", b, "clone me")
	}
}

func TestCloneInto(t *testing.T) {
	dir := t.TempDir()
	src := newSrc(t, dir)

	// Pre-fill dst with more bytes than src holds: the clone must
	// replace the content entirely, not just overwrite a prefix.
	dst := filepath.Join(dir, "dst")
	if err := os.WriteFile(dst, bytes.Repeat([]byte("x"), 100), 0o644); err != nil {
		t.Fatal(err)
	}
	w, err := os.OpenFile(dst, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	err = CloneInto(w, src)
	if errors.Is(err, ErrNoCloneInto) {
		if runtime.GOOS == "linux" {
			t.Fatalf("CloneInto = ErrNoCloneInto on %s", runtime.GOOS)
		}
		return
	}
	if err != nil {
		t.Skipf("cloning unsupported on this filesystem: %v", err)
	}
	b, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "clone me" {
		t.Errorf("clone content = %q, want %q", b, "clone me")
	}
}
