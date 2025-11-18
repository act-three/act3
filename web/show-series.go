package web

import (
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
)

func (w *web) showSeries(req *http.Request) (http.Handler, error) {
	return w.withTxR(func(tr *model.TxR) (http.Handler, error) {
		ctx := req.Context()
		sr, err := tr.Series(ctx, req.PathValue("id"))
		if err != nil {
			return nil, err
		}
		sed := sr.EditionByTitle(model.AirDate)
		return media(sr.Title(),
			html.Div(
				attr.Class("p-4 font-bold"),
			)(
				html.Text(sr.Title()),
			),
			html.Div(
				attr.Class("p-4"),
			)(
				html.Text("Show: regular & specials"),
			),
			html.Div()(
				html.RangeSeq(sed.Seasons(), func(sn *model.Season) html.Node {
					return html.Div(
						attr.Class(""),
					)(
						html.Div()(html.Text(sn.Name())),
						html.Div()(
							html.RangeSeq(sn.Episodes(model.AnyEpisode), func(ep *model.Episode) html.Node {
								return html.Div()(
									html.A(
										attr.Href(ep.PlayURL()),
									)(
										html.Text(ep.Title()),
									),
								)
							}),
						),
					)
				}),
			),
		), nil
	})
}
