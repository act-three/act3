package web

import (
	"database/sql"
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	"ily.dev/act3/view"
)

func (c *Config) browseWork(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tr *model.TxR) (html.Node, error) {
		ctx := req.Context()
		slug0 := req.PathValue("slug0")
		slug1 := req.PathValue("slug1")
		slug2 := req.PathValue("slug2")

		if isEpisodeSlug(slug2) {
			return c.browseEpisode(ctx, tr, slug0, slug1, slug2)
		} else if isEpisodeSlug(slug1) {
			return c.browseEpisode(ctx, tr, slug0, "", slug1)
		}

		sed, err := tr.SeriesEditionBySlug(ctx, slug0, slug1)
		if err == nil {
			return view.BrowseSeriesEdition(sed), nil
		}
		if err != sql.ErrNoRows {
			return nil, err
		}

		med, err := tr.MovieEditionBySlug(ctx, slug0, slug1)
		if err == sql.ErrNoRows {
			return nil, errNotFound
		} else if err != nil {
			return nil, err
		}
		dls, err := tr.RenditionForDownloadListForMovie(ctx, med.MovieHead().ID())
		if err != nil {
			return nil, err
		}
		return view.BrowseMovieEdition(med, dls), nil
	})
}

func (c *Config) browseEpisode(ctx model.Context, tr *model.TxR, seriesSlug, edSlug, epSlug string) (html.Node, error) {
	ep, err := tr.EpisodeBySlug(ctx, seriesSlug, edSlug, epSlug)
	if err != nil {
		return nil, err
	}
	dls, err := tr.RenditionForDownloadList(ctx, ep.ID())
	if err != nil {
		return nil, err
	}
	return view.BrowseEpisode(ep, dls), nil
}

// isEpisodeSlug reports whether slug looks like an episode slug,
// matching "sNN-" (e.g. "s02-special") or "sNNeNN-" (e.g. "s01e01-pilot").
func isEpisodeSlug(slug string) bool {
	if len(slug) < 4 || slug[0] != 's' || !isDigit(slug[1]) || !isDigit(slug[2]) {
		return false
	}
	if slug[3] == '-' {
		return true
	}
	return len(slug) >= 7 && slug[3] == 'e' && isDigit(slug[4]) && isDigit(slug[5]) && slug[6] == '-'
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }
