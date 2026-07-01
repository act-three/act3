package web

import (
	"ily.dev/act3/model"
	"ily.dev/act3/model/kind"
)

// trashRedirectTarget returns the path the acting user should be sent
// to after trashing id, or "" if the current page is still valid. It
// runs before tx.Trash so parent lookups see the live rows. Editions
// redirect to their parent movie/series root, which re-routes to the
// newly-promoted default edition.
func trashRedirectTarget(tx *model.TxRW, k kind.Trash, id string) string {
	switch k.(type) {
	case kind.Movie:
		return "/app/movies"
	case kind.Series:
		return "/app/series"
	case kind.Episode:
		eps := tx.EpisodeEditions(id)
		if len(eps) == 0 { // can't happen
			return "/app/series"
		}
		return model.SeriesEditionEditorPath(
			eps[0].SeriesHead(),
			eps[0].SeriesEditionHead(),
		)
	case kind.MovieEdition:
		return tx.MovieHeadByEditionID(id).EditorPath()
	case kind.SeriesEdition:
		sed := tx.SeriesEditionHead(id)
		sr := tx.SeriesHead(sed.SeriesID())
		return sr.EditorPath()
	default:
		return ""
	}
}
