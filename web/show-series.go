package web

import (
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	"ily.dev/act3/view"
)

func (c *Config) showSeries(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tr *model.TxR) (html.Node, error) {
		ctx := req.Context()
		sr, err := tr.Series(ctx, req.PathValue("id"))
		if err != nil {
			return nil, err
		}
		return view.MediaSeries(sr), nil
	})
}
