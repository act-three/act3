package xos

import (
	"os"
	"runtime"

	"golang.org/x/sys/unix"
)

// Cloning uses the FICLONE ioctl, which ZFS implements as block
// cloning (reflink), and not copy_file_range, whose contract is
// only "copy, cloning if possible": when cloning is unavailable
// (for example with zfs_bclone_enabled=0),
// copy_file_range quietly performs a full byte copy —
// exactly what Clone promises never to do.

func clone(dst string, src *os.File) error {
	w, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	err = unix.IoctlFileClone(int(w.Fd()), int(src.Fd()))
	runtime.KeepAlive(src) // hold src open across the raw-fd syscall
	if cerr := w.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(dst)
	}
	return err
}

func cloneInto(dst, src *os.File) error {
	err := unix.IoctlFileClone(int(dst.Fd()), int(src.Fd()))
	runtime.KeepAlive(dst) // hold both open across the raw-fd syscall
	runtime.KeepAlive(src)
	return err
}
