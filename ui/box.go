package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Box(attrs ...attr.Node) html.Element {
	return html.Div(
		group(attrs...),
	)
}
