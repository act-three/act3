package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func ToolbarPrimary(attrs ...attr.Node) html.Element {
	return html.Header(
		attr.Class("u-toolbar-primary"),
		attr.Group(attrs...),
	).
		With(ButtonSurface).(html.Element)
}

func ToolbarSecondary(attrs ...attr.Node) html.Element {
	return html.Header(
		attr.Class("u-toolbar-secondary"),
		attr.Group(attrs...),
	).
		With(ButtonSurface).(html.Element)
}
