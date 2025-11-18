package fsinfo

import (
	"syscall"

	"kr.dev/errorfmt"
)

type statVFS struct {
	ID    uint32
	Bsize uint64 /* file system block size */
	//Frsize  uint64 /* fundamental fs block size */
	Blocks uint64 /* number of blocks (unit f_frsize) */
	Bfree  uint64 /* free blocks in file system */
	Bavail uint64 /* free blocks for non-root */
	Files  uint64 /* total file inodes */
	Ffree  uint64 /* free file inodes */
	Favail uint64 /* free file inodes for to non-root */
	Fsid   uint64 /* file system id */
	Flag   uint64 /* bit mask of f_flag values */
}

func statvfs(name string) (st *statVFS, err error) {
	errorfmt.Handlef("statvfs %s: %w", name, &err)
	var stat syscall.Statfs_t
	err = syscall.Statfs(name, &stat)
	if err != nil {
		return nil, err
	}
	return statvfsFromStatfst(&stat)
}
