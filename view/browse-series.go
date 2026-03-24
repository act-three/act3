package view

import (
	"slices"

	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func BrowseSeriesEdition(sed *model.SeriesEdition) html.Node {
	seasons := slices.Values(([]*model.Season)(nil)) // empty
	if sed != nil {
		seasons = sed.Seasons()
	}
	sr := sed.SeriesHead()
	return browse(sr.Title(), sed.TVmazeImageURL())(
		Grid12(Class("v-series"))(
			Box(Class("v-series-sidebar"))(
				Box(Class("v-series-sidebar-inner"))(
					ImageFrame()(
						PosterImg(PosterFill, attr.Src(sed.TVmazeImageURL())),
					),
					Box(Class("v-series-sidebar-section"))(Text(sr.Title(), FontBold)),
					Box(Class("v-series-sidebar-section"))(
						Text("Show: regular & specials"),
					),
					html.RangeSeq(seasons, func(sn *model.Season) html.Node {
						return Box()(
							Text(sn.Name()),
						)
					}),
				),
			),
			FlexCol(Class("v-series-content"))(
				html.RangeSeq(seasons, browseSeriesSeason),
			),
		),
	)
}

func browseSeriesSeason(sn *model.Season) html.Node {
	return Box()(
		Box(Class("v-series-season-header"))(Text(sn.Name(), FontBold)),
		FlexCol(Class("v-series-episodes"))(
			html.RangeSeq(sn.Episodes(model.AnyEpisode), browseSeriesEpisode),
		),
	)
}

func browseSeriesEpisode(ep *model.Episode) html.Node {
	const doHideSpoilers = false
	spoiler := group()
	if doHideSpoilers {
		spoiler = Attr("data-spoiler")
	}
	vids := ep.Videos()
	playable := slices.IndexFunc(vids, func(v *model.Video) bool {
		return v.MVPlaylist() != ""
	})
	return Grid8(Class("v-series-episode"), spoiler)(
		FlexCol(Class("v-series-episode-info"))(
			FlexRow(Class("v-series-episode-header"))(
				Box()(
					expr.IfElse(playable >= 0,
						func() html.Node {
							return Button(
								attr.Href(ep.PlayerURL(vids[playable])),
								attr.Attr("data-turbo-frame")("player"),
								ButtonSurface,
								ButtonCircle,
							)(Icon("solid/play"))
						},
						func() html.Node {
							return Button(Disabled(true), ButtonSurface, ButtonCircle)(Icon("line/x-close"))
						},
					),
				),
				Link(ep.DetailURL(), Class("v-series-episode"))(
					FlexCol()(
						Box(Class("v-series-episode-number"))(Text(ep.SnnEnn(), FontNormal)),
						FlexRow()(
							Box()(Text(ep.Title(), Class("v-series-episode-title"))),
						),
					),
				),
			),
			Box(Class("v-series-episode-summary"))(
				TextNode(Size2, LineClamp4)(html.Safe(ep.Summary())),
				Box(Class("v-series-spoiler-overlay")),
			),
		),
		Box(HoverOverlay, Class("v-series-episode-thumb"))(
			html.A(attr.Href(ep.DetailURL()))(
				PosterImg(PosterFill, PosterAspect169, Class("v-series-episode-thumb"), attr.Src(ep.ImageURL())),
			),
			Box(Class("v-series-spoiler-overlay")),
			Box(Class("v-series-episode-progress"))(
				Progress(0.1, ProgressSM),
			),
		),
	)
}
