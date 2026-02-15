package web

import (
	"net/http"

	"ily.dev/act3/model"
	"ily.dev/act3/view"
)

func (w *web) showSeries(req *http.Request) (http.Handler, error) {
	return w.withTxR(func(tr *model.TxR) (http.Handler, error) {
		ctx := req.Context()
		sr, err := tr.Series(ctx, req.PathValue("id"))
		if err != nil {
			return nil, err
		}
		return page(view.MediaSeries(sr)), nil
	})
}
