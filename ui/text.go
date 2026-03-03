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
	FontLight  = attr.Class("u-text+weight-light")
	FontNormal = attr.Class("u-text+weight-normal")
	FontMedium = attr.Class("u-text+weight-medium")
	FontBold   = attr.Class("u-text+weight-bold")
)

var (
	TextAlignLeft   = attr.Class("u-text+align-left")
	TextAlignCenter = attr.Class("u-text+align-center")
	TextAlignRight  = attr.Class("u-text+align-right")
)

var (
	TextWrap    = attr.Class("u-text+wrap-wrap")
	TextNowrap  = attr.Class("u-text+wrap-nowrap")
	TextPretty  = attr.Class("u-text+wrap-pretty")
	TextBalance = attr.Class("u-text+wrap-balance")
)

var (
	LineClamp1 = attr.Class("u-text+clamp-1")
	LineClamp2 = attr.Class("u-text+clamp-2")
	LineClamp3 = attr.Class("u-text+clamp-3")
	LineClamp4 = attr.Class("u-text+clamp-4")
	LineClamp5 = attr.Class("u-text+clamp-5")
)

var (
	TextSelectAuto = attr.Class("u-text+select-auto")
	TextSelectNone = attr.Class("u-text+select-none")
)

var (
	TextSize1 = attr.Class("u-text+size-1")
	TextSize2 = attr.Class("u-text+size-2")
	TextSize3 = attr.Class("u-text+size-3")
	TextSize4 = attr.Class("u-text+size-4")
	TextSize5 = attr.Class("u-text+size-5")
	TextSize6 = attr.Class("u-text+size-6")
	TextSize7 = attr.Class("u-text+size-7")
	TextSize8 = attr.Class("u-text+size-8")
	TextSize9 = attr.Class("u-text+size-9")
)

var (
	TextTrimStart = attr.Class("u-text+trim-start")
	TextTrimEnd   = attr.Class("u-text+trim-end")
	TextTrimBoth  = attr.Class("u-text+trim-both")
)

var TextTruncate = attr.Class("u-text+truncate")
