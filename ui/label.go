package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Label(icon, text string, attrs ...attr.Node) html.Node {
	return html.Div(
		Class("u-label"),
		group(attrs...),
	)(
		Icon(icon),
		Text(text),
	)
}
