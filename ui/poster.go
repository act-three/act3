package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

var (
	PosterFill          = attr.Attr("data-fill")
	PosterAspect23      = attr.Attr("data-aspect")("2-3")
	PosterAspect169     = attr.Attr("data-aspect")("16-9")
	PosterAspect1000185 = attr.Attr("data-aspect")("1000-185")
)

// PosterImg renders an <img> styled for poster display.
// Defaults to 2:3 aspect ratio with object-cover.
func PosterImg(attrs ...attr.Node) html.Element {
	return html.Img(
		Class("u-poster"),
		group(attrs...),
	)
}
