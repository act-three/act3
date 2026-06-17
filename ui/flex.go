package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/html"
)

func FlexRow(attrs ...domi.Attr) domi.Element {
	return html.Div(
		Class("u-flex-row"),
		group(attrs...),
	)
}

func FlexCol(attrs ...domi.Attr) domi.Element {
	return html.Div(
		Class("u-flex-col"),
		group(attrs...),
	)
}
