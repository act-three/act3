package view

import (
	"slices"

	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	"ily.dev/act3/expr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/stimulus"
)

func BrowseSeriesEdition(sed *model.SeriesEdition, editions []*model.SeriesWork, uploads []model.Upload) (title string, n domi.Node) {
	seasons := slices.Values(([]*model.Season)(nil)) // empty
	if sed != nil {
		seasons = sed.Seasons()
	}
	sr := sed.SeriesHead()
	return sr.Title(), browse(uploads, sed.Poster())(
		Grid12(
			Class("v-series"),
			stimulus.Controller("series"),
			stimulus.Value("series", "mode")(""),
		)(
			FlexCol(Class("v-series-sidebar"))(
				FlexCol(Class("v-series-sidebar-inner"), Gap4)(
					ImageFrame()(
						PosterImg(AspectPoster, PosterFill, imgAttrs(sed.Poster())),
					),
					iff(isUserAdmin(), func() domi.Node {
						return FlexCol(Class("v-series-sidebar-section"))(
							Link(
								sr.EditorPath(),
							)(Text("View in Editor", Size3,
								Style("display: inline-block"),
							)),
						)
					}),

					iff(haveSpecials(sed), func() domi.Node {
						return FlexRow(Gap2)(
							Button(
								stimulus.Action("click->series#setRegular"),
								stimulus.Target("series", "regular"),
							)(Text("Regular")),
							Button(
								stimulus.Action("click->series#setSpecial"),
								stimulus.Target("series", "special"),
							)(Text("Specials")),
						)
					}),

					FlexCol(Class("v-series-sidebar-section"))(
						rangeSeq(seasons, func(sn *model.Season) domi.Node {
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
					iff(len(editions) > 1, func() domi.Node {
						return browseSeriesEditionSelect(editions, sed)
					}),

					TextNode(Size3, LineClamp5)(domi.Safe(sed.Summary())),
				),
				FlexCol(Style("gap:5rem"))(
					rangeSeq(seasons, browseSeriesSeason),
				),
			),
		),
	)
}

func browseSeriesEditionSelect(editions []*model.SeriesWork, current *model.SeriesEdition) domi.Node {
	return FlexRow(Gap2, Style("flex-wrap:wrap"))(
		rangeNodes(editions, func(ed *model.SeriesWork) domi.Node {
			selected := group()
			if ed.SeriesEditionHead.ID() == current.ID() {
				selected = ButtonSelected
			}
			return ButtonLink(ed.TheaterPath(),
				ButtonSurface, ButtonSize3,
				selected,
			)(Text(ed.Label()))
		}),
	)
}

func browseSeriesSeason(sn *model.Season) domi.Node {
	return Box(attr.ID(sn.Slug()), Class("v-series-season"))(
		Box(Class("v-series-season-header"))(Text(sn.Title(), TextBold)),
		FlexCol(Class("v-series-episodes"))(
			rangeSeq(sn.Episodes(model.AnyEpisode), browseSeriesEpisode),
		),
	)
}

func browseSeriesEpisode(ep *model.Episode) domi.Node {
	spoiler := group()
	if hideSpoilers(ep) {
		spoiler = Attr("data-spoiler")("")
	}
	active := ep.ActiveVideo()
	typ := Attr("data-type")(ep.CoarseType())
	return Grid8(Class("v-series-episode"), typ, spoiler)(
		FlexCol(Class("v-series-episode-info"))(
			FlexRow(Class("v-series-episode-header"))(
				Box()(
					expr.IfElse(active != nil,
						func() domi.Node {
							return playForm(ep.PlayIDs(),
								Button(
									attr.Type("submit"),
									ButtonSurface,
									ButtonCircle,
								)(Icon("solid/play")),
							)
						},
						func() domi.Node {
							return Button(Disabled(true), ButtonSurface, ButtonCircle)(Icon("line/x"))
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
				TextNode(Size2, LineClamp4)(domi.Safe(ep.Summary())),
				Box(Class("v-series-spoiler-overlay")),
			),
		),
		Box(HoverOverlay, Class("v-series-episode-thumb"))(
			html.A(Href(ep.TheaterPath()))(
				PosterImg(AspectThumbnail, PosterFill, Class("v-series-episode-thumb"), imgAttrs(ep.Thumbnail())),
			),
			Box(Class("v-series-spoiler-overlay")),
		),
	)
}

// hideSpoilers reports whether an episode's thumbnail and summary
// should be blurred as a spoiler. The demo shows the first three
// episodes of every series in the clear and blurs the rest.
func hideSpoilers(ep *model.Episode) bool {
	switch ep.SnnEnn() {
	case "S1E1", "S1E2", "S1E3":
		return false
	default:
		return true
	}
}

func haveSpecials(sed *model.SeriesEdition) bool {
	for range sed.Episodes(model.AnySpecial) {
		return true
	}
	return false
}
