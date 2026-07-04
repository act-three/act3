package web

import (
	"context"
	"net/url"
	"testing"

	"ily.dev/act3/database"
	"ily.dev/act3/model"
	"ily.dev/act3/model/kind"
	"ily.dev/act3/msg"
	"ily.dev/act3/storage"
)

func TestReplaceSlugSuffix(t *testing.T) {
	tests := []struct {
		name      string
		current   []string
		oldSuffix []string
		newSuffix string
		want      string
	}{
		{"theater movie", []string{"dune"}, []string{"dune"}, "/dune-part-one", "/dune-part-one"},
		{"editor movie", []string{"app", "movies", "dune"}, []string{"dune"}, "/dune-part-one", "/app/movies/dune-part-one"},
		{"editor episode", []string{"app", "series", "voyager", "s1e1"}, []string{"voyager", "s1e1"}, "/star-trek-voyager/s1e1-caretaker", "/app/series/star-trek-voyager/s1e1-caretaker"},
		{"collection playlist", []string{"phase-three", "playlist"}, []string{"phase-three", "playlist"}, "/phase-3/playlist", "/phase-3/playlist"},
		{"empty canonical path", []string{"dune"}, []string{"dune"}, "", ""},
		{"suffix longer than path", []string{"dune"}, []string{"app", "movies", "dune"}, "/dune", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := replaceSlugSuffix(tt.current, tt.oldSuffix, tt.newSuffix); got != tt.want {
				t.Fatalf("replaceSlugSuffix() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReplaceURLFollowsCanonicalSlug(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	var moviePath, medID string
	if err := m.WithTxRW(ctx, func(tx *model.TxRW) error {
		mw, err := tx.MovieCreate("Dune", "")
		if err != nil {
			return err
		}
		moviePath = mw.EditorPath()
		medID = mw.MovieEditionHead.ID()
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	a := newTestApp(t, m, moviePath)
	if err := m.WithTxRW(ctx, func(tx *model.TxRW) error {
		return tx.MovieEditionTitleSet(medID, "Dune Part One")
	}); err != nil {
		t.Fatal(err)
	}

	if c := a.Update(ctx, &msg.ModelEvent{}); c == nil {
		t.Fatal("Update(ModelEvent) returned nil cmd, want ReplaceURL cmd")
	}
	if len(a.notes) != 0 {
		t.Fatalf("notes = %v, want none", a.notes)
	}
}

func TestReplaceURLIgnoresStaleDescriptor(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	var movieID, moviePath string
	if err := m.WithTxRW(ctx, func(tx *model.TxRW) error {
		mw, err := tx.MovieCreate("Dune", "")
		if err != nil {
			return err
		}
		movieID = mw.MovieHead.ID()
		moviePath = mw.EditorPath()
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	a := newTestApp(t, m, moviePath)
	if err := m.WithTxRW(ctx, func(tx *model.TxRW) error {
		return tx.Trash(kind.Movie{}, movieID)
	}); err != nil {
		t.Fatal(err)
	}

	if c := a.Update(ctx, &msg.ModelEvent{}); c != nil {
		t.Fatalf("Update(ModelEvent) returned %T, want nil", c)
	}
	if len(a.notes) != 0 {
		t.Fatalf("notes = %v, want none", a.notes)
	}
}

// TestTombstonedSlugCanonicalized verifies that arriving at a
// tombstoned slug — on session start (a bookmark or stale link) or by
// in-session navigation — resolves the page and yields a ReplaceURL
// cmd toward the canonical path.
func TestTombstonedSlugCanonicalized(t *testing.T) {
	ctx := context.Background()
	m := newTestModel(t)
	if err := m.WithTxRW(ctx, func(tx *model.TxRW) error {
		mw, err := tx.MovieCreate("Dune", "")
		if err != nil {
			return err
		}
		return tx.MovieEditionTitleSet(mw.MovieEditionHead.ID(), "Dune Part One")
	}); err != nil {
		t.Fatal(err)
	}

	a, c := newApp(ctx, &Config{Model: m}, &url.URL{Path: "/dune"})
	if c == nil {
		t.Fatal("newApp at tombstoned /dune returned nil cmd, want ReplaceURL cmd")
	}
	if title, _ := a.View(ctx); title != "Dune Part One — Act Three" {
		t.Fatalf("View at tombstoned /dune has title %q, want the movie's", title)
	}

	a = newTestApp(t, m, "/dune-part-one")
	if c := a.Update(ctx, &msg.URLChange{URL: &url.URL{Path: "/dune"}}); c == nil {
		t.Fatal("Update(URLChange to tombstoned /dune) returned nil cmd, want ReplaceURL cmd")
	}
	if title, _ := a.View(ctx); title != "Dune Part One — Act Three" {
		t.Fatalf("View after navigating to tombstoned /dune has title %q, want the movie's", title)
	}

	// A canonical path needs no correction.
	a = newTestApp(t, m, "/dune-part-one")
	if c := a.Update(ctx, &msg.URLChange{URL: &url.URL{Path: "/dune-part-one"}}); c != nil {
		t.Fatalf("Update(URLChange to canonical path) returned %T, want nil", c)
	}
}

func newTestApp(t *testing.T, m *model.Model, path string) *app {
	t.Helper()
	a, _ := newApp(context.Background(), &Config{Model: m}, &url.URL{Path: path})
	return a
}

func newTestModel(t *testing.T) *model.Model {
	t.Helper()
	dbr, dbw, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		dbr.Close()
		dbw.Close()
	})
	sd, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m, err := model.New(dbr, dbw, model.Config{Store: sd})
	if err != nil {
		t.Fatal(err)
	}
	return m
}
