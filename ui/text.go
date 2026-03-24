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
	FontLight  = attr.Attr("data-weight")("light")
	FontNormal = attr.Attr("data-weight")("normal")
	FontMedium = attr.Attr("data-weight")("medium")
	FontBold   = attr.Attr("data-weight")("bold")
)

var (
	TextAlignLeft   = attr.Attr("data-align")("left")
	TextAlignCenter = attr.Attr("data-align")("center")
	TextAlignRight  = attr.Attr("data-align")("right")
)

var (
	TextWrap    = attr.Attr("data-wrap")("wrap")
	TextNowrap  = attr.Attr("data-wrap")("nowrap")
	TextPretty  = attr.Attr("data-wrap")("pretty")
	TextBalance = attr.Attr("data-wrap")("balance")
)

var (
	LineClamp1 = attr.Attr("data-clamp")("1")
	LineClamp2 = attr.Attr("data-clamp")("2")
	LineClamp3 = attr.Attr("data-clamp")("3")
	LineClamp4 = attr.Attr("data-clamp")("4")
	LineClamp5 = attr.Attr("data-clamp")("5")
)

var (
	TextSelectAuto = attr.Attr("data-select")("auto")
	TextSelectNone = attr.Attr("data-select")("none")
)

var (
	Size1 = attr.Attr("data-size")("1")
	Size2 = attr.Attr("data-size")("2")
	Size3 = attr.Attr("data-size")("3")
	Size4 = attr.Attr("data-size")("4")
	Size5 = attr.Attr("data-size")("5")
	Size6 = attr.Attr("data-size")("6")
	Size7 = attr.Attr("data-size")("7")
	Size8 = attr.Attr("data-size")("8")
	Size9 = attr.Attr("data-size")("9")
)

var (
	TextTrimStart = attr.Attr("data-trim")("start")
	TextTrimEnd   = attr.Attr("data-trim")("end")
	TextTrimBoth  = attr.Attr("data-trim")("both")
)

var TextTruncate = attr.Attr("data-truncate")
