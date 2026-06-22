package web

import (
	"ily.dev/act3/model"
)

// trashRedirectTarget returns the path the acting user should be sent
// to after trashing id, or "" if the current page is still valid. It
// runs before tx.Trash so parent lookups see the live rows. Editions
// redirect to their parent movie/series root, which re-routes to the
// newly-promoted default edition.
func trashRedirectTarget(tx *model.TxRW, id string) string {
	switch model.KindOf(id) {
	case model.TrashKindMovie:
		return "/app/movies"
	case model.TrashKindSeries:
		return "/app/series"
	case model.TrashKindEpisode:
		eps := tx.EpisodeEditions(id)
		if len(eps) == 0 { // can't happen
			return "/app/series"
		}
		return model.SeriesEditionEditorPath(
			eps[0].SeriesHead(),
			eps[0].SeriesEditionHead(),
		)
	case model.TrashKindMovieEdition:
		return tx.MovieHeadByEditionID(id).EditorPath()
	case model.TrashKindSeriesEdition:
		sed := tx.SeriesEditionHead(id)
		sr := tx.SeriesHead(sed.SeriesID())
		return sr.EditorPath()
	}
	return ""
}
