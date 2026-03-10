package view

import (
	"slices"

	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func MediaEpisode(
	ep *model.Episode,
	dls []*model.RenditionForDownload,
) html.Node {
	return media(ep.Title(), ep.ImageURL())(
		Grid12(Class("pt-16"))(
			FlexCol(ColSpan7, Class("gap-4"))(
				Text(ep.SeriesHead().Title()),
				Grid7()(
					Text("S01E01", Class("text-gray-11")),
					Box(ColSpan2)(
						Text("2026-03-05", Class("text-gray-11")),
					),
				),
				Text(ep.Title(), TextSize7),
				FlexRow(Gap3)(
					FlexCol(Class("w-[168px]"))(
						playButton(ep),
					),
					FlexCol(Class("w-[168px]"))(
						Button(Disabled(true), ButtonSize3)(Icon("solid/play"), Text("Play from 5:18:02")),
					),
					FlexCol()(
						Button(ButtonGhost, ButtonSize3)(Icon("line/check-circle")),
					),
					FlexCol()(
						Button(ButtonGhost, ButtonSize3)(Icon("solid/check-circle")),
					),
					FlexCol()(
						Button(ButtonGhost, ButtonSize3)(Icon("line/download-01")),
						//html.Range(dls, func(r *model.RenditionForDownload) html.Node {
						//	return html.Div()(
						//		html.A(
						//			attr.Href(r.URL()),
						//			attr.Download(r.Filename()),
						//		)(
						//			html.Text(r.Label()),
						//		),
						//	)
						//}),
					),
					FlexCol()(
						Button(ButtonGhost, ButtonSize3)(Icon("line/info-circle")),
					),
				),
				audioTrackSelect(ep),
				Button(ButtonSurface)(Text("Subtitles")),
				TextNode()(html.Safe(ep.Summary())),
			),
			Box(),
			Box(ColSpan4)(
				ImageFrame()(
					PosterImg(PosterFill, PosterAspect169, attr.Src(ep.ImageURL())),
				),
			),
		),
	)
}

func audioTrackSelect(ep *model.Episode) html.Node {
	v := ep.Videos()
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
	return Select(SelectSurface, SelectSize3, SelectValue(tracks[0].ID()))(
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

func playButton(ep *model.Episode) html.Node {
	v := ep.Videos()
	// TODO(april): provide user-select if there are multiple videos.
	playable := slices.IndexFunc(v, func(v *model.Video) bool {
		return v.MVPlaylist() != ""
	})
	return expr.IfElse(playable >= 0,
		func() html.Node {
			return Button(
				attr.Href(ep.PlayerURL(v[playable])),
				attr.Attr("data-turbo-frame")("player"),
				ButtonSize3)(Icon("solid/play"), Text("Start"))
		},
		func() html.Node {
			return Button(Disabled(true), ButtonSize3)(Icon("line/x-close"), Text("Start"))
		},
	)
}
