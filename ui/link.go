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
	LinkUnderlineAuto   = attr.Attr("data-underline")("auto")
	LinkUnderlineAlways = attr.Attr("data-underline")("always")
	LinkUnderlineHover  = attr.Attr("data-underline")("hover")
)
