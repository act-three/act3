package model

import (
	"path"

	"ily.dev/act3/database/schema"
	"ily.dev/act3/model/kind"
)

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
// Slugs resolve through tombstones. Callers can compare the path
// against SlugPath of the returned odesc and redirect if they differ.
//
// SlugResolve returns nil when no object is found.
func (tx *TxR) SlugResolve(path []string) map[string]string {
	s, err := tx.q.SlugGet(getSlug(0, path))
	if err != nil {
		return nil
	}
	k, err := kind.ParseSlugOwner(s.Kind)
	if err != nil {
		return nil
	}
	switch k.(type) {
	case kind.Movie:
		if len(path) > 2 {
			return nil
		}
		med, err := tx.movieEditionByID(s.Target, getSlug(1, path))
		if err != nil {
			return nil
		}
		return map[string]string{
			"kind": KindMovieEdition,
			"med":  med.ID(),
			"mo":   med.MovieHead().ID(),
		}
	case kind.Series:
		return tx.resolveSeries(s.Target, path[1:])
	case kind.Collection:
		var pageKind string
		switch {
		case len(path) == 1:
			pageKind = KindCollectionOverview
		case len(path) == 2 && path[1] == "playlist":
			pageKind = KindCollectionPlaylist
		default:
			return nil
		}
		col, ok := tx.collectionByID(s.Target)
		if !ok {
			return nil
		}
		return map[string]string{"kind": pageKind, "col": col.ID()}
	}
	return nil
}

// SlugPath returns the canonical slug path for odesc,
// or the empty string if odesc is stale, malformed, or no longer addressable.
//
// The returned path matches the paths returned
// from method TheaterPath on individual object types:
// /movie[/edition], /series[/edition][/episode], or /collection[/playlist].
// Editor routes can add their own prefix to the same slug suffix.
func (tx *TxR) SlugPath(odesc map[string]string) string {
	if odesc == nil {
		return ""
	}
	switch odesc["kind"] {
	case KindMovieEdition:
		med, ok := txfind1(tx.q.MovieEditionGet(odesc["med"]))
		if !ok || med.DeletedAt != nil || med.MovieID != odesc["mo"] {
			return ""
		}
		slug := tx.slugForTarget(kind.Movie{}, med.MovieID)
		if slug == "" {
			return ""
		}
		return path.Join("/", slug, med.Slug)
	case KindSeriesEdition:
		sed, ok := txfind1(tx.q.SeriesEditionGet(odesc["sed"]))
		if !ok || sed.DeletedAt != nil || sed.SeriesID != odesc["sr"] {
			return ""
		}
		slug := tx.slugForTarget(kind.Series{}, sed.SeriesID)
		if slug == "" {
			return ""
		}
		return path.Join("/", slug, sed.Slug)
	case KindEpisode:
		sed, ok := txfind1(tx.q.SeriesEditionGet(odesc["sed"]))
		if !ok || sed.DeletedAt != nil || sed.SeriesID != odesc["sr"] {
			return ""
		}
		snep, ok := txfind1(tx.q.SeasonEpisodeGet(schema.SeasonEpisodeGetParams{
			SeasonID:  odesc["sn"],
			EpisodeID: odesc["ep"],
		}))
		if !ok || snep.EditionID != sed.ID {
			return ""
		}
		slug := tx.slugForTarget(kind.Series{}, sed.SeriesID)
		if slug == "" {
			return ""
		}
		return path.Join("/", slug, sed.Slug, snep.Slug)
	case KindCollectionOverview:
		slug := tx.slugForTarget(kind.Collection{}, odesc["col"])
		if slug == "" {
			return ""
		}
		return path.Join("/", slug)
	case KindCollectionPlaylist:
		slug := tx.slugForTarget(kind.Collection{}, odesc["col"])
		if slug == "" {
			return ""
		}
		return path.Join("/", slug, "playlist")
	}
	return ""
}

// slugMove points target's live Slug-table row at slug, leaving a
// tombstone at the slug target previously answered to (so old URLs
// keep resolving) and displacing any tombstone holding the new slug.
// With no previous row — creation, or restore after trash removed the
// target's rows — it simply claims the slug.
func (tx *TxRW) slugMove(k kind.SlugOwner, target, slug string) error {
	if err := tx.q.SlugTombstone(target); err != nil {
		return err
	}
	return tx.q.SlugClaim(schema.SlugClaimParams{
		Slug: slug, Kind: k.String(), Target: target,
	})
}

func (tx *TxR) slugForTarget(k kind.SlugOwner, target string) string {
	s, ok := txfind1(tx.q.SlugGetByTarget(target))
	if !ok || s.Kind != k.String() {
		return ""
	}
	return s.Slug
}

// resolveSeries resolves the segments under a series slug: nothing (the
// default edition), an edition slug, an episode of the default edition,
// or an edition slug followed by an episode slug. An ambiguous single
// segment prefers a live edition over an episode over a tombstoned
// edition, so tombstones never shadow live slugs.
func (tx *TxR) resolveSeries(seriesID string, rest []string) map[string]string {
	switch len(rest) {
	case 0:
		return tx.resolveSeriesEdition(seriesID, "")
	case 1:
		if odesc := tx.resolveSeriesEdition(seriesID, rest[0]); odesc != nil {
			return odesc
		}
		if odesc := tx.resolveEpisode(seriesID, "", rest[0]); odesc != nil {
			return odesc
		}
		return tx.resolveSeriesEditionTombstone(seriesID, rest[0])
	case 2:
		return tx.resolveEpisode(seriesID, rest[0], rest[1])
	}
	return nil
}

func (tx *TxR) resolveSeriesEdition(seriesID, edSlug string) map[string]string {
	sed, err := tx.q.SeriesEditionGetBySlug(schema.SeriesEditionGetBySlugParams{
		SeriesID: seriesID,
		Slug:     edSlug,
	})

	if err != nil {
		return nil
	}
	return map[string]string{
		"kind": KindSeriesEdition,
		"sr":   sed.SeriesID,
		"sed":  sed.ID,
	}
}

func (tx *TxR) resolveSeriesEditionTombstone(seriesID, edSlug string) map[string]string {
	sed, err := tx.seriesEditionByTombstone(seriesID, edSlug)
	if err != nil {
		return nil
	}
	return map[string]string{
		"kind": KindSeriesEdition,
		"sr":   sed.SeriesID,
		"sed":  sed.ID,
	}
}

func (tx *TxR) resolveEpisode(seriesID, edSlug, epSlug string) map[string]string {
	sed, err := tx.q.SeriesEditionGetBySlug(schema.SeriesEditionGetBySlugParams{
		SeriesID: seriesID,
		Slug:     edSlug,
	})

	if err != nil && edSlug != "" {
		sed, err = tx.seriesEditionByTombstone(seriesID, edSlug)
	}
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
