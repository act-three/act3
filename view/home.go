package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func Home(works []*model.Work) html.Node {
	var washURLs []string
	for _, w := range works {
		if w.ImageURL() != "" {
			washURLs = append(washURLs, w.ImageURL())
		}
	}
	return media("Act Three", washURLs...)(
		FlexRow(Gap4, Class("py-4"))(
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
		FlexRow(Class(`
					w-full
					flex-wrap
					justify-center
					content-start
					gap-[1px]
				`))(
			html.Range(works, workPosterLink),
		),
	)
}

func workPosterLink(w *model.Work) html.Node {
	return Box(HoverOverlay, Class("aspect-2/3 w-[187px]"))(
		html.A(
			Class("block w-full h-full"),
			attr.Href(w.PlayURL()),
		)(
			PosterImg(PosterFill, Class("h-full"), attr.Src(w.ImageURL())),
		),
	)
}
