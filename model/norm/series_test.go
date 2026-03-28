package norm

import (
	"testing"

	"ily.dev/act3/service/tvmaze"
)

func TestTVmazeEpisodes(t *testing.T) {
	tests := []struct {
		name         string
		seriesSlug   string
		eps          []tvmaze.Episode
		wantSlugs    []string
		wantSortKeys []string
		wantLabels   []string
		wantTypes    []string
	}{
		{
			// Dark Shadows episodes around the Thanksgiving 1966
			// preemption: DS-108 (regular), DS-109 and DS-110
			// (preempted, no number, no airdate), DS-111 and DS-112
			// (regular, numbering continues at 109).
			// Source: TVmaze show 13129, episodes 650326–650330.
			name:       "dark shadows thanksgiving preemption",
			seriesSlug: "dark-shadows",
			eps: []tvmaze.Episode{
				{ID: 650326, Name: "DS-108", Season: 1966, Number: 108, Type: "regular", Airdate: "1966-11-23", Runtime: 30},
				{ID: 650327, Name: "DS-109", Season: 1966, Number: 0, Type: "insignificant_special", Airdate: "", Runtime: 30},
				{ID: 650328, Name: "DS-110", Season: 1966, Number: 0, Type: "insignificant_special", Airdate: "", Runtime: 30},
				{ID: 650329, Name: "DS-111", Season: 1966, Number: 109, Type: "regular", Airdate: "1966-11-28", Runtime: 30},
				{ID: 650330, Name: "DS-112", Season: 1966, Number: 110, Type: "regular", Airdate: "1966-11-29", Runtime: 30},
			},
			wantSlugs: []string{
				"dark-shadows/s1966e108-ds-108",
				"dark-shadows/s1966-special-ds-109",
				"dark-shadows/s1966-special-ds-110",
				"dark-shadows/s1966e109-ds-111",
				"dark-shadows/s1966e110-ds-112",
			},
			wantSortKeys: []string{
				"1966-11-23-00108-650326",
				"AAAA-AA-AA-AAAAA-650327",
				"AAAA-AA-AA-AAAAA-650328",
				"1966-11-28-00109-650329",
				"1966-11-29-00110-650330",
			},
			wantLabels: []string{"108", "Special", "Special", "109", "110"},
			wantTypes:  []string{"regular", "insignificant_special", "insignificant_special", "regular", "regular"},
		},
		{
			// Game of Thrones season 6 specials: three
			// insignificant_specials in the same season with
			// distinct titles.
			// Source: TVmaze show 82, episodes 633741, 708089, 929474.
			name:       "game of thrones season 6 specials",
			seriesSlug: "game-of-thrones",
			eps: []tvmaze.Episode{
				{ID: 633741, Name: "Inside Game of Thrones - The Best Seat in the House", Season: 6, Number: 0, Type: "insignificant_special", Airdate: "2016-02-29", Runtime: 5},
				{ID: 708089, Name: "Inside Game of Thrones - Prosthetics", Season: 6, Number: 0, Type: "insignificant_special", Airdate: "2016-03-29", Runtime: 4},
				{ID: 929474, Name: "18 Hours at the Paint Hall", Season: 6, Number: 0, Type: "insignificant_special", Airdate: "2016-07-03", Runtime: 30},
			},
			wantSlugs: []string{
				"game-of-thrones/s06-special-inside-game-of-thrones-the-best-seat-in-the-house",
				"game-of-thrones/s06-special-inside-game-of-thrones-prosthetics",
				"game-of-thrones/s06-special-18-hours-at-the-paint-hall",
			},
			wantSortKeys: []string{
				"2016-02-29-AAAAA-633741",
				"2016-03-29-AAAAA-708089",
				"2016-07-03-AAAAA-929474",
			},
			wantLabels: []string{"Special", "Special", "Special"},
			wantTypes:  []string{"insignificant_special", "insignificant_special", "insignificant_special"},
		},
		{
			// Game of Thrones season 1: regular episodes and a
			// special in the same season.
			// Source: TVmaze show 82, episodes 4952, 4953, 4993.
			name:       "game of thrones regular and special",
			seriesSlug: "game-of-thrones",
			eps: []tvmaze.Episode{
				{ID: 4952, Name: "Winter is Coming", Season: 1, Number: 1, Type: "regular", Airdate: "2011-04-17", Runtime: 60},
				{ID: 4953, Name: "The Kingsroad", Season: 1, Number: 2, Type: "regular", Airdate: "2011-04-24", Runtime: 60},
				{ID: 4993, Name: "Inside Game of Thrones", Season: 1, Number: 0, Type: "insignificant_special", Airdate: "2010-12-05", Runtime: 11},
			},
			wantSlugs: []string{
				"game-of-thrones/s01e01-winter-is-coming",
				"game-of-thrones/s01e02-the-kingsroad",
				"game-of-thrones/s01-special-inside-game-of-thrones",
			},
			wantSortKeys: []string{
				"2011-04-17-00001-4952",
				"2011-04-24-00002-4953",
				"2010-12-05-AAAAA-4993",
			},
			wantLabels: []string{"1", "2", "Special"},
			wantTypes:  []string{"regular", "regular", "insignificant_special"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			neps := TVmazeEpisodes(tt.seriesSlug, tt.eps)

			if len(neps) != len(tt.wantSlugs) {
				t.Fatalf("got %d episodes, want %d", len(neps), len(tt.wantSlugs))
			}

			for i, ne := range neps {
				if ne.Episode.Slug != tt.wantSlugs[i] {
					t.Errorf("episode %d slug = %q, want %q", i, ne.Episode.Slug, tt.wantSlugs[i])
				}
				if ne.SeasonEpisode.SortKey != tt.wantSortKeys[i] {
					t.Errorf("episode %d sortKey = %q, want %q", i, ne.SeasonEpisode.SortKey, tt.wantSortKeys[i])
				}
				if ne.SeasonEpisode.Label != tt.wantLabels[i] {
					t.Errorf("episode %d label = %q, want %q", i, ne.SeasonEpisode.Label, tt.wantLabels[i])
				}
				if ne.Episode.Type != tt.wantTypes[i] {
					t.Errorf("episode %d type = %q, want %q", i, ne.Episode.Type, tt.wantTypes[i])
				}
			}
		})
	}
}
