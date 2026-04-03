package view

import (
	"slices"

	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/turbo"
)

func BrowseEpisode(
	ep *model.Episode,
	dls []*model.RenditionForDownload,
) html.Node {
	return browse(ep.Title(), ep.ThumbnailURL())(
		Grid12(Class("v-detail"))(
			FlexCol(ColSpan7, Class("v-detail-info"))(
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
						attr.Style("display: inline-block"),
					))
				}),
				FlexRow(Gap3)(
					FlexCol(Class("v-detail-play"))(
						browsePlayButton(ep),
					),
					FlexCol(Class("v-detail-play"))(
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
					),
					FlexCol()(
						Button(ButtonGhost, ButtonSize3)(Icon("line/info-circle")),
					),
				),
				browseAudioTrackSelect(ep),
				Button(ButtonSurface)(Text("Subtitles")),
				TextNode()(html.Safe(ep.Summary())),
			),
			Box(),
			Box(ColSpan4)(
				ImageFrame()(
					PosterImg(PosterFill, PosterAspect169, attr.Src(ep.ThumbnailURL())),
				),
			),
		),
	)
}

func browseAudioTrackSelect(ep *model.Episode) html.Node {
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
	return Select(SelectSize3, SelectValue(tracks[0].ID()))(
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

func browsePlayButton(ep *model.Episode) html.Node {
	v := ep.Videos()
	// TODO(april): provide user-select if there are multiple videos.
	playable := slices.IndexFunc(v, func(v *model.Video) bool {
		return v.MVPlaylist() != ""
	})
	return expr.IfElse(playable >= 0,
		func() html.Node {
			return Button(
				attr.Href(ep.PlayerPath(v[playable])),
				attr.Attr("data-turbo-frame")("player"),
				ButtonSize3)(Icon("solid/play"), Text("Start"))
		},
		func() html.Node {
			return Button(Disabled(true), ButtonSize3)(Icon("line/x-close"), Text("Start"))
		},
	)
}
