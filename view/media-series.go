package view

import (
	"slices"

	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func MediaSeries(sr *model.Series) html.Node {
	seasons := slices.Values(([]*model.Season)(nil)) // empty
	if sed := sr.EditionByTitle(model.AirDate); sed != nil {
		seasons = sed.Seasons()
	}
	return media(sr.Title(), sr.TVmazeImageURL())(
		Grid12(Class("pt-10"))(
			Box(Class("col-span-3"))(
				Box(Class("sticky top-20"))(
					ImageFrame()(
						PosterImg(PosterFill, attr.Src(sr.TVmazeImageURL())),
					),
					Box(Class("p-4"))(Text(sr.Title(), FontBold)),
					Box(Class("p-4"))(
						Text("Show: regular & specials"),
					),
					html.RangeSeq(seasons, func(sn *model.Season) html.Node {
						return Box(
							Class(""),
						)(
							Text(sn.Name()),
						)
					}),
				),
			),
			FlexCol(Class("col-span-8 col-start-5 gap-20"))(
				html.RangeSeq(seasons, mediaSeriesSeason),
			),
		),
	)
}

func mediaSeriesSeason(sn *model.Season) html.Node {
	return Box()(
		Box(Class(`
			py-4
			pb-8
			text-gray-11
			text-xl
			border-t-[2px]
			border-gray-11
		`))(Text(sn.Name(), FontBold)),
		FlexCol(Class("gap-12 py-2"))(
			html.RangeSeq(sn.Episodes(model.AnyEpisode), mediaSeriesEpisode),
		),
	)
}

func mediaSeriesEpisode(ep *model.Episode) html.Node {
	const doHideSpoilers = false
	hideSpoilersText := group()
	hideSpoilersImage := group()
	if doHideSpoilers {
		hideSpoilersText = Class("backdrop-blur-xs")
		hideSpoilersImage = Class("backdrop-blur-md")
	}
	vids := ep.Videos()
	playable := slices.IndexFunc(vids, func(v *model.Video) bool {
		return v.MVPlaylist() != ""
	})
	return Grid8(Class("text-gray-11"))(
		FlexCol(Class("col-span-5 gap-2"))(
			FlexRow(Class("items-center gap-4"))(
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
				Link(ep.DetailURL(), Class("text-gray-11"))(
					FlexCol()(
						Box(
							Class(`
								text-gray-11/60
								decoration-gray-11/60!
							`),
						)(Text(ep.SnnEnn(), FontNormal)),
						FlexRow()(
							Box()(Text(ep.Title(),
								Class("font-semibold"),
							)),
						),
					),
				),
			),
			Box(Class("relative"))(
				TextNode(Class("text-sm"), LineClamp4)(html.Safe(ep.Summary())),
				Box(
					Class(`
					absolute
					-inset-2
					pointer-events-none
				`),
					hideSpoilersText,
				),
			),
		),
		Box(HoverOverlay, Class(`col-span-3
			aspect-16/9
			bg-gray-6
			rounded-xs
			overflow-hidden
		`))(
			html.A(attr.Href(ep.DetailURL()))(
				PosterImg(PosterFill, PosterAspect169, Class("h-full"), attr.Src(ep.ImageURL())),
			),
			Box(
				Class(`
					absolute
					inset-0
					pointer-events-none
				`),
				hideSpoilersImage,
			),
			Box(Class("absolute bottom-0 left-0 right-0"))(
				Progress(0.1, ProgressSM),
			),
		),
	)
}
