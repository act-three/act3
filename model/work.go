package model

import (
	"cmp"
	"slices"

	"ily.dev/act3/web/static"
)

func PosterPath(id string) string {
	if id != "" {
		return "/-/blob/" + id
	}
	return static.Path("/static/poster-fallback.png")
}

// Work represents either a movie or a series — the common
// fields needed to display both in a unified list.
type Work interface {
	Title() string
	PosterPath() string
	TheaterPath() string
}

// WorkList returns all movies and series as a unified list,
// sorted by title.
func (tx *TxR) WorkList(ctx Context) ([]Work, error) {
	mws, err := tx.MovieWorkList(ctx)
	if err != nil {
		return nil, err
	}
	sws, err := tx.SeriesWorkList(ctx)
	if err != nil {
		return nil, err
	}

	works := make([]Work, 0, len(mws)+len(sws))
	for _, mw := range mws {
		works = append(works, mw)
	}
	for _, sw := range sws {
		works = append(works, sw)
	}
	slices.SortFunc(works, func(a, b Work) int {
		return cmp.Compare(a.Title(), b.Title())
	})
	return works, nil
}
