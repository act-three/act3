package web

import (
	"database/sql"
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	"ily.dev/act3/view"
)

func (c *Config) browseWork(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tr *model.TxR) (html.Node, error) {
		ctx := req.Context()
		slug := req.PathValue("slug")

		sr, err := tr.SeriesBySlug(ctx, slug)
		if err == nil {
			return view.BrowseSeries(sr), nil
		}
		if err != sql.ErrNoRows {
			return nil, err
		}

		mo, err := tr.MovieBySlug(ctx, slug)
		if err == sql.ErrNoRows {
			return nil, errNotFound
		} else if err != nil {
			return nil, err
		}

		dls, dlErr := tr.RenditionForDownloadListForMovie(ctx, mo.ID())
		if dlErr != nil {
			return nil, dlErr
		}
		return view.BrowseMovie(mo, dls), nil
	})
}
