package xos

import (
	"os"
	"runtime"

	"golang.org/x/sys/unix"
)

// The clone is taken from src's descriptor, not its path,
// so it works even after the original path is unlinked.

func clone(dst string, src *os.File) error {
	err := unix.Fclonefileat(int(src.Fd()), unix.AT_FDCWD, dst, 0)
	runtime.KeepAlive(src) // hold src open across the raw-fd syscall
	return err
}

// clonefile only creates its destination;
// Darwin has no way to clone into an existing file.
func cloneInto(dst, src *os.File) error {
	return errNoCloneInto
}
