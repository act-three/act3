package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Code(attrs ...attr.Node) html.Element {
	return html.Div(
		Class("u-code"),
		group(attrs...),
	)
}

var (
	CodeSize1 = Attr("data-code-size")("1")
	CodeSize2 = Attr("data-code-size")("2")
	CodeSize3 = Attr("data-code-size")("3")
	CodeSize4 = Attr("data-code-size")("4")
)

var (
	CodeWrap   = Attr("data-code-wrap")("wrap")
	CodeNowrap = Attr("data-code-wrap")("nowrap")
)
