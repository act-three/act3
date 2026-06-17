package view

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"

	"ily.dev/act3/expr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func BrowseMovieEdition(
	med *model.MovieEdition,
	editions []*model.MovieWork,
	dls []*model.RenditionForDownload,
	audioOpts []model.AudioOption,
	subOpts []model.SubtitleOption,
	uploads []model.Upload,
) (title string, n domi.Node) {
	v := med.ActiveVideo()
	return med.Title(), browse(uploads, med.Poster())(
		Grid12(Class("v-detail"))(
			FlexCol(ColSpan7, Class("v-detail-info"))(
				iff(len(editions) > 1, func() domi.Node {
					return browseMovieEditionSelect(editions, med)
				}),

				Text(med.Title(), Size7),
				iff(isUserAdmin(), func() domi.Node {
					return FlexRow()(
						Link(
							med.EditorPath(),
						)(Text("View in Editor", Size3,
							Style("display: inline-block"),
						)),
					)
				}),

				FlexRow(Gap8)(
					Text(med.Year(), Class("v-detail-muted")),
					Text(med.RuntimeString()+" min", Class("v-detail-muted")),
				),
				playForm(med.PlayIDs(),
					FlexRow(Gap3)(
						FlexCol(Class("v-detail-play"))(
							browseMoviePlayButton(v),
						),
						browseDownloadButton(dls),
					),
					playableAudioSelect(audioOpts),
					playableSubtitleSelect(subOpts),
				),
				TextNode()(domi.Safe(med.Summary())),
			),
			Box(),
			Box(ColSpan4)(
				ImageFrame()(
					PosterImg(AspectPoster, PosterFill, imgAttrs(med.Poster())),
				),
			),
		),
	)
}

func browseMoviePlayButton(v *model.Video) domi.Node {
	return expr.IfElse(v != nil,
		func() domi.Node {
			return Button(
				attr.Type("submit"),
				ButtonSize3,
			)(Icon("solid/play"), Text("Play"))
		},
		func() domi.Node {
			return Button(Disabled(true), ButtonSize3)(
				Icon("line/x"), Text("Play"))
		},
	)
}

func browseMovieEditionSelect(editions []*model.MovieWork, current *model.MovieEdition) domi.Node {
	return FlexRow(Gap2, Style("flex-wrap:wrap"))(
		rangeNodes(editions, func(ed *model.MovieWork) domi.Node {
			selected := group()
			if ed.MovieEditionHead.ID() == current.ID() {
				selected = ButtonSelected
			}
			return ButtonLink(ed.TheaterPath(),
				ButtonSurface, ButtonSize3,
				selected,
			)(Text(ed.Label()))
		}),
	)
}
