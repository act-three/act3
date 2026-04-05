package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
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
		attr.Attr("data-kind")(w.Kind()),
	)(
		html.A(
			Class("v-poster-grid-link"),
			attr.Href(w.TheaterPath()),
		)(
			PosterImg(PosterFill, attr.Src(w.PosterPath())),
		),
	)
}
