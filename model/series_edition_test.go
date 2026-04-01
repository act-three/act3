package model

import (
	"testing"

	"ily.dev/act3/database/schema"
)

func TestNewSeriesEdition(t *testing.T) {
	sr := &SeriesHead{schema.Series{
		ID:     "series-1",
		Title:  "Test Series",
		Status: "Running",
	}}

	t.Run("creates edition with season and episodes", func(t *testing.T) {
		sedData := schema.SeriesEdition{
			ID:       "edition-1",
			Label:    "Original",
			SeriesID: "series-1",
			Summary:  "A test series",
		}
		sns := []schema.Season{
			{
				ID:        "season-1",
				EditionID: "edition-1",
				SortKey:   "01",
				Title:     "Season 1",
				Number:    1,
			},
		}
		eps := []schema.Episode{
			{
				ID:      "episode-1",
				Title:   "Pilot",
				Summary: "First episode",
				Type:    "regular",
				Runtime: 30,
			},
		}
		sneps := []schema.SeasonEpisode{
			{
				SeasonID:  "season-1",
				EpisodeID: "episode-1",
				SortKey:   1,
				Label:     "E01",
			},
		}

		sed := newSeriesEdition(sr, sedData, sns, snepMapBySeasonID(sneps), epMapByID(eps), noProgress, nil)

		if sed.ID() != "edition-1" {
			t.Errorf("expected edition ID 'edition-1', got '%s'", sed.ID())
		}
		if sed.Summary() != "A test series" {
			t.Errorf("expected summary 'A test series', got '%s'", sed.Summary())
		}

		seasonCount := 0
		for sn := range sed.Seasons() {
			seasonCount++
			if sn.Title() != "Season 1" {
				t.Errorf("expected season title 'Season 1', got '%s'", sn.Title())
			}
		}
		if seasonCount != 1 {
			t.Errorf("expected 1 season, got %d", seasonCount)
		}
	})

	t.Run("creates edition with no seasons", func(t *testing.T) {
		sedData := schema.SeriesEdition{
			ID:       "edition-1",
			Label:    "Empty",
			SeriesID: "series-1",
		}

		sed := newSeriesEdition(sr, sedData, nil, nil, nil, noProgress, nil)

		if sed.ID() != "edition-1" {
			t.Errorf("expected edition ID 'edition-1', got '%s'", sed.ID())
		}

		seasonCount := 0
		for range sed.Seasons() {
			seasonCount++
		}
		if seasonCount != 0 {
			t.Errorf("expected 0 seasons, got %d", seasonCount)
		}
	})

	t.Run("associates multiple seasons correctly", func(t *testing.T) {
		sedData := schema.SeriesEdition{
			ID:       "edition-1",
			Label:    "Edition 1",
			SeriesID: "series-1",
		}
		sns := []schema.Season{
			{
				ID:        "season-1",
				EditionID: "edition-1",
				SortKey:   "01",
				Title:     "Season 1",
			},
			{
				ID:        "season-2",
				EditionID: "edition-1",
				SortKey:   "02",
				Title:     "Season 2",
			},
		}

		sed := newSeriesEdition(sr, sedData, sns, nil, nil, noProgress, nil)

		seasonCount := 0
		for range sed.Seasons() {
			seasonCount++
		}
		if seasonCount != 2 {
			t.Errorf("expected 2 seasons, got %d", seasonCount)
		}
	})

	t.Run("sets SeriesHead back-reference", func(t *testing.T) {
		sedData := schema.SeriesEdition{
			ID:       "edition-1",
			Label:    "Original",
			SeriesID: "series-1",
		}

		sed := newSeriesEdition(sr, sedData, nil, nil, nil, noProgress, nil)

		if sed.SeriesHead().ID() != "series-1" {
			t.Errorf("expected series ID 'series-1', got '%s'", sed.SeriesHead().ID())
		}
	})
}
