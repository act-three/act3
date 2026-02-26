package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func ToolbarPrimary(attrs ...attr.Node) html.Element {
	return html.Header(
		attr.Class("a$toolbar-primary"),
		attr.Group(attrs...),
	).
		With(ButtonSurface).(html.Element)
}

func ToolbarSecondary(attrs ...attr.Node) html.Element {
	return html.Header(
		attr.Class("a$toolbar-secondary"),
		attr.Group(attrs...),
	).
		With(ButtonSize1).
		With(ButtonSurface).(html.Element)
}
