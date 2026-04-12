package plan

import (
	"fmt"
	"testing"
)

type testEpisode struct {
	id      string
	season  int
	episode int
}

func (e testEpisode) ID() string { return e.id }
func (e testEpisode) SnnEnn() string {
	if e.episode == 0 {
		return fmt.Sprintf("S%d Special", e.season)
	}
	return fmt.Sprintf("S%dE%d", e.season, e.episode)
}

func TestPlan(t *testing.T) {
	eps := []Episode{
		testEpisode{"ep-s1e1", 1, 1},
		testEpisode{"ep-s1e2", 1, 2},
		testEpisode{"ep-s1sp1", 1, 0},
		testEpisode{"ep-s2e1", 2, 1},
		testEpisode{"ep-s2e2", 2, 2},
		testEpisode{"ep-s2e3", 2, 3},
		testEpisode{"ep-s2sp1", 2, 0},
	}
	p := NewPlanner(eps)

	tests := []struct {
		name string
		want []string
	}{
		// Regular episodes.
		{"Show.S01E01.720p.mkv", []string{"ep-s1e1"}},
		{"Show.S01E02.720p.mkv", []string{"ep-s1e2"}},
		{"Show.S02E01.720p.mkv", []string{"ep-s2e1"}},
		{"Show.S02E03.1080p.mkv", []string{"ep-s2e3"}},

		// Case insensitive S/E prefix.
		{"show.s01e01.mkv", []string{"ep-s1e1"}},
		{"show.s02e02.mkv", []string{"ep-s2e2"}},

		// Specials via season 0 (numbered in input order).
		{"Show.S00E01.Special.mkv", []string{"ep-s1sp1"}},
		{"Show.S00E02.Special.mkv", []string{"ep-s2sp1"}},

		// No match: season/episode out of range.
		{"Show.S03E01.mkv", nil},
		{"Show.S01E05.mkv", nil},
		{"Show.S00E03.mkv", nil},
		{"Show.S00E00.mkv", nil},

		// No SnnEnn pattern at all.
		{"Show.Episode1.mkv", nil},
		{"randomfile.txt", nil},
		{"", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.Plan(tt.name)
			if !sliceEqual(got, tt.want) {
				t.Errorf("Plan(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestPlannerZeroValue(t *testing.T) {
	var p Planner
	if got := p.Plan("Show.S01E01.mkv"); got != nil {
		t.Errorf("zero Planner.Plan() = %v, want nil", got)
	}
}

func TestNewPlannerEmpty(t *testing.T) {
	p := NewPlanner(nil)
	if got := p.Plan("Show.S01E01.mkv"); got != nil {
		t.Errorf("empty Planner.Plan() = %v, want nil", got)
	}
}

func sliceEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
