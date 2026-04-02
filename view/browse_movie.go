package view

import (
	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/turbo"
)

func BrowseMovieEdition(
	med *model.MovieEdition,
	editions []*model.MovieWork,
	dls []*model.RenditionForDownload,
) html.Node {
	return browse(med.Title(), med.PosterPath())(
		Grid12(Class("v-detail"))(
			FlexCol(ColSpan7, Class("v-detail-info"))(
				html.If(len(editions) > 1, func() html.Node {
					return browseMovieEditionSelect(editions, med)
				}),
				Text(med.Title(), Size7),
				html.If(isUserAdmin(), func() html.Node {
					return FlexRow()(
						Link(
							med.EditorPath(),
							turbo.DataFrame("_top"),
						)(Text("View in Editor", Size3,
							attr.Style("display: inline-block"),
						)),
					)
				}),
				FlexRow(Gap8)(
					Text(med.Year(), Class("v-detail-muted")),
					Text(med.RuntimeString()+" min", Class("v-detail-muted")),
				),
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
					PosterImg(PosterFill, attr.Src(med.PosterPath())),
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
				attr.Href(med.PlayerPath(v)),
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
	return Select(SelectSize3,
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

func browseMovieEditionSelect(editions []*model.MovieWork, current *model.MovieEdition) html.Node {
	return FlexRow(Gap2, attr.Style("flex-wrap:wrap"))(
		html.Range(editions, func(ed *model.MovieWork) html.Node {
			selected := attr.Group()
			if ed.MovieEditionHead.ID() == current.ID() {
				selected = attr.Attr("data-selected")
			}
			return Button(
				ButtonSurface, ButtonSize3,
				attr.Href(ed.TheaterPath()),
				selected,
			)(Text(ed.Label()))
		}),
	)
}
