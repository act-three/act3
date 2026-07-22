package view

import (
	"ily.dev/domi"
	"ily.dev/domi/html"

	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/stimulus"
)

func Home(
	works []model.Work,
	cols []*model.CollectionHead,
	uploads []model.Upload,
) (title string, n domi.Node) {
	var washImages []model.Image
	for _, w := range works {
		im, _ := w.PosterField()
		washImages = append(washImages, im)
	}
	return "", browse(uploads, washImages...)(
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
					// Opaque: the search box is a client-owned
					// filter (see home.js); domi must neither
					// commit nor revert what the user types.
					domi.WithKeyOpaque("search", html.Input(
						Class("v-home-search-input"),
						stimulus.Action("input->home#search"),
					)),
				),
			),
			FlexCol(Class("v-home-collections"))(
				rangeNodes(cols, func(c *model.CollectionHead) domi.Node {
					return collectionBannerLink(c, Attr("data-search-hidden")(""))
				}),
			),
			posterGrid(works),
			Box(Size4, Class("v-home-no-results"), stimulus.Target("home", "noResults"))(
				Text("No Results"),
			),
		),
	)
}
