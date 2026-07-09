// Package xos provides extensions to the standard os package.
package xos

import (
	"errors"
	"io"
	"os"
)

// errNoCloneInto reports that the platform has no operation that
// clones into an existing open file.
// It is deliberately not errors.ErrUnsupported,
// which syscall errnos such as EOPNOTSUPP also match:
// a clone the filesystem refused
// stays distinguishable from a clone that cannot even be attempted.
var errNoCloneInto = errors.New("no clone-into-open-file operation on this platform")

// Clone copies src's current contents to a new file at dst,
// as a copy-on-write reflink where the filesystem allows:
// ZFS block cloning on Linux, APFS clonefile on Darwin.
// Where it doesn't, Clone falls back to a plain byte copy,
// recording the degradation for [CloneDegradation];
// either way the caller gets its bytes.
// dst must not already exist,
// and nothing is left at dst when Clone fails.
// dst's file metadata is platform-dependent:
// the fallback and Linux create it with mode 0o644
// (as modified by the umask),
// while Darwin's clonefile copies src's permissions and timestamps.
func Clone(dst string, src *os.File) error {
	cerr := clone(dst, src)
	if cerr == nil {
		return nil
	}
	if err := copyFileTo(dst, src); err != nil {
		return err
	}
	degradeClone(src.Name(), dst, cerr)
	return nil
}

// CloneInto copies src's current contents into the open file dst,
// replacing whatever dst held,
// as a copy-on-write reflink where the platform and filesystem allow.
// Where they don't, CloneInto falls back to a plain byte copy.
// A fallback the filesystem forced is recorded for [CloneDegradation];
// a fallback on a platform with no clone-into operation at all
// (Darwin's clonefile cannot target an existing file) is not:
// copying is the expected mode of operation there.
// After a fallback, the file offsets of dst and src are unspecified.
func CloneInto(dst, src *os.File) error {
	cerr := cloneInto(dst, src)
	if cerr == nil {
		return nil
	}
	if err := copyInto(dst, src); err != nil {
		return err
	}
	if !errors.Is(cerr, errNoCloneInto) {
		degradeClone(src.Name(), dst.Name(), cerr)
	}
	return nil
}

// copyFileTo is the clone fallback:
// a plain byte copy of src's contents to a new file at dst.
func copyFileTo(dst string, src *os.File) (err error) {
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return err
	}
	w, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := w.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			os.Remove(dst)
		}
	}()
	_, err = io.Copy(w, src)
	return err
}

// copyInto is the clone-into fallback:
// a plain byte copy of src's contents replacing dst's.
func copyInto(dst, src *os.File) error {
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := dst.Truncate(0); err != nil {
		return err
	}
	if _, err := dst.Seek(0, io.SeekStart); err != nil {
		return err
	}
	_, err := io.Copy(dst, src)
	return err
}
