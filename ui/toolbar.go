package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/html"
)

func ToolbarPrimary(attrs ...domi.Attr) domi.Element {
	return html.Header(
		Class("u-toolbar-primary"),
		group(attrs...),
	)
}
