package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func FlexRow(attrs ...attr.Node) html.Element {
	return html.Div(
		Class("a$flex-row"),
		group(attrs...),
	)
}

func FlexCol(attrs ...attr.Node) html.Element {
	return html.Div(
		Class("a$flex-col"),
		group(attrs...),
	)
}
