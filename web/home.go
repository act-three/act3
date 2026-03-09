package web

import (
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	"ily.dev/act3/view"
)

func (c *Config) home(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tr *model.TxR) (html.Node, error) {
		ctx := req.Context()
		series, err := tr.SeriesHeadList(ctx)
		if err != nil {
			return nil, err
		}
		movies, err := tr.MovieHeadList(ctx)
		if err != nil {
			return nil, err
		}
		return view.Home(series, movies), nil
	})
}
