package fsinfo

import (
	"bufio"
	"errors"
	"io/fs"
	"os"
	"slices"
	"strings"
	"syscall"
)

type FS struct {
	Path  []string // mount points (including bind mounts)
	Size  uint64   // size in bytes
	Avail uint64   // available bytes
	Type  string   // filesystem type (devtmpfs, zfs, ext4, sysfs, etc)
}

// Returned slice is sorted.
// Returned path slices are sorted.
func GetInfo() ([]*FS, error) {
	f, err := os.Open("/proc/self/mounts")
	if err != nil {
		return nil, err
	}
	s := bufio.NewScanner(f)

	ms := make(map[uint64][]string)
	ts := make(map[uint64]string)
	for s.Scan() {
		fields := strings.Split(s.Text(), " ")
		if len(fields) < 2 {
			return nil, errors.New("bad mount list")
		}
		st, err := os.Lstat(fields[1])
		if errors.Is(err, fs.ErrPermission) {
			continue
		} else if err != nil {
			return nil, err
		}
		dev := uint64(st.Sys().(*syscall.Stat_t).Dev)
		ms[dev] = append(ms[dev], fields[1])
		ts[dev] = fields[2]
	}

	var a []*FS
	for dev, paths := range ms {
		st, err := statvfs(paths[0])
		if err != nil {
			return nil, err
		}
		slices.Sort(paths)
		a = append(a, &FS{
			Path:  paths,
			Size:  st.Bsize * st.Blocks,
			Avail: st.Bsize * st.Bavail,
			Type:  ts[dev],
		})
	}
	slices.SortFunc(a, func(a, b *FS) int {
		if a.Path[0] < b.Path[0] {
			return -1
		} else if a.Path[0] > b.Path[0] {
			return 1
		}
		return 0
	})
	return a, nil
}
