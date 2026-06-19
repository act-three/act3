package model

import (
	"cmp"
	"slices"
)

// Work represents either a movie or a series — the common
// fields needed to display both in a unified list.
type Work interface {
	Title() string
	PosterField() (Image, []string)
	EditorPath() string
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
	Title() string                 // Movie or Episode title
	Info() []string                // Edtion Title, Series Title, etc
	ImageField() (Image, []string) // poster or thumbnail
	ImageAspect() (n, d int)       // (2, 3) or (16, 9)
	ReleaseDate() string           // YYYY-MM-DD
	Runtime() string               // "123" (minutes)
	TheaterPath() string
	PlayIDs() PlayIDs
}

// PlayIDs identifies the video to play and the content it belongs to:
// an episode within a series edition, or a movie edition.
type PlayIDs struct {
	VideoID                    string // required
	EpisodeID, SeriesEditionID string // set for episode
	MovieEditionID             string // set for movie
}

// Playable reports whether p names a video to play.
func (p PlayIDs) Playable() bool { return p.VideoID != "" }

// WorkList returns all movies and series as a unified list,
// sorted by title.
func (tx *TxR) WorkList() ([]Work, error) {
	mws, err := tx.MovieWorkList()
	if err != nil {
		return nil, err
	}
	sws, err := tx.SeriesWorkList()
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
