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
	return html.Div(
		attr.Class("u-text"),
		group(attrs...),
	)(
		html.Text(s),
	)
}

var (
	FontNormal = attr.Class("u-text+weight-normal")
	FontMedium = attr.Class("u-text+weight-medium")
	FontBold   = attr.Class("u-text+weight-bold")
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

var TextTruncate = attr.Class("u-text+truncate")
