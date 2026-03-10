package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

var (
	PosterFill      = Class("u-poster+fill")
	PosterAspect23  = Class("u-poster+2-3")
	PosterAspect169 = Class("u-poster+16-9")
)

// PosterImg renders an <img> styled for poster display.
// Defaults to 2:3 aspect ratio with object-cover.
func PosterImg(attrs ...attr.Node) html.Element {
	return html.Img(
		Class("u-poster"),
		group(attrs...),
	)
}
