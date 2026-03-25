package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

// Theme renders a div with display:contents that sets CSS
// custom properties (accent color, radius) inherited by all
// children. Use the exported class attrs to configure it.
func Theme(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class("u-theme"),
		group(attrs...),
	)
}

// Accent colors
var AccentCrimson = attr.Attr("data-accent")("crimson")

// Radius presets
var (
	RadiusNone   = attr.Attr("data-radius")("none")
	RadiusSmall  = attr.Attr("data-radius")("small")
	RadiusMedium = attr.Attr("data-radius")("medium")
	RadiusLarge  = attr.Attr("data-radius")("large")
	RadiusFull   = attr.Attr("data-radius")("full")
)
