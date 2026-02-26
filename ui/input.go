package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func InputText(attrs ...attr.Node) html.Element {
	return html.Input(
		attr.Class("a$input"),
		attr.Group(attrs...),
		attr.Attr("data-slot")("input"),
	)
}

func InputSubmit(attrs ...attr.Node) html.Element {
	return html.Input(
		attr.Group(attrs...),
		attr.Type("submit"),
	)
}
