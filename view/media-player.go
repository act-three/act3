package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/turbo"
)

func MediaPlayer(v *model.Video) html.Node {
	return turbo.Frame("player")(
		html.Div(
			attr.ID("full-player"),
			Class("fixed inset-0 bg-white/50"),
			attr.Attr("data-controller")("player"),
			attr.Attr("data-player-icon-url-value")(PlyrIconURL),
			attr.Attr("data-player-title-value")(v.ID()),
			attr.Attr("data-action")("click->player#dismiss:self"),
		)(
			html.Div(Class("absolute inset-0"))(
				html.Video(
					attr.Attr("playsinline"),
					attr.Attr("controls"),
					attr.Attr("data-player-target")("video"),
					Class("max-w-full max-h-full"),
				)(
					html.Source(
						attr.Src(v.PlaylistURL()),
						attr.Type("application/vnd.apple.mpegurl"),
					),
				),
			),
		),
	)
}
