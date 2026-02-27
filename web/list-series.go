package web

import (
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	"ily.dev/act3/view"
)

func (c *Config) listSeries(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tr *model.TxR) (html.Node, error) {
		ctx := req.Context()
		a, err := tr.SeriesHeadList(ctx)
		if err != nil {
			return nil, err
		}
		return view.MediaListSeries(a), nil
	})
}
