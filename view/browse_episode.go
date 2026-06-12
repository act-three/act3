package view

import (
	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/turbo"
)

func BrowseEpisode(
	ep *model.Episode,
	dls []*model.RenditionForDownload,
	audioOpts []model.AudioOption,
	subOpts []model.SubtitleOption,
) html.Node {
	v := ep.ActiveVideo()
	baseURL := ""
	if v != nil {
		baseURL = ep.VideoPlayerPath(v)
	}
	spoiler := group()
	if hideSpoilers(ep) {
		spoiler = Attr("data-spoiler")
	}
	return browse(ep.Title(), ep.Thumbnail())(
		Grid12(Class("v-detail"), spoiler)(
			FlexCol(ColSpan7, Class("v-detail-info"), playable(baseURL))(
				Link(ep.EditionTheaterPath())(Text(ep.SeriesHead().Title())),
				Grid7()(
					Text(ep.SnnEnn(), Class("v-detail-muted")),
					Box(ColSpan2)(
						Text(ep.Airdate(), Class("v-detail-muted")),
					),
				),
				Text(ep.Title(), Size7),
				html.If(isUserAdmin(), func() html.Node {
					return Link(
						ep.EditorPath(),
						turbo.DataFrame("_top"),
					)(Text("View in Editor", Size3,
						Style("display: inline-block"),
					))
				}),
				FlexRow(Gap3)(
					FlexCol(Class("v-detail-play"))(
						browsePlayButton(ep, v),
					),
					browseDownloadButton(dls),
				),
				playableAudioSelect(audioOpts),
				playableSubtitleSelect(subOpts),
				Box(Class("v-detail-summary"))(
					TextNode()(html.Safe(ep.Summary())),
					Box(Class("v-detail-spoiler-overlay")),
				),
			),
			Box(),
			Box(ColSpan4)(
				ImageFrame(Class("v-detail-thumb"))(
					PosterImg(AspectThumbnail, PosterFill, imgAttrs(ep.ThumbnailField())),
					Box(Class("v-detail-spoiler-overlay")),
				),
			),
		),
	)
}

func browsePlayButton(ep *model.Episode, v *model.Video) html.Node {
	return expr.IfElse(v != nil,
		func() html.Node {
			return Button(
				Href(ep.VideoPlayerPath(v)),
				Attr("data-turbo-frame")("player"),
				playableTarget,
				ButtonSize3)(Icon("solid/play"), Text("Start"))
		},
		func() html.Node {
			return Button(Disabled(true), ButtonSize3)(Icon("line/x"), Text("Start"))
		},
	)
}
