package fsinfo

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kr.dev/errorfmt"
)

// Probe attempts to match remote, which is a directory on a remote system
// (for instance, as reported by Transmission via its RPC API as DownloadDir),
// to a directory on the local system,
// under the assumption that remote is mounted on both systems.
// The name should be a path to a plain file within remote.
//
// Probe uses local mount points and suffixes of the given remote path
// to form a list of candidates.
// It then calls lstat on the given name within each candidate.
// The first successful result is assumed to be the corresponding local dir.
func Probe(remote, name string) (local string, err error) {
	defer errorfmt.Handlef("probe %s: %w", remote, &err)

	mounts, err := mountPoints()
	if err != nil {
		return "", err
	}

	parts := splitPath(remote)

	// For each mount point, try each suffix of remote's path components,
	// from longest (most specific) to the empty suffix (mount point itself).
	// The lstat is performed through os.Root so a name containing ".." or
	// an intermediate symlink out of the candidate cannot make a mismatched
	// candidate appear to match.
	for _, mount := range mounts {
		for i := range len(parts) + 1 {
			candidate := filepath.Join(mount, filepath.Join(parts[i:]...))
			root, err := os.OpenRoot(candidate)
			if err != nil {
				continue
			}
			_, err = root.Lstat(name)
			root.Close()
			if err == nil {
				return candidate, nil
			}
		}
	}

	return "", fmt.Errorf("not found")
}

func splitPath(p string) []string {
	p = filepath.Clean(p)
	if p == "/" || p == "." {
		return nil
	}
	return strings.Split(strings.TrimPrefix(p, "/"), "/")
}

func mountPoints() ([]string, error) {
	f, err := os.Open("/proc/self/mounts")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	seen := make(map[string]bool)
	var paths []string
	s := bufio.NewScanner(f)
	for s.Scan() {
		fields := strings.SplitN(s.Text(), " ", 3)
		if len(fields) < 2 {
			continue
		}
		p := fields[1]
		if !seen[p] {
			seen[p] = true
			paths = append(paths, p)
		}
	}
	return paths, s.Err()
}
