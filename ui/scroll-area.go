package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func ScrollArea(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class("a$scroll-area"),
		attr.Group(attrs...),
	)
}
