package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Link(url string, attrs ...attr.Node) html.Element {
	return html.A(
		Href(url),
		Class("u-link"),
		group(attrs...),
	)
}

var (
	LinkUnderlineAuto   = Attr("data-underline")("auto")
	LinkUnderlineAlways = Attr("data-underline")("always")
	LinkUnderlineHover  = Attr("data-underline")("hover")
)
