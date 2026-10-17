package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/html"
)

var (
	InputSize1 = Attr("data-input-size")("1")
	InputSize2 = Attr("data-input-size")("2")
	InputSize3 = Attr("data-input-size")("3")
)

func InputText(attrs ...domi.Attr) domi.Node {
	return html.Input(
		Class("u-input"),
		group(attrs...),
	)
}
