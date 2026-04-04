package view

import (
	"ily.dev/act3/html"
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
		posterGrid(works),
	)
}
