package web

import (
	"net/http"

	"ily.dev/act3/model"
	"ily.dev/act3/view"
)

func (w *web) listSeries(req *http.Request) (http.Handler, error) {
	return w.withTxR(func(tr *model.TxR) (http.Handler, error) {
		ctx := req.Context()
		a, err := tr.SeriesHeadList(ctx)
		if err != nil {
			return nil, err
		}
		return page(view.MediaListSeries(a)), nil
	})
}
