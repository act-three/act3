package web

import (
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
)

func (w *web) listSeries(req *http.Request) (http.Handler, error) {
	return w.withTxR(func(tr *model.TxR) (http.Handler, error) {
		ctx := req.Context()
		a, err := tr.SeriesHeadList(ctx)
		if err != nil {
			return nil, err
		}
		return media("Act Three",
			html.Div(
				attr.Class("p-4"),
			)(
				html.Div(
					attr.Class("py-2 font-bold"),
				)(
					html.Text("All Series"),
				),
				html.Div()(
					html.Range(a, func(sr *model.SeriesHead) html.Node {
						return html.Div()(
							html.A(
								attr.Href(sr.PlayURL()),
							)(
								html.Text(sr.Title()),
							),
						)

					}),
				),
			),
		), nil
	})
}
