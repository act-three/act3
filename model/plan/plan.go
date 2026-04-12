// Package plan assigns episodes to download files
// based on SnnEnn patterns in filenames.
package plan

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Episode interface {
	ID() string
	SnnEnn() string // "S1E5" for regulars, "S1 Special" for specials
}

// A Planner contains a representation of a list of episodes.
// Its Plan method assigns some number of episodes to a given file name.
//
// The zero value of Planner is an empty planner;
// it always produces empty assignments.
type Planner struct {
	byAddr map[addr]Episode
}

// NewPlanner initializes a new Planner with the given episodes.
// Regular episodes are indexed by their season and episode numbers
// as reported by SnnEnn.
// Specials from all seasons are collected into a virtual season 0,
// numbered by their position in the input.
func NewPlanner(eps []Episode) *Planner {
	p := &Planner{byAddr: make(map[addr]Episode)}
	var nSpecial int
	for _, ep := range eps {
		s := ep.SnnEnn()
		var sn, en int
		if _, err := fmt.Sscanf(s, "S%dE%d", &sn, &en); err == nil {
			p.byAddr[addr{sn, en}] = ep
		} else if strings.HasSuffix(s, " Special") {
			nSpecial++
			p.byAddr[addr{0, nSpecial}] = ep
		} else {
			panic("plan: unexpected SnnEnn format: " + s)
		}
	}
	return p
}

// Plan finds matching episodes in p for name
// and returns the corresponding episode IDs.
// If no episode matches, Plan returns nil.
func (p *Planner) Plan(name string) []string {
	ep := p.byAddr[scanSpan(name)]
	if ep == nil {
		return nil
	}
	return []string{ep.ID()}
}

type addr struct{ snn, enn int }

var reSpan = regexp.MustCompile(`\b[Ss](\d\d)[Ee](\d\d)\b`)

func scanSpan(s string) addr {
	// TODO(april): scan more forms, incl range & list.
	m := reSpan.FindStringSubmatch(s)
	if m == nil {
		return addr{}
	}
	sn, _ := strconv.Atoi(m[1])
	en, _ := strconv.Atoi(m[2])
	return addr{sn, en}
}
