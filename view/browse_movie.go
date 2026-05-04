package view

import (
	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/turbo"
)

func BrowseMovieEdition(
	med *model.MovieEdition,
	editions []*model.MovieWork,
	dls []*model.RenditionForDownload,
	audioOpts []model.AudioOption,
	subOpts []model.SubtitleOption,
) html.Node {
	v := med.ActiveVideo()
	baseURL := ""
	if v != nil {
		baseURL = med.VideoPlayerPath(v)
	}
	return browse(med.Title(), med.Poster())(
		Grid12(Class("v-detail"))(
			FlexCol(ColSpan7, Class("v-detail-info"), playable(baseURL))(
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
							Style("display: inline-block"),
						)),
					)
				}),
				FlexRow(Gap8)(
					Text(med.Year(), Class("v-detail-muted")),
					Text(med.RuntimeString()+" min", Class("v-detail-muted")),
				),
				FlexRow(Gap3)(
					FlexCol(Class("v-detail-play"))(
						browseMoviePlayButton(med, v),
					),
					FlexCol(Class("v-detail-play"))(
						Button(Disabled(true), ButtonSize3)(Icon("line/x"), Text("Play from 18:02")),
					),
					FlexCol()(
						Button(Disabled(true), ButtonGhost, ButtonSize3)(
							Icon("line/check-circle")),
					),
					browseDownloadButton(dls),
				),
				playableAudioSelect(audioOpts),
				playableSubtitleSelect(subOpts),
				TextNode()(html.Safe(med.Summary())),
			),
			Box(),
			Box(ColSpan4)(
				ImageFrame()(
					PosterImg(AspectPoster, PosterFill, imgAttrs(med.PosterField())),
				),
			),
		),
	)
}

func browseMoviePlayButton(med *model.MovieEdition, v *model.Video) html.Node {
	return expr.IfElse(v != nil,
		func() html.Node {
			return Button(
				Href(med.VideoPlayerPath(v)),
				Attr("data-turbo-frame")("player"),
				playableTarget,
				ButtonSize3,
			)(Icon("solid/play"), Text("Play"))
		},
		func() html.Node {
			return Button(Disabled(true), ButtonSize3)(
				Icon("line/x"), Text("Play"))
		},
	)
}

func browseMovieEditionSelect(editions []*model.MovieWork, current *model.MovieEdition) html.Node {
	return FlexRow(Gap2, Style("flex-wrap:wrap"))(
		html.Range(editions, func(ed *model.MovieWork) html.Node {
			selected := group()
			if ed.MovieEditionHead.ID() == current.ID() {
				selected = Attr("data-selected")
			}
			return Button(
				ButtonSurface, ButtonSize3,
				Href(ed.TheaterPath()),
				selected,
			)(Text(ed.Label()))
		}),
	)
}
