package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Code(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class("u-code"),
		group(attrs...),
	)
}

var (
	CodeSize1 = attr.Attr("data-size")("1")
	CodeSize2 = attr.Attr("data-size")("2")
	CodeSize3 = attr.Attr("data-size")("3")
	CodeSize4 = attr.Attr("data-size")("4")
)

var (
	CodeWrap   = attr.Attr("data-wrap")("wrap")
	CodeNowrap = attr.Attr("data-wrap")("nowrap")
)
