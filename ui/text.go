package ui

import (
	"fmt"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Textf(format string, arg ...any) html.Node {
	return Text(fmt.Sprintf(format, arg...))
}

func Text(s string, attrs ...attr.Node) html.Node {
	return TextNode(attrs...)(html.Text(s))
}

func TextNode(attrs ...attr.Node) html.Element {
	return func(nodes ...html.Node) html.Node {
		return html.Div(
			attr.Class("u-text"),
			group(attrs...),
		)(
			nodes...,
		)
	}
}

var (
	LineClamp1 = attr.Attr("data-clamp")("1")
	LineClamp2 = attr.Attr("data-clamp")("2")
	LineClamp3 = attr.Attr("data-clamp")("3")
	LineClamp4 = attr.Attr("data-clamp")("4")
	LineClamp5 = attr.Attr("data-clamp")("5")
)

var (
	TextTrimStart = attr.Attr("data-trim")("start")
	TextTrimEnd   = attr.Attr("data-trim")("end")
	TextTrimBoth  = attr.Attr("data-trim")("both")
)
