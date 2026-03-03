package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func MediaListSeries(a []*model.SeriesHead) html.Node {
	return media("Act Three")(
		Box(Class(`
				w-full
				flex
				flex-row
				flex-wrap
				justify-center
				content-start
				gap-[1px]
			`))(
			html.Range(a, seriesPosterLink),
		),
	)
}

func seriesPosterLink(sr *model.SeriesHead) html.Node {
	return html.Div(Class(`
		aspect-2/3
		w-[187px]
		relative
		hover:after:content-[""]
		hover:after:absolute
		hover:after:inset-0
		hover:after:bg-black/40
		hover:after:pointer-events-none
		`))(
		html.A(
			Class("block w-full h-full"),
			attr.Href(sr.PlayURL()),
		)(
			html.Img(
				Class("w-full h-full object-cover"),
				attr.Src(sr.TVmazeImageURL()),
			),
		),
	)
}
