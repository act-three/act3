package model

import (
	"slices"
)

// Work represents either a movie or a series — the common
// fields needed to display both in a unified list.
type Work struct {
	title    string
	imageURL string
	playURL  string
}

func (w *Work) Title() string    { return w.title }
func (w *Work) ImageURL() string { return w.imageURL }
func (w *Work) PlayURL() string  { return w.playURL }

func workFromSeries(sr *SeriesHead) *Work {
	return &Work{
		title:    sr.Title(),
		imageURL: sr.TVmazeImageURL(),
		playURL:  sr.PlayURL(),
	}
}

func workFromMovie(mo *MovieHead) *Work {
	return &Work{
		title:    mo.Title(),
		imageURL: mo.ImageURL(),
		playURL:  mo.PlayURL(),
	}
}

// WorkList returns all movies and series as a unified list,
// sorted by title.
func (tx *TxR) WorkList(ctx Context) ([]*Work, error) {
	series, err := tx.q.SeriesList(ctx)
	if err != nil {
		return nil, err
	}
	movies, err := tx.q.MovieList(ctx)
	if err != nil {
		return nil, err
	}
	works := make([]*Work, 0, len(series)+len(movies))
	for i := range series {
		works = append(works, workFromSeries(&SeriesHead{series[i]}))
	}
	for i := range movies {
		works = append(works, workFromMovie(&MovieHead{movies[i]}))
	}
	slices.SortFunc(works, func(a, b *Work) int {
		if a.title < b.title {
			return -1
		}
		if a.title > b.title {
			return 1
		}
		return 0
	})
	return works, nil
}
