package model

import (
	"testing"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/model/progress"
)

var noProgress = func(string) []*progress.Item { return nil }

func TestSeriesHeadMethods(t *testing.T) {
	tvmazeID := int64(12345)

	sr := &SeriesHead{schema.Series{
		ID:          "series-1",
		Slug:        "test-series",
		Title:       "Test Series",
		Status:      "Running",
		PremieredOn: "2020-01-01",
		TVmazeID:    &tvmazeID,
	}}

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

	t.Run("Status returns correct value", func(t *testing.T) {
		if sr.Status() != "Running" {
			t.Errorf("expected 'Running', got '%s'", sr.Status())
		}
	})

	t.Run("PremieredOn returns correct value", func(t *testing.T) {
		if sr.PremieredOn() != "2020-01-01" {
			t.Errorf("expected '2020-01-01', got '%s'", sr.PremieredOn())
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

	t.Run("TheaterPath returns correct format", func(t *testing.T) {
		expected := "/test-series"
		if sr.TheaterPath() != expected {
			t.Errorf("expected '%s', got '%s'", expected, sr.TheaterPath())
		}
	})

	t.Run("EditorPath returns correct format", func(t *testing.T) {
		if sr.EditorPath() != "/app/series/test-series" {
			t.Errorf("expected '/app/series/test-series', got '%s'", sr.EditorPath())
		}
	})
}
