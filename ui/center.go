package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/html"
)

// Center renders a full-size container that centers its
// children both horizontally and vertically.
func Center(attrs ...domi.Attr) domi.Element {
	return html.Div(
		Class("u-center"),
		group(attrs...),
	)
}
