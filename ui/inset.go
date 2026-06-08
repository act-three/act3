package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/html"
)

var (
	InsetSideAll    = Attr("data-side")("all")
	InsetSideX      = Attr("data-side")("x")
	InsetSideY      = Attr("data-side")("y")
	InsetSideTop    = Attr("data-side")("top")
	InsetSideBottom = Attr("data-side")("bottom")
	InsetSideLeft   = Attr("data-side")("left")
	InsetSideRight  = Attr("data-side")("right")
)

func Inset(attrs ...domi.Attr) domi.Element {
	return html.Div(
		Class("u-inset"),
		group(attrs...),
	)
}
