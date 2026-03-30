package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

// Theme renders a div with display:contents that sets CSS
// custom properties (accent color) inherited by all children.
// Use the exported class attrs to configure it.
func Theme(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class("u-theme"),
		group(attrs...),
	)
}

// Accent colors
var AccentHotPink = attr.Attr("data-accent")("hotpink")

// Roles
var Destructive = attr.Attr("data-role")("destructive")

// Utility attrs for text.
// Set on text component or any ancestor.

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
	TextLight  = attr.Style("font-weight:300")
	TextNormal = attr.Style("font-weight:400")
	TextMedium = attr.Style("font-weight:500")
	TextBold   = attr.Style("font-weight:700")
)

var (
	TextWrap    = attr.Style("text-wrap:pretty")
	TextNowrap  = attr.Style("text-wrap:nowrap")
	TextBalance = attr.Style("text-wrap:balance")
)

var (
	TextAlignLeft   = attr.Style("text-align:left")
	TextAlignCenter = attr.Style("text-align:center")
	TextAlignRight  = attr.Style("text-align:right")
)

var (
	TextSelectAuto = attr.Attr("data-select")("auto")
	TextSelectNone = attr.Attr("data-select")("none")
)
