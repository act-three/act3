package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

// ImageFrame wraps content (typically an image) with
// rounded corners and overflow hidden.
func ImageFrame(attrs ...attr.Node) html.Element {
	return html.Div(
		Class("u-image-frame"),
		group(attrs...),
	)
}

// HoverOverlay adds a semi-transparent darkening overlay
// on hover via a ::after pseudo-element.
// The parent element must have position: relative.
var HoverOverlay = Class("u-hover-overlay")
