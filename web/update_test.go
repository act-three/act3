package web

import (
	"errors"
	"testing"

	"ily.dev/act3/model"
	"ily.dev/act3/msg"
)

func TestSearchErrorState(t *testing.T) {
	failure := errors.New("request failed")
	for _, tt := range []struct {
		name      string
		dialog    dialog
		msg       msg.Msg
		searching bool
	}{
		{"series search", &seriesAddDialog{query: "Dune", searching: true}, &msg.SeriesSearchError{Query: "Dune", Err: failure}, false},
		{"stale series search", &seriesAddDialog{query: "Voyager", searching: true}, &msg.SeriesSearchError{Query: "Dune", Err: failure}, true},
		{"closed series search", nil, &msg.SeriesSearchError{Query: "Dune", Err: failure}, false},
		{"movie search", &movieAddDialog{query: "Dune", searching: true}, &msg.MovieSearchError{Query: "Dune", Err: failure}, false},
		{"stale movie search", &movieAddDialog{query: "Alien", searching: true}, &msg.MovieSearchError{Query: "Dune", Err: failure}, true},
		{"closed movie search", nil, &msg.MovieSearchError{Query: "Dune", Err: failure}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a := &app{dialog: tt.dialog}
			a.Update(t.Context(), tt.msg)
			var searching bool
			switch d := a.dialog.(type) {
			case *seriesAddDialog:
				searching = d.searching
			case *movieAddDialog:
				searching = d.searching
			default:
				if a.dialog != nil {
					t.Fatal("search error reopened the dialog")
				}
				return
			}
			if searching != tt.searching {
				t.Errorf("searching = %v, want %v", searching, tt.searching)
			}
		})
	}
}

func TestUpdateWritesSynchronously(t *testing.T) {
	a := &app{model: newTestModel(t)}
	var id string
	if err := a.model.WithTxRW(t.Context(), func(tx *model.TxRW) error {
		movie, err := tx.MovieCreate("Dune", "")
		if err == nil {
			id = movie.MovieEditionHead.ID()
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	a.Update(t.Context(), &msg.MovieEditionSetTitle{ID: id, Title: "Dune Part One"})
	if err := a.model.WithTxR(t.Context(), func(tx *model.TxR) error {
		if got := tx.MovieEditionHead(id).Title(); got != "Dune Part One" {
			t.Errorf("title after Update = %q, want Dune Part One", got)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestDoNavTransactionOutcome(t *testing.T) {
	failure := errors.New("write failed")
	for _, tt := range []struct {
		name string
		dest string
		err  error
	}{
		{"success", "/dune", nil},
		{"no destination", "", nil},
		{"failure without destination", "", failure},
		{"failure with destination", "/dune", failure},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a := &app{model: newTestModel(t)}
			a.doNav(t.Context(), func(tx *model.TxRW) (string, error) {
				if _, err := tx.MovieCreate("Dune", ""); err != nil {
					return "", err
				}
				return tt.dest, tt.err
			})
			count := 1
			if tt.err != nil {
				count = 0
			}
			if err := a.model.WithTxR(t.Context(), func(tx *model.TxR) error {
				if got := len(tx.MovieWorkList()); got != count {
					t.Errorf("movies after doNav = %d, want %d", got, count)
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		})
	}
}
