package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func Home(a []*model.SeriesHead) html.Node {
	return media("Act Three", homeWashURLs(a)...)(
		Grid12(Class("pt-4"))(
			FlexRow(ColSpan12, Gap4)(
				ButtonGroup(ButtonGroupRadiusLarge)(
					Button(ButtonSurface)(Text("Title")),
					Button(ButtonSurface)(Icon("line/switch-vertical-01")),
				),
				ButtonGroup(ButtonGroupRadiusLarge)(
					Button(ButtonSurface)(Text("Movies")),
					Button(ButtonSurface)(Text("Series")),
				),
				Button(ButtonSurface, ButtonRadiusLarge)(Icon("line/filter-lines")),
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

func homeWashURLs(a []*model.SeriesHead) (u []string) {
	for _, sr := range a {
		u = append(u, sr.TVmazeImageURL())
	}
	return u
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
