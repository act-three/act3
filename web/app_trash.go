package web

import (
	"context"

	"ily.dev/act3/model"
)

// trashRedirectTarget returns the path the acting user should be sent
// to after trashing id, or "" if the current page is still valid. It
// runs before tx.Trash so parent lookups see the live rows. Editions
// redirect to their parent movie/series root, which re-routes to the
// newly-promoted default edition.
func trashRedirectTarget(ctx context.Context, tx *model.TxRW, id string) (string, error) {
	switch model.KindOf(id) {
	case model.TrashKindMovie:
		return "/app/movies", nil
	case model.TrashKindSeries:
		return "/app/series", nil
	case model.TrashKindEpisode:
		eps, err := tx.EpisodeEditions(ctx, id)
		if err != nil {
			return "", err
		}
		if len(eps) == 0 {
			return "/app/series", nil
		}
		return model.SeriesEditionEditorPath(eps[0].SeriesHead(), eps[0].SeriesEditionHead()), nil
	case model.TrashKindMovieEdition:
		mo, err := tx.MovieHeadByEditionID(ctx, id)
		if err != nil {
			return "", err
		}
		return mo.EditorPath(), nil
	case model.TrashKindSeriesEdition:
		sed, err := tx.SeriesEditionHead(ctx, id)
		if err != nil {
			return "", err
		}
		sr, err := tx.SeriesHead(ctx, sed.SeriesID())
		if err != nil {
			return "", err
		}
		return sr.EditorPath(), nil
	}
	return "", nil
}
