package ffmpeg

import (
	"os"

	"ily.dev/act3/xos"
)

// Spool traffic — staging inputs in and collecting outputs out of
// job directories — moves multi-gigabyte media for every encode, so
// it clones rather than copies (xos.Clone, xos.CloneInto). Cloning
// is expected to work on every supported production and dev
// filesystem; a fallback to a plain copy keeps encodes flowing but
// is loud (xos.CloneDegradation), because a silently degraded spool
// would waste time and disk until someone happened to notice.

// collectFile places the file at src into the open file dst.
func collectFile(dst *os.File, src string) error {
	r, err := os.Open(src)
	if err != nil {
		return err
	}
	defer r.Close()
	return xos.CloneInto(dst, r)
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
