package ui

import (
	"fmt"

	"ily.dev/domi"
	"ily.dev/domi/html"
)

func Textf(format string, arg ...any) domi.Node {
	return Text(fmt.Sprintf(format, arg...))
}

func Text(s string, attrs ...domi.Attr) domi.Node {
	return TextNode(attrs...)(domi.Text(s))
}

func TextNode(attrs ...domi.Attr) domi.Element {
	return func(nodes ...domi.Node) domi.Node {
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
