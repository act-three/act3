package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func ScrollArea(attrs ...attr.Node) html.Element {
	return html.Div(
		attr.Class(`
			flex-1
			w-full
			h-full
			overflow-auto
			overscroll-contain
		`),
		attr.Group(attrs...),
	)
}
