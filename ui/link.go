package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/html"
)

func Link(url string, attrs ...domi.Attr) domi.Element {
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
