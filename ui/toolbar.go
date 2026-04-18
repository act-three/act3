package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func ToolbarPrimary(attrs ...attr.Node) html.Element {
	return html.Header(
		Class("u-toolbar-primary"),
		group(attrs...),
	)
}

func ToolbarSecondary(attrs ...attr.Node) html.Element {
	return html.Header(
		Class("u-toolbar-secondary"),
		group(attrs...),
	)
}
