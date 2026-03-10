package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

// Center renders a full-size container that centers its
// children both horizontally and vertically.
func Center(attrs ...attr.Node) html.Element {
	return html.Div(
		Class("u-center"),
		group(attrs...),
	)
}
