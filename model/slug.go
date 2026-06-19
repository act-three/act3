package model

import "ily.dev/act3/database/schema"

// KindMovieEdition etc discriminate object pages.
// Used as key "kind" in an object descriptor.
const (
	KindMovieEdition       = "movie-edition"
	KindSeriesEdition      = "series-edition"
	KindEpisode            = "episode"
	KindCollectionOverview = "collection-overview"
	KindCollectionPlaylist = "collection-playlist"
)

// SlugResolve resolves path (a URL path containing slugs)
// to an object descriptor.
//
// The returned odesc holds the object's "kind" and the ids of the object
// and every container whose identity or numbering its slug depends on.
// The keys are named after the object id prefixes used in the database,
// such as "med" for a movie edition, "ep" for an episode, etc.
//
// SlugResolve returns nil when no object is found.
func (tx *TxR) SlugResolve(path []string) map[string]string {
	slug := getSlug(0, path)
	s, err := tx.q.SlugGet(slug)
	if err != nil {
		return nil
	}
	switch s.Kind {
	case "movie":
		if len(path) > 2 {
			return nil
		}
		med, err := tx.MovieEditionBySlug(slug, getSlug(1, path))
		if err != nil {
			return nil
		}
		return map[string]string{
			"kind": KindMovieEdition,
			"med":  med.ID(),
			"mo":   med.MovieHead().ID(),
		}
	case "series":
		return tx.resolveSeries(slug, path[1:])
	case "collection":
		var kind string
		switch {
		case len(path) == 1:
			kind = KindCollectionOverview
		case len(path) == 2 && path[1] == "playlist":
			kind = KindCollectionPlaylist
		default:
			return nil
		}
		col, err := tx.CollectionBySlug(slug)
		if err != nil {
			return nil
		}
		return map[string]string{"kind": kind, "col": col.ID()}
	}
	return nil
}

// resolveSeries resolves the segments under a series slug: nothing (the
// default edition), an edition slug, an episode of the default edition,
// or an edition slug followed by an episode slug.
func (tx *TxR) resolveSeries(seriesSlug string, rest []string) map[string]string {
	switch len(rest) {
	case 0:
		sed, err := tx.SeriesEditionBySlug(seriesSlug, "")
		if err != nil {
			return nil
		}
		return map[string]string{
			"kind": KindSeriesEdition,
			"sr":   sed.SeriesHead().ID(),
			"sed":  sed.ID(),
		}
	case 1:
		if sed, err := tx.SeriesEditionBySlug(seriesSlug, rest[0]); err == nil {
			return map[string]string{
				"kind": KindSeriesEdition,
				"sr":   sed.SeriesHead().ID(),
				"sed":  sed.ID(),
			}
		}
		return tx.resolveEpisode(seriesSlug, "", rest[0])
	case 2:
		return tx.resolveEpisode(seriesSlug, rest[0], rest[1])
	}
	return nil
}

func (tx *TxR) resolveEpisode(seriesSlug, edSlug, epSlug string) map[string]string {
	sed, err := tx.q.SeriesEditionGetBySlug(schema.SeriesEditionGetBySlugParams{
		SeriesSlug:  seriesSlug,
		EditionSlug: edSlug,
	})

	if err != nil {
		return nil
	}
	snep, err := tx.q.SeasonEpisodeGetBySlug(schema.SeasonEpisodeGetBySlugParams{
		EditionID: sed.ID,
		Slug:      epSlug,
	})

	if err != nil {
		return nil
	}
	return map[string]string{
		"kind": KindEpisode,
		"sr":   sed.SeriesID,
		"sed":  sed.ID,
		"sn":   snep.SeasonID,
		"ep":   snep.EpisodeID,
	}
}

func getSlug(i int, a []string) string {
	if i >= len(a) {
		return ""
	}
	return a[i]
}
