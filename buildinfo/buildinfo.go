// Package buildinfo reports metadata about the running binary.
package buildinfo

import (
	"runtime/debug"
	"time"
)

var (
	info = Info{
		StartTime: time.Now().UTC(),

		// set in embed.go
		BuildTime: "(unknown)",
		ChangeID:  "(unknown)",
		CommitID:  "(unknown)",
		Log:       "(unknown)",
	}
	bi, _ = debug.ReadBuildInfo()
)

// Info describes the running binary.
type Info struct {
	StartTime time.Time // process start time
	BuildTime string    // RFC3339 time stamp
	ChangeID  string    // jj change id of @
	CommitID  string    // jj commit id of @
	Log       string    // jj log graph of @ relative to main
}

// Get returns metadata about the running binary.
func Get() (Info, *debug.BuildInfo) {
	return info, bi
}
