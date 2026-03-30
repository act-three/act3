package norm

import (
	"testing"

	"ily.dev/act3/service/tvmaze"
)

func TestTVmazeEpisodes(t *testing.T) {
	tests := []struct {
		name      string
		eps       []tvmaze.Episode
		wantNames []string
		wantTypes []string
	}{
		{
			// Dark Shadows episodes around the Thanksgiving 1966
			// preemption: DS-108 (regular), DS-109 and DS-110
			// (preempted, no number, no airdate), DS-111 and DS-112
			// (regular, numbering continues at 109).
			// Source: TVmaze show 13129, episodes 650326–650330.
			// Sorted: DS-108, DS-111, DS-112, DS-109, DS-110
			// (by airdate, then number, then tvmazeID;
			// preempted episodes sort last due to empty airdate).
			name: "dark shadows thanksgiving preemption",
			eps: []tvmaze.Episode{
				{ID: 650326, Name: "DS-108", Season: 1966, Number: 108, Type: "regular", Airdate: "1966-11-23", Runtime: 30},
				{ID: 650327, Name: "DS-109", Season: 1966, Number: 0, Type: "insignificant_special", Airdate: "", Runtime: 30},
				{ID: 650328, Name: "DS-110", Season: 1966, Number: 0, Type: "insignificant_special", Airdate: "", Runtime: 30},
				{ID: 650329, Name: "DS-111", Season: 1966, Number: 109, Type: "regular", Airdate: "1966-11-28", Runtime: 30},
				{ID: 650330, Name: "DS-112", Season: 1966, Number: 110, Type: "regular", Airdate: "1966-11-29", Runtime: 30},
			},
			wantNames: []string{"DS-108", "DS-111", "DS-112", "DS-109", "DS-110"},
			wantTypes: []string{"regular", "regular", "regular", "insignificant_special", "insignificant_special"},
		},
		{
			// Game of Thrones season 6 specials: three
			// insignificant_specials in the same season with
			// distinct titles.
			// Source: TVmaze show 82, episodes 633741, 708089, 929474.
			name: "game of thrones season 6 specials",
			eps: []tvmaze.Episode{
				{ID: 633741, Name: "Inside Game of Thrones - The Best Seat in the House", Season: 6, Number: 0, Type: "insignificant_special", Airdate: "2016-02-29", Runtime: 5},
				{ID: 708089, Name: "Inside Game of Thrones - Prosthetics", Season: 6, Number: 0, Type: "insignificant_special", Airdate: "2016-03-29", Runtime: 4},
				{ID: 929474, Name: "18 Hours at the Paint Hall", Season: 6, Number: 0, Type: "insignificant_special", Airdate: "2016-07-03", Runtime: 30},
			},
			wantNames: []string{"Inside Game of Thrones - The Best Seat in the House", "Inside Game of Thrones - Prosthetics", "18 Hours at the Paint Hall"},
			wantTypes: []string{"insignificant_special", "insignificant_special", "insignificant_special"},
		},
		{
			// Game of Thrones season 1: regular episodes and a
			// special in the same season. The special aired before
			// the regulars, so it sorts first.
			// Source: TVmaze show 82, episodes 4952, 4953, 4993.
			name: "game of thrones regular and special",
			eps: []tvmaze.Episode{
				{ID: 4952, Name: "Winter is Coming", Season: 1, Number: 1, Type: "regular", Airdate: "2011-04-17", Runtime: 60},
				{ID: 4953, Name: "The Kingsroad", Season: 1, Number: 2, Type: "regular", Airdate: "2011-04-24", Runtime: 60},
				{ID: 4993, Name: "Inside Game of Thrones", Season: 1, Number: 0, Type: "insignificant_special", Airdate: "2010-12-05", Runtime: 11},
			},
			wantNames: []string{"Inside Game of Thrones", "Winter is Coming", "The Kingsroad"},
			wantTypes: []string{"insignificant_special", "regular", "regular"},
		},
		{
			name: "unknown type omitted",
			eps: []tvmaze.Episode{
				{ID: 1, Name: "Ep 1", Season: 1, Number: 1, Type: "regular", Airdate: "2020-01-01", Runtime: 30},
				{ID: 2, Name: "Ep 2", Season: 1, Number: 2, Type: "brand_new_type", Airdate: "2020-01-02", Runtime: 30},
				{ID: 3, Name: "Ep 3", Season: 1, Number: 3, Type: "regular", Airdate: "2020-01-03", Runtime: 30},
			},
			wantNames: []string{"Ep 1", "Ep 3"},
			wantTypes: []string{"regular", "regular"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			neps := TVmazeEpisodes(tt.eps)

			if len(neps) != len(tt.wantNames) {
				t.Fatalf("got %d episodes, want %d", len(neps), len(tt.wantNames))
			}

			for i, ne := range neps {
				if ne.Episode.Title != tt.wantNames[i] {
					t.Errorf("episode %d title = %q, want %q", i, ne.Episode.Title, tt.wantNames[i])
				}
				if ne.Episode.Type != tt.wantTypes[i] {
					t.Errorf("episode %d type = %q, want %q", i, ne.Episode.Type, tt.wantTypes[i])
				}
				if ne.SeasonEpisode.Slug == "" {
					t.Errorf("episode %d slug is empty", i)
				}
				if ne.SeasonEpisode.Number != 0 {
					t.Errorf("episode %d number = %d, want 0", i, ne.SeasonEpisode.Number)
				}
				if ne.SeasonEpisode.Label != "" {
					t.Errorf("episode %d label = %q, want empty", i, ne.SeasonEpisode.Label)
				}
			}
		})
	}
}
