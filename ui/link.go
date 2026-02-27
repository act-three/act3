package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Link(url string, attrs ...attr.Node) html.Element {
	return html.A(
		attr.Href(url),
		attr.Class("u-link"),
		group(attrs...),
	)
}

var (
	LinkUnderlineAuto   = attr.Class("u-link+underline-auto")
	LinkUnderlineAlways = attr.Class("u-link+underline-always")
	LinkUnderlineHover  = attr.Class("u-link+underline-hover")
)
