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

		kind, err := tr.SlugResolve(ctx, slug0)
		if err == sql.ErrNoRows {
			return nil, errNotFound
		} else if err != nil {
			return nil, err
		}

		switch kind {
		case model.SlugSeries:
			return c.browseSeries(ctx, tr, slug0, slug1, slug2)
		case model.SlugMovie:
			return c.browseMovie(ctx, tr, slug0, slug1)
		default:
			return nil, errNotFound
		}
	})
}

func (c *Config) browseSeries(ctx model.Context, tr *model.TxR, seriesSlug, slug1, slug2 string) (html.Node, error) {
	if slug2 != "" {
		return c.browseEpisode(ctx, tr, seriesSlug, slug1, slug2)
	}
	if slug1 != "" {
		sed, err := tr.SeriesEditionBySlug(ctx, seriesSlug, slug1)
		if err == sql.ErrNoRows {
			return c.browseEpisode(ctx, tr, seriesSlug, "", slug1)
		} else if err != nil {
			return nil, err
		}
		return view.BrowseSeriesEdition(sed), nil
	}
	sed, err := tr.SeriesEditionBySlug(ctx, seriesSlug, "")
	if err != nil {
		return nil, err
	}
	return view.BrowseSeriesEdition(sed), nil
}

func (c *Config) browseMovie(ctx model.Context, tr *model.TxR, movieSlug, edSlug string) (html.Node, error) {
	med, err := tr.MovieEditionBySlug(ctx, movieSlug, edSlug)
	if err != nil {
		return nil, err
	}
	editions, err := tr.MovieEditionList(ctx, med.MovieHead())
	if err != nil {
		return nil, err
	}
	dls, err := tr.RenditionForDownloadListForMovie(ctx, med.MovieHead().ID())
	if err != nil {
		return nil, err
	}
	return view.BrowseMovieEdition(med, editions, dls), nil
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
