package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Label(icon, text string, attrs ...attr.Node) html.Node {
	return LabelNode(icon, Text(text), attrs...)
}

// LabelNode is Label with an arbitrary body node in place of plain
// text, for callers that need to control how the text renders — e.g.
// wrapping it in TruncTail.
func LabelNode(icon string, body html.Node, attrs ...attr.Node) html.Node {
	return html.Div(
		Class("u-label"),
		group(attrs...),
	)(
		Icon(icon),
		body,
	)
}
