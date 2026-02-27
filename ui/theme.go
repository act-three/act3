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
var AccentCrimson = Class("u-theme+accent-color+crimson")

// Scale presets
var (
	Scale1 = Class("u-theme+scale+1")
	Scale2 = Class("u-theme+scale+2")
	Scale3 = Class("u-theme+scale+3")
	Scale4 = Class("u-theme+scale+4")
	Scale5 = Class("u-theme+scale+5")
)

// Radius presets
var (
	RadiusNone   = Class("u-theme+radius+none")
	RadiusSmall  = Class("u-theme+radius+small")
	RadiusMedium = Class("u-theme+radius+medium")
	RadiusLarge  = Class("u-theme+radius+large")
	RadiusFull   = Class("u-theme+radius+full")
)
