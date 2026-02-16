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
		Box(Class("mx-auto w-4xl relative"))(
			Box(Class("sticky w-lg"))(
				Box()(
					html.Img(attr.Src(sr.TVmazeImageURL())),
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
			Box(Class("absolute top-0 left-60"))(
				html.RangeSeq(sed.Seasons(), func(sn *model.Season) html.Node {
					return Box(
						Class("py-8"),
					)(
						Box(Class(`
							py-4
							text-gray-11
							text-xl
							border-t-[2px]
							border-gray-11
						`))(Text(sn.Name())).With(FontBold),
						FlexCol(Class("gap-2 py-2"))(
							html.RangeSeq(sn.Episodes(model.AnyEpisode), mediaSeriesEpisode),
						),
					)
				}),
			),
		),
	)
}

func mediaSeriesEpisode(ep *model.Episode) html.Node {
	return FlexRow(Class("text-gray-11 gap-2"))(
		FlexCol(Class("flex-none w-96 h-45 gap-2"))(
			FlexRow(Class("items-center gap-2"))(
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
						)(Text(ep.SnnEnn())),
						FlexRow()(
							Box()(Text(ep.Title())),
						),
					),
				),
			),
			Box(Class("relative"))(
				Text(ep.Summary(), Class("text-sm")).With(LineClamp4),
				Box(
					Class(`
					absolute
					inset-0
					backdrop-blur-xs
					pointer-events-none
				`),
				),
			),
		),
		Box(Class(`flex-none w-80 h-45 bg-gray-6
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
					backdrop-blur-md
					pointer-events-none
				`),
			),
		),
	)
}
