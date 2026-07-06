// Package xos provides extensions to the standard os package.
package xos

import (
	"errors"
	"os"
)

// ErrNoCloneInto reports that the platform has no operation that
// clones into an existing open file.
// It is deliberately not errors.ErrUnsupported,
// which syscall errnos such as EOPNOTSUPP also match:
// a clone the filesystem refused
// stays distinguishable from a clone that cannot even be attempted.
var ErrNoCloneInto = errors.New("no clone-into-open-file operation on this platform")

// Clone clones src's current contents to a new file at dst
// as a copy-on-write reflink:
// ZFS block cloning on Linux, APFS clonefile on Darwin.
// It clones or fails, never copies bytes:
// the caller decides what a failed clone means.
// dst must not already exist,
// and nothing is left at dst when Clone fails.
// dst's file metadata is platform-dependent:
// Linux creates it with mode 0o644 (as modified by the umask),
// while Darwin's clonefile copies src's permissions and timestamps.
func Clone(dst string, src *os.File) error { return clone(dst, src) }

// CloneInto clones src's current contents into the open file dst
// as a copy-on-write reflink, replacing whatever dst held.
// It clones or fails, never copies bytes.
// On platforms with no such operation it returns [ErrNoCloneInto].
func CloneInto(dst, src *os.File) error { return cloneInto(dst, src) }
