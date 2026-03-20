package model

import (
	"testing"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/model/progress"
)

var noProgress = func(string) []*progress.Item { return nil }

func TestNewSeries(t *testing.T) {
	t.Run("creates series with single edition and episodes", func(t *testing.T) {
		// Setup test data
		srData := schema.Series{
			ID:             "series-1",
			Title:          "Test Series",
			Summary:        "A test series",
			Status:         "Running",
			Language:       "English",
			TVmazeImageURL: "https://example.com/image.jpg",
		}

		seds := []schema.SeriesEdition{
			{
				ID:       "edition-1",
				Title:    "Original",
				SeriesID: "series-1",
			},
		}

		sns := []schema.Season{
			{
				ID:        "season-1",
				EditionID: "edition-1",
				SortKey:   "01",
				Name:      "Season 1",
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
				SortKey:   "01",
				Label:     "E01",
			},
		}

		// Create series
		sr := newSeries(srData, seds, sns, sneps, eps, noProgress, nil)

		// Verify series head data
		if sr.ID() != "series-1" {
			t.Errorf("expected series ID 'series-1', got '%s'", sr.ID())
		}
		if sr.Title() != "Test Series" {
			t.Errorf("expected series title 'Test Series', got '%s'", sr.Title())
		}
		if sr.Summary() != "A test series" {
			t.Errorf("expected summary 'A test series', got '%s'", sr.Summary())
		}

		// Verify edition mapping by ID
		ed := sr.soByID["edition-1"]
		if ed == nil {
			t.Fatal("expected edition with ID 'edition-1' to exist")
		}
		if ed.Title() != "Original" {
			t.Errorf("expected edition title 'Original', got '%s'", ed.Title())
		}

		// Verify edition mapping by title
		edByTitle := sr.soByTitle["Original"]
		if edByTitle == nil {
			t.Fatal("expected edition with title 'Original' to exist")
		}
		if edByTitle.ID() != "edition-1" {
			t.Errorf("expected edition ID 'edition-1', got '%s'", edByTitle.ID())
		}
	})

	t.Run("creates series with multiple editions", func(t *testing.T) {
		srData := schema.Series{
			ID:    "series-2",
			Title: "Multi-Edition Series",
		}

		seds := []schema.SeriesEdition{
			{
				ID:       "edition-1",
				Title:    "Original",
				SeriesID: "series-2",
			},
			{
				ID:       "edition-2",
				Title:    "Director's Cut",
				SeriesID: "series-2",
			},
		}

		sns := []schema.Season{
			{
				ID:        "season-1",
				EditionID: "edition-1",
				Name:      "Season 1",
			},
			{
				ID:        "season-2",
				EditionID: "edition-2",
				Name:      "Season 1 DC",
			},
		}

		sr := newSeries(srData, seds, sns, nil, nil, noProgress, nil)

		// Verify both editions exist
		if len(sr.soByID) != 2 {
			t.Errorf("expected 2 editions, got %d", len(sr.soByID))
		}
		if len(sr.soByTitle) != 2 {
			t.Errorf("expected 2 editions by title, got %d", len(sr.soByTitle))
		}

		// Verify specific editions
		if sr.soByID["edition-1"] == nil {
			t.Error("expected edition-1 to exist")
		}
		if sr.soByID["edition-2"] == nil {
			t.Error("expected edition-2 to exist")
		}
		if sr.soByTitle["Original"] == nil {
			t.Error("expected 'Original' edition to exist")
		}
		if sr.soByTitle["Director's Cut"] == nil {
			t.Error("expected 'Director's Cut' edition to exist")
		}
	})

	t.Run("creates series with no editions", func(t *testing.T) {
		srData := schema.Series{
			ID:    "series-3",
			Title: "Empty Series",
		}

		sr := newSeries(srData, nil, nil, nil, nil, noProgress, nil)

		if sr.ID() != "series-3" {
			t.Errorf("expected series ID 'series-3', got '%s'", sr.ID())
		}
		if len(sr.soByID) != 0 {
			t.Errorf("expected 0 editions, got %d", len(sr.soByID))
		}
		if len(sr.soByTitle) != 0 {
			t.Errorf("expected 0 editions by title, got %d", len(sr.soByTitle))
		}
	})

	t.Run("properly associates seasons with editions", func(t *testing.T) {
		srData := schema.Series{
			ID:    "series-4",
			Title: "Season Association Test",
		}

		seds := []schema.SeriesEdition{
			{
				ID:       "edition-1",
				Title:    "Edition 1",
				SeriesID: "series-4",
			},
			{
				ID:       "edition-2",
				Title:    "Edition 2",
				SeriesID: "series-4",
			},
		}

		sns := []schema.Season{
			{
				ID:        "season-1",
				EditionID: "edition-1",
				Name:      "Season 1 for Edition 1",
			},
			{
				ID:        "season-2",
				EditionID: "edition-1",
				Name:      "Season 2 for Edition 1",
			},
			{
				ID:        "season-3",
				EditionID: "edition-2",
				Name:      "Season 1 for Edition 2",
			},
		}

		sr := newSeries(srData, seds, sns, nil, nil, noProgress, nil)

		// Verify editions exist
		ed1 := sr.soByID["edition-1"]
		ed2 := sr.soByID["edition-2"]

		if ed1 == nil {
			t.Fatal("expected edition-1 to exist")
		}
		if ed2 == nil {
			t.Fatal("expected edition-2 to exist")
		}

		// Count seasons for each edition
		ed1Seasons := 0
		for range ed1.Seasons() {
			ed1Seasons++
		}

		ed2Seasons := 0
		for range ed2.Seasons() {
			ed2Seasons++
		}

		if ed1Seasons != 2 {
			t.Errorf("expected 2 seasons for edition-1, got %d", ed1Seasons)
		}
		if ed2Seasons != 1 {
			t.Errorf("expected 1 season for edition-2, got %d", ed2Seasons)
		}
	})
}

func TestSeriesEditionByTitle(t *testing.T) {
	t.Run("returns edition when it exists", func(t *testing.T) {
		srData := schema.Series{
			ID:    "series-1",
			Title: "Test Series",
		}

		seds := []schema.SeriesEdition{
			{
				ID:       "edition-1",
				Title:    "Original",
				SeriesID: "series-1",
			},
		}

		sr := newSeries(srData, seds, nil, nil, nil, noProgress, nil)

		ed := sr.EditionByTitle("Original")
		if ed == nil {
			t.Fatal("expected edition to exist")
		}
		if ed.ID() != "edition-1" {
			t.Errorf("expected edition ID 'edition-1', got '%s'", ed.ID())
		}
	})

	t.Run("returns nil when edition does not exist", func(t *testing.T) {
		srData := schema.Series{
			ID:    "series-1",
			Title: "Test Series",
		}

		sr := newSeries(srData, nil, nil, nil, nil, noProgress, nil)

		ed := sr.EditionByTitle("Nonexistent")
		if ed != nil {
			t.Error("expected nil for nonexistent edition")
		}
	})

	t.Run("returns nil when series is nil", func(t *testing.T) {
		var sr *Series
		ed := sr.EditionByTitle("Original")
		if ed != nil {
			t.Error("expected nil for nil series")
		}
	})
}

func TestSeriesEditionSeq(t *testing.T) {
	t.Run("iterates editions in sorted order by title", func(t *testing.T) {
		srData := schema.Series{
			ID:    "series-1",
			Title: "Test Series",
		}

		seds := []schema.SeriesEdition{
			{
				ID:       "edition-3",
				Title:    "Zebra Edition",
				SeriesID: "series-1",
			},
			{
				ID:       "edition-1",
				Title:    "Alpha Edition",
				SeriesID: "series-1",
			},
			{
				ID:       "edition-2",
				Title:    "Beta Edition",
				SeriesID: "series-1",
			},
		}

		sr := newSeries(srData, seds, nil, nil, nil, noProgress, nil)

		titles := []string{}
		for ed := range sr.SeriesEditionSeq() {
			titles = append(titles, ed.Title())
		}

		expected := []string{"Alpha Edition", "Beta Edition", "Zebra Edition"}
		if len(titles) != len(expected) {
			t.Fatalf("expected %d editions, got %d", len(expected), len(titles))
		}

		for i, title := range titles {
			if title != expected[i] {
				t.Errorf("at index %d: expected '%s', got '%s'", i, expected[i], title)
			}
		}
	})

	t.Run("allows early termination", func(t *testing.T) {
		srData := schema.Series{
			ID:    "series-1",
			Title: "Test Series",
		}

		seds := []schema.SeriesEdition{
			{ID: "edition-1", Title: "A", SeriesID: "series-1"},
			{ID: "edition-2", Title: "B", SeriesID: "series-1"},
			{ID: "edition-3", Title: "C", SeriesID: "series-1"},
		}

		sr := newSeries(srData, seds, nil, nil, nil, noProgress, nil)

		count := 0
		for range sr.SeriesEditionSeq() {
			count++
			if count == 2 {
				break
			}
		}

		if count != 2 {
			t.Errorf("expected to iterate 2 times, got %d", count)
		}
	})
}

func TestSeriesHeadMethods(t *testing.T) {
	premiered := "2020-01-01"
	tvmazeID := int64(12345)

	srData := schema.Series{
		ID:             "series-1",
		Slug:           "test-series",
		Title:          "Test Series",
		Summary:        "A test summary",
		Status:         "Running",
		PremieredOn:    &premiered,
		TVmazeID:       &tvmazeID,
		TVmazeImageURL: "https://example.com/image.jpg",
	}

	sr := newSeries(srData, nil, nil, nil, nil, noProgress, nil)

	t.Run("ID returns correct value", func(t *testing.T) {
		if sr.ID() != "series-1" {
			t.Errorf("expected 'series-1', got '%s'", sr.ID())
		}
	})

	t.Run("Slug returns correct value", func(t *testing.T) {
		if sr.Slug() != "test-series" {
			t.Errorf("expected 'test-series', got '%s'", sr.Slug())
		}
	})

	t.Run("Title returns correct value", func(t *testing.T) {
		if sr.Title() != "Test Series" {
			t.Errorf("expected 'Test Series', got '%s'", sr.Title())
		}
	})

	t.Run("Summary returns correct value", func(t *testing.T) {
		if sr.Summary() != "A test summary" {
			t.Errorf("expected 'A test summary', got '%s'", sr.Summary())
		}
	})

	t.Run("Status returns correct value", func(t *testing.T) {
		if sr.Status() != "Running" {
			t.Errorf("expected 'Running', got '%s'", sr.Status())
		}
	})

	t.Run("PremieredOn returns correct value", func(t *testing.T) {
		if sr.PremieredOn() == nil {
			t.Fatal("expected non-nil premiered date")
		}
		if *sr.PremieredOn() != "2020-01-01" {
			t.Errorf("expected '2020-01-01', got '%s'", *sr.PremieredOn())
		}
	})

	t.Run("TVmazeID returns correct value", func(t *testing.T) {
		if sr.TVmazeID() == nil {
			t.Fatal("expected non-nil TVmaze ID")
		}
		if *sr.TVmazeID() != 12345 {
			t.Errorf("expected 12345, got %d", *sr.TVmazeID())
		}
	})

	t.Run("TVmazeImageURL returns correct value", func(t *testing.T) {
		if sr.TVmazeImageURL() != "https://example.com/image.jpg" {
			t.Errorf("expected 'https://example.com/image.jpg', got '%s'", sr.TVmazeImageURL())
		}
	})

	t.Run("PlayURL returns correct format", func(t *testing.T) {
		expected := "/test-series"
		if sr.PlayURL() != expected {
			t.Errorf("expected '%s', got '%s'", expected, sr.PlayURL())
		}
	})

	t.Run("EditURL returns correct format", func(t *testing.T) {
		got := sr.EditURL()
		if got == "" {
			t.Error("expected non-empty EditURL")
		}
		if got != "/app/series/test-series" {
			t.Errorf("expected '/app/series/test-series', got '%s'", got)
		}
	})
}
