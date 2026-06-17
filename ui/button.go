package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/html"
)

func Button(attrs ...domi.Attr) domi.Element {
	return func(nodes ...domi.Node) domi.Node {
		return html.Button(
			Class("u-button"),
			group(attrs...),
		)(nodes...)
	}
}

// ButtonLink renders a button-styled link, as an <a> with the given href.
func ButtonLink(href string, attrs ...domi.Attr) domi.Element {
	return func(nodes ...domi.Node) domi.Node {
		return html.A(
			Class("u-button"),
			Href(href),
			group(attrs...),
		)(nodes...)
	}
}

// Variants — set data-button on the button or any ancestor to inherit.
var (
	ButtonSolid   = Attr("data-button")("solid")
	ButtonSurface = Attr("data-button")("surface")
	ButtonGhost   = Attr("data-button")("ghost")
)

// Sizes — set data-button-size on the button or any ancestor to inherit.
var (
	ButtonSize1 = Attr("data-button-size")("1")
	ButtonSize2 = Attr("data-button-size")("2")
	ButtonSize3 = Attr("data-button-size")("3")
	ButtonSize4 = Attr("data-button-size")("4")
)

// Shapes
var ButtonCircle = Attr("data-radius")("circle")

// ButtonSelected highlights a surface button as selected.
var ButtonSelected = Attr("data-selected")("")
