//go:build !linux

package fsinfo

// mountPoints simply returns {"/"} (the mount point we know exists a priori)
// on platforms with no known reasonable way to enumerate mount points.
func mountPoints() ([]string, error) {
	return []string{"/"}, nil
}
