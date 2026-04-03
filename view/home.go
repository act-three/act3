package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func Home(works []model.Work) html.Node {
	var washURLs []string
	for _, w := range works {
		washURLs = append(washURLs, w.PosterPath())
	}
	return browse("Act Three", washURLs...)(
		FlexRow(Gap4, Class("v-home-toolbar"))(
			FlexRow(Gap1)(
				Button(ButtonSurface)(Text("Title")),
				Button(ButtonSurface)(Icon("line/switch-vertical-01")),
			),
			FlexRow(Gap1)(
				Button(ButtonSurface)(Text("Movies")),
				Button(ButtonSurface)(Text("Series")),
			),
			Button(ButtonSurface)(Icon("line/filter-lines")),
			InputText()(),
		),
		FlexRow(Class("v-home-grid"))(
			html.Range(works, workPosterLink),
		),
	)
}

func workPosterLink(w model.Work) html.Node {
	return Box(HoverOverlay, Class("v-home-poster"))(
		html.A(
			Class("v-home-poster-link"),
			attr.Href(w.TheaterPath()),
		)(
			PosterImg(PosterFill, attr.Src(w.PosterPath())),
		),
	)
}
