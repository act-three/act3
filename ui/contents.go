package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Contents(attrs ...attr.Node) html.Element {
	return html.Div(
		Class("contents"),
		group(attrs...),
	)
}
