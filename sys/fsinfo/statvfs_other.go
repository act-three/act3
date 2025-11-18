//go:build !(linux || darwin)

package fsinfo

import "syscall"

func statvfsFromStatfst(stat *syscall.Statfs_t) (*statVFS, error) {
	return nil, syscall.ENOTSUP
}
