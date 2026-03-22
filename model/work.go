package model

import (
	"cmp"
	"slices"
)

// Work represents either a movie or a series — the common
// fields needed to display both in a unified list.
type Work interface {
	Title() string
	ImageURL() string
	PlayURL() string
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
