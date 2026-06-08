package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/html"
)

func Label(icon, text string, attrs ...domi.Attr) domi.Node {
	return LabelNode(icon, Text(text), attrs...)
}

// LabelNode is Label with an arbitrary body node in place of plain
// text, for callers that need to control how the text renders — e.g.
// wrapping it in TruncTail.
func LabelNode(icon string, body domi.Node, attrs ...domi.Attr) domi.Node {
	return html.Div(
		Class("u-label"),
		group(attrs...),
	)(
		Icon(icon),
		body,
	)
}
