package ffmpeg

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync/atomic"

	"ily.dev/act3/xos"
)

// Spool traffic — staging inputs in and collecting outputs out of
// job directories — moves multi-gigabyte media for every encode,
// so it clones (xos.Clone) rather than copies. Cloning is
// expected to work on every supported production and dev
// filesystem. When a clone fails, the operation falls back to a
// plain copy so encodes keep flowing, but the failure is loud —
// logged as an error and remembered for the UI — because a
// silently degraded spool would waste time and disk until someone
// happened to notice.

// cloneDegraded records the first clone failure since startup.
var cloneDegraded atomic.Pointer[error]

// CloneDegraded returns the first spool clone failure since startup,
// or nil if cloning is working as expected.
// A non-nil value means spool traffic has degraded to full copies
// and the storage configuration needs attention.
func CloneDegraded() error {
	if p := cloneDegraded.Load(); p != nil {
		return *p
	}
	return nil
}

// degradeClone makes a clone failure loud: it logs the failure
// and remembers the first one for [CloneDegraded].
func degradeClone(op string, err error) {
	err = fmt.Errorf("spool clone degraded to copy: %s: %w", op, err)
	cloneDegraded.CompareAndSwap(nil, &err)
	slog.Error("spool-clone-degraded", "op", op, "err", err)
}

// stageFile places src's contents at dst by cloning,
// falling back loudly to a plain copy.
func stageFile(src *os.File, dst string) error {
	err := xos.Clone(dst, src)
	if err == nil {
		return nil
	}
	degradeClone(fmt.Sprintf("stage %s", dst), err)
	return copyFileTo(src, dst)
}

// collectFile places the file at src into the open file dst by
// cloning, falling back to a plain copy. The fallback is loud
// except where no clone-into-file operation exists at all
// (Darwin's clonefile cannot target an existing file).
func collectFile(dst *os.File, src string) error {
	r, err := os.Open(src)
	if err != nil {
		return err
	}
	defer r.Close()
	err = xos.CloneInto(dst, r)
	if err == nil {
		return nil
	}
	if !errors.Is(err, xos.ErrNoCloneInto) {
		degradeClone(fmt.Sprintf("collect %s", src), err)
	}
	_, err = io.Copy(dst, r)
	return err
}

// collectInto places the job output at src into the file at dst
// (typically the store's in-progress temp file), replacing any
// content dst already has.
func collectInto(dst, src string) error {
	w, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	err = collectFile(w, src)
	if cerr := w.Close(); err == nil {
		err = cerr
	}
	return err
}

// copyFileTo is the staging fallback: a plain copy of src's
// contents to a new file at dst.
func copyFileTo(src *os.File, dst string) (err error) {
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
