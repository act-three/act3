package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/html"
)

var PosterFill = Attr("data-fill")("")

// PosterImg renders an <img> styled for poster/banner/thumbnail
// display: object-fit cover, with the given aspect ratio.
func PosterImg(a Aspect, attrs ...domi.Attr) domi.Node {
	return html.Img(
		Class("u-poster"),
		Stylef("aspect-ratio: %s", a),
		group(attrs...),
	)
}
