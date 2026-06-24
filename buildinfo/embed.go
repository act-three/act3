//go:build buildinfoembed

package buildinfo

import (
	_ "embed"
)

var (
	//go:embed embed-date.txt
	date string

	//go:embed embed-changeid.txt
	changeID string

	//go:embed embed-commitid.txt
	commitID string

	//go:embed embed-log.txt
	log string
)

func init() {
	info.BuildTime = date
	info.ChangeID = changeID
	info.CommitID = commitID
	info.Log = log
}
