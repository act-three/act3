package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/stimulus"
)

func Home(
	works []model.Work,
	cols []*model.CollectionHead,
) html.Node {
	var washImages []model.Image
	for _, w := range works {
		im, _ := w.PosterField()
		washImages = append(washImages, im)
	}
	return browse("Act Three", washImages...)(
		Box(
			Class("v-home"),
			stimulus.Controller("home"),
			stimulus.Value("home", "mode")(""),
		)(
			FlexRow(Gap4, Class("v-home-toolbar"))(
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
					return collectionBannerLink(c, Attr("data-search-hidden"))
				}),
			),
			posterGrid(works),
			Box(Size4, Class("v-home-no-results"), stimulus.Target("home", "noResults"))(
				Text("No Results"),
			),
		),
	)
}
