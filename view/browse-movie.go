package view

import (
	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func BrowseMovieEdition(
	med *model.MovieEdition,
	dls []*model.RenditionForDownload,
) html.Node {
	return browse(med.Title(), med.ImageURL())(
		Grid12(Class("v-detail"))(
			FlexCol(ColSpan7, Class("v-detail-info"))(
				expr.IfElse(med.Year() != "",
					func() html.Node {
						return Text(med.Year(), Class("v-detail-muted"))
					},
					func() html.Node { return html.Group() },
				),
				Text(med.Title(), TextSize7),
				FlexRow(Gap3)(
					FlexCol(Class("v-detail-play"))(
						browseMoviePlayButton(med),
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
				browseMovieAudioTrackSelect(med),
				TextNode()(html.Safe(med.Summary())),
			),
			Box(),
			Box(ColSpan4)(
				ImageFrame()(
					PosterImg(PosterFill, attr.Src(med.ImageURL())),
				),
			),
		),
	)
}

func browseMoviePlayButton(med *model.MovieEdition) html.Node {
	v := med.Playable()
	return expr.IfElse(v != nil,
		func() html.Node {
			return Button(
				attr.Href(med.PlayerURL(v)),
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

func browseMovieAudioTrackSelect(med *model.MovieEdition) html.Node {
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
