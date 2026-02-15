package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
)

func MediaListSeries(a []*model.SeriesHead) html.Node {
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
	)
}
