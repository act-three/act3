package view

import (
	"ily.dev/domi"
	"ily.dev/domi/html"

	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func posterGrid(works []model.Work) domi.Node {
	return Box(Class("v-poster-grid"))(
		rangeNodes(works, posterGridLink),
	)
}

func posterGridLink(w model.Work) domi.Node {
	im, _ := w.PosterField()
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
			PosterImg(AspectPoster, PosterFill, imgAttrs(im)),
		),
	)
}
