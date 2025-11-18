package web

import (
	"net/http"
	"strconv"

	"ily.dev/act3/model"
	"ily.dev/act3/web/editseries"
	"ily.dev/act3/web/list"
	"ily.dev/act3/web/turbo"
	webpkg "ily.dev/act3/web/web"
)

func (w *web) doAddSeries(req *http.Request) (h http.Handler, err error) {
	defer decorateErrorFrame("add-series-errors", &err)
	return w.withTxRW(func(tx *model.TxRW) (http.Handler, error) {
		ctx := req.Context()
		ss, err := tx.SeriesCreateByTVmazeID(ctx, req.FormValue("id"))
		if err != nil {
			return nil, err
		}
		return webpkg.Page(
			turbo.Frame("tvmaze-"+strconv.FormatInt(*ss.TVmazeID(), 10))(
				seriesResultLink(ss),
				turbo.Prepend(editseries.ListItems,
					list.Items([]*model.SeriesHead{ss}, editseries.ListItem),
				),
			),
		), nil
	})
}
