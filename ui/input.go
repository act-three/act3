package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

var (
	InputSize1 = attr.Attr("data-size")("1")
	InputSize2 = attr.Attr("data-size")("2")
	InputSize3 = attr.Attr("data-size")("3")
)

func InputText(attrs ...attr.Node) html.Element {
	return html.Input(
		attr.Class("u-input"),
		attr.Group(attrs...),
	)
}

func InputSubmit(attrs ...attr.Node) html.Element {
	return html.Input(
		attr.Group(attrs...),
		attr.Type("submit"),
	)
}
