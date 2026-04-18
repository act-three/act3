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
			Class("u-text"),
			group(attrs...),
		)(
			nodes...,
		)
	}
}

var (
	LineClamp1 = Attr("data-clamp")("1")
	LineClamp2 = Attr("data-clamp")("2")
	LineClamp3 = Attr("data-clamp")("3")
	LineClamp4 = Attr("data-clamp")("4")
	LineClamp5 = Attr("data-clamp")("5")
)

var (
	TextTrimStart = Attr("data-trim")("start")
	TextTrimEnd   = Attr("data-trim")("end")
	TextTrimBoth  = Attr("data-trim")("both")
)
