package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func posterGrid(works []model.Work) html.Node {
	return Box(Class("v-poster-grid"))(
		html.Range(works, posterGridLink),
	)
}

func posterGridLink(w model.Work) html.Node {
	return Box(
		HoverOverlay,
		Class("v-poster-grid-poster"),
		Attr("data-kind")(w.Kind()),
		Attr("data-title")(w.Title()),
	)(
		html.A(
			Class("v-poster-grid-link"),
			Href(w.TheaterPath()),
		)(
			PosterImg(PosterFill, imgAttrs(w.PosterField())),
		),
	)
}
