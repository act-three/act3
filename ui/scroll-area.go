package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

// ScrollArea is a simple overflow container using native
// scrollbars. Unlike the Radix ScrollArea component, which
// provides custom styled scrollbar tracks and thumbs, this
// relies on the browser's default scrollbar rendering.
func ScrollArea(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class("a$scroll-area"),
		attr.Group(attrs...),
	)
}
