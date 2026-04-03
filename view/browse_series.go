package view

import (
	"slices"

	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/stimulus"
	"ily.dev/act3/ui/turbo"
)

func BrowseSeriesEdition(sed *model.SeriesEdition, editions []*model.SeriesWork) html.Node {
	seasons := slices.Values(([]*model.Season)(nil)) // empty
	if sed != nil {
		seasons = sed.Seasons()
	}
	sr := sed.SeriesHead()
	return browse(sr.Title(), sed.PosterPath())(
		Grid12(
			Class("v-series"),
			stimulus.Controller("series"),
			stimulus.Value("series", "mode")("all"),
		)(
			FlexCol(Class("v-series-sidebar"))(
				FlexCol(Class("v-series-sidebar-inner"), Gap4)(
					ImageFrame()(
						PosterImg(PosterFill, attr.Src(sed.PosterPath())),
					),
					html.If(isUserAdmin(), func() html.Node {
						return FlexCol(Class("v-series-sidebar-section"))(
							Link(
								sr.EditorPath(),
								turbo.DataFrame("_top"),
							)(Text("View in Editor", Size3,
								attr.Style("display: inline-block"),
							)),
						)
					}),
					html.If(haveSpecials(sed), func() html.Node {
						return FlexRow(Gap2)(
							Button(
								stimulus.Action("click->series#setRegular"),
								stimulus.Target("series", "regular"),
								attr.Attr("data-selected"),
							)(Text("Regular")),
							Button(
								stimulus.Action("click->series#setSpecial"),
								stimulus.Target("series", "special"),
							)(Text("Specials")),
							Button(
								stimulus.Action("click->series#setAll"),
								stimulus.Target("series", "all"),
							)(Text("All")),
						)
					}),
					FlexCol(Class("v-series-sidebar-section"))(
						html.RangeSeq(seasons, func(sn *model.Season) html.Node {
							return Box()(
								Link("#" + sn.Slug())(
									Text(sn.Title()),
								),
							)
						}),
					),
				),
			),
			FlexCol(Class("v-series-content"), Gap8)(
				FlexCol(Gap4)(
					Text(sr.Title(), Size7),
					html.If(len(editions) > 1, func() html.Node {
						return browseSeriesEditionSelect(editions, sed)
					}),
					TextNode(Size3)(html.Safe(sed.Summary())),
				),
				FlexCol(attr.Style("gap:5rem"))(
					html.RangeSeq(seasons, browseSeriesSeason),
				),
			),
		),
	)
}

func browseSeriesEditionSelect(editions []*model.SeriesWork, current *model.SeriesEdition) html.Node {
	return FlexRow(Gap2, attr.Style("flex-wrap:wrap"))(
		html.Range(editions, func(ed *model.SeriesWork) html.Node {
			selected := attr.Group()
			if ed.SeriesEditionHead.ID() == current.ID() {
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

func browseSeriesSeason(sn *model.Season) html.Node {
	return Box(attr.ID(sn.Slug()), Class("v-series-season"))(
		Box(Class("v-series-season-header"))(Text(sn.Title(), TextBold)),
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
	typ := attr.Attr("data-type")(ep.CoarseType())
	return Grid8(Class("v-series-episode"), typ, spoiler)(
		FlexCol(Class("v-series-episode-info"))(
			FlexRow(Class("v-series-episode-header"))(
				Box()(
					expr.IfElse(playable >= 0,
						func() html.Node {
							return Button(
								attr.Href(ep.PlayerPath(vids[playable])),
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
				Link(ep.TheaterPath())(
					FlexCol()(
						Box(Class("v-series-episode-number"))(Text(ep.SnnEnn(), TextNormal)),
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
			html.A(attr.Href(ep.TheaterPath()))(
				PosterImg(PosterFill, PosterAspect169, Class("v-series-episode-thumb"), attr.Src(ep.ThumbnailURL())),
			),
			Box(Class("v-series-spoiler-overlay")),
			Box(Class("v-series-episode-progress"))(
				Progress(0.1, ProgressSM),
			),
		),
	)
}

func haveSpecials(sed *model.SeriesEdition) bool {
	for range sed.Episodes(model.AnySpecial) {
		return true
	}
	return false
}
