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

var (
	ButtonGroupHorizontal = attr.Attr("data-button-group")("horizontal")
	ButtonGroupVertical   = attr.Attr("data-button-group")("vertical")
)
