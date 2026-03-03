package view

import (
	"math/rand/v2"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func Home(a []*model.SeriesHead) html.Node {
	sr := a[rand.IntN(len(a))]
	return media("Act Three")(
		Box(Class("fixed inset-0 -z-1 blur-3xl saturate-180 opacity-20 scale-110"))(
			html.Img(
				Class("w-full aspect-2/3 object-cover"),
				attr.Src(sr.TVmazeImageURL()),
			),
		),
		Grid12(Class("pt-4"))(
			FlexRow(ColSpan12, Gap4)(
				ButtonGroup(ButtonGroupRadiusLarge)(
					Button(ButtonSurface)(Text("Title")),
					Button(ButtonSurface)(Icon("arrow-down-a-z")),
				),
				ButtonGroup(ButtonGroupRadiusLarge)(
					Button(ButtonSurface)(Text("Movies")),
					Button(ButtonSurface)(Text("Series")),
				),
				Button(ButtonSurface, ButtonRadiusLarge)(Icon("list-filter")),
				InputText()(),
			),
			FlexRow(ColSpan12, Class(`
					w-full
					flex-wrap
					justify-center
					content-start
					gap-[1px]
				`))(
				html.Range(a, seriesPosterLink),
			),
		),
		//html.Div()(html.A(attr.Href("/movies"))(html.Text("All Movies"))),
		//html.Div()(html.A(attr.Href("/series"))(html.Text("All Series"))),

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
