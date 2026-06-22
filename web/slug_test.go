package web

import (
	"context"
	"net/url"
	"testing"

	"ily.dev/act3/database"
	"ily.dev/act3/model"
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
		return tx.Trash(movieID)
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

func newTestApp(t *testing.T, m *model.Model, path string) *app {
	t.Helper()
	a, _ := newApp(context.Background(), &Config{Model: m}, &url.URL{Path: path})
	if a.odesc == nil {
		t.Fatalf("new app at %q has nil odesc", path)
	}
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
