package storage

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestClone(t *testing.T) {
	dir := t.TempDir()
	d, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "src")
	if err := os.WriteFile(path, []byte("clone me"), 0o644); err != nil {
		t.Fatal(err)
	}
	src, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()

	key, err := d.Clone(src)
	if err != nil {
		t.Fatal(err)
	}
	f, err := d.Open(key)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "clone me" {
		t.Errorf("clone content = %q, want %q", b, "clone me")
	}

	// The temp staging name must not linger in the store root.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			t.Errorf("unexpected file in store root: %s", e.Name())
		}
	}
}
