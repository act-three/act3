package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func MediaSeries(sr *model.Series) html.Node {
	sed := sr.EditionByTitle(model.AirDate)
	return media(sr.Title())(
		Box(Class("fixed inset-0 -z-1 blur-3xl saturate-180 opacity-20 scale-110"))(
			html.Img(
				Class("w-full aspect-2/3 object-cover "),
				attr.Src(sr.TVmazeImageURL()),
			),
		),
		Grid12(Class("pt-6"))(
			Box(Class("col-span-3"))(
				Box(Class("sticky top-20"))(
					Box()(
						html.Img(
							Class("w-full aspect-2/3 object-cover"),
							attr.Src(sr.TVmazeImageURL()),
						),
					),
					Box(Class("p-4 font-bold"))(Text(sr.Title())),
					Box(Class("p-4"))(
						Text("Show: regular & specials"),
					),
					html.RangeSeq(sed.Seasons(), func(sn *model.Season) html.Node {
						return Box(
							Class(""),
						)(
							Text(sn.Name()),
						)
					}),
				),
			),
			FlexCol(Class("col-span-8 col-start-5 gap-20"))(
				html.RangeSeq(sed.Seasons(), mediaSeriesSeason),
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
		`))(Text(sn.Name())).With(FontBold),
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
	return Grid8(Class("text-gray-11"))(
		FlexCol(Class("col-span-4 gap-2"))(
			FlexRow(Class("items-center gap-4"))(
				Box()(
					Button()(Icon("play")).
						With(ButtonBordered).
						With(ButtonCircle),
				),
				Link(ep.DetailURL(), Class("text-gray-11"))(
					FlexCol()(
						Box(
							Class(`
								text-gray-11/60
								decoration-gray-11/60!
							`),
						)(Text(ep.SnnEnn(), Class("font-normal"))),
						FlexRow()(
							Box()(Text(ep.Title(),
								Class("font-semibold"),
							)),
						),
					),
				),
			),
			Box(Class("relative"))(
				Text(ep.Summary(), Class("text-md")).With(LineClamp4),
				Box(
					Class(`
					absolute
					inset-0
					pointer-events-none
				`),
					hideSpoilersText,
				),
			),
		),
		Box(Class(`
			col-span-4
			aspect-16/9
			bg-gray-6
			relative
			hover:after:content-[""]
			hover:after:absolute
			hover:after:inset-0
			hover:after:bg-black/40
			hover:after:pointer-events-none
		`))(
			html.A(attr.Href(ep.DetailURL()))(
				html.Img(
					Class("w-full h-full object-cover"),
					attr.Src(ep.ImageURL()),
				),
			),
			Box(
				Class(`
					absolute
					inset-0
					pointer-events-none
				`),
				hideSpoilersImage,
			),
		),
	)
}
