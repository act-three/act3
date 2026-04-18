package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
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

func Inset(attrs ...attr.Node) html.Element {
	return html.Div(
		Class("u-inset"),
		group(attrs...),
	)
}
