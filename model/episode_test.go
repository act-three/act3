package model

import (
	"context"
	"testing"

	"ily.dev/act3/database"
	"ily.dev/act3/database/schema"
)

func TestEpisodeTypeByNameMatchesSchema(t *testing.T) {
	_, dbw, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer dbw.Close()
	q := schema.New(dbw)
	ctx := context.Background()

	for name := range episodeTypeByName {
		_, err := q.EpisodeCreate(ctx, schema.EpisodeCreateParams{
			Type: name,
		})
		if err != nil {
			t.Errorf("EpisodeCreate with Type %q: %v", name, err)
		}
	}
}

func TestEpisodeHasType(t *testing.T) {
	tests := []struct {
		dbType  string
		include EpisodeType
		want    bool
	}{
		{"regular", Regular, true},
		{"regular", Significant, true},
		{"regular", AnyEpisode, true},
		{"regular", AnySpecial, false},

		{"significant_special", SignificantSpecial, true},
		{"significant_special", Significant, true},
		{"significant_special", AnySpecial, true},
		{"significant_special", AnyEpisode, true},
		{"significant_special", Regular, false},

		{"insignificant_special", InsignificantSpecial, true},
		{"insignificant_special", AnySpecial, true},
		{"insignificant_special", AnyEpisode, true},
		{"insignificant_special", Significant, false},
		{"insignificant_special", Regular, false},
	}
	for _, tt := range tests {
		ep := &Episode{ep: schema.Episode{Type: tt.dbType}}
		ep.type_ = episodeTypeByName[tt.dbType]
		got := ep.HasType(tt.include)
		if got != tt.want {
			t.Errorf("Episode{Type: %q}.HasType(%d) = %v, want %v",
				tt.dbType, tt.include, got, tt.want)
		}
	}
}
