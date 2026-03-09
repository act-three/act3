package view

import (
	"slices"

	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func MediaMovie(
	mo *model.Movie,
	dls []*model.RenditionForDownload,
) html.Node {
	return media(mo.Title(), mo.ImageURL())(
		Grid12(Class("pt-16"))(
			FlexCol(ColSpan7, Class("gap-4"))(
				expr.IfElse(mo.YearDisplay() != "",
					func() html.Node {
						return Text(mo.YearDisplay(),
							Class("text-gray-11"))
					},
					func() html.Node { return html.Group() },
				),
				Text(mo.Title(), TextSize7),
				FlexRow(Gap3)(
					FlexCol(Class("w-[168px]"))(
						moviePlayButton(mo),
					),
					FlexCol()(
						Button(ButtonGhost, ButtonSize3)(
							Icon("line/check-circle")),
					),
					FlexCol()(
						Button(ButtonGhost, ButtonSize3)(
							Icon("line/download-01")),
					),
					FlexCol()(
						Button(ButtonGhost, ButtonSize3)(
							Icon("line/info-circle")),
					),
				),
				movieAudioTrackSelect(mo),
				TextNode()(html.Safe(mo.Summary())),
			),
			Box(),
			Box(ColSpan4)(
				Box(Class("rounded-sm overflow-hidden"))(
					html.Img(
						Class("w-full aspect-2/3 object-cover"),
						attr.Src(mo.ImageURL()),
					),
				),
			),
		),
	)
}

func moviePlayButton(mo *model.Movie) html.Node {
	v := mo.Videos()
	playable := slices.IndexFunc(v, func(v *model.Video) bool {
		return v.MVPlaylist() != ""
	})
	return expr.IfElse(playable >= 0,
		func() html.Node {
			return Button(
				attr.Href(mo.PlayerURL(v[playable])),
				attr.Attr("data-turbo-frame")("player"),
				ButtonSize3,
			)(Icon("solid/play"), Text("Play"))
		},
		func() html.Node {
			return Button(Disabled(true), ButtonSize3)(
				Icon("line/x-close"), Text("Play"))
		},
	)
}

func movieAudioTrackSelect(mo *model.Movie) html.Node {
	v := mo.Videos()
	playable := slices.IndexFunc(v, func(v *model.Video) bool {
		return v.MVPlaylist() != ""
	})
	if playable < 0 {
		return html.Group()
	}
	tracks := v[playable].AudioTracks()
	if len(tracks) == 0 {
		return html.Group()
	}
	return Select(SelectSurface, SelectSize3,
		SelectValue(tracks[0].ID()),
	)(
		SelectTrigger()(
			Icon("line/recording-01"),
			SelectLabel(tracks[0].Label()),
		),
		SelectContent()(
			html.Range(tracks, func(t *model.AudioTrack) html.Node {
				return SelectItem(t.ID())(html.Text(t.Label()))
			}),
		),
	)
}
