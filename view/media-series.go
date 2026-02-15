package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
)

func MediaSeries(sr *model.Series) html.Node {
	sed := sr.EditionByTitle(model.AirDate)
	return media(
		sr.Title(),
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
	)
}
