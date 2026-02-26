package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

var (
	InsetSideAll    = Class("u-inset+all")
	InsetSideX      = Class("u-inset+x")
	InsetSideY      = Class("u-inset+y")
	InsetSideTop    = Class("u-inset+top")
	InsetSideBottom = Class("u-inset+bottom")
	InsetSideLeft   = Class("u-inset+left")
	InsetSideRight  = Class("u-inset+right")
)

func Inset(attrs ...attr.Node) html.Element {
	return html.Div(
		Class("u-inset"),
		group(attrs...),
	)
}
