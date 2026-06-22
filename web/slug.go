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
	a.doR(ctx, func(tx *model.TxR) error {
		dest = leafPath(tx, a.odesc)
		return nil
	})
	if dest == "" {
		return nil
	}
	return domi.ReplaceURL[msg.Msg](dest)
}

// leafPath loads the object the descriptor addresses and returns its
// URL in the descriptor's section.
func leafPath(tx *model.TxR, odesc map[string]string) string {
	theater := odesc["section"] == sectionTheater
	switch odesc["kind"] {
	case model.KindMovieEdition:
		med := tx.MovieEdition(odesc["med"])
		if theater {
			return med.TheaterPath()
		}
		return med.EditorPath()
	case model.KindSeriesEdition:
		sed := tx.SeriesEdition(odesc["sed"])
		if theater {
			return sed.TheaterPath()
		}
		return sed.EditorPath()
	case model.KindEpisode:
		ep := tx.EpisodeInEdition(odesc["ep"], odesc["sed"])
		if theater {
			return ep.TheaterPath()
		}
		return ep.EditorPath()
	case model.KindCollectionOverview:
		col := tx.CollectionHead(odesc["col"])
		if theater {
			return col.TheaterPath()
		}
		return col.EditorPath()
	case model.KindCollectionPlaylist:
		col := tx.CollectionHead(odesc["col"])
		return col.PlaylistPath()
	}
	return ""
}
