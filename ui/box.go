package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/html"
)

func Box(attrs ...domi.Attr) domi.Element {
	return html.Div(
		group(attrs...),
	)
}
