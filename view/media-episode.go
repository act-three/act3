package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
)

func MediaEpisode(
	ep *model.Episode,
	streams []*model.RenditionForStreaming,
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
			html.Text("Streams:"),
			html.Range(streams, func(r *model.RenditionForStreaming) html.Node {
				return html.Div()(
					html.Video(
						attr.Controls,
						attr.Class("w-sm"),
					)(
						html.Source(
							attr.Src(r.URL()),
							attr.Type("video/mp4"),
						),
					),
				)
			}),
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
