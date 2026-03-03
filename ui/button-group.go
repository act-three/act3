package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

// ButtonGroup renders a group of buttons with merged borders
// and rounded corners only on the outer edges.
func ButtonGroup(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Role("group"),
		attr.Class("u-button-group"),
		group(attrs...),
	)
}

var ButtonGroupVertical = attr.Class("u-button-group+vertical")

var (
	ButtonGroupRadiusNone   = attr.Class("u-button-group+radius-none")
	ButtonGroupRadiusMedium = attr.Class("u-button-group+radius-medium")
	ButtonGroupRadiusFull   = attr.Class("u-button-group+radius-full")
)
