package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/html"
)

// Theme renders a div with display:contents that sets CSS
// custom properties (accent color) inherited by all children.
// Use the exported class attrs to configure it.
func Theme(attrs ...domi.Attr) domi.Element {
	return html.Div(
		Class("u-theme"),
		group(attrs...),
	)
}

// Accent colors
var AccentHotPink = Attr("data-accent")("hotpink")

// Roles
var Destructive = Attr("data-role")("destructive")

// Utility attrs for text.
// Set on text component or any ancestor.

var (
	Size1 = Attr("data-size")("1")
	Size2 = Attr("data-size")("2")
	Size3 = Attr("data-size")("3")
	Size4 = Attr("data-size")("4")
	Size5 = Attr("data-size")("5")
	Size6 = Attr("data-size")("6")
	Size7 = Attr("data-size")("7")
	Size8 = Attr("data-size")("8")
	Size9 = Attr("data-size")("9")
)

var (
	TextLight  = Style("font-weight:300")
	TextNormal = Style("font-weight:400")
	TextMedium = Style("font-weight:500")
	TextBold   = Style("font-weight:700")
)

var (
	TextWrap    = Style("text-wrap:pretty")
	TextNowrap  = Style("text-wrap:nowrap")
	TextBalance = Style("text-wrap:balance")
)

var (
	TextAlignLeft   = Style("text-align:left")
	TextAlignCenter = Style("text-align:center")
	TextAlignRight  = Style("text-align:right")
)

var (
	TextSelectAuto = Attr("data-select")("auto")
	TextSelectNone = Attr("data-select")("none")
)
