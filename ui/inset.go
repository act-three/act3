package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

var (
	InsetSideAll    = Class("a$inset+all")
	InsetSideX      = Class("a$inset+x")
	InsetSideY      = Class("a$inset+y")
	InsetSideTop    = Class("a$inset+top")
	InsetSideBottom = Class("a$inset+bottom")
	InsetSideLeft   = Class("a$inset+left")
	InsetSideRight  = Class("a$inset+right")
)

func Inset(attrs ...attr.Node) html.Element {
	return html.Div(
		Class("a$inset"),
		group(attrs...),
	)
}
