package kind

import "fmt"

// A TorrentTarget is the kind of object a torrent download is assigned to.
//
//sumtype:decl
type TorrentTarget interface {
	fmt.Stringer
	torrentTarget()
}

func (MovieEdition) torrentTarget()  {}
func (SeriesEdition) torrentTarget() {}

var torrentTargets = []TorrentTarget{MovieEdition{}, SeriesEdition{}}

// ParseTorrentTarget returns the TorrentTarget named by s.
func ParseTorrentTarget(s string) (TorrentTarget, error) {
	for _, k := range torrentTargets {
		if k.String() == s {
			return k, nil
		}
	}
	return nil, fmt.Errorf("kind: bad TorrentTarget %q", s)
}
