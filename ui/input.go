package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

var (
	InputSize1 = Attr("data-input-size")("1")
	InputSize2 = Attr("data-input-size")("2")
	InputSize3 = Attr("data-input-size")("3")
)

func InputText(attrs ...attr.Node) html.Element {
	return html.Input(
		Class("u-input"),
		group(attrs...),
	)
}

func InputSubmit(attrs ...attr.Node) html.Element {
	return html.Input(
		group(attrs...),
		attr.Type("submit"),
	)
}
