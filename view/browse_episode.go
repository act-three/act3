package view

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"

	"ily.dev/act3/expr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func BrowseEpisode(
	ep *model.Episode,
	dls []*model.RenditionForDownload,
	audioOpts []model.AudioOption,
	subOpts []model.SubtitleOption,
	uploads []model.Upload,
) (title string, n domi.Node) {
	v := ep.ActiveVideo()
	return ep.Title(), browse(uploads, ep.Thumbnail())(
		Grid12(Class("v-detail"), domi.Bool("data-spoiler")(hideSpoilers(ep)))(
			FlexCol(ColSpan7, Class("v-detail-info"))(
				Link(ep.EditionTheaterPath())(Text(ep.SeriesHead().Title())),
				Grid7()(
					Text(ep.SnnEnn(), Class("v-detail-muted")),
					Box(ColSpan2)(
						Text(ep.Airdate(), Class("v-detail-muted")),
					),
					iff(ep.Runtime() != "", func() domi.Node {
						return Text(ep.Runtime()+" min", Class("v-detail-muted"))
					}),
				),
				Text(ep.Title(), Size7),
				iff(isUserAdmin(), func() domi.Node {
					return Link(
						ep.EditorPath(),
					)(Text("View in Editor", Size3,
						Style("display: inline-block"),
					))
				}),

				playForm(ep.PlayIDs(),
					FlexRow(Gap3)(
						FlexCol(Class("v-detail-play"))(
							browsePlayButton(v),
						),
						browseDownloadButton(dls),
					),
					playableAudioSelect(audioOpts),
					playableSubtitleSelect(subOpts),
				),
				Box(Class("v-detail-summary"))(
					TextNode()(domi.Safe(ep.Summary())),
					Box(Class("v-detail-spoiler-overlay")),
				),
			),
			Box(),
			Box(ColSpan4)(
				ImageFrame(Class("v-detail-thumb"))(
					PosterImg(AspectThumbnail, PosterFill, imgAttrs(ep.Thumbnail())),
					Box(Class("v-detail-spoiler-overlay")),
				),
			),
		),
	)
}

func browsePlayButton(v *model.Video) domi.Node {
	return expr.IfElse(v != nil,
		func() domi.Node {
			return Button(
				attr.Type("submit"),
				ButtonSize3)(Icon("solid/play"), Text("Start"))
		},
		func() domi.Node {
			return Button(Disabled(true), ButtonSize3)(Icon("line/x"), Text("Start"))
		},
	)
}
