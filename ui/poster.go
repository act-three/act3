package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

var PosterFill = Attr("data-fill")

// PosterImg renders an <img> styled for poster/banner/thumbnail
// display: object-fit cover, with the given aspect ratio.
func PosterImg(a Aspect, attrs ...attr.Node) html.Element {
	return html.Img(
		Class("u-poster"),
		Stylef("aspect-ratio: %s", a),
		group(attrs...),
	)
}
