package view

import (
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
	med := mo.EditionByTitle(model.DefaultEdition)
	return media(mo.Title(), mo.ImageURL())(
		Grid12(Class("v-detail"))(
			FlexCol(ColSpan7, Class("v-detail-info"))(
				expr.IfElse(mo.YearDisplay() != "",
					func() html.Node {
						return Text(mo.YearDisplay(), Class("v-detail-muted"))
					},
					func() html.Node { return html.Group() },
				),
				Text(mo.Title(), TextSize7),
				FlexRow(Gap3)(
					FlexCol(Class("v-detail-play"))(
						moviePlayButton(mo, med),
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
				movieAudioTrackSelect(med),
				TextNode()(html.Safe(mo.Summary())),
			),
			Box(),
			Box(ColSpan4)(
				ImageFrame()(
					PosterImg(PosterFill, attr.Src(mo.ImageURL())),
				),
			),
		),
	)
}

func moviePlayButton(mo *model.Movie, med *model.MovieEdition) html.Node {
	if med == nil {
		return Button(Disabled(true), ButtonSize3)(
			Icon("line/x-close"), Text("Play"))
	}
	v := med.Playable()
	return expr.IfElse(v != nil,
		func() html.Node {
			return Button(
				attr.Href(mo.PlayerURL(v)),
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

func movieAudioTrackSelect(med *model.MovieEdition) html.Node {
	if med == nil {
		return html.Group()
	}
	v := med.Playable()
	if v == nil {
		return html.Group()
	}
	tracks := v.AudioTracks()
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
