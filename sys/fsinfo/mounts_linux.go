package fsinfo

import (
	"bufio"
	"os"
	"strings"
)

// mountPoints returns the local filesystem mount points,
// parsed from /proc/self/mounts.
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
