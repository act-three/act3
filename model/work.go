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

func BannerPath(id string) string {
	if id != "" {
		return "/-/blob/" + id
	}
	return static.Path("/static/banner-fallback.png")
}

func ThumbnailPath(id string) string {
	if id != "" {
		return "/-/blob/" + id
	}
	return static.Path("/static/thumbnail-fallback.png")
}

// Work represents either a movie or a series — the common
// fields needed to display both in a unified list.
type Work interface {
	Title() string
	PosterPath() string
	TheaterPath() string
	Kind() string // "movie" or "series"
}

var (
	_ Playable = (*MovieEdition)(nil)
	_ Playable = (*Episode)(nil)
)

// Playable represents something that can be played.
// It is either a movie edition or an episode.
// This interface contains the common data needed
// to display episodes and movies in a unified list.
type Playable interface {
	Title() string           // Movie or Episode title
	Info() []string          // Edtion Title, Series Title, etc
	ImagePath() string       // poster or thumbnail
	ImageAspect() (n, d int) // (2, 3) or (16, 9)
	ReleaseDate() string     // YYYY-MM-DD
	Runtime() string         // "123" (minutes)
	TheaterPath() string
	PlayerPath() string // empty if unplayable
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
