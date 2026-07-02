package storage

import (
	"errors"
	"io/fs"
	"iter"
	"path"
	"strings"
	"time"

	"ily.dev/act3/encoding/base32c"
	"kr.dev/walk"
)

// keyChars reports, per byte, membership in the storage key alphabet.
var keyChars = func() (t [256]bool) {
	for i := range len(base32c.Alphabet) {
		t[base32c.Alphabet[i]] = true
	}
	return
}()

// ScanKeys yields every substring of s that is a well-formed storage
// key, including keys embedded in longer runs of key-alphabet
// characters.
// It reports possible keys, not stored ones:
// a match need not name a blob that exists.
func ScanKeys(s string) iter.Seq[string] {
	return func(yield func(string) bool) {
		run := 0
		for i := range len(s) {
			if !keyChars[s[i]] {
				run = 0
				continue
			}
			run++
			if run >= b32Size && !yield(s[i+1-b32Size:i+1]) {
				return
			}
		}
	}
}

// Sweep removes every blob that live does not report as live
// and that was last modified before cutoff,
// along with temporary files left behind by [Dir.Copy] or
// [Dir.CreateFunc] calls interrupted by a crash,
// subject to the same cutoff.
// It returns the number of files removed.
//
// The cutoff protects blobs created but not yet recorded:
// callers must choose it comfortably older than the longest gap
// between a blob's creation and the recording of its key.
// Sweep is safe to run concurrently with blob creation and removal.
func (d *Dir) Sweep(cutoff time.Time, live func(key string) bool) (removed int, err error) {
	var errs []error
	rm := func(p string) {
		err := d.root.Remove(p)
		if err == nil {
			removed++
		} else if !errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	old := func(e fs.DirEntry) bool {
		fi, err := e.Info()
		return err == nil && fi.ModTime().Before(cutoff)
	}

	w := walk.New(d.root.FS(), ".")
	for w.Next() {
		if w.Err() != nil {
			errs = append(errs, w.Err())
			continue
		}
		e, p := w.Entry(), w.Path()
		if e.IsDir() {
			if p != "." && len(e.Name()) != 2 {
				w.SkipDir() // not a fanout dir; don't descend
			}
			continue
		}
		if !e.Type().IsRegular() || !old(e) {
			continue
		}
		switch strings.Count(p, "/") {
		case 0: // store root: temp files from interrupted writes
			if isTempName(p) {
				rm(p)
			}
		case 2: // fanout leaf: a blob
			key := strings.ReplaceAll(p, "/", "")
			if _, err := keyPath(key, path.Join); err == nil && !live(key) {
				rm(p)
			}
		}
	}
	return removed, errors.Join(errs...)
}

// isTempName reports whether name matches the form of the temporary
// names Copy and CreateFunc write under: 8 characters of the
// [crypto/rand.Text] alphabet.
func isTempName(name string) bool {
	if len(name) != 8 {
		return false
	}
	for i := range len(name) {
		c := name[i]
		if !('A' <= c && c <= 'Z' || '2' <= c && c <= '7') {
			return false
		}
	}
	return true
}
