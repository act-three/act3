package web

import (
	"context"
	"maps"

	"ily.dev/domi"

	"ily.dev/act3/model"
	"ily.dev/act3/msg"
	"ily.dev/act3/xiter"
)

// Sections a resolved object page can live in, for odesc["section"].
const (
	sectionTheater = "theater"
	sectionEditor  = "editor"
)

// follow returns a ReplaceURL cmd to update the current page's path.
// If a.odesc doesn't contain id (the current page doesn't depend on id),
// follow returns nil.
func (a *app) follow(ctx context.Context, id string) cmd {
	if !xiter.Contains(maps.Values(a.odesc), id) {
		return nil
	}
	var dest string
	a.doR(func(tx *model.TxR) (err error) {
		dest, err = leafPath(ctx, tx, a.odesc)
		return err
	})
	if dest == "" {
		return nil
	}
	return domi.ReplaceURL[msg.Msg](dest)
}

// leafPath loads the object the descriptor addresses and returns its
// URL in the descriptor's section.
func leafPath(ctx context.Context, tx *model.TxR, odesc map[string]string) (string, error) {
	theater := odesc["section"] == sectionTheater
	switch odesc["kind"] {
	case model.KindMovieEdition:
		med, err := tx.MovieEdition(ctx, odesc["med"])
		if err != nil {
			return "", err
		}
		if theater {
			return med.TheaterPath(), nil
		}
		return med.EditorPath(), nil
	case model.KindSeriesEdition:
		sed, err := tx.SeriesEdition(ctx, odesc["sed"])
		if err != nil {
			return "", err
		}
		if theater {
			return sed.TheaterPath(), nil
		}
		return sed.EditorPath(), nil
	case model.KindEpisode:
		ep, err := tx.EpisodeInEdition(ctx, odesc["ep"], odesc["sed"])
		if err != nil {
			return "", err
		}
		if theater {
			return ep.TheaterPath(), nil
		}
		return ep.EditorPath(), nil
	case model.KindCollectionOverview:
		col, err := tx.CollectionHead(ctx, odesc["col"])
		if err != nil {
			return "", err
		}
		if theater {
			return col.TheaterPath(), nil
		}
		return col.EditorPath(), nil
	case model.KindCollectionPlaylist:
		col, err := tx.CollectionHead(ctx, odesc["col"])
		if err != nil {
			return "", err
		}
		return col.PlaylistPath(), nil
	}
	return "", nil
}
