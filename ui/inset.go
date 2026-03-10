package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

var (
	InsetSideAll    = attr.Attr("data-side")("all")
	InsetSideX      = attr.Attr("data-side")("x")
	InsetSideY      = attr.Attr("data-side")("y")
	InsetSideTop    = attr.Attr("data-side")("top")
	InsetSideBottom = attr.Attr("data-side")("bottom")
	InsetSideLeft   = attr.Attr("data-side")("left")
	InsetSideRight  = attr.Attr("data-side")("right")
)

func Inset(attrs ...attr.Node) html.Element {
	return html.Div(
		Class("u-inset"),
		group(attrs...),
	)
}
