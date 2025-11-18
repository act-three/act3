package fsinfo

import "syscall"

func statvfsFromStatfst(stat *syscall.Statfs_t) (*statVFS, error) {
	return &statVFS{
		Bsize:   uint64(stat.Bsize),
		Frsize:  uint64(stat.Frsize),
		Blocks:  stat.Blocks,
		Bfree:   stat.Bfree,
		Bavail:  stat.Bavail,
		Files:   stat.Files,
		Ffree:   stat.Ffree,
		Favail:  stat.Ffree,         // not sure how to calculate Favail
		Flag:    uint64(stat.Flags), // assuming POSIX?
		Namemax: uint64(stat.Namelen),
	}, nil
}
