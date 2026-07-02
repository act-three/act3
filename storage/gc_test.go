package storage

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestScanKeys(t *testing.T) {
	const key = "0123456789ABCDEFGHJKMNPQ" // 24 chars, all in alphabet
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"exact", key, []string{key}},
		{"punctuated", "/-/x/" + key + ".mp4", []string{key}},
		{"embedded in run", "Z" + key, []string{"Z" + key[:23], key}},
		{"lowercase", strings.ToLower(key), nil},
		{"too short", key[:23], nil},
		{"excluded letter splits run", key[:12] + "L" + key[12:], nil},
		{"two keys", key + " " + key, []string{key, key}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slices.Collect(ScanKeys(tt.in))
			if !slices.Equal(got, tt.want) {
				t.Errorf("ScanKeys(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestSweep(t *testing.T) {
	dir := t.TempDir()
	d, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	blob := func() string {
		key, err := d.Copy(strings.NewReader("blob"))
		if err != nil {
			t.Fatal(err)
		}
		return key
	}
	liveKey, deadKey, youngKey := blob(), blob(), blob()

	write := func(name string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("AAAA2222") // stale temp
	write("BBBB3333") // fresh temp
	write("notes.txt")

	backdate := func(name string) {
		old := time.Now().Add(-48 * time.Hour)
		if err := os.Chtimes(filepath.Join(dir, name), old, old); err != nil {
			t.Fatal(err)
		}
	}
	for _, key := range []string{liveKey, deadKey} {
		p, err := keyPath(key, filepath.Join)
		if err != nil {
			t.Fatal(err)
		}
		backdate(p)
	}
	backdate("AAAA2222")
	backdate("notes.txt")

	cutoff := time.Now().Add(-24 * time.Hour)
	removed, err := d.Sweep(cutoff, func(key string) bool { return key == liveKey })
	if err != nil {
		t.Fatal(err)
	}
	if removed != 2 {
		t.Errorf("Sweep removed %d files, want 2", removed)
	}

	open := func(key string) error {
		f, err := d.Open(key)
		if err == nil {
			f.Close()
		}
		return err
	}
	if err := open(liveKey); err != nil {
		t.Errorf("live blob: %v", err)
	}
	if err := open(youngKey); err != nil {
		t.Errorf("young blob: %v", err)
	}
	if err := open(deadKey); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("dead blob: err = %v, want fs.ErrNotExist", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "AAAA2222")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("stale temp: err = %v, want fs.ErrNotExist", err)
	}
	for _, name := range []string{"BBBB3333", "notes.txt"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}
