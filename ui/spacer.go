package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
)

func Spacer(attrs ...attr.Node) html.Node {
	return html.Div(
		Class("u-spacer"),
		group(attrs...),
	)()
}

var (
	SpacerSize1    = attr.Attr("data-size")("1")
	SpacerSize2    = attr.Attr("data-size")("2")
	SpacerSize3    = attr.Attr("data-size")("3")
	SpacerSize4    = attr.Attr("data-size")("4")
	SpacerSize5    = attr.Attr("data-size")("5")
	SpacerSize6    = attr.Attr("data-size")("6")
	SpacerSize7    = attr.Attr("data-size")("7")
	SpacerSize8    = attr.Attr("data-size")("8")
	SpacerSizeGrow = attr.Attr("data-size")("grow")
)
