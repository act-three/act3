package fsinfo

import "syscall"

// struct statfs { /* when _DARWIN_FEATURE_64_BIT_INODE is NOT defined */
//     short   f_otype;    /* type of file system (reserved: zero) */
//     short   f_oflags;   /* copy of mount flags (reserved: zero) */
//     long    f_bsize;    /* fundamental file system block size */
//     long    f_iosize;   /* optimal transfer block size */
//     long    f_blocks;   /* total data blocks in file system */
//     long    f_bfree;    /* free blocks in fs */
//     long    f_bavail;   /* free blocks avail to non-superuser */
//     long    f_files;    /* total file nodes in file system */
//     long    f_ffree;    /* free file nodes in fs */
//     fsid_t  f_fsid;     /* file system id */
//     uid_t   f_owner;    /* user that mounted the file system */
//     short   f_reserved1;        /* reserved for future use */
//     short   f_type;     /* type of file system (reserved) */
//     long    f_flags;    /* copy of mount flags */
//     long    f_reserved2[2];     /* reserved for future use */
//     char    f_fstypename[MFSNAMELEN]; /* fs type name */
//     char    f_mntonname[MNAMELEN];    /* directory on which mounted */
//     char    f_mntfromname[MNAMELEN];  /* mounted file system */
//     char    f_reserved3;        /* reserved for future use */
//     long    f_reserved4[4];     /* reserved for future use */
// }
// struct statfs { /* when _DARWIN_FEATURE_64_BIT_INODE is defined */
//     uint32_t    f_bsize;        /* fundamental file system block size */
//     int32_t     f_iosize;       /* optimal transfer block size */
//     uint64_t    f_blocks;       /* total data blocks in file system */
//     uint64_t    f_bfree;        /* free blocks in fs */
//     uint64_t    f_bavail;       /* free blocks avail to non-superuser */
//     uint64_t    f_files;        /* total file nodes in file system */
//     uint64_t    f_ffree;        /* free file nodes in fs */
//     fsid_t      f_fsid;         /* file system id */
//     uid_t       f_owner;        /* user that mounted the filesystem */
//     uint32_t    f_type;         /* type of filesystem */
//     uint32_t    f_flags;        /* copy of mount exported flags */
//     uint32_t    f_fssubtype;    /* fs sub-type (flavor) */
//     char        f_fstypename[MFSTYPENAMELEN];   /* fs type name */
//     char        f_mntonname[MAXPATHLEN];        /* directory on which mounted */
//     char        f_mntfromname[MAXPATHLEN];      /* mounted filesystem */
//     uint32_t    f_reserved[8];  /* For future use */
// };

func statvfsFromStatfst(stat *syscall.Statfs_t) (*statVFS, error) {
	return &statVFS{
		Bsize: uint64(stat.Bsize),
		//Frsize: uint64(stat.Frsize),
		Blocks: stat.Blocks,
		Bfree:  stat.Bfree,
		Bavail: stat.Bavail,
		Files:  stat.Files,
		Ffree:  stat.Ffree,
		Favail: stat.Ffree,         // not sure how to calculate Favail
		Flag:   uint64(stat.Flags), // assuming POSIX?
	}, nil
}
