package view

import (
	"ily.dev/act3/database/schema"
	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func MediaEpisode(
	ep *model.Episode,
	videos []schema.Video,
	dls []*model.RenditionForDownload,
) html.Node {
	return media(ep.Title())(
		html.Div(
			attr.Class("p-4 font-bold"),
		)(
			html.Text(ep.Title()),
		),
		html.Div(
			attr.Class("p-4"),
		)(
			expr.IfElse(len(videos) > 0,
				func() html.Node {
					vid := videos[0] // TODO(april): rules for multiple videos
					return html.Div()(
						html.Video(
							attr.Controls,
							attr.Class("w-lg"),
						)(
							html.Source(
								attr.Src("/vid/"+vid.ID+".m3u8"),
								attr.Type("application/vnd.apple.mpegurl"),
							),
						),
					)
				},
				func() html.Node {
					return Text("No Streams")
				},
			),
		),
		html.Div(
			attr.Class("p-4"),
		)(
			html.Text("Downloads:"),
			html.Range(dls, func(r *model.RenditionForDownload) html.Node {
				return html.Div()(
					html.A(
						attr.Href(r.URL()),
						attr.Download(r.Filename()),
					)(
						html.Text(r.Label()),
					),
				)
			}),
		),
	)
}
