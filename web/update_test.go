package web

import (
	"errors"
	"testing"

	"ily.dev/act3/msg"
)

func TestUpdateErrorNotes(t *testing.T) {
	failure := errors.New("request failed")
	for _, tt := range []struct {
		name   string
		dialog dialog
		msg    msg.Msg
		notify bool
	}{
		{"action", nil, &msg.Error{Err: failure}, true},
		{"series search", &seriesAddDialog{query: "Dune", searching: true}, &msg.SeriesSearchError{Query: "Dune", Err: failure}, true},
		{"stale series search", &seriesAddDialog{query: "Voyager", searching: true}, &msg.SeriesSearchError{Query: "Dune", Err: failure}, false},
		{"closed series search", nil, &msg.SeriesSearchError{Query: "Dune", Err: failure}, false},
		{"movie search", &movieAddDialog{query: "Dune", searching: true}, &msg.MovieSearchError{Query: "Dune", Err: failure}, true},
		{"stale movie search", &movieAddDialog{query: "Alien", searching: true}, &msg.MovieSearchError{Query: "Dune", Err: failure}, false},
		{"closed movie search", nil, &msg.MovieSearchError{Query: "Dune", Err: failure}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a := &app{dialog: tt.dialog}
			if got := a.Update(t.Context(), tt.msg) != nil; got != tt.notify {
				t.Fatalf("notification returned = %v, want %v", got, tt.notify)
			}
			var searching bool
			switch d := a.dialog.(type) {
			case *seriesAddDialog:
				searching = d.searching
			case *movieAddDialog:
				searching = d.searching
			default:
				return
			}
			if searching == tt.notify {
				t.Errorf("searching = %v, want %v", searching, !tt.notify)
			}
		})
	}
}
