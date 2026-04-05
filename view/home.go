package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/stimulus"
)

func Home(
	works []model.Work,
	cols []*model.CollectionHead,
) html.Node {
	var washURLs []string
	for _, w := range works {
		washURLs = append(washURLs, w.PosterPath())
	}
	return browse("Act Three", washURLs...)(
		Box(
			Class("v-home"),
			stimulus.Controller("home"),
			stimulus.Value("home", "mode")(""),
		)(
			FlexRow(Gap4, Class("v-home-toolbar"))(
				FlexRow(Gap1)(
					Button(ButtonSurface)(Text("Title")),
					Button(ButtonSurface, ButtonCircle)(Icon("line/switch-vertical-01")),
				),
				FlexRow(Gap1)(
					Button(
						ButtonSurface,
						stimulus.Action("click->home#setMovie"),
						stimulus.Target("home", "movie"),
					)(Text("Movies")),
					Button(
						ButtonSurface,
						stimulus.Action("click->home#setSeries"),
						stimulus.Target("home", "series"),
					)(Text("Series")),
				),
				Button(ButtonSurface, ButtonCircle)(Icon("line/filter-lines")),
				Box(Class("v-home-search"))(
					Icon("line/search-sm"),
					html.Input(
						Class("v-home-search-input"),
						stimulus.Action("input->home#search"),
					),
				),
			),
			FlexCol(Class("v-home-collections"))(
				html.Range(cols, func(c *model.CollectionHead) html.Node {
					return collectionBannerLink(c, attr.Attr("data-search-hidden"))
				}),
			),
			posterGrid(works),
		),
	)
}
